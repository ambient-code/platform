package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ambient-code-public-api/types"
)

func TestCompactEvents(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "RUN_STARTED", "runId": "r1"},
		{"type": "TEXT_MESSAGE_START", "runId": "r1", "messageId": "m1", "role": "assistant"},
		{"type": "TEXT_MESSAGE_CONTENT", "runId": "r1", "messageId": "m1", "delta": "Hello "},
		{"type": "TEXT_MESSAGE_CONTENT", "runId": "r1", "messageId": "m1", "delta": "world"},
		{"type": "TEXT_MESSAGE_CONTENT", "runId": "r1", "messageId": "m1", "delta": "!"},
		{"type": "TEXT_MESSAGE_END", "runId": "r1", "messageId": "m1"},
		{"type": "TOOL_CALL_START", "runId": "r1", "toolCallId": "tc1", "toolCallName": "read"},
		{"type": "TOOL_CALL_ARGS", "runId": "r1", "toolCallId": "tc1", "delta": `{"file"`},
		{"type": "TOOL_CALL_ARGS", "runId": "r1", "toolCallId": "tc1", "delta": `: "main.go"}`},
		{"type": "TOOL_CALL_END", "runId": "r1", "toolCallId": "tc1"},
		{"type": "RUN_FINISHED", "runId": "r1"},
	}

	compacted := compactEvents(events)

	// Should merge TEXT_MESSAGE_CONTENT: 3 deltas → 1
	// Should merge TOOL_CALL_ARGS: 2 deltas → 1
	// Other events pass through: RUN_STARTED, TEXT_MESSAGE_START, TEXT_MESSAGE_END, TOOL_CALL_START, TOOL_CALL_END, RUN_FINISHED = 6
	// Total: 6 + 1 + 1 = 8
	if len(compacted) != 8 {
		t.Fatalf("Expected 8 compacted events, got %d", len(compacted))
	}

	// Find the merged text content event
	for _, e := range compacted {
		if e["type"] == "TEXT_MESSAGE_CONTENT" {
			if e["delta"] != "Hello world!" {
				t.Errorf("Expected merged delta 'Hello world!', got %q", e["delta"])
			}
		}
		if e["type"] == "TOOL_CALL_ARGS" {
			if e["delta"] != `{"file": "main.go"}` {
				t.Errorf("Expected merged delta '{\"file\": \"main.go\"}', got %q", e["delta"])
			}
		}
	}
}

func TestCompactEvents_DifferentMessageIDs(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "TEXT_MESSAGE_CONTENT", "messageId": "m1", "delta": "A"},
		{"type": "TEXT_MESSAGE_CONTENT", "messageId": "m2", "delta": "B"},
		{"type": "TEXT_MESSAGE_CONTENT", "messageId": "m1", "delta": "C"},
	}

	compacted := compactEvents(events)

	// Different messageIds should NOT merge
	if len(compacted) != 3 {
		t.Fatalf("Expected 3 events (different messageIds), got %d", len(compacted))
	}
}

func TestCompactEvents_Empty(t *testing.T) {
	compacted := compactEvents([]map[string]interface{}{})
	if len(compacted) != 0 {
		t.Errorf("Expected 0 events, got %d", len(compacted))
	}
}

func TestExtractTranscript(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "RUN_STARTED", "runId": "r1"},
		{"type": "TEXT_MESSAGE_START", "runId": "r1"},
		{"type": "MESSAGES_SNAPSHOT", "runId": "r1", "messages": []interface{}{
			map[string]interface{}{
				"id":      "msg-1",
				"role":    "user",
				"content": "Hello",
			},
			map[string]interface{}{
				"id":      "msg-2",
				"role":    "assistant",
				"content": "Hi there!",
				"toolCalls": []interface{}{
					map[string]interface{}{
						"id":       "tc-1",
						"name":     "read",
						"args":     `{"file": "main.go"}`,
						"status":   "completed",
						"duration": float64(100),
					},
				},
			},
			map[string]interface{}{
				"id":         "msg-3",
				"role":       "tool",
				"content":    "file content here",
				"toolCallId": "tc-1",
				"name":       "read",
			},
		}},
		{"type": "RUN_FINISHED", "runId": "r1"},
	}

	messages := extractTranscript(events)

	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(messages))
	}

	if messages[0].Role != "user" || messages[0].Content != "Hello" {
		t.Errorf("First message: role=%q content=%q", messages[0].Role, messages[0].Content)
	}

	if messages[1].Role != "assistant" || messages[1].Content != "Hi there!" {
		t.Errorf("Second message: role=%q content=%q", messages[1].Role, messages[1].Content)
	}
	if len(messages[1].ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(messages[1].ToolCalls))
	}
	if messages[1].ToolCalls[0].Name != "read" {
		t.Errorf("Expected tool call name 'read', got %q", messages[1].ToolCalls[0].Name)
	}
	if messages[1].ToolCalls[0].Duration != 100 {
		t.Errorf("Expected tool call duration 100, got %d", messages[1].ToolCalls[0].Duration)
	}

	if messages[2].Role != "tool" || messages[2].ToolCallID != "tc-1" {
		t.Errorf("Third message: role=%q toolCallId=%q", messages[2].Role, messages[2].ToolCallID)
	}
}

func TestExtractTranscript_NoSnapshot(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "RUN_STARTED", "runId": "r1"},
		{"type": "TEXT_MESSAGE_START", "runId": "r1"},
		{"type": "RUN_FINISHED", "runId": "r1"},
	}

	messages := extractTranscript(events)
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages when no snapshot, got %d", len(messages))
	}
}

func TestExtractTranscript_UsesLastSnapshot(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "MESSAGES_SNAPSHOT", "messages": []interface{}{
			map[string]interface{}{"id": "old", "role": "user", "content": "Old"},
		}},
		{"type": "MESSAGES_SNAPSHOT", "messages": []interface{}{
			map[string]interface{}{"id": "new", "role": "user", "content": "New"},
		}},
	}

	messages := extractTranscript(events)
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	if messages[0].Content != "New" {
		t.Errorf("Expected last snapshot content 'New', got %q", messages[0].Content)
	}
}

func makeExportBackend(t *testing.T, events []map[string]interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		eventsJSON, _ := json.Marshal(events)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sessionId":  "test-session",
			"aguiEvents": json.RawMessage(eventsJSON),
		})
	}))
}

func TestE2E_GetOutput_Transcript(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "MESSAGES_SNAPSHOT", "messages": []interface{}{
			map[string]interface{}{"id": "m1", "role": "user", "content": "Hello"},
		}},
	}
	backend := makeExportBackend(t, events)
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/test-session/output", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response types.TranscriptOutputResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Format != "transcript" {
		t.Errorf("Expected format 'transcript', got %q", response.Format)
	}
	if len(response.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(response.Messages))
	}
}

func TestE2E_GetOutput_Events(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "RUN_STARTED", "runId": "r1"},
		{"type": "RUN_FINISHED", "runId": "r1"},
	}
	backend := makeExportBackend(t, events)
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/test-session/output?format=events", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response types.EventsOutputResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Format != "events" {
		t.Errorf("Expected format 'events', got %q", response.Format)
	}
	if len(response.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(response.Events))
	}
}

func TestE2E_GetOutput_Compact(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "TEXT_MESSAGE_CONTENT", "messageId": "m1", "delta": "A"},
		{"type": "TEXT_MESSAGE_CONTENT", "messageId": "m1", "delta": "B"},
	}
	backend := makeExportBackend(t, events)
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/test-session/output?format=compact", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response types.EventsOutputResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Format != "compact" {
		t.Errorf("Expected format 'compact', got %q", response.Format)
	}
	if len(response.Events) != 1 {
		t.Errorf("Expected 1 compacted event, got %d", len(response.Events))
	}
}

func TestE2E_GetOutput_InvalidFormat(t *testing.T) {
	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/test-session/output?format=xml", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid format, got %d", w.Code)
	}
}

func TestE2E_GetOutput_InvalidRunID(t *testing.T) {
	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/test-session/output?run_id=not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid run_id, got %d", w.Code)
	}
}

func TestE2E_GetOutput_RunIDFilter(t *testing.T) {
	events := []map[string]interface{}{
		{"type": "RUN_STARTED", "runId": "11111111-1111-1111-1111-111111111111"},
		{"type": "RUN_FINISHED", "runId": "11111111-1111-1111-1111-111111111111"},
		{"type": "RUN_STARTED", "runId": "22222222-2222-2222-2222-222222222222"},
		{"type": "RUN_FINISHED", "runId": "22222222-2222-2222-2222-222222222222"},
	}
	backend := makeExportBackend(t, events)
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/test-session/output?format=events&run_id=11111111-1111-1111-1111-111111111111", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response types.EventsOutputResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if len(response.Events) != 2 {
		t.Errorf("Expected 2 events after filtering, got %d", len(response.Events))
	}
}
