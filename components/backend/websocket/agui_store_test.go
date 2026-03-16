package websocket

import (
	"ambient-code-backend/types"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDeriveAgentStatus(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "agui-store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the StateBaseDir to our temp directory for testing
	origStateBaseDir := StateBaseDir
	StateBaseDir = tmpDir
	defer func() { StateBaseDir = origStateBaseDir }()

	t.Run("empty file returns empty status", func(t *testing.T) {
		sessionID := "test-session-empty"
		sessionsDir := filepath.Join(tmpDir, "sessions", sessionID)
		if err := os.MkdirAll(sessionsDir, 0755); err != nil {
			t.Fatalf("Failed to create sessions dir: %v", err)
		}

		// Create empty events file
		eventsFile := filepath.Join(sessionsDir, "agui-events.jsonl")
		if err := os.WriteFile(eventsFile, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to write events file: %v", err)
		}

		status := DeriveAgentStatus(sessionID)
		if status != "" {
			t.Errorf("Expected empty status for empty file, got %q", status)
		}
	})

	t.Run("RUN_STARTED only returns working", func(t *testing.T) {
		sessionID := "test-session-run-started"
		sessionsDir := filepath.Join(tmpDir, "sessions", sessionID)
		if err := os.MkdirAll(sessionsDir, 0755); err != nil {
			t.Fatalf("Failed to create sessions dir: %v", err)
		}

		// Create events file with RUN_STARTED event
		event := map[string]interface{}{
			"type":  types.EventTypeRunStarted,
			"runId": "run-123",
		}
		eventData, _ := json.Marshal(event)
		eventsFile := filepath.Join(sessionsDir, "agui-events.jsonl")
		if err := os.WriteFile(eventsFile, append(eventData, '\n'), 0644); err != nil {
			t.Fatalf("Failed to write events file: %v", err)
		}

		status := DeriveAgentStatus(sessionID)
		if status != types.AgentStatusWorking {
			t.Errorf("Expected %q for RUN_STARTED, got %q", types.AgentStatusWorking, status)
		}
	})

	t.Run("RUN_FINISHED returns idle", func(t *testing.T) {
		sessionID := "test-session-run-finished"
		sessionsDir := filepath.Join(tmpDir, "sessions", sessionID)
		if err := os.MkdirAll(sessionsDir, 0755); err != nil {
			t.Fatalf("Failed to create sessions dir: %v", err)
		}

		// Create events file with RUN_STARTED then RUN_FINISHED
		events := []map[string]interface{}{
			{"type": types.EventTypeRunStarted, "runId": "run-123"},
			{"type": types.EventTypeRunFinished, "runId": "run-123"},
		}
		eventsFile := filepath.Join(sessionsDir, "agui-events.jsonl")
		f, err := os.Create(eventsFile)
		if err != nil {
			t.Fatalf("Failed to create events file: %v", err)
		}
		for _, evt := range events {
			data, _ := json.Marshal(evt)
			f.Write(append(data, '\n'))
		}
		f.Close()

		status := DeriveAgentStatus(sessionID)
		if status != types.AgentStatusIdle {
			t.Errorf("Expected %q for RUN_FINISHED, got %q", types.AgentStatusIdle, status)
		}
	})

	t.Run("RUN_FINISHED with same-run AskUserQuestion returns waiting_input", func(t *testing.T) {
		sessionID := "test-session-waiting-input"
		sessionsDir := filepath.Join(tmpDir, "sessions", sessionID)
		if err := os.MkdirAll(sessionsDir, 0755); err != nil {
			t.Fatalf("Failed to create sessions dir: %v", err)
		}

		// Create events file with RUN_STARTED, AskUserQuestion TOOL_CALL_START, then RUN_FINISHED
		// Scanning backwards: RUN_FINISHED → looks deeper → finds AskUserQuestion with same runId
		events := []map[string]interface{}{
			{"type": types.EventTypeRunStarted, "runId": "run-123"},
			{"type": types.EventTypeToolCallStart, "runId": "run-123", "toolCallName": "AskUserQuestion"},
			{"type": types.EventTypeRunFinished, "runId": "run-123"},
		}
		eventsFile := filepath.Join(sessionsDir, "agui-events.jsonl")
		f, err := os.Create(eventsFile)
		if err != nil {
			t.Fatalf("Failed to create events file: %v", err)
		}
		for _, evt := range events {
			data, _ := json.Marshal(evt)
			f.Write(append(data, '\n'))
		}
		f.Close()

		status := DeriveAgentStatus(sessionID)
		if status != types.AgentStatusWaitingInput {
			t.Errorf("Expected %q for same-run AskUserQuestion, got %q", types.AgentStatusWaitingInput, status)
		}
	})

	t.Run("RUN_FINISHED with different-run AskUserQuestion returns idle", func(t *testing.T) {
		sessionID := "test-session-different-run"
		sessionsDir := filepath.Join(tmpDir, "sessions", sessionID)
		if err := os.MkdirAll(sessionsDir, 0755); err != nil {
			t.Fatalf("Failed to create sessions dir: %v", err)
		}

		// Create events file with old AskUserQuestion from run-456, then run-123 finishes
		// Scanning backwards: RUN_FINISHED(run-123) → looks deeper → finds AskUserQuestion(run-456) → different run → idle
		events := []map[string]interface{}{
			{"type": types.EventTypeRunStarted, "runId": "run-456"},
			{"type": types.EventTypeToolCallStart, "runId": "run-456", "toolCallName": "AskUserQuestion"},
			{"type": types.EventTypeRunFinished, "runId": "run-456"},
			{"type": types.EventTypeRunStarted, "runId": "run-123"},
			{"type": types.EventTypeRunFinished, "runId": "run-123"},
		}
		eventsFile := filepath.Join(sessionsDir, "agui-events.jsonl")
		f, err := os.Create(eventsFile)
		if err != nil {
			t.Fatalf("Failed to create events file: %v", err)
		}
		for _, evt := range events {
			data, _ := json.Marshal(evt)
			f.Write(append(data, '\n'))
		}
		f.Close()

		status := DeriveAgentStatus(sessionID)
		if status != types.AgentStatusIdle {
			t.Errorf("Expected %q for different-run AskUserQuestion, got %q", types.AgentStatusIdle, status)
		}
	})

	t.Run("RUN_ERROR returns idle", func(t *testing.T) {
		sessionID := "test-session-run-error"
		sessionsDir := filepath.Join(tmpDir, "sessions", sessionID)
		if err := os.MkdirAll(sessionsDir, 0755); err != nil {
			t.Fatalf("Failed to create sessions dir: %v", err)
		}

		// Create events file with RUN_STARTED then RUN_ERROR
		events := []map[string]interface{}{
			{"type": types.EventTypeRunStarted, "runId": "run-123"},
			{"type": types.EventTypeRunError, "runId": "run-123"},
		}
		eventsFile := filepath.Join(sessionsDir, "agui-events.jsonl")
		f, err := os.Create(eventsFile)
		if err != nil {
			t.Fatalf("Failed to create events file: %v", err)
		}
		for _, evt := range events {
			data, _ := json.Marshal(evt)
			f.Write(append(data, '\n'))
		}
		f.Close()

		status := DeriveAgentStatus(sessionID)
		if status != types.AgentStatusIdle {
			t.Errorf("Expected %q for RUN_ERROR, got %q", types.AgentStatusIdle, status)
		}
	})

	t.Run("case-insensitive AskUserQuestion detection", func(t *testing.T) {
		sessionID := "test-session-case-insensitive"
		sessionsDir := filepath.Join(tmpDir, "sessions", sessionID)
		if err := os.MkdirAll(sessionsDir, 0755); err != nil {
			t.Fatalf("Failed to create sessions dir: %v", err)
		}

		// Test various casings of AskUserQuestion
		testCases := []string{"askuserquestion", "ASKUSERQUESTION", "AskUserQuestion", "AsKuSeRqUeStIoN"}
		for _, toolName := range testCases {
			events := []map[string]interface{}{
				{"type": types.EventTypeRunStarted, "runId": "run-123"},
				{"type": types.EventTypeToolCallStart, "runId": "run-123", "toolCallName": toolName},
				{"type": types.EventTypeRunFinished, "runId": "run-123"},
			}
			eventsFile := filepath.Join(sessionsDir, "agui-events.jsonl")
			f, err := os.Create(eventsFile)
			if err != nil {
				t.Fatalf("Failed to create events file: %v", err)
			}
			for _, evt := range events {
				data, _ := json.Marshal(evt)
				f.Write(append(data, '\n'))
			}
			f.Close()

			status := DeriveAgentStatus(sessionID)
			if status != types.AgentStatusWaitingInput {
				t.Errorf("Expected %q for toolName %q, got %q", types.AgentStatusWaitingInput, toolName, status)
			}
		}
	})

	t.Run("non-existent session returns empty status", func(t *testing.T) {
		status := DeriveAgentStatus("non-existent-session")
		if status != "" {
			t.Errorf("Expected empty status for non-existent session, got %q", status)
		}
	})
}
