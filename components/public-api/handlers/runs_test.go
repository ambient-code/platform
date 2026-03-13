package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ambient-code-public-api/types"
)

func TestDeriveRunSummaries(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "RUN_STARTED", "runId": "run-1", "timestamp": float64(1000)},
		{"type": "TEXT_MESSAGE_START", "runId": "run-1", "role": "user", "content": "Hello"},
		{"type": "TEXT_MESSAGE_CONTENT", "runId": "run-1", "delta": "Hi"},
		{"type": "RUN_FINISHED", "runId": "run-1", "timestamp": float64(2000)},
		{"type": "RUN_STARTED", "runId": "run-2", "timestamp": float64(3000)},
		{"type": "TEXT_MESSAGE_START", "runId": "run-2", "role": "user", "content": "Fix bug"},
		{"type": "RUN_ERROR", "runId": "run-2", "timestamp": float64(4000)},
	}

	runs := deriveRunSummaries(events)

	if len(runs) != 2 {
		t.Fatalf("Expected 2 runs, got %d", len(runs))
	}

	// First run
	if runs[0].RunID != "run-1" {
		t.Errorf("Expected run-1, got %s", runs[0].RunID)
	}
	if runs[0].Status != "completed" {
		t.Errorf("Expected completed, got %s", runs[0].Status)
	}
	if runs[0].StartedAt != 1000 {
		t.Errorf("Expected started_at 1000, got %d", runs[0].StartedAt)
	}
	if runs[0].FinishedAt != 2000 {
		t.Errorf("Expected finished_at 2000, got %d", runs[0].FinishedAt)
	}
	if runs[0].UserMessage != "Hello" {
		t.Errorf("Expected user message 'Hello', got %q", runs[0].UserMessage)
	}
	if runs[0].EventCount != 4 {
		t.Errorf("Expected 4 events, got %d", runs[0].EventCount)
	}

	// Second run
	if runs[1].RunID != "run-2" {
		t.Errorf("Expected run-2, got %s", runs[1].RunID)
	}
	if runs[1].Status != "error" {
		t.Errorf("Expected error, got %s", runs[1].Status)
	}
	if runs[1].UserMessage != "Fix bug" {
		t.Errorf("Expected user message 'Fix bug', got %q", runs[1].UserMessage)
	}
}

func TestDeriveRunSummaries_Empty(t *testing.T) {
	runs := deriveRunSummaries([]map[string]interface{}{})
	if len(runs) != 0 {
		t.Errorf("Expected 0 runs, got %d", len(runs))
	}
}

func TestDeriveRunSummaries_NoRunID(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "RUN_STARTED"},
		{"type": "TEXT_MESSAGE_START", "role": "user"},
	}
	runs := deriveRunSummaries(events)
	if len(runs) != 0 {
		t.Errorf("Expected 0 runs for events without runId, got %d", len(runs))
	}
}

func TestDeriveRunSummaries_RunningStatus(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "RUN_STARTED", "runId": "run-1", "timestamp": float64(1000)},
		{"type": "TEXT_MESSAGE_START", "runId": "run-1", "role": "user", "content": "Hello"},
	}
	runs := deriveRunSummaries(events)
	if len(runs) != 1 {
		t.Fatalf("Expected 1 run, got %d", len(runs))
	}
	if runs[0].Status != "running" {
		t.Errorf("Expected running, got %s", runs[0].Status)
	}
}

func TestToInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int64
	}{
		{"float64", float64(1234), 1234},
		{"int64", int64(5678), 5678},
		{"json.Number", json.Number("9999"), 9999},
		{"string", "not-a-number", 0},
		{"nil", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toInt64(tt.input)
			if result != tt.expected {
				t.Errorf("toInt64(%v) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestE2E_CreateRun(t *testing.T) {
	var receivedBody map[string]interface{}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// GET session (for phase check)
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/agentic-sessions/") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"metadata": map[string]interface{}{"name": "test-session"},
				"status":   map[string]interface{}{"phase": "Running"},
			})
			return
		}

		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/agui/run") {
			t.Errorf("Expected path to contain /agui/run, got %s", r.URL.Path)
		}

		decoder := json.NewDecoder(r.Body)
		decoder.Decode(&receivedBody)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"runId":    "test-run-id",
			"threadId": "test-session",
		})
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/test-session/runs",
		strings.NewReader(`{"messages": [{"role": "user", "content": "Hello"}]}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d: %s", w.Code, w.Body.String())
	}

	var response types.CreateRunResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.RunID != "test-run-id" {
		t.Errorf("Expected run_id 'test-run-id', got %q", response.RunID)
	}
	if response.ThreadID != "test-session" {
		t.Errorf("Expected thread_id 'test-session', got %q", response.ThreadID)
	}

	// Verify messages were forwarded
	if receivedBody["messages"] == nil {
		t.Error("Expected messages in request body")
	}
}

func makeRunningSessionBackend(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"metadata": map[string]interface{}{"name": "test-session"},
			"status":   map[string]interface{}{"phase": "Running"},
		})
	}))
}

func TestE2E_CreateRun_InvalidBody(t *testing.T) {
	backend := makeRunningSessionBackend(t)
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/test-session/runs",
		strings.NewReader(`{"messages": []}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestE2E_CreateRun_NoMessages(t *testing.T) {
	backend := makeRunningSessionBackend(t)
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/test-session/runs",
		strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestE2E_CreateRun_SessionNotRunning(t *testing.T) {
	backend := makeSessionBackend(t, "Completed", "/agui/run", http.StatusOK)
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/test-session/runs",
		strings.NewReader(`{"messages": [{"role": "user", "content": "Hello"}]}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422 for completed session, got %d: %s", w.Code, w.Body.String())
	}
}

func TestE2E_GetSessionRuns(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/export") {
			t.Errorf("Expected path to contain /export, got %s", r.URL.Path)
		}

		events := []map[string]interface{}{
			{"type": "RUN_STARTED", "runId": "run-abc", "timestamp": float64(1000)},
			{"type": "TEXT_MESSAGE_START", "runId": "run-abc", "role": "user", "content": "Hello"},
			{"type": "RUN_FINISHED", "runId": "run-abc", "timestamp": float64(2000)},
		}
		eventsJSON, _ := json.Marshal(events)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sessionId":  "test-session",
			"aguiEvents": json.RawMessage(eventsJSON),
		})
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/test-session/runs", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response types.SessionRunsResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.SessionID != "test-session" {
		t.Errorf("Expected session_id 'test-session', got %q", response.SessionID)
	}
	if len(response.Runs) != 1 {
		t.Fatalf("Expected 1 run, got %d", len(response.Runs))
	}
	if response.Runs[0].Status != "completed" {
		t.Errorf("Expected completed status, got %q", response.Runs[0].Status)
	}
	if response.Runs[0].UserMessage != "Hello" {
		t.Errorf("Expected user message 'Hello', got %q", response.Runs[0].UserMessage)
	}
}

func TestE2E_GetSessionRuns_BackendError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Session not found"})
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/nonexistent/runs", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestE2E_SendMessage(t *testing.T) {
	var receivedBody map[string]interface{}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// GET session (for phase check)
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/agentic-sessions/") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"metadata": map[string]interface{}{"name": "test-session"},
				"status":   map[string]interface{}{"phase": "Running"},
			})
			return
		}

		decoder := json.NewDecoder(r.Body)
		decoder.Decode(&receivedBody)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"runId":    "generated-run-id",
			"threadId": "test-session",
		})
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/test-session/message",
		strings.NewReader(`{"content": "Fix the bug please"}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d: %s", w.Code, w.Body.String())
	}

	var response types.SendMessageResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.RunID != "generated-run-id" {
		t.Errorf("Expected run_id 'generated-run-id', got %q", response.RunID)
	}
	if response.ThreadID != "test-session" {
		t.Errorf("Expected thread_id 'test-session', got %q", response.ThreadID)
	}

	// Verify threadId was set to session ID
	if receivedBody["threadId"] != "test-session" {
		t.Errorf("Expected threadId 'test-session', got %v", receivedBody["threadId"])
	}
	// Verify runId was generated
	if receivedBody["runId"] == nil || receivedBody["runId"] == "" {
		t.Error("Expected runId to be generated")
	}
}

func TestE2E_SendMessage_EmptyContent(t *testing.T) {
	backend := makeRunningSessionBackend(t)
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/test-session/message",
		strings.NewReader(`{"content": ""}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty content, got %d: %s", w.Code, w.Body.String())
	}
}

func TestE2E_SendMessage_SessionNotRunning(t *testing.T) {
	backend := makeSessionBackend(t, "Failed", "/agui/run", http.StatusOK)
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/test-session/message",
		strings.NewReader(`{"content": "Hello"}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422 for failed session, got %d: %s", w.Code, w.Body.String())
	}
}
