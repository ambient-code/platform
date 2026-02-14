// Package types provides HTTP API types for the Ambient Platform Public API.
package types

import (
	"fmt"
	"net/url"
	"strings"
)

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

// Validate validates the CreateSessionRequest for common issues
func (r *CreateSessionRequest) Validate() error {
	if strings.TrimSpace(r.Task) == "" {
		return fmt.Errorf("task cannot be empty")
	}
	
	if len(r.Task) > 10000 {
		return fmt.Errorf("task exceeds maximum length of 10,000 characters")
	}
	
	// Validate model if provided
	if r.Model != "" && !isValidModel(r.Model) {
		return fmt.Errorf("invalid model: %s", r.Model)
	}
	
	// Validate repositories
	for i, repo := range r.Repos {
		if err := repo.Validate(); err != nil {
			return fmt.Errorf("repository %d: %w", i, err)
		}
	}
	
	return nil
}

// Validate validates the RepoHTTP for common issues
func (r *RepoHTTP) Validate() error {
	if strings.TrimSpace(r.URL) == "" {
		return fmt.Errorf("repository URL cannot be empty")
	}
	
	// Parse and validate URL
	parsedURL, err := url.Parse(r.URL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	
	// Check for supported schemes
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return fmt.Errorf("unsupported URL scheme: %s (must be http or https)", parsedURL.Scheme)
	}
	
	// Check for common invalid URLs
	if strings.Contains(r.URL, "example.com") || strings.Contains(r.URL, "localhost") {
		return fmt.Errorf("URL appears to be a placeholder or localhost")
	}
	
	return nil
}

// isValidModel checks if the model name is in the expected format
func isValidModel(model string) bool {
	validModels := []string{
		"claude-3.5-sonnet",
		"claude-3.5-haiku", 
		"claude-3-opus",
		"claude-3-sonnet",
		"claude-3-haiku",
	}
	
	for _, validModel := range validModels {
		if model == validModel {
			return true
		}
	}
	
	return false
}