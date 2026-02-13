package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ambient/platform-sdk/types"
)

// Client is a simple HTTP client for the Ambient Platform API
type Client struct {
	baseURL    string
	token      string
	project    string
	httpClient *http.Client
}

// NewClient creates a new HTTP client for the Ambient Platform
func NewClient(baseURL, token, project string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		project: project,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithTimeout creates a new HTTP client with custom timeout
func NewClientWithTimeout(baseURL, token, project string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		project: project,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// CreateSession creates a new agentic session
func (c *Client) CreateSession(ctx context.Context, req *types.CreateSessionRequest) (*types.CreateSessionResponse, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/v1/sessions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.token)
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

	if resp.StatusCode != http.StatusCreated {
		var errResp types.ErrorResponse
		if json.Unmarshal(body, &errResp) == nil {
			return nil, fmt.Errorf("API error (%d): %s - %s", resp.StatusCode, errResp.Error, errResp.Message)
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var createResp types.CreateSessionResponse
	if err := json.Unmarshal(body, &createResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &createResp, nil
}

// GetSession retrieves a session by ID
func (c *Client) GetSession(ctx context.Context, sessionID string) (*types.SessionResponse, error) {
	url := c.baseURL + "/v1/sessions/" + sessionID
	
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token)
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
			return nil, fmt.Errorf("API error (%d): %s - %s", resp.StatusCode, errResp.Error, errResp.Message)
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
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

	httpReq.Header.Set("Authorization", "Bearer "+c.token)
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
			return nil, fmt.Errorf("API error (%d): %s - %s", resp.StatusCode, errResp.Error, errResp.Message)
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
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