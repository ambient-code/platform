package handlers

import (
	"crypto/rand"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

const featureFlagGranularDrivePermissions = "granular-drive-permissions"

// InitDriveIntegration constructs the Drive integration handlers from
// environment variables and registers them under the project-scoped
// route group. Safe to call when Google OAuth is not configured — the
// routes are still registered but the setup endpoint will fail at runtime
// if credentials are missing.
func InitDriveIntegration(api *gin.RouterGroup) {
	clientID := os.Getenv("GOOGLE_OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET")
	apiKey := os.Getenv("GOOGLE_API_KEY")
	appID := os.Getenv("GOOGLE_APP_ID")

	// HMAC secret for signing OAuth state parameters.
	// Falls back to OAUTH_STATE_SECRET, then generates a random key.
	hmacSecret := []byte(os.Getenv("DRIVE_HMAC_SECRET"))
	if len(hmacSecret) == 0 {
		hmacSecret = []byte(os.Getenv("OAUTH_STATE_SECRET"))
	}
	if len(hmacSecret) == 0 {
		hmacSecret = make([]byte, 32)
		if _, err := rand.Read(hmacSecret); err != nil {
			log.Printf("WARNING: failed to generate HMAC secret for Drive integration: %v", err)
		}
	}

	storage := NewDriveStorage(K8sClient, Namespace)
	integrationHandler := NewDriveIntegrationHandler(storage, clientID, clientSecret, hmacSecret, apiKey, appID)
	fileGrantsHandler := NewDriveFileGrantsHandler(storage, integrationHandler.oauthConfig)

	projectGroup := api.Group("/projects/:projectName", ValidateProjectContext())
	RegisterDriveIntegrationRoutes(projectGroup, integrationHandler, fileGrantsHandler)

	log.Printf("Drive integration routes registered (clientID configured: %v)", clientID != "")
}

// RegisterDriveIntegrationRoutes registers all Google Drive integration
// endpoints under the provided router group. All endpoints are gated
// behind the granular-drive-permissions feature flag.
func RegisterDriveIntegrationRoutes(router *gin.RouterGroup, integrationHandler *DriveIntegrationHandler, fileGrantsHandler *DriveFileGrantsHandler) {
	drive := router.Group("/integrations/google-drive")
	drive.Use(requireFeatureFlag(featureFlagGranularDrivePermissions))
	{
		drive.POST("/setup", integrationHandler.HandleDriveSetup)
		drive.GET("/callback", integrationHandler.HandleDriveCallback)
		drive.GET("/picker-token", integrationHandler.HandlePickerToken)

		drive.GET("/files", fileGrantsHandler.HandleListFileGrants)
		drive.PUT("/files", fileGrantsHandler.HandleUpdateFileGrants)

		drive.GET("/", integrationHandler.HandleGetDriveIntegration)
		drive.DELETE("/", integrationHandler.HandleDisconnectDriveIntegration)
	}
}

// requireFeatureFlag returns a Gin middleware that aborts with 404 when the
// named feature flag is disabled, effectively hiding the endpoints.
func requireFeatureFlag(flagName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !FeatureEnabled(flagName) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.Next()
	}
}
