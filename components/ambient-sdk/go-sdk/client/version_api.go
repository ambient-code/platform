package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ServerVersion struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	GitTag    string `json:"git_tag"`
}

func (c *Client) ServerVersion(ctx context.Context) (*ServerVersion, error) {
	var result ServerVersion
	if err := c.do(ctx, http.MethodGet, "/version", nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func FetchServerVersion(ctx context.Context, baseURL string, insecureSkipVerify bool) (*ServerVersion, error) {
	url := strings.TrimSuffix(baseURL, "/") + "/api/ambient/v1/version"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	httpClient := &http.Client{Timeout: 10 * time.Second}
	if insecureSkipVerify {
		t := http.DefaultTransport.(*http.Transport).Clone()
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		}
		t.TLSClientConfig.InsecureSkipVerify = true //nolint:gosec
		httpClient.Transport = t
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result ServerVersion
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &result, nil
}
