package reconciler

import (
	"context"
	"fmt"
	"strings"

	"github.com/ambient-code/platform/components/ambient-control-plane/internal/informer"
	"github.com/ambient-code/platform/components/ambient-control-plane/internal/kubeclient"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/rs/zerolog"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ProjectReconciler struct {
	factory *SDKClientFactory
	kube    *kubeclient.KubeClient
	logger  zerolog.Logger
}

func NewProjectReconciler(factory *SDKClientFactory, kube *kubeclient.KubeClient, logger zerolog.Logger) *ProjectReconciler {
	return &ProjectReconciler{
		factory: factory,
		kube:    kube,
		logger:  logger.With().Str("reconciler", "projects").Logger(),
	}
}

func (r *ProjectReconciler) Resource() string {
	return "projects"
}

func (r *ProjectReconciler) Reconcile(ctx context.Context, event informer.ResourceEvent) error {
	if event.Object.Project == nil {
		r.logger.Warn().Msg("expected project object in project event")
		return nil
	}
	project := *event.Object.Project

	r.logger.Info().
		Str("event", string(event.Type)).
		Str("project_id", project.ID).
		Str("name", project.Name).
		Msg("project event received")

	switch event.Type {
	case informer.EventAdded, informer.EventModified:
		return r.ensureNamespace(ctx, project)
	case informer.EventDeleted:
		r.logger.Info().Str("project_id", project.ID).Msg("project deleted — namespace retained for safety")
	}
	return nil
}

func (r *ProjectReconciler) ensureNamespace(ctx context.Context, project types.Project) error {
	name := namespaceForProject(project)

	_, err := r.kube.GetNamespace(ctx, name)
	if err == nil {
		r.logger.Debug().Str("namespace", name).Msg("namespace already exists")
		return nil
	}
	if !k8serrors.IsNotFound(err) {
		return fmt.Errorf("checking namespace %s: %w", name, err)
	}

	ns := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": name,
				"labels": map[string]interface{}{
					LabelManaged:   "true",
					LabelProjectID: project.ID,
					LabelManagedBy: "ambient-control-plane",
				},
				"annotations": map[string]interface{}{
					"ambient-code.io/project-name": project.Name,
				},
			},
		},
	}

	if _, err := r.kube.CreateNamespace(ctx, ns); err != nil {
		return fmt.Errorf("creating namespace %s: %w", name, err)
	}

	r.logger.Info().Str("namespace", name).Str("project_id", project.ID).Msg("namespace created")
	return nil
}

func namespaceForProject(project types.Project) string {
	return strings.ToLower(project.ID)
}
