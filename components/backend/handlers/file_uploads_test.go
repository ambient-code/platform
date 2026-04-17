package handlers

import (
	"testing"
)

func TestFileUploadS3Key(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		session   string
		filePath  string
		expected  string
	}{
		{
			name:      "simple file",
			namespace: "my-project",
			session:   "session-abc123",
			filePath:  "document.pdf",
			expected:  "my-project/session-abc123/file-uploads/document.pdf",
		},
		{
			name:      "nested path",
			namespace: "my-project",
			session:   "session-abc123",
			filePath:  "subdir/image.png",
			expected:  "my-project/session-abc123/file-uploads/subdir/image.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fileUploadS3Key(tt.namespace, tt.session, tt.filePath)
			if got != tt.expected {
				t.Errorf("fileUploadS3Key() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFileUploadS3Prefix(t *testing.T) {
	got := fileUploadS3Prefix("my-project", "session-abc123")
	expected := "my-project/session-abc123/file-uploads/"
	if got != expected {
		t.Errorf("fileUploadS3Prefix() = %q, want %q", got, expected)
	}
}
