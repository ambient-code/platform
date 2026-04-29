package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

// doMultiStatus is like do but accepts multiple success status codes.
// Used for endpoints that can return 200 or 201 depending on whether
// a new resource was created or an existing one was returned.
func (c *Client) doMultiStatus(ctx context.Context, method, path string, body []byte, result interface{}, expectedStatuses ...int) error {
	url := c.baseURL + "/api/ambient/v1" + path

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-Ambient-Project", c.project)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	for _, status := range expectedStatuses {
		if resp.StatusCode == status {
			if result != nil && len(respBody) > 0 {
				if err := json.Unmarshal(respBody, result); err != nil {
					return fmt.Errorf("unmarshal response: %w", err)
				}
			}
			return nil
		}
	}

	var apiErr types.APIError
	if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Code != "" {
		apiErr.StatusCode = resp.StatusCode
		return &apiErr
	}
	return &types.APIError{
		StatusCode: resp.StatusCode,
		Code:       "http_error",
		Reason:     fmt.Sprintf("HTTP %d: unexpected status", resp.StatusCode),
	}
}

// streamingHTTPClient returns an HTTP client with no timeout, suitable for
// long-lived streaming connections (SSE, etc.).
func (c *Client) streamingHTTPClient() *http.Client {
	clone := *c.httpClient
	clone.Timeout = 0
	return &clone
}
