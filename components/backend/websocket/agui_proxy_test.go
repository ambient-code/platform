package websocket

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"ambient-code-backend/handlers"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
