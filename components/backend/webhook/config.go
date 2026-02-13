package webhook

import (
	"context"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// WebhookSecretName is the name of the Kubernetes Secret containing the GitHub webhook secret
	WebhookSecretName = "github-webhook-secret"
	// WebhookSecretKey is the key within the Secret that contains the HMAC secret value
	WebhookSecretKey = "secret"
	// WebhookSecretNamespaceEnv is the environment variable for the webhook secret namespace
	WebhookSecretNamespaceEnv = "WEBHOOK_SECRET_NAMESPACE"
	// DefaultNamespace is the default Kubernetes namespace if not specified
	DefaultNamespace = "ambient-code"
)

// Config holds the webhook handler configuration
type Config struct {
	WebhookSecret string
}

// LoadConfig loads the webhook configuration from Kubernetes Secret (FR-008)
// It first tries to load from WEBHOOK_SECRET environment variable (for local dev)
// If not present, it loads from Kubernetes Secret in the specified namespace
func LoadConfig(ctx context.Context, k8sClient kubernetes.Interface) (*Config, error) {
	// Try environment variable first (for local development)
	if secret := os.Getenv("WEBHOOK_SECRET"); secret != "" {
		return &Config{
			WebhookSecret: secret,
		}, nil
	}

	// Load from Kubernetes Secret
	namespace := os.Getenv(WebhookSecretNamespaceEnv)
	if namespace == "" {
		namespace = DefaultNamespace
	}

	secretClient := k8sClient.CoreV1().Secrets(namespace)
	secret, err := secretClient.Get(ctx, WebhookSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to load webhook secret from namespace %s: %w", namespace, err)
	}

	webhookSecretBytes, ok := secret.Data[WebhookSecretKey]
	if !ok {
		return nil, fmt.Errorf("webhook secret key '%s' not found in Secret %s/%s", WebhookSecretKey, namespace, WebhookSecretName)
	}

	if len(webhookSecretBytes) == 0 {
		return nil, fmt.Errorf("webhook secret is empty in Secret %s/%s", namespace, WebhookSecretName)
	}

	return &Config{
		WebhookSecret: string(webhookSecretBytes),
	}, nil
}
