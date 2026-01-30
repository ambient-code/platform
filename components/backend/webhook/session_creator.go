package webhook

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

const (
	// SessionCreationTimeout is the maximum time to wait for synchronous session creation (FR-014)
	SessionCreationTimeout = 5 * time.Second
)

// SessionCreator creates agentic sessions from webhook events (FR-014)
type SessionCreator struct {
	k8sClient     kubernetes.Interface
	dynamicClient dynamic.Interface
	gvr           schema.GroupVersionResource
	logger        *WebhookLogger
}

// NewSessionCreator creates a new session creator
func NewSessionCreator(k8sClient kubernetes.Interface, dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, logger *WebhookLogger) *SessionCreator {
	return &SessionCreator{
		k8sClient:     k8sClient,
		dynamicClient: dynamicClient,
		gvr:           gvr,
		logger:        logger,
	}
}

// CreateSession creates an agentic session from webhook context (FR-014)
// It uses deterministic session naming (FR-024) and creates the session synchronously
// The namespace parameter specifies where to create the session (must be authorized via ProjectSettings)
// Returns the session ID if successful, error otherwise
func (sc *SessionCreator) CreateSession(ctx context.Context, namespace string, sessionCtx *SessionContext, deliveryID string) (string, error) {
	// Generate deterministic session name (FR-024)
	sessionName := GenerateSessionName(sessionCtx.Repository, sessionCtx.PRNumber, sessionCtx.IssueNumber, deliveryID)

	// Build initial prompt based on trigger context
	initialPrompt := sc.buildInitialPrompt(sessionCtx)

	// Create session spec
	session := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      sessionName,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"source":               "webhook",
					"github.com/repo":      sessionCtx.Repository,
					"github.com/event":     sessionCtx.EventType,
					"webhook/delivery-id":  deliveryID,
				},
				"annotations": map[string]interface{}{
					"webhook/github-url":      sessionCtx.GitHubURL,
					"webhook/triggered-by":    sessionCtx.TriggeredBy,
					"webhook/trigger-reason":  sessionCtx.TriggerReason,
				},
			},
			"spec": map[string]interface{}{
				"displayName":   fmt.Sprintf("Webhook: %s", sessionCtx.TriggerReason),
				"project":       namespace,
				"initialPrompt": initialPrompt,
				"llmSettings": map[string]interface{}{
					"model":       "sonnet",
					"temperature": 0.7,
					"maxTokens":   4000,
				},
				"timeout": 300, // 5 minute timeout
				"environmentVariables": map[string]string{
					"WEBHOOK_DELIVERY_ID": deliveryID,
					"GITHUB_REPOSITORY":   sessionCtx.Repository,
					"GITHUB_EVENT_TYPE":   sessionCtx.EventType,
				},
			},
			"status": map[string]interface{}{
				"phase": "Pending",
			},
		},
	}

	// Add PR number label if applicable
	if sessionCtx.PRNumber != nil {
		if err := unstructured.SetNestedField(session.Object, fmt.Sprintf("%d", *sessionCtx.PRNumber), "metadata", "labels", "github.com/pr-number"); err != nil {
			return "", fmt.Errorf("failed to set PR number label: %w", err)
		}
	}

	// Add issue number label if applicable
	if sessionCtx.IssueNumber != nil {
		if err := unstructured.SetNestedField(session.Object, fmt.Sprintf("%d", *sessionCtx.IssueNumber), "metadata", "labels", "github.com/issue-number"); err != nil {
			return "", fmt.Errorf("failed to set issue number label: %w", err)
		}
	}

	// Add OwnerReferences to namespace for proper cleanup (C2 fix)
	ns, err := sc.k8sClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		sc.logger.LogError(deliveryID, "session_creator", "Failed to get namespace for OwnerReferences", err)
		// Continue without OwnerReferences - not critical for session creation
	} else {
		ownerRefs := []interface{}{
			map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Namespace",
				"name":       namespace,
				"uid":        string(ns.UID),
			},
		}
		if err := unstructured.SetNestedSlice(session.Object, ownerRefs, "metadata", "ownerReferences"); err != nil {
			sc.logger.LogError(deliveryID, "session_creator", "Failed to set OwnerReferences", err)
			// Continue without OwnerReferences - not critical for session creation
		}
	}

	// Create context with timeout for synchronous creation (FR-014)
	createCtx, cancel := context.WithTimeout(ctx, SessionCreationTimeout)
	defer cancel()

	// Attempt synchronous creation
	sc.logger.LogDebug(deliveryID, "Creating agentic session", map[string]interface{}{
		"session_name": sessionName,
		"namespace":    namespace,
	})

	created, err := sc.dynamicClient.Resource(sc.gvr).Namespace(namespace).Create(createCtx, session, metav1.CreateOptions{})
	if err != nil {
		sc.logger.LogSessionCreationFailed(deliveryID, sessionCtx.EventType, "kubernetes_api_error", err.Error())
		return "", fmt.Errorf("failed to create agentic session: %w", err)
	}

	// Extract session ID from created object
	sessionID, found, err := unstructured.NestedString(created.Object, "metadata", "name")
	if err != nil || !found {
		sc.logger.LogSessionCreationFailed(deliveryID, sessionCtx.EventType, "missing_session_id", "Created session but could not extract name")
		return "", fmt.Errorf("created session but could not extract name")
	}

	sc.logger.LogSessionCreated(deliveryID, sessionID, sessionCtx.EventType, sessionCtx.Repository, sessionCtx.GitHubURL)
	return sessionID, nil
}

// buildInitialPrompt constructs the initial prompt for the session based on context
func (sc *SessionCreator) buildInitialPrompt(sessionCtx *SessionContext) string {
	switch sessionCtx.EventType {
	case "issue_comment":
		if sessionCtx.PRNumber != nil {
			// PR comment with @amber keyword
			return fmt.Sprintf(
				"You have been requested to help with a GitHub pull request.\n\n"+
					"**Repository**: %s\n"+
					"**Pull Request**: #%d\n"+
					"**URL**: %s\n"+
					"**Requested by**: @%s\n\n"+
					"**User Comment**:\n%s\n\n"+
					"Please analyze the pull request and provide a helpful response based on the user's request.",
				sessionCtx.Repository,
				*sessionCtx.PRNumber,
				sessionCtx.GitHubURL,
				sessionCtx.TriggeredBy,
				sessionCtx.CommentBody,
			)
		} else {
			// Standalone issue comment
			return fmt.Sprintf(
				"You have been requested to help with a GitHub issue.\n\n"+
					"**Repository**: %s\n"+
					"**Issue**: #%d\n"+
					"**URL**: %s\n"+
					"**Requested by**: @%s\n\n"+
					"**User Comment**:\n%s\n\n"+
					"Please help address the user's request regarding this issue.",
				sessionCtx.Repository,
				*sessionCtx.IssueNumber,
				sessionCtx.GitHubURL,
				sessionCtx.TriggeredBy,
				sessionCtx.CommentBody,
			)
		}

	case "pull_request":
		// Auto-review (Phase 1B)
		return fmt.Sprintf(
			"A new pull request has been created and auto-review is enabled.\n\n"+
				"**Repository**: %s\n"+
				"**Pull Request**: #%d\n"+
				"**URL**: %s\n\n"+
				"Please perform an automatic code review of this pull request.",
			sessionCtx.Repository,
			*sessionCtx.PRNumber,
			sessionCtx.GitHubURL,
		)

	case "workflow_run":
		// CI failure analysis (Phase 1C)
		return fmt.Sprintf(
			"A GitHub Actions workflow has failed.\n\n"+
				"**Repository**: %s\n"+
				"**URL**: %s\n\n"+
				"Please analyze the workflow failure and provide diagnostic information.",
			sessionCtx.Repository,
			sessionCtx.GitHubURL,
		)

	default:
		return fmt.Sprintf(
			"Webhook event received from GitHub.\n\n"+
				"**Repository**: %s\n"+
				"**Event Type**: %s\n"+
				"**URL**: %s\n",
			sessionCtx.Repository,
			sessionCtx.EventType,
			sessionCtx.GitHubURL,
		)
	}
}
