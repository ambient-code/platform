package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"ambient-code-backend/git"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// identityAPITimeout is the HTTP client timeout for GitHub/GitLab user identity API calls.
const identityAPITimeout = 10 * time.Second

// sessionActiveUserMap tracks the authenticated user who initiated the current
// run for each session. Set by the AG-UI proxy (SetSessionActiveUser), read by
// enforceCredentialRBAC as a fallback when the caller token has expired but
// the BOT_TOKEN needs to fetch the non-owner user's credentials.
// Key: sessionName, Value: string (userID).
var sessionActiveUserMap sync.Map

// SetSessionActiveUser records the authenticated user for a session's active run.
// Key is project/session to avoid cross-namespace collisions.
func SetSessionActiveUser(project, sessionName, userID string) {
	sessionActiveUserMap.Store(project+"/"+sessionName, userID)
}

// ClearSessionActiveUser removes the active user entry for a session.
func ClearSessionActiveUser(project, sessionName string) {
	sessionActiveUserMap.Delete(project + "/" + sessionName)
}

// getEffectiveUserID determines which user's credentials should be used.
// Returns currentUserID from X-Runner-Current-User header if present,
// otherwise falls back to ownerUserID (session owner).
func getEffectiveUserID(c *gin.Context, ownerUserID string) string {
	currentUserID := c.GetHeader("X-Runner-Current-User")
	if currentUserID != "" {
		return currentUserID
	}
	return ownerUserID
}

// resolveAuthenticatedUser returns the authenticated user ID for the request.
// Prefers the middleware-resolved userID; falls back to SelfSubjectReview for
// forwarded caller tokens that bypass the OAuth proxy. Caches the result on
// the gin context to avoid repeated K8s API calls within the same request.
// Returns (userID, isBotToken). isBotToken is true only when the token is
// confirmed as a ServiceAccount (not a resolution failure).
func resolveAuthenticatedUser(c *gin.Context, reqK8s kubernetes.Interface) (string, bool) {
	if uid := c.GetString("userID"); uid != "" {
		return uid, false
	}
	if uid := c.GetString("resolvedUserID"); uid != "" {
		return uid, false
	}
	if reqK8s != nil {
		resolved, err := resolveTokenIdentity(c.Request.Context(), reqK8s)
		if err == nil {
			uid := strings.ReplaceAll(resolved, ":", "-")
			if strings.HasPrefix(uid, "system-serviceaccount-") {
				return "", true // confirmed BOT_TOKEN
			}
			c.Set("resolvedUserID", uid)
			return uid, false
		}
		// SelfSubjectReview failed. If no X-Forwarded-User was set by the
		// OAuth proxy, this is likely a ServiceAccount token in an environment
		// where SelfSubjectReview isn't available. Treat as BOT_TOKEN to avoid
		// breaking SA-authenticated callers.
		if c.GetHeader("X-Forwarded-User") == "" {
			log.Printf("SelfSubjectReview failed and no X-Forwarded-User — treating as BOT_TOKEN: %v", err)
			return "", true
		}
		log.Printf("SelfSubjectReview failed for user request, denying: %v", err)
		return "", false
	}
	return "", false
}

// enforceCredentialRBAC extracts the owner and effective user from the session
// CR, checks RBAC, and returns the effectiveUserID to use for credential lookup.
// Returns ("", false) if the request should be rejected.
func enforceCredentialRBAC(c *gin.Context, reqK8s kubernetes.Interface, reqDyn dynamic.Interface, project, session string) (string, bool) {
	gvr := GetAgenticSessionV1Alpha1Resource()
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		} else {
			log.Printf("Failed to get session %s/%s: %v", project, session, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		}
		return "", false
	}

	ownerUserID, found, err := unstructured.NestedString(obj.Object, "spec", "userContext", "userId")
	if !found || err != nil || ownerUserID == "" {
		log.Printf("Failed to extract userID from session %s/%s: found=%v, err=%v", project, session, found, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in session"})
		return "", false
	}

	effectiveUserID := getEffectiveUserID(c, ownerUserID)
	authenticatedUserID, isBotToken := resolveAuthenticatedUser(c, reqK8s)

	// RBAC: BOT_TOKEN → owner OR active run user; user token → must match effective user
	allowed := false
	if isBotToken {
		// Confirmed ServiceAccount: allow owner creds always. For non-owner,
		// check that the requested user matches the active user set by the
		// AG-UI proxy (supports fallback when caller token expires mid-run).
		if effectiveUserID == ownerUserID {
			allowed = true
		} else if activeUser, ok := sessionActiveUserMap.Load(project + "/" + session); ok {
			if activeUserStr, ok := activeUser.(string); ok {
				allowed = effectiveUserID == activeUserStr
			}
		}
	} else if authenticatedUserID == "" {
		// Resolution failed (not BOT_TOKEN, not a resolved user) — deny
		allowed = false
	} else if effectiveUserID == ownerUserID {
		allowed = authenticatedUserID == ownerUserID
	} else {
		allowed = authenticatedUserID == effectiveUserID
	}

	if !allowed {
		log.Printf("RBAC violation: user %s attempted access (owner=%s, current=%s)", authenticatedUserID, ownerUserID, effectiveUserID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return "", false
	}

	log.Printf("CREDENTIAL_ACCESS: session=%s/%s same_as_owner=%t", project, session, effectiveUserID == ownerUserID)
	return effectiveUserID, true
}

// GetGitHubTokenForSession handles GET /api/projects/:project/agentic-sessions/:session/credentials/github
// Returns PAT (priority 1) or freshly minted GitHub App token (priority 2)
func GetGitHubTokenForSession(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	effectiveUserID, ok := enforceCredentialRBAC(c, reqK8s, reqDyn, project, session)
	if !ok {
		return
	}

	// Try to get GitHub token using standard precedence (PAT > App > project fallback)
	// Need to convert K8sClient interface to *kubernetes.Clientset for git.GetGitHubToken
	k8sClientset, ok := K8sClient.(*kubernetes.Clientset)
	if !ok {
		log.Printf("Failed to convert K8sClient to *kubernetes.Clientset")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
		return
	}

	token, expiresAt, err := git.GetGitHubToken(c.Request.Context(), k8sClientset, DynamicClient, project, effectiveUserID)
	if err != nil {
		log.Printf("Failed to get GitHub token for user %s: %v", effectiveUserID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Fetch user identity from GitHub API for git config
	// Fix for: GitHub credentials aren't mounted to session - need git identity
	userName, userEmail := fetchGitHubUserIdentity(c.Request.Context(), token)
	if userName != "" {
		log.Printf("Returning GitHub credentials with identity for session %s/%s", project, session)
	}

	resp := gin.H{
		"token":    token,
		"userName": userName,
		"email":    userEmail,
		"provider": "github",
	}
	if !expiresAt.IsZero() {
		resp["expiresAt"] = expiresAt.Format(time.RFC3339)
	}
	c.JSON(http.StatusOK, resp)
}

// GetGoogleCredentialsForSession handles GET /api/projects/:project/agentic-sessions/:session/credentials/google
// Returns fresh Google OAuth credentials (refreshes if needed)
func GetGoogleCredentialsForSession(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	effectiveUserID, ok := enforceCredentialRBAC(c, reqK8s, reqDyn, project, session)
	if !ok {
		return
	}

	// Get Google credentials from cluster storage
	creds, err := GetGoogleCredentials(c.Request.Context(), effectiveUserID)
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Google credentials not configured"})
			return
		}
		log.Printf("Failed to get Google credentials for user %s: %v", effectiveUserID, err)
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
		log.Printf("Google token expired for user %s, refreshing...", effectiveUserID)
		newCreds, err := refreshGoogleAccessToken(c.Request.Context(), creds)
		if err != nil {
			log.Printf("Failed to refresh Google token for user %s: %v", effectiveUserID, err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Google token expired and refresh failed. Please re-authenticate."})
			return
		}
		creds = newCreds
		log.Printf("✓ Refreshed Google token for user %s", effectiveUserID)
	}

	c.JSON(http.StatusOK, gin.H{
		"accessToken":  creds.AccessToken,
		"refreshToken": creds.RefreshToken,
		"email":        creds.Email,
		"scopes":       creds.Scopes,
		"expiresAt":    creds.ExpiresAt.Format(time.RFC3339),
	})
}

// GetJiraCredentialsForSession handles GET /api/projects/:project/agentic-sessions/:session/credentials/jira
// Returns Jira credentials for the session's user
func GetJiraCredentialsForSession(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	effectiveUserID, ok := enforceCredentialRBAC(c, reqK8s, reqDyn, project, session)
	if !ok {
		return
	}

	// Get Jira credentials
	creds, err := GetJiraCredentials(c.Request.Context(), effectiveUserID)
	if err != nil {
		log.Printf("Failed to get Jira credentials for user %s: %v", effectiveUserID, err)
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

	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	effectiveUserID, ok := enforceCredentialRBAC(c, reqK8s, reqDyn, project, session)
	if !ok {
		return
	}

	// Get GitLab credentials
	creds, err := GetGitLabCredentials(c.Request.Context(), effectiveUserID)
	if err != nil {
		log.Printf("Failed to get GitLab credentials for user %s: %v", effectiveUserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get GitLab credentials"})
		return
	}

	if creds == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "GitLab credentials not configured"})
		return
	}

	// Fetch user identity from GitLab API for git config
	// Fix for: need to distinguish between GitHub and GitLab providers
	userName, userEmail := fetchGitLabUserIdentity(c.Request.Context(), creds.Token, creds.InstanceURL)
	if userName != "" {
		log.Printf("Returning GitLab credentials with identity for session %s/%s", project, session)
	}

	c.JSON(http.StatusOK, gin.H{
		"token":       creds.Token,
		"instanceUrl": creds.InstanceURL,
		"userName":    userName,
		"email":       userEmail,
		"provider":    "gitlab",
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

	client := &http.Client{Timeout: identityAPITimeout}
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

// fetchGitHubUserIdentity fetches user name and email from GitHub API
// Returns the user's name (or login as fallback) and email for git config
func fetchGitHubUserIdentity(ctx context.Context, token string) (userName, email string) {
	if token == "" {
		return "", ""
	}

	if ctx.Err() != nil {
		return "", ""
	}

	client := &http.Client{Timeout: identityAPITimeout}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		log.Printf("Failed to create GitHub user request: %v", err)
		return "", ""
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to fetch GitHub user: %v", err)
		return "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		if resp.StatusCode == http.StatusForbidden {
			log.Printf("GitHub API /user returned 403 (token may lack 'read:user' scope): %s", string(errBody))
		} else {
			log.Printf("GitHub API /user returned status %d: %s", resp.StatusCode, string(errBody))
		}
		return "", ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read GitHub user response: %v", err)
		return "", ""
	}

	var ghUser struct {
		Login string `json:"login"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &ghUser); err != nil {
		log.Printf("Failed to parse GitHub user response: %v", err)
		return "", ""
	}

	// Use Name if available, fall back to Login
	userName = ghUser.Name
	if userName == "" {
		userName = ghUser.Login
	}
	email = ghUser.Email

	log.Printf("Fetched GitHub user identity: name=%q hasEmail=%t", userName, email != "")
	return userName, email
}

// GetGerritCredentialsForSession handles GET /api/projects/:project/agentic-sessions/:session/credentials/gerrit
// Returns all Gerrit instances' credentials for the session's user
func GetGerritCredentialsForSession(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	effectiveUserID, ok := enforceCredentialRBAC(c, reqK8s, reqDyn, project, session)
	if !ok {
		return
	}

	// Get all Gerrit instances for the user
	instances, err := listGerritCredentials(c.Request.Context(), effectiveUserID)
	if err != nil {
		log.Printf("Failed to get Gerrit credentials for user %s: %v", effectiveUserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Gerrit credentials"})
		return
	}

	if len(instances) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Gerrit credentials not configured"})
		return
	}

	result := make([]gin.H, 0, len(instances))
	for _, creds := range instances {
		instance := gin.H{
			"instanceName": creds.InstanceName,
			"url":          creds.URL,
			"authMethod":   creds.AuthMethod,
		}
		if creds.AuthMethod == "http_basic" {
			instance["username"] = creds.Username
			instance["httpToken"] = creds.HTTPToken
		} else if creds.AuthMethod == "git_cookies" {
			instance["gitcookiesContent"] = creds.GitcookiesContent
		}
		result = append(result, instance)
	}

	c.JSON(http.StatusOK, gin.H{"instances": result})
}

// fetchGitLabUserIdentity fetches user name and email from GitLab API
// Returns the user's name and email for git config
func fetchGitLabUserIdentity(ctx context.Context, token, instanceURL string) (userName, email string) {
	if token == "" {
		return "", ""
	}

	if ctx.Err() != nil {
		return "", ""
	}

	// Default to gitlab.com if no instance URL
	apiURL := "https://gitlab.com/api/v4/user"
	if instanceURL != "" && instanceURL != "https://gitlab.com" {
		apiURL = strings.TrimSuffix(instanceURL, "/") + "/api/v4/user"
	}

	client := &http.Client{Timeout: identityAPITimeout}
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		log.Printf("Failed to create GitLab user request: %v", err)
		return "", ""
	}

	req.Header.Set("PRIVATE-TOKEN", token)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to fetch GitLab user: %v", err)
		return "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		log.Printf("GitLab API /user returned status %d: %s", resp.StatusCode, string(errBody))
		return "", ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read GitLab user response: %v", err)
		return "", ""
	}

	var glUser struct {
		Username string `json:"username"`
		Name     string `json:"name"`
		Email    string `json:"email"`
	}
	if err := json.Unmarshal(body, &glUser); err != nil {
		log.Printf("Failed to parse GitLab user response: %v", err)
		return "", ""
	}

	// Use Name if available, fall back to Username
	userName = glUser.Name
	if userName == "" {
		userName = glUser.Username
	}
	email = glUser.Email

	log.Printf("Fetched GitLab user identity: name=%q hasEmail=%t", userName, email != "")
	return userName, email
}
