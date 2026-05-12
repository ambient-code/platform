package websocket

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"ambient-code-backend/handlers"
	"ambient-code-backend/tests/test_utils"
	"ambient-code-backend/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

// Note: isActivityEvent was removed — all non-empty event types now reset
// the inactivity timer. The inline check `eventType != ""` in
// persistStreamedEvent handles this directly.

// --- runnerHTTPClient session token tests ---

func TestRunnerHTTPClient_UsesSessionTokenTransport(t *testing.T) {
	transport := runnerHTTPClient.Transport
	if transport == nil {
		t.Fatal("runnerHTTPClient.Transport is nil — must use handlers.NewRunnerTransport to inject X-Ambient-Session-Token")
	}

	typeName := fmt.Sprintf("%T", transport)
	if typeName == "*http.Transport" {
		t.Errorf(
			"runnerHTTPClient.Transport is a plain *http.Transport — must wrap with handlers.NewRunnerTransport "+
				"so X-Ambient-Session-Token is injected on runner requests (got %s)", typeName,
		)
	}
}

func TestConnectToRunner_SendsSessionToken(t *testing.T) {
	const expectedToken = "test-agui-token-value"
	const sessionName = "tok-session"
	const namespace = "tok-project"

	fakeClient := k8sfake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("ambient-runner-token-%s", sessionName),
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"agui-token": []byte(expectedToken),
		},
	})

	oldClient := handlers.K8sClientMw
	handlers.K8sClientMw = fakeClient
	defer func() { handlers.K8sClientMw = oldClient }()

	var receivedToken string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("X-Ambient-Session-Token")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	runnerURL := fmt.Sprintf("http://session-%s.%s.svc.cluster.local:8001/", sessionName, namespace)

	oldHTTPClient := runnerHTTPClient
	defer func() { runnerHTTPClient = oldHTTPClient }()

	runnerHTTPClient = &http.Client{
		Transport: handlers.NewRunnerTransport(&rewriteHostTransport{
			realURL: ts.URL,
		}),
	}

	resp, err := connectToRunner(runnerURL, []byte(`{}`), "", "", "")
	if err != nil {
		t.Fatalf("connectToRunner failed: %v", err)
	}
	resp.Body.Close()

	if receivedToken != expectedToken {
		t.Errorf("Expected X-Ambient-Session-Token=%q, got %q", expectedToken, receivedToken)
	}
}

func TestConnectToRunner_NoTokenWhenSecretMissing(t *testing.T) {
	fakeClient := k8sfake.NewSimpleClientset()

	oldClient := handlers.K8sClientMw
	handlers.K8sClientMw = fakeClient
	defer func() { handlers.K8sClientMw = oldClient }()

	var receivedToken string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("X-Ambient-Session-Token")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	runnerURL := "http://session-no-secret.no-project.svc.cluster.local:8001/"

	oldHTTPClient := runnerHTTPClient
	defer func() { runnerHTTPClient = oldHTTPClient }()

	runnerHTTPClient = &http.Client{
		Transport: handlers.NewRunnerTransport(&rewriteHostTransport{
			realURL: ts.URL,
		}),
	}

	resp, err := connectToRunner(runnerURL, []byte(`{}`), "", "", "")
	if err != nil {
		t.Fatalf("connectToRunner failed: %v", err)
	}
	resp.Body.Close()

	if receivedToken != "" {
		t.Errorf("Expected no X-Ambient-Session-Token when secret missing, got %q", receivedToken)
	}
}

type rewriteHostTransport struct {
	realURL string
}

func (t *rewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rewritten := req.Clone(req.Context())
	rewritten.URL.Scheme = "http"
	rewritten.URL.Host = t.realURL[len("http://"):]
	return http.DefaultTransport.RoundTrip(rewritten)
}

// --- getRunnerEndpoint tests ---

func TestGetRunnerEndpoint_DefaultPort(t *testing.T) {
	// When no port is cached, getRunnerEndpoint should use DefaultRunnerPort
	sessionPortMap.Delete("test-session") // ensure clean state

	endpoint := getRunnerEndpoint("my-project", "test-session")
	expected := "http://session-test-session.my-project.svc.cluster.local:8001/"
	if endpoint != expected {
		t.Errorf("Expected %q, got %q", expected, endpoint)
	}
}

func TestGetRunnerEndpoint_CachedPort(t *testing.T) {
	// When a port is cached in sessionPortMap, getRunnerEndpoint should use it
	sessionPortMap.Store("test-session-custom", 9090)
	defer sessionPortMap.Delete("test-session-custom")

	endpoint := getRunnerEndpoint("my-project", "test-session-custom")
	expected := "http://session-test-session-custom.my-project.svc.cluster.local:9090/"
	if endpoint != expected {
		t.Errorf("Expected %q, got %q", expected, endpoint)
	}
}

func TestGetRunnerEndpoint_UsesRegistryPort(t *testing.T) {
	// Simulate caching a non-default port from the registry (as cacheSessionPort does)
	sessionPortMap.Store("gemini-session", 9090)
	defer sessionPortMap.Delete("gemini-session")

	endpoint := getRunnerEndpoint("dev-project", "gemini-session")
	expected := "http://session-gemini-session.dev-project.svc.cluster.local:9090/"
	if endpoint != expected {
		t.Errorf("Expected %q, got %q", expected, endpoint)
	}
}

func TestGetRunnerEndpoint_DifferentPorts(t *testing.T) {
	// Multiple sessions with different ports
	sessionPortMap.Store("session-a", 8001)
	sessionPortMap.Store("session-b", 9090)
	sessionPortMap.Store("session-c", 8080)
	defer func() {
		sessionPortMap.Delete("session-a")
		sessionPortMap.Delete("session-b")
		sessionPortMap.Delete("session-c")
	}()

	tests := []struct {
		name     string
		session  string
		port     int
		expected string
	}{
		{"port 8001", "session-a", 8001, "http://session-session-a.ns.svc.cluster.local:8001/"},
		{"port 9090", "session-b", 9090, "http://session-session-b.ns.svc.cluster.local:9090/"},
		{"port 8080", "session-c", 8080, "http://session-session-c.ns.svc.cluster.local:8080/"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := getRunnerEndpoint("ns", tc.session)
			if endpoint != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, endpoint)
			}
		})
	}
}

func TestDefaultRunnerPort_Constant(t *testing.T) {
	// Verify the DefaultRunnerPort constant is 8001
	if handlers.DefaultRunnerPort != 8001 {
		t.Errorf("Expected DefaultRunnerPort=8001, got %d", handlers.DefaultRunnerPort)
	}
}

// --- triggerDisplayNameGenerationIfNeeded tests (regression for #1561) ---

func setupDisplayNameTest(t *testing.T, spec map[string]interface{}) (cleanup func()) {
	t.Helper()

	oldDynamic := handlers.DynamicClient
	oldK8sProjects := handlers.K8sClientProjects
	oldGVRFunc := handlers.GetAgenticSessionV1Alpha1Resource

	agenticSessionGVR := schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1alpha1",
		Resource: "agenticsessions",
	}
	handlers.GetAgenticSessionV1Alpha1Resource = func() schema.GroupVersionResource {
		return agenticSessionGVR
	}

	fakeClients := test_utils.NewFakeClientSet()
	handlers.DynamicClient = fakeClients.GetDynamicClient()
	handlers.K8sClientProjects = fakeClients.GetK8sClient()

	err := test_utils.CreateAgenticSessionInFakeClient(
		handlers.DynamicClient, "test-project", "test-session", spec,
	)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	return func() {
		handlers.DynamicClient = oldDynamic
		handlers.K8sClientProjects = oldK8sProjects
		handlers.GetAgenticSessionV1Alpha1Resource = oldGVRFunc
	}
}

func getDisplayName(t *testing.T, dc dynamic.Interface) string {
	t.Helper()
	gvr := handlers.GetAgenticSessionV1Alpha1Resource()
	item, err := dc.Resource(gvr).Namespace("test-project").Get(
		context.Background(), "test-session", metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}
	dn, _, _ := unstructured.NestedString(item.Object, "spec", "displayName")
	return dn
}

func TestTriggerDisplayName_InitialPromptNotSkipped(t *testing.T) {
	cleanup := setupDisplayNameTest(t, map[string]interface{}{
		"initialPrompt": "Help me debug auth",
	})
	defer cleanup()

	called := false
	oldFn := handlers.GenerateDisplayNameAsync
	handlers.GenerateDisplayNameAsync = func(projectName, sessionName, userMessage string, sessionCtx handlers.SessionContext) {
		called = true
	}
	defer func() { handlers.GenerateDisplayNameAsync = oldFn }()

	msgs := []types.Message{
		{ID: "msg-1", Role: "user", Content: "Help me debug auth"},
	}

	triggerDisplayNameGenerationIfNeeded("test-project", "test-session", msgs)

	if !called {
		t.Error("Expected GenerateDisplayNameAsync to be called for initialPrompt message when displayName is empty")
	}
}

func TestTriggerDisplayName_SkipsWhenNameAlreadySet(t *testing.T) {
	cleanup := setupDisplayNameTest(t, map[string]interface{}{
		"initialPrompt": "Help me debug auth",
		"displayName":   "Debug Auth Middleware",
	})
	defer cleanup()

	msgs := []types.Message{
		{ID: "msg-1", Role: "user", Content: "Help me debug auth"},
	}

	triggerDisplayNameGenerationIfNeeded("test-project", "test-session", msgs)

	// displayName should remain unchanged — ShouldGenerateDisplayName
	// returns false when displayName is already set.
	dn := getDisplayName(t, handlers.DynamicClient)
	if dn != "Debug Auth Middleware" {
		t.Errorf("Expected displayName to remain %q, got %q", "Debug Auth Middleware", dn)
	}
}

func TestTriggerDisplayName_SkipsWhenNoUserMessage(t *testing.T) {
	cleanup := setupDisplayNameTest(t, map[string]interface{}{
		"initialPrompt": "Help me debug auth",
	})
	defer cleanup()

	// Only assistant messages — no user content to generate from
	msgs := []types.Message{
		{ID: "msg-1", Role: "assistant", Content: "I'll help you debug"},
	}

	triggerDisplayNameGenerationIfNeeded("test-project", "test-session", msgs)

	dn := getDisplayName(t, handlers.DynamicClient)
	if dn != "" {
		t.Errorf("Expected empty displayName, got %q", dn)
	}
}

func TestTriggerDisplayName_SkipsWhenDynamicClientNil(t *testing.T) {
	oldDynamic := handlers.DynamicClient
	handlers.DynamicClient = nil
	defer func() { handlers.DynamicClient = oldDynamic }()

	msgs := []types.Message{
		{ID: "msg-1", Role: "user", Content: "Help me debug auth"},
	}

	// Should return early without panic when DynamicClient is nil
	triggerDisplayNameGenerationIfNeeded("test-project", "test-session", msgs)
}
