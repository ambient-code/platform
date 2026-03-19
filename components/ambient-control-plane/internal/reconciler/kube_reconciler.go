package reconciler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ambient-code/platform/components/ambient-control-plane/internal/informer"
	"github.com/ambient-code/platform/components/ambient-control-plane/internal/kubeclient"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/rs/zerolog"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type KubeReconcilerConfig struct {
	RunnerImage           string
	BackendURL            string
	RunnerGRPCURL         string
	RunnerGRPCUseTLS      bool
	AnthropicAPIKey       string
	VertexEnabled         bool
	VertexProjectID       string
	VertexRegion          string
	VertexCredentialsPath string
	VertexSecretName      string
	VertexSecretNamespace string
	RunnerImageNamespace  string
}

type SimpleKubeReconciler struct {
	factory *SDKClientFactory
	kube    *kubeclient.KubeClient
	cfg     KubeReconcilerConfig
	logger  zerolog.Logger
}

func NewKubeReconciler(factory *SDKClientFactory, kube *kubeclient.KubeClient, cfg KubeReconcilerConfig, logger zerolog.Logger) *SimpleKubeReconciler {
	return &SimpleKubeReconciler{
		factory: factory,
		kube:    kube,
		cfg:     cfg,
		logger:  logger.With().Str("reconciler", "kube").Logger(),
	}
}

func (r *SimpleKubeReconciler) Resource() string {
	return "sessions"
}

func (r *SimpleKubeReconciler) Reconcile(ctx context.Context, event informer.ResourceEvent) error {
	if event.Object.Session == nil {
		r.logger.Warn().Msg("expected session object in session event")
		return nil
	}
	session := *event.Object.Session

	r.logger.Info().
		Str("event", string(event.Type)).
		Str("session_id", session.ID).
		Str("name", session.Name).
		Str("phase", session.Phase).
		Msg("session event received")

	switch event.Type {
	case informer.EventAdded:
		if session.Phase == PhasePending || session.Phase == "" {
			return r.provisionSession(ctx, session)
		}
	case informer.EventModified:
		switch session.Phase {
		case PhasePending:
			return r.provisionSession(ctx, session)
		case PhaseStopping:
			return r.deprovisionSession(ctx, session, PhaseStopped)
		}
	case informer.EventDeleted:
		return r.cleanupSession(ctx, session)
	}
	return nil
}

func (r *SimpleKubeReconciler) provisionSession(ctx context.Context, session types.Session) error {
	if session.ProjectID == "" {
		return fmt.Errorf("session %s has no project_id; refusing to provision", session.ID)
	}

	sdk, err := r.factory.ForProject(session.ProjectID)
	if err != nil {
		return fmt.Errorf("session %s: creating SDK client for project %s: %w", session.ID, session.ProjectID, err)
	}
	if _, err := sdk.Projects().Get(ctx, session.ProjectID); err != nil {
		return fmt.Errorf("session %s: project %s not found in API server; refusing to provision: %w", session.ID, session.ProjectID, err)
	}

	namespace := namespaceForSession(session)

	r.logger.Info().Str("session_id", session.ID).Str("namespace", namespace).Msg("provisioning session")

	if err := r.ensureNamespaceExists(ctx, namespace, session); err != nil {
		return err
	}

	sessionLabel := sessionLabelSelector(session.ID)

	if err := r.ensureSecret(ctx, namespace, session, sessionLabel); err != nil {
		return fmt.Errorf("ensuring secret: %w", err)
	}

	if r.cfg.VertexEnabled {
		if err := r.ensureVertexSecret(ctx, namespace); err != nil {
			return fmt.Errorf("ensuring vertex secret: %w", err)
		}
	}

	if err := r.ensureServiceAccount(ctx, namespace, session, sessionLabel); err != nil {
		return fmt.Errorf("ensuring service account: %w", err)
	}

	if err := r.ensurePod(ctx, namespace, session, sessionLabel); err != nil {
		return fmt.Errorf("ensuring pod: %w", err)
	}

	if err := r.ensureService(ctx, namespace, session, sessionLabel); err != nil {
		return fmt.Errorf("ensuring service: %w", err)
	}

	r.updateSessionPhase(ctx, session, PhaseRunning)
	return nil
}

func (r *SimpleKubeReconciler) deprovisionSession(ctx context.Context, session types.Session, nextPhase string) error {
	namespace := namespaceForSession(session)
	selector := sessionLabelSelector(session.ID)

	r.logger.Info().Str("session_id", session.ID).Str("namespace", namespace).Msg("deprovisioning session")

	if err := r.kube.DeletePodsByLabel(ctx, namespace, selector); err != nil && !k8serrors.IsNotFound(err) {
		r.logger.Warn().Err(err).Msg("deleting pods")
	}

	r.updateSessionPhase(ctx, session, nextPhase)
	return nil
}

func (r *SimpleKubeReconciler) cleanupSession(ctx context.Context, session types.Session) error {
	namespace := namespaceForSession(session)
	selector := sessionLabelSelector(session.ID)

	r.logger.Info().Str("session_id", session.ID).Str("namespace", namespace).Msg("cleaning up session resources")

	if err := r.kube.DeletePodsByLabel(ctx, namespace, selector); err != nil && !k8serrors.IsNotFound(err) {
		r.logger.Warn().Err(err).Msg("deleting pods")
	}
	if err := r.kube.DeleteSecretsByLabel(ctx, namespace, selector); err != nil && !k8serrors.IsNotFound(err) {
		r.logger.Warn().Err(err).Msg("deleting secrets")
	}
	if err := r.kube.DeleteServiceAccountsByLabel(ctx, namespace, selector); err != nil && !k8serrors.IsNotFound(err) {
		r.logger.Warn().Err(err).Msg("deleting service accounts")
	}
	if err := r.kube.DeleteServicesByLabel(ctx, namespace, selector); err != nil && !k8serrors.IsNotFound(err) {
		r.logger.Warn().Err(err).Msg("deleting services")
	}

	return nil
}

func (r *SimpleKubeReconciler) ensureService(ctx context.Context, namespace string, session types.Session, labelSelector string) error {
	name := serviceName(session.ID)

	if _, err := r.kube.GetService(ctx, namespace, name); err == nil {
		return nil
	}

	svc := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels":    sessionLabels(session.ID, session.ProjectID),
			},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{
					"ambient-code.io/session-id": session.ID,
				},
				"ports": []interface{}{
					map[string]interface{}{
						"name":       "agui",
						"port":       int64(8001),
						"targetPort": int64(8001),
						"protocol":   "TCP",
					},
				},
				"type": "ClusterIP",
			},
		},
	}

	if _, err := r.kube.CreateService(ctx, svc); err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("creating service %s: %w", name, err)
	}

	r.logger.Debug().Str("service", name).Str("namespace", namespace).Msg("runner service created")
	return nil
}

func (r *SimpleKubeReconciler) ensureNamespaceExists(ctx context.Context, namespace string, session types.Session) error {
	if _, err := r.kube.GetNamespace(ctx, namespace); err == nil {
		return nil
	}

	ns := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": namespace,
				"labels": map[string]interface{}{
					LabelManaged:   "true",
					LabelProjectID: session.ProjectID,
					LabelManagedBy: "ambient-control-plane",
				},
			},
		},
	}

	if _, err := r.kube.CreateNamespace(ctx, ns); err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("creating namespace %s: %w", namespace, err)
	}

	r.logger.Info().Str("namespace", namespace).Msg("namespace created for session")

	if r.cfg.RunnerImageNamespace != "" {
		if err := r.ensureImagePullAccess(ctx, namespace); err != nil {
			r.logger.Warn().Err(err).Str("namespace", namespace).Msg("failed to grant image pull access")
		}
	}

	return nil
}

func (r *SimpleKubeReconciler) ensureImagePullAccess(ctx context.Context, namespace string) error {
	name := "ambient-image-puller"
	rb := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "RoleBinding",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": r.cfg.RunnerImageNamespace,
			},
			"roleRef": map[string]interface{}{
				"apiGroup": "rbac.authorization.k8s.io",
				"kind":     "ClusterRole",
				"name":     "system:image-puller",
			},
			"subjects": []interface{}{
				map[string]interface{}{
					"apiGroup": "rbac.authorization.k8s.io",
					"kind":     "Group",
					"name":     fmt.Sprintf("system:serviceaccounts:%s", namespace),
				},
			},
		},
	}
	if _, err := r.kube.CreateRoleBinding(ctx, r.cfg.RunnerImageNamespace, rb); err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("creating image-puller rolebinding in %s for %s: %w", r.cfg.RunnerImageNamespace, namespace, err)
	}
	r.logger.Debug().Str("namespace", namespace).Str("image_namespace", r.cfg.RunnerImageNamespace).Msg("image pull access granted")
	return nil
}

func (r *SimpleKubeReconciler) ensureSecret(ctx context.Context, namespace string, session types.Session, labelSelector string) error {
	name := secretName(session.ID)

	if _, err := r.kube.GetSecret(ctx, namespace, name); err == nil {
		return nil
	}

	token := r.factory.Token()

	secret := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels":    sessionLabels(session.ID, session.ProjectID),
			},
			"stringData": map[string]interface{}{
				"api-token": token,
			},
		},
	}

	if _, err := r.kube.CreateSecret(ctx, secret); err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("creating secret %s: %w", name, err)
	}

	r.logger.Debug().Str("secret", name).Str("namespace", namespace).Msg("secret created")
	return nil
}

func (r *SimpleKubeReconciler) ensureServiceAccount(ctx context.Context, namespace string, session types.Session, labelSelector string) error {
	name := serviceAccountName(session.ID)

	if _, err := r.kube.GetServiceAccount(ctx, namespace, name); err == nil {
		return nil
	}

	sa := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ServiceAccount",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels":    sessionLabels(session.ID, session.ProjectID),
			},
			"automountServiceAccountToken": false,
		},
	}

	if _, err := r.kube.CreateServiceAccount(ctx, sa); err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("creating service account %s: %w", name, err)
	}

	r.logger.Debug().Str("service_account", name).Str("namespace", namespace).Msg("service account created")
	return nil
}

func (r *SimpleKubeReconciler) ensurePod(ctx context.Context, namespace string, session types.Session, labelSelector string) error {
	name := podName(session.ID)

	if _, err := r.kube.GetPod(ctx, namespace, name); err == nil {
		r.logger.Debug().Str("pod", name).Msg("pod already exists")
		return nil
	}

	saName := serviceAccountName(session.ID)
	secretName := secretName(session.ID)

	runnerImage := r.cfg.RunnerImage
	imagePullPolicy := "Always"
	if strings.HasPrefix(runnerImage, "localhost/") {
		imagePullPolicy = "IfNotPresent"
	}

	labels := sessionLabels(session.ID, session.ProjectID)

	pod := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels":    labels,
				"annotations": map[string]interface{}{
					"ambient-code.io/session-id":   session.ID,
					"ambient-code.io/session-name": session.Name,
				},
			},
			"spec": map[string]interface{}{
				"serviceAccountName":            saName,
				"automountServiceAccountToken":  false,
				"restartPolicy":                 "Never",
				"terminationGracePeriodSeconds": int64(60),
				"volumes":                       r.buildVolumes(),
				"containers": []interface{}{
					map[string]interface{}{
						"name":            "ambient-code-runner",
						"image":           runnerImage,
						"imagePullPolicy": imagePullPolicy,
						"ports": []interface{}{
							map[string]interface{}{
								"name":          "agui",
								"containerPort": int64(8001),
								"protocol":      "TCP",
							},
						},
						"volumeMounts": r.buildVolumeMounts(),
						"env":          r.buildEnv(session, secretName),
						"resources": map[string]interface{}{
							"requests": map[string]interface{}{
								"cpu":    "500m",
								"memory": "512Mi",
							},
							"limits": map[string]interface{}{
								"cpu":    "2000m",
								"memory": "4Gi",
							},
						},
						"securityContext": map[string]interface{}{
							"allowPrivilegeEscalation": false,
							"capabilities": map[string]interface{}{
								"drop": []interface{}{"ALL"},
							},
						},
					},
				},
			},
		},
	}

	if _, err := r.kube.CreatePod(ctx, pod); err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("creating pod %s: %w", name, err)
	}

	r.logger.Info().Str("pod", name).Str("namespace", namespace).Str("image", runnerImage).Msg("runner pod created")
	return nil
}

func (r *SimpleKubeReconciler) buildVolumes() []interface{} {
	vols := []interface{}{
		map[string]interface{}{
			"name":     "workspace",
			"emptyDir": map[string]interface{}{},
		},
		map[string]interface{}{
			"name": "service-ca",
			"configMap": map[string]interface{}{
				"name":     "openshift-service-ca.crt",
				"optional": true,
			},
		},
	}
	if r.cfg.VertexEnabled {
		vols = append(vols, map[string]interface{}{
			"name": "vertex",
			"secret": map[string]interface{}{
				"secretName": r.cfg.VertexSecretName,
			},
		})
	}
	return vols
}

func (r *SimpleKubeReconciler) buildVolumeMounts() []interface{} {
	mounts := []interface{}{
		map[string]interface{}{
			"name":      "workspace",
			"mountPath": "/workspace",
		},
		map[string]interface{}{
			"name":      "service-ca",
			"mountPath": "/etc/pki/ca-trust/extracted/pem/service-ca.crt",
			"subPath":   "service-ca.crt",
			"readOnly":  true,
		},
	}
	if r.cfg.VertexEnabled {
		mounts = append(mounts, map[string]interface{}{
			"name":      "vertex",
			"mountPath": "/app/vertex",
			"readOnly":  true,
		})
	}
	return mounts
}

func (r *SimpleKubeReconciler) ensureVertexSecret(ctx context.Context, namespace string) error {
	src, err := r.kube.GetSecret(ctx, r.cfg.VertexSecretNamespace, r.cfg.VertexSecretName)
	if err != nil {
		return fmt.Errorf("reading vertex secret %s/%s: %w", r.cfg.VertexSecretNamespace, r.cfg.VertexSecretName, err)
	}

	if _, err := r.kube.GetSecret(ctx, namespace, r.cfg.VertexSecretName); err == nil {
		return nil
	}

	data, _, _ := unstructured.NestedMap(src.Object, "data")

	dst := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]interface{}{
				"name":      r.cfg.VertexSecretName,
				"namespace": namespace,
				"labels": map[string]interface{}{
					LabelManaged:   "true",
					LabelManagedBy: "ambient-control-plane",
				},
			},
			"type": "Opaque",
			"data": data,
		},
	}

	if _, err := r.kube.CreateSecret(ctx, dst); err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("copying vertex secret to %s: %w", namespace, err)
	}

	r.logger.Debug().Str("namespace", namespace).Str("secret", r.cfg.VertexSecretName).Msg("vertex secret copied")
	return nil
}

func (r *SimpleKubeReconciler) buildEnv(session types.Session, credSecretName string) []interface{} {
	useVertex := "0"
	if r.cfg.VertexEnabled {
		useVertex = "1"
	}

	env := []interface{}{
		envVar("SESSION_ID", session.ID),
		envVar("AGENTIC_SESSION_NAME", session.Name),
		envVar("AGENTIC_SESSION_NAMESPACE", namespaceForSession(session)),
		envVar("PROJECT_NAME", session.ProjectID),
		envVar("WORKSPACE_PATH", "/workspace"),
		envVar("ARTIFACTS_DIR", "artifacts"),
		envVar("AGUI_PORT", "8001"),
		envVar("USE_AGUI", "true"),
		envVar("DEBUG", "true"),
		envVar("BACKEND_API_URL", r.cfg.BackendURL),
		envVar("USE_VERTEX", useVertex),
		envVar("CLAUDE_CODE_USE_VERTEX", useVertex),
		envVarFromSecret("BOT_TOKEN", credSecretName, "api-token"),
		envVar("AMBIENT_GRPC_URL", r.cfg.RunnerGRPCURL),
		envVar("AMBIENT_GRPC_USE_TLS", boolToStr(r.cfg.RunnerGRPCUseTLS)),
		envVar("AMBIENT_GRPC_CA_CERT_FILE", "/etc/pki/ca-trust/extracted/pem/service-ca.crt"),
		envVar("SSL_CERT_FILE", "/etc/pki/ca-trust/extracted/pem/service-ca.crt"),
		envVar("REQUESTS_CA_BUNDLE", "/etc/pki/ca-trust/extracted/pem/service-ca.crt"),
	}

	if r.cfg.AnthropicAPIKey != "" {
		env = append(env, envVar("ANTHROPIC_API_KEY", r.cfg.AnthropicAPIKey))
	}

	if r.cfg.VertexEnabled {
		env = append(env,
			envVar("ANTHROPIC_VERTEX_PROJECT_ID", r.cfg.VertexProjectID),
			envVar("CLOUD_ML_REGION", r.cfg.VertexRegion),
			envVar("GOOGLE_APPLICATION_CREDENTIALS", r.cfg.VertexCredentialsPath),
			envVar("GCE_METADATA_HOST", "metadata.invalid"),
			envVar("GCE_METADATA_TIMEOUT", "1"),
		)
	}

	if session.Prompt != "" {
		env = append(env, envVar("INITIAL_PROMPT", session.Prompt))
	}
	if session.LlmModel != "" {
		env = append(env, envVar("LLM_MODEL", session.LlmModel))
	}
	if session.LlmTemperature != 0 {
		env = append(env, envVar("LLM_TEMPERATURE", fmt.Sprintf("%g", session.LlmTemperature)))
	}
	if session.LlmMaxTokens != 0 {
		env = append(env, envVar("LLM_MAX_TOKENS", fmt.Sprintf("%d", session.LlmMaxTokens)))
	}
	if session.Timeout != 0 {
		env = append(env, envVar("TIMEOUT", fmt.Sprintf("%d", session.Timeout)))
	}
	if session.RepoURL != "" {
		env = append(env, envVar("REPOS_JSON", fmt.Sprintf(`[{"url":%q}]`, session.RepoURL)))
	}

	return env
}

func (r *SimpleKubeReconciler) updateSessionPhase(ctx context.Context, session types.Session, newPhase string) {
	if session.Phase == newPhase {
		return
	}
	if session.ProjectID == "" {
		r.logger.Debug().Str("session_id", session.ID).Msg("skipping phase update: no project_id")
		return
	}

	sdk, err := r.factory.ForProject(session.ProjectID)
	if err != nil {
		r.logger.Warn().Err(err).Str("session_id", session.ID).Msg("failed to get SDK client for phase update")
		return
	}

	patch := map[string]interface{}{"phase": newPhase}

	if newPhase == PhaseRunning && session.StartTime == nil {
		now := time.Now()
		patch["start_time"] = &now
	}
	if (newPhase == PhaseCompleted || newPhase == PhaseFailed || newPhase == PhaseStopped) && session.CompletionTime == nil {
		now := time.Now()
		patch["completion_time"] = &now
	}

	if _, err := sdk.Sessions().UpdateStatus(ctx, session.ID, patch); err != nil {
		r.logger.Warn().Err(err).Str("session_id", session.ID).Str("phase", newPhase).Msg("failed to update session phase")
		return
	}

	r.logger.Info().
		Str("session_id", session.ID).
		Str("old_phase", session.Phase).
		Str("new_phase", newPhase).
		Msg("session phase updated")
}

func sessionLabelSelector(sessionID string) string {
	return fmt.Sprintf("ambient-code.io/session-id=%s", sessionID)
}

func sessionLabels(sessionID, projectID string) map[string]interface{} {
	return map[string]interface{}{
		"ambient-code.io/session-id": sessionID,
		LabelProjectID:               projectID,
		LabelManaged:                 "true",
		LabelManagedBy:               "ambient-control-plane",
	}
}

func safeResourceName(sessionID string) string {
	return strings.ToLower(sessionID[:min(len(sessionID), 40)])
}

func serviceName(sessionID string) string {
	return fmt.Sprintf("session-%s", safeResourceName(sessionID))
}

func podName(sessionID string) string {
	return fmt.Sprintf("session-%s-runner", safeResourceName(sessionID))
}

func secretName(sessionID string) string {
	return fmt.Sprintf("session-%s-creds", safeResourceName(sessionID))
}

func serviceAccountName(sessionID string) string {
	return fmt.Sprintf("session-%s-sa", safeResourceName(sessionID))
}

func envVar(name, value string) interface{} {
	return map[string]interface{}{"name": name, "value": value}
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func envVarFromSecret(name, secretName, key string) interface{} {
	return map[string]interface{}{
		"name": name,
		"valueFrom": map[string]interface{}{
			"secretKeyRef": map[string]interface{}{
				"name": secretName,
				"key":  key,
			},
		},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
