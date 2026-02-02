package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ambient-code-backend/git"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
)

// GetGitHubTokenForSession handles GET /api/projects/:project/agentic-sessions/:session/credentials/github
// Returns PAT (priority 1) or freshly minted GitHub App token (priority 2)
func GetGitHubTokenForSession(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	// Get user-scoped K8s client
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	// Get userID from session CR
	gvr := GetAgenticSessionV1Alpha1Resource()
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get session %s/%s: %v", project, session, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	// Extract userID from spec.userContext using type-safe unstructured helpers
	userID, found, err := unstructured.NestedString(obj.Object, "spec", "userContext", "userId")
	if !found || err != nil || userID == "" {
		log.Printf("Failed to extract userID from session %s/%s: found=%v, err=%v", project, session, found, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in session"})
		return
	}

	// Verify authenticated user owns this session (RBAC: prevent accessing other users' credentials)
	// Note: BOT_TOKEN (session ServiceAccount) won't have userID in context, which is fine -
	// BOT_TOKEN is already scoped to this specific session via RBAC
	authenticatedUserID := c.GetString("userID")
	if authenticatedUserID != "" && authenticatedUserID != userID {
		log.Printf("RBAC violation: user %s attempted to access credentials for session owned by %s", authenticatedUserID, userID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: session belongs to different user"})
		return
	}
	// If authenticatedUserID is empty, this is likely BOT_TOKEN (session-scoped ServiceAccount)
	// which is allowed because it's already restricted to this session via K8s RBAC

	// Try to get GitHub token using standard precedence (PAT > App > project fallback)
	// Need to convert K8sClient interface to *kubernetes.Clientset for git.GetGitHubToken
	k8sClientset, ok := K8sClient.(*kubernetes.Clientset)
	if !ok {
		log.Printf("Failed to convert K8sClient to *kubernetes.Clientset")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
		return
	}

	token, err := git.GetGitHubToken(c.Request.Context(), k8sClientset, DynamicClient, project, userID)
	if err != nil {
		log.Printf("Failed to get GitHub token for user %s: %v", userID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// GetGoogleCredentialsForSession handles GET /api/projects/:project/agentic-sessions/:session/credentials/google
// Returns fresh Google OAuth credentials (refreshes if needed)
func GetGoogleCredentialsForSession(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	// Get user-scoped K8s client
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	// Get userID from session CR
	gvr := GetAgenticSessionV1Alpha1Resource()
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get session %s/%s: %v", project, session, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	// Extract userID from spec.userContext using type-safe unstructured helpers
	userID, found, err := unstructured.NestedString(obj.Object, "spec", "userContext", "userId")
	if !found || err != nil || userID == "" {
		log.Printf("Failed to extract userID from session %s/%s: found=%v, err=%v", project, session, found, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in session"})
		return
	}

	// Verify authenticated user owns this session (RBAC: prevent accessing other users' credentials)
	// Note: BOT_TOKEN (session ServiceAccount) won't have userID in context, which is fine -
	// BOT_TOKEN is already scoped to this specific session via RBAC
	authenticatedUserID := c.GetString("userID")
	if authenticatedUserID != "" && authenticatedUserID != userID {
		log.Printf("RBAC violation: user %s attempted to access credentials for session owned by %s", authenticatedUserID, userID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: session belongs to different user"})
		return
	}
	// If authenticatedUserID is empty, this is likely BOT_TOKEN (session-scoped ServiceAccount)
	// which is allowed because it's already restricted to this session via K8s RBAC

	// Get Google credentials from cluster storage
	creds, err := GetGoogleCredentials(c.Request.Context(), userID)
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Google credentials not configured"})
			return
		}
		log.Printf("Failed to get Google credentials for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Google credentials"})
		return
	}

	if creds == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Google credentials not configured"})
		return
	}

	// Check if token needs refresh
	needsRefresh := time.Now().After(creds.ExpiresAt.Add(-5 * time.Minute)) // Refresh 5min before expiry

	if needsRefresh && creds.RefreshToken != "" {
		// Refresh the token
		log.Printf("Google token expired for user %s, refreshing...", userID)
		newCreds, err := refreshGoogleAccessToken(c.Request.Context(), creds)
		if err != nil {
			log.Printf("Failed to refresh Google token for user %s: %v", userID, err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Google token expired and refresh failed. Please re-authenticate."})
			return
		}
		creds = newCreds
		log.Printf("✓ Refreshed Google token for user %s", userID)
	}

	c.JSON(http.StatusOK, gin.H{
		"accessToken": creds.AccessToken,
		"email":       creds.Email,
		"scopes":      creds.Scopes,
		"expiresAt":   creds.ExpiresAt.Format(time.RFC3339),
	})
}

// GetJiraCredentialsForSession handles GET /api/projects/:project/agentic-sessions/:session/credentials/jira
// Returns Jira credentials for the session's user
func GetJiraCredentialsForSession(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	// Get user-scoped K8s client
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	// Get userID from session CR
	gvr := GetAgenticSessionV1Alpha1Resource()
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get session %s/%s: %v", project, session, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	// Extract userID from spec.userContext using type-safe unstructured helpers
	userID, found, err := unstructured.NestedString(obj.Object, "spec", "userContext", "userId")
	if !found || err != nil || userID == "" {
		log.Printf("Failed to extract userID from session %s/%s: found=%v, err=%v", project, session, found, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in session"})
		return
	}

	// Verify authenticated user owns this session (RBAC: prevent accessing other users' credentials)
	// Note: BOT_TOKEN (session ServiceAccount) won't have userID in context, which is fine -
	// BOT_TOKEN is already scoped to this specific session via RBAC
	authenticatedUserID := c.GetString("userID")
	if authenticatedUserID != "" && authenticatedUserID != userID {
		log.Printf("RBAC violation: user %s attempted to access credentials for session owned by %s", authenticatedUserID, userID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: session belongs to different user"})
		return
	}
	// If authenticatedUserID is empty, this is likely BOT_TOKEN (session-scoped ServiceAccount)
	// which is allowed because it's already restricted to this session via K8s RBAC

	// Get Jira credentials
	creds, err := GetJiraCredentials(c.Request.Context(), userID)
	if err != nil {
		log.Printf("Failed to get Jira credentials for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Jira credentials"})
		return
	}

	if creds == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Jira credentials not configured"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":      creds.URL,
		"email":    creds.Email,
		"apiToken": creds.APIToken,
	})
}

// GetGitLabTokenForSession handles GET /api/projects/:project/agentic-sessions/:session/credentials/gitlab
// Returns GitLab token for the session's user
func GetGitLabTokenForSession(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	// Get user-scoped K8s client
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	// Get userID from session CR
	gvr := GetAgenticSessionV1Alpha1Resource()
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get session %s/%s: %v", project, session, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	// Extract userID from spec.userContext using type-safe unstructured helpers
	userID, found, err := unstructured.NestedString(obj.Object, "spec", "userContext", "userId")
	if !found || err != nil || userID == "" {
		log.Printf("Failed to extract userID from session %s/%s: found=%v, err=%v", project, session, found, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in session"})
		return
	}

	// Verify authenticated user owns this session (RBAC: prevent accessing other users' credentials)
	// Note: BOT_TOKEN (session ServiceAccount) won't have userID in context, which is fine -
	// BOT_TOKEN is already scoped to this specific session via RBAC
	authenticatedUserID := c.GetString("userID")
	if authenticatedUserID != "" && authenticatedUserID != userID {
		log.Printf("RBAC violation: user %s attempted to access credentials for session owned by %s", authenticatedUserID, userID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: session belongs to different user"})
		return
	}
	// If authenticatedUserID is empty, this is likely BOT_TOKEN (session-scoped ServiceAccount)
	// which is allowed because it's already restricted to this session via K8s RBAC

	// Get GitLab credentials
	creds, err := GetGitLabCredentials(c.Request.Context(), userID)
	if err != nil {
		log.Printf("Failed to get GitLab credentials for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get GitLab credentials"})
		return
	}

	if creds == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "GitLab credentials not configured"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":       creds.Token,
		"instanceUrl": creds.InstanceURL,
	})
}

// refreshGoogleAccessToken refreshes a Google OAuth access token using the refresh token
func refreshGoogleAccessToken(ctx context.Context, oldCreds *GoogleOAuthCredentials) (*GoogleOAuthCredentials, error) {
	if oldCreds.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	// Get OAuth provider config
	provider, err := getOAuthProvider("google")
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth provider: %w", err)
	}

	// Call Google's token refresh endpoint
	tokenURL := "https://oauth2.googleapis.com/token"
	payload := map[string]string{
		"client_id":     provider.ClientID,
		"client_secret": provider.ClientSecret,
		"refresh_token": oldCreds.RefreshToken,
		"grant_type":    "refresh_token",
	}

	tokenData, err := exchangeOAuthToken(ctx, tokenURL, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update credentials with new token
	newCreds := &GoogleOAuthCredentials{
		UserID:       oldCreds.UserID,
		Email:        oldCreds.Email,
		AccessToken:  tokenData.AccessToken,
		RefreshToken: oldCreds.RefreshToken, // Reuse existing refresh token
		Scopes:       oldCreds.Scopes,
		ExpiresAt:    time.Now().Add(time.Duration(tokenData.ExpiresIn) * time.Second),
		UpdatedAt:    time.Now(),
	}

	// Store updated credentials
	if err := storeGoogleCredentials(ctx, newCreds); err != nil {
		return nil, fmt.Errorf("failed to store refreshed credentials: %w", err)
	}

	return newCreds, nil
}

// exchangeOAuthToken makes a token exchange request to an OAuth provider
func exchangeOAuthToken(ctx context.Context, tokenURL string, payload map[string]string) (*OAuthTokenResponse, error) {
	// Convert map to form data
	form := url.Values{}
	for k, v := range payload {
		form.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var tokenResp OAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tokenResp, nil
}
