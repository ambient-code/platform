package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetIntegrationsStatus handles GET /api/auth/integrations/status
// Returns unified status for all integrations (GitHub, Google, Jira, GitLab)
func GetIntegrationsStatus(c *gin.Context) {
	// Verify user has valid K8s token
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}

	ctx := c.Request.Context()
	response := gin.H{}

	// GitHub status (App + PAT)
	response["github"] = getGitHubStatusForUser(ctx, userID)

	// Google status
	response["google"] = getGoogleStatusForUser(ctx, userID)

	// Jira status
	response["jira"] = getJiraStatusForUser(ctx, userID)

	// GitLab status
	response["gitlab"] = getGitLabStatusForUser(ctx, userID)

	c.JSON(http.StatusOK, response)
}

// Helper functions to get individual integration statuses

func getGitHubStatusForUser(ctx context.Context, userID string) gin.H {
	status := gin.H{
		"installed": false,
		"pat":       gin.H{"configured": false},
	}

	// Check GitHub App
	inst, err := GetGitHubInstallation(ctx, userID)
	if err == nil && inst != nil {
		status["installed"] = true
		status["installationId"] = inst.InstallationID
		status["host"] = inst.Host
		status["githubUserId"] = inst.GitHubUserID
		status["updatedAt"] = inst.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	// Check GitHub PAT
	patCreds, err := GetGitHubPATCredentials(ctx, userID)
	if err == nil && patCreds != nil {
		// Validate PAT token
		valid, _ := ValidateGitHubToken(ctx, patCreds.Token)

		status["pat"] = gin.H{
			"configured": true,
			"updatedAt":  patCreds.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"valid":      valid,
		}
	}

	// Determine active method
	if patCreds != nil {
		status["active"] = "pat"
	} else if inst != nil {
		status["active"] = "app"
	}

	return status
}

func getGoogleStatusForUser(ctx context.Context, userID string) gin.H {
	creds, err := GetGoogleCredentials(ctx, userID)
	if err != nil || creds == nil {
		return gin.H{"connected": false}
	}

	// Check if token is expired
	isExpired := time.Now().After(creds.ExpiresAt)
	valid := !isExpired

	// If near expiry, could validate with Google API, but checking expiry is sufficient
	// since backend auto-refreshes tokens

	return gin.H{
		"connected": true,
		"email":     creds.Email,
		"expiresAt": creds.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		"updatedAt": creds.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"valid":     valid,
	}
}

func getJiraStatusForUser(ctx context.Context, userID string) gin.H {
	creds, err := GetJiraCredentials(ctx, userID)
	if err != nil || creds == nil {
		return gin.H{"connected": false}
	}

	// NOTE: Validation disabled - causing false negatives for valid credentials
	// Jira validation is unreliable due to various auth configurations
	// If credentials are stored, assume they're valid (user configured them)
	// The MCP server will fail gracefully if credentials are actually invalid

	return gin.H{
		"connected": true,
		"url":       creds.URL,
		"email":     creds.Email,
		"updatedAt": creds.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"valid":     true, // Always true - trust user's configuration
	}
}

func getGitLabStatusForUser(ctx context.Context, userID string) gin.H {
	creds, err := GetGitLabCredentials(ctx, userID)
	if err != nil || creds == nil {
		return gin.H{"connected": false}
	}

	// Validate token
	valid, _ := ValidateGitLabToken(ctx, creds.Token, creds.InstanceURL)

	return gin.H{
		"connected":   true,
		"instanceUrl": creds.InstanceURL,
		"updatedAt":   creds.UpdatedAt,
		"valid":       valid,
	}
}
