package websocket

import (
	"testing"

	"ambient-code-backend/handlers"
	"ambient-code-backend/types"
)

func TestIsActivityEvent(t *testing.T) {
	activityEvents := []struct {
		name      string
		eventType string
	}{
		{"RUN_STARTED", types.EventTypeRunStarted},
		{"TEXT_MESSAGE_START", types.EventTypeTextMessageStart},
		{"TEXT_MESSAGE_CONTENT", types.EventTypeTextMessageContent},
		{"TOOL_CALL_START", types.EventTypeToolCallStart},
	}

	for _, tc := range activityEvents {
		t.Run(tc.name+" is activity", func(t *testing.T) {
			if !isActivityEvent(tc.eventType) {
				t.Errorf("expected %s to be an activity event", tc.name)
			}
		})
	}

	nonActivityEvents := []struct {
		name      string
		eventType string
	}{
		{"RUN_FINISHED", types.EventTypeRunFinished},
		{"RUN_ERROR", types.EventTypeRunError},
		{"STEP_STARTED", types.EventTypeStepStarted},
		{"STEP_FINISHED", types.EventTypeStepFinished},
		{"TEXT_MESSAGE_END", types.EventTypeTextMessageEnd},
		{"TOOL_CALL_ARGS", types.EventTypeToolCallArgs},
		{"TOOL_CALL_END", types.EventTypeToolCallEnd},
		{"STATE_SNAPSHOT", types.EventTypeStateSnapshot},
		{"STATE_DELTA", types.EventTypeStateDelta},
		{"MESSAGES_SNAPSHOT", types.EventTypeMessagesSnapshot},
		{"RAW", types.EventTypeRaw},
		{"META", types.EventTypeMeta},
		{"empty string", ""},
		{"unknown event", "UNKNOWN_EVENT"},
	}

	for _, tc := range nonActivityEvents {
		t.Run(tc.name+" is not activity", func(t *testing.T) {
			if isActivityEvent(tc.eventType) {
				t.Errorf("expected %s to NOT be an activity event", tc.name)
			}
		})
	}
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
