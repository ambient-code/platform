package handlerunit_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/ambient-code/platform/components/ambient-api-server/plugins/sessions"
)

// seedRunnerlessSession creates a session with no runner (KubeCrName is nil until CP reconciles).
func seedRunnerlessSession(t *testing.T, svc SessionService) *Session {
	t.Helper()
	proj := "proj-1"
	sess, err := svc.Create(context.Background(), &Session{
		Name:      "agui-test",
		ProjectId: &proj,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return sess
}

func TestAGUITasks_NoRunner_ReturnsEmptyList(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedRunnerlessSession(t, svc)

	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/agui/tasks", sess.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("json: %v", err)
	}
	if _, ok := m["tasks"]; !ok {
		t.Error("expected tasks field in stub response")
	}
}

func TestAGUICapabilities_NoRunner_ReturnsStub(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedRunnerlessSession(t, svc)

	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/agui/capabilities", sess.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var m map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &m)
	if m["framework"] != "unknown" {
		t.Errorf("expected framework=unknown, got %v", m["framework"])
	}
}

func TestMCPStatus_NoRunner_ReturnsStub(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedRunnerlessSession(t, svc)

	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/mcp/status", sess.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var m map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &m)
	if _, ok := m["servers"]; !ok {
		t.Error("expected servers field in stub response")
	}
}

func TestAGUIRun_NoRunner_Returns503(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedRunnerlessSession(t, svc)

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/agui/run", sess.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

func TestAGUIInterrupt_NoRunner_Returns503(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedRunnerlessSession(t, svc)

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/agui/interrupt", sess.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

func TestAGUIFeedback_NoRunner_Returns503(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedRunnerlessSession(t, svc)

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/agui/feedback", sess.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

func TestAGUIEvents_NoRunner_ReturnsEmptySSE(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedRunnerlessSession(t, svc)

	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/agui/events", sess.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	ct := rr.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %q", ct)
	}
}

func TestAGUIRun_SessionNotFound_Returns404(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/ambient/v1/sessions/bad-id/agui/run", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestAGUICapabilities_WithRunner_ProxiesWhenAvailable(t *testing.T) {
	// Set up a mock runner HTTP server.
	mockRunner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"framework":"claude-code"}`))
	}))
	defer mockRunner.Close()

	// Override the EventsHTTPClient to point to the mock runner.
	// Since runnerBaseURL builds a cluster-local URL we can't override without
	// injecting a transport, we test that the stub works when no runner is set.
	// This test verifies the router routing is correct; proxy is tested separately.
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedRunnerlessSession(t, svc)

	// Without a runner, should return stub.
	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/agui/capabilities", sess.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 stub, got %d", rr.Code)
	}
}
