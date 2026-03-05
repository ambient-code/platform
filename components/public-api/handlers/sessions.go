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

	// Forward query parameters (e.g., labelSelector) verbatim to the backend.
	// The backend enforces its own validation and RBAC via the user's token.
	if rawQuery := c.Request.URL.RawQuery; rawQuery != "" {
		path = path + "?" + rawQuery
	}

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
		"initialPrompt": req.Task,
	}
	if req.Model != "" {
		backendReq["llmSettings"] = map[string]interface{}{
			"model": req.Model,
		}
	}
	if req.DisplayName != "" {
		backendReq["displayName"] = req.DisplayName
	}
	if req.Timeout != nil {
		backendReq["timeout"] = *req.Timeout
	}
	if len(req.Labels) > 0 {
		keys := make([]string, 0, len(req.Labels))
		for k := range req.Labels {
			keys = append(keys, k)
		}
		if key, prefix, ok := validateLabelKeys(keys); !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("label key %q uses reserved prefix %q", key, prefix)})
			return
		}
		// Send as annotations (not labels) so the write/read paths are symmetric.
		// The read path extracts user labels from metadata.annotations.
		backendReq["annotations"] = req.Labels
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
		if annotationsRaw, ok := metadata["annotations"].(map[string]interface{}); ok {
			labels := filterUserLabels(annotationsRaw)
			if len(labels) > 0 {
				session.Labels = labels
			}
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
		}
		if prompt, ok := spec["prompt"].(string); ok && session.Task == "" {
			session.Task = prompt
		}
		if model, ok := spec["model"].(string); ok {
			session.Model = model
		}
		if llm, ok := spec["llmSettings"].(map[string]interface{}); ok {
			if model, ok := llm["model"].(string); ok && session.Model == "" {
				session.Model = model
			}
		}
		if displayName, ok := spec["displayName"].(string); ok {
			session.DisplayName = displayName
		}
		if timeout, ok := spec["timeout"].(float64); ok {
			session.Timeout = int(timeout)
		}
		if reposRaw, ok := spec["repos"].([]interface{}); ok {
			session.Repos = extractRepos(reposRaw)
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

// internalLabelPrefixes are K8s/system label prefixes that should not be exposed to users
var internalLabelPrefixes = []string{
	"app.kubernetes.io/",
	"kubectl.kubernetes.io/",
	"meta.kubernetes.io/",
	"vteam.ambient-code/",
	"ambient-code.io/",
}

// validateLabelKeys checks that no label keys use reserved internal prefixes
func validateLabelKeys(keys []string) (string, string, bool) {
	for _, k := range keys {
		for _, prefix := range internalLabelPrefixes {
			if strings.HasPrefix(k, prefix) {
				return k, prefix, false
			}
		}
	}
	return "", "", true
}

// filterUserLabels returns only user-defined labels, stripping internal/system labels
func filterUserLabels(labelsRaw map[string]interface{}) map[string]string {
	labels := make(map[string]string)
	for k, v := range labelsRaw {
		internal := false
		for _, prefix := range internalLabelPrefixes {
			if strings.HasPrefix(k, prefix) {
				internal = true
				break
			}
		}
		if !internal {
			if s, ok := v.(string); ok {
				labels[k] = s
			}
		}
	}
	return labels
}

// extractRepos converts the repos array from the backend response to typed Repo objects
func extractRepos(reposRaw []interface{}) []types.Repo {
	repos := make([]types.Repo, 0, len(reposRaw))
	for _, r := range reposRaw {
		repoMap, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		repo := types.Repo{}

		// Handle both flat format (url/branch at top level) and nested format (input.url/input.branch)
		if url, ok := repoMap["url"].(string); ok {
			repo.URL = url
		}
		if branch, ok := repoMap["branch"].(string); ok {
			repo.Branch = branch
		}
		if input, ok := repoMap["input"].(map[string]interface{}); ok {
			if url, ok := input["url"].(string); ok {
				repo.URL = url
			}
			if branch, ok := input["branch"].(string); ok {
				repo.Branch = branch
			}
		}
		if repo.URL != "" {
			repos = append(repos, repo)
		}
	}
	return repos
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
