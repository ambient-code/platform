package webhook

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel string

const (
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
	LevelDebug LogLevel = "DEBUG"
)

// WebhookLogger provides structured logging for webhook events (FR-022)
// All logs are written to stdout in JSON format for easy ingestion by log aggregators
type WebhookLogger struct {
	logger *log.Logger
}

// NewWebhookLogger creates a new webhook logger that writes to stdout
func NewWebhookLogger() *WebhookLogger {
	return &WebhookLogger{
		logger: log.New(os.Stdout, "", 0), // No prefix, we'll structure everything in JSON
	}
}

// LogWebhookReceived logs when a webhook is received
func (wl *WebhookLogger) LogWebhookReceived(deliveryID, eventType string, payloadSize int) {
	wl.log(LevelInfo, "webhook_received", map[string]interface{}{
		"delivery_id":  deliveryID,
		"event_type":   eventType,
		"payload_size": payloadSize,
	})
}

// LogSignatureVerified logs successful signature verification
func (wl *WebhookLogger) LogSignatureVerified(deliveryID, eventType string) {
	wl.log(LevelInfo, "signature_verified", map[string]interface{}{
		"delivery_id": deliveryID,
		"event_type":  eventType,
	})
}

// LogSignatureInvalid logs failed signature verification
func (wl *WebhookLogger) LogSignatureInvalid(deliveryID, eventType, reason string) {
	wl.log(LevelWarn, "signature_invalid", map[string]interface{}{
		"delivery_id": deliveryID,
		"event_type":  eventType,
		"reason":      reason,
	})
}

// LogAuthorizationChecked logs installation authorization check
func (wl *WebhookLogger) LogAuthorizationChecked(deliveryID, repository string, authorized bool, installationID int64) {
	wl.log(LevelInfo, "authorization_checked", map[string]interface{}{
		"delivery_id":     deliveryID,
		"repository":      repository,
		"authorized":      authorized,
		"installation_id": installationID,
	})
}

// LogKeywordDetected logs when a keyword is detected in a comment
func (wl *WebhookLogger) LogKeywordDetected(deliveryID, keyword, command string) {
	wl.log(LevelInfo, "keyword_detected", map[string]interface{}{
		"delivery_id": deliveryID,
		"keyword":     keyword,
		"command":     command,
	})
}

// LogDuplicateDetected logs when a duplicate delivery ID is detected
func (wl *WebhookLogger) LogDuplicateDetected(deliveryID, eventType string) {
	wl.log(LevelInfo, "duplicate_detected", map[string]interface{}{
		"delivery_id": deliveryID,
		"event_type":  eventType,
	})
}

// LogSessionCreated logs successful session creation
func (wl *WebhookLogger) LogSessionCreated(deliveryID, sessionID, eventType, repository, githubURL string) {
	wl.log(LevelInfo, "session_created", map[string]interface{}{
		"delivery_id": deliveryID,
		"session_id":  sessionID,
		"event_type":  eventType,
		"repository":  repository,
		"github_url":  githubURL,
	})
}

// LogSessionCreationFailed logs failed session creation
func (wl *WebhookLogger) LogSessionCreationFailed(deliveryID, eventType, reason, errorMsg string) {
	wl.log(LevelError, "session_creation_failed", map[string]interface{}{
		"delivery_id": deliveryID,
		"event_type":  eventType,
		"reason":      reason,
		"error":       errorMsg,
	})
}

// LogGitHubCommentPosted logs when a confirmation/error comment is posted to GitHub
func (wl *WebhookLogger) LogGitHubCommentPosted(deliveryID, commentType, githubURL string, success bool) {
	wl.log(LevelInfo, "github_comment_posted", map[string]interface{}{
		"delivery_id":  deliveryID,
		"comment_type": commentType, // confirmation, error
		"github_url":   githubURL,
		"success":      success,
	})
}

// LogWebhookProcessed logs the completion of webhook processing with full context
func (wl *WebhookLogger) LogWebhookProcessed(deliveryID, eventType, repository, githubUser string, status string, durationMs int64) {
	wl.log(LevelInfo, "webhook_processed", map[string]interface{}{
		"delivery_id":   deliveryID,
		"event_type":    eventType,
		"repository":    repository,
		"github_user":   githubUser,
		"status":        status, // success, rejected, failed
		"duration_ms":   durationMs,
	})
}

// LogError logs a general error with context
func (wl *WebhookLogger) LogError(deliveryID, component, message string, err error) {
	fields := map[string]interface{}{
		"delivery_id": deliveryID,
		"component":   component,
		"message":     message,
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	wl.log(LevelError, "error", fields)
}

// LogDebug logs debug information (only if DEBUG env var is set)
func (wl *WebhookLogger) LogDebug(deliveryID, message string, fields map[string]interface{}) {
	if os.Getenv("DEBUG") != "true" {
		return // Skip debug logs unless explicitly enabled
	}

	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["delivery_id"] = deliveryID
	fields["message"] = message

	wl.log(LevelDebug, "debug", fields)
}

// log is the internal method that formats and writes structured logs to stdout
func (wl *WebhookLogger) log(level LogLevel, event string, fields map[string]interface{}) {
	// Create the log entry structure
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     string(level),
		"event":     event,
		"component": "webhook",
	}

	// Merge in the additional fields
	for k, v := range fields {
		entry[k] = v
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple log if JSON marshaling fails
		wl.logger.Printf("[%s] %s: %v (JSON marshal error: %v)", level, event, fields, err)
		return
	}

	// Write to stdout
	wl.logger.Println(string(jsonBytes))
}

// LogWithFullPayload logs the webhook event with the complete GitHub payload (Phase 3 - audit trail)
// This is used for comprehensive audit logging
func (wl *WebhookLogger) LogWithFullPayload(deliveryID, eventType string, payload map[string]interface{}, status string) {
	wl.log(LevelInfo, "webhook_audit", map[string]interface{}{
		"delivery_id":    deliveryID,
		"event_type":     eventType,
		"status":         status,
		"github_payload": payload,
	})
}

// Example structured log output format:
// {
//   "timestamp": "2026-01-29T16:45:22Z",
//   "level": "INFO",
//   "event": "webhook_received",
//   "component": "webhook",
//   "delivery_id": "12345678-1234-1234-1234-123456789abc",
//   "event_type": "issue_comment",
//   "payload_size": 2048
// }
