package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ambient-code-backend/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// DriveIntegrationHandler handles Google Drive integration endpoints.
type DriveIntegrationHandler struct {
	storage      *DriveStorage
	oauthConfig  *oauth2.Config
	hmacSecret   []byte
	googleAPIKey string
	googleAppID  string
}

// NewDriveIntegrationHandler creates a new handler with the given dependencies.
func NewDriveIntegrationHandler(
	storage *DriveStorage,
	clientID, clientSecret string,
	hmacSecret []byte,
	googleAPIKey, googleAppID string,
) *DriveIntegrationHandler {
	return &DriveIntegrationHandler{
		storage: storage,
		oauthConfig: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     google.Endpoint,
		},
		hmacSecret:   hmacSecret,
		googleAPIKey: googleAPIKey,
		googleAppID:  googleAppID,
	}
}

// HandleDriveSetup initiates the Google Drive OAuth flow with the appropriate scope.
// POST /api/projects/:projectName/integrations/google-drive/setup
func (h *DriveIntegrationHandler) HandleDriveSetup(c *gin.Context) {
	projectName := c.Param("projectName")

	var req models.SetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	// Default to granular permissions
	if req.PermissionScope == "" {
		req.PermissionScope = models.PermissionScopeGranular
	}

	// Get scopes based on permission scope
	scopes := GetGoogleDriveScopes(req.PermissionScope)

	// Configure OAuth with the appropriate scopes and redirect URI
	config := *h.oauthConfig
	config.Scopes = scopes
	config.RedirectURL = req.RedirectURI

	// Generate HMAC-signed state parameter for CSRF protection
	stateData := fmt.Sprintf("%s:%s:%d", projectName, string(req.PermissionScope), time.Now().UnixNano())
	state := h.signState(stateData)

	// Generate the authorization URL
	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))

	c.JSON(http.StatusOK, models.SetupResponse{
		AuthURL: authURL,
		State:   state,
	})
}

// HandleDriveCallback handles the OAuth callback from Google.
// GET /api/projects/:projectName/integrations/google-drive/callback
func (h *DriveIntegrationHandler) HandleDriveCallback(c *gin.Context) {
	projectName := c.Param("projectName")
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing parameters",
			"details": "code and state are required",
		})
		return
	}

	// Verify the HMAC-signed state parameter
	if !h.verifyState(state) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid state",
			"details": "state parameter verification failed",
		})
		return
	}

	// Extract permission scope from state
	scope := h.extractScopeFromState(state)

	// Exchange the authorization code for tokens
	token, err := h.oauthConfig.Exchange(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "token exchange failed",
			"details": err.Error(),
		})
		return
	}

	// Get the user ID from the request context (set by auth middleware)
	userID := c.GetString("userID")
	if userID == "" {
		userID = "default-user" // Fallback for development
	}

	// Create the integration record
	integration := models.NewDriveIntegration(userID, projectName, scope)

	// Save the integration
	if err := h.storage.SaveIntegration(c.Request.Context(), integration); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to save integration",
			"details": err.Error(),
		})
		return
	}

	// Save the tokens in K8s Secrets
	if err := h.storage.SaveTokens(
		c.Request.Context(),
		integration,
		token.AccessToken,
		token.RefreshToken,
		token.Expiry,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to save tokens",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.CallbackResponse{
		IntegrationID: integration.ID,
		Status:        string(models.IntegrationStatusActive),
		PickerToken:   token.AccessToken,
	})
}

// HandlePickerToken returns a fresh access token for the Google Picker.
// GET /api/projects/:projectName/integrations/google-drive/picker-token
func (h *DriveIntegrationHandler) HandlePickerToken(c *gin.Context) {
	projectName := c.Param("projectName")
	userID := c.GetString("userID")
	if userID == "" {
		userID = "default-user"
	}

	// Get stored tokens
	accessToken, refreshToken, expiresAt, err := h.storage.GetTokens(c.Request.Context(), projectName, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "no active Google Drive integration found",
		})
		return
	}

	// Check if the token is expired and refresh if needed
	if time.Now().After(expiresAt) {
		tokenSource := h.oauthConfig.TokenSource(c.Request.Context(), &oauth2.Token{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			Expiry:       expiresAt,
		})

		newToken, err := tokenSource.Token()
		if err != nil {
			// Token refresh failed — integration may be disconnected externally
			integration, getErr := h.storage.GetIntegration(c.Request.Context(), projectName, userID)
			if getErr == nil {
				integration.Disconnect()
				_ = h.storage.SaveIntegration(c.Request.Context(), integration)
			}
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "token refresh failed, please re-authenticate",
			})
			return
		}

		accessToken = newToken.AccessToken

		// Update stored tokens
		_ = h.storage.SaveTokens(c.Request.Context(), &models.DriveIntegration{
			ProjectName: projectName,
			ID:          userID,
		}, newToken.AccessToken, newToken.RefreshToken, newToken.Expiry)

		expiresAt = newToken.Expiry
	}

	expiresIn := int(time.Until(expiresAt).Seconds())
	if expiresIn < 0 {
		expiresIn = 0
	}

	c.JSON(http.StatusOK, models.PickerTokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   expiresIn,
	})
}

// HandleGetDriveIntegration returns the current state of the Drive integration.
// GET /api/projects/:projectName/integrations/google-drive
func (h *DriveIntegrationHandler) HandleGetDriveIntegration(c *gin.Context) {
	projectName := c.Param("projectName")
	userID := c.GetString("userID")
	if userID == "" {
		userID = "default-user"
	}

	integration, err := h.storage.GetIntegration(c.Request.Context(), projectName, userID)
	if err != nil || integration == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "no Google Drive integration found",
		})
		return
	}

	c.JSON(http.StatusOK, integration)
}

// HandleDisconnectDriveIntegration disconnects the Drive integration.
// DELETE /api/projects/:projectName/integrations/google-drive
func (h *DriveIntegrationHandler) HandleDisconnectDriveIntegration(c *gin.Context) {
	projectName := c.Param("projectName")
	userID := c.GetString("userID")
	if userID == "" {
		userID = "default-user"
	}

	// Get the integration to check it exists
	integration, err := h.storage.GetIntegration(c.Request.Context(), projectName, userID)
	if err != nil || integration == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "no active integration found",
		})
		return
	}

	// Revoke the Google token
	accessToken, _, _, tokenErr := h.storage.GetTokens(c.Request.Context(), projectName, userID)
	if tokenErr == nil && accessToken != "" {
		revokeURL := fmt.Sprintf("https://oauth2.googleapis.com/revoke?token=%s", accessToken)
		// Best-effort revocation — don't block on failure
		resp, err := http.Post(revokeURL, "application/x-www-form-urlencoded", nil)
		if err == nil {
			resp.Body.Close()
		}
	}

	// Delete tokens
	_ = h.storage.DeleteTokens(c.Request.Context(), projectName, userID)

	// Update integration status
	integration.Disconnect()
	_ = h.storage.SaveIntegration(c.Request.Context(), integration)

	// Delete the integration record
	_ = h.storage.DeleteIntegration(c.Request.Context(), projectName, userID)

	c.Status(http.StatusNoContent)
}

// signState creates an HMAC-signed state parameter.
func (h *DriveIntegrationHandler) signState(data string) string {
	mac := hmac.New(sha256.New, h.hmacSecret)
	mac.Write([]byte(data))
	signature := hex.EncodeToString(mac.Sum(nil))
	return data + "." + signature
}

// verifyState verifies an HMAC-signed state parameter.
func (h *DriveIntegrationHandler) verifyState(state string) bool {
	parts := strings.SplitN(state, ".", 2)
	if len(parts) != 2 {
		return false
	}
	expected := h.signState(parts[0])
	return hmac.Equal([]byte(expected), []byte(state))
}

// extractScopeFromState extracts the permission scope from the state parameter.
func (h *DriveIntegrationHandler) extractScopeFromState(state string) models.PermissionScope {
	parts := strings.SplitN(state, ".", 2)
	if len(parts) == 0 {
		return models.PermissionScopeGranular
	}
	dataParts := strings.Split(parts[0], ":")
	if len(dataParts) >= 2 {
		scope := models.PermissionScope(dataParts[1])
		if scope == models.PermissionScopeFull || scope == models.PermissionScopeGranular {
			return scope
		}
	}
	return models.PermissionScopeGranular
}
