package websocket

import (
	"testing"

	"ambient-code-backend/handlers"
)

// Note: isActivityEvent was removed — all non-empty event types now reset
// the inactivity timer. The inline check `eventType != ""` in
// persistStreamedEvent handles this directly.

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
