package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// networkError extracts the inner error from a *url.Error, stripping the
// URL and HTTP method so they cannot leak into user-facing messages.
func networkError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}

// ValidateGitHubToken checks if a GitHub token is valid by calling the GitHub API
func ValidateGitHubToken(ctx context.Context, token string) (bool, error) {
	if token == "" {
		return false, fmt.Errorf("token is empty")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request")
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("request failed: %w", networkError(err))
	}
	defer resp.Body.Close()

	// 200 = valid, 401 = invalid/expired
	return resp.StatusCode == http.StatusOK, nil
}

// ValidateGitLabToken checks if a GitLab token is valid
func ValidateGitLabToken(ctx context.Context, token, instanceURL string) (bool, error) {
	if token == "" {
		return false, fmt.Errorf("token is empty")
	}
	if instanceURL == "" {
		instanceURL = "https://gitlab.com"
	}

	client := &http.Client{Timeout: 10 * time.Second}
	apiURL := fmt.Sprintf("%s/api/v4/user", instanceURL)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("request failed: %w", networkError(err))
	}
	defer resp.Body.Close()

	// 200 = valid, 401 = invalid/expired
	return resp.StatusCode == http.StatusOK, nil
}

// ValidateJiraToken checks if Jira credentials are valid
// Uses /rest/api/*/myself endpoint which accepts Basic Auth (API tokens)
func ValidateJiraToken(ctx context.Context, url, email, apiToken string) (bool, error) {
	if url == "" || email == "" || apiToken == "" {
		return false, fmt.Errorf("missing required credentials")
	}

	client := &http.Client{Timeout: 15 * time.Second}

	// Try API v3 first (Jira Cloud), fallback to v2 (Jira Server/DC)
	apiURLs := []string{
		fmt.Sprintf("%s/rest/api/3/myself", url),
		fmt.Sprintf("%s/rest/api/2/myself", url),
	}

	var got401 bool
	var lastNetErr error
	var lastReqErr error
	var sawHTTPResponse bool

	for _, apiURL := range apiURLs {
		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if err != nil {
			lastReqErr = err
			continue
		}

		// Jira uses Basic Auth with email:token
		req.SetBasicAuth(email, apiToken)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastNetErr = networkError(err)
			continue
		}
		sawHTTPResponse = true
		defer resp.Body.Close()

		// 200 = valid, 401 = invalid, 404 = wrong API version (try next)
		if resp.StatusCode == http.StatusOK {
			return true, nil
		}
		if resp.StatusCode == http.StatusUnauthorized {
			got401 = true
			continue
		}
	}

	// If got 401 on any attempt, credentials are definitely invalid
	if got401 {
		return false, nil
	}

	// If all attempts failed with network errors, surface the cause
	if lastNetErr != nil && !sawHTTPResponse {
		return false, fmt.Errorf("request failed: %w", lastNetErr)
	}

	// If no HTTP response was ever received and request construction failed,
	// surface the request-construction error instead of assuming valid.
	if !sawHTTPResponse && lastReqErr != nil {
		return false, fmt.Errorf("failed to create request: %w", lastReqErr)
	}

	// Couldn't validate - assume valid to avoid false negatives
	return true, nil
}

// ValidateGoogleToken checks if Google OAuth token is valid
func ValidateGoogleToken(ctx context.Context, accessToken string) (bool, error) {
	if accessToken == "" {
		return false, fmt.Errorf("token is empty")
	}

	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v1/userinfo", nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("request failed: %w", networkError(err))
	}
	defer resp.Body.Close()

	// 200 = valid, 401 = invalid/expired
	return resp.StatusCode == http.StatusOK, nil
}

// TestJiraConnection handles POST /api/auth/jira/test
// Tests Jira credentials without saving them
func TestJiraConnection(c *gin.Context) {
	var req struct {
		URL      string `json:"url" binding:"required"`
		Email    string `json:"email" binding:"required"`
		APIToken string `json:"apiToken" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	valid, err := ValidateJiraToken(c.Request.Context(), req.URL, req.Email, req.APIToken)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": err.Error()})
		return
	}

	if !valid {
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": "Invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true, "message": "Jira connection successful"})
}

// validateGerritTokenFn is a package-level var for test mockability
var validateGerritTokenFn = ValidateGerritToken

// ValidateGerritToken checks if Gerrit credentials are valid by calling /a/accounts/self
func ValidateGerritToken(ctx context.Context, gerritURL, authMethod, username, httpToken, gitcookiesContent string) (bool, error) {
	if gerritURL == "" {
		return false, fmt.Errorf("missing Gerrit URL")
	}

	apiURL := fmt.Sprintf("%s/a/accounts/self", strings.TrimSuffix(gerritURL, "/"))

	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: ssrfSafeTransport(),
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request")
	}

	switch authMethod {
	case gerritAuthHTTPBasic:
		if username == "" || httpToken == "" {
			return false, fmt.Errorf("missing username or HTTP token")
		}
		req.SetBasicAuth(username, httpToken)
	case gerritAuthGitCookies:
		if gitcookiesContent == "" {
			return false, fmt.Errorf("missing gitcookies content")
		}
		cookie, err := parseGitcookies(gerritURL, gitcookiesContent)
		if err != nil {
			return false, fmt.Errorf("failed to parse gitcookies: %w", err)
		}
		req.Header.Set("Cookie", cookie)
	default:
		return false, fmt.Errorf("unsupported auth method: %s", authMethod)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("request failed: %w", networkError(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return false, nil
	}

	return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// TestGerritConnection handles POST /api/auth/gerrit/test
// Tests Gerrit credentials without saving them
func TestGerritConnection(c *gin.Context) {
	if !FeatureEnabledForRequest(c, "gerrit.enabled") {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	var req struct {
		URL               string `json:"url" binding:"required"`
		AuthMethod        string `json:"authMethod" binding:"required"`
		Username          string `json:"username"`
		HTTPToken         string `json:"httpToken"`
		GitcookiesContent string `json:"gitcookiesContent"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate URL (SSRF protection)
	if err := validateGerritURL(req.URL); err != nil {
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": fmt.Sprintf("Invalid URL: %s", err.Error())})
		return
	}

	valid, err := validateGerritTokenFn(c.Request.Context(), req.URL, req.AuthMethod, req.Username, req.HTTPToken, req.GitcookiesContent)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": err.Error()})
		return
	}

	if !valid {
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": "Invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true, "message": "Gerrit connection successful"})
}

// TestGitLabConnection handles POST /api/auth/gitlab/test
// Tests GitLab credentials without saving them
func TestGitLabConnection(c *gin.Context) {
	var req struct {
		PersonalAccessToken string `json:"personalAccessToken" binding:"required"`
		InstanceURL         string `json:"instanceUrl"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.InstanceURL == "" {
		req.InstanceURL = "https://gitlab.com"
	}

	valid, err := ValidateGitLabToken(c.Request.Context(), req.PersonalAccessToken, req.InstanceURL)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": err.Error()})
		return
	}

	if !valid {
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": "Invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true, "message": "GitLab connection successful"})
}

// TestCodeRabbitConnection handles POST /api/auth/coderabbit/test
func TestCodeRabbitConnection(c *gin.Context) {
	var req struct {
		APIKey string `json:"apiKey" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	valid, err := ValidateCodeRabbitAPIKey(c.Request.Context(), req.APIKey)
	if err != nil {
		log.Printf("CodeRabbit API key validation failed: %v", err)
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": "Failed to validate API key with CodeRabbit"})
		return
	}

	if !valid {
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": "Invalid API key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true, "message": "CodeRabbit connection successful"})
}
