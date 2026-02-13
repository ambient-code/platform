// Package websocket provides AG-UI protocol endpoints for event streaming.
// See: https://docs.ag-ui.com/quickstart/introduction
package websocket

import (
	"ambient-code-backend/handlers"
	"ambient-code-backend/types"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AG-UI run state tracking and storage
var (
	StateBaseDir string // Base directory for session state persistence (moved from hub.go)

	aguiRuns   = make(map[string]*AGUIRunState) // runID -> state
	aguiRunsMu sync.RWMutex

	// Thread-level subscribers: sessionID -> channels for ALL runs in thread
	threadSubscribers   = make(map[string]map[chan interface{}]bool)
	threadSubscribersMu sync.RWMutex
)

// AGUIRunState tracks the state of an AG-UI run
type AGUIRunState struct {
	ThreadID     string
	RunID        string
	ParentRunID  string
	SessionID    string // maps to our sessionName
	ProjectName  string
	Status       string // "running", "completed", "error"
	StartedAt    time.Time
	subscribers  map[chan *types.BaseEvent]bool
	fullEventSub map[chan interface{}]bool // For full events with all fields
	subscriberMu sync.RWMutex
}

// Subscribe adds a subscriber to this run's events
func (r *AGUIRunState) Subscribe() chan *types.BaseEvent {
	ch := make(chan *types.BaseEvent, 100)
	r.subscriberMu.Lock()
	r.subscribers[ch] = true
	r.subscriberMu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber from this run's events
func (r *AGUIRunState) Unsubscribe(ch chan *types.BaseEvent) {
	r.subscriberMu.Lock()
	delete(r.subscribers, ch)
	close(ch)
	r.subscriberMu.Unlock()
}

// Broadcast sends an event to all subscribers
func (r *AGUIRunState) Broadcast(event *types.BaseEvent) {
	r.subscriberMu.RLock()
	defer r.subscriberMu.RUnlock()
	for ch := range r.subscribers {
		select {
		case ch <- event:
		default:
			// Channel full, skip
		}
	}
}

// BroadcastFull broadcasts full event with all fields (not just BaseEvent)
func (r *AGUIRunState) BroadcastFull(event interface{}) {
	r.subscriberMu.RLock()
	defer r.subscriberMu.RUnlock()

	// Send to full event subscribers
	for ch := range r.fullEventSub {
		select {
		case ch <- event:
		default:
			// Channel full, skip
		}
	}

	// Also send BaseEvent to legacy subscribers
	if baseEvent, ok := extractBaseEvent(event); ok {
		for ch := range r.subscribers {
			select {
			case ch <- baseEvent:
			default:
				// Channel full, skip
			}
		}
	}
}

// RouteAGUIEvent routes an AG-UI event directly from WebSocket to subscribers
// This is the simplified flow - no SessionMessage wrapping, no translation needed
func RouteAGUIEvent(sessionID string, event map[string]interface{}) {
	eventType, ok := event["type"].(string)
	if !ok {
		log.Printf("AGUI: Event missing type field, skipping")
		return
	}

	// Find active run for this session
	var activeRunState *AGUIRunState
	aguiRunsMu.RLock()
	for _, state := range aguiRuns {
		if state.SessionID == sessionID && state.Status == "running" {
			activeRunState = state
			break
		}
	}
	aguiRunsMu.RUnlock()

	// If no active run found, check if event has a runId we should create
	if activeRunState == nil {
		// Ensure timestamp is set before any early returns
		if event["timestamp"] == nil || event["timestamp"] == "" {
			event["timestamp"] = time.Now().UTC().Format(types.AGUITimestampFormat)
		}

		// Don't create lazy runs for terminal events - they should only apply to existing runs
		if isTerminalEventType(eventType) {
			go persistAGUIEventMap(sessionID, "", event)
			return
		}

		eventRunID, ok := event["runId"].(string)
		if ok && eventRunID != "" {
			// Create run lazily from event's runId
			threadID := sessionID
			activeRunState = &AGUIRunState{
				ThreadID:     threadID,
				RunID:        eventRunID,
				SessionID:    sessionID,
				Status:       "running",
				StartedAt:    time.Now(),
				subscribers:  make(map[chan *types.BaseEvent]bool),
				fullEventSub: make(map[chan interface{}]bool),
			}
			aguiRunsMu.Lock()
			aguiRuns[eventRunID] = activeRunState
			aguiRunsMu.Unlock()
		} else {
			go persistAGUIEventMap(sessionID, "", event)
			return
		}
	}

	threadID := activeRunState.ThreadID
	runID := activeRunState.RunID

	// CRITICAL: Use runId from event if present (event is source of truth)
	// Don't use activeRunState.RunID which might be stale
	if eventRunID, ok := event["runId"].(string); ok && eventRunID != "" {
		runID = eventRunID
	}
	if eventThreadID, ok := event["threadId"].(string); ok && eventThreadID != "" {
		threadID = eventThreadID
	}

	// Fill in missing IDs and timestamp
	if event["threadId"] == nil || event["threadId"] == "" {
		event["threadId"] = threadID
	}
	if event["runId"] == nil || event["runId"] == "" {
		event["runId"] = runID
	}
	// Add timestamp if not present - critical for message timestamp tracking
	if event["timestamp"] == nil || event["timestamp"] == "" {
		event["timestamp"] = time.Now().UTC().Format(types.AGUITimestampFormat)
	}

	// Broadcast to run-specific SSE subscribers
	activeRunState.BroadcastFull(event)

	// Also broadcast to thread-level subscribers (clients watching entire session)
	threadSubscribersMu.RLock()
	if subscribers, exists := threadSubscribers[sessionID]; exists {
		for ch := range subscribers {
			select {
			case ch <- event:
			default:
			}
		}
	}
	threadSubscribersMu.RUnlock()

	// Persist the event (use runID from event, not activeRunState)
	go persistAGUIEventMap(sessionID, runID, event)

	// Check for terminal events - mark run as complete
	if isTerminalEventType(eventType) {
		activeRunState.Status = getTerminalStatusFromType(eventType)

		// Schedule cleanup of run state (no need to compact async - we compact on SSE connect)
		go scheduleRunCleanup(runID, 5*time.Minute)
	}
}

// loadCompactedMessages loads pre-compacted messages from completed runs
// NOTE: Removed loadCompactedMessages and compactAndPersistRun functions.
// We now use "compact-on-read" strategy in streamThreadEvents.
// This eliminates race conditions, dual-file complexity, and async compaction issues.

// persistAGUIEventMap persists a map[string]interface{} event to disk
func persistAGUIEventMap(sessionID, runID string, event map[string]interface{}) {
	path := fmt.Sprintf("%s/sessions/%s/agui-events.jsonl", StateBaseDir, sessionID)
	_ = ensureDir(fmt.Sprintf("%s/sessions/%s", StateBaseDir, sessionID))

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("AGUI: failed to marshal event for persistence: %v", err)
		return
	}

	f, err := openFileAppend(path)
	if err != nil {
		log.Printf("AGUI: failed to open event log: %v", err)
		return
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		log.Printf("AGUI: failed to write event: %v", err)
		return
	}

}

// isTerminalEventType checks if an event type indicates run completion
func isTerminalEventType(eventType string) bool {
	switch eventType {
	case types.EventTypeRunFinished, types.EventTypeRunError:
		return true
	}
	return false
}

// getTerminalStatusFromType returns the run status for a terminal event type
func getTerminalStatusFromType(eventType string) string {
	switch eventType {
	case types.EventTypeRunFinished:
		return "completed"
	case types.EventTypeRunError:
		return "error"
	default:
		return "completed"
	}
}

// extractBaseEvent extracts the BaseEvent from any AG-UI event type
func extractBaseEvent(event interface{}) (*types.BaseEvent, bool) {
	switch e := event.(type) {
	case *types.BaseEvent:
		return e, true
	case *types.TextMessageStartEvent:
		return &e.BaseEvent, true
	case *types.TextMessageContentEvent:
		return &e.BaseEvent, true
	case *types.TextMessageEndEvent:
		return &e.BaseEvent, true
	case *types.ToolCallStartEvent:
		return &e.BaseEvent, true
	case *types.ToolCallArgsEvent:
		return &e.BaseEvent, true
	case *types.ToolCallEndEvent:
		return &e.BaseEvent, true
	case *types.StepStartedEvent:
		return &e.BaseEvent, true
	case *types.StepFinishedEvent:
		return &e.BaseEvent, true
	case *types.RunStartedEvent:
		return &e.BaseEvent, true
	case *types.RunFinishedEvent:
		return &e.BaseEvent, true
	case *types.RunErrorEvent:
		return &e.BaseEvent, true
	case *types.StateSnapshotEvent:
		return &e.BaseEvent, true
	case *types.StateDeltaEvent:
		return &e.BaseEvent, true
	case *types.MessagesSnapshotEvent:
		return &e.BaseEvent, true
	case *types.ActivitySnapshotEvent:
		return &e.BaseEvent, true
	case *types.ActivityDeltaEvent:
		return &e.BaseEvent, true
	case *types.RawEvent:
		return &e.BaseEvent, true
	default:
		return nil, false
	}
}

// LEGACY: Old HandleAGUIRun function removed - replaced by HandleAGUIRunProxy
// The new proxy forwards requests to the runner's FastAPI server instead of using WebSocket

// streamThreadEvents streams events from ALL runs in a thread (session)
// This is the correct AG-UI pattern: client connects to thread, not individual runs
func streamThreadEvents(c *gin.Context, projectName, sessionName string) {
	threadID := sessionName
	eventCh := make(chan interface{}, 100)
	ctx := c.Request.Context()

	// Subscribe to all current and future runs for this session
	threadSubscribersMu.Lock()
	if threadSubscribers[sessionName] == nil {
		threadSubscribers[sessionName] = make(map[chan interface{}]bool)
	}
	threadSubscribers[sessionName][eventCh] = true
	threadSubscribersMu.Unlock()

	defer func() {
		threadSubscribersMu.Lock()
		delete(threadSubscribers[sessionName], eventCh)
		if len(threadSubscribers[sessionName]) == 0 {
			delete(threadSubscribers, sessionName)
		}
		threadSubscribersMu.Unlock()
		close(eventCh)
	}()

	// OPTION 1: Compact-on-Read Strategy (COMPLETED RUNS ONLY)
	// Load events from agui-events.jsonl and compact only COMPLETED runs
	// Active/in-progress runs will be streamed raw

	// Declare outside so it's accessible later for replaying active runs
	activeRunIDs := make(map[string]bool)

	events, err := loadEventsForRun(sessionName, "")
	if err == nil && len(events) > 0 {

		// CRITICAL FIX: Determine which runs are TRULY active by checking event log
		// A run is only active if NO terminal event exists in the log
		runHasTerminalEvent := make(map[string]bool)
		for _, event := range events {
			eventRunID, ok := event["runId"].(string)
			if !ok {
				continue
			}
			eventType, ok := event["type"].(string)
			if !ok {
				continue
			}

			if eventRunID != "" && isTerminalEventType(eventType) {
				runHasTerminalEvent[eventRunID] = true
			}
		}

		// Check in-memory state and override with event log truth
		// Also fix stale in-memory state
		aguiRunsMu.Lock()
		for _, state := range aguiRuns {
			if state.SessionID == sessionName {
				runID := state.RunID
				// Only consider active if NO terminal event in log
				if !runHasTerminalEvent[runID] {
					activeRunIDs[runID] = true
				} else {
					// Fix stale memory state
					if state.Status == "running" {
						state.Status = "completed"
					}
				}
			}
		}
		aguiRunsMu.Unlock()

		// Filter to only events from COMPLETED runs (have terminal event)
		// Also collect session-level META events (feedback, etc.) which may not have runId
		completedEvents := make([]map[string]interface{}, 0)
		sessionMetaEvents := make([]map[string]interface{}, 0)
		skippedCount := 0
		for _, event := range events {
			eventType, _ := event["type"].(string)
			eventRunID, ok := event["runId"].(string)

			// META events may not have runId (session-level feedback) - collect them separately
			if eventType == types.EventTypeMeta {
				sessionMetaEvents = append(sessionMetaEvents, event)
				continue
			}

			if !ok {
				continue
			}

			// Skip events without runId
			if eventRunID == "" {
				skippedCount++
				continue
			}

			// Skip events from active runs (no terminal event yet)
			if activeRunIDs[eventRunID] {
				skippedCount++
				continue
			}

			// Include events from completed runs
			completedEvents = append(completedEvents, event)
		}

		if len(completedEvents) > 0 {
			// Compact only completed run events
			messages := CompactEvents(completedEvents)

			// Send single MESSAGES_SNAPSHOT with compacted messages from COMPLETED runs
			if len(messages) > 0 {
				snapshot := &types.MessagesSnapshotEvent{
					BaseEvent: types.NewBaseEvent(types.EventTypeMessagesSnapshot, threadID, "thread-snapshot"),
					Messages:  messages,
				}
				writeSSEEvent(c.Writer, snapshot)
				c.Writer.(http.Flusher).Flush()
			}
		}

		// Replay ALL session META events (feedback, tags, annotations)
		// META events are session-level and not part of MESSAGES_SNAPSHOT
		// They must be replayed regardless of runId to survive reconnects
		if len(sessionMetaEvents) > 0 {
			for _, event := range sessionMetaEvents {
				writeSSEEvent(c.Writer, event)
			}
			c.Writer.(http.Flusher).Flush()
		}
	} else if err != nil {
		log.Printf("AGUI: Failed to load events: %v", err)
	}

	// Replay ALL active runs (not just most recent)
	// CRITICAL: This ensures all non-compacted events are sent to client
	aguiRunsMu.RLock()
	activeRunStates := make([]*AGUIRunState, 0)
	for _, state := range aguiRuns {
		if state.SessionID == sessionName && activeRunIDs[state.RunID] {
			activeRunStates = append(activeRunStates, state)
		}
	}
	aguiRunsMu.RUnlock()

	if len(activeRunStates) > 0 {

		// Load all events once
		allEvents, err := loadEventsForRun(sessionName, "")
		if err == nil {
			for _, activeRunState := range activeRunStates {
				// Send RUN_STARTED for this active run
				runStarted := &types.RunStartedEvent{
					BaseEvent: types.NewBaseEvent(types.EventTypeRunStarted, threadID, activeRunState.RunID),
				}
				if activeRunState.ParentRunID != "" {
					runStarted.ParentRunID = activeRunState.ParentRunID
				}
				writeSSEEvent(c.Writer, runStarted)

				// Send state snapshot
				sendBasicStateSnapshot(c, activeRunState, projectName, sessionName)

				// Collect events for this run
				runEvents := make([]map[string]interface{}, 0)
				for _, event := range allEvents {
					eventRunID, ok := event["runId"].(string)
					if ok && eventRunID == activeRunState.RunID {
						runEvents = append(runEvents, event)
					}
				}

				// Replay raw events
				if len(runEvents) > 0 {
					for _, event := range runEvents {
						writeSSEEvent(c.Writer, event)
					}
				}
			}
			c.Writer.(http.Flusher).Flush()
		}
	}

	// Stream events from all future runs with keepalive
	keepaliveTicker := time.NewTicker(15 * time.Second)
	defer keepaliveTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-keepaliveTicker.C:
			// Send SSE comment to prevent gateway timeout
			_, err := c.Writer.Write([]byte(": keepalive\n\n"))
			if err != nil {
				log.Printf("AGUI: Keepalive write failed, closing stream: %v", err)
				return
			}
			c.Writer.(http.Flusher).Flush()
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			writeSSEEvent(c.Writer, event)
			c.Writer.(http.Flusher).Flush()
		}
	}
}

// HandleAGUIEvents handles GET /api/projects/:projectName/agentic-sessions/:sessionName/agui/events
// This is the AG-UI SSE stream endpoint
// See: https://docs.ag-ui.com/quickstart/middleware
func HandleAGUIEvents(c *gin.Context) {
	projectName := c.Param("projectName")
	sessionName := c.Param("sessionName")
	runID := c.Query("runId")

	// SECURITY: Authenticate user and get user-scoped K8s client
	reqK8s, _ := handlers.GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// SECURITY: Verify user has permission to read this session
	ctx := context.Background()
	ssar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Group:     "vteam.ambient-code",
				Resource:  "agenticsessions",
				Verb:      "get",
				Namespace: projectName,
				Name:      sessionName,
			},
		},
	}
	res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, metav1.CreateOptions{})
	if err != nil || !res.Status.Allowed {
		log.Printf("AGUI Events: User not authorized to read session %s/%s", projectName, sessionName)
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// If no runId specified, stream the entire THREAD (all runs for this session)
	// This is the correct AG-UI pattern: client connects once to thread stream
	if runID == "" {
		streamThreadEvents(c, projectName, sessionName)
		return
	}

	// Legacy: specific run streaming (kept for compatibility)

	var runState *AGUIRunState
	aguiRunsMu.RLock()
	runState = aguiRuns[runID]
	aguiRunsMu.RUnlock()

	if runState == nil {
		// Create an implicit run for this connection
		threadID := sessionName
		runState = &AGUIRunState{
			ThreadID:     threadID,
			RunID:        runID,
			SessionID:    sessionName,
			ProjectName:  projectName,
			Status:       "running",
			StartedAt:    time.Now(),
			subscribers:  make(map[chan *types.BaseEvent]bool),
			fullEventSub: make(map[chan interface{}]bool),
		}
		aguiRunsMu.Lock()
		aguiRuns[runID] = runState
		aguiRunsMu.Unlock()
	}

	// Subscribe to full events (includes Delta, ToolCallID, etc.)
	fullEventCh := make(chan interface{}, 100)
	runState.subscriberMu.Lock()
	runState.fullEventSub[fullEventCh] = true
	runState.subscriberMu.Unlock()
	defer func() {
		runState.subscriberMu.Lock()
		delete(runState.fullEventSub, fullEventCh)
		runState.subscriberMu.Unlock()
		close(fullEventCh)
	}()

	// Send initial sync events (with panic recovery)
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("AGUI: panic in sendInitialSyncEvents: %v", r)
			}
		}()
		sendInitialSyncEvents(c, runState, projectName, sessionName)
	}()

	// Create context for client disconnection
	streamCtx := c.Request.Context()

	// Stream events
	for {
		select {
		case <-streamCtx.Done():
			return
		case event, ok := <-fullEventCh:
			if !ok {
				return
			}
			writeSSEEvent(c.Writer, event)
			c.Writer.(http.Flusher).Flush()
		}
	}
}

// sendInitialSyncEvents sends snapshot events on connection/reconnection
// This implements the reconnect/restore strategy per AG-UI serialization guidance
func sendInitialSyncEvents(c *gin.Context, runState *AGUIRunState, projectName, sessionName string) {
	threadID := runState.ThreadID
	runID := runState.RunID

	// 1. Send RUN_STARTED
	runStarted := &types.RunStartedEvent{
		BaseEvent: types.NewBaseEvent(types.EventTypeRunStarted, threadID, runID),
	}
	if runState.ParentRunID != "" {
		runStarted.ParentRunID = runState.ParentRunID
	}
	writeSSEEvent(c.Writer, runStarted)

	// 2. Send basic state snapshot (always succeeds)
	sendBasicStateSnapshot(c, runState, projectName, sessionName)

	// 3. Compact stored events and send MESSAGES_SNAPSHOT
	// Per AG-UI spec: compact at read-time, not write-time
	events, err := loadEventsForRun(sessionName, runID)
	if err != nil {
		log.Printf("AGUI: Failed to load events for %s: %v", sessionName, err)
	}

	if len(events) > 0 {
		messages := CompactEvents(events)

		if len(messages) > 0 {
			snapshot := &types.MessagesSnapshotEvent{
				BaseEvent: types.NewBaseEvent(types.EventTypeMessagesSnapshot, threadID, runID),
				Messages:  messages,
			}
			writeSSEEvent(c.Writer, snapshot)
		}
	}
}

// sendBasicStateSnapshot sends a basic state snapshot with session metadata
func sendBasicStateSnapshot(c *gin.Context, runState *AGUIRunState, projectName, sessionName string) {
	threadID := runState.ThreadID
	runID := runState.RunID

	stateSnapshot := &types.StateSnapshotEvent{
		BaseEvent: types.NewBaseEvent(types.EventTypeStateSnapshot, threadID, runID),
		State: map[string]interface{}{
			"sessionName": sessionName,
			"projectName": projectName,
			"status":      runState.Status,
		},
	}

	// Enrich with session data if available
	sessionData, err := getSessionState(projectName, sessionName)
	if err == nil && sessionData != nil {
		for k, v := range sessionData {
			stateSnapshot.State[k] = v
		}
	}
	writeSSEEvent(c.Writer, stateSnapshot)
}

// writeSSEEvent writes an event in SSE format
func writeSSEEvent(w http.ResponseWriter, event interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("AGUI: failed to marshal event: %v", err)
		return
	}

	fmt.Fprintf(w, "data: %s\n\n", data)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// scheduleRunCleanup removes a run from the active runs map after a delay
func scheduleRunCleanup(runID string, delay time.Duration) {
	time.Sleep(delay)
	aguiRunsMu.Lock()
	if run, ok := aguiRuns[runID]; ok {
		// Only delete if run is no longer active
		if run.Status != "running" {
			delete(aguiRuns, runID)
		}
	}
	aguiRunsMu.Unlock()
}

// cleanupOldRuns periodically cleans up old inactive runs
func init() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			cleanupInactiveRuns()
		}
	}()
}

func cleanupInactiveRuns() {
	cutoff := time.Now().Add(-30 * time.Minute)
	aguiRunsMu.Lock()
	defer aguiRunsMu.Unlock()
	for runID, run := range aguiRuns {
		if run.Status != "running" && run.StartedAt.Before(cutoff) {
			delete(aguiRuns, runID)
		}
	}
}

// Legacy translation functions removed - AG-UI events now route directly via RouteAGUIEvent

// Helper functions for state and message retrieval

func getSessionState(projectName, sessionName string) (map[string]interface{}, error) {
	// Get session from K8s and extract relevant state
	if handlers.DynamicClient == nil {
		// Return basic state if K8s client not available
		return map[string]interface{}{
			"phase":       "Unknown",
			"interactive": true,
		}, nil
	}

	gvr := handlers.GetAgenticSessionV1Alpha1Resource()
	item, err := handlers.DynamicClient.Resource(gvr).Namespace(projectName).Get(
		context.Background(), sessionName, metav1.GetOptions{},
	)
	if err != nil {
		log.Printf("AGUI: failed to get session state: %v", err)
		return map[string]interface{}{
			"phase":       "Unknown",
			"interactive": true,
		}, nil
	}

	state := make(map[string]interface{})

	// Extract spec fields
	if spec, ok := item.Object["spec"].(map[string]interface{}); ok {
		if interactive, ok := spec["interactive"].(bool); ok {
			state["interactive"] = interactive
		}
		if displayName, ok := spec["displayName"].(string); ok {
			state["displayName"] = displayName
		}
		if repos, ok := spec["repos"].([]interface{}); ok {
			state["repos"] = repos
		}
		if workflow, ok := spec["activeWorkflow"].(map[string]interface{}); ok {
			state["activeWorkflow"] = workflow
		}
	}

	// Extract status fields
	if status, ok := item.Object["status"].(map[string]interface{}); ok {
		if phase, ok := status["phase"].(string); ok {
			state["phase"] = phase
		}
		if sdkSessionID, ok := status["sdkSessionId"].(string); ok {
			state["sdkSessionId"] = sdkSessionID
		}
		if restartCount, ok := status["sdkRestartCount"].(int64); ok {
			state["sdkRestartCount"] = restartCount
		} else if restartCount, ok := status["sdkRestartCount"].(float64); ok {
			state["sdkRestartCount"] = int(restartCount)
		}
		if reconciledRepos, ok := status["reconciledRepos"].([]interface{}); ok {
			state["reconciledRepos"] = reconciledRepos
		}
	}

	return state, nil
}

// AG-UI event persistence
// Implements append-only event log per AG-UI serialization guidance:
// https://docs.ag-ui.com/concepts/serialization#serialization

// persistRunMetadata saves run metadata for indexing
func persistRunMetadata(sessionID string, meta types.AGUIRunMetadata) {
	path := fmt.Sprintf("%s/sessions/%s/agui-runs.jsonl", StateBaseDir, sessionID)

	_ = ensureDir(fmt.Sprintf("%s/sessions/%s", StateBaseDir, sessionID))

	data, err := json.Marshal(meta)
	if err != nil {
		log.Printf("AGUI: failed to marshal run metadata: %v", err)
		return
	}

	f, err := openFileAppend(path)
	if err != nil {
		log.Printf("AGUI: failed to open runs index: %v", err)
		return
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		log.Printf("AGUI: failed to write run metadata: %v", err)
	}
}

// loadRunsFromDisk loads persisted run metadata from disk
func loadRunsFromDisk(sessionID string) []types.AGUIRunMetadata {
	path := fmt.Sprintf("%s/sessions/%s/agui-runs.jsonl", StateBaseDir, sessionID)
	runs := make([]types.AGUIRunMetadata, 0)

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("AGUI: failed to read runs index: %v", err)
		}
		return runs
	}

	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var meta types.AGUIRunMetadata
		if err := json.Unmarshal(line, &meta); err == nil {
			runs = append(runs, meta)
		}
	}

	return runs
}

// loadEventsForRun loads all events for a session (thread) from disk
// Per AG-UI spec: all runs in a thread share the same event log
// Includes automatic migration from legacy message format
func loadEventsForRun(sessionID, runID string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("%s/sessions/%s/agui-events.jsonl", StateBaseDir, sessionID)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Check if legacy messages.json exists and migrate
			if err := MigrateLegacySessionToAGUI(sessionID); err != nil {
				log.Printf("LegacyMigration: Failed to migrate session %s: %v", sessionID, err)
			} else {
				// Try reading again after migration
				data, err = os.ReadFile(path)
				if err != nil {
					return []map[string]interface{}{}, nil
				}
			}
			if len(data) == 0 {
				return []map[string]interface{}{}, nil
			}
		} else {
			return nil, err
		}
	}

	events := make([]map[string]interface{}, 0)
	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var event map[string]interface{}
		if err := json.Unmarshal(line, &event); err == nil {
			// Filter by runID if specified
			if runID != "" {
				eventRunID, ok := event["runId"].(string)
				if !ok || eventRunID != runID {
					continue
				}
			}
			events = append(events, event)
		}
	}

	return events, nil
}

// splitLines splits bytes by newline
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			line := data[start:i]
			if len(line) > 0 {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

// HandleAGUIHistory handles GET /api/projects/:projectName/agentic-sessions/:sessionName/agui/history
// Returns compacted message history for a session
func HandleAGUIHistory(c *gin.Context) {
	projectName := c.Param("projectName")
	sessionName := c.Param("sessionName")

	// SECURITY: Authenticate user and get user-scoped K8s client
	reqK8s, _ := handlers.GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// SECURITY: Verify user has permission to read this session
	ctx := context.Background()
	ssar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Group:     "vteam.ambient-code",
				Resource:  "agenticsessions",
				Verb:      "get",
				Namespace: projectName,
				Name:      sessionName,
			},
		},
	}
	res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, metav1.CreateOptions{})
	if err != nil || !res.Status.Allowed {
		log.Printf("AGUI History: User not authorized to read session %s/%s", projectName, sessionName)
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}
	runID := c.Query("runId")

	// Compact events to messages
	var messages []types.Message
	if runID != "" {
		events, err := loadEventsForRun(sessionName, runID)
		if err == nil {
			messages = CompactEvents(events)
		}
	}

	// Get runs for this session
	runs := getRunsForSession(sessionName)

	c.JSON(http.StatusOK, gin.H{
		"threadId": sessionName,
		"runId":    runID,
		"messages": messages,
		"runs":     runs,
	})
}

// HandleAGUIRuns handles GET /api/projects/:projectName/agentic-sessions/:sessionName/agui/runs
// Returns list of runs for a session (thread)
func HandleAGUIRuns(c *gin.Context) {
	projectName := c.Param("projectName")
	sessionName := c.Param("sessionName")

	// SECURITY: Authenticate user and get user-scoped K8s client
	reqK8s, _ := handlers.GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// SECURITY: Verify user has permission to read this session
	ctx := context.Background()
	ssar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Group:     "vteam.ambient-code",
				Resource:  "agenticsessions",
				Verb:      "get",
				Namespace: projectName,
				Name:      sessionName,
			},
		},
	}
	res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, metav1.CreateOptions{})
	if err != nil || !res.Status.Allowed {
		log.Printf("AGUI Runs: User not authorized to read session %s/%s", projectName, sessionName)
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	runs := getRunsForSession(sessionName)

	c.JSON(http.StatusOK, gin.H{
		"threadId": sessionName,
		"runs":     runs,
	})
}

func getRunsForSession(sessionID string) []types.AGUIRunMetadata {
	// First load from disk (historical runs)
	runs := loadRunsFromDisk(sessionID)

	// Create a set of run IDs from disk
	diskRunIDs := make(map[string]bool)
	for _, r := range runs {
		diskRunIDs[r.RunID] = true
	}

	// Add any active runs not yet persisted
	aguiRunsMu.RLock()
	for _, run := range aguiRuns {
		if run.SessionID == sessionID && !diskRunIDs[run.RunID] {
			meta := types.AGUIRunMetadata{
				ThreadID:    run.ThreadID,
				RunID:       run.RunID,
				ParentRunID: run.ParentRunID,
				SessionName: run.SessionID,
				ProjectName: run.ProjectName,
				StartedAt:   run.StartedAt.Format(time.RFC3339),
				Status:      run.Status,
			}
			runs = append(runs, meta)
		}
	}
	aguiRunsMu.RUnlock()

	return runs
}

// Helper file operations

func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func openFileAppend(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
}

// Integration with existing hub - modify hub.go run() to also broadcast to AG-UI subscribers
// This is done by calling BroadcastToSessionSubscribers in the hub's broadcast case

func init() {
	// Hook into the hub to also broadcast to AG-UI subscribers
	// We'll need to modify hub.go to call BroadcastToSessionSubscribers
}
