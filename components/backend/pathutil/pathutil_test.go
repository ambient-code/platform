package pathutil

import (
	"path/filepath"
	"testing"
)

func TestIsPathWithinBase(t *testing.T) {
	tests := []struct {
		name     string
		abs      string
		baseDir  string
		expected bool
	}{
		{
			name:     "valid path within base",
			abs:      "/app/workspace/file.txt",
			baseDir:  "/app/workspace",
			expected: true,
		},
		{
			name:     "valid nested path",
			abs:      "/app/workspace/subdir/file.txt",
			baseDir:  "/app/workspace",
			expected: true,
		},
		{
			name:     "same path",
			abs:      "/app/workspace",
			baseDir:  "/app/workspace",
			expected: true,
		},
		{
			name:     "path traversal with ..",
			abs:      "/app/workspace/../secrets/file.txt",
			baseDir:  "/app/workspace",
			expected: false,
		},
		{
			name:     "path outside base",
			abs:      "/app/secrets/file.txt",
			baseDir:  "/app/workspace",
			expected: false,
		},
		{
			name:     "path with trailing slash",
			abs:      "/app/workspace/file.txt/",
			baseDir:  "/app/workspace",
			expected: true,
		},
		{
			name:     "path with multiple ..",
			abs:      "/app/workspace/subdir/../../secrets/file.txt",
			baseDir:  "/app/workspace",
			expected: false,
		},
		{
			name:     "path with . components",
			abs:      "/app/workspace/./subdir/./file.txt",
			baseDir:  "/app/workspace",
			expected: true,
		},
		{
			name:     "relative base and abs",
			abs:      "workspace/file.txt",
			baseDir:  "workspace",
			expected: true,
		},
		{
			name:     "relative path traversal",
			abs:      "workspace/../secrets/file.txt",
			baseDir:  "workspace",
			expected: false,
		},
		{
			name:     "Windows-style path (forward slash)",
			abs:      "C:/app/workspace/file.txt",
			baseDir:  "C:/app/workspace",
			expected: true,
		},
		{
			name:     "Windows-style path traversal",
			abs:      "C:/app/workspace/../secrets/file.txt",
			baseDir:  "C:/app/workspace",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPathWithinBase(tt.abs, tt.baseDir)
			if result != tt.expected {
				// Show cleaned paths for debugging
				cleanAbs := filepath.Clean(tt.abs)
				cleanBase := filepath.Clean(tt.baseDir)
				relPath, _ := filepath.Rel(cleanBase, cleanAbs)
				t.Errorf("IsPathWithinBase(%q, %q) = %v, want %v\n  cleanAbs=%q\n  cleanBase=%q\n  relPath=%q",
					tt.abs, tt.baseDir, result, tt.expected, cleanAbs, cleanBase, relPath)
			}
		})
	}
}
