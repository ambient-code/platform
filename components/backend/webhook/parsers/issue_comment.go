package parsers

import (
	"encoding/json"
	"fmt"

	"ambient-code-backend/webhook"
)

// ParseIssueComment parses an issue_comment webhook payload (FR-011, FR-012)
// Returns a SessionContext with all necessary information for session creation
func ParseIssueComment(payload []byte) (*webhook.SessionContext, error) {
	var event webhook.IssueCommentPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal issue_comment payload: %w", err)
	}

	// Only process "created" action (ignore edited, deleted)
	if event.Action != "created" {
		return nil, fmt.Errorf("ignoring issue_comment action: %s", event.Action)
	}

	// Determine if this is a PR or a standalone issue
	isPR := event.Issue.PullRequest != nil
	var prNumber *int
	var issueNumber *int
	var githubURL string

	if isPR {
		// This is a PR comment
		num := event.Issue.Number
		prNumber = &num
		githubURL = event.Issue.HTMLURL
	} else {
		// This is a standalone issue comment
		num := event.Issue.Number
		issueNumber = &num
		githubURL = event.Issue.HTMLURL
	}

	// Create session context
	ctx := &webhook.SessionContext{
		Source:        "webhook",
		EventType:     "issue_comment",
		Repository:    event.Repository.FullName,
		GitHubURL:     githubURL,
		PRNumber:      prNumber,
		IssueNumber:   issueNumber,
		TriggeredBy:   event.Comment.User.Login,
		TriggerReason: "keyword_detected", // Will be validated by keyword detector
		CommentBody:   event.Comment.Body,
	}

	return ctx, nil
}

// IsPRComment checks if the issue_comment event is for a PR (not a standalone issue)
func IsPRComment(payload []byte) (bool, error) {
	var event webhook.IssueCommentPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return false, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return event.Issue.PullRequest != nil, nil
}
