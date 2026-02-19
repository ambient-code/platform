package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	openapi "github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
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

var AllConditionTypes = []string{
	ConditionReady,
	ConditionSecretsReady,
	ConditionPodCreated,
	ConditionPodScheduled,
	ConditionRunnerStarted,
	ConditionReposReconciled,
	ConditionWorkflowReconciled,
	ConditionReconciled,
}

var AllPhases = []string{
	PhasePending,
	PhaseCreating,
	PhaseRunning,
	PhaseStopping,
	PhaseStopped,
	PhaseCompleted,
	PhaseFailed,
}

var TerminalPhases = []string{
	PhaseStopped,
	PhaseCompleted,
	PhaseFailed,
}

type Reconciler interface {
	Resource() string
	Reconcile(ctx context.Context, event informer.ResourceEvent) error
}

type FieldDiff struct {
	Field    string
	APIValue string
	K8sValue string
	Category string
}

type SessionReconciler struct {
	client          *openapi.APIClient
	kube            *kubeclient.KubeClient
	logger          zerolog.Logger
	lastWritebackAt sync.Map
}

func NewSessionReconciler(client *openapi.APIClient, kube *kubeclient.KubeClient, logger zerolog.Logger) *SessionReconciler {
	return &SessionReconciler{
		client: client,
		kube:   kube,
		logger: logger.With().Str("reconciler", "sessions").Logger(),
	}
}

func (r *SessionReconciler) Resource() string {
	return "sessions"
}

func (r *SessionReconciler) Reconcile(ctx context.Context, event informer.ResourceEvent) error {
	session, ok := event.Object.(openapi.Session)
	if !ok {
		r.logger.Warn().
			Str("actual_type", fmt.Sprintf("%T", event.Object)).
			Msg("type assertion failed: expected openapi.Session")
		return nil
	}

	r.logger.Info().
		Str("event", string(event.Type)).
		Str("session_id", session.GetId()).
		Str("name", session.GetName()).
		Msg("session event received")

	switch event.Type {
	case informer.EventAdded:
		return r.handleAdded(ctx, session)
	case informer.EventModified:
		return r.handleModified(ctx, session)
	case informer.EventDeleted:
		return r.handleDeleted(ctx, session)
	default:
		r.logger.Warn().
			Str("event_type", string(event.Type)).
			Msg("unknown event type")
		return nil
	}
}

func (r *SessionReconciler) handleAdded(ctx context.Context, session openapi.Session) error {
	crName := r.crNameForSession(session)
	if crName == "" {
		return fmt.Errorf("session %s has no kube_cr_name or id", session.GetName())
	}

	existing, err := r.kube.GetAgenticSession(ctx, crName)
	if err == nil {
		r.logger.Info().
			Str("cr_name", crName).
			Msg("CR already exists for new API session, updating")
		return r.updateCR(ctx, session, existing)
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("checking for existing CR %s: %w", crName, err)
	}

	cr := SessionToUnstructured(session, r.kube.Namespace())
	created, err := r.kube.CreateAgenticSession(ctx, cr)
	if err != nil {
		return fmt.Errorf("creating CR %s: %w", crName, err)
	}

	r.logger.Info().
		Str("cr_name", crName).
		Str("session_id", session.GetId()).
		Msg("created AgenticSession CR")

	r.writeStatusToAPI(ctx, session.GetId(), created)
	return nil
}

func autoBranchName(session openapi.Session) string {
	if session.KubeCrName != nil && *session.KubeCrName != "" {
		return "ambient/" + strings.ToLower(*session.KubeCrName)
	}
	if session.Id != nil && *session.Id != "" {
		return "ambient/" + strings.ToLower(*session.Id)
	}
	return "ambient/session"
}

func (r *SessionReconciler) isWritebackEcho(session openapi.Session) bool {
	sessionID := session.GetId()
	if sessionID == "" {
		return false
	}
	val, ok := r.lastWritebackAt.Load(sessionID)
	if !ok {
		return false
	}
	lastWB := val.(time.Time)
	return session.GetUpdatedAt().Truncate(time.Microsecond).Equal(lastWB)
}

func (r *SessionReconciler) handleModified(ctx context.Context, session openapi.Session) error {
	if r.isWritebackEcho(session) {
		r.logger.Debug().
			Str("session_id", session.GetId()).
			Msg("skipping write-back echo — updated_at matches last status write-back")
		return nil
	}

	crName := r.crNameForSession(session)
	if crName == "" {
		return fmt.Errorf("session %s has no kube_cr_name or id", session.GetName())
	}

	existing, err := r.kube.GetAgenticSession(ctx, crName)
	if errors.IsNotFound(err) {
		r.logger.Info().
			Str("cr_name", crName).
			Msg("CR not found for modified session, creating")
		cr := SessionToUnstructured(session, r.kube.Namespace())
		created, err := r.kube.CreateAgenticSession(ctx, cr)
		if err != nil {
			return fmt.Errorf("creating CR %s: %w", crName, err)
		}
		r.writeStatusToAPI(ctx, session.GetId(), created)
		return nil
	}
	if err != nil {
		return fmt.Errorf("getting CR %s: %w", crName, err)
	}

	return r.updateCR(ctx, session, existing)
}

func (r *SessionReconciler) handleDeleted(ctx context.Context, session openapi.Session) error {
	crName := r.crNameForSession(session)
	if crName == "" {
		r.logger.Warn().
			Str("session_id", session.GetId()).
			Msg("cannot determine CR name for deleted session")
		return nil
	}

	err := r.kube.DeleteAgenticSession(ctx, crName)
	if errors.IsNotFound(err) {
		r.logger.Debug().
			Str("cr_name", crName).
			Msg("CR already absent for deleted session")
		return nil
	}
	if err != nil {
		return fmt.Errorf("deleting CR %s: %w", crName, err)
	}

	r.lastWritebackAt.Delete(session.GetId())

	r.logger.Info().
		Str("cr_name", crName).
		Str("session_id", session.GetId()).
		Msg("deleted AgenticSession CR")
	return nil
}

func (r *SessionReconciler) updateCR(ctx context.Context, session openapi.Session, existing *unstructured.Unstructured) error {
	updated := existing.DeepCopy()
	spec := buildSpec(session)
	if err := unstructured.SetNestedField(updated.Object, spec, "spec"); err != nil {
		return fmt.Errorf("setting spec on CR: %w", err)
	}

	result, err := r.kube.UpdateAgenticSession(ctx, updated)
	if err != nil {
		return fmt.Errorf("updating CR %s: %w", existing.GetName(), err)
	}

	r.logger.Info().
		Str("cr_name", existing.GetName()).
		Str("session_id", session.GetId()).
		Msg("updated AgenticSession CR")

	r.writeStatusToAPI(ctx, session.GetId(), result)
	return nil
}

func (r *SessionReconciler) writeStatusToAPI(ctx context.Context, sessionID string, cr *unstructured.Unstructured) {
	if r.client == nil || sessionID == "" || cr == nil {
		return
	}

	patch := CRStatusToStatusPatch(cr)

	response, _, err := r.client.DefaultAPI.
		ApiAmbientApiServerV1SessionsIdStatusPatch(ctx, sessionID).
		SessionStatusPatchRequest(patch).
		Execute()
	if err != nil {
		r.logger.Warn().
			Err(err).
			Str("session_id", sessionID).
			Msg("failed to write status back to API server")
		return
	}

	if response != nil && response.HasUpdatedAt() {
		r.lastWritebackAt.Store(sessionID, response.GetUpdatedAt().Truncate(time.Microsecond))
	}

	r.logger.Info().
		Str("session_id", sessionID).
		Str("kube_cr_uid", string(cr.GetUID())).
		Msg("wrote status back to API server")
}

func CRStatusToStatusPatch(cr *unstructured.Unstructured) openapi.SessionStatusPatchRequest {
	patch := *openapi.NewSessionStatusPatchRequest()

	if uid := string(cr.GetUID()); uid != "" {
		patch.SetKubeCrUid(uid)
	}
	if ns := cr.GetNamespace(); ns != "" {
		patch.SetKubeNamespace(ns)
	}

	if phase, found, _ := unstructured.NestedString(cr.Object, "status", "phase"); found && phase != "" {
		patch.SetPhase(phase)
	}

	if startTimeStr, found, _ := unstructured.NestedString(cr.Object, "status", "startTime"); found && startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			patch.SetStartTime(t)
		}
	}

	if completionTimeStr, found, _ := unstructured.NestedString(cr.Object, "status", "completionTime"); found && completionTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, completionTimeStr); err == nil {
			patch.SetCompletionTime(t)
		}
	}

	if sdkSessionID, found, _ := unstructured.NestedString(cr.Object, "status", "sdkSessionId"); found && sdkSessionID != "" {
		patch.SetSdkSessionId(sdkSessionID)
	}

	if restartCount, found, _ := unstructured.NestedInt64(cr.Object, "status", "sdkRestartCount"); found {
		patch.SetSdkRestartCount(int32(restartCount))
	}

	if conditions, found, _ := unstructured.NestedSlice(cr.Object, "status", "conditions"); found {
		if data, err := json.Marshal(conditions); err == nil {
			patch.SetConditions(string(data))
		}
	}

	if reconciledRepos, found, _ := unstructured.NestedSlice(cr.Object, "status", "reconciledRepos"); found {
		if data, err := json.Marshal(reconciledRepos); err == nil {
			patch.SetReconciledRepos(string(data))
		}
	}

	if reconciledWorkflow, found, _ := unstructured.NestedMap(cr.Object, "status", "reconciledWorkflow"); found {
		if data, err := json.Marshal(reconciledWorkflow); err == nil {
			patch.SetReconciledWorkflow(string(data))
		}
	}

	return patch
}

func (r *SessionReconciler) crNameForSession(session openapi.Session) string {
	if session.KubeCrName != nil && *session.KubeCrName != "" {
		return strings.ToLower(*session.KubeCrName)
	}
	if session.Id != nil && *session.Id != "" {
		return strings.ToLower(*session.Id)
	}
	return ""
}

func (r *SessionReconciler) diffAgainstKubernetes(ctx context.Context, session openapi.Session) []FieldDiff {
	sessionName := session.GetName()
	if sessionName == "" {
		r.logger.Warn().
			Str("session_id", session.GetId()).
			Msg("session has no name, cannot look up k8s resource")
		return nil
	}

	cr, err := r.kube.GetAgenticSession(ctx, sessionName)
	if err != nil {
		r.logger.Warn().
			Err(err).
			Str("session_name", sessionName).
			Str("namespace", r.kube.Namespace()).
			Msg("k8s lookup failed for session")
		return nil
	}

	return r.compareSessionToCR(session, cr)
}

func (r *SessionReconciler) compareSessionToCR(session openapi.Session, cr *unstructured.Unstructured) []FieldDiff {
	var diffs []FieldDiff

	crName := cr.GetName()
	if apiName := session.GetName(); apiName != crName {
		diffs = append(diffs, FieldDiff{
			Field:    "name",
			APIValue: apiName,
			K8sValue: crName,
			Category: "identity",
		})
	}

	displayName, _, _ := unstructured.NestedString(cr.Object, "spec", "displayName")
	if apiName := session.GetName(); apiName != displayName {
		diffs = append(diffs, FieldDiff{
			Field:    "name↔displayName",
			APIValue: fmt.Sprintf("name=%q", apiName),
			K8sValue: fmt.Sprintf("spec.displayName=%q", displayName),
			Category: "field-mapping",
		})
	}

	crPrompt, _, _ := unstructured.NestedString(cr.Object, "spec", "initialPrompt")
	apiPrompt := ptrStr(session.Prompt)
	if apiPrompt != crPrompt {
		diffs = append(diffs, FieldDiff{
			Field:    "prompt↔initialPrompt",
			APIValue: truncate(apiPrompt, 80),
			K8sValue: truncate(crPrompt, 80),
			Category: "field-mapping",
		})
	}

	apiRepoURL := ptrStr(session.RepoUrl)
	crRepos, _, _ := unstructured.NestedSlice(cr.Object, "spec", "repos")
	crRepoURLs := extractRepoURLs(crRepos)
	if apiRepoURL != "" && !containsString(crRepoURLs, apiRepoURL) {
		diffs = append(diffs, FieldDiff{
			Field:    "repo_url↔repos",
			APIValue: fmt.Sprintf("repo_url=%q", apiRepoURL),
			K8sValue: fmt.Sprintf("spec.repos=%v", crRepoURLs),
			Category: "structural",
		})
	} else if apiRepoURL == "" && len(crRepoURLs) > 0 {
		diffs = append(diffs, FieldDiff{
			Field:    "repo_url↔repos",
			APIValue: "(empty)",
			K8sValue: fmt.Sprintf("spec.repos=%v", crRepoURLs),
			Category: "structural",
		})
	}

	if session.WorkflowId != nil {
		crWorkflow, crWfFound, _ := unstructured.NestedMap(cr.Object, "spec", "activeWorkflow")
		if !crWfFound {
			diffs = append(diffs, FieldDiff{
				Field:    "workflow_id↔activeWorkflow",
				APIValue: fmt.Sprintf("workflow_id=%q", ptrStr(session.WorkflowId)),
				K8sValue: "(not set)",
				Category: "field-mapping",
			})
		} else {
			crGitURL, _ := crWorkflow["gitUrl"].(string)
			diffs = append(diffs, FieldDiff{
				Field:    "workflow_id↔activeWorkflow",
				APIValue: fmt.Sprintf("workflow_id=%q", ptrStr(session.WorkflowId)),
				K8sValue: fmt.Sprintf("spec.activeWorkflow.gitUrl=%q", crGitURL),
				Category: "field-mapping",
			})
		}
	}

	diffs = append(diffs, r.findAPIOnlyFields(session)...)
	diffs = append(diffs, r.findK8sOnlyFields(cr)...)

	return diffs
}

func (r *SessionReconciler) findAPIOnlyFields(session openapi.Session) []FieldDiff {
	var diffs []FieldDiff

	if session.CreatedByUserId != nil && *session.CreatedByUserId != "" {
		diffs = append(diffs, FieldDiff{
			Field:    "created_by_user_id",
			APIValue: *session.CreatedByUserId,
			K8sValue: "(no equivalent — k8s uses spec.userContext.userId)",
			Category: "api-only",
		})
	}

	if session.AssignedUserId != nil && *session.AssignedUserId != "" {
		diffs = append(diffs, FieldDiff{
			Field:    "assigned_user_id",
			APIValue: *session.AssignedUserId,
			K8sValue: "(no equivalent field in CRD)",
			Category: "api-only",
		})
	}

	if session.Id != nil && *session.Id != "" {
		diffs = append(diffs, FieldDiff{
			Field:    "id",
			APIValue: *session.Id,
			K8sValue: "(k8s uses metadata.uid)",
			Category: "identity-mapping",
		})
	}

	return diffs
}

func (r *SessionReconciler) findK8sOnlyFields(cr *unstructured.Unstructured) []FieldDiff {
	var diffs []FieldDiff

	if _, found, _ := unstructured.NestedBool(cr.Object, "spec", "interactive"); found {
		diffs = append(diffs, FieldDiff{
			Field:    "spec.interactive",
			APIValue: "(no equivalent field in API)",
			K8sValue: "present",
			Category: "k8s-only",
		})
	}

	if _, found, _ := unstructured.NestedMap(cr.Object, "spec", "llmSettings"); found {
		diffs = append(diffs, FieldDiff{
			Field:    "spec.llmSettings",
			APIValue: "(no equivalent field in API)",
			K8sValue: "present (model, temperature, maxTokens)",
			Category: "k8s-only",
		})
	}

	if v, found, _ := unstructured.NestedInt64(cr.Object, "spec", "timeout"); found {
		diffs = append(diffs, FieldDiff{
			Field:    "spec.timeout",
			APIValue: "(no equivalent field in API)",
			K8sValue: fmt.Sprintf("%d", v),
			Category: "k8s-only",
		})
	}

	if _, found, _ := unstructured.NestedMap(cr.Object, "spec", "userContext"); found {
		diffs = append(diffs, FieldDiff{
			Field:    "spec.userContext",
			APIValue: "(no equivalent — API has created_by_user_id)",
			K8sValue: "present (userId, displayName, groups)",
			Category: "k8s-only",
		})
	}

	if _, found, _ := unstructured.NestedMap(cr.Object, "spec", "resourceOverrides"); found {
		diffs = append(diffs, FieldDiff{
			Field:    "spec.resourceOverrides",
			APIValue: "(no equivalent field in API)",
			K8sValue: "present (cpu, memory, storageClass, priorityClass)",
			Category: "k8s-only",
		})
	}

	if _, found, _ := unstructured.NestedStringMap(cr.Object, "spec", "environmentVariables"); found {
		diffs = append(diffs, FieldDiff{
			Field:    "spec.environmentVariables",
			APIValue: "(no equivalent field in API)",
			K8sValue: "present",
			Category: "k8s-only",
		})
	}

	if _, found, _ := unstructured.NestedString(cr.Object, "spec", "project"); found {
		diffs = append(diffs, FieldDiff{
			Field:    "spec.project",
			APIValue: "(no equivalent field in API)",
			K8sValue: "present",
			Category: "k8s-only",
		})
	}

	if _, found, _ := unstructured.NestedMap(cr.Object, "status"); found {
		diffs = append(diffs, FieldDiff{
			Field:    "status",
			APIValue: "(no status fields in API Session)",
			K8sValue: "present (phase, conditions, reconciledRepos, etc.)",
			Category: "k8s-only",
		})
	}

	return diffs
}

func (r *SessionReconciler) logFieldDiffs(sessionID, sessionName string, diffs []FieldDiff) {
	categories := map[string][]FieldDiff{}
	for _, d := range diffs {
		categories[d.Category] = append(categories[d.Category], d)
	}

	r.logger.Warn().
		Str("session_id", sessionID).
		Str("session_name", sessionName).
		Int("total_diffs", len(diffs)).
		Msg("API↔K8s field differences detected")

	for cat, catDiffs := range categories {
		for _, d := range catDiffs {
			r.logger.Info().
				Str("session_id", sessionID).
				Str("category", cat).
				Str("field", d.Field).
				Str("api_value", d.APIValue).
				Str("k8s_value", d.K8sValue).
				Msg("field diff")
		}
	}
}

type WorkflowReconciler struct {
	client *openapi.APIClient
	kube   *kubeclient.KubeClient
	logger zerolog.Logger
}

func NewWorkflowReconciler(client *openapi.APIClient, kube *kubeclient.KubeClient, logger zerolog.Logger) *WorkflowReconciler {
	return &WorkflowReconciler{
		client: client,
		kube:   kube,
		logger: logger.With().Str("reconciler", "workflows").Logger(),
	}
}

func (r *WorkflowReconciler) Resource() string {
	return "workflows"
}

func (r *WorkflowReconciler) Reconcile(ctx context.Context, event informer.ResourceEvent) error {
	workflow, ok := event.Object.(openapi.Workflow)
	if !ok {
		r.logger.Warn().
			Str("actual_type", fmt.Sprintf("%T", event.Object)).
			Msg("type assertion failed: expected openapi.Workflow")
		return nil
	}

	r.logger.Info().
		Str("event", string(event.Type)).
		Str("workflow_id", workflow.GetId()).
		Str("name", workflow.GetName()).
		Msg("workflow event received (no k8s CRD — database-only resource)")

	return nil
}

type TaskReconciler struct {
	client *openapi.APIClient
	kube   *kubeclient.KubeClient
	logger zerolog.Logger
}

func NewTaskReconciler(client *openapi.APIClient, kube *kubeclient.KubeClient, logger zerolog.Logger) *TaskReconciler {
	return &TaskReconciler{
		client: client,
		kube:   kube,
		logger: logger.With().Str("reconciler", "tasks").Logger(),
	}
}

func (r *TaskReconciler) Resource() string {
	return "tasks"
}

func (r *TaskReconciler) Reconcile(ctx context.Context, event informer.ResourceEvent) error {
	task, ok := event.Object.(openapi.Task)
	if !ok {
		r.logger.Warn().
			Str("actual_type", fmt.Sprintf("%T", event.Object)).
			Msg("type assertion failed: expected openapi.Task")
		return nil
	}

	r.logger.Info().
		Str("event", string(event.Type)).
		Str("task_id", task.GetId()).
		Str("name", task.GetName()).
		Msg("task event received (no k8s CRD — database-only resource)")

	return nil
}

func SessionToUnstructured(session openapi.Session, namespace string) *unstructured.Unstructured {
	crName := ""
	if session.KubeCrName != nil && *session.KubeCrName != "" {
		crName = strings.ToLower(*session.KubeCrName)
	} else if session.Id != nil && *session.Id != "" {
		crName = strings.ToLower(*session.Id)
	}

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      crName,
				"namespace": namespace,
			},
			"spec": buildSpec(session),
		},
	}

	if session.Labels != nil && *session.Labels != "" {
		var labelMap map[string]string
		if err := json.Unmarshal([]byte(*session.Labels), &labelMap); err == nil {
			labels := make(map[string]interface{}, len(labelMap))
			for k, v := range labelMap {
				labels[k] = v
			}
			unstructured.SetNestedField(obj.Object, labels, "metadata", "labels")
		}
	}

	if session.Annotations != nil && *session.Annotations != "" {
		var annotationMap map[string]string
		if err := json.Unmarshal([]byte(*session.Annotations), &annotationMap); err == nil {
			annotations := make(map[string]interface{}, len(annotationMap))
			for k, v := range annotationMap {
				annotations[k] = v
			}
			unstructured.SetNestedField(obj.Object, annotations, "metadata", "annotations")
		}
	}

	return obj
}

func buildSpec(session openapi.Session) map[string]interface{} {
	spec := map[string]interface{}{}

	spec["displayName"] = session.GetName()

	if session.Prompt != nil {
		spec["initialPrompt"] = *session.Prompt
	}

	if session.Interactive != nil {
		spec["interactive"] = *session.Interactive
	}

	if session.Timeout != nil {
		spec["timeout"] = int64(*session.Timeout)
	}

	if session.ProjectId != nil {
		spec["project"] = *session.ProjectId
	}

	branch := autoBranchName(session)
	if session.Repos != nil && *session.Repos != "" {
		var repos []interface{}
		if err := json.Unmarshal([]byte(*session.Repos), &repos); err == nil {
			for _, r := range repos {
				if m, ok := r.(map[string]interface{}); ok {
					if _, hasBranch := m["branch"]; !hasBranch {
						m["branch"] = branch
					}
				}
			}
			spec["repos"] = repos
		}
	} else if session.RepoUrl != nil && *session.RepoUrl != "" {
		spec["repos"] = []interface{}{
			map[string]interface{}{
				"url":    *session.RepoUrl,
				"branch": branch,
			},
		}
	}

	if session.LlmModel != nil || session.LlmTemperature != nil || session.LlmMaxTokens != nil {
		llmSettings := map[string]interface{}{}
		if session.LlmModel != nil {
			llmSettings["model"] = *session.LlmModel
		}
		if session.LlmTemperature != nil {
			llmSettings["temperature"] = *session.LlmTemperature
		}
		if session.LlmMaxTokens != nil {
			llmSettings["maxTokens"] = int64(*session.LlmMaxTokens)
		}
		spec["llmSettings"] = llmSettings
	}

	if session.BotAccountName != nil && *session.BotAccountName != "" {
		spec["botAccount"] = map[string]interface{}{
			"name": *session.BotAccountName,
		}
	}

	if session.ResourceOverrides != nil && *session.ResourceOverrides != "" {
		var overrides map[string]interface{}
		if err := json.Unmarshal([]byte(*session.ResourceOverrides), &overrides); err == nil {
			spec["resourceOverrides"] = overrides
		}
	}

	if session.EnvironmentVariables != nil && *session.EnvironmentVariables != "" {
		var envVars map[string]interface{}
		if err := json.Unmarshal([]byte(*session.EnvironmentVariables), &envVars); err == nil {
			spec["environmentVariables"] = envVars
		}
	}

	if session.CreatedByUserId != nil && *session.CreatedByUserId != "" {
		userContext := map[string]interface{}{
			"userId": *session.CreatedByUserId,
		}
		spec["userContext"] = userContext
	}

	return spec
}

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func extractRepoURLs(repos []interface{}) []string {
	var urls []string
	for _, r := range repos {
		if m, ok := r.(map[string]interface{}); ok {
			if u, ok := m["url"].(string); ok {
				urls = append(urls, u)
			}
		}
	}
	return urls
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}

const (
	LabelManaged   = "ambient-code.io/managed"
	LabelProjectID = "ambient-code.io/project-id"
	LabelManagedBy = "ambient-code.io/managed-by"
)

type ProjectReconciler struct {
	client *openapi.APIClient
	kube   *kubeclient.KubeClient
	logger zerolog.Logger
}

func NewProjectReconciler(client *openapi.APIClient, kube *kubeclient.KubeClient, logger zerolog.Logger) *ProjectReconciler {
	return &ProjectReconciler{
		client: client,
		kube:   kube,
		logger: logger.With().Str("reconciler", "projects").Logger(),
	}
}

func (r *ProjectReconciler) Resource() string {
	return "projects"
}

func (r *ProjectReconciler) Reconcile(ctx context.Context, event informer.ResourceEvent) error {
	project, ok := event.Object.(openapi.Project)
	if !ok {
		r.logger.Warn().
			Str("actual_type", fmt.Sprintf("%T", event.Object)).
			Msg("type assertion failed: expected openapi.Project")
		return nil
	}

	r.logger.Info().
		Str("event", string(event.Type)).
		Str("project_id", project.GetId()).
		Str("name", project.GetName()).
		Msg("project event received")

	switch event.Type {
	case informer.EventAdded:
		return r.ensureNamespace(ctx, project)
	case informer.EventModified:
		return r.ensureNamespace(ctx, project)
	case informer.EventDeleted:
		r.logger.Info().
			Str("project_name", project.GetName()).
			Msg("project deleted — namespace retained for safety")
		return nil
	default:
		return nil
	}
}

func (r *ProjectReconciler) ensureNamespace(ctx context.Context, project openapi.Project) error {
	nsName := project.GetName()
	if nsName == "" {
		return fmt.Errorf("project has no name")
	}

	projectID := project.GetId()

	existing, err := r.kube.GetNamespace(ctx, nsName)
	if err == nil {
		return r.reconcileNamespaceLabels(ctx, existing, projectID)
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("checking namespace %s: %w", nsName, err)
	}

	ns := buildNamespace(nsName, projectID)
	_, err = r.kube.CreateNamespace(ctx, ns)
	if err != nil {
		return fmt.Errorf("creating namespace %s: %w", nsName, err)
	}

	r.logger.Info().
		Str("namespace", nsName).
		Str("project_id", projectID).
		Msg("created namespace for project")
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

	r.logger.Info().
		Str("namespace", ns.GetName()).
		Msg("updated namespace labels")
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
	client *openapi.APIClient
	kube   *kubeclient.KubeClient
	logger zerolog.Logger
}

func NewProjectSettingsReconciler(client *openapi.APIClient, kube *kubeclient.KubeClient, logger zerolog.Logger) *ProjectSettingsReconciler {
	return &ProjectSettingsReconciler{
		client: client,
		kube:   kube,
		logger: logger.With().Str("reconciler", "project_settings").Logger(),
	}
}

func (r *ProjectSettingsReconciler) Resource() string {
	return "project_settings"
}

func (r *ProjectSettingsReconciler) Reconcile(ctx context.Context, event informer.ResourceEvent) error {
	ps, ok := event.Object.(openapi.ProjectSettings)
	if !ok {
		r.logger.Warn().
			Str("actual_type", fmt.Sprintf("%T", event.Object)).
			Msg("type assertion failed: expected openapi.ProjectSettings")
		return nil
	}

	r.logger.Info().
		Str("event", string(event.Type)).
		Str("settings_id", ps.GetId()).
		Str("project_id", ps.GetProjectId()).
		Msg("project_settings event received")

	switch event.Type {
	case informer.EventAdded, informer.EventModified:
		return r.reconcileRoleBindings(ctx, ps)
	case informer.EventDeleted:
		r.logger.Info().
			Str("project_id", ps.GetProjectId()).
			Msg("project settings deleted — role bindings retained for safety")
		return nil
	default:
		return nil
	}
}

func (r *ProjectSettingsReconciler) reconcileRoleBindings(ctx context.Context, ps openapi.ProjectSettings) error {
	groupAccessJSON := ptrStr(ps.GroupAccess)
	if groupAccessJSON == "" {
		return nil
	}

	var entries []GroupAccessEntry
	if err := json.Unmarshal([]byte(groupAccessJSON), &entries); err != nil {
		r.logger.Warn().
			Err(err).
			Str("project_id", ps.GetProjectId()).
			Msg("failed to parse group_access JSON")
		return nil
	}

	namespace := ps.GetProjectId()
	if namespace == "" {
		return fmt.Errorf("project settings has no project_id")
	}

	for _, entry := range entries {
		if entry.Group == "" || entry.Role == "" {
			continue
		}
		rbName := fmt.Sprintf("ambient-%s-%s", entry.Group, entry.Role)
		if err := r.ensureRoleBinding(ctx, namespace, rbName, entry); err != nil {
			r.logger.Warn().
				Err(err).
				Str("namespace", namespace).
				Str("rolebinding", rbName).
				Msg("failed to reconcile role binding")
		}
	}
	return nil
}

func (r *ProjectSettingsReconciler) ensureRoleBinding(ctx context.Context, namespace, rbName string, entry GroupAccessEntry) error {
	existing, err := r.kube.GetRoleBinding(ctx, namespace, rbName)
	if err == nil {
		updated := existing.DeepCopy()
		unstructured.SetNestedField(updated.Object, entry.Role, "roleRef", "name")
		subjects := []interface{}{
			map[string]interface{}{
				"kind":     "Group",
				"name":     entry.Group,
				"apiGroup": "rbac.authorization.k8s.io",
			},
		}
		unstructured.SetNestedSlice(updated.Object, subjects, "subjects")
		_, err = r.kube.UpdateRoleBinding(ctx, namespace, updated)
		return err
	}
	if !errors.IsNotFound(err) {
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
