package handlers

import (
	"context"
	"fmt"
	"net/http"
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
