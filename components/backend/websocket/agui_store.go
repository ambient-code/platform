// Package websocket provides AG-UI protocol endpoints for event streaming.
//
// agui_store.go — Event persistence and snapshot assembly.
//
// Write path: append every event to agui-events.jsonl (append-only log).
// Read path:  load events, find the last MESSAGES_SNAPSHOT (sent by the
//             runner at the end of each run), merge any newer messages
//             from subsequent RUN_STARTEDs, and repair orphaned child
//             tool results.
package websocket

import (
	"ambient-code-backend/types"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// StateBaseDir is the root directory for session state persistence.
// Set from the STATE_BASE_DIR env var (default "/workspace") at startup.
var StateBaseDir string

// ─── Write path ──────────────────────────────────────────────────────

// persistEvent appends a single AG-UI event to the session's JSONL log.
func persistEvent(sessionID string, event map[string]interface{}) {
	dir := fmt.Sprintf("%s/sessions/%s", StateBaseDir, sessionID)
	path := dir + "/agui-events.jsonl"
	_ = ensureDir(dir)

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("AGUI Store: failed to marshal event: %v", err)
		return
	}

	f, err := openFileAppend(path)
	if err != nil {
		log.Printf("AGUI Store: failed to open event log: %v", err)
		return
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		log.Printf("AGUI Store: failed to write event: %v", err)
	}
}

// ─── Read path ───────────────────────────────────────────────────────

// loadEvents reads all AG-UI events for a session from the JSONL log.
// Automatically triggers legacy migration if the log doesn't exist but
// a pre-AG-UI messages.jsonl file does.
func loadEvents(sessionID string) []map[string]interface{} {
	path := fmt.Sprintf("%s/sessions/%s/agui-events.jsonl", StateBaseDir, sessionID)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Attempt legacy migration (messages.jsonl → agui-events.jsonl)
			if mErr := MigrateLegacySessionToAGUI(sessionID); mErr != nil {
				log.Printf("AGUI Store: legacy migration failed for %s: %v", sessionID, mErr)
			}
			// Retry after migration
			data, err = os.ReadFile(path)
			if err != nil {
				return nil
			}
		} else {
			log.Printf("AGUI Store: failed to read event log for %s: %v", sessionID, err)
			return nil
		}
	}

	events := make([]map[string]interface{}, 0, 64)
	for _, line := range splitLines(data) {
		if len(line) == 0 {
			continue
		}
		var evt map[string]interface{}
		if err := json.Unmarshal(line, &evt); err == nil {
			events = append(events, evt)
		}
	}
	return events
}

// ─── Compaction ──────────────────────────────────────────────────────

// compactEvents builds a MESSAGES_SNAPSHOT for reconnect.
//
// Simple approach:
//  1. Use the last MESSAGES_SNAPSHOT from the event log (the runner
//     sends one at the end of each completed run — it's authoritative).
//  2. If no snapshot exists, fall back to the last RUN_STARTED's
//     input.messages (the conversation CopilotKit sent us).
//  3. Pick up any new messages from RUN_STARTEDs after the snapshot
//     (handles: user sends a message then refreshes before the runner
//     finishes — their new message is in the next RUN_STARTED).
//  4. Repair orphaned child tool results (sub-agent tool calls missing
//     from the snapshot's toolCalls definitions).
//
// Returns nil if no message data exists.
func compactEvents(events []map[string]interface{}) map[string]interface{} {
	if len(events) == 0 {
		return nil
	}

	// 1. Find the last MESSAGES_SNAPSHOT
	var snapshotMessages []interface{}
	snapshotIdx := -1
	for i, evt := range events {
		if t, _ := evt["type"].(string); t == types.EventTypeMessagesSnapshot {
			if msgs, ok := evt["messages"].([]interface{}); ok && len(msgs) > 0 {
				snapshotMessages = msgs
				snapshotIdx = i
			}
		}
	}

	// 2. No snapshot? Use the last RUN_STARTED.input.messages
	if snapshotMessages == nil {
		for i := len(events) - 1; i >= 0; i-- {
			if t, _ := events[i]["type"].(string); t == types.EventTypeRunStarted {
				input, _ := events[i]["input"].(map[string]interface{})
				if input == nil {
					continue
				}
				if msgs, _ := input["messages"].([]interface{}); len(msgs) > 0 {
					snapshotMessages = msgs
					snapshotIdx = i
					break
				}
			}
		}
	}

	if snapshotMessages == nil {
		return nil
	}

	// 3. Collect new messages from RUN_STARTEDs after the snapshot
	//    (dedup by ID against the snapshot)
	seenIDs := make(map[string]bool, len(snapshotMessages))
	for _, m := range snapshotMessages {
		if msg, ok := m.(map[string]interface{}); ok {
			if id, _ := msg["id"].(string); id != "" {
				seenIDs[id] = true
			}
		}
	}

	var extraMessages []interface{}
	for _, evt := range events[snapshotIdx+1:] {
		if t, _ := evt["type"].(string); t != types.EventTypeRunStarted {
			continue
		}
		input, _ := evt["input"].(map[string]interface{})
		if input == nil {
			continue
		}
		msgs, _ := input["messages"].([]interface{})
		for _, m := range msgs {
			msg, ok := m.(map[string]interface{})
			if !ok {
				continue
			}
			id, _ := msg["id"].(string)
			if id != "" && !seenIDs[id] {
				seenIDs[id] = true
				extraMessages = append(extraMessages, msg)
			}
		}
	}

	allMessages := make([]interface{}, 0, len(snapshotMessages)+len(extraMessages))
	allMessages = append(allMessages, snapshotMessages...)
	allMessages = append(allMessages, extraMessages...)

	// 4. Repair orphaned child tool results
	allMessages = repairOrphanedToolResults(allMessages, events)

	if len(allMessages) == 0 {
		return nil
	}

	return map[string]interface{}{
		"type":     types.EventTypeMessagesSnapshot,
		"messages": allMessages,
	}
}

// repairOrphanedToolResults scans the message list for tool result
// messages (role:"tool") whose toolCallId doesn't match any tool call
// defined in an assistant message's toolCalls array.  For each group of
// orphaned results, it scans the full event log for the corresponding
// TOOL_CALL_START/END events and creates an assistant message with
// proper toolCalls definitions, inserted just before the first orphan.
//
// This ensures CopilotKit can match every tool result to a tool call
// definition, even for sub-agent child tool calls that the runner's
// MESSAGES_SNAPSHOT didn't include.
func repairOrphanedToolResults(messages []interface{}, events []map[string]interface{}) []interface{} {
	// Collect all defined tool call IDs from assistant messages
	definedToolCallIDs := make(map[string]bool)
	for _, m := range messages {
		collectToolCallIDs(m, definedToolCallIDs)
	}

	// Find orphaned tool result toolCallIds
	var orphanedIDs []string
	for _, m := range messages {
		msg, ok := m.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := msg["role"].(string)
		if role != "tool" {
			continue
		}
		tcID, _ := msg["toolCallId"].(string)
		if tcID != "" && !definedToolCallIDs[tcID] {
			orphanedIDs = append(orphanedIDs, tcID)
		}
	}

	if len(orphanedIDs) == 0 {
		return messages
	}

	log.Printf("AGUI Store: repairing %d orphaned tool results from event log", len(orphanedIDs))

	// Build tool call definitions from the event log
	orphanSet := make(map[string]bool, len(orphanedIDs))
	for _, id := range orphanedIDs {
		orphanSet[id] = true
	}

	// Scan events for TOOL_CALL_START/ARGS/END to reconstruct definitions
	type toolDef struct {
		id   string
		name string
		args string
	}
	toolDefs := make(map[string]*toolDef)
	for _, evt := range events {
		eventType, _ := evt["type"].(string)
		switch eventType {
		case types.EventTypeToolCallStart:
			tcID, _ := evt["toolCallId"].(string)
			if !orphanSet[tcID] {
				continue
			}
			tcName, _ := evt["toolCallName"].(string)
			toolDefs[tcID] = &toolDef{id: tcID, name: tcName}
		case types.EventTypeToolCallArgs:
			tcID, _ := evt["toolCallId"].(string)
			if td, ok := toolDefs[tcID]; ok {
				delta, _ := evt["delta"].(string)
				td.args += delta
			}
		}
	}

	// Build toolCalls array for the repair assistant message
	var repairedToolCalls []map[string]interface{}
	for _, id := range orphanedIDs {
		td, ok := toolDefs[id]
		if !ok {
			continue
		}
		repairedToolCalls = append(repairedToolCalls, map[string]interface{}{
			"id":   td.id,
			"type": "function",
			"function": map[string]interface{}{
				"name":      td.name,
				"arguments": td.args,
			},
		})
	}

	if len(repairedToolCalls) == 0 {
		return messages
	}

	// Create a synthetic assistant message with the missing tool calls.
	// Insert it just before the first orphaned tool result so the order
	// is: assistant(toolCalls) → tool results.
	repairMsg := map[string]interface{}{
		"id":        generateEventID(),
		"role":      "assistant",
		"toolCalls": repairedToolCalls,
	}

	// Find insertion point: just before first orphaned result
	insertIdx := len(messages)
	for i, m := range messages {
		msg, ok := m.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := msg["role"].(string)
		tcID, _ := msg["toolCallId"].(string)
		if role == "tool" && orphanSet[tcID] {
			insertIdx = i
			break
		}
	}

	// Insert the repair message
	result := make([]interface{}, 0, len(messages)+1)
	result = append(result, messages[:insertIdx]...)
	result = append(result, repairMsg)
	result = append(result, messages[insertIdx:]...)

	log.Printf("AGUI Store: inserted synthetic assistant message with %d child tool calls", len(repairedToolCalls))

	return result
}

// collectToolCallIDs extracts all toolCall IDs from an assistant message
// (any message with a toolCalls array) and adds them to the provided set.
// Used by compaction to identify which tool-result messages are valid
// (i.e. have a matching parent assistant tool call) vs. orphaned sub-agent
// internal results.
func collectToolCallIDs(m interface{}, ids map[string]bool) {
	msg, ok := m.(map[string]interface{})
	if !ok {
		return
	}
	tcs, ok := msg["toolCalls"].([]interface{})
	if !ok {
		return
	}
	for _, tc := range tcs {
		tcMap, ok := tc.(map[string]interface{})
		if !ok {
			continue
		}
		if id, ok := tcMap["id"].(string); ok && id != "" {
			ids[id] = true
		}
	}
}

// collectCustomEvents returns CUSTOM events from the event log for replay.
// Custom events are not messages and cannot be included in MESSAGES_SNAPSHOT.
// The proxy replays them as separate SSE events after the snapshot so
// CopilotKit can re-process frontend actions, state, etc.
//
// META events (user feedback — thumbs up/down) are also collected and
// converted to CUSTOM events with name "ambient:feedback", because META
// is a draft event type not yet in the AG-UI core Zod schema.
func collectCustomEvents(events []map[string]interface{}) []map[string]interface{} {
	var custom []map[string]interface{}
	for _, evt := range events {
		t, _ := evt["type"].(string)
		switch t {
		case types.EventTypeCustom:
			custom = append(custom, evt)
		case types.EventTypeMeta:
			// Convert META → CUSTOM so it passes Zod validation
			custom = append(custom, map[string]interface{}{
				"type": types.EventTypeCustom,
				"name": "ambient:feedback",
				"value": map[string]interface{}{
					"metaType": evt["metaType"],
					"payload":  evt["payload"],
				},
			})
		}
	}
	return custom
}

// ─── Timestamp sanitization ──────────────────────────────────────────

// sanitizeEventTimestamp ensures the "timestamp" field in an event map
// is an epoch-millisecond number (int64 / float64), as required by the
// AG-UI protocol (BaseEventSchema: z.number().optional()).
//
// Old persisted events may contain ISO-8601 strings — this converts
// them to epoch ms for backward compatibility.  If the value is already
// a number or absent, it is left untouched.
func sanitizeEventTimestamp(evt map[string]interface{}) {
	ts, ok := evt["timestamp"]
	if !ok || ts == nil {
		return // absent — fine, it's optional
	}

	switch ts.(type) {
	case float64, int64, json.Number:
		return // already a number — nothing to do
	case string:
		s := ts.(string)
		if s == "" {
			delete(evt, "timestamp")
			return
		}
		// Try parsing as RFC3339 / RFC3339Nano (the old format)
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
			if t, err := time.Parse(layout, s); err == nil {
				evt["timestamp"] = t.UnixMilli()
				return
			}
		}
		// Unparseable string — remove rather than send invalid data
		log.Printf("AGUI Store: removing unparseable timestamp %q", s)
		delete(evt, "timestamp")
	}
}

// ─── SSE helpers ─────────────────────────────────────────────────────

// writeSSEEvent marshals an event and writes it in SSE data: format.
// If the event is a map, timestamps are sanitized to epoch ms first.
func writeSSEEvent(w http.ResponseWriter, event interface{}) {
	// Sanitize timestamps on map events (replayed from store)
	if m, ok := event.(map[string]interface{}); ok {
		sanitizeEventTimestamp(m)
	}
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("AGUI Store: failed to marshal SSE event: %v", err)
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// ─── File helpers ────────────────────────────────────────────────────

func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func openFileAppend(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
