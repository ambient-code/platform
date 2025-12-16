// Package test_utils provides Kubernetes utilities for testing
package test_utils

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ambient-code-backend/k8s"
	"ambient-code-backend/tests/config"

	. "github.com/onsi/gomega"
	authnv1 "k8s.io/api/authentication/v1"
	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// boolPtr returns a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}

// mockSSARAction implements k8stesting.Action for SSAR checks in Create reactor
type mockSSARAction struct {
	resource  schema.GroupVersionResource
	namespace string
	verb      string
}

func (m *mockSSARAction) GetVerb() string {
	return m.verb
}

func (m *mockSSARAction) GetResource() schema.GroupVersionResource {
	return m.resource
}

func (m *mockSSARAction) GetSubresource() string {
	return ""
}

func (m *mockSSARAction) GetNamespace() string {
	return m.namespace
}

func (m *mockSSARAction) GetName() string {
	return ""
}

func (m *mockSSARAction) Matches(verb, resource string) bool {
	return m.verb == verb && m.resource.Resource == resource
}

func (m *mockSSARAction) DeepCopy() k8stesting.Action {
	return &mockSSARAction{
		resource:  m.resource,
		namespace: m.namespace,
		verb:      m.verb,
	}
}

// K8sTestUtils provides utilities for Kubernetes testing
type K8sTestUtils struct {
	K8sClient     kubernetes.Interface
	DynamicClient dynamic.Interface
	Namespace     string
	scheme        *runtime.Scheme
	// SSARAllowedFunc allows tests to customize SSAR behavior
	// If nil, defaults to returning allowed=true
	SSARAllowedFunc func(action k8stesting.Action) bool
}

// NewK8sTestUtils creates new Kubernetes test utilities
func NewK8sTestUtils(useRealCluster bool, namespace string) *K8sTestUtils {
	utils := &K8sTestUtils{
		Namespace: namespace,
		scheme:    runtime.NewScheme(),
	}

	// Register custom resources with the scheme for fake dynamic client
	registerCustomResources(utils.scheme)

	var fakeClient *k8sfake.Clientset
	if useRealCluster {
		// TODO: Implement real cluster client creation
		// For now, use fake clients even when real cluster is requested
		fakeClient = k8sfake.NewSimpleClientset()
		utils.DynamicClient = fake.NewSimpleDynamicClientWithCustomListKinds(utils.scheme, getCustomListKinds())
	} else {
		fakeClient = k8sfake.NewSimpleClientset()
		baseDynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(utils.scheme, getCustomListKinds())
		utils.DynamicClient = &TypeSafeDynamicClient{base: baseDynamicClient}
	}

	// Configure fake client to return allowed=true for all SelfSubjectAccessReview calls
	// This allows RBAC checks to pass in tests
	// Tests can override SSARAllowedFunc to customize behavior
	fakeClient.PrependReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		ssar := action.(k8stesting.CreateAction).GetObject().(*authv1.SelfSubjectAccessReview)
		allowed := true
		if utils.SSARAllowedFunc != nil {
			allowed = utils.SSARAllowedFunc(action)
		}
		ssar.Status = authv1.SubjectAccessReviewStatus{
			Allowed: allowed,
			Reason:  "Mocked for tests",
		}
		return true, ssar, nil
	})

	// Configure fake client to check SSAR before allowing Create operations on RBAC-protected resources
	// This simulates Kubernetes RBAC enforcement for rolebindings and namespaces
	fakeClient.PrependReactor("create", "*", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		createAction, ok := action.(k8stesting.CreateAction)
		if !ok {
			return false, nil, nil
		}

		resource := createAction.GetResource()

		// Only check SSAR for RBAC-protected resources
		if resource.Resource == "rolebindings" || resource.Resource == "namespaces" {
			// Check if this is a test setup operation (has test-framework label)
			// Test setup RoleBindings should always be allowed to bypass SSAR checks
			obj := createAction.GetObject()
			isTestSetup := false
			if objMeta, ok := obj.(metav1.Object); ok {
				labels := objMeta.GetLabels()
				if labels != nil && labels["test-framework"] == "ambient-code-backend" {
					// This is a test setup operation, allow it to bypass SSAR
					isTestSetup = true
				}
			}

			// For test setup operations, skip SSAR check
			if !isTestSetup {
				// For handler operations, check SSAR
				mockSSARAction := &mockSSARAction{
					resource:  resource,
					namespace: createAction.GetNamespace(),
					verb:      "create",
				}

				// Check if SSAR would allow this operation
				allowed := true
				if utils.SSARAllowedFunc != nil {
					allowed = utils.SSARAllowedFunc(mockSSARAction)
				}

				// If not allowed, return Forbidden error to simulate Kubernetes RBAC rejection
				if !allowed {
					// Extract name from the object if possible
					name := ""
					if objMeta, ok := obj.(metav1.Object); ok {
						name = objMeta.GetName()
					}

					// Return handled=true with error to prevent the creation
					return true, nil, errors.NewForbidden(
						schema.GroupResource{Group: resource.Group, Resource: resource.Resource},
						name,
						fmt.Errorf("insufficient permissions to create %s", resource.Resource),
					)
				}
			}
		}

		// Allow the operation to proceed
		return false, nil, nil
	})

	utils.K8sClient = fakeClient

	return utils
}

// registerCustomResources registers our custom resources with the scheme
func registerCustomResources(scheme *runtime.Scheme) {
	// Register the custom resources from our k8s package
	agenticSessionGVK := schema.GroupVersionKind{
		Group:   "vteam.ambient-code",
		Version: "v1alpha1",
		Kind:    "AgenticSession",
	}

	projectSettingsGVK := schema.GroupVersionKind{
		Group:   "vteam.ambient-code",
		Version: "v1alpha1",
		Kind:    "ProjectSettings",
	}

	// Register the types with the scheme
	scheme.AddKnownTypeWithName(agenticSessionGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(projectSettingsGVK, &unstructured.Unstructured{})

	// Register the list types
	agenticSessionListGVK := schema.GroupVersionKind{
		Group:   "vteam.ambient-code",
		Version: "v1alpha1",
		Kind:    "AgenticSessionList",
	}

	projectSettingsListGVK := schema.GroupVersionKind{
		Group:   "vteam.ambient-code",
		Version: "v1alpha1",
		Kind:    "ProjectSettingsList",
	}

	scheme.AddKnownTypeWithName(agenticSessionListGVK, &unstructured.UnstructuredList{})
	scheme.AddKnownTypeWithName(projectSettingsListGVK, &unstructured.UnstructuredList{})
}

// getCustomListKinds returns the mapping of resource to list kind for our custom resources
func getCustomListKinds() map[schema.GroupVersionResource]string {
	return map[schema.GroupVersionResource]string{
		k8s.GetAgenticSessionV1Alpha1Resource(): "AgenticSessionList",
		k8s.GetProjectSettingsResource():        "ProjectSettingsList",
	}
}

// CreateNamespace creates a test namespace
func (k *K8sTestUtils) CreateNamespace(ctx context.Context, name string) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"test-framework": "ambient-code-backend",
				"created-by":     "unit-tests",
			},
		},
	}

	_, err := k.K8sClient.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil // Namespace already exists, which is fine for tests
	}
	return err
}

// DeleteNamespace deletes a test namespace
func (k *K8sTestUtils) DeleteNamespace(ctx context.Context, name string) error {
	policy := metav1.DeletePropagationForeground
	err := k.K8sClient.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &policy,
	})
	if errors.IsNotFound(err) {
		return nil // Namespace doesn't exist, which is fine
	}
	return err
}

// CreateSecret creates a test secret
func (k *K8sTestUtils) CreateSecret(ctx context.Context, namespace, name string, data map[string][]byte) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"test-framework": "ambient-code-backend",
			},
		},
		Data: data,
		Type: corev1.SecretTypeOpaque,
	}

	createdSecret, err := k.K8sClient.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred(), "Failed to create test secret")
	return createdSecret
}

// GetSecret retrieves a secret
func (k *K8sTestUtils) GetSecret(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	return k.K8sClient.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
}

// DeleteSecret deletes a secret
func (k *K8sTestUtils) DeleteSecret(ctx context.Context, namespace, name string) error {
	err := k.K8sClient.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

// CreateConfigMap creates a test config map
func (k *K8sTestUtils) CreateConfigMap(ctx context.Context, namespace, name string, data map[string]string) *corev1.ConfigMap {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"test-framework": "ambient-code-backend",
			},
		},
		Data: data,
	}

	createdConfigMap, err := k.K8sClient.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred(), "Failed to create test config map")
	return createdConfigMap
}

// CreateCustomResource creates a custom resource using dynamic client
// Optionally accepts an owner object to set OwnerReferences for automatic cleanup
func (k *K8sTestUtils) CreateCustomResource(ctx context.Context, gvr schema.GroupVersionResource, namespace string, obj *unstructured.Unstructured, owner ...*unstructured.Unstructured) *unstructured.Unstructured {
	obj.SetNamespace(namespace)
	if obj.GetLabels() == nil {
		obj.SetLabels(make(map[string]string))
	}
	labels := obj.GetLabels()
	labels["test-framework"] = "ambient-code-backend"
	obj.SetLabels(labels)

	// Set OwnerReferences if owner is provided (for automatic cleanup)
	if len(owner) > 0 && owner[0] != nil {
		ownerRef := metav1.OwnerReference{
			APIVersion: owner[0].GetAPIVersion(),
			Kind:       owner[0].GetKind(),
			Name:       owner[0].GetName(),
			UID:        owner[0].GetUID(),
			Controller: boolPtr(true),
		}
		ownerRefs := []metav1.OwnerReference{ownerRef}
		obj.SetOwnerReferences(ownerRefs)
	}

	created, err := k.DynamicClient.Resource(gvr).Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred(), "Failed to create custom resource")
	return created
}

// GetCustomResource retrieves a custom resource
func (k *K8sTestUtils) GetCustomResource(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) (*unstructured.Unstructured, error) {
	return k.DynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// UpdateCustomResource updates a custom resource
func (k *K8sTestUtils) UpdateCustomResource(ctx context.Context, gvr schema.GroupVersionResource, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return k.DynamicClient.Resource(gvr).Namespace(obj.GetNamespace()).Update(ctx, obj, metav1.UpdateOptions{})
}

// DeleteCustomResource deletes a custom resource
func (k *K8sTestUtils) DeleteCustomResource(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) error {
	err := k.DynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

// WaitForCustomResourceCondition waits for a custom resource to meet a condition
func (k *K8sTestUtils) WaitForCustomResourceCondition(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string, conditionFn func(*unstructured.Unstructured) bool) error {
	timeout := *config.TestTimeout
	pollInterval := 1 * time.Second

	// Use context with timeout instead of Gomega Eventually for error handling
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctxWithTimeout.Done():
			return fmt.Errorf("timeout waiting for resource condition")
		default:
			obj, err := k.GetCustomResource(ctx, gvr, namespace, name)
			if err == nil && conditionFn(obj) {
				return nil
			}
			time.Sleep(pollInterval)
		}
	}
}

// AssertResourceExists asserts that a resource exists
func (k *K8sTestUtils) AssertResourceExists(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) {
	_, err := k.GetCustomResource(ctx, gvr, namespace, name)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Resource %s/%s should exist", namespace, name))
}

// AssertResourceNotExists asserts that a resource does not exist
func (k *K8sTestUtils) AssertResourceNotExists(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) {
	_, err := k.GetCustomResource(ctx, gvr, namespace, name)
	Expect(errors.IsNotFound(err)).To(BeTrue(), fmt.Sprintf("Resource %s/%s should not exist", namespace, name))
}

// AssertResourceHasStatus asserts that a resource has the expected status
func (k *K8sTestUtils) AssertResourceHasStatus(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string, expectedStatus map[string]interface{}) {
	obj, err := k.GetCustomResource(ctx, gvr, namespace, name)
	Expect(err).NotTo(HaveOccurred(), "Failed to get resource")

	status, found, err := unstructured.NestedMap(obj.Object, "status")
	Expect(err).NotTo(HaveOccurred(), "Failed to extract status")
	Expect(found).To(BeTrue(), "Resource should have status field")

	for key, expectedValue := range expectedStatus {
		actualValue, found, err := unstructured.NestedFieldNoCopy(status, strings.Split(key, ".")...)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to get status field %s", key))
		Expect(found).To(BeTrue(), fmt.Sprintf("Status field %s should exist", key))
		Expect(actualValue).To(Equal(expectedValue), fmt.Sprintf("Status field %s should equal %v", key, expectedValue))
	}
}

// CleanupTestResources cleans up all test resources in the namespace
func (k *K8sTestUtils) CleanupTestResources(ctx context.Context, namespace string) {
	// Delete all secrets with test label
	secretList, err := k.K8sClient.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "test-framework=ambient-code-backend",
	})
	if err == nil {
		for _, secret := range secretList.Items {
			_ = k.DeleteSecret(ctx, namespace, secret.Name)
		}
	}

	// Delete all config maps with test label
	configMapList, err := k.K8sClient.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "test-framework=ambient-code-backend",
	})
	if err == nil {
		for _, cm := range configMapList.Items {
			_ = k.K8sClient.CoreV1().ConfigMaps(namespace).Delete(ctx, cm.Name, metav1.DeleteOptions{})
		}
	}
}

// MockK8sError creates mock Kubernetes errors for testing error handling
type MockK8sError struct {
	StatusCode int
	Reason     metav1.StatusReason
	Message    string
}

func (e *MockK8sError) Error() string {
	return e.Message
}

func (e *MockK8sError) Status() metav1.Status {
	return metav1.Status{
		Code:    int32(e.StatusCode),
		Reason:  e.Reason,
		Message: e.Message,
	}
}

// NewNotFoundError creates a mock "not found" error
func NewNotFoundError(resource, name string) *MockK8sError {
	return &MockK8sError{
		StatusCode: 404,
		Reason:     metav1.StatusReasonNotFound,
		Message:    fmt.Sprintf("%s %q not found", resource, name),
	}
}

// NewForbiddenError creates a mock "forbidden" error
func NewForbiddenError(resource, name string) *MockK8sError {
	return &MockK8sError{
		StatusCode: 403,
		Reason:     metav1.StatusReasonForbidden,
		Message:    fmt.Sprintf("access to %s %q is forbidden", resource, name),
	}
}

// TypeSafeDynamicClient wraps the fake dynamic client to handle type conversion
// for unstructured objects before they undergo DeepCopy operations
type TypeSafeDynamicClient struct {
	base dynamic.Interface
}

// Resource returns a TypeSafeNamespaceableResourceInterface for the given GroupVersionResource
func (t *TypeSafeDynamicClient) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	return &TypeSafeNamespaceableResourceInterface{
		base: t.base.Resource(resource),
		gvr:  resource,
	}
}

// TypeSafeNamespaceableResourceInterface wraps NamespaceableResourceInterface
type TypeSafeNamespaceableResourceInterface struct {
	base dynamic.NamespaceableResourceInterface
	gvr  schema.GroupVersionResource
}

// Namespace returns a TypeSafeResourceInterface for the given namespace
func (t *TypeSafeNamespaceableResourceInterface) Namespace(namespace string) dynamic.ResourceInterface {
	return &TypeSafeResourceInterface{
		base: t.base.Namespace(namespace),
		gvr:  t.gvr,
	}
}

// Apply delegates to the base implementation (not used in tests)
func (t *TypeSafeNamespaceableResourceInterface) Apply(ctx context.Context, name string, obj *unstructured.Unstructured, options metav1.ApplyOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return t.base.Apply(ctx, name, obj, options, subresources...)
}

// ApplyStatus delegates to the base implementation
func (t *TypeSafeNamespaceableResourceInterface) ApplyStatus(ctx context.Context, name string, obj *unstructured.Unstructured, options metav1.ApplyOptions) (*unstructured.Unstructured, error) {
	return t.base.ApplyStatus(ctx, name, obj, options)
}

// Create delegates to the base implementation
func (t *TypeSafeNamespaceableResourceInterface) Create(ctx context.Context, obj *unstructured.Unstructured, options metav1.CreateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return t.base.Create(ctx, obj, options, subresources...)
}

// Update delegates to the base implementation
func (t *TypeSafeNamespaceableResourceInterface) Update(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return t.base.Update(ctx, obj, options, subresources...)
}

// UpdateStatus delegates to the base implementation
func (t *TypeSafeNamespaceableResourceInterface) UpdateStatus(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	return t.base.UpdateStatus(ctx, obj, options)
}

// Delete delegates to the base implementation
func (t *TypeSafeNamespaceableResourceInterface) Delete(ctx context.Context, name string, options metav1.DeleteOptions, subresources ...string) error {
	return t.base.Delete(ctx, name, options, subresources...)
}

// DeleteCollection delegates to the base implementation
func (t *TypeSafeNamespaceableResourceInterface) DeleteCollection(ctx context.Context, options metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return t.base.DeleteCollection(ctx, options, listOptions)
}

// Get delegates to the base implementation
func (t *TypeSafeNamespaceableResourceInterface) Get(ctx context.Context, name string, options metav1.GetOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return t.base.Get(ctx, name, options, subresources...)
}

// List delegates to the base implementation
func (t *TypeSafeNamespaceableResourceInterface) List(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return t.base.List(ctx, opts)
}

// Watch delegates to the base implementation
func (t *TypeSafeNamespaceableResourceInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return t.base.Watch(ctx, opts)
}

// Patch delegates to the base implementation
func (t *TypeSafeNamespaceableResourceInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, options metav1.PatchOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return t.base.Patch(ctx, name, pt, data, options, subresources...)
}

// TypeSafeResourceInterface wraps ResourceInterface with type conversion
type TypeSafeResourceInterface struct {
	base dynamic.ResourceInterface
	gvr  schema.GroupVersionResource
}

// Apply delegates to the base implementation (not used in tests)
func (t *TypeSafeResourceInterface) Apply(ctx context.Context, name string, obj *unstructured.Unstructured, options metav1.ApplyOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return t.base.Apply(ctx, name, obj, options, subresources...)
}

// ApplyStatus delegates to the base implementation
func (t *TypeSafeResourceInterface) ApplyStatus(ctx context.Context, name string, obj *unstructured.Unstructured, options metav1.ApplyOptions) (*unstructured.Unstructured, error) {
	return t.base.ApplyStatus(ctx, name, obj, options)
}

// Create handles type conversion before delegating to the base implementation
func (t *TypeSafeResourceInterface) Create(ctx context.Context, obj *unstructured.Unstructured, options metav1.CreateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	// Convert the object to ensure DeepCopy compatibility
	convertedObj := convertTypesForDeepCopy(obj)
	return t.base.Create(ctx, convertedObj, options, subresources...)
}

// Update handles type conversion before delegating to the base implementation
func (t *TypeSafeResourceInterface) Update(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	convertedObj := convertTypesForDeepCopy(obj)
	return t.base.Update(ctx, convertedObj, options, subresources...)
}

// UpdateStatus delegates to the base implementation
func (t *TypeSafeResourceInterface) UpdateStatus(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	convertedObj := convertTypesForDeepCopy(obj)
	return t.base.UpdateStatus(ctx, convertedObj, options)
}

// Delete delegates to the base implementation
func (t *TypeSafeResourceInterface) Delete(ctx context.Context, name string, options metav1.DeleteOptions, subresources ...string) error {
	return t.base.Delete(ctx, name, options, subresources...)
}

// DeleteCollection delegates to the base implementation
func (t *TypeSafeResourceInterface) DeleteCollection(ctx context.Context, options metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return t.base.DeleteCollection(ctx, options, listOptions)
}

// Get delegates to the base implementation
func (t *TypeSafeResourceInterface) Get(ctx context.Context, name string, options metav1.GetOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return t.base.Get(ctx, name, options, subresources...)
}

// List delegates to the base implementation
func (t *TypeSafeResourceInterface) List(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return t.base.List(ctx, opts)
}

// Watch delegates to the base implementation
func (t *TypeSafeResourceInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return t.base.Watch(ctx, opts)
}

// Patch delegates to the base implementation
func (t *TypeSafeResourceInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, options metav1.PatchOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return t.base.Patch(ctx, name, pt, data, options, subresources...)
}

// convertTypesForDeepCopy recursively converts problematic types in unstructured objects
func convertTypesForDeepCopy(obj *unstructured.Unstructured) *unstructured.Unstructured {
	// First convert types, then create new unstructured object
	convertedData := convertMapTypes(obj.Object)

	// Create a new unstructured object with converted data
	converted := &unstructured.Unstructured{Object: convertedData}
	return converted
}

// convertMapTypes recursively converts map values to DeepCopy-safe types
func convertMapTypes(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range data {
		result[key] = convertValueTypes(value)
	}
	return result
}

// convertValueTypes converts individual values to DeepCopy-safe types
func convertValueTypes(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		return convertMapTypes(v)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = convertValueTypes(item)
		}
		return result
	case []map[string]interface{}:
		// Handle slices of maps specifically
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = convertMapTypes(item)
		}
		return result
	case []string:
		// Convert []string to []interface{} for DeepCopy compatibility
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result
	case int:
		return int64(v) // Convert int to int64 for better JSON compatibility
	case float32:
		return float64(v) // Convert float32 to float64 for better JSON compatibility
	default:
		return value // Return as-is for string, bool, nil, etc.
	}
}

// CreateTestRole creates a test Role with specified permissions.
// This allows tests to pre-create Roles with different permission sets and reuse them.
//
// Parameters:
//   - namespace: The namespace where the Role will be created
//   - roleName: The name of the Role to create
//   - verbs: List of verbs (e.g., ["get", "list", "create", "update", "delete"])
//   - resource: The resource type (e.g., "agenticsessions", "projectsettings", "*" for all)
//   - apiGroup: Optional API group (defaults to "vteam.ambient-code" if empty)
//
// Returns:
//   - The created Role
//   - error: Any error that occurred during creation
func (k *K8sTestUtils) CreateTestRole(ctx context.Context, namespace, roleName string, verbs []string, resource, apiGroup string) (*rbacv1.Role, error) {
	if apiGroup == "" {
		apiGroup = "vteam.ambient-code"
	}

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: namespace,
			Labels: map[string]string{
				"test-framework": "ambient-code-backend",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{apiGroup},
				Resources: []string{resource},
				Verbs:     verbs,
			},
			// Also grant permissions for standard K8s resources that handlers might need
			{
				APIGroups: []string{""},
				Resources: []string{"secrets", "serviceaccounts"},
				Verbs:     []string{"get", "list", "create", "update", "patch"},
			},
		},
	}

	createdRole, err := k.K8sClient.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("failed to create Role: %w", err)
	}

	// If role already exists, get it
	if errors.IsAlreadyExists(err) {
		createdRole, err = k.K8sClient.RbacV1().Roles(namespace).Get(ctx, roleName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get existing Role: %w", err)
		}
	}

	return createdRole, nil
}

// CreateValidTestToken creates a ServiceAccount, RoleBinding, and returns a valid test token
// that matches the RBAC security model. The token can be used with SetAuthHeader or SetValidTestToken.
//
// This ensures tests use tokens that would work with real RBAC, not just arbitrary strings.
//
// Parameters:
//   - namespace: The namespace where the ServiceAccount and RoleBinding will be created
//   - verbs: List of verbs (e.g., ["get", "list", "create", "update", "delete"]) to grant permissions for
//   - resource: The resource type (e.g., "agenticsessions", "projectsettings", "*" for all)
//   - saName: Optional ServiceAccount name (auto-generated if empty)
//   - roleName: Optional pre-existing Role name (if provided, uses this Role instead of creating a new one)
//
// Returns:
//   - token: A JWT-like token string with the correct format and sub claim
//   - saName: The name of the created ServiceAccount
//   - error: Any error that occurred during creation
func (k *K8sTestUtils) CreateValidTestToken(ctx context.Context, namespace string, verbs []string, resource string, saName string, roleName string) (token string, createdSAName string, err error) {
	// Generate ServiceAccount name if not provided
	if saName == "" {
		saName = fmt.Sprintf("test-sa-%d", time.Now().UnixNano())
	}

	// Create ServiceAccount
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: namespace,
			Labels: map[string]string{
				"test-framework": "ambient-code-backend",
				"app":            "ambient-access-key",
			},
		},
	}
	_, err = k.K8sClient.CoreV1().ServiceAccounts(namespace).Create(ctx, sa, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return "", "", fmt.Errorf("failed to create ServiceAccount: %w", err)
	}

	// Use pre-existing Role if provided, otherwise create a new one
	var finalRoleName string
	if roleName != "" {
		// Verify the Role exists
		_, err = k.K8sClient.RbacV1().Roles(namespace).Get(ctx, roleName, metav1.GetOptions{})
		if err != nil {
			return "", "", fmt.Errorf("pre-existing Role %s not found: %w", roleName, err)
		}
		finalRoleName = roleName
	} else {
		// Create Role with specified permissions
		finalRoleName = fmt.Sprintf("test-role-%s-%d", saName, time.Now().UnixNano())
		_, err = k.CreateTestRole(ctx, namespace, finalRoleName, verbs, resource, "")
		if err != nil {
			return "", "", fmt.Errorf("failed to create Role: %w", err)
		}
	}

	// Create RoleBinding
	rbName := fmt.Sprintf("test-rb-%s-%d", saName, time.Now().UnixNano())
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbName,
			Namespace: namespace,
			Labels: map[string]string{
				"test-framework": "ambient-code-backend",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     finalRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: namespace,
			},
		},
	}
	_, err = k.K8sClient.RbacV1().RoleBindings(namespace).Create(ctx, rb, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return "", "", fmt.Errorf("failed to create RoleBinding: %w", err)
	}

	// Create a mock JWT token with the correct format
	// Format: header.payload.signature (all base64url encoded)
	// The payload contains the 'sub' claim: system:serviceaccount:<namespace>:<sa-name>
	sub := fmt.Sprintf("system:serviceaccount:%s:%s", namespace, saName)

	// Create minimal JWT payload
	payload := map[string]interface{}{
		"sub": sub,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	payloadJSON, _ := json.Marshal(payload)

	// Base64url encode (without padding)
	headerB64 := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9" // Standard JWT header
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signatureB64 := "test-signature" // Fake signature for tests

	token = fmt.Sprintf("%s.%s.%s", headerB64, payloadB64, signatureB64)

	// Configure fake client to accept this token via TokenReview
	// This ensures TokenReview calls return authenticated=true for this token
	if fakeClient, ok := k.K8sClient.(*k8sfake.Clientset); ok {
		fakeClient.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			tr := action.(k8stesting.CreateAction).GetObject().(*authnv1.TokenReview)
			// Check if the token matches our test token format
			if tr.Spec.Token == token {
				tr.Status = authnv1.TokenReviewStatus{
					Authenticated: true,
					User: authnv1.UserInfo{
						Username: sub,
						UID:      fmt.Sprintf("test-uid-%s", saName),
					},
				}
				return true, tr, nil
			}
			// For other tokens, return unauthenticated
			tr.Status = authnv1.TokenReviewStatus{
				Authenticated: false,
			}
			return true, tr, nil
		})
	}

	return token, saName, nil
}
