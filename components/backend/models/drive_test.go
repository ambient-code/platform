package models

import (
	"testing"
)

func TestNewDriveIntegration(t *testing.T) {
	t.Run("creates integration with correct defaults", func(t *testing.T) {
		// Act
		integration := NewDriveIntegration("user-1", "project-1", PermissionScopeGranular)

		// Assert
		if integration.ID == "" {
			t.Error("expected non-empty ID")
		}
		if integration.UserID != "user-1" {
			t.Errorf("expected UserID 'user-1', got %q", integration.UserID)
		}
		if integration.ProjectName != "project-1" {
			t.Errorf("expected ProjectName 'project-1', got %q", integration.ProjectName)
		}
		if integration.Provider != "google" {
			t.Errorf("expected Provider 'google', got %q", integration.Provider)
		}
		if integration.PermissionScope != PermissionScopeGranular {
			t.Errorf("expected PermissionScope 'granular', got %q", integration.PermissionScope)
		}
		if integration.Status != IntegrationStatusActive {
			t.Errorf("expected Status 'active', got %q", integration.Status)
		}
		if integration.CreatedAt.IsZero() {
			t.Error("expected non-zero CreatedAt")
		}
		if integration.UpdatedAt.IsZero() {
			t.Error("expected non-zero UpdatedAt")
		}
		if !integration.CreatedAt.Equal(integration.UpdatedAt) {
			t.Error("expected CreatedAt and UpdatedAt to be equal on creation")
		}
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		a := NewDriveIntegration("user-1", "project-1", PermissionScopeGranular)
		b := NewDriveIntegration("user-1", "project-1", PermissionScopeGranular)

		if a.ID == b.ID {
			t.Error("expected different IDs for separate calls")
		}
	})
}

func TestDriveIntegration_Activate(t *testing.T) {
	tests := []struct {
		name        string
		fromStatus  IntegrationStatus
		expectError bool
	}{
		{"from expired", IntegrationStatusExpired, false},
		{"from disconnected", IntegrationStatusDisconnected, false},
		{"from error", IntegrationStatusError, false},
		{"from active fails", IntegrationStatusActive, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			integration := &DriveIntegration{Status: tc.fromStatus}

			// Act
			err := integration.Activate()

			// Assert
			if tc.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				if integration.Status != tc.fromStatus {
					t.Errorf("expected status to remain %q, got %q", tc.fromStatus, integration.Status)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if integration.Status != IntegrationStatusActive {
					t.Errorf("expected status 'active', got %q", integration.Status)
				}
				if integration.UpdatedAt.IsZero() {
					t.Error("expected UpdatedAt to be set")
				}
			}
		})
	}
}

func TestDriveIntegration_Disconnect(t *testing.T) {
	integration := &DriveIntegration{Status: IntegrationStatusActive}

	integration.Disconnect()

	if integration.Status != IntegrationStatusDisconnected {
		t.Errorf("expected status 'disconnected', got %q", integration.Status)
	}
	if integration.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestDriveIntegration_MarkExpired(t *testing.T) {
	integration := &DriveIntegration{Status: IntegrationStatusActive}

	integration.MarkExpired()

	if integration.Status != IntegrationStatusExpired {
		t.Errorf("expected status 'expired', got %q", integration.Status)
	}
	if integration.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestDriveIntegration_MarkError(t *testing.T) {
	integration := &DriveIntegration{Status: IntegrationStatusActive}

	integration.MarkError()

	if integration.Status != IntegrationStatusError {
		t.Errorf("expected status 'error', got %q", integration.Status)
	}
	if integration.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestFileGrant_Validate(t *testing.T) {
	tests := []struct {
		name        string
		grant       FileGrant
		expectError bool
		errContains string
	}{
		{
			name:        "empty googleFileId",
			grant:       FileGrant{GoogleFileID: "", FileName: "file.txt", MimeType: "text/plain"},
			expectError: true,
			errContains: "googleFileId",
		},
		{
			name:        "empty fileName",
			grant:       FileGrant{GoogleFileID: "abc", FileName: "", MimeType: "text/plain"},
			expectError: true,
			errContains: "fileName",
		},
		{
			name:        "empty mimeType",
			grant:       FileGrant{GoogleFileID: "abc", FileName: "file.txt", MimeType: ""},
			expectError: true,
			errContains: "mimeType",
		},
		{
			name:        "valid grant",
			grant:       FileGrant{GoogleFileID: "abc", FileName: "file.txt", MimeType: "text/plain"},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.grant.Validate()

			if tc.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				if tc.errContains != "" && err != nil {
					if got := err.Error(); got == "" || !contains(got, tc.errContains) {
						t.Errorf("expected error to contain %q, got %q", tc.errContains, got)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestPickerFile_ToFileGrant(t *testing.T) {
	size := int64(1024)
	pf := &PickerFile{
		ID:        "google-file-id",
		Name:      "document.pdf",
		MimeType:  "application/pdf",
		URL:       "https://drive.google.com/file/d/google-file-id",
		SizeBytes: &size,
		IsFolder:  false,
	}

	grant := pf.ToFileGrant("integration-123")

	if grant.ID == "" {
		t.Error("expected non-empty ID")
	}
	if grant.IntegrationID != "integration-123" {
		t.Errorf("expected IntegrationID 'integration-123', got %q", grant.IntegrationID)
	}
	if grant.GoogleFileID != "google-file-id" {
		t.Errorf("expected GoogleFileID 'google-file-id', got %q", grant.GoogleFileID)
	}
	if grant.FileName != "document.pdf" {
		t.Errorf("expected FileName 'document.pdf', got %q", grant.FileName)
	}
	if grant.MimeType != "application/pdf" {
		t.Errorf("expected MimeType 'application/pdf', got %q", grant.MimeType)
	}
	if grant.FileURL != "https://drive.google.com/file/d/google-file-id" {
		t.Errorf("expected FileURL to match, got %q", grant.FileURL)
	}
	if grant.SizeBytes == nil || *grant.SizeBytes != 1024 {
		t.Errorf("expected SizeBytes 1024, got %v", grant.SizeBytes)
	}
	if grant.IsFolder != false {
		t.Error("expected IsFolder false")
	}
	if grant.Status != FileGrantStatusActive {
		t.Errorf("expected Status 'active', got %q", grant.Status)
	}
	if grant.GrantedAt.IsZero() {
		t.Error("expected non-zero GrantedAt")
	}
}

func TestFileGrant_MarkUnavailable(t *testing.T) {
	grant := &FileGrant{Status: FileGrantStatusActive}

	grant.MarkUnavailable()

	if grant.Status != FileGrantStatusUnavailable {
		t.Errorf("expected status 'unavailable', got %q", grant.Status)
	}
}

func TestFileGrant_Revoke(t *testing.T) {
	grant := &FileGrant{Status: FileGrantStatusActive}

	grant.Revoke()

	if grant.Status != FileGrantStatusRevoked {
		t.Errorf("expected status 'revoked', got %q", grant.Status)
	}
}

func TestFileGrant_Reactivate(t *testing.T) {
	grant := &FileGrant{Status: FileGrantStatusRevoked}

	grant.Reactivate()

	if grant.Status != FileGrantStatusActive {
		t.Errorf("expected status 'active', got %q", grant.Status)
	}
}
