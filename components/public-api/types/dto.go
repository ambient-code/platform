package types

// SessionResponse is the simplified session response for the public API
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
	Task  string `json:"task" binding:"required"`
	Model string `json:"model,omitempty"`
	Repos []Repo `json:"repos,omitempty"`
}

// Repo represents a repository configuration
type Repo struct {
	URL    string `json:"url" binding:"required"`
	Branch string `json:"branch,omitempty"`
}

// ErrorResponse is a standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
