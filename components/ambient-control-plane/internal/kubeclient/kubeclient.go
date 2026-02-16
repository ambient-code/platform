package kubeclient

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var AgenticSessionGVR = schema.GroupVersionResource{
	Group:    "vteam.ambient-code",
	Version:  "v1alpha1",
	Resource: "agenticsessions",
}

var NamespaceGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "namespaces",
}

var RoleBindingGVR = schema.GroupVersionResource{
	Group:    "rbac.authorization.k8s.io",
	Version:  "v1",
	Resource: "rolebindings",
}

type KubeClient struct {
	dynamic   dynamic.Interface
	namespace string
	logger    zerolog.Logger
}

func New(kubeconfig string, namespace string, logger zerolog.Logger) (*KubeClient, error) {
	cfg, err := buildRestConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("building kubeconfig: %w", err)
	}

	cfg.QPS = 50
	cfg.Burst = 100

	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	kc := &KubeClient{
		dynamic:   dynClient,
		namespace: namespace,
		logger:    logger.With().Str("component", "kubeclient").Logger(),
	}

	kc.logger.Info().
		Str("namespace", namespace).
		Msg("kubernetes client initialized")

	return kc, nil
}

func buildRestConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	if envPath := os.Getenv("KUBECONFIG"); envPath != "" {
		return clientcmd.BuildConfigFromFlags("", envPath)
	}

	home, _ := os.UserHomeDir()
	localPath := home + "/.kube/config"
	if _, err := os.Stat(localPath); err == nil {
		return clientcmd.BuildConfigFromFlags("", localPath)
	}

	return rest.InClusterConfig()
}

func NewFromDynamic(dynClient dynamic.Interface, namespace string, logger zerolog.Logger) *KubeClient {
	return &KubeClient{
		dynamic:   dynClient,
		namespace: namespace,
		logger:    logger.With().Str("component", "kubeclient").Logger(),
	}
}

func (kc *KubeClient) GetAgenticSession(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(AgenticSessionGVR).Namespace(kc.namespace).Get(ctx, name, metav1.GetOptions{})
}

func (kc *KubeClient) ListAgenticSessions(ctx context.Context) (*unstructured.UnstructuredList, error) {
	return kc.dynamic.Resource(AgenticSessionGVR).Namespace(kc.namespace).List(ctx, metav1.ListOptions{})
}

func (kc *KubeClient) CreateAgenticSession(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(AgenticSessionGVR).Namespace(kc.namespace).Create(ctx, obj, metav1.CreateOptions{})
}

func (kc *KubeClient) UpdateAgenticSession(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(AgenticSessionGVR).Namespace(kc.namespace).Update(ctx, obj, metav1.UpdateOptions{})
}

func (kc *KubeClient) DeleteAgenticSession(ctx context.Context, name string) error {
	return kc.dynamic.Resource(AgenticSessionGVR).Namespace(kc.namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (kc *KubeClient) Namespace() string {
	return kc.namespace
}

func (kc *KubeClient) GetNamespace(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(NamespaceGVR).Get(ctx, name, metav1.GetOptions{})
}

func (kc *KubeClient) CreateNamespace(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(NamespaceGVR).Create(ctx, obj, metav1.CreateOptions{})
}

func (kc *KubeClient) UpdateNamespace(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(NamespaceGVR).Update(ctx, obj, metav1.UpdateOptions{})
}

func (kc *KubeClient) GetRoleBinding(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(RoleBindingGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (kc *KubeClient) CreateRoleBinding(ctx context.Context, namespace string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(RoleBindingGVR).Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{})
}

func (kc *KubeClient) UpdateRoleBinding(ctx context.Context, namespace string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(RoleBindingGVR).Namespace(namespace).Update(ctx, obj, metav1.UpdateOptions{})
}

func (kc *KubeClient) DeleteRoleBinding(ctx context.Context, namespace, name string) error {
	return kc.dynamic.Resource(RoleBindingGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (kc *KubeClient) ListRoleBindings(ctx context.Context, namespace string, labelSelector string) (*unstructured.UnstructuredList, error) {
	opts := metav1.ListOptions{}
	if labelSelector != "" {
		opts.LabelSelector = labelSelector
	}
	return kc.dynamic.Resource(RoleBindingGVR).Namespace(namespace).List(ctx, opts)
}
