// Package websocket provides AG-UI protocol endpoints including HTTP proxy to runner.
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
)

// HandleAGUIRunProxy proxies AG-UI run requests to runner's FastAPI server
// This replaces the WebSocket-based communication with HTTP/SSE
func HandleAGUIRunProxy(c *gin.Context) {
	projectName := c.Param("projectName")
	sessionName := c.Param("sessionName")

	// SECURITY: Authenticate user and get user-scoped K8s client
	reqK8s, _ := handlers.GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// SECURITY: Verify user has permission to update this session
	ctx := context.Background()
	ssar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Group:     "vteam.ambient-code",
				Resource:  "agenticsessions",
				Verb:      "update",
				Namespace: projectName,
				Name:      sessionName,
			},
		},
	}
	res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, metav1.CreateOptions{})
	if err != nil || !res.Status.Allowed {
		log.Printf("AGUI Proxy: User not authorized to update session %s/%s", projectName, sessionName)
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	log.Printf("AGUI Proxy: Forwarding run request for %s/%s", projectName, sessionName)

	var input types.RunAgentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("AGUI Proxy: Failed to parse input: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid input: %v", err)})
		return
	}
	log.Printf("AGUI Proxy: Input has %d messages", len(input.Messages))

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

	log.Printf("AGUI Proxy: Creating run %s for session %s (threadId=%s)", runID, sessionName, threadID)

	// Create run state for tracking
	runState := &AGUIRunState{
		ThreadID:     threadID,
		RunID:        runID,
		ParentRunID:  input.ParentRunID,
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

	// Persist run metadata
	go persistRunMetadata(sessionName, types.AGUIRunMetadata{
		ThreadID:    threadID,
		RunID:       runID,
		ParentRunID: input.ParentRunID,
		SessionName: sessionName,
		ProjectName: projectName,
		StartedAt:   runState.StartedAt.Format(time.RFC3339),
		Status:      "running",
	})

	// NOTE: User messages are now echoed by the runner (AG-UI server pattern)
	// The runner emits TEXT_MESSAGE_START/CONTENT/END events which are persisted
	// when they stream through this proxy. No need to echo them here.

	// Trigger async display name generation on first user message
	// This generates a descriptive name using Claude Haiku based on the message
	go triggerDisplayNameGenerationIfNeeded(projectName, sessionName, input.Messages)

	// Get runner endpoint
	runnerURL, err := getRunnerEndpoint(projectName, sessionName)
	if err != nil {
		log.Printf("AGUI Proxy: Failed to get runner endpoint: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Runner not available"})
		return
	}

	log.Printf("AGUI Proxy: Runner endpoint: %s", runnerURL)

	// Serialize input for proxy request
	bodyBytes, err := json.Marshal(input)
	if err != nil {
		log.Printf("AGUI Proxy: Failed to serialize input: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize input"})
		return
	}

	log.Printf("AGUI Proxy: Run %s starting, will consume runner stream in background", runID)

	// Start background goroutine that owns the entire HTTP lifecycle
	// This ensures the connection stays open after we return to client
	// Note: We use context.Background() (not request context) because this goroutine
	// must continue running after the HTTP request completes. The timeout and terminal
	// event handling prevent unbounded goroutine accumulation.
	go func() {
		// Create request with long timeout (detached from client request lifecycle)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()

		// Execute request with retries (runner may not be ready immediately after startup)
		client := &http.Client{
			Timeout: 0, // No timeout, context handles it
		}

		var resp *http.Response
		maxRetries := 15
		retryDelay := 500 * time.Millisecond

		for attempt := 1; attempt <= maxRetries; attempt++ {
			// Create fresh request for each attempt (body reader needs reset)
			proxyReq, err := http.NewRequestWithContext(ctx, "POST", runnerURL, bytes.NewReader(bodyBytes))
			if err != nil {
				log.Printf("AGUI Proxy: Failed to create request in background: %v", err)
				updateRunStatus(runID, "error")
				return
			}

			// Forward headers
			proxyReq.Header.Set("Content-Type", "application/json")
			proxyReq.Header.Set("Accept", "text/event-stream")

			resp, err = client.Do(proxyReq)
			if err == nil {
				break // Success!
			}

			// Check if it's a connection refused error (runner not ready yet)
			errStr := err.Error()
			isConnectionRefused := strings.Contains(errStr, "connection refused") ||
				strings.Contains(errStr, "no such host") ||
				strings.Contains(errStr, "dial tcp")

			if !isConnectionRefused || attempt == maxRetries {
				log.Printf("AGUI Proxy: Background request failed after %d attempts: %v", attempt, err)
				updateRunStatus(runID, "error")
				return
			}

			log.Printf("AGUI Proxy: Runner not ready (attempt %d/%d), retrying in %v...", attempt, maxRetries, retryDelay)

			select {
			case <-ctx.Done():
				log.Printf("AGUI Proxy: Context cancelled during retry for run %s", runID)
				return
			case <-time.After(retryDelay):
				// Exponential backoff with cap at 5 seconds
				retryDelay = time.Duration(float64(retryDelay) * 1.5)
				if retryDelay > 5*time.Second {
					retryDelay = 5 * time.Second
				}
			}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Printf("AGUI Proxy: Runner returned status %d: %s", resp.StatusCode, string(body))
			updateRunStatus(runID, "error")
			return
		}

		log.Printf("AGUI Proxy: Background stream started for run %s", runID)

		reader := bufio.NewReader(resp.Body)

		for {
			// Check if context was cancelled (timeout or cleanup)
			select {
			case <-ctx.Done():
				log.Printf("AGUI Proxy: Context cancelled for run %s", runID)
				return
			default:
			}

			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					log.Printf("AGUI Proxy: Background stream ended for run %s", runID)
					break
				}
				log.Printf("AGUI Proxy: Background stream read error: %v", err)
				break
			}

			// Parse and persist SSE events
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data: ") {
				jsonData := strings.TrimPrefix(line, "data: ")
				handleStreamedEvent(sessionName, runID, threadID, jsonData, runState)
			}
		}

		// Mark run as completed
		aguiRunsMu.RLock()
		currentStatus := "completed"
		if state, exists := aguiRuns[runID]; exists && state.Status == "error" {
			currentStatus = "error"
		}
		aguiRunsMu.RUnlock()

		updateRunStatus(runID, currentStatus)
		log.Printf("AGUI Proxy: Background stream completed for run %s (status=%s)", runID, currentStatus)
	}()

	// Return run metadata immediately (don't wait for stream)
	// Events will be broadcast to GET /agui/events subscribers
	streamURL := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s/agui/events", projectName, sessionName)

	c.JSON(http.StatusOK, gin.H{
		"threadId":  threadID,
		"runId":     runID,
		"streamUrl": streamURL,
		"status":    "started",
	})
}

// handleStreamedEvent parses and persists a streamed AG-UI event
func handleStreamedEvent(sessionID, runID, threadID, jsonData string, runState *AGUIRunState) {
	var event map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &event); err != nil {
		log.Printf("AGUI Proxy: Failed to parse event JSON: %v", err)
		return
	}

	eventType, _ := event["type"].(string)

	// Ensure threadId, runId, and timestamp are set
	if _, ok := event["threadId"]; !ok {
		event["threadId"] = threadID
	}
	if _, ok := event["runId"]; !ok {
		event["runId"] = runID
	}
	// Add timestamp if not present - critical for message timestamp tracking
	if _, ok := event["timestamp"]; !ok {
		event["timestamp"] = time.Now().UTC().Format(types.AGUITimestampFormat)
	}

	// Check for terminal events
	switch eventType {
	case types.EventTypeRunFinished:
		updateRunStatus(runID, "completed")
	case types.EventTypeRunError:
		updateRunStatus(runID, "error")
	}

	// Persist event
	persistAGUIEventMap(sessionID, runID, event)

	// Broadcast to subscribers (for SSE /events endpoint)
	if runState != nil {
		runState.BroadcastFull(event)
	}

	// Also broadcast to thread subscribers
	broadcastToThread(sessionID, event)
}

// updateRunStatus updates the status of a run
func updateRunStatus(runID, status string) {
	aguiRunsMu.Lock()
	if state, exists := aguiRuns[runID]; exists {
		state.Status = status
		// Update persisted metadata
		go persistRunMetadata(state.SessionID, types.AGUIRunMetadata{
			ThreadID:    state.ThreadID,
			RunID:       state.RunID,
			ParentRunID: state.ParentRunID,
			SessionName: state.SessionID,
			ProjectName: state.ProjectName,
			StartedAt:   state.StartedAt.Format(time.RFC3339),
			Status:      status,
		})
	}
	aguiRunsMu.Unlock()
}

// HandleAGUIInterrupt sends interrupt signal to runner to stop current execution
// POST /api/projects/:projectName/agentic-sessions/:sessionName/agui/interrupt
func HandleAGUIInterrupt(c *gin.Context) {
	projectName := c.Param("projectName")
	sessionName := c.Param("sessionName")

	// SECURITY: Authenticate user and get user-scoped K8s client
	reqK8s, _ := handlers.GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// SECURITY: Verify user has permission to update this session
	ctx := context.Background()
	ssar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Group:     "vteam.ambient-code",
				Resource:  "agenticsessions",
				Verb:      "update",
				Namespace: projectName,
				Name:      sessionName,
			},
		},
	}
	res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, metav1.CreateOptions{})
	if err != nil || !res.Status.Allowed {
		log.Printf("AGUI Interrupt: User not authorized to update session %s/%s", projectName, sessionName)
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	log.Printf("AGUI Interrupt: Request for %s/%s", projectName, sessionName)

	var input struct {
		RunID string `json:"runId"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "runId required"})
		return
	}

	// Get runner endpoint
	runnerURL, err := getRunnerEndpoint(projectName, sessionName)
	if err != nil {
		log.Printf("AGUI Interrupt: Failed to get runner endpoint: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Runner not available"})
		return
	}

	interruptURL := strings.TrimSuffix(runnerURL, "/") + "/interrupt"
	log.Printf("AGUI Interrupt: Forwarding to runner: %s", interruptURL)

	// POST to runner's interrupt endpoint
	req, err := http.NewRequest("POST", interruptURL, bytes.NewReader([]byte("{}")))
	if err != nil {
		log.Printf("AGUI Interrupt: Failed to create request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("AGUI Interrupt: Request failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("AGUI Interrupt: Runner returned %d: %s", resp.StatusCode, string(body))
		c.JSON(resp.StatusCode, gin.H{"error": string(body)})
		return
	}

	log.Printf("AGUI Interrupt: Successfully interrupted run %s", input.RunID)
	c.JSON(http.StatusOK, gin.H{"message": "Interrupt signal sent"})
}

// HandleMCPStatus proxies MCP status requests to runner
// GET /api/projects/:projectName/agentic-sessions/:sessionName/mcp/status
func HandleMCPStatus(c *gin.Context) {
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
		log.Printf("MCP Status: User not authorized to read session %s/%s", projectName, sessionName)
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	// Get runner endpoint
	runnerURL, err := getRunnerEndpoint(projectName, sessionName)
	if err != nil {
		log.Printf("MCP Status: Failed to get runner endpoint: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Runner not available"})
		return
	}

	mcpStatusURL := strings.TrimSuffix(runnerURL, "/") + "/mcp/status"
	log.Printf("MCP Status: Forwarding to runner: %s", mcpStatusURL)

	// GET from runner's MCP status endpoint
	req, err := http.NewRequest("GET", mcpStatusURL, nil)
	if err != nil {
		log.Printf("MCP Status: Failed to create request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("MCP Status: Request failed: %v", err)
		// Runner might not be running yet - return empty list
		c.JSON(http.StatusOK, gin.H{"servers": []interface{}{}, "totalCount": 0})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("MCP Status: Runner returned %d: %s", resp.StatusCode, string(body))
		c.JSON(http.StatusOK, gin.H{"servers": []interface{}{}, "totalCount": 0})
		return
	}

	// Forward runner response to client
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("MCP Status: Failed to decode response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse runner response"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// getRunnerEndpoint returns the AG-UI server endpoint for a session
// The operator creates a Service named "session-{sessionName}" in the project namespace
func getRunnerEndpoint(projectName, sessionName string) (string, error) {
	// Use naming convention for service discovery
	// Format: http://session-{sessionName}.{projectName}.svc.cluster.local:8001/
	// The operator creates this Service automatically when spawning the runner Job
	return fmt.Sprintf("http://session-%s.%s.svc.cluster.local:8001/", sessionName, projectName), nil
}

// broadcastToThread sends event to all thread-level subscribers
func broadcastToThread(sessionID string, event interface{}) {
	threadSubscribersMu.RLock()
	subs, exists := threadSubscribers[sessionID]
	threadSubscribersMu.RUnlock()

	if !exists {
		return
	}

	for ch := range subs {
		select {
		case ch <- event:
		default:
			// Channel full, skip
		}
	}
}

// triggerDisplayNameGenerationIfNeeded checks if the session needs a display name
// and triggers async generation using the first REAL user message (not auto-sent initialPrompt)
func triggerDisplayNameGenerationIfNeeded(projectName, sessionName string, messages []types.Message) {
	// Extract first user message
	var userMessage string
	for _, msg := range messages {
		if msg.Role == "user" && msg.Content != "" {
			userMessage = msg.Content
			break
		}
	}

	if userMessage == "" {
		log.Printf("DisplayNameGen: No user message found in run request for %s/%s", projectName, sessionName)
		return
	}

	// Check if session already has a display name
	if handlers.DynamicClient == nil {
		log.Printf("DisplayNameGen: DynamicClient not initialized, skipping display name generation")
		return
	}

	gvr := handlers.GetAgenticSessionV1Alpha1Resource()
	ctx := context.Background()

	item, err := handlers.DynamicClient.Resource(gvr).Namespace(projectName).Get(ctx, sessionName, metav1.GetOptions{})
	if err != nil {
		log.Printf("DisplayNameGen: Failed to get session %s/%s: %v", projectName, sessionName, err)
		return
	}

	// Extract spec using unstructured helpers (per CLAUDE.md guidelines)
	spec, found, err := unstructured.NestedMap(item.Object, "spec")
	if err != nil || !found {
		log.Printf("DisplayNameGen: Failed to get spec for %s/%s", projectName, sessionName)
		return
	}

	// Skip if this message is the auto-sent initialPrompt (not a real user message)
	initialPrompt, _, _ := unstructured.NestedString(spec, "initialPrompt")
	if initialPrompt != "" && strings.TrimSpace(userMessage) == strings.TrimSpace(initialPrompt) {
		log.Printf("DisplayNameGen: Skipping auto-sent initialPrompt for %s/%s", projectName, sessionName)
		return
	}

	// Check if display name generation is needed
	if !handlers.ShouldGenerateDisplayName(spec) {
		log.Printf("DisplayNameGen: Session %s/%s already has display name, skipping", projectName, sessionName)
		return
	}

	// Extract session context for better name generation
	sessionCtx := handlers.ExtractSessionContext(spec)

	log.Printf("DisplayNameGen: Triggering async generation for %s/%s with message: %q",
		projectName, sessionName, truncateForLog(userMessage, 50))

	// Trigger async generation (runs in background, fails silently)
	handlers.GenerateDisplayNameAsync(projectName, sessionName, userMessage, sessionCtx)
}

// truncateForLog truncates a string for logging purposes
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// HandleAGUIFeedback forwards AG-UI META events (user feedback) to the runner
// POST /api/projects/:projectName/agentic-sessions/:sessionName/agui/feedback
// Frontend constructs the full META event, backend validates and forwards
// See: https://docs.ag-ui.com/drafts/meta-events#user-feedback
func HandleAGUIFeedback(c *gin.Context) {
	// SECURITY: Sanitize URL path params to prevent log injection
	projectName := handlers.SanitizeForLog(c.Param("projectName"))
	sessionName := handlers.SanitizeForLog(c.Param("sessionName"))

	// SECURITY: Authenticate user and get user-scoped K8s client
	reqK8s, _ := handlers.GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// SECURITY: Verify user has permission to update this session
	ctx := context.Background()
	ssar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Group:     "vteam.ambient-code",
				Resource:  "agenticsessions",
				Verb:      "update",
				Namespace: projectName,
				Name:      sessionName,
			},
		},
	}
	res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, metav1.CreateOptions{})
	if err != nil || !res.Status.Allowed {
		log.Printf("AGUI Feedback: User not authorized to update session %s/%s", projectName, sessionName)
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	// Parse AG-UI META event from frontend
	// Frontend constructs the full event, we just validate and forward
	var metaEvent map[string]interface{}
	if err := c.ShouldBindJSON(&metaEvent); err != nil {
		log.Printf("AGUI Feedback: Failed to parse META event: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid META event: %v", err)})
		return
	}

	// Validate it's a META event
	eventType, ok := metaEvent["type"].(string)
	if !ok || eventType != types.EventTypeMeta {
		log.Printf("AGUI Feedback: Invalid event type: %v", eventType)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Expected META event type"})
		return
	}

	// Extract metaType for logging
	metaType, _ := metaEvent["metaType"].(string)
	username := handlers.SanitizeForLog(c.GetHeader("X-Forwarded-User"))
	log.Printf("AGUI Feedback: Received %s feedback from %s for session %s/%s",
		handlers.SanitizeForLog(metaType), username, projectName, sessionName)

	// Get runner endpoint
	runnerURL, err := getRunnerEndpoint(projectName, sessionName)
	if err != nil {
		log.Printf("AGUI Feedback: Failed to get runner endpoint: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Runner not available"})
		return
	}

	// Serialize event for POST to runner (forward as-is)
	bodyBytes, err := json.Marshal(metaEvent)
	if err != nil {
		log.Printf("AGUI Feedback: Failed to serialize META event: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize event"})
		return
	}

	// POST to runner's feedback endpoint
	feedbackURL := strings.TrimSuffix(runnerURL, "/") + "/feedback"
	log.Printf("AGUI Feedback: Forwarding META event to runner: %s", feedbackURL)

	req, err := http.NewRequest("POST", feedbackURL, bytes.NewReader(bodyBytes))
	if err != nil {
		log.Printf("AGUI Feedback: Failed to create request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Runner might not be running - log but don't fail (feedback is best-effort)
		log.Printf("AGUI Feedback: Request failed (runner may not be running): %v", err)
		c.JSON(http.StatusAccepted, gin.H{
			"message": "Feedback queued (runner not available)",
			"status":  "pending",
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("AGUI Feedback: Runner returned %d: %s", resp.StatusCode, string(body))
		c.JSON(resp.StatusCode, gin.H{"error": string(body)})
		return
	}

	log.Printf("AGUI Feedback: Successfully forwarded %s feedback to runner", handlers.SanitizeForLog(metaType))

	// Broadcast the META event on the event stream so UI can see feedback submissions
	// This allows the frontend to display "Feedback submitted" or track which traces have feedback
	broadcastToThread(sessionName, metaEvent)

	// CRITICAL: Persist the META event so it survives reconnects and session restarts
	// Without this, feedback events are lost when clients disconnect
	// Extract runId from event payload if present (feedback is associated with a specific run/message)
	runID := ""
	if payload, ok := metaEvent["payload"].(map[string]interface{}); ok {
		if rid, ok := payload["runId"].(string); ok {
			runID = rid
		}
	}
	// Fallback: try top-level runId
	if runID == "" {
		if rid, ok := metaEvent["runId"].(string); ok {
			runID = rid
		}
	}
	go persistAGUIEventMap(sessionName, runID, metaEvent)

	c.JSON(http.StatusOK, gin.H{
		"message": "Feedback submitted successfully",
		"status":  "sent",
	})
}
