// Package websocket provides AG-UI protocol endpoints for event streaming.
//
// agui_store.go — Event persistence, compaction, and replay.
//
// Write path:  append every event to agui-events.jsonl.
// Read path:   load + compact events for reconnect replay.
// Compaction:  Go port of @ag-ui/client compactEvents — concatenates
//
//	TEXT_MESSAGE_CONTENT and TOOL_CALL_ARGS deltas.
package websocket

import (
	"ambient-code-backend/types"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// ─── Write mutex eviction ────────────────────────────────────────────
// writeMutexes entries are evicted after writeMutexEvictAge of inactivity
// to prevent unbounded sync.Map growth on long-running backends.

const writeMutexEvictAge = 30 * time.Minute

func init() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			evictStaleWriteMutexes()
		}
	}()
}

// evictStaleWriteMutexes removes write mutex entries that haven't been
// used within writeMutexEvictAge.
func evictStaleWriteMutexes() {
	threshold := time.Now().Add(-writeMutexEvictAge).Unix()
	writeMutexes.Range(func(key, value interface{}) bool {
		entry := value.(*writeMutexEntry)
		if atomic.LoadInt64(&entry.lastUsed) < threshold {
			writeMutexes.Delete(key)
		}
		return true
	})
}

// StateBaseDir is the root directory for session state persistence.
// Set from the STATE_BASE_DIR env var (default "/workspace") at startup.
var StateBaseDir string

// ─── Live event pipe (multi-client broadcast) ───────────────────────
// The run handler pipes raw SSE lines to ALL connect handlers tailing
// the same session.  Zero latency — same as the direct run() path.

type sessionBroadcast struct {
	mu   sync.Mutex
	subs map[int]chan string
	next int
}

var liveBroadcasts sync.Map // sessionName → *sessionBroadcast

func getBroadcast(sessionName string) *sessionBroadcast {
	val, _ := liveBroadcasts.LoadOrStore(sessionName, &sessionBroadcast{
		subs: make(map[int]chan string),
	})
	return val.(*sessionBroadcast)
}

// publishLine sends a raw SSE line to ALL connect handlers tailing this session.
func publishLine(sessionName, line string) {
	b := getBroadcast(sessionName)
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subs {
		select {
		case ch <- line:
		default: // slow client — drop (it's persisted to JSONL)
		}
	}
}

// subscribeLive creates a channel to receive live SSE lines for a session.
// Multiple clients can subscribe to the same session simultaneously.
func subscribeLive(sessionName string) (<-chan string, func()) {
	b := getBroadcast(sessionName)
	ch := make(chan string, 256)

	b.mu.Lock()
	id := b.next
	b.next++
	b.subs[id] = ch
	b.mu.Unlock()

	return ch, func() {
		b.mu.Lock()
		delete(b.subs, id)
		b.mu.Unlock()
	}
}

// ─── Write path ──────────────────────────────────────────────────────

// writeMutexEntry wraps a per-session mutex with a last-used timestamp
// for eviction of idle entries.
type writeMutexEntry struct {
	mu       sync.Mutex
	lastUsed int64 // unix seconds, updated atomically
}

// writeMutexes serialises JSONL appends per session, preventing
// interleaved writes from concurrent goroutines (e.g. run handler +
// feedback handler writing to the same session file simultaneously).
var writeMutexes sync.Map // sessionID → *writeMutexEntry

func getWriteMutex(sessionID string) *sync.Mutex {
	now := time.Now().Unix()
	val, _ := writeMutexes.LoadOrStore(sessionID, &writeMutexEntry{lastUsed: now})
	entry := val.(*writeMutexEntry)
	atomic.StoreInt64(&entry.lastUsed, now)
	return &entry.mu
}

// persistEvent appends a single AG-UI event to the session's JSONL log.
// Writes are serialised per-session via a mutex to prevent interleaving.
func persistEvent(sessionID string, event map[string]interface{}) {
	dir := fmt.Sprintf("%s/sessions/%s", StateBaseDir, sessionID)
	path := dir + "/agui-events.jsonl"
	_ = ensureDir(dir)

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("AGUI Store: failed to marshal event: %v", err)
		return
	}

	mu := getWriteMutex(sessionID)
	mu.Lock()
	defer mu.Unlock()

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
// Go port of @ag-ui/client compactEvents.  Concatenates streaming deltas
// so reconnect replays are compact and fast.

type pendingText struct {
	start       map[string]interface{}
	deltas      []string
	end         map[string]interface{}
	otherEvents []map[string]interface{}
}

type pendingTool struct {
	start       map[string]interface{}
	deltas      []string
	end         map[string]interface{}
	otherEvents []map[string]interface{}
}

// compactStreamingEvents concatenates TEXT_MESSAGE_CONTENT and TOOL_CALL_ARGS
// deltas for the same messageId/toolCallId.  All other events pass through.
func compactStreamingEvents(events []map[string]interface{}) []map[string]interface{} {
	compacted := make([]map[string]interface{}, 0, len(events)/2)

	textByID := make(map[string]*pendingText)
	var textOrder []string
	toolByID := make(map[string]*pendingTool)
	var toolOrder []string

	getText := func(id string) *pendingText {
		if p, ok := textByID[id]; ok {
			return p
		}
		p := &pendingText{}
		textByID[id] = p
		textOrder = append(textOrder, id)
		return p
	}
	getTool := func(id string) *pendingTool {
		if p, ok := toolByID[id]; ok {
			return p
		}
		p := &pendingTool{}
		toolByID[id] = p
		toolOrder = append(toolOrder, id)
		return p
	}

	flushText := func(id string) {
		p := textByID[id]
		if p == nil {
			return
		}
		if p.start != nil {
			compacted = append(compacted, p.start)
		}
		if len(p.deltas) > 0 {
			combined := ""
			for _, d := range p.deltas {
				combined += d
			}
			compacted = append(compacted, map[string]interface{}{
				"type":      types.EventTypeTextMessageContent,
				"messageId": id,
				"delta":     combined,
			})
		}
		if p.end != nil {
			compacted = append(compacted, p.end)
		}
		compacted = append(compacted, p.otherEvents...)
		delete(textByID, id)
	}

	flushTool := func(id string) {
		p := toolByID[id]
		if p == nil {
			return
		}
		if p.start != nil {
			compacted = append(compacted, p.start)
		}
		if len(p.deltas) > 0 {
			combined := ""
			for _, d := range p.deltas {
				combined += d
			}
			compacted = append(compacted, map[string]interface{}{
				"type":       types.EventTypeToolCallArgs,
				"toolCallId": id,
				"delta":      combined,
			})
		}
		if p.end != nil {
			compacted = append(compacted, p.end)
		}
		compacted = append(compacted, p.otherEvents...)
		delete(toolByID, id)
	}

	for _, evt := range events {
		eventType, _ := evt["type"].(string)
		switch eventType {
		case types.EventTypeTextMessageStart:
			if id, _ := evt["messageId"].(string); id != "" {
				getText(id).start = evt
			} else {
				compacted = append(compacted, evt)
			}
		case types.EventTypeTextMessageContent:
			if id, _ := evt["messageId"].(string); id != "" {
				delta, _ := evt["delta"].(string)
				getText(id).deltas = append(getText(id).deltas, delta)
			} else {
				compacted = append(compacted, evt)
			}
		case types.EventTypeTextMessageEnd:
			if id, _ := evt["messageId"].(string); id != "" {
				getText(id).end = evt
				flushText(id)
			} else {
				compacted = append(compacted, evt)
			}
		case types.EventTypeToolCallStart:
			if id, _ := evt["toolCallId"].(string); id != "" {
				getTool(id).start = evt
			} else {
				compacted = append(compacted, evt)
			}
		case types.EventTypeToolCallArgs:
			if id, _ := evt["toolCallId"].(string); id != "" {
				delta, _ := evt["delta"].(string)
				getTool(id).deltas = append(getTool(id).deltas, delta)
			} else {
				compacted = append(compacted, evt)
			}
		case types.EventTypeToolCallEnd:
			if id, _ := evt["toolCallId"].(string); id != "" {
				getTool(id).end = evt
				flushTool(id)
			} else {
				compacted = append(compacted, evt)
			}
		default:
			// Buffer "other" events into ALL currently open (incomplete)
			// sequences so they replay in the correct position after
			// compaction.  If no sequences are open, emit directly.
			buffered := false
			for _, id := range textOrder {
				if p := textByID[id]; p != nil && p.start != nil && p.end == nil {
					p.otherEvents = append(p.otherEvents, evt)
					buffered = true
				}
			}
			for _, id := range toolOrder {
				if p := toolByID[id]; p != nil && p.start != nil && p.end == nil {
					p.otherEvents = append(p.otherEvents, evt)
					buffered = true
				}
			}
			if !buffered {
				compacted = append(compacted, evt)
			}
		}
	}

	// Flush incomplete sequences (mid-run reconnect)
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
