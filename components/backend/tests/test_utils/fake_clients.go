package test_utils

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

// FakeClientSet provides a wrapper around fake clients that implements the expected interfaces
type FakeClientSet struct {
	K8sClient     kubernetes.Interface
	DynamicClient dynamic.Interface
}

// NewFakeClientSet creates a new fake client set for testing
func NewFakeClientSet() *FakeClientSet {
	scheme := runtime.NewScheme()

	return &FakeClientSet{
		K8sClient:     k8sfake.NewSimpleClientset(),
		DynamicClient: fake.NewSimpleDynamicClient(scheme),
	}
}

// GetK8sClient returns the Kubernetes client interface
func (f *FakeClientSet) GetK8sClient() kubernetes.Interface {
	return f.K8sClient
}

// GetDynamicClient returns the dynamic client interface
func (f *FakeClientSet) GetDynamicClient() dynamic.Interface {
	return f.DynamicClient
}

// MockK8sClientsForRequest provides a mock implementation of GetK8sClientsForRequest
type MockK8sClientsForRequest func(c interface{}) (kubernetes.Interface, dynamic.Interface)

// MockValidateSecretAccess provides a mock implementation of ValidateSecretAccess
type MockValidateSecretAccess func(ctx context.Context, clientset kubernetes.Interface, project, verb string) error

// CreateAgenticSessionInFakeClient creates a test AgenticSession in the fake dynamic client
func CreateAgenticSessionInFakeClient(dynamicClient dynamic.Interface, namespace, name string, spec map[string]interface{}) error {
	agenticSessionGVR := schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1alpha1",
		Resource: "agenticsessions",
	}

	sessionObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": spec,
		},
	}

	_, err := dynamicClient.Resource(agenticSessionGVR).Namespace(namespace).Create(
		context.Background(), sessionObj, v1.CreateOptions{})
	return err
}
