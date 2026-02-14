package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

// Client is a simple HTTP client for the Ambient Platform API
type Client struct {
	baseURL    string
	token      types.SecureToken
	project    string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewClient creates a new HTTP client for the Ambient Platform
// Returns an error if token validation fails
func NewClient(baseURL, token, project string) (*Client, error) {
	return NewClientWithTimeout(baseURL, token, project, 30*time.Second)
}

// NewClientWithTimeout creates a new HTTP client with custom timeout
// Returns an error if token validation fails
func NewClientWithTimeout(baseURL, token, project string, timeout time.Duration) (*Client, error) {
	secureToken := types.SecureToken(token)
	if err := secureToken.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	
	// Create logger with ReplaceAttr for additional sensitive data protection
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		ReplaceAttr: sanitizeLogAttrs,
	}))
	
	client := &Client{
		baseURL: baseURL,
		token:   secureToken,
		project: project,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
	
	// Log client creation (without any sensitive data)
	client.logger.Info("Ambient client created",
		"base_url", baseURL,
		"project", project,
		"timeout", timeout)
	
	return client, nil
}

// CreateSession creates a new agentic session
func (c *Client) CreateSession(ctx context.Context, req *types.CreateSessionRequest) (*types.CreateSessionResponse, error) {
	// Validate the request first
	if err := req.Validate(); err != nil {
		c.logger.Error("Session creation failed validation", "error", err)
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	c.logger.Info("Creating session", 
		"task_length", len(req.Task),
		"model", req.Model,
		"repo_count", len(req.Repos))

	jsonBody, err := json.Marshal(req)
	if err != nil {
		c.logger.Error("Failed to marshal request", "error", err)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/v1/sessions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.token.String())
	httpReq.Header.Set("X-Ambient-Project", c.project)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.logger.Error("HTTP request failed", "error", err, "url", url)
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("Failed to read response body", "error", err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		c.logger.Error("Session creation failed", 
			"status_code", resp.StatusCode,
			"response_body_length", len(body))
		
		var errResp types.ErrorResponse
		if json.Unmarshal(body, &errResp) == nil {
			// Log full error details for debugging (sanitized by slog)
			c.logger.Debug("API error details", "error", errResp.Error, "message", errResp.Message)
			// Return generic error message to avoid exposing sensitive details
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error)
		}
		// Don't expose raw response body - return generic error
		return nil, fmt.Errorf("API error (%d): request failed", resp.StatusCode)
	}

	var createResp types.CreateSessionResponse
	if err := json.Unmarshal(body, &createResp); err != nil {
		c.logger.Error("Failed to unmarshal response", "error", err)
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.Info("Session created successfully", 
		"session_id", createResp.ID,
		"status_code", resp.StatusCode)

	return &createResp, nil
}

// GetSession retrieves a session by ID
func (c *Client) GetSession(ctx context.Context, sessionID string) (*types.SessionResponse, error) {
	url := c.baseURL + "/v1/sessions/" + sessionID
	
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token.String())
	httpReq.Header.Set("X-Ambient-Project", c.project)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp types.ErrorResponse
		if json.Unmarshal(body, &errResp) == nil {
			// Return generic error without exposing full details
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error)
		}
		return nil, fmt.Errorf("API error (%d): request failed", resp.StatusCode)
	}

	var session types.SessionResponse
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &session, nil
}

// ListSessions lists all sessions
func (c *Client) ListSessions(ctx context.Context) (*types.SessionListResponse, error) {
	url := c.baseURL + "/v1/sessions"
	
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token.String())
	httpReq.Header.Set("X-Ambient-Project", c.project)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp types.ErrorResponse
		if json.Unmarshal(body, &errResp) == nil {
			// Return generic error without exposing full details
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error)
		}
		return nil, fmt.Errorf("API error (%d): request failed", resp.StatusCode)
	}

	var listResp types.SessionListResponse
	if err := json.Unmarshal(body, &listResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &listResp, nil
}

// WaitForCompletion polls a session until it reaches a terminal state
func (c *Client) WaitForCompletion(ctx context.Context, sessionID string, pollInterval time.Duration) (*types.SessionResponse, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			session, err := c.GetSession(ctx, sessionID)
			if err != nil {
				return nil, err
			}

			switch session.Status {
			case types.StatusCompleted, types.StatusFailed:
				return session, nil
			case types.StatusPending, types.StatusRunning:
				// Continue polling
				continue
			default:
				return nil, fmt.Errorf("unknown session status: %s", session.Status)
			}
		}
	}
}

// sanitizeLogAttrs implements the ReplaceAttr approach for slog
// This provides a global safety net for any sensitive data that might be logged
func sanitizeLogAttrs(_ []string, attr slog.Attr) slog.Attr {
	key := attr.Key
	value := attr.Value
	
	// Sanitize by exact key name (case-sensitive for precision)
	switch key {
	case "token", "Token", "TOKEN":
		return slog.String(key, "[REDACTED]")
	case "password", "Password", "PASSWORD":
		return slog.String(key, "[REDACTED]")
	case "secret", "Secret", "SECRET":
		return slog.String(key, "[REDACTED]")
	case "apikey", "api_key", "ApiKey", "API_KEY":
		return slog.String(key, "[REDACTED]")
	case "authorization", "Authorization", "AUTHORIZATION":
		return slog.String(key, "[REDACTED]")
	}
	
	// Sanitize by key patterns (case-insensitive)
	keyLower := strings.ToLower(key)
	if strings.HasSuffix(keyLower, "_token") || strings.HasSuffix(keyLower, "_password") ||
	   strings.HasSuffix(keyLower, "_secret") || strings.HasSuffix(keyLower, "_key") {
		return slog.String(key, "[REDACTED]")
	}
	
	// Sanitize by value content ONLY for obvious token patterns
	if value.Kind() == slog.KindString {
		str := value.String()
		// Only redact clear Bearer token patterns
		if strings.HasPrefix(str, "Bearer ") || strings.HasPrefix(str, "bearer ") {
			return slog.String(key, "[REDACTED_BEARER]")
		}
		// Only redact SHA256 tokens (common OpenShift token format)
		if strings.HasPrefix(str, "sha256~") {
			return slog.String(key, "[REDACTED_SHA256_TOKEN]")
		}
		// Only redact JWT patterns (starts with ey and contains dots)
		if strings.HasPrefix(str, "ey") && strings.Count(str, ".") >= 2 && len(str) > 50 {
			return slog.String(key, "[REDACTED_JWT]")
		}
	}
	
	return attr
}