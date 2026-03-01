// Package websocket provides AG-UI protocol endpoints including HTTP proxy to runner.
//
// agui_proxy.go — HTTP handlers that proxy AG-UI requests to the runner pod
// and persist every event to the append-only event log.
//
// Two jobs:
//  1. Passthrough: POST to runner, pipe SSE back to client.
//  2. Persist: append every event to agui-events.jsonl as it flows through.
//
// Reconnection is handled by InMemoryAgentRunner on the frontend.
// The backend only persists events for cross-restart recovery.
package websocket

import (
	"ambient-code-backend/handlers"
	"ambient-code-backend/types"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
)

// runnerProxySecretCache caches proxy secrets per session to avoid repeated K8s API calls.
// Key: "namespace/sessionName", Value: proxy secret string.
var runnerProxySecretCache sync.Map

// getRunnerProxySecret reads the proxy-secret key from the runner token secret.
// Returns empty string if the secret does not exist or the key is absent (graceful degradation
// for sessions created before proxy authentication was introduced).
func getRunnerProxySecret(projectName, sessionName string) string {
	cacheKey := projectName + "/" + sessionName
	if cached, ok := runnerProxySecretCache.Load(cacheKey); ok {
		return cached.(string)
	}

	if handlers.K8sClientMw == nil {
		return ""
	}

	secretName := fmt.Sprintf("ambient-runner-token-%s", sessionName)
	secret, err := handlers.K8sClientMw.CoreV1().Secrets(projectName).Get(
		context.Background(), secretName, metav1.GetOptions{},
	)
	if err != nil {
		log.Printf("AGUI Proxy: could not read runner proxy secret for %s/%s: %v", projectName, sessionName, err)
		return ""
	}

	proxySecret := strings.TrimSpace(string(secret.Data["proxy-secret"]))
	if proxySecret != "" {
		runnerProxySecretCache.Store(cacheKey, proxySecret)
	}
	return proxySecret
}

const (
	// activityDebounceInterval is the minimum interval between CR status updates for lastActivityTime.
	// Inactivity timeout is measured in hours, so minute-level granularity is sufficient.
	activityDebounceInterval = 60 * time.Second

	// activityUpdateTimeout bounds how long a single activity status update can take.
	activityUpdateTimeout = 10 * time.Second
)

// activityUpdateSem limits concurrent goroutines spawned by updateLastActivityTime.
// With 60s debounce, at most one goroutine per session is active at a time under normal
// conditions; this cap protects against pathological bursts (e.g., many sessions starting
// simultaneously with immediate=true).
var activityUpdateSem = make(chan struct{}, 50)

// lastActivityUpdateTimes tracks the last time we updated lastActivityTime on the CR
// for each session to avoid excessive API calls. Key: "namespace/sessionName"
var lastActivityUpdateTimes sync.Map

// sessionProjectMap maps sessionName → projectName so that persistStreamedEvent
// (which only receives sessionID) can look up the project for activity tracking.
// Populated by HandleAGUIRunProxy on each run request.
var sessionProjectMap sync.Map

// HandleAGUIEvents serves the AG-UI event stream over SSE.  Clients
// (typically EventSource) connect here to receive all events for a
// session — both persisted history and live events from active runs.
//
// This is the "read" side of the AG-UI middleware pattern:
//
//	POST /agui/run  → starts a run, returns JSON metadata immediately
//	GET  /agui/events → SSE stream of all thread events (past + future)
func HandleAGUIEvents(c *gin.Context) {
	projectName := c.Param("projectName")
	sessionName := c.Param("sessionName")

	// SECURITY: Authenticate + RBAC (read access)
	reqK8s, _ := handlers.GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}
	if !checkAccess(reqK8s, projectName, sessionName, "get") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	log.Printf("AGUI Events: client connected for %s/%s", projectName, sessionName)

	// ── SSE response headers ─────────────────────────────────────
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	// Subscribe to live broadcast pipe BEFORE loading persisted events.
	// This ordering prevents a race where events published between
	// loadEvents() and subscribeLive() would be missed by the client.
	liveCh, cleanup := subscribeLive(sessionName)
	defer cleanup()

	events := loadEvents(sessionName)

	if len(events) > 0 {
		// Check if the last run is finished.
		runFinished := false
		if last := events[len(events)-1]; last != nil {
			if t, _ := last["type"].(string); t == types.EventTypeRunFinished {
				runFinished = true
			}
		}

		if runFinished {
			// Finished runs get compacted replay (fast, small).
			compacted := compactStreamingEvents(events)
			log.Printf("AGUI Events: %d raw → %d compacted events for %s (finished)", len(events), len(compacted), sessionName)
			for _, evt := range compacted {
				writeSSEEvent(c.Writer, evt)
			}
		} else {
			// Active run — send raw events to preserve streaming structure.
			log.Printf("AGUI Events: replaying %d raw events for %s (running)", len(events), sessionName)
			for _, evt := range events {
				writeSSEEvent(c.Writer, evt)
			}
		}
		c.Writer.Flush()
	}

	// Drain live events buffered during replay — they are already
	// covered by the persisted events we just sent.
	drainLiveChannel(liveCh)

	// Tail live events until client disconnects.
	// Send SSE comments as keepalive every 15s to prevent proxies
	// (Next.js, nginx, ALB) from dropping the idle connection.
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	clientGone := c.Request.Context().Done()
	for {
		select {
		case <-clientGone:
			log.Printf("AGUI Events: client disconnected for %s", sessionName)
			return
		case line, ok := <-liveCh:
			if !ok {
				return
			}
			fmt.Fprint(c.Writer, line)
			c.Writer.Flush()
		case <-heartbeat.C:
			// SSE comment — ignored by EventSource but keeps connection alive
			fmt.Fprint(c.Writer, ": heartbeat\n\n")
			c.Writer.Flush()
		}
	}
}

// HandleAGUIRunProxy accepts an AG-UI run request, forwards it to the
// runner pod in a background goroutine, and returns JSON metadata
// immediately.  Events are persisted and broadcast to GET /agui/events
// subscribers via the live broadcast pipe.
func HandleAGUIRunProxy(c *gin.Context) {
	projectName := c.Param("projectName")
	sessionName := c.Param("sessionName")

	// SECURITY: Authenticate + RBAC
	reqK8s, _ := handlers.GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}
	if !checkAccess(reqK8s, projectName, sessionName, "update") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	// Parse input (messages are json.RawMessage pass-through)
	var input types.RunAgentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("AGUI Proxy: Failed to parse input: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid input: %v", err)})
		return
	}

	// Generate or use provided IDs
	threadID := input.ThreadID
	if threadID == "" {
		threadID = sessionName
	}
	runID := input.RunID
	if runID == "" {
		runID = uuid.New().String()
	}
	input.ThreadID = threadID
	input.RunID = runID

	// Count actual messages
	var rawMessages []json.RawMessage
	if len(input.Messages) > 0 {
		_ = json.Unmarshal(input.Messages, &rawMessages)
	}

	log.Printf("AGUI Proxy: run=%s session=%s/%s msgs=%d", truncID(runID), projectName, sessionName, len(rawMessages))

	// Store project→session mapping for activity tracking in persistStreamedEvent
	sessionProjectMap.Store(sessionName, projectName)

	// Parse messages for display name generation and hidden metadata
	var minimalMsgs []types.Message
	if len(rawMessages) > 0 {
		for _, raw := range rawMessages {
			var msg types.Message
			if err := json.Unmarshal(raw, &msg); err == nil {
				minimalMsgs = append(minimalMsgs, msg)
			}
		}
		go triggerDisplayNameGenerationIfNeeded(projectName, sessionName, minimalMsgs)
	}

	// Emit message_metadata RAW events for hidden messages (e.g. auto-sent
	// workflow prompts).  These must be persisted and broadcast BEFORE the
	// runner starts emitting events so GET /agui/events subscribers hide
	// the messages before they arrive via TEXT_MESSAGE_* events.
	for _, msg := range minimalMsgs {
		if isMessageHidden(msg.Metadata) {
			emitHiddenMessageMetadata(sessionName, runID, threadID, msg.ID)
		}
	}

	// ── Forward to runner in background, return JSON immediately ──
	bodyBytes, err := json.Marshal(input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize input"})
		return
	}

	runnerURL := getRunnerEndpoint(projectName, sessionName)
	proxySecret := getRunnerProxySecret(projectName, sessionName)

	// Start background goroutine to proxy runner SSE → persist + broadcast
	go proxyRunnerStream(runnerURL, bodyBytes, sessionName, runID, threadID, proxySecret)

	// Return metadata immediately — events arrive via GET /agui/events
	c.JSON(http.StatusOK, gin.H{
		"runId":    runID,
		"threadId": threadID,
	})
}

// proxyRunnerStream connects to the runner's SSE endpoint, reads events,
// persists them, and publishes them to the live broadcast pipe.  Runs in
// a background goroutine so the POST /agui/run handler can return immediately.
func proxyRunnerStream(runnerURL string, bodyBytes []byte, sessionName, runID, threadID, proxySecret string) {
	log.Printf("AGUI Proxy: connecting to runner at %s", runnerURL)
	resp, err := connectToRunner(runnerURL, bodyBytes, proxySecret)
	if err != nil {
		log.Printf("AGUI Proxy: runner unavailable for %s: %v", sessionName, err)
		// Publish error events so GET /agui/events subscribers see the failure
		publishAndPersistErrorEvents(sessionName, runID, threadID, "Runner is not available")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("AGUI Proxy: runner returned %d: %s", resp.StatusCode, string(body))
		publishAndPersistErrorEvents(sessionName, runID, threadID, fmt.Sprintf("Runner error: HTTP %d", resp.StatusCode))
		return
	}

	// Pipe SSE from runner: persist each event and broadcast to subscribers
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("AGUI Proxy: stream read error: %v", err)
			}
			break
		}

		trimmed := strings.TrimSpace(line)

		// Persist every data event to JSONL
		if strings.HasPrefix(trimmed, "data: ") {
			jsonData := strings.TrimPrefix(trimmed, "data: ")
			persistStreamedEvent(sessionName, runID, threadID, jsonData)
		}

		// Publish raw SSE line to all GET /agui/events subscribers
		publishLine(sessionName, line)
	}

	log.Printf("AGUI Proxy: run %s stream ended", truncID(runID))
}

// publishAndPersistErrorEvents generates RUN_STARTED + RUN_ERROR events,
// persists them, and publishes to the live broadcast so subscribers get
// notified of runner failures.
func publishAndPersistErrorEvents(sessionName, runID, threadID, message string) {
	// RUN_STARTED
	startEvt := map[string]interface{}{
		"type":     "RUN_STARTED",
		"threadId": threadID,
		"runId":    runID,
	}
	persistEvent(sessionName, startEvt)
	startData, _ := json.Marshal(startEvt)
	publishLine(sessionName, fmt.Sprintf("data: %s\n\n", startData))

	// RUN_ERROR
	errEvt := map[string]interface{}{
		"type":     "RUN_ERROR",
		"message":  message,
		"threadId": threadID,
		"runId":    runID,
	}
	persistEvent(sessionName, errEvt)
	errData, _ := json.Marshal(errEvt)
	publishLine(sessionName, fmt.Sprintf("data: %s\n\n", errData))
}

// ─── Hidden message helpers ──────────────────────────────────────────

// isMessageHidden checks if a message's metadata contains hidden: true.
func isMessageHidden(metadata interface{}) bool {
	if metadata == nil {
		return false
	}
	m, ok := metadata.(map[string]interface{})
	if !ok {
		return false
	}
	hidden, _ := m["hidden"].(bool)
	return hidden
}

// emitHiddenMessageMetadata persists and broadcasts a RAW event that
// tells the frontend to hide a specific message (e.g. auto-sent workflow
// prompts or initial prompts).
func emitHiddenMessageMetadata(sessionName, runID, threadID, messageID string) {
	evt := map[string]interface{}{
		"type":     "RAW",
		"threadId": threadID,
		"runId":    runID,
		"event": map[string]interface{}{
			"type":      "message_metadata",
			"messageId": messageID,
			"hidden":    true,
		},
	}
	persistEvent(sessionName, evt)
	data, _ := json.Marshal(evt)
	publishLine(sessionName, fmt.Sprintf("data: %s\n\n", data))
}

// persistStreamedEvent parses a raw JSON event, ensures IDs, and
// appends it to the event log.  No in-memory state, no broadcasting.
//
// NOTE: We intentionally do NOT inject timestamps.  The AG-UI spec
// defines timestamp as z.number().optional() (epoch ms).  If the
// runner omits it, the field stays absent — the proxy should not
// invent fields the source didn't emit.
func persistStreamedEvent(sessionID, runID, threadID, jsonData string) {
	var event map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &event); err != nil {
		return
	}

	// Ensure required fields (threadId + runId are needed for compaction)
	if event["threadId"] == nil || event["threadId"] == "" {
		event["threadId"] = threadID
	}
	if event["runId"] == nil || event["runId"] == "" {
		event["runId"] = runID
	}

	persistEvent(sessionID, event)

	// Update lastActivityTime on CR for activity events (debounced).
	// Extract event type to check; projectName is derived from the
	// sessionID-to-project mapping populated by HandleAGUIRunProxy.
	eventType, _ := event["type"].(string)
	if isActivityEvent(eventType) {
		if projectName, ok := sessionProjectMap.Load(sessionID); ok {
			updateLastActivityTime(projectName.(string), sessionID, eventType == types.EventTypeRunStarted)
		}
	}
}

// ─── POST /agui/interrupt ────────────────────────────────────────────

// HandleAGUIInterrupt sends interrupt signal to the runner.
func HandleAGUIInterrupt(c *gin.Context) {
	projectName := c.Param("projectName")
	sessionName := c.Param("sessionName")

	reqK8s, _ := handlers.GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}
	if !checkAccess(reqK8s, projectName, sessionName, "update") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	runnerURL := getRunnerEndpoint(projectName, sessionName)
	interruptURL := strings.TrimSuffix(runnerURL, "/") + "/interrupt"

	req, err := http.NewRequest("POST", interruptURL, bytes.NewReader([]byte("{}")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if ps := getRunnerProxySecret(projectName, sessionName); ps != "" {
		req.Header.Set("Authorization", "Bearer "+ps)
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(resp.StatusCode, gin.H{"error": string(body)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Interrupt signal sent"})
}

// ─── POST /agui/feedback ─────────────────────────────────────────────

// HandleAGUIFeedback forwards feedback to the runner, which sends it to
// Langfuse and returns a RAW event.  The backend persists that event
// so it survives reconnects.
//
// RAW events don't need to be within run boundaries (RUN_STARTED/
// RUN_FINISHED), unlike CUSTOM events which cause AG-UI validation
// errors when replayed outside a run.
func HandleAGUIFeedback(c *gin.Context) {
	projectName := c.Param("projectName")
	sessionName := c.Param("sessionName")

	reqK8s, _ := handlers.GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}
	if !checkAccess(reqK8s, projectName, sessionName, "update") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	var metaEvent map[string]interface{}
	if err := c.ShouldBindJSON(&metaEvent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid feedback event: %v", err)})
		return
	}

	eventType, _ := metaEvent["type"].(string)
	if eventType != types.EventTypeMeta {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Expected META event type"})
		return
	}

	// Forward to runner — it sends to Langfuse and returns a RAW event
	runnerURL := getRunnerEndpoint(projectName, sessionName)
	feedbackURL := strings.TrimSuffix(runnerURL, "/") + "/feedback"

	bodyBytes, _ := json.Marshal(metaEvent)
	req, err := http.NewRequest("POST", feedbackURL, bytes.NewReader(bodyBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if ps := getRunnerProxySecret(projectName, sessionName); ps != "" {
		req.Header.Set("Authorization", "Bearer "+ps)
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		c.JSON(http.StatusAccepted, gin.H{"error": "Runner unavailable — feedback not recorded", "status": "failed"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("AGUI Feedback: runner returned %d for %s: %s", resp.StatusCode, sessionName, string(body))
		c.JSON(resp.StatusCode, gin.H{"error": "Runner rejected feedback", "status": "failed"})
		return
	}

	// Runner returned a RAW event — persist it directly (no run wrapping needed).
	var rawEvent map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawEvent); err != nil {
		log.Printf("AGUI Feedback: failed to decode runner response for %s: %v", sessionName, err)
		c.JSON(http.StatusOK, gin.H{"message": "Feedback sent but not persisted", "status": "sent"})
		return
	}

	go func() {
		threadID := sessionName
		rawEvent["threadId"] = threadID
		persistEvent(sessionName, rawEvent)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "Feedback submitted", "status": "sent"})
}

// ─── GET /agui/capabilities ──────────────────────────────────────────

// HandleCapabilities proxies GET /capabilities to the runner.
func HandleCapabilities(c *gin.Context) {
	projectName := c.Param("projectName")
	sessionName := c.Param("sessionName")

	reqK8s, _ := handlers.GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}
	if !checkAccess(reqK8s, projectName, sessionName, "get") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	runnerURL := getRunnerEndpoint(projectName, sessionName)
	capURL := strings.TrimSuffix(runnerURL, "/") + "/capabilities"

	req, err := http.NewRequest("GET", capURL, nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"framework": "unknown"})
		return
	}
	if ps := getRunnerProxySecret(projectName, sessionName); ps != "" {
		req.Header.Set("Authorization", "Bearer "+ps)
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"framework":         "unknown",
			"agent_features":    []interface{}{},
			"platform_features": []interface{}{},
			"file_system":       false,
			"mcp":               false,
		})
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.JSON(http.StatusOK, gin.H{"framework": "unknown"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ─── GET /mcp/status ─────────────────────────────────────────────────

// HandleMCPStatus proxies MCP status requests to the runner.
func HandleMCPStatus(c *gin.Context) {
	projectName := c.Param("projectName")
	sessionName := c.Param("sessionName")

	reqK8s, _ := handlers.GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}
	if !checkAccess(reqK8s, projectName, sessionName, "get") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	runnerURL := getRunnerEndpoint(projectName, sessionName)
	mcpURL := strings.TrimSuffix(runnerURL, "/") + "/mcp/status"

	req, err := http.NewRequest("GET", mcpURL, nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"servers": []interface{}{}, "totalCount": 0})
		return
	}
	if ps := getRunnerProxySecret(projectName, sessionName); ps != "" {
		req.Header.Set("Authorization", "Bearer "+ps)
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"servers": []interface{}{}, "totalCount": 0})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusOK, gin.H{"servers": []interface{}{}, "totalCount": 0})
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.JSON(http.StatusOK, gin.H{"servers": []interface{}{}, "totalCount": 0})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ─── Runner connection ───────────────────────────────────────────────

// runnerHTTPClient is a shared HTTP client for long-lived SSE connections
// to runner pods.  Reusing the transport avoids per-call socket churn and
// background goroutine growth under load.
var runnerHTTPClient = &http.Client{
	Timeout: 0, // No overall timeout — SSE streams are long-lived
	Transport: &http.Transport{
		IdleConnTimeout:       5 * time.Minute,  // Close idle connections after 5 min
		ResponseHeaderTimeout: 30 * time.Second, // Fail fast if runner doesn't respond to headers
	},
}

// connectToRunner POSTs to the runner with retry and exponential backoff.
//
// The runner pod may not be reachable immediately after creation due to
// container startup time and K8s Service DNS propagation. Retries on
// "connection refused", "no such host", and "dial tcp" errors with
// exponential backoff (500ms initial, 1.5x, capped at 5s, 15 attempts).
//
// proxySecret is the RUNNER_PROXY_SECRET shared between the backend and the runner.
// When non-empty it is sent as "Authorization: Bearer {proxySecret}" so the runner
// can reject direct connections that bypass the backend proxy.
func connectToRunner(runnerURL string, bodyBytes []byte, proxySecret string) (*http.Response, error) {
	maxAttempts := 15
	retryDelay := 500 * time.Millisecond
	maxDelay := 5 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequest("POST", runnerURL, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")
		if proxySecret != "" {
			req.Header.Set("Authorization", "Bearer "+proxySecret)
		}

		resp, err := runnerHTTPClient.Do(req)
		if err == nil {
			return resp, nil
		}

		errStr := err.Error()
		isTransient := strings.Contains(errStr, "connection refused") ||
			strings.Contains(errStr, "no such host") ||
			strings.Contains(errStr, "dial tcp")

		if !isTransient {
			return nil, fmt.Errorf("runner request failed: %w", err)
		}

		if attempt < maxAttempts {
			log.Printf("AGUI Proxy: runner not ready (attempt %d/%d), retrying in %v", attempt, maxAttempts, retryDelay)
			time.Sleep(retryDelay)
			retryDelay = time.Duration(float64(retryDelay) * 1.5)
			if retryDelay > maxDelay {
				retryDelay = maxDelay
			}
		}
	}

	return nil, fmt.Errorf("runner not available after %d attempts", maxAttempts)
}

// getRunnerEndpoint returns the AG-UI server endpoint for a session.
// The operator creates a Service named "session-{sessionName}" in the
// project namespace.
func getRunnerEndpoint(projectName, sessionName string) string {
	return fmt.Sprintf("http://session-%s.%s.svc.cluster.local:8001/", sessionName, projectName)
}

// drainLiveChannel discards any buffered lines already in the channel.
// Called after replaying persisted events to skip duplicates that were
// published to the live pipe while the replay was in progress.
func drainLiveChannel(ch <-chan string) {
	for {
		select {
		case <-ch:
			// discard — already replayed from persisted events
		default:
			return // buffer is empty
		}
	}
}

// truncID returns the first 8 chars of an ID for logging, or the
// full string if shorter.
func truncID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// ─── Auth helper ─────────────────────────────────────────────────────

// checkAccess performs a SelfSubjectAccessReview for the given verb on
// the AgenticSession resource.
func checkAccess(reqK8s kubernetes.Interface, projectName, sessionName, verb string) bool {
	ssar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Group:     "vteam.ambient-code",
				Resource:  "agenticsessions",
				Verb:      verb,
				Namespace: projectName,
				Name:      sessionName,
			},
		},
	}
	res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(
		context.Background(), ssar, metav1.CreateOptions{},
	)
	if err != nil || !res.Status.Allowed {
		return false
	}
	return true
}

// ─── Display name generation ─────────────────────────────────────────

// triggerDisplayNameGenerationIfNeeded checks if the session needs a
// display name and triggers async generation using the first user message.
func triggerDisplayNameGenerationIfNeeded(projectName, sessionName string, messages []types.Message) {
	var userMessage string
	for _, msg := range messages {
		if msg.Role == "user" && msg.Content != "" {
			userMessage = msg.Content
			break
		}
	}
	if userMessage == "" {
		return
	}

	if handlers.DynamicClient == nil {
		return
	}

	gvr := handlers.GetAgenticSessionV1Alpha1Resource()
	item, err := handlers.DynamicClient.Resource(gvr).Namespace(projectName).Get(
		context.Background(), sessionName, metav1.GetOptions{},
	)
	if err != nil {
		return
	}

	spec, found, err := unstructured.NestedMap(item.Object, "spec")
	if err != nil || !found {
		return
	}

	// Skip if this message is the auto-sent initialPrompt
	initialPrompt, _, _ := unstructured.NestedString(spec, "initialPrompt")
	if initialPrompt != "" && strings.TrimSpace(userMessage) == strings.TrimSpace(initialPrompt) {
		return
	}

	if !handlers.ShouldGenerateDisplayName(spec) {
		return
	}

	sessionCtx := handlers.ExtractSessionContext(spec)
	handlers.GenerateDisplayNameAsync(projectName, sessionName, userMessage, sessionCtx)
}

// isActivityEvent returns true for AG-UI event types that indicate session activity.
func isActivityEvent(eventType string) bool {
	switch eventType {
	case types.EventTypeRunStarted,
		types.EventTypeTextMessageStart,
		types.EventTypeTextMessageContent,
		types.EventTypeToolCallStart:
		return true
	default:
		return false
	}
}

// updateLastActivityTime updates the lastActivityTime field on the AgenticSession CR status.
// Updates are debounced to avoid excessive API calls. RUN_STARTED events bypass the debounce
// to immediately mark the session as active.
func updateLastActivityTime(projectName, sessionName string, immediate bool) {
	if handlers.DynamicClient == nil {
		log.Printf("Activity tracking: DynamicClient is nil, skipping update for %s/%s", projectName, sessionName)
		return
	}

	key := projectName + "/" + sessionName
	now := time.Now()

	if !immediate {
		if lastUpdate, ok := lastActivityUpdateTimes.Load(key); ok {
			if now.Sub(lastUpdate.(time.Time)) < activityDebounceInterval {
				return // Debounce: too soon since last update
			}
		}
	}

	lastActivityUpdateTimes.Store(key, now)

	// Bound concurrency: drop the update if all slots are busy (debounce will retry later).
	select {
	case activityUpdateSem <- struct{}{}:
	default:
		return
	}

	// Run in goroutine to avoid blocking event processing
	go func() {
		defer func() { <-activityUpdateSem }()

		gvr := handlers.GetAgenticSessionV1Alpha1Resource()
		ctx, cancel := context.WithTimeout(context.Background(), activityUpdateTimeout)
		defer cancel()

		obj, err := handlers.DynamicClient.Resource(gvr).Namespace(projectName).Get(ctx, sessionName, metav1.GetOptions{})
		if err != nil {
			log.Printf("Activity tracking: failed to get session %s/%s: %v", projectName, sessionName, err)
			return
		}

		status, _, _ := unstructured.NestedMap(obj.Object, "status")
		if status == nil {
			status = make(map[string]any)
		}
		status["lastActivityTime"] = now.UTC().Format(time.RFC3339)
		if err := unstructured.SetNestedField(obj.Object, status, "status"); err != nil {
			log.Printf("Activity tracking: failed to set status for %s/%s: %v", projectName, sessionName, err)
			return
		}

		_, err = handlers.DynamicClient.Resource(gvr).Namespace(projectName).UpdateStatus(ctx, obj, metav1.UpdateOptions{})
		if err != nil {
			log.Printf("Activity tracking: failed to update lastActivityTime for %s/%s: %v", projectName, sessionName, err)
		}
	}()
}
