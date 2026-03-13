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

// StartSession handles POST /v1/sessions/:id/start
//
// Defense-in-depth: The gateway fetches the session phase before forwarding.
// The backend also validates phase transitions, so this is a redundant guard
// that provides faster feedback and reduces unnecessary backend writes.
func StartSession(c *gin.Context) {
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

	if phase == "" {
		log.Printf("Session %s has no phase, treating as unknown", sessionID)
		c.JSON(http.StatusConflict, gin.H{"error": "Session state is unknown"})
		return
	}

	if phase == "running" || phase == "pending" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Session is already running or pending"})
		return
	}

	path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s/start", project, sessionID)
	resp, cancel, err := ProxyRequest(c, http.MethodPost, path, nil)
	if err != nil {
		log.Printf("Backend request failed for start session %s: %v", sessionID, err)
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

	c.JSON(http.StatusAccepted, transformSession(backendResp))
}

// StopSession handles POST /v1/sessions/:id/stop
//
// Defense-in-depth: The gateway fetches the session phase before forwarding.
// The backend also validates phase transitions, so this is a redundant guard
// that provides faster feedback and reduces unnecessary backend writes.
func StopSession(c *gin.Context) {
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
		return
	}

	if phase == "" {
		log.Printf("Session %s has no phase, treating as unknown", sessionID)
		c.JSON(http.StatusConflict, gin.H{"error": "Session state is unknown"})
		return
	}

	if phase == "completed" || phase == "failed" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Session is not in a running state"})
		return
	}

	path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s/stop", project, sessionID)
	resp, cancel, err := ProxyRequest(c, http.MethodPost, path, nil)
	if err != nil {
		log.Printf("Backend request failed for stop session %s: %v", sessionID, err)
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

	c.JSON(http.StatusAccepted, transformSession(backendResp))
}

// InterruptSession handles POST /v1/sessions/:id/interrupt
func InterruptSession(c *gin.Context) {
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

	path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s/agui/interrupt", project, sessionID)
	resp, cancel, err := ProxyRequest(c, http.MethodPost, path, []byte("{}"))
	if err != nil {
		log.Printf("Backend request failed for interrupt session %s: %v", sessionID, err)
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

	c.JSON(http.StatusOK, types.MessageResponse{Message: "Interrupt signal sent"})
}

// getSessionPhase fetches the session from the backend and returns its normalized phase.
// On error, it writes the appropriate error response to the gin context.
func getSessionPhase(c *gin.Context, project, sessionID string) (string, error) {
	path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s", project, sessionID)
	resp, cancel, err := ProxyRequest(c, http.MethodGet, path, nil)
	if err != nil {
		log.Printf("Backend request failed for get session phase %s: %v", sessionID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return "", fmt.Errorf("backend unavailable")
	}
	defer cancel()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return "", fmt.Errorf("internal server error")
	}

	if resp.StatusCode != http.StatusOK {
		forwardErrorResponse(c, resp.StatusCode, body)
		return "", fmt.Errorf("backend returned %d", resp.StatusCode)
	}

	var backendResp map[string]interface{}
	if err := json.Unmarshal(body, &backendResp); err != nil {
		log.Printf("Failed to parse backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return "", fmt.Errorf("internal server error")
	}

	phase := ""
	if status, ok := backendResp["status"].(map[string]interface{}); ok {
		if p, ok := status["phase"].(string); ok {
			phase = normalizePhase(p)
		}
	}

	return phase, nil
}
