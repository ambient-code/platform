package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ambient-code-backend/github"
)

// GitHubCommenter handles posting comments to GitHub issues and PRs (FR-015, FR-016, FR-017, FR-018)
type GitHubCommenter struct {
	tokenManager *github.TokenManager
	logger       *WebhookLogger
}

// NewGitHubCommenter creates a new GitHub commenter
func NewGitHubCommenter(tokenManager *github.TokenManager, logger *WebhookLogger) *GitHubCommenter {
	return &GitHubCommenter{
		tokenManager: tokenManager,
		logger:       logger,
	}
}

// PostConfirmationComment posts a success comment when a session is created (FR-015, FR-017)
func (gc *GitHubCommenter) PostConfirmationComment(ctx context.Context, installationID int64, repository string, issueNumber int, sessionID string, deliveryID string) error {
	// Generate confirmation message
	comment := fmt.Sprintf(
		"✅ **Session Created**\n\n"+
			"I've created an agentic session to help with your request.\n\n"+
			"**Session ID:** `%s`\n"+
			"**Status:** Processing your request\n\n"+
			"I'll post my findings here when complete. This may take a few moments...",
		sessionID,
	)

	return gc.postComment(ctx, installationID, repository, issueNumber, comment, "confirmation", deliveryID)
}

// PostErrorComment posts an error comment when session creation fails (FR-016, FR-018)
// The error type and actionable guidance are included to help users resolve the issue
func (gc *GitHubCommenter) PostErrorComment(ctx context.Context, installationID int64, repository string, issueNumber int, errorType string, errorMessage string, deliveryID string) error {
	// Generate error message with actionable guidance
	var comment string

	switch errorType {
	case "quota_exceeded":
		comment = fmt.Sprintf(
			"❌ **Session Creation Failed: Quota Exceeded**\n\n"+
				"%s\n\n"+
				"**Action Required:** Contact your administrator to increase session quotas or wait for existing sessions to complete.\n\n"+
				"_Delivery ID: `%s`_",
			errorMessage,
			deliveryID,
		)
	case "not_authorized":
		comment = fmt.Sprintf(
			"❌ **Session Creation Failed: Not Authorized**\n\n"+
				"%s\n\n"+
				"**Action Required:** Install the GitHub App for this repository. Visit your repository settings → GitHub Apps to install.\n\n"+
				"_Delivery ID: `%s`_",
			errorMessage,
			deliveryID,
		)
	case "invalid_configuration":
		comment = fmt.Sprintf(
			"❌ **Session Creation Failed: Configuration Error**\n\n"+
				"%s\n\n"+
				"**Action Required:** Check your project settings and ensure webhook integration is properly configured.\n\n"+
				"_Delivery ID: `%s`_",
			errorMessage,
			deliveryID,
		)
	default:
		comment = fmt.Sprintf(
			"❌ **Session Creation Failed**\n\n"+
				"%s\n\n"+
				"**Action Required:** This is an unexpected error. Please contact your administrator with the delivery ID below.\n\n"+
				"_Delivery ID: `%s`_\n"+
				"_Error Type: `%s`_",
			errorMessage,
			deliveryID,
			errorType,
		)
	}

	return gc.postComment(ctx, installationID, repository, issueNumber, comment, "error", deliveryID)
}

// postComment is the internal method that posts a comment to GitHub via API
func (gc *GitHubCommenter) postComment(ctx context.Context, installationID int64, repository string, issueNumber int, body string, commentType string, deliveryID string) error {
	// Mint installation token
	token, _, err := gc.tokenManager.MintInstallationToken(ctx, installationID)
	if err != nil {
		gc.logger.LogError(deliveryID, "github_commenter", "Failed to mint installation token", err)
		return fmt.Errorf("failed to mint installation token: %w", err)
	}

	// Construct GitHub API URL for posting comments
	// API: POST /repos/{owner}/{repo}/issues/{issue_number}/comments
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d/comments", repository, issueNumber)

	// Create request body
	requestBody := map[string]string{
		"body": body,
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal comment body: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "Ambient-Code-Backend")
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		gc.logger.LogError(deliveryID, "github_commenter", "Failed to post comment to GitHub", err)
		return fmt.Errorf("failed to post comment to GitHub: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		errorMsg := fmt.Sprintf("GitHub API returned status %d: %s", resp.StatusCode, string(bodyBytes))
		gc.logger.LogError(deliveryID, "github_commenter", errorMsg, nil)
		gc.logger.LogGitHubCommentPosted(deliveryID, commentType, apiURL, false)
		return fmt.Errorf("GitHub API error: %s", errorMsg)
	}

	// Success - log the comment posting
	gc.logger.LogGitHubCommentPosted(deliveryID, commentType, apiURL, true)
	return nil
}

// GetIssueURL constructs the GitHub HTML URL for an issue or PR
func GetIssueURL(repository string, issueNumber int) string {
	return fmt.Sprintf("https://github.com/%s/issues/%d", repository, issueNumber)
}

// GetPullRequestURL constructs the GitHub HTML URL for a pull request
func GetPullRequestURL(repository string, prNumber int) string {
	return fmt.Sprintf("https://github.com/%s/pull/%d", repository, prNumber)
}
