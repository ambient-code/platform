// Package types provides HTTP API types for the Ambient Platform Public API.
package types

// HTTP API Types (matching public-api/types/dto.go)

// SessionResponse is the simplified session response from the public API
type SessionResponse struct {
	ID          string `json:"id"`
	Status      string `json:"status"` // "pending", "running", "completed", "failed"
	Task        string `json:"task"`
	Model       string `json:"model,omitempty"`
	CreatedAt   string `json:"createdAt"`
	CompletedAt string `json:"completedAt,omitempty"`
	Result      string `json:"result,omitempty"`
	Error       string `json:"error,omitempty"`
}

// SessionListResponse is the response for listing sessions
type SessionListResponse struct {
	Items []SessionResponse `json:"items"`
	Total int               `json:"total"`
}

// CreateSessionRequest is the request body for creating a session
type CreateSessionRequest struct {
	Task  string     `json:"task"`
	Model string     `json:"model,omitempty"`
	Repos []RepoHTTP `json:"repos,omitempty"`
}

// CreateSessionResponse is the response from creating a session
type CreateSessionResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// RepoHTTP represents a repository configuration for HTTP API
type RepoHTTP struct {
	URL    string `json:"url"`
	Branch string `json:"branch,omitempty"`
}

// ErrorResponse is a standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// Session status constants
const (
	StatusPending   = "pending"
	StatusRunning   = "running"  
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)