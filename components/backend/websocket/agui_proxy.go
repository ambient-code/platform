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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
)

// ─── POST /agui/run — main CopilotKit entry point ───────────────────

// HandleAGUIRunProxy proxies AG-UI run requests to the runner's FastAPI
// server.  CopilotKit's HttpAgent POSTs here with Accept: text/event-stream
// and receives an SSE stream back.
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

	// Count actual messages for reconnect detection
	var rawMessages []json.RawMessage
	if len(input.Messages) > 0 {
		_ = json.Unmarshal(input.Messages, &rawMessages)
	}
	hasMessages := len(rawMessages) > 0

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

	log.Printf("AGUI Proxy: run=%s session=%s/%s msgs=%d", truncID(runID), projectName, sessionName, len(rawMessages))

	// Trigger display name generation on first real user message
	if hasMessages {
		var minimalMsgs []types.Message
		for _, raw := range rawMessages {
			var msg types.Message
			if err := json.Unmarshal(raw, &msg); err == nil {
				minimalMsgs = append(minimalMsgs, msg)
			}
		}
		go triggerDisplayNameGenerationIfNeeded(projectName, sessionName, minimalMsgs)
	}

	// ── SSE response ─────────────────────────────────────────────────
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	// ── Reconnect (page refresh): replay history then tail for live events ──
	// Subscribe to the live broadcast pipe FIRST, then load persisted events.
	// This ordering prevents a race where events published between
	// loadEvents() and subscribeLive() would be missed by the client.
	if !hasMessages {
		log.Printf("AGUI Proxy (reconnect): serving history for %s/%s", projectName, sessionName)

		// Subscribe to live pipe BEFORE loading events from disk so we
		// capture any events published while we read the file.
		liveCh, cleanup := subscribeLive(sessionName)
		defer cleanup()

		events := loadEvents(sessionName)

		// No events — fresh session, return immediately.
		if len(events) == 0 {
			log.Printf("AGUI Proxy (reconnect): no events for %s", sessionName)
			return
		}

		// Check if the run is finished.
		runFinished := false
		if last := events[len(events)-1]; last != nil {
			if t, _ := last["type"].(string); t == types.EventTypeRunFinished {
				runFinished = true
			}
		}

		// Finished runs get compacted replay (fast, small).
		// Active runs get raw events (preserves streaming structure for CopilotKit).
		if runFinished {
			compacted := compactStreamingEvents(events)
			log.Printf("AGUI Proxy (reconnect): %d raw → %d compacted events for %s (finished)", len(events), len(compacted), sessionName)
			for _, evt := range compacted {
				writeSSEEvent(c.Writer, evt)
			}
			c.Writer.Flush()
			return // cleanup() via defer; subscription was cheap and harmless
		}

		// Active run — send raw events as-is.
		log.Printf("AGUI Proxy (reconnect): replaying %d raw events for %s (running)", len(events), sessionName)
		for _, evt := range events {
			writeSSEEvent(c.Writer, evt)
		}
		c.Writer.Flush()

		// Drain live events buffered during replay — they are already
		// covered by the persisted events we just sent.  Because
		// persistStreamedEvent is called synchronously before publishLine,
		// any event in the live channel was also on disk when loadEvents()
		// ran (or arrived during replay, which is already replayed).
		drainLiveChannel(liveCh)

		clientGone := c.Request.Context().Done()
		for {
			select {
			case <-clientGone:
				log.Printf("AGUI Proxy (reconnect): client disconnected for %s", sessionName)
				return
			case line, ok := <-liveCh:
				if !ok {
					return
				}
				fmt.Fprint(c.Writer, line)
				c.Writer.Flush()

				// Check if this line contains RUN_FINISHED
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "data: ") && strings.Contains(trimmed, `"type":"RUN_FINISHED"`) {
					log.Printf("AGUI Proxy (reconnect): run finished for %s", sessionName)
					return
				}
			}
		}
	}

	// ── New message: forward to runner ────────────────────────────
	runnerURL := getRunnerEndpoint(projectName, sessionName)
	bodyBytes, err := json.Marshal(input)
	if err != nil {
		writeSSEError(c.Writer, "Failed to serialize input", threadID, runID)
		return
	}

	log.Printf("AGUI Proxy: connecting to runner at %s", runnerURL)
	resp, err := connectToRunner(runnerURL, bodyBytes)
	if err != nil {
		log.Printf("AGUI Proxy: runner unavailable for %s: %v", sessionName, err)
		writeSSERun(c.Writer, "RUN_STARTED", threadID, runID)
		writeSSEError(c.Writer, "Runner is not available", threadID, runID)
		c.Writer.Flush()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("AGUI Proxy: runner returned %d: %s", resp.StatusCode, string(body))
		writeSSERun(c.Writer, "RUN_STARTED", threadID, runID)
		writeSSEError(c.Writer, fmt.Sprintf("Runner error: HTTP %d", resp.StatusCode), threadID, runID)
		c.Writer.Flush()
		return
	}

	// ── Pipe SSE from runner to client, persist each event ───────────
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

		// Persist every event synchronously to JSONL (backup for recovery).
		if strings.HasPrefix(trimmed, "data: ") {
			jsonData := strings.TrimPrefix(trimmed, "data: ")
			persistStreamedEvent(sessionName, runID, threadID, jsonData)
		}

		// Publish raw SSE line to any connect handler tailing this session.
		publishLine(sessionName, line)

		// Forward raw SSE line to client
		fmt.Fprint(c.Writer, line)
		c.Writer.Flush()
	}

	log.Printf("AGUI Proxy: run %s stream ended", runID[:8])
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

// connectToRunner POSTs to the runner with fast-fail behaviour.
//   - 2 attempts max
//   - Immediate fail on "no such host" (runner pod doesn't exist)
//   - 1s retry only on "connection refused" (runner still starting)
func connectToRunner(runnerURL string, bodyBytes []byte) (*http.Response, error) {
	maxAttempts := 2
	retryDelay := 1 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequest("POST", runnerURL, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")

		resp, err := runnerHTTPClient.Do(req)
		if err == nil {
			return resp, nil
		}

		errStr := err.Error()
		// "no such host" = runner pod/service doesn't exist — no point retrying
		if strings.Contains(errStr, "no such host") {
			return nil, fmt.Errorf("runner not available: %w", err)
		}

		// Only retry on connection refused (runner starting up)
		if !strings.Contains(errStr, "connection refused") && !strings.Contains(errStr, "dial tcp") {
			return nil, fmt.Errorf("runner request failed: %w", err)
		}

		if attempt < maxAttempts {
			log.Printf("AGUI Proxy: runner not ready (attempt %d/%d), retrying in %v", attempt, maxAttempts, retryDelay)
			time.Sleep(retryDelay)
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

// ─── SSE formatting helpers ──────────────────────────────────────────

func writeSSERun(w http.ResponseWriter, eventType, threadID, runID string) {
	data, _ := json.Marshal(map[string]string{
		"type":     eventType,
		"threadId": threadID,
		"runId":    runID,
	})
	fmt.Fprintf(w, "data: %s\n\n", data)
}

func writeSSEError(w http.ResponseWriter, message, threadID, runID string) {
	data, _ := json.Marshal(map[string]string{
		"type":     "RUN_ERROR",
		"message":  message,
		"threadId": threadID,
		"runId":    runID,
	})
	fmt.Fprintf(w, "data: %s\n\n", data)
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
