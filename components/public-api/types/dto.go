// Package types defines data transfer objects for the public API.
package types

// SessionResponse is the simplified session response for the public API
type SessionResponse struct {
	ID          string        `json:"id"`
	Status      string        `json:"status"` // "pending", "running", "completed", "failed"
	DisplayName string        `json:"display_name,omitempty"`
	Task        string        `json:"task"`
	Model       string        `json:"model,omitempty"`
	Repos       []SessionRepo `json:"repos,omitempty"`
	CreatedAt   string        `json:"created_at"`
	CompletedAt string        `json:"completed_at,omitempty"`
	Result      string        `json:"result,omitempty"`
	Error       string        `json:"error,omitempty"`
}

// SessionListResponse is the response for listing sessions
type SessionListResponse struct {
	Items []SessionResponse `json:"items"`
	Total int               `json:"total"`
}

// CreateSessionRequest is the request body for creating a session
type CreateSessionRequest struct {
	Task                 string            `json:"task" binding:"required"`
	DisplayName          string            `json:"display_name,omitempty"`
	Model                string            `json:"model,omitempty"`
	Repos                []Repo            `json:"repos,omitempty"`
	ActiveWorkflow       *WorkflowRef      `json:"activeWorkflow,omitempty"`
	EnvironmentVariables map[string]string `json:"environmentVariables,omitempty"`
}

// WorkflowRef identifies a workflow by git repository location.
type WorkflowRef struct {
	GitURL string `json:"gitUrl" binding:"required"`
	Branch string `json:"branch,omitempty"`
	Path   string `json:"path,omitempty"`
}

// Repo represents a repository configuration
type Repo struct {
	URL    string `json:"url" binding:"required"`
	Branch string `json:"branch,omitempty"`
}

// CreateRunRequest is the request body for creating an AG-UI run
type CreateRunRequest struct {
	RunID    string       `json:"run_id,omitempty"`
	ThreadID string       `json:"thread_id,omitempty"`
	Messages []RunMessage `json:"messages" binding:"required,min=1"`
}

// RunMessage is a message in a run request
type RunMessage struct {
	ID      string `json:"id,omitempty"`
	Role    string `json:"role" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// CreateRunResponse is the response for creating a run
type CreateRunResponse struct {
	RunID    string `json:"run_id"`
	ThreadID string `json:"thread_id"`
}

// SessionRunsResponse is the response for listing runs in a session
type SessionRunsResponse struct {
	SessionID string       `json:"session_id"`
	Runs      []RunSummary `json:"runs"`
}

// RunSummary is a summary of a single run
type RunSummary struct {
	RunID       string `json:"run_id"`
	StartedAt   int64  `json:"started_at,omitempty"`
	FinishedAt  int64  `json:"finished_at,omitempty"`
	Status      string `json:"status"`
	UserMessage string `json:"user_message,omitempty"`
	EventCount  int    `json:"event_count"`
}

// SendMessageRequest is the request body for sending a message to a session
type SendMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// SendMessageResponse is the response for sending a message
type SendMessageResponse struct {
	RunID    string `json:"run_id"`
	ThreadID string `json:"thread_id"`
}

// TranscriptOutputResponse is the response for transcript format output
type TranscriptOutputResponse struct {
	SessionID string              `json:"session_id"`
	Format    string              `json:"format"`
	Messages  []TranscriptMessage `json:"messages"`
}

// TranscriptMessage is a single message in transcript output
type TranscriptMessage struct {
	ID         string               `json:"id,omitempty"`
	Role       string               `json:"role"`
	Content    string               `json:"content,omitempty"`
	ToolCalls  []TranscriptToolCall `json:"tool_calls,omitempty"`
	ToolCallID string               `json:"tool_call_id,omitempty"`
	Name       string               `json:"name,omitempty"`
	Timestamp  string               `json:"timestamp,omitempty"`
}

// TranscriptToolCall is a tool call in a transcript message
type TranscriptToolCall struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Args     string `json:"args,omitempty"`
	Result   string `json:"result,omitempty"`
	Status   string `json:"status,omitempty"`
	Duration int64  `json:"duration,omitempty"`
}

// EventsOutputResponse is the response for events/compact format output
type EventsOutputResponse struct {
	SessionID string                   `json:"session_id"`
	Format    string                   `json:"format"`
	Events    []map[string]interface{} `json:"events"`
}

// SessionRepo represents a repository configured for a session
type SessionRepo struct {
	URL    string `json:"url"`
	Branch string `json:"branch,omitempty"`
}

// MessageResponse is a simple message response
type MessageResponse struct {
	Message string `json:"message"`
}
