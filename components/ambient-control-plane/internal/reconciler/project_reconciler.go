package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
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
		if err := r.ensureNamespace(ctx, project); err != nil {
			return err
		}
		if err := r.ensureRunnerSecrets(ctx, project); err != nil {
			return err
		}
		return r.ensureCreatorRoleBinding(ctx, project)
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

var k8sNameInvalidChars = regexp.MustCompile(`[^a-z0-9-]`)

func creatorRoleBindingName(subject string) string {
	sanitized := k8sNameInvalidChars.ReplaceAllString(strings.ToLower(subject), "-")
	sanitized = strings.Trim(sanitized, "-")
	if len(sanitized) > 40 {
		sanitized = sanitized[:40]
	}
	return "ambient-admin-" + sanitized
}

func (r *ProjectReconciler) ensureCreatorRoleBinding(ctx context.Context, project types.Project) error {
	if project.Annotations == "" {
		return nil
	}

	var anns map[string]string
	if err := json.Unmarshal([]byte(project.Annotations), &anns); err != nil {
		r.logger.Warn().Str("project_id", project.ID).Err(err).Msg("failed to parse project annotations JSON; skipping creator RoleBinding")
		return nil
	}

	createdBy := strings.TrimSpace(anns["ambient-code.io/created-by"])
	if createdBy == "" {
		return nil
	}

	namespace := namespaceForProject(project)
	rbName := creatorRoleBindingName(createdBy)

	if _, err := r.kube.GetRoleBinding(ctx, namespace, rbName); err == nil {
		r.logger.Debug().Str("namespace", namespace).Str("rolebinding", rbName).Msg("creator RoleBinding already exists")
		return nil
	} else if !k8serrors.IsNotFound(err) {
		return fmt.Errorf("checking RoleBinding %s/%s: %w", namespace, rbName, err)
	}

	subjectKind := "User"
	subjectAPIGroup := "rbac.authorization.k8s.io"
	subjectNamespace := ""
	subjectName := createdBy
	if strings.HasPrefix(createdBy, "system:serviceaccount:") {
		parts := strings.SplitN(strings.TrimPrefix(createdBy, "system:serviceaccount:"), ":", 2)
		subjectKind = "ServiceAccount"
		subjectAPIGroup = ""
		if len(parts) == 2 {
			subjectNamespace = parts[0]
			subjectName = parts[1]
		}
	}

	subjectObj := map[string]interface{}{
		"kind":     subjectKind,
		"name":     subjectName,
		"apiGroup": subjectAPIGroup,
	}
	if subjectNamespace != "" {
		subjectObj["namespace"] = subjectNamespace
	}

	rb := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "RoleBinding",
			"metadata": map[string]interface{}{
				"name":      rbName,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"ambient-code.io/role":       "admin",
					"ambient-code.io/managed-by": "ambient-control-plane",
				},
			},
			"roleRef": map[string]interface{}{
				"apiGroup": "rbac.authorization.k8s.io",
				"kind":     "ClusterRole",
				"name":     "ambient-project-admin",
			},
			"subjects": []interface{}{subjectObj},
		},
	}

	if _, err := r.kube.CreateRoleBinding(ctx, namespace, rb); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("creating creator RoleBinding %s/%s: %w", namespace, rbName, err)
	}

	r.logger.Info().Str("namespace", namespace).Str("rolebinding", rbName).Str("subject", createdBy).Msg("creator RoleBinding created")
	return nil
}

func (r *ProjectReconciler) ensureRunnerSecrets(ctx context.Context, project types.Project) error {
	namespace := namespaceForProject(project)
	const secretName = "ambient-runner-secrets"

	if _, err := r.kube.GetSecret(ctx, namespace, secretName); err == nil {
		r.logger.Debug().Str("namespace", namespace).Msg("ambient-runner-secrets already exists")
		return nil
	} else if !k8serrors.IsNotFound(err) {
		return fmt.Errorf("checking ambient-runner-secrets in namespace %s: %w", namespace, err)
	}

	secret := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]interface{}{
				"name":      secretName,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"app": "ambient-runner-secrets",
				},
				"annotations": map[string]interface{}{
					"ambient-code.io/runner-secret": "true",
				},
			},
			"type": "Opaque",
			"stringData": map[string]interface{}{
				"ANTHROPIC_API_KEY": "",
			},
		},
	}

	if _, err := r.kube.CreateSecret(ctx, secret); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("creating ambient-runner-secrets in namespace %s: %w", namespace, err)
	}

	r.logger.Info().Str("namespace", namespace).Str("project_id", project.ID).Msg("ambient-runner-secrets created")
	return nil
}

func namespaceForProject(project types.Project) string {
	return strings.ToLower(project.ID)
}
