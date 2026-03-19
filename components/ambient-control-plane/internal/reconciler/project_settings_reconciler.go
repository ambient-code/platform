package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ambient-code/platform/components/ambient-control-plane/internal/informer"
	"github.com/ambient-code/platform/components/ambient-control-plane/internal/kubeclient"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/rs/zerolog"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var projectSettingsGVR = schema.GroupVersionResource{
	Group:    "vteam.ambient-code",
	Version:  "v1alpha1",
	Resource: "projectsettings",
}

type ProjectSettingsReconciler struct {
	factory *SDKClientFactory
	kube    *kubeclient.KubeClient
	logger  zerolog.Logger
}

func NewProjectSettingsReconciler(factory *SDKClientFactory, kube *kubeclient.KubeClient, logger zerolog.Logger) *ProjectSettingsReconciler {
	return &ProjectSettingsReconciler{
		factory: factory,
		kube:    kube,
		logger:  logger.With().Str("reconciler", "project_settings").Logger(),
	}
}

func (r *ProjectSettingsReconciler) Resource() string {
	return "project_settings"
}

func (r *ProjectSettingsReconciler) Reconcile(ctx context.Context, event informer.ResourceEvent) error {
	if event.Object.ProjectSettings == nil {
		r.logger.Warn().Msg("expected project settings object in project settings event")
		return nil
	}
	ps := *event.Object.ProjectSettings

	r.logger.Info().
		Str("event", string(event.Type)).
		Str("settings_id", ps.ID).
		Str("project_id", ps.ProjectID).
		Msg("project_settings event received")

	switch event.Type {
	case informer.EventAdded, informer.EventModified:
		if err := r.ensureProjectSettings(ctx, ps); err != nil {
			return err
		}
		return r.reconcileGroupAccess(ctx, ps)
	case informer.EventDeleted:
		r.logger.Info().Str("settings_id", ps.ID).Msg("project_settings deleted — K8s object retained")
	}
	return nil
}

func (r *ProjectSettingsReconciler) ensureProjectSettings(ctx context.Context, ps types.ProjectSettings) error {
	namespace := strings.ToLower(ps.ProjectID)
	if namespace == "" {
		return fmt.Errorf("project_settings %s has no project_id; skipping", ps.ID)
	}

	_, err := r.kube.GetResource(ctx, projectSettingsGVR, namespace, "projectsettings")
	if err == nil {
		r.logger.Debug().Str("namespace", namespace).Msg("projectsettings CRD already exists")
		return nil
	}
	if !k8serrors.IsNotFound(err) {
		return fmt.Errorf("checking projectsettings in namespace %s: %w", namespace, err)
	}

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "ProjectSettings",
			"metadata": map[string]interface{}{
				"name":      "projectsettings",
				"namespace": namespace,
				"labels": map[string]interface{}{
					LabelManaged:   "true",
					LabelProjectID: ps.ProjectID,
					LabelManagedBy: "ambient-control-plane",
				},
			},
			"spec": map[string]interface{}{
				"groupAccess": []interface{}{},
			},
		},
	}

	if _, err := r.kube.CreateResource(ctx, projectSettingsGVR, namespace, obj); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("creating projectsettings in namespace %s: %w", namespace, err)
	}

	r.logger.Info().Str("namespace", namespace).Str("project_id", ps.ProjectID).Msg("projectsettings CRD created")
	return nil
}

func (r *ProjectSettingsReconciler) reconcileGroupAccess(ctx context.Context, ps types.ProjectSettings) error {
	if ps.GroupAccess == "" {
		return nil
	}
	namespace := strings.ToLower(ps.ProjectID)
	if namespace == "" {
		return nil
	}

	var entries []struct {
		GroupName string `json:"groupName"`
		Role      string `json:"role"`
	}
	if err := json.Unmarshal([]byte(ps.GroupAccess), &entries); err != nil {
		r.logger.Warn().Err(err).Str("project_id", ps.ProjectID).Msg("failed to parse group_access JSON; skipping RoleBinding reconciliation")
		return nil
	}

	for _, entry := range entries {
		if entry.GroupName == "" || entry.Role == "" {
			continue
		}
		if err := r.ensureGroupRoleBinding(ctx, namespace, entry.GroupName, entry.Role); err != nil {
			r.logger.Error().Err(err).Str("namespace", namespace).Str("group", entry.GroupName).Str("role", entry.Role).Msg("failed to ensure group RoleBinding")
		}
	}
	return nil
}

func (r *ProjectSettingsReconciler) ensureGroupRoleBinding(ctx context.Context, namespace, groupName, role string) error {
	clusterRole := mapRoleToClusterRole(role)
	rbName := fmt.Sprintf("%s-%s", groupName, role)

	if _, err := r.kube.GetRoleBinding(ctx, namespace, rbName); err == nil {
		r.logger.Debug().Str("namespace", namespace).Str("rolebinding", rbName).Msg("group RoleBinding already exists")
		return nil
	} else if !k8serrors.IsNotFound(err) {
		return fmt.Errorf("checking group RoleBinding %s/%s: %w", namespace, rbName, err)
	}

	rb := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "RoleBinding",
			"metadata": map[string]interface{}{
				"name":      rbName,
				"namespace": namespace,
				"labels": map[string]interface{}{
					LabelManaged:   "true",
					LabelManagedBy: "ambient-control-plane",
				},
			},
			"roleRef": map[string]interface{}{
				"apiGroup": "rbac.authorization.k8s.io",
				"kind":     "ClusterRole",
				"name":     clusterRole,
			},
			"subjects": []interface{}{
				map[string]interface{}{
					"apiGroup": "rbac.authorization.k8s.io",
					"kind":     "Group",
					"name":     groupName,
				},
			},
		},
	}

	if _, err := r.kube.CreateRoleBinding(ctx, namespace, rb); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("creating group RoleBinding %s/%s: %w", namespace, rbName, err)
	}

	r.logger.Info().Str("namespace", namespace).Str("rolebinding", rbName).Str("group", groupName).Str("cluster_role", clusterRole).Msg("group RoleBinding created")
	return nil
}

func mapRoleToClusterRole(role string) string {
	switch strings.ToLower(role) {
	case "admin":
		return "ambient-project-admin"
	case "edit":
		return "ambient-project-edit"
	default:
		return "ambient-project-view"
	}
}
