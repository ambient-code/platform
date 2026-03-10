package reconciler

import (
	"context"

	"github.com/ambient-code/platform/components/ambient-control-plane/internal/informer"
	"github.com/ambient-code/platform/components/ambient-control-plane/internal/kubeclient"
	"github.com/rs/zerolog"
)

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
		Msg("project_settings event received - no action needed for MVP")

	return nil
}
