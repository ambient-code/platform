// Package types provides HTTP API types for the Ambient Platform Public API.
package types

import (
	"fmt"
	"log/slog"
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

// SecureToken is a type-safe wrapper for authentication tokens that implements
// slog.LogValuer for automatic sanitization in logs
type SecureToken string

// LogValue implements slog.LogValuer to safely log tokens
func (t SecureToken) LogValue() slog.Value {
	if len(t) == 0 {
		return slog.StringValue("[EMPTY]")
	}
	if len(t) < 8 {
		return slog.StringValue("[TOO_SHORT]")
	}
	// Show only first 6 characters + length for debugging
	return slog.StringValue(fmt.Sprintf("%s***(%d_chars)", string(t)[:6], len(t)))
}

// String returns the actual token value - use with care
func (t SecureToken) String() string {
	return string(t)
}

// IsValid performs comprehensive token validation
func (t SecureToken) IsValid() error {
	if len(t) == 0 {
		return fmt.Errorf("token cannot be empty")
	}
	
	tokenStr := string(t)
	
	// Check for common placeholder values
	placeholders := []string{
		"YOUR_TOKEN_HERE", "your-token-here", "token", "password", 
		"secret", "example", "test", "demo", "placeholder", "TODO",
	}
	for _, placeholder := range placeholders {
		if strings.EqualFold(tokenStr, placeholder) {
			return fmt.Errorf("token appears to be a placeholder value")
		}
	}
	
	// Check minimum length for security
	if len(tokenStr) < 10 {
		return fmt.Errorf("token is too short (minimum 10 characters required)")
	}
	
	// Validate known token formats
	if err := t.validateTokenFormat(tokenStr); err != nil {
		return fmt.Errorf("invalid token format: %w", err)
	}
	
	return nil
}

// validateTokenFormat checks for known secure token formats
func (t SecureToken) validateTokenFormat(token string) error {
	// OpenShift SHA256 tokens
	if strings.HasPrefix(token, "sha256~") {
		if len(token) < 20 {
			return fmt.Errorf("OpenShift token too short")
		}
		return nil
	}
	
	// JWT tokens (3 base64 parts separated by dots)
	if strings.Count(token, ".") == 2 {
		parts := strings.Split(token, ".")
		if len(parts) == 3 {
			// Basic JWT structure validation
			for i, part := range parts {
				if len(part) == 0 {
					return fmt.Errorf("JWT part %d is empty", i+1)
				}
				// JWT parts should be base64-like (alphanumeric + _- characters)
				for _, char := range part {
					if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
						(char >= '0' && char <= '9') || char == '_' || char == '-') {
						return fmt.Errorf("JWT contains invalid characters")
					}
				}
			}
			return nil
		}
	}
	
	// GitHub tokens
	if strings.HasPrefix(token, "ghp_") || strings.HasPrefix(token, "gho_") || 
	   strings.HasPrefix(token, "ghu_") || strings.HasPrefix(token, "ghs_") {
		if len(token) < 40 {
			return fmt.Errorf("GitHub token too short")
		}
		return nil
	}
	
	// Generic validation for other tokens
	if len(token) < 20 {
		return fmt.Errorf("token appears too short for secure authentication")
	}
	
	// Must contain alphanumeric characters
	hasAlpha := false
	hasNumeric := false
	for _, char := range token {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' {
			hasAlpha = true
		}
		if char >= '0' && char <= '9' {
			hasNumeric = true
		}
	}
	
	if !hasAlpha || !hasNumeric {
		return fmt.Errorf("token must contain both alphabetic and numeric characters")
	}
	
	return nil
}

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