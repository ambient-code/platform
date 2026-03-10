package server

import (
	"fmt"
	"os"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	K8sClient       *kubernetes.Clientset
	DynamicClient   dynamic.Interface
	Namespace       string
	StateBaseDir    string
	PvcBaseDir      string
	BaseKubeConfig  *rest.Config
	OperatorImage   string
	ImagePullPolicy string
)

// InitK8sClients initializes Kubernetes clients and configuration
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

	// Increase rate limits for parallel operations (default is QPS=5, Burst=10)
	// This is needed for parallel SSAR checks when listing projects
	config.QPS = 100
	config.Burst = 200

	// Create standard Kubernetes client
	K8sClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	// Create dynamic client for CRD operations
	DynamicClient, err = dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %v", err)
	}

	// Save base config for per-request impersonation/user-token clients
	BaseKubeConfig = config

	return nil
}

// InitConfig initializes configuration from environment variables
func InitConfig() {
	// Get namespace from environment or use default
	Namespace = os.Getenv("NAMESPACE")
	if Namespace == "" {
		Namespace = "default"
	}

	// Get state storage base directory
	StateBaseDir = os.Getenv("STATE_BASE_DIR")
	if StateBaseDir == "" {
		StateBaseDir = "/workspace"
	}

	// Get PVC base directory for RFE workspaces
	PvcBaseDir = os.Getenv("PVC_BASE_DIR")
	if PvcBaseDir == "" {
		PvcBaseDir = "/workspace"
	}

	// Get operator image for scheduled session trigger jobs
	OperatorImage = os.Getenv("OPERATOR_IMAGE")
	if OperatorImage == "" {
		OperatorImage = "quay.io/ambient_code/vteam_operator:latest"
	}

	// Get image pull policy (used for trigger containers in scheduled sessions)
	ImagePullPolicy = os.Getenv("IMAGE_PULL_POLICY")
	if ImagePullPolicy == "" {
		ImagePullPolicy = "IfNotPresent"
	}
}
