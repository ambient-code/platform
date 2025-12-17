// Package config provides Kubernetes client initialization and configuration management for the operator.
package config

import (
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Package-level variables (exported for use by handlers and services)
var (
	K8sClient     kubernetes.Interface
	DynamicClient dynamic.Interface
)

// Config holds the operator configuration
type Config struct {
	Namespace              string
	BackendNamespace       string
	AmbientCodeRunnerImage string
	ContentServiceImage    string
	ImagePullPolicy        corev1.PullPolicy
	// Runner type to image mappings for pluggable agents
	RunnerImages map[string]string
}

// InitK8sClients initializes the Kubernetes clients
func InitK8sClients() error {
	var config *rest.Config
	var err error

	// Try in-cluster config first
	if config, err = rest.InClusterConfig(); err != nil {
		// If in-cluster config fails, try kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = fmt.Sprintf("%s/.kube/config", os.Getenv("HOME"))
		}

		if config, err = clientcmd.BuildConfigFromFlags("", kubeconfig); err != nil {
			return fmt.Errorf("failed to create Kubernetes config: %v", err)
		}
	}

	// Create standard Kubernetes client
	K8sClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	// Create dynamic client for custom resources
	DynamicClient, err = dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %v", err)
	}

	return nil
}

// LoadConfig loads the operator configuration from environment variables
func LoadConfig() *Config {
	// Get namespace from environment or use default
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	// Get backend namespace from environment or use operator namespace
	backendNamespace := os.Getenv("BACKEND_NAMESPACE")
	if backendNamespace == "" {
		backendNamespace = namespace // Default to same namespace as operator
	}

	// Get ambient-code runner image from environment or use default
	ambientCodeRunnerImage := os.Getenv("AMBIENT_CODE_RUNNER_IMAGE")
	if ambientCodeRunnerImage == "" {
		ambientCodeRunnerImage = "quay.io/ambient_code/vteam_claude_runner:latest"
	}

	// Image for per-namespace content service (defaults to backend image)
	contentServiceImage := os.Getenv("CONTENT_SERVICE_IMAGE")
	if contentServiceImage == "" {
		contentServiceImage = "quay.io/ambient_code/vteam_backend:latest"
	}

	// Get image pull policy from environment or use default
	imagePullPolicyStr := os.Getenv("IMAGE_PULL_POLICY")
	if imagePullPolicyStr == "" {
		imagePullPolicyStr = "Always"
	}
	imagePullPolicy := corev1.PullPolicy(imagePullPolicyStr)

	// Initialize runner image mappings for pluggable agents
	runnerImages := make(map[string]string)
	runnerImages["claude-sdk"] = ambientCodeRunnerImage // Default Claude SDK runner

	// Load additional runner images from environment variables
	if langGraphImage := os.Getenv("LANGGRAPH_RUNNER_IMAGE"); langGraphImage != "" {
		runnerImages["langgraph"] = langGraphImage
	}
	if crewAIImage := os.Getenv("CREWAI_RUNNER_IMAGE"); crewAIImage != "" {
		runnerImages["crewai"] = crewAIImage
	}
	if customImage := os.Getenv("CUSTOM_RUNNER_IMAGE"); customImage != "" {
		runnerImages["custom"] = customImage
	}

	return &Config{
		Namespace:              namespace,
		BackendNamespace:       backendNamespace,
		AmbientCodeRunnerImage: ambientCodeRunnerImage,
		ContentServiceImage:    contentServiceImage,
		ImagePullPolicy:        imagePullPolicy,
		RunnerImages:           runnerImages,
	}
}

// GetRunnerImage returns the appropriate runner image based on runnerConfig
// If custom image is specified, it takes precedence
// Otherwise, looks up the runner type in the image registry
// Falls back to default Claude SDK runner if not found
func (c *Config) GetRunnerImage(runnerType string, customImage string) string {
	// Custom image override takes precedence
	if customImage != "" {
		return customImage
	}

	// Look up runner type in registry
	if runnerType != "" {
		if image, ok := c.RunnerImages[runnerType]; ok {
			return image
		}
	}

	// Default to Claude SDK runner
	return c.AmbientCodeRunnerImage
}
