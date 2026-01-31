package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"ambient-code-public-api/types"

	"github.com/gin-gonic/gin"
)

// ListSessions handles GET /v1/sessions
func ListSessions(c *gin.Context) {
	project := GetProject(c)
	if !ValidateProjectName(project) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
		return
	}
	path := fmt.Sprintf("/api/projects/%s/agentic-sessions", project)

	resp, err := ProxyRequest(c, http.MethodGet, path, nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Forward non-OK responses with consistent error format
	if resp.StatusCode != http.StatusOK {
		forwardErrorResponse(c, resp.StatusCode, body)
		return
	}

	var backendResp struct {
		Items []map[string]interface{} `json:"items"`
	}
	if err := json.Unmarshal(body, &backendResp); err != nil {
		log.Printf("Failed to parse backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Transform to simplified DTOs
	sessions := make([]types.SessionResponse, 0, len(backendResp.Items))
	for _, item := range backendResp.Items {
		sessions = append(sessions, transformSession(item))
	}

	c.JSON(http.StatusOK, types.SessionListResponse{
		Items: sessions,
		Total: len(sessions),
	})
}

// GetSession handles GET /v1/sessions/:id
func GetSession(c *gin.Context) {
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
	path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s", project, sessionID)

	resp, err := ProxyRequest(c, http.MethodGet, path, nil)
	if err != nil {
		log.Printf("Backend request failed for session %s: %v", sessionID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		forwardErrorResponse(c, resp.StatusCode, body)
		return
	}

	var backendResp map[string]interface{}
	if err := json.Unmarshal(body, &backendResp); err != nil {
		log.Printf("Failed to parse backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, transformSession(backendResp))
}

// CreateSession handles POST /v1/sessions
func CreateSession(c *gin.Context) {
	project := GetProject(c)
	if !ValidateProjectName(project) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
		return
	}

	var req types.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Transform to backend format
	backendReq := map[string]interface{}{
		"prompt": req.Task,
	}
	if req.Model != "" {
		backendReq["model"] = req.Model
	}
	if len(req.Repos) > 0 {
		repos := make([]map[string]interface{}, len(req.Repos))
		for i, r := range req.Repos {
			repos[i] = map[string]interface{}{
				"input": map[string]interface{}{
					"url":    r.URL,
					"branch": r.Branch,
				},
			}
		}
		backendReq["repos"] = repos
	}

	reqBody, err := json.Marshal(backendReq)
	if err != nil {
		log.Printf("Failed to marshal request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	path := fmt.Sprintf("/api/projects/%s/agentic-sessions", project)

	resp, err := ProxyRequest(c, http.MethodPost, path, reqBody)
	if err != nil {
		log.Printf("Backend request failed for create session: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		forwardErrorResponse(c, resp.StatusCode, respBody)
		return
	}

	// Parse response to get session ID
	var backendResp map[string]interface{}
	if err := json.Unmarshal(respBody, &backendResp); err != nil {
		log.Printf("Failed to parse backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	// Return simplified response
	c.JSON(http.StatusCreated, gin.H{
		"id":      backendResp["name"],
		"message": "Session created",
	})
}

// DeleteSession handles DELETE /v1/sessions/:id
func DeleteSession(c *gin.Context) {
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
	path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s", project, sessionID)

	resp, err := ProxyRequest(c, http.MethodDelete, path, nil)
	if err != nil {
		log.Printf("Backend request failed for delete session %s: %v", sessionID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		c.Status(http.StatusNoContent)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	forwardErrorResponse(c, resp.StatusCode, body)
}

// forwardErrorResponse forwards backend error with consistent JSON format
func forwardErrorResponse(c *gin.Context, statusCode int, body []byte) {
	// Try to parse as JSON error response
	var errorResp map[string]interface{}
	if err := json.Unmarshal(body, &errorResp); err == nil {
		// Backend returned valid JSON, forward it
		c.JSON(statusCode, errorResp)
		return
	}

	// Backend returned non-JSON, wrap in standard error format
	c.JSON(statusCode, gin.H{"error": "Request failed"})
}

// transformSession converts backend session format to simplified DTO
func transformSession(data map[string]interface{}) types.SessionResponse {
	session := types.SessionResponse{}

	// Extract metadata
	if metadata, ok := data["metadata"].(map[string]interface{}); ok {
		if name, ok := metadata["name"].(string); ok {
			session.ID = name
		}
		if creationTimestamp, ok := metadata["creationTimestamp"].(string); ok {
			session.CreatedAt = creationTimestamp
		}
	}

	// If no metadata, try top-level name (list response format)
	if session.ID == "" {
		if name, ok := data["name"].(string); ok {
			session.ID = name
		}
	}

	// Extract spec
	if spec, ok := data["spec"].(map[string]interface{}); ok {
		if prompt, ok := spec["prompt"].(string); ok {
			session.Task = prompt
		}
		if model, ok := spec["model"].(string); ok {
			session.Model = model
		}
	}

	// Extract status
	if status, ok := data["status"].(map[string]interface{}); ok {
		if phase, ok := status["phase"].(string); ok {
			session.Status = normalizePhase(phase)
		}
		if completionTime, ok := status["completionTime"].(string); ok {
			session.CompletedAt = completionTime
		}
		if result, ok := status["result"].(string); ok {
			session.Result = result
		}
		if errMsg, ok := status["error"].(string); ok {
			session.Error = errMsg
		}
	}

	// Default status if not set
	if session.Status == "" {
		session.Status = "pending"
	}

	return session
}

// normalizePhase converts K8s phase to simplified status
func normalizePhase(phase string) string {
	switch phase {
	case "Pending", "Creating", "Initializing":
		return "pending"
	case "Running", "Active":
		return "running"
	case "Completed", "Succeeded":
		return "completed"
	case "Failed", "Error":
		return "failed"
	default:
		return phase
	}
}
