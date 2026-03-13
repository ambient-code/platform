package websocket

import (
	"ambient-code-backend/types"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestCompactToSnapshots(t *testing.T) {
	t.Run("collapses text messages into MESSAGES_SNAPSHOT", func(t *testing.T) {
		events := []map[string]interface{}{
			{"type": types.EventTypeRunStarted, "runId": "r1", "threadId": "t1"},
			{"type": types.EventTypeTextMessageStart, "messageId": "msg1", "role": "user"},
			{"type": types.EventTypeTextMessageContent, "messageId": "msg1", "delta": "Hello "},
			{"type": types.EventTypeTextMessageContent, "messageId": "msg1", "delta": "world"},
			{"type": types.EventTypeTextMessageEnd, "messageId": "msg1"},
			{"type": types.EventTypeRunFinished, "runId": "r1", "threadId": "t1"},
		}

		result := compactToSnapshots(events)

		// Should have: RUN_STARTED + RUN_FINISHED + MESSAGES_SNAPSHOT
		if len(result) != 3 {
			t.Fatalf("Expected 3 events, got %d: %+v", len(result), result)
		}

		// Check lifecycle events pass through
		if result[0]["type"] != types.EventTypeRunStarted {
			t.Errorf("Expected RUN_STARTED, got %v", result[0]["type"])
		}
		if result[1]["type"] != types.EventTypeRunFinished {
			t.Errorf("Expected RUN_FINISHED, got %v", result[1]["type"])
		}

		// Check MESSAGES_SNAPSHOT
		snap := result[2]
		if snap["type"] != types.EventTypeMessagesSnapshot {
			t.Fatalf("Expected MESSAGES_SNAPSHOT, got %v", snap["type"])
		}
		msgs, ok := snap["messages"].([]map[string]interface{})
		if !ok {
			t.Fatalf("messages field is not []map[string]interface{}")
		}
		if len(msgs) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(msgs))
		}
		if msgs[0]["content"] != "Hello world" {
			t.Errorf("Expected 'Hello world', got %v", msgs[0]["content"])
		}
		if msgs[0]["role"] != "user" {
			t.Errorf("Expected role 'user', got %v", msgs[0]["role"])
		}
	})

	t.Run("handles tool calls with results", func(t *testing.T) {
		events := []map[string]interface{}{
			{"type": types.EventTypeRunStarted, "runId": "r1"},
			{"type": types.EventTypeTextMessageStart, "messageId": "msg1", "role": "assistant"},
			{"type": types.EventTypeTextMessageContent, "messageId": "msg1", "delta": "Let me help"},
			{"type": types.EventTypeTextMessageEnd, "messageId": "msg1"},
			{"type": types.EventTypeToolCallStart, "toolCallId": "tc1", "toolCallName": "Read", "messageId": "msg1"},
			{"type": types.EventTypeToolCallArgs, "toolCallId": "tc1", "delta": "{\"path\":"},
			{"type": types.EventTypeToolCallArgs, "toolCallId": "tc1", "delta": "\"/foo\"}"},
			{"type": types.EventTypeToolCallEnd, "toolCallId": "tc1", "result": "file contents here"},
			{"type": types.EventTypeRunFinished, "runId": "r1"},
		}

		result := compactToSnapshots(events)

		// Find MESSAGES_SNAPSHOT
		var snap map[string]interface{}
		for _, evt := range result {
			if evt["type"] == types.EventTypeMessagesSnapshot {
				snap = evt
				break
			}
		}
		if snap == nil {
			t.Fatal("No MESSAGES_SNAPSHOT found")
		}

		msgs := snap["messages"].([]map[string]interface{})
		// Should have: assistant message + tool result message
		if len(msgs) != 2 {
			t.Fatalf("Expected 2 messages, got %d: %+v", len(msgs), msgs)
		}

		// Check assistant message has tool calls
		assistantMsg := msgs[0]
		if assistantMsg["role"] != "assistant" {
			t.Errorf("Expected assistant role, got %v", assistantMsg["role"])
		}
		toolCalls, ok := assistantMsg["toolCalls"].([]map[string]interface{})
		if !ok || len(toolCalls) != 1 {
			t.Fatalf("Expected 1 tool call, got %v", assistantMsg["toolCalls"])
		}
		if toolCalls[0]["name"] != "Read" {
			t.Errorf("Expected tool name 'Read', got %v", toolCalls[0]["name"])
		}
		if toolCalls[0]["args"] != "{\"path\":\"/foo\"}" {
			t.Errorf("Expected concatenated args, got %v", toolCalls[0]["args"])
		}

		// Check tool result message
		toolMsg := msgs[1]
		if toolMsg["role"] != types.RoleTool {
			t.Errorf("Expected tool role, got %v", toolMsg["role"])
		}
		if toolMsg["content"] != "file contents here" {
			t.Errorf("Expected tool result content, got %v", toolMsg["content"])
		}
	})

	t.Run("passes through RAW events", func(t *testing.T) {
		events := []map[string]interface{}{
			{"type": types.EventTypeRunStarted, "runId": "r1"},
			{"type": types.EventTypeRaw, "event": map[string]interface{}{"type": "message_metadata", "hidden": true}},
			{"type": types.EventTypeRunFinished, "runId": "r1"},
		}

		result := compactToSnapshots(events)

		rawCount := 0
		for _, evt := range result {
			if evt["type"] == types.EventTypeRaw {
				rawCount++
			}
		}
		if rawCount != 1 {
			t.Errorf("Expected 1 RAW event, got %d", rawCount)
		}
	})

	t.Run("handles multiple messages", func(t *testing.T) {
		events := []map[string]interface{}{
			{"type": types.EventTypeRunStarted, "runId": "r1"},
			{"type": types.EventTypeTextMessageStart, "messageId": "msg1", "role": "user"},
			{"type": types.EventTypeTextMessageContent, "messageId": "msg1", "delta": "Hello"},
			{"type": types.EventTypeTextMessageEnd, "messageId": "msg1"},
			{"type": types.EventTypeTextMessageStart, "messageId": "msg2", "role": "assistant"},
			{"type": types.EventTypeTextMessageContent, "messageId": "msg2", "delta": "Hi there!"},
			{"type": types.EventTypeTextMessageEnd, "messageId": "msg2"},
			{"type": types.EventTypeRunFinished, "runId": "r1"},
		}

		result := compactToSnapshots(events)

		var snap map[string]interface{}
		for _, evt := range result {
			if evt["type"] == types.EventTypeMessagesSnapshot {
				snap = evt
				break
			}
		}
		if snap == nil {
			t.Fatal("No MESSAGES_SNAPSHOT found")
		}

		msgs := snap["messages"].([]map[string]interface{})
		if len(msgs) != 2 {
			t.Fatalf("Expected 2 messages, got %d", len(msgs))
		}
		if msgs[0]["content"] != "Hello" {
			t.Errorf("Expected 'Hello', got %v", msgs[0]["content"])
		}
		if msgs[1]["content"] != "Hi there!" {
			t.Errorf("Expected 'Hi there!', got %v", msgs[1]["content"])
		}
	})

	t.Run("empty events returns empty result", func(t *testing.T) {
		result := compactToSnapshots(nil)
		if len(result) != 0 {
			t.Errorf("Expected 0 events, got %d", len(result))
		}
	})

	t.Run("preserves message metadata", func(t *testing.T) {
		events := []map[string]interface{}{
			{"type": types.EventTypeTextMessageStart, "messageId": "msg1", "role": "user", "metadata": map[string]interface{}{"hidden": true}},
			{"type": types.EventTypeTextMessageContent, "messageId": "msg1", "delta": "test"},
			{"type": types.EventTypeTextMessageEnd, "messageId": "msg1"},
		}

		result := compactToSnapshots(events)

		var snap map[string]interface{}
		for _, evt := range result {
			if evt["type"] == types.EventTypeMessagesSnapshot {
				snap = evt
				break
			}
		}
		if snap == nil {
			t.Fatal("No MESSAGES_SNAPSHOT found")
		}
		msgs := snap["messages"].([]map[string]interface{})
		meta, ok := msgs[0]["metadata"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected metadata on message")
		}
		if meta["hidden"] != true {
			t.Errorf("Expected hidden=true, got %v", meta["hidden"])
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

	t.Run("finished session returns snapshot-compacted events", func(t *testing.T) {
		sessionID := "test-replay-finished"
		writeEvents(sessionID, []map[string]interface{}{
			{"type": types.EventTypeRunStarted, "runId": "r1"},
			{"type": types.EventTypeTextMessageStart, "messageId": "msg1", "role": "user"},
			{"type": types.EventTypeTextMessageContent, "messageId": "msg1", "delta": "Hello"},
			{"type": types.EventTypeTextMessageEnd, "messageId": "msg1"},
			{"type": types.EventTypeRunFinished, "runId": "r1"},
		})

		result := loadEventsForReplay(sessionID)

		// Should be compacted: RUN_STARTED + MESSAGES_SNAPSHOT + RUN_FINISHED
		if len(result) != 3 {
			t.Fatalf("Expected 3 events, got %d", len(result))
		}

		hasSnapshot := false
		for _, evt := range result {
			if evt["type"] == types.EventTypeMessagesSnapshot {
				hasSnapshot = true
			}
		}
		if !hasSnapshot {
			t.Error("Expected MESSAGES_SNAPSHOT in replay events")
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

	t.Run("writes and serves from compacted cache", func(t *testing.T) {
		sessionID := "test-replay-cache"
		writeEvents(sessionID, []map[string]interface{}{
			{"type": types.EventTypeRunStarted, "runId": "r1"},
			{"type": types.EventTypeTextMessageStart, "messageId": "msg1", "role": "user"},
			{"type": types.EventTypeTextMessageContent, "messageId": "msg1", "delta": "cached"},
			{"type": types.EventTypeTextMessageEnd, "messageId": "msg1"},
			{"type": types.EventTypeRunFinished, "runId": "r1"},
		})

		// First call — compacts and writes cache
		result1 := loadEventsForReplay(sessionID)

		// Wait for async cache write
		time.Sleep(100 * time.Millisecond)

		// Verify compacted file exists
		compactedPath := filepath.Join(tmpDir, "sessions", sessionID, "agui-events-compacted.jsonl")
		if _, err := os.Stat(compactedPath); os.IsNotExist(err) {
			t.Fatal("Expected compacted file to be written")
		}

		// Second call — should serve from cache
		result2 := loadEventsForReplay(sessionID)

		if len(result1) != len(result2) {
			t.Errorf("Expected same event count from cache: %d vs %d", len(result1), len(result2))
		}
	})

	t.Run("cache invalidated on new RUN_STARTED", func(t *testing.T) {
		sessionID := "test-replay-invalidate"
		writeEvents(sessionID, []map[string]interface{}{
			{"type": types.EventTypeRunStarted, "runId": "r1"},
			{"type": types.EventTypeRunFinished, "runId": "r1"},
		})

		// Load to create cache
		loadEventsForReplay(sessionID)
		time.Sleep(100 * time.Millisecond)

		compactedPath := filepath.Join(tmpDir, "sessions", sessionID, "agui-events-compacted.jsonl")
		if _, err := os.Stat(compactedPath); os.IsNotExist(err) {
			t.Fatal("Expected compacted file to exist before invalidation")
		}

		// Simulate new run starting — persistEvent with RUN_STARTED should remove cache
		persistEvent(sessionID, map[string]interface{}{
			"type":  types.EventTypeRunStarted,
			"runId": "r2",
		})

		if _, err := os.Stat(compactedPath); !os.IsNotExist(err) {
			t.Error("Expected compacted file to be removed after RUN_STARTED")
		}
	})
}
