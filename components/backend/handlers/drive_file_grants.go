package handlers

import (
	"net/http"
	"strings"
	"time"

	"ambient-code-backend/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// DriveFileGrantsHandler handles file grant CRUD endpoints.
type DriveFileGrantsHandler struct {
	storage     *DriveStorage
	oauthConfig *oauth2.Config
}

// NewDriveFileGrantsHandler creates a new handler with the given storage.
func NewDriveFileGrantsHandler(storage *DriveStorage, oauthConfig *oauth2.Config) *DriveFileGrantsHandler {
	return &DriveFileGrantsHandler{storage: storage, oauthConfig: oauthConfig}
}

// HandleUpdateFileGrants replaces the current file grant set with the provided list.
// PUT /api/projects/:projectName/integrations/google-drive/files
func (h *DriveFileGrantsHandler) HandleUpdateFileGrants(c *gin.Context) {
	projectName := c.Param("projectName")
	userID := c.GetString("userID")
	if userID == "" {
		userID = "default-user"
	}

	var req models.UpdateFileGrantsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	if len(req.Files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "at least one file is required",
			"details": "files array must not be empty",
		})
		return
	}

	// Get the integration to find the integration ID
	integration, err := h.storage.GetIntegration(c.Request.Context(), projectName, userID)
	if err != nil || integration == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "no active Google Drive integration found",
		})
		return
	}

	// Get existing file grants for comparison
	existingGrants, _ := h.storage.ListFileGrants(c.Request.Context(), integration.ID)
	existingByFileID := make(map[string]models.FileGrant)
	for _, g := range existingGrants {
		existingByFileID[g.GoogleFileID] = g
	}

	// Build the new grant set
	newByFileID := make(map[string]bool)
	var newGrants []models.FileGrant
	added := 0

	for _, pf := range req.Files {
		newByFileID[pf.ID] = true

		if existing, found := existingByFileID[pf.ID]; found {
			// Keep existing grant (preserve timestamps)
			existing.Reactivate()
			newGrants = append(newGrants, existing)
		} else {
			// Create new grant from picker file
			grant := pf.ToFileGrant(integration.ID)
			if err := grant.Validate(); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "invalid file data",
					"details": err.Error(),
				})
				return
			}
			newGrants = append(newGrants, *grant)
			added++
		}
	}

	// Count removed files
	removed := 0
	for fileID := range existingByFileID {
		if !newByFileID[fileID] {
			removed++
		}
	}

	// Persist the updated file grants
	if err := h.storage.UpdateFileGrants(c.Request.Context(), integration.ID, newGrants); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to update file grants",
			"details": err.Error(),
		})
		return
	}

	// Update the file count on the integration
	integration.FileCount = len(newGrants)
	_ = h.storage.SaveIntegration(c.Request.Context(), integration)

	c.JSON(http.StatusOK, models.UpdateFileGrantsResponse{
		Files:   newGrants,
		Added:   added,
		Removed: removed,
	})
}

// HandleListFileGrants returns all file grants for the integration.
// GET /api/projects/:projectName/integrations/google-drive/files
func (h *DriveFileGrantsHandler) HandleListFileGrants(c *gin.Context) {
	projectName := c.Param("projectName")
	userID := c.GetString("userID")
	if userID == "" {
		userID = "default-user"
	}

	integration, err := h.storage.GetIntegration(c.Request.Context(), projectName, userID)
	if err != nil || integration == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "no active Google Drive integration found",
		})
		return
	}

	grants, err := h.storage.ListFileGrants(c.Request.Context(), integration.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to list file grants",
			"details": err.Error(),
		})
		return
	}

	// Optionally verify file availability via Drive API (T025)
	verifyAvailability := c.Query("verify") == "true"
	if verifyAvailability {
		grants = h.verifyFileAvailability(c, integration, grants)
	}

	c.JSON(http.StatusOK, models.ListFileGrantsResponse{
		Files:      grants,
		TotalCount: len(grants),
	})
}

// verifyFileAvailability checks each file grant against the Drive API
// and updates status to "unavailable" for deleted/inaccessible files.
func (h *DriveFileGrantsHandler) verifyFileAvailability(
	c *gin.Context,
	integration *models.DriveIntegration,
	grants []models.FileGrant,
) []models.FileGrant {
	userID := c.GetString("userID")
	if userID == "" {
		userID = "default-user"
	}

	accessToken, refreshToken, expiresAt, err := h.storage.GetTokens(
		c.Request.Context(), integration.ProjectName, userID,
	)
	if err != nil {
		return grants
	}

	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       expiresAt,
	}
	tokenSource := h.oauthConfig.TokenSource(c.Request.Context(), token)

	srv, err := drive.NewService(c.Request.Context(), option.WithTokenSource(tokenSource))
	if err != nil {
		return grants
	}

	updated := false
	now := time.Now().UTC()
	for i := range grants {
		if grants[i].Status != models.FileGrantStatusActive {
			continue
		}

		_, err := srv.Files.Get(grants[i].GoogleFileID).
			Fields("id").
			SupportsAllDrives(true).
			Do()

		if err != nil {
			grants[i].MarkUnavailable()
			updated = true
		} else {
			grants[i].LastVerifiedAt = &now
		}
	}

	if updated {
		_ = h.storage.UpdateFileGrants(c.Request.Context(), integration.ID, grants)
	}

	return grants
}

// CheckDriveAccess is a helper that detects revoked access (T027).
// Call this when any Drive API operation returns 401/403.
func CheckDriveAccess(storage *DriveStorage, c *gin.Context, projectName, userID string, apiErr error) {
	if apiErr == nil {
		return
	}

	errMsg := apiErr.Error()
	isAuthError := false
	for _, code := range []string{"401", "403", "invalid_grant", "Token has been expired or revoked"} {
		if strings.Contains(errMsg, code) {
			isAuthError = true
			break
		}
	}

	if isAuthError {
		integration, err := storage.GetIntegration(c.Request.Context(), projectName, userID)
		if err == nil {
			integration.Disconnect()
			_ = storage.SaveIntegration(c.Request.Context(), integration)
		}
	}
}
