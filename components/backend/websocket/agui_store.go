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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

const (
	// Scanner buffer sizes for reading JSONL files
	scannerInitialBufferSize = 64 * 1024   // 64KB initial buffer
	scannerMaxLineSize       = 1024 * 1024 // 1MB max line size
)

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

	// Invalidate compacted cache when a new run starts or an error arrives after caching
	eventType, _ := event["type"].(string)
	if eventType == types.EventTypeRunStarted || eventType == types.EventTypeRunError {
		compactedPath := dir + "/agui-events-compacted.jsonl"
		os.Remove(compactedPath) // best-effort; ignore errors
	}
}

// ─── Read path ───────────────────────────────────────────────────────

// loadEvents reads all AG-UI events for a session from the JSONL log
// using a streaming scanner to avoid loading the entire file into memory.
// Automatically triggers legacy migration if the log doesn't exist but
// a pre-AG-UI messages.jsonl file does.
func loadEvents(sessionID string) []map[string]interface{} {
	path := fmt.Sprintf("%s/sessions/%s/agui-events.jsonl", StateBaseDir, sessionID)

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Attempt legacy migration (messages.jsonl → agui-events.jsonl)
			if mErr := MigrateLegacySessionToAGUI(sessionID); mErr != nil {
				log.Printf("AGUI Store: legacy migration failed for %s: %v", sessionID, mErr)
			}
			// Retry after migration
			f, err = os.Open(path)
			if err != nil {
				return nil
			}
		} else {
			log.Printf("AGUI Store: failed to read event log for %s: %v", sessionID, err)
			return nil
		}
	}
	defer f.Close()

	events := make([]map[string]interface{}, 0, 64)
	scanner := bufio.NewScanner(f)
	// Allow lines up to 1MB (default 64KB may truncate large tool outputs)
	scanner.Buffer(make([]byte, 0, scannerInitialBufferSize), scannerMaxLineSize)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var evt map[string]interface{}
		if err := json.Unmarshal(line, &evt); err == nil {
			events = append(events, evt)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("AGUI Store: error scanning event log for %s: %v", sessionID, err)
	}
	return events
}

// DeriveAgentStatus reads a session's event log and returns the agent
// status derived from the last significant events.
//
// Returns "" if the status cannot be determined (no events, file missing, etc.).
func DeriveAgentStatus(sessionID string) string {
	path := fmt.Sprintf("%s/sessions/%s/agui-events.jsonl", StateBaseDir, sessionID)

	// Read only the tail of the file to avoid loading entire event log into memory.
	// 64KB is sufficient for recent lifecycle events (scanning backwards).
	const maxTailBytes = 64 * 1024

	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return ""
	}

	fileSize := stat.Size()
	var data []byte

	if fileSize <= maxTailBytes {
		// File is small, read it all
		data, err = os.ReadFile(path)
		if err != nil {
			return ""
		}
	} else {
		// File is large, seek to tail and read last N bytes
		offset := fileSize - maxTailBytes
		_, err = file.Seek(offset, 0)
		if err != nil {
			return ""
		}

		data = make([]byte, maxTailBytes)
		n, err := file.Read(data)
		if err != nil {
			return ""
		}
		data = data[:n]

		// Skip partial first line (we seeked into the middle of a line)
		if idx := bytes.IndexByte(data, '\n'); idx >= 0 {
			data = data[idx+1:]
		}
	}

	lines := splitLines(data)

	// Scan backwards.  We only care about lifecycle and AskUserQuestion events.
	//   RUN_STARTED                       → "working"
	//   RUN_FINISHED / RUN_ERROR          → "idle", unless same run had AskUserQuestion
	//   TOOL_CALL_START (AskUserQuestion) → "waiting_input"
	var runEndRunID string // set when we hit RUN_FINISHED/RUN_ERROR and need to look deeper
	for i := len(lines) - 1; i >= 0; i-- {
		if len(lines[i]) == 0 {
			continue
		}
		var evt map[string]interface{}
		if err := json.Unmarshal(lines[i], &evt); err != nil {
			continue
		}
		evtType, _ := evt["type"].(string)

		switch evtType {
		case types.EventTypeRunStarted:
			if runEndRunID != "" {
				// We were scanning for an AskUserQuestion but hit RUN_STARTED first → idle
				return types.AgentStatusIdle
			}
			return types.AgentStatusWorking

		case types.EventTypeRunFinished, types.EventTypeRunError:
			if runEndRunID == "" {
				// First run-end seen; scan deeper within this run for AskUserQuestion
				runEndRunID, _ = evt["runId"].(string)
			}

		case types.EventTypeToolCallStart:
			if runEndRunID != "" {
				// Only relevant if we're scanning within the ended run
				if evtRunID, _ := evt["runId"].(string); evtRunID != "" && evtRunID != runEndRunID {
					return types.AgentStatusIdle
				}
			}
			if toolName, _ := evt["toolCallName"].(string); isAskUserQuestionToolCall(toolName) {
				return types.AgentStatusWaitingInput
			}
		}
	}

	if runEndRunID != "" {
		return types.AgentStatusIdle
	}
	return ""
}

// ─── Snapshot compaction (AG-UI serialization spec) ──────────────────
//
// compactToSnapshots collapses a finished event stream into MESSAGES_SNAPSHOT
// events per the AG-UI serialization spec. This is far more aggressive than
// delta compaction: entire TEXT_MESSAGE and TOOL_CALL sequences become fully
// assembled Message objects inside a single MESSAGES_SNAPSHOT event.
//
// See: https://docs.ag-ui.com/concepts/serialization

// compactToSnapshots converts a finished event stream into snapshot events.
// It assembles TEXT_MESSAGE and TOOL_CALL sequences into Message objects,
// emits a MESSAGES_SNAPSHOT, and passes through lifecycle + RAW events.
func compactToSnapshots(events []map[string]interface{}) []map[string]interface{} {
	var messages []map[string]interface{}
	var result []map[string]interface{}

	// Track in-progress text messages and tool calls
	textContent := make(map[string]*strings.Builder) // messageId → accumulated content
	textRole := make(map[string]string)              // messageId → role
	textMeta := make(map[string]interface{})         // messageId → metadata
	toolArgs := make(map[string]*strings.Builder)    // toolCallId → accumulated args
	toolName := make(map[string]string)              // toolCallId → tool name
	toolResult := make(map[string]string)            // toolCallId → result content
	var messageOrder []string                        // ordered messageIds
	messageToolCalls := make(map[string][]string)    // messageId → []toolCallId

	for _, evt := range events {
		eventType, _ := evt["type"].(string)
		switch eventType {
		case types.EventTypeTextMessageStart:
			msgID, _ := evt["messageId"].(string)
			if msgID == "" {
				continue
			}
			role, _ := evt["role"].(string)
			textRole[msgID] = role
			textContent[msgID] = &strings.Builder{}
			if meta := evt["metadata"]; meta != nil {
				textMeta[msgID] = meta
			}
			messageOrder = append(messageOrder, msgID)

		case types.EventTypeTextMessageContent:
			msgID, _ := evt["messageId"].(string)
			if msgID == "" {
				continue
			}
			delta, _ := evt["delta"].(string)
			if b, ok := textContent[msgID]; ok {
				b.WriteString(delta)
			}

		case types.EventTypeTextMessageEnd:
			// Content is finalized in the message assembly below

		case types.EventTypeToolCallStart:
			tcID, _ := evt["toolCallId"].(string)
			if tcID == "" {
				continue
			}
			name, _ := evt["toolCallName"].(string)
			toolName[tcID] = name
			toolArgs[tcID] = &strings.Builder{}
			// Associate tool call with its parent message
			parentMsgID, _ := evt["messageId"].(string)
			if parentMsgID != "" {
				messageToolCalls[parentMsgID] = append(messageToolCalls[parentMsgID], tcID)
			}

		case types.EventTypeToolCallArgs:
			tcID, _ := evt["toolCallId"].(string)
			if tcID == "" {
				continue
			}
			delta, _ := evt["delta"].(string)
			if b, ok := toolArgs[tcID]; ok {
				b.WriteString(delta)
			}

		case types.EventTypeToolCallEnd:
			tcID, _ := evt["toolCallId"].(string)
			if tcID == "" {
				continue
			}
			if res, ok := evt["result"].(string); ok && res != "" {
				toolResult[tcID] = res
			}

		case types.EventTypeRunStarted, types.EventTypeRunFinished, types.EventTypeRunError,
			types.EventTypeStepStarted, types.EventTypeStepFinished:
			// Pass through lifecycle events as-is
			result = append(result, evt)

		case types.EventTypeRaw:
			// Pass through RAW events (e.g. hidden message metadata, feedback)
			result = append(result, evt)

		case types.EventTypeMessagesSnapshot:
			// If there's already a snapshot in the stream, pass it through
			result = append(result, evt)

		case types.EventTypeStateSnapshot, types.EventTypeStateDelta:
			// Pass through state events
			result = append(result, evt)
		}
	}

	// Assemble messages from tracked state
	for _, msgID := range messageOrder {
		role := textRole[msgID]
		content := ""
		if b, ok := textContent[msgID]; ok {
			content = b.String()
		}

		msg := map[string]interface{}{
			"id":   msgID,
			"role": role,
		}
		if content != "" {
			msg["content"] = content
		}
		if meta, ok := textMeta[msgID]; ok {
			msg["metadata"] = meta
		}

		// Attach tool calls if this is an assistant message
		if tcIDs, ok := messageToolCalls[msgID]; ok && len(tcIDs) > 0 {
			var toolCalls []map[string]interface{}
			for _, tcID := range tcIDs {
				args := ""
				if b, ok := toolArgs[tcID]; ok {
					args = b.String()
				}
				tc := map[string]interface{}{
					"id":   tcID,
					"name": toolName[tcID],
					"args": args,
				}
				toolCalls = append(toolCalls, tc)
			}
			msg["toolCalls"] = toolCalls
		}

		messages = append(messages, msg)

		// Emit tool result messages after the assistant message
		if tcIDs, ok := messageToolCalls[msgID]; ok {
			for _, tcID := range tcIDs {
				if res, ok := toolResult[tcID]; ok {
					toolMsg := map[string]interface{}{
						"id":         generateEventID(),
						"role":       types.RoleTool,
						"content":    res,
						"toolCallId": tcID,
						"name":       toolName[tcID],
					}
					messages = append(messages, toolMsg)
				}
			}
		}
	}

	// Emit MESSAGES_SNAPSHOT if we have messages
	if len(messages) > 0 {
		snapshot := map[string]interface{}{
			"type":     types.EventTypeMessagesSnapshot,
			"messages": messages,
		}
		result = append(result, snapshot)
	}

	return result
}

// loadEventsForReplay loads events for SSE replay, using cached snapshots
// for finished sessions. For finished sessions, it writes a compacted snapshot
// file on first access and serves from it on subsequent reads.
func loadEventsForReplay(sessionID string) []map[string]interface{} {
	compactedPath := fmt.Sprintf("%s/sessions/%s/agui-events-compacted.jsonl", StateBaseDir, sessionID)

	// Try to serve from cached compacted file first
	if events, err := readJSONLFile(compactedPath); err == nil && len(events) > 0 {
		log.Printf("AGUI Events: serving %d cached snapshot events for %s", len(events), sessionID)
		return events
	}

	// Load raw events
	events := loadEvents(sessionID)
	if len(events) == 0 {
		return events
	}

	// Check if the last run is finished
	last := events[len(events)-1]
	if last == nil {
		return events
	}
	lastType, _ := last["type"].(string)
	if lastType != types.EventTypeRunFinished && lastType != types.EventTypeRunError {
		// Active run — return raw events to preserve streaming structure
		log.Printf("AGUI Events: replaying %d raw events for %s (running)", len(events), sessionID)
		return events
	}

	// Finished run — compact to snapshots
	compacted := compactToSnapshots(events)
	log.Printf("AGUI Events: %d raw → %d snapshot events for %s (finished)", len(events), len(compacted), sessionID)

	// Persist compacted file for future reads (best-effort, non-blocking)
	go writeCompactedFile(compactedPath, compacted)

	return compacted
}

// writeCompactedFile writes snapshot events to the compacted JSONL file.
func writeCompactedFile(path string, events []map[string]interface{}) {
	f, err := os.CreateTemp(filepath.Dir(path), "compacted-*.tmp")
	if err != nil {
		log.Printf("AGUI Store: failed to create compacted temp file: %v", err)
		return
	}
	tmpPath := f.Name()

	w := bufio.NewWriter(f)
	for _, evt := range events {
		data, err := json.Marshal(evt)
		if err != nil {
			f.Close()
			os.Remove(tmpPath)
			return
		}
		w.Write(data)
		w.WriteByte('\n')
	}
	if err := w.Flush(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return
	}
	f.Close()

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		log.Printf("AGUI Store: failed to rename compacted file: %v", err)
		os.Remove(tmpPath)
	}
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
