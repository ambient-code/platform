package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	// BackendURL is the internal backend service URL
	BackendURL = getEnvOrDefault("BACKEND_URL", "http://backend-service:8080")

	// BackendTimeout is the timeout for backend requests (configurable via BACKEND_TIMEOUT env var)
	BackendTimeout = getTimeoutFromEnv("BACKEND_TIMEOUT", 30*time.Second)

	// HTTPClient is the shared HTTP client for backend requests
	HTTPClient = &http.Client{
		Timeout: BackendTimeout,
	}
)

func getTimeoutFromEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ProxyRequest forwards a request to the backend and returns the response
func ProxyRequest(c *gin.Context, method, path string, body []byte) (*http.Response, error) {
	fullURL := fmt.Sprintf("%s%s", BackendURL, path)

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	// Create context with explicit timeout (in addition to HTTP client timeout)
	// This ensures we respect context cancellation from the client
	ctx, cancel := context.WithTimeout(c.Request.Context(), BackendTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Forward the token
	token := GetToken(c)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// Forward content type
	if contentType := c.GetHeader("Content-Type"); contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Set accept header
	req.Header.Set("Accept", "application/json")

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("backend request failed: %w", err)
	}

	return resp, nil
}

// ProxyAndRespond proxies a request and writes the response directly
func ProxyAndRespond(c *gin.Context, method, path string, body []byte) {
	resp, err := ProxyRequest(c, method, path, body)
	if err != nil {
		// Log detailed error internally, return generic message to user
		// SECURITY: Never expose internal error details (may contain URLs, tokens, etc.)
		log.Printf("Backend request failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read backend response"})
		return
	}

	// Forward response
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}
