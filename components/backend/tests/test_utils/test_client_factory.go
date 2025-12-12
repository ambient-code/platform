package test_utils

import (
	"context"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// TestClientFactory provides a way to inject fake clients into handlers
type TestClientFactory struct {
	fakeClients   *FakeClientSet
	mockFunctions *MockedFunctions
}

// MockedFunctions holds references to original functions that need to be mocked
type MockedFunctions struct {
	// Store original function references for restoration
	OriginalGetK8sClientsForRequest interface{}
	OriginalValidateSecretAccess    interface{}

	// Mock implementations
	MockGetK8sClientsForRequest MockK8sClientsForRequest
	MockValidateSecretAccess    MockValidateSecretAccess
}

// NewTestClientFactory creates a new test client factory with fake clients
func NewTestClientFactory() *TestClientFactory {
	fakeClients := NewFakeClientSet()

	return &TestClientFactory{
		fakeClients: fakeClients,
		mockFunctions: &MockedFunctions{
			// Default mock implementations
			MockGetK8sClientsForRequest: func(c interface{}) (kubernetes.Interface, dynamic.Interface) {
				return fakeClients.GetK8sClient(), fakeClients.GetDynamicClient()
			},
			MockValidateSecretAccess: func(ctx context.Context, clientset kubernetes.Interface, project, verb string) error {
				// Allow all operations by default in tests
				return nil
			},
		},
	}
}

// GetFakeClients returns the fake client set
func (tcf *TestClientFactory) GetFakeClients() *FakeClientSet {
	return tcf.fakeClients
}

// GetMockedFunctions returns the mocked functions
func (tcf *TestClientFactory) GetMockedFunctions() *MockedFunctions {
	return tcf.mockFunctions
}

// SetupMocks sets up the mock functions (to be called in BeforeEach)
// Note: This would be used if the handlers supported dependency injection
func (tcf *TestClientFactory) SetupMocks() {
	// In a real implementation, this would inject the mocks into the handlers
	// For now, this serves as a placeholder for the test setup pattern
}

// RestoreMocks restores the original functions (to be called in AfterEach)
// Note: This would be used if the handlers supported dependency injection
func (tcf *TestClientFactory) RestoreMocks() {
	// In a real implementation, this would restore the original functions
	// For now, this serves as a placeholder for the test teardown pattern
}

// WithCustomGetK8sClientsForRequest allows customizing the mock implementation
func (tcf *TestClientFactory) WithCustomGetK8sClientsForRequest(mock MockK8sClientsForRequest) *TestClientFactory {
	tcf.mockFunctions.MockGetK8sClientsForRequest = mock
	return tcf
}

// WithCustomValidateSecretAccess allows customizing the mock implementation
func (tcf *TestClientFactory) WithCustomValidateSecretAccess(mock MockValidateSecretAccess) *TestClientFactory {
	tcf.mockFunctions.MockValidateSecretAccess = mock
	return tcf
}

// CreateTestAgenticSession creates a test AgenticSession in the fake dynamic client
func (tcf *TestClientFactory) CreateTestAgenticSession(namespace, name string, spec map[string]interface{}) error {
	return CreateAgenticSessionInFakeClient(tcf.fakeClients.GetDynamicClient(), namespace, name, spec)
}
