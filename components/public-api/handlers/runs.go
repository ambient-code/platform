package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"

	"ambient-code-public-api/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateRun handles POST /v1/sessions/:id/runs
//
// Defense-in-depth: The gateway fetches the session phase before forwarding.
// The backend also validates phase transitions, so this is a redundant guard
// that provides faster feedback and reduces unnecessary backend writes.
func CreateRun(c *gin.Context) {
	project := GetProject(c)
	if !ValidateProjectName(project) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
		return
	}
	sessionID := c.Param("id")
	if !ValidateSessionID(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	phase, err := getSessionPhase(c, project, sessionID)
	if err != nil {
		return // getSessionPhase already wrote the error response
	}

	if phase != "running" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Session is not in a running state"})
		return
	}

	var req types.CreateRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Build AG-UI RunAgentInput messages
	messages := make([]map[string]interface{}, len(req.Messages))
	for i, msg := range req.Messages {
		m := map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if msg.ID != "" {
			m["id"] = msg.ID
		} else {
			m["id"] = uuid.New().String()
		}
		messages[i] = m
	}

	messagesJSON, err := json.Marshal(messages)
	if err != nil {
		log.Printf("Failed to marshal messages: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Build RunAgentInput
	backendReq := map[string]interface{}{
		"messages": json.RawMessage(messagesJSON),
	}

	if req.ThreadID != "" {
		backendReq["threadId"] = req.ThreadID
	} else {
		backendReq["threadId"] = sessionID
	}

	if req.RunID != "" {
		backendReq["runId"] = req.RunID
	}

	reqBody, err := json.Marshal(backendReq)
	if err != nil {
		log.Printf("Failed to marshal request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s/agui/run", project, sessionID)
	resp, cancel, err := ProxyRequest(c, http.MethodPost, path, reqBody)
	if err != nil {
		log.Printf("Backend request failed for create run: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer cancel()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		forwardErrorResponse(c, resp.StatusCode, body)
		return
	}

	var backendResp map[string]interface{}
	if err := json.Unmarshal(body, &backendResp); err != nil {
		log.Printf("Failed to parse backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	runID, _ := backendResp["runId"].(string)
	threadID, _ := backendResp["threadId"].(string)

	c.JSON(http.StatusAccepted, types.CreateRunResponse{
		RunID:    runID,
		ThreadID: threadID,
	})
}

// SendMessage handles POST /v1/sessions/:id/message
//
// Defense-in-depth: The gateway fetches the session phase before forwarding.
// The backend also validates phase transitions, so this is a redundant guard
// that provides faster feedback and reduces unnecessary backend writes.
func SendMessage(c *gin.Context) {
	project := GetProject(c)
	if !ValidateProjectName(project) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
		return
	}
	sessionID := c.Param("id")
	if !ValidateSessionID(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	phase, err := getSessionPhase(c, project, sessionID)
	if err != nil {
		return // getSessionPhase already wrote the error response
	}

	if phase != "running" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Session is not in a running state"})
		return
	}

	var req types.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	runID := uuid.New().String()
	messageID := uuid.New().String()

	messages := []map[string]interface{}{
		{
			"id":      messageID,
			"role":    "user",
			"content": req.Content,
		},
	}

	messagesJSON, err := json.Marshal(messages)
	if err != nil {
		log.Printf("Failed to marshal messages: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	backendReq := map[string]interface{}{
		"threadId": sessionID,
		"runId":    runID,
		"messages": json.RawMessage(messagesJSON),
	}

	reqBody, err := json.Marshal(backendReq)
	if err != nil {
		log.Printf("Failed to marshal request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s/agui/run", project, sessionID)
	resp, cancel, err := ProxyRequest(c, http.MethodPost, path, reqBody)
	if err != nil {
		log.Printf("Backend request failed for send message: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer cancel()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		forwardErrorResponse(c, resp.StatusCode, body)
		return
	}

	var backendResp map[string]interface{}
	if err := json.Unmarshal(body, &backendResp); err != nil {
		log.Printf("Failed to parse backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	respRunID, _ := backendResp["runId"].(string)
	respThreadID, _ := backendResp["threadId"].(string)

	c.JSON(http.StatusAccepted, types.SendMessageResponse{
		RunID:    respRunID,
		ThreadID: respThreadID,
	})
}

// GetSessionRuns handles GET /v1/sessions/:id/runs
func GetSessionRuns(c *gin.Context) {
	project := GetProject(c)
	if !ValidateProjectName(project) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
		return
	}
	sessionID := c.Param("id")
	if !ValidateSessionID(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	events, statusCode, err := fetchSessionEvents(c, project, sessionID)
	if err != nil {
		if statusCode > 0 {
			c.JSON(statusCode, gin.H{"error": "Request failed"})
		} else {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		}
		return
	}

	runs := deriveRunSummaries(events)

	c.JSON(http.StatusOK, types.SessionRunsResponse{
		SessionID: sessionID,
		Runs:      runs,
	})
}

// fetchSessionEvents retrieves AG-UI events from the backend export endpoint.
// Returns the events array, an HTTP status code for errors, and any error.
func fetchSessionEvents(c *gin.Context, project, sessionID string) ([]map[string]interface{}, int, error) {
	path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s/export", project, sessionID)

	resp, cancel, err := ProxyRequest(c, http.MethodGet, path, nil)
	if err != nil {
		log.Printf("Backend request failed for export: %v", err)
		return nil, 0, fmt.Errorf("backend unavailable")
	}
	defer cancel()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read export response: %v", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("internal server error")
	}

	if resp.StatusCode != http.StatusOK {
		// Try to extract error message from backend response
		var errorResp map[string]interface{}
		if jsonErr := json.Unmarshal(body, &errorResp); jsonErr == nil {
			if errMsg, ok := errorResp["error"].(string); ok {
				return nil, resp.StatusCode, fmt.Errorf("%s", errMsg)
			}
		}
		return nil, resp.StatusCode, fmt.Errorf("request failed")
	}

	var exportResp struct {
		AGUIEvents json.RawMessage `json:"aguiEvents"`
	}
	if err := json.Unmarshal(body, &exportResp); err != nil {
		log.Printf("Failed to parse export response: %v", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("internal server error")
	}

	var events []map[string]interface{}
	if err := json.Unmarshal(exportResp.AGUIEvents, &events); err != nil {
		log.Printf("Failed to parse aguiEvents: %v", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("internal server error")
	}

	return events, 0, nil
}

// deriveRunSummaries groups events by runId and builds run summaries.
func deriveRunSummaries(events []map[string]interface{}) []types.RunSummary {
	type runData struct {
		summary types.RunSummary
		order   int
	}

	runMap := make(map[string]*runData)
	orderCounter := 0

	for _, event := range events {
		runID, _ := event["runId"].(string)
		if runID == "" {
			continue
		}

		rd, exists := runMap[runID]
		if !exists {
			rd = &runData{
				summary: types.RunSummary{
					RunID:  runID,
					Status: "running",
				},
				order: orderCounter,
			}
			orderCounter++
			runMap[runID] = rd
		}

		rd.summary.EventCount++

		eventType, _ := event["type"].(string)
		timestamp := toInt64(event["timestamp"])

		switch eventType {
		case "RUN_STARTED":
			if timestamp > 0 {
				rd.summary.StartedAt = timestamp
			}
		case "RUN_FINISHED":
			rd.summary.Status = "completed"
			if timestamp > 0 {
				rd.summary.FinishedAt = timestamp
			}
		case "RUN_ERROR":
			rd.summary.Status = "error"
			if timestamp > 0 {
				rd.summary.FinishedAt = timestamp
			}
		case "TEXT_MESSAGE_START":
			role, _ := event["role"].(string)
			content, _ := event["content"].(string)
			if role == "user" && content != "" && rd.summary.UserMessage == "" {
				rd.summary.UserMessage = content
			}
		}
	}

	// Build sorted slice
	sorted := make([]*runData, 0, len(runMap))
	for _, rd := range runMap {
		sorted = append(sorted, rd)
	}
	slices.SortFunc(sorted, func(a, b *runData) int {
		return a.order - b.order
	})
	runs := make([]types.RunSummary, 0, len(sorted))
	for _, rd := range sorted {
		runs = append(runs, rd.summary)
	}

	return runs
}

// toInt64 converts a JSON number (float64) to int64.
func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case json.Number:
		i, _ := n.Int64()
		return i
	default:
		return 0
	}
}
