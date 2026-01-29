// Package websocket provides AG-UI protocol endpoints for event streaming.
package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ambient-code-backend/handlers"

	"github.com/gin-gonic/gin"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExportResponse contains the exported session data
type ExportResponse struct {
	SessionID      string          `json:"sessionId"`
	ProjectName    string          `json:"projectName"`
	ExportDate     string          `json:"exportDate"`
	AGUIEvents     json.RawMessage `json:"aguiEvents"`
	LegacyMessages json.RawMessage `json:"legacyMessages,omitempty"`
	HasLegacy      bool            `json:"hasLegacy"`
}

// HandleExportSession exports session chat data as JSON
// GET /api/projects/:projectName/agentic-sessions/:sessionName/export
func HandleExportSession(c *gin.Context) {
	projectName := c.Param("projectName")
	sessionName := c.Param("sessionName")

	log.Printf("Export: Exporting session %s/%s", projectName, sessionName)

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
		log.Printf("Export: User not authorized to read session %s/%s", projectName, sessionName)
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	// SECURITY: Validate sessionName to prevent path traversal
	if !isValidSessionName(sessionName) {
		log.Printf("Export: Invalid session name detected: %s", sessionName)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session name"})
		return
	}

	// Build paths safely using filepath.Join and validate they're within StateBaseDir
	baseDir := filepath.Clean(StateBaseDir)
	sessionDir := filepath.Join(baseDir, "sessions", sessionName)
	sessionDir = filepath.Clean(sessionDir)

	// SECURITY: Ensure path is within allowed directory (prevent path traversal)
	if !strings.HasPrefix(sessionDir, baseDir) {
		log.Printf("Export: Security - path traversal attempt detected: %s", sessionName)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session name"})
		return
	}

	aguiEventsPath := filepath.Join(sessionDir, "agui-events.jsonl")
	legacyMigratedPath := filepath.Join(sessionDir, "messages.jsonl.migrated")
	legacyOriginalPath := filepath.Join(sessionDir, "messages.jsonl")

	// Check if session directory exists
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		log.Printf("Export: Session directory not found: %s", sessionDir)
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	response := ExportResponse{
		SessionID:   sessionName,
		ProjectName: projectName,
		ExportDate:  time.Now().UTC().Format(time.RFC3339),
		HasLegacy:   false,
	}

	// Read AG-UI events
	aguiData, err := readJSONLFile(aguiEventsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No AG-UI events yet - return empty array
			response.AGUIEvents = json.RawMessage("[]")
		} else {
			log.Printf("Export: Error reading AG-UI events: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read session events"})
			return
		}
	} else {
		// Pretty-print the events array
		prettyJSON, err := json.MarshalIndent(aguiData, "", "  ")
		if err != nil {
			log.Printf("Export: Error formatting AG-UI events: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to format events"})
			return
		}
		response.AGUIEvents = prettyJSON
	}

	// Check for legacy messages - try migrated file first, then original
	legacyPath := ""
	if _, err := os.Stat(legacyMigratedPath); err == nil {
		legacyPath = legacyMigratedPath
		log.Printf("Export: Found migrated legacy file: %s", legacyMigratedPath)
	} else if _, err := os.Stat(legacyOriginalPath); err == nil {
		legacyPath = legacyOriginalPath
		log.Printf("Export: Found original legacy file: %s", legacyOriginalPath)
	}

	if legacyPath != "" {
		legacyData, err := readJSONLFile(legacyPath)
		if err != nil {
			log.Printf("Export: Warning - failed to read legacy messages: %v", err)
		} else {
			prettyJSON, err := json.MarshalIndent(legacyData, "", "  ")
			if err != nil {
				log.Printf("Export: Warning - failed to format legacy messages: %v", err)
			} else {
				response.LegacyMessages = prettyJSON
				response.HasLegacy = true
			}
		}
	}

	log.Printf("Export: Successfully exported session %s (hasLegacy=%v)", sessionName, response.HasLegacy)

	// Set headers for JSON download
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-export.json\"", sessionName))

	c.JSON(http.StatusOK, response)
}

// isValidSessionName validates that the session name is a valid Kubernetes resource name
// and doesn't contain path traversal characters
func isValidSessionName(name string) bool {
	// Must not be empty
	if name == "" {
		return false
	}

	// Must not contain path traversal characters
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return false
	}

	// Kubernetes DNS label format: lowercase alphanumeric, hyphens allowed (not at start/end)
	// Max 63 characters
	if len(name) > 63 {
		return false
	}

	// Check each character
	for i, ch := range name {
		isLower := ch >= 'a' && ch <= 'z'
		isDigit := ch >= '0' && ch <= '9'
		isHyphen := ch == '-'

		if !isLower && !isDigit && !isHyphen {
			return false
		}

		// Hyphen not allowed at start or end
		if isHyphen && (i == 0 || i == len(name)-1) {
			return false
		}
	}

	return true
}

// readJSONLFile reads a JSONL file and returns parsed array of objects
func readJSONLFile(path string) ([]map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var events []map[string]interface{}
	lines := splitLines(data)

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var event map[string]interface{}
		if err := json.Unmarshal(line, &event); err != nil {
			// Skip malformed lines
			log.Printf("Export: Skipping malformed JSON line: %v", err)
			continue
		}
		events = append(events, event)
	}

	return events, nil
}
