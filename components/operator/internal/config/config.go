// Package config provides Kubernetes client initialization and configuration management for the operator.
package config

import (
	"fmt"
	"os"
	"strconv"

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
	StateSyncImage         string
	ImagePullPolicy        corev1.PullPolicy
	S3Endpoint             string
	S3Bucket               string
	PodFSGroup             *int64
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

	// Increase QPS and Burst to avoid client-side throttling
	// Default is QPS=5, Burst=10 which is too low for operators handling concurrent sessions
	config.QPS = 100
	config.Burst = 200

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

	// Get state-sync image from environment or use default
	stateSyncImage := os.Getenv("STATE_SYNC_IMAGE")
	if stateSyncImage == "" {
		stateSyncImage = "quay.io/ambient_code/vteam_state_sync:latest"
	}

	// Get image pull policy from environment or use default
	imagePullPolicyStr := os.Getenv("IMAGE_PULL_POLICY")
	if imagePullPolicyStr == "" {
		imagePullPolicyStr = "Always"
	}
	imagePullPolicy := corev1.PullPolicy(imagePullPolicyStr)

	// Get S3 configuration from environment
	s3Endpoint := os.Getenv("S3_ENDPOINT")
	if s3Endpoint == "" {
		s3Endpoint = "https://s3.amazonaws.com"
	}

	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket == "" {
		s3Bucket = "ambient-sessions"
	}

	// Optional pod fsGroup (useful for permissive clusters like kind)
	var podFSGroup *int64
	if fsGroupStr := os.Getenv("POD_FSGROUP"); fsGroupStr != "" {
		if fsGroupValue, err := strconv.ParseInt(fsGroupStr, 10, 64); err == nil {
			podFSGroup = &fsGroupValue
		}
	}

	return &Config{
		Namespace:              namespace,
		BackendNamespace:       backendNamespace,
		AmbientCodeRunnerImage: ambientCodeRunnerImage,
		StateSyncImage:         stateSyncImage,
		ImagePullPolicy:        imagePullPolicy,
		S3Endpoint:             s3Endpoint,
		S3Bucket:               s3Bucket,
		PodFSGroup:             podFSGroup,
	}
}
