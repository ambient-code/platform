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

func TestLoadEventsForReplay(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agui-replay-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origStateBaseDir := StateBaseDir
	StateBaseDir = tmpDir
	defer func() { StateBaseDir = origStateBaseDir }()

	writeEvents := func(sessionID string, events []map[string]interface{}) {
		sessionsDir := filepath.Join(tmpDir, "sessions", sessionID)
		if err := os.MkdirAll(sessionsDir, 0755); err != nil {
			t.Fatalf("Failed to create sessions dir: %v", err)
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
	}

	t.Run("finished session with MESSAGES_SNAPSHOT gets compacted", func(t *testing.T) {
		sessionID := "test-replay-finished"
		// Simulate what the runner sends: streaming events + MESSAGES_SNAPSHOT + RUN_FINISHED
		writeEvents(sessionID, []map[string]interface{}{
			{"type": types.EventTypeRunStarted, "runId": "r1"},
			{"type": types.EventTypeTextMessageStart, "messageId": "msg1", "role": "user"},
			{"type": types.EventTypeTextMessageContent, "messageId": "msg1", "delta": "Hello"},
			{"type": types.EventTypeTextMessageEnd, "messageId": "msg1"},
			{"type": types.EventTypeMessagesSnapshot, "messages": []interface{}{
				map[string]interface{}{"id": "msg1", "role": "user", "content": "Hello"},
			}},
			{"type": types.EventTypeRunFinished, "runId": "r1"},
		})

		// Manually trigger compaction (in production, persistEvent does this on RUN_FINISHED)
		compactFinishedRun(sessionID)

		// Wait for async compaction

		result := loadEventsForReplay(sessionID)

		// Should be compacted: RUN_STARTED + MESSAGES_SNAPSHOT + RUN_FINISHED
		if len(result) != 3 {
			t.Fatalf("Expected 3 events after compaction, got %d", len(result))
		}

		hasSnapshot := false
		for _, evt := range result {
			if evt["type"] == types.EventTypeMessagesSnapshot {
				hasSnapshot = true
			}
			// Verify streaming events were removed
			eventType := evt["type"]
			if eventType == types.EventTypeTextMessageStart ||
				eventType == types.EventTypeTextMessageContent ||
				eventType == types.EventTypeTextMessageEnd {
				t.Errorf("Streaming event %s should have been removed", eventType)
			}
		}
		if !hasSnapshot {
			t.Error("Expected MESSAGES_SNAPSHOT in compacted events")
		}
	})

	t.Run("active session returns raw events", func(t *testing.T) {
		sessionID := "test-replay-active"
		writeEvents(sessionID, []map[string]interface{}{
			{"type": types.EventTypeRunStarted, "runId": "r1"},
			{"type": types.EventTypeTextMessageStart, "messageId": "msg1", "role": "user"},
			{"type": types.EventTypeTextMessageContent, "messageId": "msg1", "delta": "Hello"},
		})

		result := loadEventsForReplay(sessionID)

		// Active run — should return raw events unchanged
		if len(result) != 3 {
			t.Fatalf("Expected 3 raw events, got %d", len(result))
		}
		if result[0]["type"] != types.EventTypeRunStarted {
			t.Errorf("Expected RUN_STARTED, got %v", result[0]["type"])
		}
		if result[1]["type"] != types.EventTypeTextMessageStart {
			t.Errorf("Expected TEXT_MESSAGE_START, got %v", result[1]["type"])
		}
	})

	t.Run("corrupted session without MESSAGES_SNAPSHOT keeps raw events", func(t *testing.T) {
		sessionID := "test-replay-corrupted"
		// Simulate a corrupted session: has RUN_FINISHED but no MESSAGES_SNAPSHOT
		writeEvents(sessionID, []map[string]interface{}{
			{"type": types.EventTypeRunStarted, "runId": "r1"},
			{"type": types.EventTypeTextMessageStart, "messageId": "msg1", "role": "user"},
			{"type": types.EventTypeTextMessageContent, "messageId": "msg1", "delta": "Hello"},
			{"type": types.EventTypeTextMessageEnd, "messageId": "msg1"},
			{"type": types.EventTypeRunFinished, "runId": "r1"},
		})

		// Try to compact (should fail gracefully and keep raw events)
		compactFinishedRun(sessionID)

		result := loadEventsForReplay(sessionID)

		// Should still have all raw events (compaction failed due to missing MESSAGES_SNAPSHOT)
		if len(result) != 5 {
			t.Fatalf("Expected 5 raw events (corruption fallback), got %d", len(result))
		}

		// Verify streaming events are still present
		hasStreamingEvents := false
		for _, evt := range result {
			eventType := evt["type"]
			if eventType == types.EventTypeTextMessageStart ||
				eventType == types.EventTypeTextMessageContent {
				hasStreamingEvents = true
			}
		}
		if !hasStreamingEvents {
			t.Error("Expected streaming events to be preserved for corrupted session")
		}
	})

	t.Run("STATE_SNAPSHOT is preserved during compaction", func(t *testing.T) {
		sessionID := "test-replay-state"
		// Simulate session with STATE_SNAPSHOT
		writeEvents(sessionID, []map[string]interface{}{
			{"type": types.EventTypeRunStarted, "runId": "r1"},
			{"type": types.EventTypeStateSnapshot, "snapshot": map[string]interface{}{"count": 42}},
			{"type": types.EventTypeTextMessageStart, "messageId": "msg1", "role": "assistant"},
			{"type": types.EventTypeTextMessageContent, "messageId": "msg1", "delta": "Done"},
			{"type": types.EventTypeTextMessageEnd, "messageId": "msg1"},
			{"type": types.EventTypeMessagesSnapshot, "messages": []interface{}{
				map[string]interface{}{"id": "msg1", "role": "assistant", "content": "Done"},
			}},
			{"type": types.EventTypeRunFinished, "runId": "r1"},
		})

		compactFinishedRun(sessionID)

		result := loadEventsForReplay(sessionID)

		// Should have: RUN_STARTED + STATE_SNAPSHOT + MESSAGES_SNAPSHOT + RUN_FINISHED = 4 events
		if len(result) != 4 {
			t.Fatalf("Expected 4 events after compaction, got %d", len(result))
		}

		hasStateSnapshot := false
		for _, evt := range result {
			if evt["type"] == types.EventTypeStateSnapshot {
				hasStateSnapshot = true
			}
		}
		if !hasStateSnapshot {
			t.Error("Expected STATE_SNAPSHOT to be preserved during compaction")
		}
	})

	t.Run("META and CUSTOM events are preserved during compaction", func(t *testing.T) {
		sessionID := "test-replay-custom"
		// Simulate session with user feedback and custom events
		writeEvents(sessionID, []map[string]interface{}{
			{"type": types.EventTypeRunStarted, "runId": "r1"},
			{"type": types.EventTypeTextMessageStart, "messageId": "msg1", "role": "assistant"},
			{"type": types.EventTypeTextMessageContent, "messageId": "msg1", "delta": "Hello"},
			{"type": types.EventTypeTextMessageEnd, "messageId": "msg1"},
			{"type": types.EventTypeMeta, "metaType": "thumbs_up", "payload": map[string]interface{}{"messageId": "msg1"}},
			{"type": types.EventTypeCustom, "customType": "platform_event", "data": "important"},
			{"type": types.EventTypeRaw, "event": map[string]interface{}{"type": "message_metadata", "hidden": true}},
			{"type": types.EventTypeMessagesSnapshot, "messages": []interface{}{
				map[string]interface{}{"id": "msg1", "role": "assistant", "content": "Hello"},
			}},
			{"type": types.EventTypeRunFinished, "runId": "r1"},
		})

		compactFinishedRun(sessionID)

		result := loadEventsForReplay(sessionID)

		// Should have: RUN_STARTED + META + CUSTOM + RAW + MESSAGES_SNAPSHOT + RUN_FINISHED = 6 events
		if len(result) != 6 {
			t.Fatalf("Expected 6 events after compaction, got %d", len(result))
		}

		hasMeta := false
		hasCustom := false
		hasRaw := false
		for _, evt := range result {
			eventType := evt["type"]
			if eventType == types.EventTypeMeta {
				hasMeta = true
			}
			if eventType == types.EventTypeCustom {
				hasCustom = true
			}
			if eventType == types.EventTypeRaw {
				hasRaw = true
			}
			// Verify streaming events were removed
			if eventType == types.EventTypeTextMessageStart ||
				eventType == types.EventTypeTextMessageContent ||
				eventType == types.EventTypeTextMessageEnd {
				t.Errorf("Streaming event %s should have been removed", eventType)
			}
		}
		if !hasMeta {
			t.Error("Expected META event to be preserved during compaction")
		}
		if !hasCustom {
			t.Error("Expected CUSTOM event to be preserved during compaction")
		}
		if !hasRaw {
			t.Error("Expected RAW event to be preserved during compaction")
		}
	})
}
