package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

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
		// Don't wrap error - could leak token from request details
		return false, fmt.Errorf("request failed")
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
		// Don't wrap error - could leak token from request details
		return false, fmt.Errorf("request failed")
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

	for _, apiURL := range apiURLs {
		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if err != nil {
			continue
		}

		// Jira uses Basic Auth with email:token
		req.SetBasicAuth(email, apiToken)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
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
		// Don't wrap error - could leak token from request details
		return false, fmt.Errorf("request failed")
	}
	defer resp.Body.Close()

	// 200 = valid, 401 = invalid/expired
	return resp.StatusCode == http.StatusOK, nil
}

// ValidateGerritToken checks if Gerrit credentials are valid
// Uses /a/accounts/self endpoint which accepts Basic Auth or Cookie-based auth
// Gerrit REST API responses are prefixed with )]}'  (XSSI protection)
func ValidateGerritToken(ctx context.Context, gerritURL, authMethod, username, httpToken, gitcookiesContent string) (bool, error) {
	if gerritURL == "" {
		return false, fmt.Errorf("Gerrit URL is required")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	apiURL := fmt.Sprintf("%s/a/accounts/self", gerritURL)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request")
	}
	req.Header.Set("Accept", "application/json")

	switch authMethod {
	case "http_basic":
		if username == "" || httpToken == "" {
			return false, fmt.Errorf("username and HTTP token are required for HTTP basic auth")
		}
		req.SetBasicAuth(username, httpToken)

	case "git_cookies":
		if gitcookiesContent == "" {
			return false, fmt.Errorf("gitcookies content is required")
		}
		// Parse gitcookies content to extract cookie for the target host
		cookie := parseGitcookies(gerritURL, gitcookiesContent)
		if cookie == "" {
			return false, fmt.Errorf("no matching cookie found for host in gitcookies content")
		}
		req.Header.Set("Cookie", cookie)

	default:
		return false, fmt.Errorf("unsupported auth method: %s", authMethod)
	}

	resp, err := client.Do(req)
	if err != nil {
		// Don't wrap error - could leak credentials from request details
		return false, fmt.Errorf("request failed")
	}
	defer resp.Body.Close()

	// 200 = valid, 401/403 = invalid
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return false, nil
	}

	// Other status codes - can't validate, assume valid to avoid false negatives
	return true, nil
}

// parseGitcookies extracts the cookie value for a given Gerrit URL from gitcookies content.
// Gitcookies format: host\tFALSE\t/\tTRUE\t2147483647\to\tvalue
func parseGitcookies(gerritURL, content string) string {
	parsed, err := url.Parse(gerritURL)
	if err != nil {
		return ""
	}
	host := parsed.Hostname()

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) >= 7 {
			cookieHost := strings.TrimPrefix(fields[0], ".")
			if cookieHost == host || strings.HasSuffix(host, "."+cookieHost) {
				return fields[5] + "=" + fields[6]
			}
		}
	}
	return ""
}

// TestGerritConnection handles POST /api/auth/gerrit/test
// Tests Gerrit credentials without saving them
func TestGerritConnection(c *gin.Context) {
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

	valid, err := ValidateGerritToken(c.Request.Context(), req.URL, req.AuthMethod, req.Username, req.HTTPToken, req.GitcookiesContent)
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
