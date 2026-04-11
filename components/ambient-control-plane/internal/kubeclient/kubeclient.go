package kubeclient

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

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

var PodGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "pods",
}

var ServiceGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "services",
}

var SecretGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "secrets",
}

var ServiceAccountGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "serviceaccounts",
}

var RoleGVR = schema.GroupVersionResource{
	Group:    "rbac.authorization.k8s.io",
	Version:  "v1",
	Resource: "roles",
}

var NetworkPolicyGVR = schema.GroupVersionResource{
	Group:    "networking.k8s.io",
	Version:  "v1",
	Resource: "networkpolicies",
}

type KubeClient struct {
	dynamic dynamic.Interface
	logger  zerolog.Logger
}

func New(kubeconfig string, logger zerolog.Logger) (*KubeClient, error) {
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
		dynamic: dynClient,
		logger:  logger.With().Str("component", "kubeclient").Logger(),
	}

	kc.logger.Info().Msg("kubernetes client initialized")

	return kc, nil
}

func buildRestConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	home, _ := os.UserHomeDir()
	localPath := home + "/.kube/config"
	if _, err := os.Stat(localPath); err == nil {
		return clientcmd.BuildConfigFromFlags("", localPath)
	}

	return rest.InClusterConfig()
}

func NewFromTokenFile(tokenFile string, logger zerolog.Logger) (*KubeClient, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("building in-cluster config for token file client: %w", err)
	}

	tokenBytes, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("reading token file %s: %w", tokenFile, err)
	}

	cfg.BearerToken = strings.TrimSpace(string(tokenBytes))
	cfg.BearerTokenFile = ""
	cfg.QPS = 50
	cfg.Burst = 100

	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client from token file: %w", err)
	}

	kc := &KubeClient{
		dynamic: dynClient,
		logger:  logger.With().Str("component", "kubeclient-project").Logger(),
	}

	kc.logger.Info().Str("token_file", tokenFile).Msg("project kubernetes client initialized")

	return kc, nil
}

func NewFromDynamic(dynClient dynamic.Interface, logger zerolog.Logger) *KubeClient {
	return &KubeClient{
		dynamic: dynClient,
		logger:  logger.With().Str("component", "kubeclient").Logger(),
	}
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

func (kc *KubeClient) DeleteNamespace(ctx context.Context, name string) error {
	return kc.dynamic.Resource(NamespaceGVR).Delete(ctx, name, metav1.DeleteOptions{})
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

// Pod operations
func (kc *KubeClient) GetPod(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(PodGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (kc *KubeClient) CreatePod(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(PodGVR).Namespace(obj.GetNamespace()).Create(ctx, obj, metav1.CreateOptions{})
}

func (kc *KubeClient) DeletePod(ctx context.Context, namespace, name string, opts *metav1.DeleteOptions) error {
	if opts == nil {
		opts = &metav1.DeleteOptions{}
	}
	return kc.dynamic.Resource(PodGVR).Namespace(namespace).Delete(ctx, name, *opts)
}

func (kc *KubeClient) DeletePodsByLabel(ctx context.Context, namespace, labelSelector string) error {
	return kc.deleteCollectionWithFallback(ctx, PodGVR, namespace, labelSelector)
}

// Service operations
func (kc *KubeClient) GetService(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(ServiceGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (kc *KubeClient) CreateService(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(ServiceGVR).Namespace(obj.GetNamespace()).Create(ctx, obj, metav1.CreateOptions{})
}

func (kc *KubeClient) DeleteServicesByLabel(ctx context.Context, namespace, labelSelector string) error {
	return kc.deleteCollectionWithFallback(ctx, ServiceGVR, namespace, labelSelector)
}

// Secret operations
func (kc *KubeClient) GetSecret(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(SecretGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (kc *KubeClient) CreateSecret(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(SecretGVR).Namespace(obj.GetNamespace()).Create(ctx, obj, metav1.CreateOptions{})
}

func (kc *KubeClient) UpdateSecret(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(SecretGVR).Namespace(obj.GetNamespace()).Update(ctx, obj, metav1.UpdateOptions{})
}

func (kc *KubeClient) DeleteSecretsByLabel(ctx context.Context, namespace, labelSelector string) error {
	return kc.deleteCollectionWithFallback(ctx, SecretGVR, namespace, labelSelector)
}

// ServiceAccount operations
func (kc *KubeClient) GetServiceAccount(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(ServiceAccountGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (kc *KubeClient) CreateServiceAccount(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(ServiceAccountGVR).Namespace(obj.GetNamespace()).Create(ctx, obj, metav1.CreateOptions{})
}

func (kc *KubeClient) DeleteServiceAccountsByLabel(ctx context.Context, namespace, labelSelector string) error {
	return kc.deleteCollectionWithFallback(ctx, ServiceAccountGVR, namespace, labelSelector)
}

// Role operations
func (kc *KubeClient) GetRole(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(RoleGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (kc *KubeClient) CreateRole(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(RoleGVR).Namespace(obj.GetNamespace()).Create(ctx, obj, metav1.CreateOptions{})
}

func (kc *KubeClient) DeleteRolesByLabel(ctx context.Context, namespace, labelSelector string) error {
	return kc.deleteCollectionWithFallback(ctx, RoleGVR, namespace, labelSelector)
}

func (kc *KubeClient) DeleteRoleBindingsByLabel(ctx context.Context, namespace, labelSelector string) error {
	return kc.deleteCollectionWithFallback(ctx, RoleBindingGVR, namespace, labelSelector)
}

func (kc *KubeClient) deleteCollectionWithFallback(ctx context.Context, gvr schema.GroupVersionResource, namespace, labelSelector string) error {
	err := kc.dynamic.Resource(gvr).Namespace(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: labelSelector})
	if err == nil {
		return nil
	}
	if !k8serrors.IsForbidden(err) {
		return err
	}

	kc.logger.Warn().Str("resource", gvr.Resource).Str("namespace", namespace).Msg("deletecollection forbidden, falling back to list+delete")

	list, listErr := kc.dynamic.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if listErr != nil {
		return fmt.Errorf("fallback list %s: %w", gvr.Resource, listErr)
	}

	var lastErr error
	for i := range list.Items {
		name := list.Items[i].GetName()
		if delErr := kc.dynamic.Resource(gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{}); delErr != nil && !k8serrors.IsNotFound(delErr) {
			kc.logger.Warn().Err(delErr).Str("resource", gvr.Resource).Str("name", name).Msg("fallback delete failed")
			lastErr = delErr
		}
	}
	return lastErr
}

func (kc *KubeClient) GetNetworkPolicy(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(NetworkPolicyGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (kc *KubeClient) CreateNetworkPolicy(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(NetworkPolicyGVR).Namespace(obj.GetNamespace()).Create(ctx, obj, metav1.CreateOptions{})
}

func (kc *KubeClient) GetResource(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (kc *KubeClient) CreateResource(ctx context.Context, gvr schema.GroupVersionResource, namespace string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return kc.dynamic.Resource(gvr).Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{})
}
