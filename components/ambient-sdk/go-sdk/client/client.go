package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

type ClientOption func(*Client)

type Client struct {
	baseURL    string
	basePath   string
	token      SecureToken
	project    string
	httpClient *http.Client
	logger     *slog.Logger
}

func NewClient(baseURL, token, project string, opts ...ClientOption) (*Client, error) {
	secureToken := SecureToken(token)
	if err := secureToken.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if project == "" {
		return nil, fmt.Errorf("project cannot be empty")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		ReplaceAttr: sanitizeLogAttrs,
	}))

	c := &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		basePath: "/api/ambient-api-server/v1",
		token:    secureToken,
		project:  project,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.logger.Info("ambient client created",
		"base_url", c.baseURL,
		"project", c.project)

	return c, nil
}

func NewClientFromEnv(opts ...ClientOption) (*Client, error) {
	baseURL := os.Getenv("AMBIENT_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	token := os.Getenv("AMBIENT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("AMBIENT_TOKEN environment variable is required")
	}
	project := os.Getenv("AMBIENT_PROJECT")
	if project == "" {
		return nil, fmt.Errorf("AMBIENT_PROJECT environment variable is required")
	}
	return NewClient(baseURL, token, project, opts...)
}

func WithBasePath(path string) ClientOption {
	return func(c *Client) {
		c.basePath = path
	}
}

func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, expectedStatus int, result any) error {
	fullURL := c.baseURL + c.basePath + path

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token.String())
	req.Header.Set("X-Ambient-Project", c.project)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != expectedStatus {
		var apiErr types.APIError
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Code != "" {
			apiErr.StatusCode = resp.StatusCode
			return &apiErr
		}
		return &types.APIError{
			StatusCode: resp.StatusCode,
			Code:       http.StatusText(resp.StatusCode),
			Reason:     fmt.Sprintf("unexpected status %d", resp.StatusCode),
		}
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}

func (c *Client) doWithQuery(ctx context.Context, method, path string, body []byte, expectedStatus int, result any, opts *types.ListOptions) error {
	if opts != nil {
		params := url.Values{}
		if opts.Page > 0 {
			params.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.Size > 0 {
			params.Set("size", strconv.Itoa(opts.Size))
		}
		if opts.Search != "" {
			params.Set("search", opts.Search)
		}
		if opts.OrderBy != "" {
			params.Set("orderBy", opts.OrderBy)
		}
		if opts.Fields != "" {
			params.Set("fields", opts.Fields)
		}
		if encoded := params.Encode(); encoded != "" {
			path = path + "?" + encoded
		}
	}
	return c.do(ctx, method, path, body, expectedStatus, result)
}

type SecureToken string

func (t SecureToken) LogValue() slog.Value {
	if len(t) == 0 {
		return slog.StringValue("[EMPTY]")
	}
	if len(t) < 8 {
		return slog.StringValue("[TOO_SHORT]")
	}
	return slog.StringValue(fmt.Sprintf("%s***(%d chars)", string(t)[:6], len(t)))
}

func (t SecureToken) String() string {
	return string(t)
}

func (t SecureToken) IsValid() error {
	if len(t) == 0 {
		return fmt.Errorf("token cannot be empty")
	}

	tokenStr := string(t)

	placeholders := []string{
		"YOUR_TOKEN_HERE", "your-token-here", "token", "password",
		"secret", "example", "test", "demo", "placeholder", "TODO",
	}
	for _, placeholder := range placeholders {
		if strings.EqualFold(tokenStr, placeholder) {
			return fmt.Errorf("token appears to be a placeholder value")
		}
	}

	if len(tokenStr) < 10 {
		return fmt.Errorf("token is too short (minimum 10 characters)")
	}

	if strings.HasPrefix(tokenStr, "sha256~") {
		if len(tokenStr) < 20 {
			return fmt.Errorf("OpenShift token too short")
		}
		return nil
	}

	if strings.Count(tokenStr, ".") == 2 {
		parts := strings.Split(tokenStr, ".")
		allValid := true
		for _, part := range parts {
			if len(part) == 0 {
				allValid = false
				break
			}
		}
		if allValid {
			return nil
		}
	}

	if strings.HasPrefix(tokenStr, "ghp_") || strings.HasPrefix(tokenStr, "gho_") ||
		strings.HasPrefix(tokenStr, "ghu_") || strings.HasPrefix(tokenStr, "ghs_") {
		if len(tokenStr) < 40 {
			return fmt.Errorf("GitHub token too short")
		}
		return nil
	}

	return nil
}

func sanitizeLogAttrs(_ []string, attr slog.Attr) slog.Attr {
	key := attr.Key
	value := attr.Value

	switch key {
	case "token", "Token", "TOKEN",
		"password", "Password", "PASSWORD",
		"secret", "Secret", "SECRET",
		"apikey", "api_key", "ApiKey", "API_KEY",
		"authorization", "Authorization", "AUTHORIZATION":
		return slog.String(key, "[REDACTED]")
	}

	keyLower := strings.ToLower(key)
	if strings.HasSuffix(keyLower, "_token") || strings.HasSuffix(keyLower, "_password") ||
		strings.HasSuffix(keyLower, "_secret") || strings.HasSuffix(keyLower, "_key") {
		return slog.String(key, "[REDACTED]")
	}

	if value.Kind() == slog.KindString {
		str := value.String()
		if strings.HasPrefix(str, "Bearer ") || strings.HasPrefix(str, "bearer ") {
			return slog.String(key, "[REDACTED_BEARER]")
		}
		if strings.HasPrefix(str, "sha256~") {
			return slog.String(key, "[REDACTED_SHA256_TOKEN]")
		}
		if strings.HasPrefix(str, "ey") && strings.Count(str, ".") >= 2 && len(str) > 50 {
			return slog.String(key, "[REDACTED_JWT]")
		}
	}

	return attr
}
