// Package websocket provides AG-UI protocol endpoints for event streaming.
//
// agui_store.go — Event persistence (append-only JSONL).
//
// The backend proxy persists every AG-UI event to disk as a side effect
// of piping them from the runner to the frontend.  This is purely a
// backup for cross-restart recovery — InMemoryAgentRunner on the
// frontend handles reconnection and in-session persistence.
package websocket

import (
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

	switch v := ts.(type) {
	case float64, int64, json.Number:
		return // already a number — nothing to do
	case string:
		if v == "" {
			delete(evt, "timestamp")
			return
		}
		// Try parsing as RFC3339 / RFC3339Nano (the old format)
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
			if t, err := time.Parse(layout, v); err == nil {
				evt["timestamp"] = t.UnixMilli()
				return
			}
		}
		// Unparseable string — remove rather than send invalid data
		log.Printf("AGUI Store: removing unparseable timestamp %q", v)
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
