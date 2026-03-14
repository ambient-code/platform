package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const featureFlagGranularDrivePermissions = "granular-drive-permissions"

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
