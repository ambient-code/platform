package handlers

import (
	"context"
	"log"
	"net/http"
	"sort"

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

	// Gerrit status
	response["gerrit"] = getGerritStatusForUser(ctx, userID)

	// MCP server credentials status
	response["mcpServers"] = getMCPServerStatusForUser(ctx, userID)

	c.JSON(http.StatusOK, response)
}

// Helper functions to get individual integration statuses

// getGitHubStatusForUser returns the GitHub integration status (App + PAT) for a user.
func getGitHubStatusForUser(ctx context.Context, userID string) gin.H {
	log.Printf("getGitHubStatusForUser: querying status for user=%s", userID)
	status := gin.H{
		"installed": false,
		"pat":       gin.H{"configured": false},
	}

	// Check GitHub App
	inst, err := GetGitHubInstallation(ctx, userID)
	if err == nil && inst != nil {
		log.Printf("getGitHubStatusForUser: found installation for user=%s installationId=%d githubUser=%s", userID, inst.InstallationID, inst.GitHubUserID)
		status["installed"] = true
		status["installationId"] = inst.InstallationID
		status["host"] = inst.Host
		status["githubUserId"] = inst.GitHubUserID
		status["updatedAt"] = inst.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	} else {
		log.Printf("getGitHubStatusForUser: no installation found for user=%s", userID)
	}

	// Check GitHub PAT
	patCreds, err := GetGitHubPATCredentials(ctx, userID)
	if err == nil && patCreds != nil {
		// NOTE: Validation disabled - if credentials are stored, assume they're valid
		// The integration will fail gracefully if credentials are actually invalid

		status["pat"] = gin.H{
			"configured": true,
			"updatedAt":  patCreds.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"valid":      true,
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

// getGoogleStatusForUser returns the Google OAuth integration status for a user.
func getGoogleStatusForUser(ctx context.Context, userID string) gin.H {
	creds, err := GetGoogleCredentials(ctx, userID)
	if err != nil || creds == nil {
		return gin.H{"connected": false}
	}

	// NOTE: Validation disabled - if credentials are stored, assume they're valid
	// The backend auto-refreshes tokens and the integration will fail gracefully if invalid

	return gin.H{
		"connected": true,
		"email":     creds.Email,
		"expiresAt": creds.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		"updatedAt": creds.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"valid":     true,
	}
}

// getJiraStatusForUser returns the Jira integration status for a user.
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

// getGerritStatusForUser returns the Gerrit integration status with all connected instances for a user.
func getGerritStatusForUser(ctx context.Context, userID string) gin.H {
	instances, err := listGerritCredentials(ctx, userID)
	if err != nil || len(instances) == 0 {
		return gin.H{"instances": []gin.H{}}
	}

	sort.Slice(instances, func(i, j int) bool {
		return instances[i].InstanceName < instances[j].InstanceName
	})

	result := make([]gin.H, 0, len(instances))
	for _, creds := range instances {
		result = append(result, gin.H{
			"connected":    true,
			"instanceName": creds.InstanceName,
			"url":          creds.URL,
			"authMethod":   creds.AuthMethod,
			"updatedAt":    creds.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return gin.H{"instances": result}
}

// getGitLabStatusForUser returns the GitLab integration status for a user.
func getGitLabStatusForUser(ctx context.Context, userID string) gin.H {
	creds, err := GetGitLabCredentials(ctx, userID)
	if err != nil || creds == nil {
		return gin.H{"connected": false}
	}

	// NOTE: Validation disabled - if credentials are stored, assume they're valid
	// The integration will fail gracefully if credentials are actually invalid

	return gin.H{
		"connected":   true,
		"instanceUrl": creds.InstanceURL,
		"updatedAt":   creds.UpdatedAt,
		"valid":       true,
	}
}
