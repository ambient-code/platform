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

// PatchSession handles PATCH /v1/sessions/:id
// It inspects the request body and routes to the correct backend endpoint:
//   - stopped: false → POST /start (resume session)
//   - stopped: true  → POST /stop  (stop session)
//   - displayName/timeout → PUT (update session spec)
//   - labels/removeLabels → PATCH (update annotations)
func PatchSession(c *gin.Context) {
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

	var req types.PatchSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Classify the request into categories
	hasStopped := req.Stopped != nil
	hasUpdate := req.DisplayName != nil || req.Timeout != nil
	hasLabels := len(req.Labels) > 0 || len(req.RemoveLabels) > 0

	// Count how many categories are present
	categories := 0
	if hasStopped {
		categories++
	}
	if hasUpdate {
		categories++
	}
	if hasLabels {
		categories++
	}

	if categories == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request body must contain at least one of: stopped, displayName, timeout, labels, removeLabels"})
		return
	}
	if categories > 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot mix stopped, spec updates (displayName/timeout), and label changes in the same request"})
		return
	}

	basePath := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s", project, sessionID)

	switch {
	case hasStopped:
		patchSessionStartStop(c, basePath, *req.Stopped)
	case hasUpdate:
		patchSessionUpdate(c, basePath, req)
	case hasLabels:
		patchSessionLabels(c, basePath, req)
	}
}

// patchSessionStartStop routes to the backend start or stop endpoint
func patchSessionStartStop(c *gin.Context, basePath string, stopped bool) {
	var path string
	if stopped {
		path = basePath + "/stop"
	} else {
		path = basePath + "/start"
	}

	resp, err := ProxyRequest(c, http.MethodPost, path, nil)
	if err != nil {
		log.Printf("Backend request failed for start/stop: %v", err)
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		forwardErrorResponse(c, resp.StatusCode, body)
		return
	}

	// Parse and transform the response
	var backendResp map[string]interface{}
	if err := json.Unmarshal(body, &backendResp); err != nil {
		log.Printf("Failed to parse backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, transformSession(backendResp))
}

// patchSessionUpdate routes to the backend PUT (UpdateSession) endpoint
func patchSessionUpdate(c *gin.Context, basePath string, req types.PatchSessionRequest) {
	// Transform to backend UpdateAgenticSessionRequest format
	backendReq := map[string]interface{}{}
	if req.DisplayName != nil {
		backendReq["displayName"] = *req.DisplayName
	}
	if req.Timeout != nil {
		backendReq["timeout"] = *req.Timeout
	}

	reqBody, err := json.Marshal(backendReq)
	if err != nil {
		log.Printf("Failed to marshal update request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	resp, err := ProxyRequest(c, http.MethodPut, basePath, reqBody)
	if err != nil {
		log.Printf("Backend request failed for update: %v", err)
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

// patchSessionLabels routes to the backend PATCH endpoint for annotation changes
func patchSessionLabels(c *gin.Context, basePath string, req types.PatchSessionRequest) {
	// Validate label keys don't use reserved prefixes
	allKeys := make([]string, 0, len(req.Labels)+len(req.RemoveLabels))
	for k := range req.Labels {
		allKeys = append(allKeys, k)
	}
	allKeys = append(allKeys, req.RemoveLabels...)
	if key, prefix, ok := validateLabelKeys(allKeys); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("label key %q uses reserved prefix %q", key, prefix)})
		return
	}

	// Transform labels to backend annotation format:
	// {"metadata": {"annotations": {"key": "value"}}} for adds
	// {"metadata": {"annotations": {"key": null}}} for removes
	annotations := map[string]interface{}{}

	for k, v := range req.Labels {
		annotations[k] = v
	}
	for _, k := range req.RemoveLabels {
		annotations[k] = nil
	}

	backendReq := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": annotations,
		},
	}

	reqBody, err := json.Marshal(backendReq)
	if err != nil {
		log.Printf("Failed to marshal label patch request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	resp, err := ProxyRequest(c, http.MethodPatch, basePath, reqBody)
	if err != nil {
		log.Printf("Backend request failed for label patch: %v", err)
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

	// Follow-up GET to return full session DTO (consistent with other PATCH responses)
	getResp, err := ProxyRequest(c, http.MethodGet, basePath, nil)
	if err != nil {
		// PATCH succeeded but GET failed — return success with minimal info
		log.Printf("Label patch succeeded but follow-up GET failed: %v", err)
		c.JSON(http.StatusOK, gin.H{"message": "Labels updated"})
		return
	}
	defer getResp.Body.Close()

	getBody, err := io.ReadAll(getResp.Body)
	if err != nil || getResp.StatusCode != http.StatusOK {
		log.Printf("Label patch succeeded but follow-up GET returned unexpected result")
		c.JSON(http.StatusOK, gin.H{"message": "Labels updated"})
		return
	}

	var backendResp map[string]interface{}
	if err := json.Unmarshal(getBody, &backendResp); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Labels updated"})
		return
	}

	c.JSON(http.StatusOK, transformSession(backendResp))
}
