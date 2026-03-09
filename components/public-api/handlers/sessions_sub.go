package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetSessionTranscript handles GET /v1/sessions/:id/transcript
// Proxies to backend GET /api/projects/{p}/agentic-sessions/{s}/export
func GetSessionTranscript(c *gin.Context) {
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

	path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s/export", project, sessionID)

	// Forward query params (e.g., format) verbatim to the backend.
	// The backend enforces its own validation and RBAC via the user's token.
	if rawQuery := c.Request.URL.RawQuery; rawQuery != "" {
		path = path + "?" + rawQuery
	}

	resp, err := ProxyRequest(c, http.MethodGet, path, nil)
	if err != nil {
		log.Printf("Backend request failed for transcript %s: %v", sessionID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer resp.Body.Close()

	// For non-OK responses, buffer to forward the error body
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read backend error response: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}
		forwardErrorResponse(c, resp.StatusCode, body)
		return
	}

	// Stream the transcript response directly to the client
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Header("Content-Type", contentType)
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		log.Printf("GetSessionTranscript: error streaming backend response for %s: %v", sessionID, err)
	}
}

// GetSessionLogs handles GET /v1/sessions/:id/logs
// Proxies to backend GET /api/projects/{p}/agentic-sessions/{s}/logs
func GetSessionLogs(c *gin.Context) {
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

	path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s/logs", project, sessionID)

	// Forward query params (tailLines, container) verbatim to the backend.
	// The backend enforces its own validation and RBAC via the user's token.
	if rawQuery := c.Request.URL.RawQuery; rawQuery != "" {
		path = path + "?" + rawQuery
	}

	resp, err := ProxyRequest(c, http.MethodGet, path, nil)
	if err != nil {
		log.Printf("Backend request failed for logs %s: %v", sessionID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer resp.Body.Close()

	// For non-OK responses, buffer to forward the error body
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read backend error response: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}
		forwardErrorResponse(c, resp.StatusCode, body)
		return
	}

	// Stream the log response directly to the client to avoid buffering up to
	// 10 MB (the backend's LimitReader cap) per concurrent request.
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		log.Printf("GetSessionLogs: error streaming backend response for %s: %v", sessionID, err)
	}
}

// GetSessionMetrics handles GET /v1/sessions/:id/metrics
// Proxies to backend GET /api/projects/{p}/agentic-sessions/{s}/metrics
func GetSessionMetrics(c *gin.Context) {
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

	path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s/metrics", project, sessionID)

	resp, err := ProxyRequest(c, http.MethodGet, path, nil)
	if err != nil {
		log.Printf("Backend request failed for metrics %s: %v", sessionID, err)
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

	c.Data(http.StatusOK, "application/json", body)
}
