// Package websocket provides AG-UI protocol endpoints for event streaming.
//
// agui_store.go — Event persistence, compaction, and replay.
//
// Write path:  append every event to agui-events.jsonl (append-only log).
// Read path:   load events and compact streaming deltas for efficient replay.
// Compaction:  mirrors @ag-ui/client compactEvents — concatenates
//              TEXT_MESSAGE_CONTENT and TOOL_CALL_ARGS deltas, preserves
//              all other events as-is.
//
// On reconnect the proxy replays the compacted events individually
// (matching InMemoryAgentRunner.connect()), NOT a rebuilt MESSAGES_SNAPSHOT.
package websocket

import (
	"ambient-code-backend/types"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
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
//
// Go port of @ag-ui/client compactEvents (compact.ts).
//
// Mirrors the InMemoryAgentRunner pattern:
//   - During run:    raw events are appended to the JSONL log.
//   - After run:     compactStreamingEvents shrinks deltas and the log
//                    is rewritten atomically.
//   - On connect():  compacted events are replayed individually.

// pendingText tracks an in-progress TEXT_MESSAGE sequence.
type pendingText struct {
	start       map[string]interface{}
	deltas      []string // accumulated delta strings
	end         map[string]interface{}
	otherEvents []map[string]interface{}
}

// pendingTool tracks an in-progress TOOL_CALL sequence.
type pendingTool struct {
	start       map[string]interface{}
	deltas      []string // accumulated delta strings
	end         map[string]interface{}
	otherEvents []map[string]interface{}
}

// compactStreamingEvents consolidates streaming deltas, matching
// @ag-ui/client compactEvents exactly:
//
//   - TEXT_MESSAGE_CONTENT events with the same messageId → one event
//     with concatenated delta.
//   - TOOL_CALL_ARGS events with the same toolCallId → one event with
//     concatenated delta.
//   - Events that arrive *between* START and END of a streaming
//     sequence are buffered and emitted after END (reordering).
//   - All other events pass through unchanged.
//   - Incomplete sequences (START without END) are flushed at the end.
func compactStreamingEvents(events []map[string]interface{}) []map[string]interface{} {
	compacted := make([]map[string]interface{}, 0, len(events)/2)

	// Ordered maps (Go maps don't preserve insertion order, so we
	// also track insertion order via slices).
	textByID := make(map[string]*pendingText)
	var textOrder []string

	toolByID := make(map[string]*pendingTool)
	var toolOrder []string

	getOrCreateText := func(id string) *pendingText {
		if p, ok := textByID[id]; ok {
			return p
		}
		p := &pendingText{}
		textByID[id] = p
		textOrder = append(textOrder, id)
		return p
	}

	getOrCreateTool := func(id string) *pendingTool {
		if p, ok := toolByID[id]; ok {
			return p
		}
		p := &pendingTool{}
		toolByID[id] = p
		toolOrder = append(toolOrder, id)
		return p
	}

	flushText := func(msgID string) {
		p := textByID[msgID]
		if p == nil {
			return
		}
		if p.start != nil {
			compacted = append(compacted, p.start)
		}
		if len(p.deltas) > 0 {
			concatenated := ""
			for _, d := range p.deltas {
				concatenated += d
			}
			compacted = append(compacted, map[string]interface{}{
				"type":      types.EventTypeTextMessageContent,
				"messageId": msgID,
				"delta":     concatenated,
			})
		}
		if p.end != nil {
			compacted = append(compacted, p.end)
		}
		for _, other := range p.otherEvents {
			compacted = append(compacted, other)
		}
		delete(textByID, msgID)
	}

	flushTool := func(tcID string) {
		p := toolByID[tcID]
		if p == nil {
			return
		}
		if p.start != nil {
			compacted = append(compacted, p.start)
		}
		if len(p.deltas) > 0 {
			concatenated := ""
			for _, d := range p.deltas {
				concatenated += d
			}
			compacted = append(compacted, map[string]interface{}{
				"type":       types.EventTypeToolCallArgs,
				"toolCallId": tcID,
				"delta":      concatenated,
			})
		}
		if p.end != nil {
			compacted = append(compacted, p.end)
		}
		for _, other := range p.otherEvents {
			compacted = append(compacted, other)
		}
		delete(toolByID, tcID)
	}

	for _, evt := range events {
		eventType, _ := evt["type"].(string)

		switch eventType {
		// ── Text message streaming ──
		case types.EventTypeTextMessageStart:
			msgID, _ := evt["messageId"].(string)
			if msgID == "" {
				compacted = append(compacted, evt)
				continue
			}
			p := getOrCreateText(msgID)
			p.start = evt

		case types.EventTypeTextMessageContent:
			msgID, _ := evt["messageId"].(string)
			if msgID == "" {
				compacted = append(compacted, evt)
				continue
			}
			p := getOrCreateText(msgID)
			delta, _ := evt["delta"].(string)
			p.deltas = append(p.deltas, delta)

		case types.EventTypeTextMessageEnd:
			msgID, _ := evt["messageId"].(string)
			if msgID == "" {
				compacted = append(compacted, evt)
				continue
			}
			p := getOrCreateText(msgID)
			p.end = evt
			flushText(msgID)

		// ── Tool call streaming ──
		case types.EventTypeToolCallStart:
			tcID, _ := evt["toolCallId"].(string)
			if tcID == "" {
				compacted = append(compacted, evt)
				continue
			}
			p := getOrCreateTool(tcID)
			p.start = evt

		case types.EventTypeToolCallArgs:
			tcID, _ := evt["toolCallId"].(string)
			if tcID == "" {
				compacted = append(compacted, evt)
				continue
			}
			p := getOrCreateTool(tcID)
			delta, _ := evt["delta"].(string)
			p.deltas = append(p.deltas, delta)

		case types.EventTypeToolCallEnd:
			tcID, _ := evt["toolCallId"].(string)
			if tcID == "" {
				compacted = append(compacted, evt)
				continue
			}
			p := getOrCreateTool(tcID)
			p.end = evt
			flushTool(tcID)

		// ── Everything else (pass-through or buffer) ──
		default:
			// If we're inside an open streaming sequence, buffer the event
			// so it gets emitted after the sequence closes (reordering).
			buffered := false
			for _, id := range textOrder {
				p := textByID[id]
				if p != nil && p.start != nil && p.end == nil {
					p.otherEvents = append(p.otherEvents, evt)
					buffered = true
					break
				}
			}
			if !buffered {
				for _, id := range toolOrder {
					p := toolByID[id]
					if p != nil && p.start != nil && p.end == nil {
						p.otherEvents = append(p.otherEvents, evt)
						buffered = true
						break
					}
				}
			}
			if !buffered {
				compacted = append(compacted, evt)
			}
		}
	}

	// Flush any remaining incomplete sequences (mid-run reconnect).
	for _, id := range textOrder {
		if textByID[id] != nil {
			flushText(id)
		}
	}
	for _, id := range toolOrder {
		if toolByID[id] != nil {
			flushTool(id)
		}
	}

	return compacted
}

// ─── Post-run log compaction ─────────────────────────────────────────

// rewriteEventLog atomically replaces the session's JSONL log with the
// provided events.  Writes to a temp file then renames, so concurrent
// readers either see the old file or the new one — never a partial write.
func rewriteEventLog(sessionID string, events []map[string]interface{}) {
	dir := fmt.Sprintf("%s/sessions/%s", StateBaseDir, sessionID)
	target := filepath.Join(dir, "agui-events.jsonl")
	tmp := target + ".tmp"

	f, err := os.Create(tmp)
	if err != nil {
		log.Printf("AGUI Store: compact rewrite failed (create tmp): %v", err)
		return
	}

	for _, evt := range events {
		data, err := json.Marshal(evt)
		if err != nil {
			continue
		}
		f.Write(append(data, '\n'))
	}
	f.Close()

	if err := os.Rename(tmp, target); err != nil {
		log.Printf("AGUI Store: compact rewrite failed (rename): %v", err)
		os.Remove(tmp)
	}
}

// ─── Reconnect cache ─────────────────────────────────────────────────

type reconnectCacheEntry struct {
	events    []map[string]interface{}
	timestamp time.Time
}

var (
	reconnectCache    sync.Map // sessionName → *reconnectCacheEntry
	reconnectCacheTTL = 2 * time.Second
)

// getCachedReconnectEvents returns cached compacted events if the cache
// entry is younger than reconnectCacheTTL.  Returns nil on miss.
func getCachedReconnectEvents(sessionName string) []map[string]interface{} {
	val, ok := reconnectCache.Load(sessionName)
	if !ok {
		return nil
	}
	entry := val.(*reconnectCacheEntry)
	if time.Since(entry.timestamp) > reconnectCacheTTL {
		reconnectCache.Delete(sessionName)
		return nil
	}
	return entry.events
}

// setCachedReconnectEvents stores compacted events in the cache.
func setCachedReconnectEvents(sessionName string, events []map[string]interface{}) {
	reconnectCache.Store(sessionName, &reconnectCacheEntry{
		events:    events,
		timestamp: time.Now(),
	})
}

// invalidateReconnectCache removes the cache entry for a session.
// Called when a new run starts (hasMessages = true).
func invalidateReconnectCache(sessionName string) {
	reconnectCache.Delete(sessionName)
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
