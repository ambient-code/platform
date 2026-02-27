package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/ambient/platform/components/ambient-control-plane/internal/informer"
	"github.com/ambient/platform/components/ambient-control-plane/internal/kubeclient"
	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	ConditionReady              = "Ready"
	ConditionSecretsReady       = "SecretsReady"
	ConditionPodCreated         = "PodCreated"
	ConditionPodScheduled       = "PodScheduled"
	ConditionRunnerStarted      = "RunnerStarted"
	ConditionReposReconciled    = "ReposReconciled"
	ConditionWorkflowReconciled = "WorkflowReconciled"
	ConditionReconciled         = "Reconciled"
)

const (
	PhasePending   = "Pending"
	PhaseCreating  = "Creating"
	PhaseRunning   = "Running"
	PhaseStopping  = "Stopping"
	PhaseStopped   = "Stopped"
	PhaseCompleted = "Completed"
	PhaseFailed    = "Failed"
)

var TerminalPhases = []string{
	PhaseStopped,
	PhaseCompleted,
	PhaseFailed,
}

type Reconciler interface {
	Resource() string
	Reconcile(ctx context.Context, event informer.ResourceEvent) error
}

type SessionReconciler struct {
	sdk             *sdkclient.Client
	kube            *kubeclient.KubeClient
	logger          zerolog.Logger
	lastWritebackAt sync.Map
}

func NewSessionReconciler(sdk *sdkclient.Client, kube *kubeclient.KubeClient, logger zerolog.Logger) *SessionReconciler {
	return &SessionReconciler{
		sdk:    sdk,
		kube:   kube,
		logger: logger.With().Str("reconciler", "sessions").Logger(),
	}
}

func (r *SessionReconciler) Resource() string {
	return "sessions"
}

func (r *SessionReconciler) Reconcile(ctx context.Context, event informer.ResourceEvent) error {
	session, ok := event.Object.(types.Session)
	if !ok {
		r.logger.Warn().
			Str("actual_type", fmt.Sprintf("%T", event.Object)).
			Msg("type assertion failed: expected types.Session")
		return nil
	}

	r.logger.Info().
		Str("event", string(event.Type)).
		Str("session_id", session.ID).
		Str("name", session.Name).
		Msg("session event received")

	switch event.Type {
	case informer.EventAdded:
		return r.handleAdded(ctx, session)
	case informer.EventModified:
		return r.handleModified(ctx, session)
	case informer.EventDeleted:
		return r.handleDeleted(ctx, session)
	default:
		return nil
	}
}

func (r *SessionReconciler) handleAdded(ctx context.Context, session types.Session) error {
	crName := crNameForSession(session)
	if crName == "" {
		return fmt.Errorf("session %s has no kube_cr_name or id", session.Name)
	}
	if !isValidK8sName(crName) {
		return fmt.Errorf("session CR name %q is not a valid Kubernetes resource name", crName)
	}

	existing, err := r.kube.GetAgenticSession(ctx, crName)
	if err == nil {
		r.logger.Info().Str("cr_name", crName).Msg("CR already exists for new API session, updating")
		return r.updateCR(ctx, session, existing)
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("checking for existing CR %s: %w", crName, err)
	}

	cr, err := sessionToUnstructured(session, r.kube.Namespace())
	if err != nil {
		return fmt.Errorf("building CR for session %s: %w", session.ID, err)
	}
	created, err := r.kube.CreateAgenticSession(ctx, cr)
	if err != nil {
		return fmt.Errorf("creating CR %s: %w", crName, err)
	}

	r.logger.Info().Str("cr_name", crName).Str("session_id", session.ID).Msg("created AgenticSession CR")
	r.writeStatusToAPI(ctx, session.ID, created)
	return nil
}

func (r *SessionReconciler) isWritebackEcho(session types.Session) bool {
	if session.ID == "" || session.UpdatedAt == nil {
		return false
	}
	val, ok := r.lastWritebackAt.Load(session.ID)
	if !ok {
		return false
	}
	lastWB := val.(time.Time)
	return session.UpdatedAt.Truncate(time.Microsecond).Equal(lastWB)
}

func (r *SessionReconciler) handleModified(ctx context.Context, session types.Session) error {
	if r.isWritebackEcho(session) {
		r.logger.Debug().Str("session_id", session.ID).Msg("skipping write-back echo")
		return nil
	}

	crName := crNameForSession(session)
	if crName == "" {
		return fmt.Errorf("session %s has no kube_cr_name or id", session.Name)
	}
	if !isValidK8sName(crName) {
		return fmt.Errorf("session CR name %q is not a valid Kubernetes resource name", crName)
	}

	existing, err := r.kube.GetAgenticSession(ctx, crName)
	if errors.IsNotFound(err) {
		r.logger.Info().Str("cr_name", crName).Msg("CR not found for modified session, creating")
		cr, err := sessionToUnstructured(session, r.kube.Namespace())
		if err != nil {
			return fmt.Errorf("building CR for session %s: %w", session.ID, err)
		}
		created, err := r.kube.CreateAgenticSession(ctx, cr)
		if err != nil {
			return fmt.Errorf("creating CR %s: %w", crName, err)
		}
		r.writeStatusToAPI(ctx, session.ID, created)
		return nil
	}
	if err != nil {
		return fmt.Errorf("getting CR %s: %w", crName, err)
	}

	return r.updateCR(ctx, session, existing)
}

func (r *SessionReconciler) handleDeleted(ctx context.Context, session types.Session) error {
	crName := crNameForSession(session)
	if crName == "" {
		r.logger.Warn().Str("session_id", session.ID).Msg("cannot determine CR name for deleted session")
		return nil
	}

	err := r.kube.DeleteAgenticSession(ctx, crName)
	if errors.IsNotFound(err) {
		r.logger.Debug().Str("cr_name", crName).Msg("CR already absent for deleted session")
		return nil
	}
	if err != nil {
		return fmt.Errorf("deleting CR %s: %w", crName, err)
	}

	r.lastWritebackAt.Delete(session.ID)
	r.logger.Info().Str("cr_name", crName).Str("session_id", session.ID).Msg("deleted AgenticSession CR")
	return nil
}

func (r *SessionReconciler) updateCR(ctx context.Context, session types.Session, existing *unstructured.Unstructured) error {
	updated := existing.DeepCopy()
	spec, err := buildSpec(session)
	if err != nil {
		return fmt.Errorf("building spec for session %s: %w", session.ID, err)
	}
	if err := unstructured.SetNestedField(updated.Object, spec, "spec"); err != nil {
		return fmt.Errorf("setting spec on CR: %w", err)
	}

	result, err := r.kube.UpdateAgenticSession(ctx, updated)
	if err != nil {
		return fmt.Errorf("updating CR %s: %w", existing.GetName(), err)
	}

	r.logger.Info().Str("cr_name", existing.GetName()).Str("session_id", session.ID).Msg("updated AgenticSession CR")
	r.writeStatusToAPI(ctx, session.ID, result)
	return nil
}

func (r *SessionReconciler) writeStatusToAPI(ctx context.Context, sessionID string, cr *unstructured.Unstructured) {
	if r.sdk == nil || sessionID == "" || cr == nil {
		return
	}

	patch := r.crStatusToStatusPatch(cr)

	response, err := r.sdk.Sessions().UpdateStatus(ctx, sessionID, patch)
	if err != nil {
		r.logger.Warn().Err(err).Str("session_id", sessionID).Msg("failed to write status back to API server")
		return
	}

	if response != nil && response.UpdatedAt != nil {
		r.lastWritebackAt.Store(sessionID, response.UpdatedAt.Truncate(time.Microsecond))
	}

	r.logger.Info().
		Str("session_id", sessionID).
		Str("kube_cr_uid", string(cr.GetUID())).
		Msg("wrote status back to API server")
}

func (r *SessionReconciler) crStatusToStatusPatch(cr *unstructured.Unstructured) map[string]any {
	patch := types.NewSessionStatusPatchBuilder()

	if uid := string(cr.GetUID()); uid != "" {
		patch.KubeCrUid(uid)
	}
	if ns := cr.GetNamespace(); ns != "" {
		patch.KubeNamespace(ns)
	}

	if phase, found, _ := unstructured.NestedString(cr.Object, "status", "phase"); found && phase != "" {
		patch.Phase(phase)
	}

	if startTimeStr, found, _ := unstructured.NestedString(cr.Object, "status", "startTime"); found && startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			patch.StartTime(&t)
		}
	}

	if completionTimeStr, found, _ := unstructured.NestedString(cr.Object, "status", "completionTime"); found && completionTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, completionTimeStr); err == nil {
			patch.CompletionTime(&t)
		}
	}

	if sdkSessionID, found, _ := unstructured.NestedString(cr.Object, "status", "sdkSessionId"); found && sdkSessionID != "" {
		patch.SdkSessionID(sdkSessionID)
	}

	if restartCount, found, _ := unstructured.NestedInt64(cr.Object, "status", "sdkRestartCount"); found {
		patch.SdkRestartCount(int(restartCount))
	}

	if conditions, found, _ := unstructured.NestedSlice(cr.Object, "status", "conditions"); found {
		if data, err := json.Marshal(conditions); err == nil {
			patch.Conditions(string(data))
		} else {
			r.logger.Warn().Err(err).Str("cr", cr.GetName()).Msg("failed to marshal conditions")
		}
	}

	if reconciledRepos, found, _ := unstructured.NestedSlice(cr.Object, "status", "reconciledRepos"); found {
		if data, err := json.Marshal(reconciledRepos); err == nil {
			patch.ReconciledRepos(string(data))
		} else {
			r.logger.Warn().Err(err).Str("cr", cr.GetName()).Msg("failed to marshal reconciledRepos")
		}
	}

	if reconciledWorkflow, found, _ := unstructured.NestedMap(cr.Object, "status", "reconciledWorkflow"); found {
		if data, err := json.Marshal(reconciledWorkflow); err == nil {
			patch.ReconciledWorkflow(string(data))
		} else {
			r.logger.Warn().Err(err).Str("cr", cr.GetName()).Msg("failed to marshal reconciledWorkflow")
		}
	}

	return patch.Build()
}

func crNameForSession(session types.Session) string {
	if session.KubeCrName != "" {
		return strings.ToLower(session.KubeCrName)
	}
	if session.ID != "" {
		return strings.ToLower(session.ID)
	}
	return ""
}

func autoBranchName(session types.Session) string {
	if session.KubeCrName != "" {
		return "ambient/" + strings.ToLower(session.KubeCrName)
	}
	if session.ID != "" {
		return "ambient/" + strings.ToLower(session.ID)
	}
	return "ambient/session"
}

func sessionToUnstructured(session types.Session, namespace string) (*unstructured.Unstructured, error) {
	crName := crNameForSession(session)

	spec, err := buildSpec(session)
	if err != nil {
		return nil, fmt.Errorf("building spec for CR %s: %w", crName, err)
	}

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      crName,
				"namespace": namespace,
			},
			"spec": spec,
		},
	}

	if session.Labels != "" {
		var labelMap map[string]string
		if err := json.Unmarshal([]byte(session.Labels), &labelMap); err != nil {
			return nil, fmt.Errorf("parsing labels JSON for CR %s: %w", crName, err)
		}
		labels := make(map[string]interface{}, len(labelMap))
		for k, v := range labelMap {
			labels[k] = v
		}
		if err := unstructured.SetNestedField(obj.Object, labels, "metadata", "labels"); err != nil {
			return nil, fmt.Errorf("setting labels on CR %s: %w", crName, err)
		}
	}

	if session.Annotations != "" {
		var annotationMap map[string]string
		if err := json.Unmarshal([]byte(session.Annotations), &annotationMap); err != nil {
			return nil, fmt.Errorf("parsing annotations JSON for CR %s: %w", crName, err)
		}
		annotations := make(map[string]interface{}, len(annotationMap))
		for k, v := range annotationMap {
			annotations[k] = v
		}
		if err := unstructured.SetNestedField(obj.Object, annotations, "metadata", "annotations"); err != nil {
			return nil, fmt.Errorf("setting annotations on CR %s: %w", crName, err)
		}
	}

	return obj, nil
}

func buildSpec(session types.Session) (map[string]interface{}, error) {
	spec := map[string]interface{}{}

	spec["displayName"] = session.Name

	if session.Prompt != "" {
		spec["initialPrompt"] = session.Prompt
	}

	if session.Timeout != 0 {
		spec["timeout"] = int64(session.Timeout)
	}

	if session.ProjectID != "" {
		spec["project"] = session.ProjectID
	}

	branch := autoBranchName(session)
	if session.Repos != "" {
		var repos []interface{}
		if err := json.Unmarshal([]byte(session.Repos), &repos); err != nil {
			return nil, fmt.Errorf("parsing repos JSON: %w", err)
		}
		for _, r := range repos {
			if m, ok := r.(map[string]interface{}); ok {
				if _, hasBranch := m["branch"]; !hasBranch {
					m["branch"] = branch
				}
			}
		}
		spec["repos"] = repos
	} else if session.RepoURL != "" {
		spec["repos"] = []interface{}{
			map[string]interface{}{
				"url":    session.RepoURL,
				"branch": branch,
			},
		}
	}

	if session.LlmModel != "" || session.LlmTemperature != 0 || session.LlmMaxTokens != 0 {
		llmSettings := map[string]interface{}{}
		if session.LlmModel != "" {
			llmSettings["model"] = session.LlmModel
		}
		if session.LlmTemperature != 0 {
			llmSettings["temperature"] = session.LlmTemperature
		}
		if session.LlmMaxTokens != 0 {
			llmSettings["maxTokens"] = int64(session.LlmMaxTokens)
		}
		spec["llmSettings"] = llmSettings
	}

	if session.BotAccountName != "" {
		spec["botAccount"] = map[string]interface{}{
			"name": session.BotAccountName,
		}
	}

	if session.ResourceOverrides != "" {
		var overrides map[string]interface{}
		if err := json.Unmarshal([]byte(session.ResourceOverrides), &overrides); err != nil {
			return nil, fmt.Errorf("parsing resourceOverrides JSON: %w", err)
		}
		spec["resourceOverrides"] = overrides
	}

	if session.EnvironmentVariables != "" {
		var envVars map[string]interface{}
		if err := json.Unmarshal([]byte(session.EnvironmentVariables), &envVars); err != nil {
			return nil, fmt.Errorf("parsing environmentVariables JSON: %w", err)
		}
		spec["environmentVariables"] = envVars
	}

	if session.CreatedByUserID != "" {
		spec["userContext"] = map[string]interface{}{
			"userId": session.CreatedByUserID,
		}
	}

	return spec, nil
}

const (
	LabelManaged   = "ambient-code.io/managed"
	LabelProjectID = "ambient-code.io/project-id"
	LabelManagedBy = "ambient-code.io/managed-by"
)

type ProjectReconciler struct {
	sdk    *sdkclient.Client
	kube   *kubeclient.KubeClient
	logger zerolog.Logger
}

func NewProjectReconciler(sdk *sdkclient.Client, kube *kubeclient.KubeClient, logger zerolog.Logger) *ProjectReconciler {
	return &ProjectReconciler{
		sdk:    sdk,
		kube:   kube,
		logger: logger.With().Str("reconciler", "projects").Logger(),
	}
}

func (r *ProjectReconciler) Resource() string {
	return "projects"
}

func (r *ProjectReconciler) Reconcile(ctx context.Context, event informer.ResourceEvent) error {
	project, ok := event.Object.(types.Project)
	if !ok {
		r.logger.Warn().
			Str("actual_type", fmt.Sprintf("%T", event.Object)).
			Msg("type assertion failed: expected types.Project")
		return nil
	}

	r.logger.Info().
		Str("event", string(event.Type)).
		Str("project_id", project.ID).
		Str("name", project.Name).
		Msg("project event received")

	switch event.Type {
	case informer.EventAdded, informer.EventModified:
		return r.ensureNamespace(ctx, project)
	case informer.EventDeleted:
		r.logger.Info().Str("project_name", project.Name).Msg("project deleted — namespace retained for safety")
		return nil
	default:
		return nil
	}
}

var validK8sName = regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]*[a-z0-9])?$`)

func isValidK8sName(name string) bool {
	return len(name) <= 63 && validK8sName.MatchString(name)
}

func (r *ProjectReconciler) ensureNamespace(ctx context.Context, project types.Project) error {
	nsName := project.Name
	if nsName == "" {
		return fmt.Errorf("project has no name")
	}
	if !isValidK8sName(nsName) {
		return fmt.Errorf("project name %q is not a valid Kubernetes namespace name (must match RFC 1123 DNS label)", nsName)
	}

	existing, err := r.kube.GetNamespace(ctx, nsName)
	if err == nil {
		return r.reconcileNamespaceLabels(ctx, existing, project.ID)
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("checking namespace %s: %w", nsName, err)
	}

	ns := buildNamespace(nsName, project.ID)
	_, err = r.kube.CreateNamespace(ctx, ns)
	if err != nil {
		return fmt.Errorf("creating namespace %s: %w", nsName, err)
	}

	r.logger.Info().Str("namespace", nsName).Str("project_id", project.ID).Msg("created namespace for project")
	return nil
}

func (r *ProjectReconciler) reconcileNamespaceLabels(ctx context.Context, ns *unstructured.Unstructured, projectID string) error {
	labels := ns.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}

	needsUpdate := false
	if labels[LabelManaged] != "true" {
		labels[LabelManaged] = "true"
		needsUpdate = true
	}
	if projectID != "" && labels[LabelProjectID] != projectID {
		labels[LabelProjectID] = projectID
		needsUpdate = true
	}
	if labels[LabelManagedBy] != "ambient-control-plane" {
		labels[LabelManagedBy] = "ambient-control-plane"
		needsUpdate = true
	}

	if !needsUpdate {
		return nil
	}

	updated := ns.DeepCopy()
	updated.SetLabels(labels)
	_, err := r.kube.UpdateNamespace(ctx, updated)
	if err != nil {
		return fmt.Errorf("updating namespace labels %s: %w", ns.GetName(), err)
	}

	r.logger.Info().Str("namespace", ns.GetName()).Msg("updated namespace labels")
	return nil
}

func buildNamespace(name, projectID string) *unstructured.Unstructured {
	labels := map[string]interface{}{
		LabelManaged:   "true",
		LabelManagedBy: "ambient-control-plane",
	}
	if projectID != "" {
		labels[LabelProjectID] = projectID
	}

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name":   name,
				"labels": labels,
			},
		},
	}
}

type GroupAccessEntry struct {
	Group string `json:"group"`
	Role  string `json:"role"`
}

type ProjectSettingsReconciler struct {
	sdk    *sdkclient.Client
	kube   *kubeclient.KubeClient
	logger zerolog.Logger
}

func NewProjectSettingsReconciler(sdk *sdkclient.Client, kube *kubeclient.KubeClient, logger zerolog.Logger) *ProjectSettingsReconciler {
	return &ProjectSettingsReconciler{
		sdk:    sdk,
		kube:   kube,
		logger: logger.With().Str("reconciler", "project_settings").Logger(),
	}
}

func (r *ProjectSettingsReconciler) Resource() string {
	return "project_settings"
}

func (r *ProjectSettingsReconciler) Reconcile(ctx context.Context, event informer.ResourceEvent) error {
	ps, ok := event.Object.(types.ProjectSettings)
	if !ok {
		r.logger.Warn().
			Str("actual_type", fmt.Sprintf("%T", event.Object)).
			Msg("type assertion failed: expected types.ProjectSettings")
		return nil
	}

	r.logger.Info().
		Str("event", string(event.Type)).
		Str("settings_id", ps.ID).
		Str("project_id", ps.ProjectID).
		Msg("project_settings event received")

	switch event.Type {
	case informer.EventAdded, informer.EventModified:
		return r.reconcileRoleBindings(ctx, ps)
	case informer.EventDeleted:
		r.logger.Info().Str("project_id", ps.ProjectID).Msg("project settings deleted — role bindings retained for safety")
		return nil
	default:
		return nil
	}
}

func (r *ProjectSettingsReconciler) reconcileRoleBindings(ctx context.Context, ps types.ProjectSettings) error {
	if ps.GroupAccess == "" {
		return nil
	}

	var entries []GroupAccessEntry
	if err := json.Unmarshal([]byte(ps.GroupAccess), &entries); err != nil {
		r.logger.Warn().Err(err).Str("project_id", ps.ProjectID).Msg("failed to parse group_access JSON")
		return nil
	}

	if ps.ProjectID == "" {
		return fmt.Errorf("project settings has no project_id")
	}

	project, err := r.sdk.Projects().Get(ctx, ps.ProjectID)
	if err != nil {
		return fmt.Errorf("looking up project %s for namespace: %w", ps.ProjectID, err)
	}
	namespace := project.Name
	if namespace == "" {
		return fmt.Errorf("project %s has no name", ps.ProjectID)
	}
	if !isValidK8sName(namespace) {
		return fmt.Errorf("project name %q is not a valid Kubernetes namespace name", namespace)
	}

	for _, entry := range entries {
		if entry.Group == "" || entry.Role == "" {
			continue
		}
		rbName := fmt.Sprintf("ambient-%s-%s", entry.Group, entry.Role)
		if !isValidK8sName(rbName) {
			r.logger.Warn().Str("rolebinding", rbName).Msg("generated RoleBinding name is not a valid K8s name, skipping")
			continue
		}
		if err := r.ensureRoleBinding(ctx, namespace, rbName, entry); err != nil {
			r.logger.Warn().Err(err).Str("namespace", namespace).Str("rolebinding", rbName).Msg("failed to reconcile role binding")
		}
	}
	return nil
}

func (r *ProjectSettingsReconciler) ensureRoleBinding(ctx context.Context, namespace, rbName string, entry GroupAccessEntry) error {
	existing, err := r.kube.GetRoleBinding(ctx, namespace, rbName)
	if err == nil {
		existingRole, _, _ := unstructured.NestedString(existing.Object, "roleRef", "name")
		if existingRole != entry.Role {
			if err := r.kube.DeleteRoleBinding(ctx, namespace, rbName); err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("deleting role binding %s/%s for roleRef change: %w", namespace, rbName, err)
			}
			r.logger.Info().
				Str("namespace", namespace).
				Str("rolebinding", rbName).
				Str("old_role", existingRole).
				Str("new_role", entry.Role).
				Msg("deleted role binding for immutable roleRef change, recreating")
		} else {
			updated := existing.DeepCopy()
			subjects := []interface{}{
				map[string]interface{}{
					"kind":     "Group",
					"name":     entry.Group,
					"apiGroup": "rbac.authorization.k8s.io",
				},
			}
			if err := unstructured.SetNestedSlice(updated.Object, subjects, "subjects"); err != nil {
				return fmt.Errorf("setting subjects on role binding %s/%s: %w", namespace, rbName, err)
			}
			_, err = r.kube.UpdateRoleBinding(ctx, namespace, updated)
			return err
		}
	} else if !errors.IsNotFound(err) {
		return err
	}

	rb := buildRoleBinding(namespace, rbName, entry)
	_, err = r.kube.CreateRoleBinding(ctx, namespace, rb)
	if err != nil {
		return fmt.Errorf("creating role binding %s/%s: %w", namespace, rbName, err)
	}
	r.logger.Info().
		Str("namespace", namespace).
		Str("rolebinding", rbName).
		Str("group", entry.Group).
		Str("role", entry.Role).
		Msg("created role binding")
	return nil
}

func buildRoleBinding(namespace, name string, entry GroupAccessEntry) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "RoleBinding",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels": map[string]interface{}{
					LabelManaged:   "true",
					LabelManagedBy: "ambient-control-plane",
				},
			},
			"roleRef": map[string]interface{}{
				"apiGroup": "rbac.authorization.k8s.io",
				"kind":     "ClusterRole",
				"name":     entry.Role,
			},
			"subjects": []interface{}{
				map[string]interface{}{
					"kind":     "Group",
					"name":     entry.Group,
					"apiGroup": "rbac.authorization.k8s.io",
				},
			},
		},
	}
}
