package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// OAuthProvider represents a generic OAuth provider configuration
type OAuthProvider struct {
	Name         string
	ClientID     string
	ClientSecret string
	TokenURL     string
	Scopes       []string
}

// OAuthCallbackData represents the stored OAuth callback data
type OAuthCallbackData struct {
	Provider     string    `json:"provider"`
	UserID       string    `json:"userId"`
	Code         string    `json:"code"`
	State        string    `json:"state"`
	Error        string    `json:"error,omitempty"`
	ErrorDesc    string    `json:"error_description,omitempty"`
	ReceivedAt   time.Time `json:"receivedAt"`
	Consumed     bool      `json:"consumed"`
	ConsumedAt   time.Time `json:"consumedAt,omitempty"`
	AccessToken  string    `json:"accessToken,omitempty"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	ExpiresIn    int64     `json:"expiresIn,omitempty"`
	TokenType    string    `json:"tokenType,omitempty"`
}

// getOAuthProvider retrieves OAuth provider configuration from environment
func getOAuthProvider(provider string) (*OAuthProvider, error) {
	provider = strings.ToLower(provider)

	switch provider {
	case "google":
		clientID := os.Getenv("GOOGLE_OAUTH_CLIENT_ID")
		clientSecret := os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET")
		if clientID == "" || clientSecret == "" {
			return nil, fmt.Errorf("google oauth not configured")
		}
		return &OAuthProvider{
			Name:         "google",
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TokenURL:     "https://oauth2.googleapis.com/token",
			Scopes: []string{
				"openid",
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
				"https://www.googleapis.com/auth/drive",
				"https://www.googleapis.com/auth/drive.readonly",
				"https://www.googleapis.com/auth/drive.file",
			},
		}, nil

	case "github":
		// GitHub uses existing handler, but we can support it here too
		clientID := os.Getenv("GITHUB_CLIENT_ID")
		clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
		if clientID == "" || clientSecret == "" {
			return nil, fmt.Errorf("GitHub OAuth not configured")
		}
		return &OAuthProvider{
			Name:         "github",
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TokenURL:     "https://github.com/login/oauth/access_token",
			Scopes:       []string{"repo", "user"},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}
}

// GetOAuthURL handles GET /api/projects/:projectName/agentic-sessions/:sessionName/oauth/:provider/url
// Returns the OAuth URL for the frontend to open in a popup with HMAC-signed state parameter
func GetOAuthURL(c *gin.Context) {
	projectName := c.Param("projectName")
	sessionName := c.Param("sessionName")
	providerName := c.Param("provider")

	// Default to google if not specified
	if providerName == "" {
		providerName = "google"
	}

	// Verify user has access to the session using user token
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	// Verify session exists and user has access
	gvr := schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1alpha1",
		Resource: "agenticsessions",
	}

	_, err := reqDyn.Resource(gvr).Namespace(projectName).Get(context.Background(), sessionName, v1.GetOptions{})
	if errors.IsNotFound(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}
	if errors.IsForbidden(err) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to session"})
		return
	}
	if err != nil {
		log.Printf("Failed to get session %s/%s: %v", projectName, sessionName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify session"})
		return
	}

	// Get OAuth provider config
	provider, err := getOAuthProvider(providerName)
	if err != nil {
		log.Printf("Failed to get OAuth provider: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": fmt.Sprintf("%s OAuth not configured", providerName)})
		return
	}

	// Build state with session context
	stateData := OAuthStateData{
		Provider:    providerName,
		ProjectName: projectName,
		SessionName: sessionName,
		Timestamp:   time.Now().Unix(),
	}

	// Serialize state to JSON
	stateJSON, err := json.Marshal(stateData)
	if err != nil {
		log.Printf("Failed to marshal state: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OAuth state"})
		return
	}

	// Get HMAC secret from environment
	secret := os.Getenv("OAUTH_STATE_SECRET")
	if secret == "" {
		log.Printf("OAUTH_STATE_SECRET not configured")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth state validation not configured"})
		return
	}

	// Generate HMAC signature
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(stateJSON)
	signature := h.Sum(nil)

	// Combine: base64(json) + "." + base64(signature)
	stateToken := base64.URLEncoding.EncodeToString(stateJSON) + "." + base64.URLEncoding.EncodeToString(signature)

	// Get backend URL for redirect URI
	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://localhost:8080"
	}
	redirectURI := fmt.Sprintf("%s/oauth2callback", backendURL)

	// Build OAuth URL with signed state
	var authURL string
	switch providerName {
	case "google":
		authURL = fmt.Sprintf(
			"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&access_type=offline&state=%s&prompt=consent",
			provider.ClientID,
			redirectURI,
			strings.Join(provider.Scopes, " "),
			stateToken,
		)
	case "github":
		authURL = fmt.Sprintf(
			"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
			provider.ClientID,
			redirectURI,
			strings.Join(provider.Scopes, " "),
			stateToken,
		)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unsupported provider: %s", providerName)})
		return
	}

	log.Printf("Generated OAuth URL for %s/%s (provider: %s, stateLen: %d)", projectName, sessionName, providerName, len(stateToken))

	c.JSON(http.StatusOK, gin.H{
		"url":   authURL,
		"state": stateToken,
	})
}

// HandleOAuth2Callback handles GET /oauth2callback
// This is a generic OAuth2 callback endpoint that can handle multiple providers
func HandleOAuth2Callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")
	errorDesc := c.Query("error_description")
	provider := c.Query("provider")

	// Default to google if no provider specified (for MCP compatibility)
	if provider == "" {
		provider = "google"
	}

	log.Printf("OAuth2 callback received - provider: %s, hasCode: %v, hasState: %v, error: %s",
		provider, code != "", state != "", errorParam)

	// Handle OAuth errors early
	if errorParam != "" {
		log.Printf("OAuth error received: %s - %s", errorParam, errorDesc)
		callbackData := OAuthCallbackData{
			Provider:   provider,
			Code:       code,
			State:      state,
			Error:      errorParam,
			ErrorDesc:  errorDesc,
			ReceivedAt: time.Now(),
			Consumed:   false,
		}
		// Store the error for MCP to retrieve
		if err := storeOAuthCallback(c.Request.Context(), state, &callbackData); err != nil {
			log.Printf("Failed to store OAuth error: %v", err)
		}
		c.HTML(http.StatusOK, "<html><body><h1>Authorization Error</h1><p>Error: "+errorParam+"</p><p>"+errorDesc+"</p><p>Provider: "+provider+"</p><p>You can close this window.</p></body></html>", nil)
		return
	}

	// Validate required parameters
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing authorization code"})
		return
	}

	// IMPORTANT: Check for cluster-level OAuth BEFORE exchanging the code
	// Authorization codes are single-use, so we must route to the correct handler first
	var stateMap map[string]interface{}
	stateBytes, err := base64.URLEncoding.DecodeString(strings.Split(state, ".")[0])
	if err == nil {
		if jsonErr := json.Unmarshal(stateBytes, &stateMap); jsonErr == nil {
			// Check if this is cluster-level OAuth
			if isCluster, ok := stateMap["cluster"].(bool); ok && isCluster {
				log.Printf("Detected cluster-level OAuth flow")

				// Handle cluster-level Google OAuth (this will exchange the code)
				if err := HandleGoogleOAuthCallback(c.Request.Context(), code, stateMap); err != nil {
					log.Printf("Cluster-level OAuth failed: %v", err)
					// Return generic error to client, details logged server-side only
					c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(
						"<html><body><h1>Authorization Error</h1><p>Failed to connect Google Drive. Please try again.</p><p>You can close this window.</p><script>window.close();</script></body></html>",
					))
					return
				}

				// Success
				c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(
					"<html><body><h1>Authorization Successful!</h1><p>Google Drive is now connected!</p><p>All your sessions will have access to Google Drive.</p><p>You can close this window.</p><script>window.close();</script></body></html>",
				))
				return
			}
		}
	}

	// Legacy session-specific OAuth flow
	// Create callback data record
	callbackData := OAuthCallbackData{
		Provider:   provider,
		Code:       code,
		State:      state,
		Error:      errorParam,
		ErrorDesc:  errorDesc,
		ReceivedAt: time.Now(),
		Consumed:   false,
	}

	// Try to get user ID from session (may not be available for MCP flows)
	if userID, exists := c.Get("userID"); exists && userID != nil {
		callbackData.UserID = userID.(string)
	}

	// Get provider configuration
	providerConfig, err := getOAuthProvider(provider)
	if err != nil {
		log.Printf("Failed to get OAuth provider config: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth provider not configured"})
		return
	}

	// Get backend URL for redirect URI (must match the one used in auth URL)
	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://localhost:8080"
	}
	redirectURI := fmt.Sprintf("%s/oauth2callback", backendURL)

	// Exchange code for token (for legacy session-specific flow)
	tokenData, err := exchangeOAuthCode(c.Request.Context(), providerConfig, code, redirectURI)
	if err != nil {
		log.Printf("Failed to exchange OAuth code: %v", err)
		callbackData.Error = "token_exchange_failed"
		callbackData.ErrorDesc = err.Error()
		// Store the failure
		if serr := storeOAuthCallback(c.Request.Context(), state, &callbackData); serr != nil {
			log.Printf("Failed to store OAuth exchange error: %v", serr)
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to exchange authorization code"})
		return
	}

	// Populate token data
	callbackData.AccessToken = tokenData.AccessToken
	callbackData.RefreshToken = tokenData.RefreshToken
	callbackData.ExpiresIn = tokenData.ExpiresIn
	callbackData.TokenType = tokenData.TokenType

	// Fallback to legacy session-specific OAuth
	stateData, err := validateAndParseOAuthState(state)
	if err != nil {
		log.Printf("ERROR: State validation failed: %v (possible CSRF attack or tampering)", err)
		// DO NOT store credentials or proceed - this is a security violation
		c.Data(http.StatusForbidden, "text/html; charset=utf-8", []byte(
			"<html><body><h1>Authorization Failed</h1><p>Provider: "+provider+"</p><p><strong>Error:</strong> Invalid or expired state parameter. This may indicate a CSRF attack or session timeout.</p><p>Please try again from the beginning.</p><p>You can close this window.</p><script>window.close();</script></body></html>",
		))
		return
	}

	// Store credentials in Kubernetes Secret in the project namespace (legacy session-specific)
	if stateData.ProjectName != "" && stateData.SessionName != "" {
		err := storeCredentialsInSecret(
			c.Request.Context(),
			stateData.ProjectName,
			stateData.SessionName,
			provider,
			tokenData.AccessToken,
			tokenData.RefreshToken,
			tokenData.ExpiresIn,
		)
		if err != nil {
			log.Printf("Failed to store credentials in Secret: %v", err)
			c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(
				"<html><body><h1>Authorization Error</h1><p>Provider: "+provider+"</p><p><strong>Error:</strong> Failed to store credentials. Please contact support.</p><p>You can close this window.</p><script>window.close();</script></body></html>",
			))
			return
		}

		log.Printf("✓ OAuth flow completed for session %s/%s", stateData.ProjectName, stateData.SessionName)
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(
			"<html><body><h1>Authorization Successful!</h1><p>Provider: "+provider+"</p><p>Google Drive credentials are now available in your session!</p><p>You can close this window.</p><script>window.close();</script></body></html>",
		))
	} else {
		log.Printf("Warning: State missing session context (projectName=%s, sessionName=%s)", stateData.ProjectName, stateData.SessionName)
		// Fallback: store in oauth-callbacks
		if err := storeOAuthCallback(c.Request.Context(), state, &callbackData); err != nil {
			log.Printf("Failed to store OAuth callback: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store OAuth data"})
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(
			"<html><body><h1>Authorization Successful!</h1><p>Provider: "+provider+"</p><p>You can close this window.</p><script>window.close();</script></body></html>",
		))
	}
}

// OAuthTokenResponse represents the token exchange response
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

// exchangeOAuthCode exchanges an authorization code for an access token
func exchangeOAuthCode(ctx context.Context, provider *OAuthProvider, code string, redirectURI string) (*OAuthTokenResponse, error) {
	// Prepare token exchange request
	formData := fmt.Sprintf("code=%s&client_id=%s&client_secret=%s&redirect_uri=%s&grant_type=authorization_code",
		code,
		provider.ClientID,
		provider.ClientSecret,
		redirectURI,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.TokenURL, strings.NewReader(formData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp OAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("empty access token received")
	}

	return &tokenResp, nil
}

// storeOAuthCallback stores OAuth callback data in a Secret for retrieval by MCP or other consumers
func storeOAuthCallback(ctx context.Context, state string, data *OAuthCallbackData) error {
	if state == "" {
		// Generate a default key if no state provided
		state = fmt.Sprintf("callback_%d", time.Now().Unix())
	}

	const secretName = "oauth-callbacks"

	for i := 0; i < 3; i++ { // retry on conflict
		secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// Create Secret
				secret = &corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      secretName,
						Namespace: Namespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{},
				}
				if _, cerr := K8sClient.CoreV1().Secrets(Namespace).Create(ctx, secret, v1.CreateOptions{}); cerr != nil && !errors.IsAlreadyExists(cerr) {
					return fmt.Errorf("failed to create Secret: %w", cerr)
				}
				// Fetch again to get resourceVersion
				secret, err = K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
				if err != nil {
					return fmt.Errorf("failed to fetch Secret after create: %w", err)
				}
			} else {
				return fmt.Errorf("failed to get Secret: %w", err)
			}
		}

		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}

		// Serialize callback data
		b, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal callback data: %w", err)
		}

		secret.Data[state] = b

		// Update Secret
		if _, uerr := K8sClient.CoreV1().Secrets(Namespace).Update(ctx, secret, v1.UpdateOptions{}); uerr != nil {
			if errors.IsConflict(uerr) {
				continue // retry
			}
			return fmt.Errorf("failed to update Secret: %w", uerr)
		}

		return nil
	}

	return fmt.Errorf("failed to update Secret after retries")
}

// GetOAuthCallback retrieves OAuth callback data by state
func GetOAuthCallback(ctx context.Context, state string) (*OAuthCallbackData, error) {
	const secretName = "oauth-callbacks"

	secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("callback not found")
		}
		return nil, fmt.Errorf("failed to read Secret: %w", err)
	}

	if secret.Data == nil {
		return nil, fmt.Errorf("callback not found")
	}

	raw, ok := secret.Data[state]
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("callback not found")
	}

	var data OAuthCallbackData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("failed to decode callback data: %w", err)
	}

	return &data, nil
}

// GetOAuthCallbackEndpoint handles GET /oauth2callback/status?state=xxx
// This allows MCP or other consumers to check the status of their OAuth flow
func GetOAuthCallbackEndpoint(c *gin.Context) {
	state := c.Query("state")
	if state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing state parameter"})
		return
	}

	data, err := GetOAuthCallback(c.Request.Context(), state)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "callback not found"})
		return
	}

	// Don't return sensitive tokens in the response
	response := gin.H{
		"provider":   data.Provider,
		"userId":     data.UserID,
		"receivedAt": data.ReceivedAt,
		"consumed":   data.Consumed,
		"hasToken":   data.AccessToken != "",
	}

	if data.Error != "" {
		response["error"] = data.Error
		response["errorDescription"] = data.ErrorDesc
	}

	c.JSON(http.StatusOK, response)
}

// // Helper function for min
// func min(a, b int) int {
// 	if a < b {
// 		return a
// 	}
// 	return b
// }

// OAuthStateData represents the session context passed in OAuth state parameter
type OAuthStateData struct {
	Provider    string `json:"provider"`
	ProjectName string `json:"projectName"`
	SessionName string `json:"sessionName"`
	Timestamp   int64  `json:"timestamp"`
}

// // parseOAuthState extracts session context from the base64-encoded state parameter
// // DEPRECATED: Use validateAndParseOAuthState for HMAC-signed states
// func parseOAuthState(state string) (*OAuthStateData, error) {
// 	// Decode base64
// 	decoded, err := base64.StdEncoding.DecodeString(state)
// 	if err != nil {
// 		// Try RawURLEncoding if standard fails
// 		decoded, err = base64.RawURLEncoding.DecodeString(state)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to decode state: %w", err)
// 		}
// 	}

// 	// Parse JSON
// 	var stateData OAuthStateData
// 	if err := json.Unmarshal(decoded, &stateData); err != nil {
// 		return nil, fmt.Errorf("failed to parse state JSON: %w", err)
// 	}

// 	return &stateData, nil
// }

// validateAndParseOAuthState validates HMAC signature and extracts session context from signed state token
// Expected format: base64(json) + "." + base64(signature)
func validateAndParseOAuthState(state string) (*OAuthStateData, error) {
	// Split state into data and signature
	parts := strings.Split(state, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid state format: expected 'data.signature'")
	}

	encodedData := parts[0]
	encodedSignature := parts[1]

	// Decode the JSON data
	stateJSON, err := base64.URLEncoding.DecodeString(encodedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode state data: %w", err)
	}

	// Decode the signature
	receivedSignature, err := base64.URLEncoding.DecodeString(encodedSignature)
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	// Get HMAC secret from environment
	secret := os.Getenv("OAUTH_STATE_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("OAUTH_STATE_SECRET not configured")
	}

	// Compute expected signature
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(stateJSON)
	expectedSignature := h.Sum(nil)

	// Compare signatures using constant-time comparison to prevent timing attacks
	if !hmac.Equal(receivedSignature, expectedSignature) {
		return nil, fmt.Errorf("invalid state signature (possible CSRF attack)")
	}

	// Parse JSON
	var stateData OAuthStateData
	if err := json.Unmarshal(stateJSON, &stateData); err != nil {
		return nil, fmt.Errorf("failed to parse state JSON: %w", err)
	}

	// Optional: validate timestamp (5 minute expiry)
	age := time.Now().Unix() - stateData.Timestamp
	if age > 300 { // 5 minutes
		return nil, fmt.Errorf("state token expired (age: %d seconds)", age)
	}
	if age < 0 {
		return nil, fmt.Errorf("state token has future timestamp (possible replay attack)")
	}

	log.Printf("✓ Validated OAuth state for %s/%s (provider: %s, age: %ds)", stateData.ProjectName, stateData.SessionName, stateData.Provider, age)

	return &stateData, nil
}

// // writeCredentialsToSessionPVC writes OAuth credentials directly to the session runner pod's PVC
// // This stores credentials at /workspace/.google-oauth-credentials.json in the session pod
// func writeCredentialsToSessionPVC(ctx context.Context, projectName, sessionName, accessToken, refreshToken string, expiresIn int64) error {
// 	// Construct workspace proxy path
// 	// The workspace proxy mounts session PVCs at /workspace-proxy/{namespace}/{sessionName}
// 	workspaceProxyBase := os.Getenv("WORKSPACE_PROXY_BASE")
// 	if workspaceProxyBase == "" {
// 		workspaceProxyBase = "/workspace-proxy"
// 	}

// 	sessionWorkspacePath := filepath.Join(workspaceProxyBase, projectName, sessionName)

// 	// Check if session workspace exists
// 	if _, err := os.Stat(sessionWorkspacePath); os.IsNotExist(err) {
// 		return fmt.Errorf("session workspace not found at %s (session pod may not be running yet)", sessionWorkspacePath)
// 	}

// 	// Prepare credentials JSON
// 	credentials := map[string]interface{}{
// 		"access_token":  accessToken,
// 		"refresh_token": refreshToken,
// 		"token_type":    "Bearer",
// 		"expires_in":    expiresIn,
// 		"created_at":    time.Now().Unix(),
// 	}

// 	credentialsJSON, err := json.MarshalIndent(credentials, "", "  ")
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal credentials: %w", err)
// 	}

// 	// Write credentials to session workspace
// 	credentialsPath := filepath.Join(sessionWorkspacePath, ".google-oauth-credentials.json")
// 	if err := os.WriteFile(credentialsPath, credentialsJSON, 0600); err != nil {
// 		return fmt.Errorf("failed to write credentials to %s: %w", credentialsPath, err)
// 	}

// 	log.Printf("✓ Wrote Google OAuth credentials to session %s/%s PVC at %s", projectName, sessionName, credentialsPath)
// 	return nil
// }

// storeCredentialsInSecret stores OAuth credentials in a Kubernetes Secret in the project namespace
// Secret name: {sessionName}-{provider}-oauth (e.g., agentic-session-123-google-oauth)
// This allows the session pod to mount or read the credentials from its own namespace
// The Secret is owned by the AgenticSession CR, so it's automatically deleted when the session is deleted
func storeCredentialsInSecret(ctx context.Context, projectName, sessionName, provider, accessToken, refreshToken string, expiresIn int64) error {
	secretName := fmt.Sprintf("%s-%s-oauth", sessionName, provider)

	// Get OAuth provider config for scopes
	providerConfig, err := getOAuthProvider(provider)
	if err != nil {
		return fmt.Errorf("failed to get OAuth provider config: %w", err)
	}

	// Calculate expiry time in ISO 8601 format as expected by workspace-mcp
	// workspace-mcp expects timezone-naive format like Python's datetime.isoformat()
	expiryTime := time.Now().Add(time.Duration(expiresIn) * time.Second)

	// Prepare credentials JSON with only the token, scopes, and expiry
	// client_id, client_secret, token_uri, and refresh_token are empty strings to minimize stored secrets
	credentials := map[string]interface{}{
		"token":         accessToken,
		"refresh_token": "",
		"token_uri":     "",
		"client_id":     "",
		"client_secret": "",
		"scopes":        providerConfig.Scopes,
		"expiry":        expiryTime.Format("2006-01-02T15:04:05"), // Timezone-naive format for Python compatibility
	}

	credentialsJSON, err := json.MarshalIndent(credentials, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Get the AgenticSession CR to set as owner
	gvr := schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1alpha1",
		Resource: "agenticsessions",
	}
	sessionObj, err := DynamicClient.Resource(gvr).Namespace(projectName).Get(ctx, sessionName, v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get AgenticSession %s/%s: %w", projectName, sessionName, err)
	}

	// Create OwnerReference to ensure Secret is deleted when session is deleted
	ownerRef := v1.OwnerReference{
		APIVersion: sessionObj.GetAPIVersion(),
		Kind:       sessionObj.GetKind(),
		Name:       sessionObj.GetName(),
		UID:        sessionObj.GetUID(),
		Controller: BoolPtr(true),
		// BlockOwnerDeletion intentionally omitted (can cause permission issues)
	}

	// Create or update Secret in project namespace
	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      secretName,
			Namespace: projectName,
			Labels: map[string]string{
				"app":                      "ambient-code",
				"ambient-code.io/session":  sessionName,
				"ambient-code.io/provider": provider,
				"ambient-code.io/oauth":    "true",
			},
			OwnerReferences: []v1.OwnerReference{ownerRef},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"credentials.json": credentialsJSON,
			"access_token":     []byte(accessToken),
			"refresh_token":    []byte(""),
		},
	}

	// Try to create the Secret
	_, err = K8sClient.CoreV1().Secrets(projectName).Create(ctx, secret, v1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// Update existing Secret
			_, err = K8sClient.CoreV1().Secrets(projectName).Update(ctx, secret, v1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update Secret %s/%s: %w", projectName, secretName, err)
			}
			log.Printf("✓ Updated OAuth credentials Secret %s/%s", projectName, secretName)
		} else {
			return fmt.Errorf("failed to create Secret %s/%s: %w", projectName, secretName, err)
		}
	} else {
		log.Printf("✓ Created OAuth credentials Secret %s/%s", projectName, secretName)
	}

	return nil
}

// ============================================================================
// Cluster-Level Google OAuth (User-Scoped, Not Session-Specific)
// ============================================================================

// GoogleOAuthCredentials represents cluster-level Google OAuth credentials for a user
type GoogleOAuthCredentials struct {
	UserID       string    `json:"userId"`
	Email        string    `json:"email,omitempty"`
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	Scopes       []string  `json:"scopes"`
	ExpiresAt    time.Time `json:"expiresAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// isValidUserID validates userID for use as a Kubernetes Secret data key
// Keys must match regex: [-._a-zA-Z0-9]+
// Note: userID is sanitized in forwardedIdentityMiddleware to ensure validity
func isValidUserID(userID string) bool {
	if userID == "" || len(userID) > 253 {
		return false
	}
	// Check against K8s Secret key regex: [-._a-zA-Z0-9]+
	// Check for invalid characters
	for _, ch := range userID {
		isValid := (ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '_' || ch == '.'
		if !isValid {
			return false
		}
	}
	return true
}

// GetGoogleOAuthURLGlobal handles POST /api/auth/google/connect
// Returns OAuth URL for cluster-level Google authentication
func GetGoogleOAuthURLGlobal(c *gin.Context) {
	// Verify user has valid K8s token (follows RBAC pattern)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	// Verify user is authenticated and userID is valid
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}
	if !isValidUserID(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user identifier"})
		return
	}

	// Get OAuth provider config
	provider, err := getOAuthProvider("google")
	if err != nil {
		log.Printf("Failed to get OAuth provider: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Google OAuth not configured"})
		return
	}

	// Build state with user context only (no session/project)
	stateData := map[string]interface{}{
		"provider":  "google",
		"userID":    userID,
		"timestamp": time.Now().Unix(),
		"cluster":   true, // Flag to indicate cluster-level OAuth
	}

	// Serialize state to JSON
	stateJSON, err := json.Marshal(stateData)
	if err != nil {
		log.Printf("Failed to marshal state: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OAuth state"})
		return
	}

	// Get HMAC secret from environment
	secret := os.Getenv("OAUTH_STATE_SECRET")
	if secret == "" {
		log.Printf("OAUTH_STATE_SECRET not configured")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth state validation not configured"})
		return
	}

	// Generate HMAC signature
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(stateJSON)
	signature := h.Sum(nil)

	// Combine: base64(json) + "." + base64(signature)
	stateToken := base64.URLEncoding.EncodeToString(stateJSON) + "." + base64.URLEncoding.EncodeToString(signature)

	// Get backend URL for redirect URI
	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://localhost:8080"
	}
	redirectURI := fmt.Sprintf("%s/oauth2callback", backendURL)

	// Build OAuth URL
	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&access_type=offline&state=%s&prompt=consent",
		provider.ClientID,
		redirectURI,
		strings.Join(provider.Scopes, " "),
		stateToken,
	)

	log.Printf("Generated cluster-level Google OAuth URL for user %s", userID)

	c.JSON(http.StatusOK, gin.H{
		"url":   authURL,
		"state": stateToken,
	})
}

// HandleGoogleOAuthCallback handles the OAuth callback for cluster-level Google auth
// This is called via the generic /oauth2callback endpoint when state contains "cluster":true
func HandleGoogleOAuthCallback(ctx context.Context, code string, stateData map[string]interface{}) error {
	userID, _ := stateData["userID"].(string)
	if userID == "" {
		return fmt.Errorf("missing userID in state")
	}

	// Get OAuth provider config
	provider, err := getOAuthProvider("google")
	if err != nil {
		return fmt.Errorf("failed to get OAuth provider: %w", err)
	}

	// Get backend URL for redirect URI
	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://localhost:8080"
	}
	redirectURI := fmt.Sprintf("%s/oauth2callback", backendURL)

	// Exchange code for tokens
	tokenData, err := exchangeOAuthCode(ctx, provider, code, redirectURI)
	if err != nil {
		return fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user's email from Google
	userEmail, err := getGoogleUserEmail(ctx, tokenData.AccessToken)
	if err != nil {
		log.Printf("Warning: failed to get user email: %v", err)
		userEmail = "" // Non-fatal
	}

	// Store credentials in cluster-level ConfigMap
	credentials := GoogleOAuthCredentials{
		UserID:       userID,
		Email:        userEmail,
		AccessToken:  tokenData.AccessToken,
		RefreshToken: tokenData.RefreshToken,
		Scopes:       provider.Scopes,
		ExpiresAt:    time.Now().Add(time.Duration(tokenData.ExpiresIn) * time.Second),
		UpdatedAt:    time.Now(),
	}

	if err := storeGoogleCredentials(ctx, &credentials); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	log.Printf("✓ Stored cluster-level Google OAuth credentials for user %s", userID)
	return nil
}

// sanitizeSecretKey converts a userID to a valid Kubernetes secret key
// IMPORTANT: This must match the sanitization in components/operator/internal/handlers/sessions.go
// Kubernetes secret keys must match: [-._a-zA-Z0-9]+
// Replaces invalid characters (: / etc.) with hyphens
func sanitizeSecretKey(userID string) string {
	// Replace colons and slashes with hyphens
	sanitized := strings.ReplaceAll(userID, ":", "-")
	sanitized = strings.ReplaceAll(sanitized, "/", "-")
	return sanitized
}

// storeGoogleCredentials stores Google OAuth credentials in cluster-level Secret
func storeGoogleCredentials(ctx context.Context, creds *GoogleOAuthCredentials) error {
	if creds == nil || creds.UserID == "" {
		return fmt.Errorf("invalid credentials payload")
	}

	const secretName = "google-oauth-credentials"
	secretKey := sanitizeSecretKey(creds.UserID)

	for i := 0; i < 3; i++ { // retry on conflict
		secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// Create Secret
				secret = &corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      secretName,
						Namespace: Namespace,
						Labels: map[string]string{
							"app":                            "ambient-code",
							"ambient-code.io/oauth":          "true",
							"ambient-code.io/oauth-provider": "google",
						},
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{},
				}
				if _, cerr := K8sClient.CoreV1().Secrets(Namespace).Create(ctx, secret, v1.CreateOptions{}); cerr != nil && !errors.IsAlreadyExists(cerr) {
					return fmt.Errorf("failed to create Secret: %w", cerr)
				}
				// Fetch again to get resourceVersion
				secret, err = K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
				if err != nil {
					return fmt.Errorf("failed to fetch Secret after create: %w", err)
				}
			} else {
				return fmt.Errorf("failed to get Secret: %w", err)
			}
		}

		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}

		b, err := json.Marshal(creds)
		if err != nil {
			return fmt.Errorf("failed to marshal credentials: %w", err)
		}
		secret.Data[secretKey] = b

		if _, uerr := K8sClient.CoreV1().Secrets(Namespace).Update(ctx, secret, v1.UpdateOptions{}); uerr != nil {
			if errors.IsConflict(uerr) {
				continue // retry
			}
			return fmt.Errorf("failed to update Secret: %w", uerr)
		}
		return nil
	}
	return fmt.Errorf("failed to update Secret after retries")
}

// GetGoogleCredentials retrieves cluster-level Google OAuth credentials for a user
func GetGoogleCredentials(ctx context.Context, userID string) (*GoogleOAuthCredentials, error) {
	const secretName = "google-oauth-credentials"
	secretKey := sanitizeSecretKey(userID)

	secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil // No credentials stored yet
		}
		return nil, fmt.Errorf("failed to get Secret: %w", err)
	}

	if secret.Data == nil || len(secret.Data[secretKey]) == 0 {
		return nil, nil // User hasn't connected yet
	}

	var creds GoogleOAuthCredentials
	if err := json.Unmarshal(secret.Data[secretKey], &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return &creds, nil
}

// GetGoogleOAuthStatusGlobal handles GET /api/auth/google/status
// Returns connection status for current user
func GetGoogleOAuthStatusGlobal(c *gin.Context) {
	// Verify user has valid K8s token (follows RBAC pattern)
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
	if !isValidUserID(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user identifier"})
		return
	}

	creds, err := GetGoogleCredentials(c.Request.Context(), userID)
	if err != nil {
		log.Printf("Failed to get Google credentials for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check connection status"})
		return
	}

	if creds == nil {
		c.JSON(http.StatusOK, gin.H{
			"connected": false,
		})
		return
	}

	// Check if token is expired
	isExpired := time.Now().After(creds.ExpiresAt)

	c.JSON(http.StatusOK, gin.H{
		"connected": true,
		"email":     creds.Email,
		"expiresAt": creds.ExpiresAt.Format(time.RFC3339),
		"expired":   isExpired,
	})
}

// DisconnectGoogleOAuthGlobal handles POST /api/auth/google/disconnect
// Removes user's Google OAuth credentials from cluster storage
func DisconnectGoogleOAuthGlobal(c *gin.Context) {
	// Verify user has valid K8s token (follows RBAC pattern)
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
	if !isValidUserID(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user identifier"})
		return
	}

	const secretName = "google-oauth-credentials"
	secretKey := sanitizeSecretKey(userID)
	ctx := c.Request.Context()

	for i := 0; i < 3; i++ { // retry on conflict
		secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// Already disconnected
				c.JSON(http.StatusOK, gin.H{"message": "Google Drive disconnected"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to access credentials"})
			return
		}

		if secret.Data == nil || len(secret.Data[secretKey]) == 0 {
			// Already disconnected
			c.JSON(http.StatusOK, gin.H{"message": "Google Drive disconnected"})
			return
		}

		delete(secret.Data, secretKey)

		if _, uerr := K8sClient.CoreV1().Secrets(Namespace).Update(ctx, secret, v1.UpdateOptions{}); uerr != nil {
			if errors.IsConflict(uerr) {
				continue // retry
			}
			log.Printf("Failed to update Secret: %v", uerr)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect"})
			return
		}

		log.Printf("✓ Removed Google OAuth credentials for user %s", userID)
		c.JSON(http.StatusOK, gin.H{"message": "Google Drive disconnected successfully"})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect after retries"})
}

// getGoogleUserEmail fetches the user's email from Google using the access token
func getGoogleUserEmail(ctx context.Context, accessToken string) (string, error) {
	// Create request with context for timeout/cancellation support
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Use client with timeout instead of DefaultClient
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var userInfo struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return "", err
	}

	return userInfo.Email, nil
}
