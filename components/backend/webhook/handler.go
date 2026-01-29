package webhook

import (
	"context"
	"time"

	"ambient-code-backend/github"
	"ambient-code-backend/webhook/parsers"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// WebhookHandler handles GitHub webhook requests (FR-001, FR-006, FR-009)
type WebhookHandler struct {
	config                *Config
	deduplicationCache    *DeduplicationCache
	installationVerifier  *InstallationVerifier
	keywordDetector       *KeywordDetector
	sessionCreator        *SessionCreator
	githubCommenter       *GitHubCommenter
	logger                *WebhookLogger
}

// NewWebhookHandler creates a new webhook handler with all dependencies
func NewWebhookHandler(
	config *Config,
	k8sClient kubernetes.Interface,
	dynamicClient dynamic.Interface,
	namespace string,
	gvr schema.GroupVersionResource,
	tokenManager *github.TokenManager,
) *WebhookHandler {
	logger := NewWebhookLogger()

	return &WebhookHandler{
		config:                config,
		deduplicationCache:    NewDeduplicationCache(24 * time.Hour),
		installationVerifier:  NewInstallationVerifier(k8sClient, namespace),
		keywordDetector:       NewKeywordDetector(),
		sessionCreator:        NewSessionCreator(dynamicClient, namespace, gvr, logger),
		githubCommenter:       NewGitHubCommenter(tokenManager, logger),
		logger:                logger,
	}
}

// HandleWebhook processes incoming GitHub webhook requests (FR-001 through FR-018)
func (wh *WebhookHandler) HandleWebhook(c *gin.Context) {
	startTime := time.Now()

	// Step 1: Validate request (FR-004, FR-005, FR-010)
	headers, payload, err := ValidateWebhookRequest(c.Request)
	if err != nil {
		deliveryID := c.GetHeader("X-GitHub-Delivery")
		if deliveryID == "" {
			deliveryID = "unknown"
		}

		// Log and respond based on validation error
		switch err {
		case ErrInvalidMethod:
			RecordWebhookRejected("invalid_method")
			RespondBadRequest(c, "Invalid HTTP method, must be POST", deliveryID)
		case ErrInvalidContentType:
			RecordWebhookRejected("invalid_content_type")
			RespondBadRequest(c, "Invalid Content-Type, must be application/json", deliveryID)
		case ErrPayloadTooLarge:
			RecordWebhookRejected("payload_too_large")
			RespondPayloadTooLarge(c, deliveryID)
		case ErrMissingHeaders:
			RecordWebhookRejected("missing_headers")
			RespondBadRequest(c, "Missing required webhook headers", deliveryID)
		default:
			RecordWebhookRejected("validation_error")
			RespondBadRequest(c, err.Error(), deliveryID)
		}
		return
	}

	deliveryID := headers.DeliveryID
	eventType := headers.Event

	// Log webhook received
	wh.logger.LogWebhookReceived(deliveryID, eventType, len(payload))
	RecordWebhookReceived(eventType)
	RecordPayloadSize(eventType, len(payload))

	// Step 2: Verify HMAC signature (FR-002, FR-003, FR-007)
	if err := VerifySignature(headers.Signature, payload, wh.config.WebhookSecret); err != nil {
		wh.logger.LogSignatureInvalid(deliveryID, eventType, err.Error())
		RecordWebhookRejected("invalid_signature")
		RespondUnauthorized(c, "Invalid webhook signature", deliveryID)
		return
	}

	wh.logger.LogSignatureVerified(deliveryID, eventType)
	RecordWebhookAccepted(eventType)

	// Step 3: Check for duplicate delivery ID (FR-023)
	if wh.deduplicationCache.IsDuplicate(deliveryID) {
		wh.logger.LogDuplicateDetected(deliveryID, eventType)
		RecordDuplicateDetected()
		// Return 200 OK to acknowledge receipt (prevents GitHub retries)
		RespondWithSuccess(c, "Webhook already processed (duplicate delivery ID)", deliveryID)
		return
	}

	// Add to deduplication cache
	wh.deduplicationCache.Add(deliveryID)

	// Step 4: Check if event type is supported (FR-012)
	if !IsSupportedEventType(eventType) {
		wh.logger.LogDebug(deliveryID, "Unsupported event type", map[string]interface{}{
			"event_type": eventType,
		})
		// Return 200 OK but don't process (not an error, just unsupported)
		RespondWithSuccess(c, "Event type not supported", deliveryID)
		return
	}

	// Step 5: Process event based on type
	ctx := context.Background()
	switch eventType {
	case "issue_comment":
		wh.handleIssueComment(ctx, c, deliveryID, payload, startTime)
	case "pull_request":
		// Phase 1B: Auto-review support
		RespondWithSuccess(c, "pull_request events not yet supported (Phase 1B)", deliveryID)
	case "workflow_run":
		// Phase 1C: CI failure debugging
		RespondWithSuccess(c, "workflow_run events not yet supported (Phase 1C)", deliveryID)
	default:
		RespondWithSuccess(c, "Event type acknowledged but not processed", deliveryID)
	}

	// Update cache size metrics
	UpdateCacheSizes(wh.deduplicationCache.Size(), wh.installationVerifier.CacheSize())

	// Record processing duration
	duration := time.Since(startTime)
	RecordProcessingDuration(eventType, duration.Seconds())
}

// handleIssueComment processes issue_comment webhook events (FR-011, FR-012, FR-013)
func (wh *WebhookHandler) handleIssueComment(ctx context.Context, c *gin.Context, deliveryID string, payload []byte, startTime time.Time) {
	// Parse the issue_comment payload
	sessionCtx, err := parsers.ParseIssueComment(payload)
	if err != nil {
		wh.logger.LogError(deliveryID, "parser", "Failed to parse issue_comment payload", err)
		RespondBadRequest(c, "Invalid issue_comment payload", deliveryID)
		return
	}

	// Phase 1A: Only process PR comments (not standalone issues)
	isPR, _ := parsers.IsPRComment(payload)
	if !isPR {
		wh.logger.LogDebug(deliveryID, "Ignoring standalone issue comment (Phase 1A)", nil)
		RespondWithSuccess(c, "Standalone issue comments not yet supported (Phase 1C)", deliveryID)
		return
	}

	// Check if keyword is detected (FR-013, FR-026)
	if !wh.keywordDetector.DetectKeyword(sessionCtx.CommentBody) {
		wh.logger.LogDebug(deliveryID, "No keyword detected in comment", map[string]interface{}{
			"comment_body": sessionCtx.CommentBody,
		})
		RespondWithSuccess(c, "No @amber keyword detected", deliveryID)
		return
	}

	wh.logger.LogKeywordDetected(deliveryID, "@amber", wh.keywordDetector.ExtractCommand(sessionCtx.CommentBody))

	// Verify GitHub App installation (FR-008, FR-009, FR-016, FR-025)
	installationID, err := wh.installationVerifier.VerifyInstallation(ctx, sessionCtx.Repository)
	if err != nil {
		wh.logger.LogAuthorizationChecked(deliveryID, sessionCtx.Repository, false, 0)
		RecordWebhookRejected("not_authorized")

		// Post error comment to GitHub
		if sessionCtx.PRNumber != nil {
			_ = wh.githubCommenter.PostErrorComment(ctx, 0, sessionCtx.Repository, *sessionCtx.PRNumber, "not_authorized", "Repository not authorized - GitHub App not installed", deliveryID)
		}

		RespondUnauthorized(c, "Repository not authorized - GitHub App not installed", deliveryID)
		return
	}

	wh.logger.LogAuthorizationChecked(deliveryID, sessionCtx.Repository, true, installationID)

	// Create agentic session (FR-014)
	sessionID, err := wh.sessionCreator.CreateSession(ctx, sessionCtx, deliveryID)
	if err != nil {
		RecordWebhookFailed(sessionCtx.EventType, "session_creation_failed")

		// Post error comment to GitHub
		if sessionCtx.PRNumber != nil {
			_ = wh.githubCommenter.PostErrorComment(ctx, installationID, sessionCtx.Repository, *sessionCtx.PRNumber, "session_creation_failed", err.Error(), deliveryID)
		}

		RespondInternalServerError(c, "Failed to create session", deliveryID)
		return
	}

	// Record session creation success
	RecordSessionCreated(sessionCtx.EventType, sessionCtx.TriggerReason)

	// Post confirmation comment to GitHub (FR-015, FR-017)
	if sessionCtx.PRNumber != nil {
		if err := wh.githubCommenter.PostConfirmationComment(ctx, installationID, sessionCtx.Repository, *sessionCtx.PRNumber, sessionID, deliveryID); err != nil {
			wh.logger.LogError(deliveryID, "github_commenter", "Failed to post confirmation comment", err)
			// Don't fail the request if comment posting fails
		}
	}

	// Log final processing status
	duration := time.Since(startTime)
	wh.logger.LogWebhookProcessed(deliveryID, sessionCtx.EventType, sessionCtx.Repository, sessionCtx.TriggeredBy, "success", duration.Milliseconds())

	// Return success response (FR-009)
	RespondWithSuccess(c, "Webhook processed successfully, session created", deliveryID)
}
