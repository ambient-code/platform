package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

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

	resp, cancel, err := ProxyRequest(c, http.MethodGet, path, nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer cancel()
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

	resp, cancel, err := ProxyRequest(c, http.MethodGet, path, nil)
	if err != nil {
		log.Printf("Backend request failed for session %s: %v", sessionID, err)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Transform to backend format
	backendReq := map[string]interface{}{
		"initialPrompt": req.Task,
	}
	if req.DisplayName != "" {
		backendReq["displayName"] = req.DisplayName
	}
	if req.Model != "" {
		backendReq["model"] = req.Model
	}
	if len(req.Repos) > 0 {
		repos := make([]map[string]interface{}, len(req.Repos))
		for i, r := range req.Repos {
			repo := map[string]interface{}{
				"url": r.URL,
			}
			if r.Branch != "" {
				repo["branch"] = r.Branch
			}
			repos[i] = repo
		}
		backendReq["repos"] = repos
	}
	if req.ActiveWorkflow != nil && strings.TrimSpace(req.ActiveWorkflow.GitURL) != "" {
		wf := map[string]interface{}{
			"gitUrl": req.ActiveWorkflow.GitURL,
		}
		if req.ActiveWorkflow.Branch != "" {
			wf["branch"] = req.ActiveWorkflow.Branch
		}
		if req.ActiveWorkflow.Path != "" {
			wf["path"] = req.ActiveWorkflow.Path
		}
		backendReq["activeWorkflow"] = wf
	}
	if len(req.EnvironmentVariables) > 0 {
		backendReq["environmentVariables"] = req.EnvironmentVariables
	}

	reqBody, err := json.Marshal(backendReq)
	if err != nil {
		log.Printf("Failed to marshal request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	path := fmt.Sprintf("/api/projects/%s/agentic-sessions", project)

	resp, cancel, err := ProxyRequest(c, http.MethodPost, path, reqBody)
	if err != nil {
		log.Printf("Backend request failed for create session: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer cancel()
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

	resp, cancel, err := ProxyRequest(c, http.MethodDelete, path, nil)
	if err != nil {
		log.Printf("Backend request failed for delete session %s: %v", sessionID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer cancel()
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

// forwardErrorResponse forwards backend error with consistent JSON format.
// SECURITY: Only forwards the "error" field to prevent leaking internal details.
func forwardErrorResponse(c *gin.Context, statusCode int, body []byte) {
	// Try to parse as JSON and extract only the "error" field
	var errorResp map[string]interface{}
	if err := json.Unmarshal(body, &errorResp); err == nil {
		if errMsg, ok := errorResp["error"].(string); ok {
			c.JSON(statusCode, gin.H{"error": errMsg})
			return
		}
	}

	// Backend returned non-JSON or no "error" field, wrap in standard error format
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
		if prompt, ok := spec["initialPrompt"].(string); ok {
			session.Task = prompt
		} else if prompt, ok := spec["prompt"].(string); ok {
			session.Task = prompt
		}
		if model, ok := spec["model"].(string); ok {
			session.Model = model
		}
		if displayName, ok := spec["displayName"].(string); ok {
			session.DisplayName = displayName
		}
		if repos, ok := spec["repos"].([]interface{}); ok {
			for _, r := range repos {
				repo, ok := r.(map[string]interface{})
				if !ok {
					continue
				}
				sr := types.SessionRepo{}
				if input, ok := repo["input"].(map[string]interface{}); ok {
					if url, ok := input["url"].(string); ok {
						sr.URL = url
					}
					if branch, ok := input["branch"].(string); ok {
						sr.Branch = branch
					}
				}
				if sr.URL != "" {
					session.Repos = append(session.Repos, sr)
				}
			}
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

// normalizePhase converts K8s phase to simplified lowercase status.
// The public API contract guarantees status values are always lowercase.
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
		return strings.ToLower(phase)
	}
}
