// Package models defines data types for the platform backend.
package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PermissionScope defines the level of Google Drive access.
type PermissionScope string

const (
	PermissionScopeGranular PermissionScope = "granular"
	PermissionScopeFull     PermissionScope = "full"
)

// IntegrationStatus represents the state of a Drive integration.
type IntegrationStatus string

const (
	IntegrationStatusActive       IntegrationStatus = "active"
	IntegrationStatusDisconnected IntegrationStatus = "disconnected"
	IntegrationStatusExpired      IntegrationStatus = "expired"
	IntegrationStatusError        IntegrationStatus = "error"
)

// FileGrantStatus represents the state of a file grant.
type FileGrantStatus string

const (
	FileGrantStatusActive      FileGrantStatus = "active"
	FileGrantStatusUnavailable FileGrantStatus = "unavailable"
	FileGrantStatusRevoked     FileGrantStatus = "revoked"
)

// DriveIntegration represents a user's Google Drive connection to the platform.
type DriveIntegration struct {
	ID              string            `json:"id"`
	UserID          string            `json:"userId"`
	ProjectName     string            `json:"projectName"`
	Provider        string            `json:"provider"`
	PermissionScope PermissionScope   `json:"permissionScope"`
	Status          IntegrationStatus `json:"status"`
	TokenExpiresAt  time.Time         `json:"tokenExpiresAt,omitempty"`
	FileCount       int               `json:"fileCount"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}

// NewDriveIntegration creates a new DriveIntegration with default values.
func NewDriveIntegration(userID, projectName string, scope PermissionScope) *DriveIntegration {
	now := time.Now().UTC()
	return &DriveIntegration{
		ID:              uuid.New().String(),
		UserID:          userID,
		ProjectName:     projectName,
		Provider:        "google",
		PermissionScope: scope,
		Status:          IntegrationStatusActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// Activate transitions the integration to active status.
func (d *DriveIntegration) Activate() error {
	switch d.Status {
	case IntegrationStatusExpired, IntegrationStatusDisconnected, IntegrationStatusError:
		d.Status = IntegrationStatusActive
		d.UpdatedAt = time.Now().UTC()
		return nil
	default:
		return fmt.Errorf("cannot activate integration with status %q", d.Status)
	}
}

// Disconnect transitions the integration to disconnected status.
func (d *DriveIntegration) Disconnect() {
	d.Status = IntegrationStatusDisconnected
	d.UpdatedAt = time.Now().UTC()
}

// MarkExpired transitions the integration to expired status.
func (d *DriveIntegration) MarkExpired() {
	d.Status = IntegrationStatusExpired
	d.UpdatedAt = time.Now().UTC()
}

// MarkError transitions the integration to error status.
func (d *DriveIntegration) MarkError() {
	d.Status = IntegrationStatusError
	d.UpdatedAt = time.Now().UTC()
}

// FileGrant represents an individual file/folder that a user has granted access to.
type FileGrant struct {
	ID             string          `json:"id"`
	IntegrationID  string          `json:"integrationId"`
	GoogleFileID   string          `json:"googleFileId"`
	FileName       string          `json:"fileName"`
	MimeType       string          `json:"mimeType"`
	FileURL        string          `json:"fileUrl"`
	SizeBytes      *int64          `json:"sizeBytes,omitempty"`
	IsFolder       bool            `json:"isFolder"`
	Status         FileGrantStatus `json:"status"`
	GrantedAt      time.Time       `json:"grantedAt"`
	LastAccessedAt *time.Time      `json:"lastAccessedAt,omitempty"`
	LastVerifiedAt *time.Time      `json:"lastVerifiedAt,omitempty"`
}

// Validate checks that a FileGrant has all required fields.
func (f *FileGrant) Validate() error {
	if f.GoogleFileID == "" {
		return fmt.Errorf("googleFileId must not be empty")
	}
	if f.FileName == "" {
		return fmt.Errorf("fileName must not be empty")
	}
	if f.MimeType == "" {
		return fmt.Errorf("mimeType must not be empty")
	}
	return nil
}

// MarkUnavailable transitions the file grant to unavailable status.
func (f *FileGrant) MarkUnavailable() {
	f.Status = FileGrantStatusUnavailable
}

// Revoke transitions the file grant to revoked status.
func (f *FileGrant) Revoke() {
	f.Status = FileGrantStatusRevoked
}

// Reactivate transitions the file grant back to active status.
func (f *FileGrant) Reactivate() {
	f.Status = FileGrantStatusActive
}

// PickerFile represents file data as returned by the Google Picker callback.
type PickerFile struct {
	ID        string `json:"id" binding:"required"`
	Name      string `json:"name" binding:"required"`
	MimeType  string `json:"mimeType" binding:"required"`
	URL       string `json:"url,omitempty"`
	SizeBytes *int64 `json:"sizeBytes,omitempty"`
	IsFolder  bool   `json:"isFolder"`
}

// ToFileGrant converts a PickerFile to a FileGrant for persistence.
func (p *PickerFile) ToFileGrant(integrationID string) *FileGrant {
	now := time.Now().UTC()
	return &FileGrant{
		ID:            uuid.New().String(),
		IntegrationID: integrationID,
		GoogleFileID:  p.ID,
		FileName:      p.Name,
		MimeType:      p.MimeType,
		FileURL:       p.URL,
		SizeBytes:     p.SizeBytes,
		IsFolder:      p.IsFolder,
		Status:        FileGrantStatusActive,
		GrantedAt:     now,
	}
}

// UpdateFileGrantsRequest is the request body for PUT /files.
type UpdateFileGrantsRequest struct {
	Files []PickerFile `json:"files" binding:"required,min=1"`
}

// UpdateFileGrantsResponse is the response body for PUT /files.
type UpdateFileGrantsResponse struct {
	Files   []FileGrant `json:"files"`
	Added   int         `json:"added"`
	Removed int         `json:"removed"`
}

// ListFileGrantsResponse is the response body for GET /files.
type ListFileGrantsResponse struct {
	Files      []FileGrant `json:"files"`
	TotalCount int         `json:"totalCount"`
}

// SetupRequest is the request body for POST /setup.
type SetupRequest struct {
	PermissionScope PermissionScope `json:"permissionScope"`
	RedirectURI     string          `json:"redirectUri" binding:"required"`
}

// SetupResponse is the response body for POST /setup.
type SetupResponse struct {
	AuthURL string `json:"authUrl"`
	State   string `json:"state"`
}

// CallbackResponse is the response body for GET /callback.
type CallbackResponse struct {
	IntegrationID string `json:"integrationId"`
	Status        string `json:"status"`
	PickerToken   string `json:"pickerToken"`
}

// PickerTokenResponse is the response body for GET /picker-token.
type PickerTokenResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int    `json:"expiresIn"`
}
