package webhook

import "time"

// GitHubWebhookHeaders contains the standard GitHub webhook headers
type GitHubWebhookHeaders struct {
	Signature   string // X-Hub-Signature-256
	Event       string // X-GitHub-Event
	DeliveryID  string // X-GitHub-Delivery
	HookID      string // X-GitHub-Hook-ID
	ContentType string // Content-Type
}

// IssueCommentPayload represents the GitHub issue_comment webhook payload
// Reference: https://docs.github.com/en/webhooks/webhook-events-and-payloads#issue_comment
type IssueCommentPayload struct {
	Action string `json:"action"` // created, edited, deleted
	Issue  struct {
		Number      int    `json:"number"`
		Title       string `json:"title"`
		State       string `json:"state"`
		HTMLURL     string `json:"html_url"`
		PullRequest *struct {
			URL     string `json:"url"`
			HTMLURL string `json:"html_url"`
		} `json:"pull_request,omitempty"` // Present if this is a PR
	} `json:"issue"`
	Comment struct {
		ID        int64     `json:"id"`
		Body      string    `json:"body"`
		User      GitHubUser `json:"user"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		HTMLURL   string    `json:"html_url"`
	} `json:"comment"`
	Repository GitHubRepository `json:"repository"`
	Sender     GitHubUser       `json:"sender"`
}

// PullRequestPayload represents the GitHub pull_request webhook payload
// Reference: https://docs.github.com/en/webhooks/webhook-events-and-payloads#pull_request
type PullRequestPayload struct {
	Action      string `json:"action"` // opened, synchronize, reopened, closed, etc.
	Number      int    `json:"number"`
	PullRequest struct {
		Number    int       `json:"number"`
		Title     string    `json:"title"`
		State     string    `json:"state"`
		HTMLURL   string    `json:"html_url"`
		Head      GitHubRef `json:"head"`
		Base      GitHubRef `json:"base"`
		User      GitHubUser `json:"user"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	} `json:"pull_request"`
	Repository GitHubRepository `json:"repository"`
	Sender     GitHubUser       `json:"sender"`
}

// WorkflowRunPayload represents the GitHub workflow_run webhook payload
// Reference: https://docs.github.com/en/webhooks/webhook-events-and-payloads#workflow_run
type WorkflowRunPayload struct {
	Action      string `json:"action"` // completed, requested, in_progress
	WorkflowRun struct {
		ID          int64     `json:"id"`
		Name        string    `json:"name"`
		Status      string    `json:"status"`      // queued, in_progress, completed
		Conclusion  string    `json:"conclusion"`  // success, failure, cancelled, etc.
		HTMLURL     string    `json:"html_url"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		HeadBranch  string    `json:"head_branch"`
		HeadSHA     string    `json:"head_sha"`
		LogsURL     string    `json:"logs_url"`
	} `json:"workflow_run"`
	Repository GitHubRepository `json:"repository"`
	Sender     GitHubUser       `json:"sender"`
}

// GitHubRepository represents the repository information in webhook payloads
type GitHubRepository struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"` // e.g., "owner/repo"
	Owner    GitHubUser `json:"owner"`
	Private  bool   `json:"private"`
	HTMLURL  string `json:"html_url"`
}

// GitHubUser represents a GitHub user in webhook payloads
type GitHubUser struct {
	Login string `json:"login"`
	ID    int64  `json:"id"`
	Type  string `json:"type"` // User, Bot, Organization
}

// GitHubRef represents a git reference (branch/tag) in PR payloads
type GitHubRef struct {
	Ref  string           `json:"ref"`  // e.g., "refs/heads/main"
	SHA  string           `json:"sha"`
	Repo GitHubRepository `json:"repo"`
}

// WebhookEvent represents a processed webhook event (in-memory only)
type WebhookEvent struct {
	DeliveryID string
	EventType  string
	Timestamp  time.Time
}

// SessionContext contains the extracted information needed to create an agentic session
type SessionContext struct {
	Source        string // "webhook"
	EventType     string // issue_comment, pull_request, workflow_run
	DeliveryID    string
	Repository    string // owner/repo
	GitHubURL     string // Link to PR/issue
	PRNumber      *int   // PR number (if applicable)
	IssueNumber   *int   // Issue number (if applicable)
	TriggeredBy   string // GitHub username
	TriggerReason string // Keyword detected, auto-review, CI failure, etc.
	CommentBody   string // Original comment text (for context)
}
