// Package pathutil provides utilities for secure path validation and manipulation.
package pathutil

import (
	"path/filepath"
	"strings"
)

// IsPathWithinBase uses filepath.Rel to robustly verify that abs is within baseDir.
// This is more secure than strings.HasPrefix across different OS platforms.
//
// Security considerations:
// - Uses filepath.Clean on both paths to normalize separators and remove .. sequences
// - Uses filepath.Rel for platform-independent path validation
// - Checks for ".." prefix in relative path to detect traversal attempts
//
// Returns true if abs is within baseDir, false otherwise.
func IsPathWithinBase(abs, baseDir string) bool {
	// Clean both paths before comparison to prevent path traversal attacks
	// filepath.Clean normalizes paths and removes . and .. components
	cleanBase := filepath.Clean(baseDir)
	cleanAbs := filepath.Clean(abs)

	// Compute relative path from base to abs
	relPath, err := filepath.Rel(cleanBase, cleanAbs)
	if err != nil {
		// filepath.Rel returns error if paths are on different volumes (Windows)
		// or if one path cannot be made relative to the other
		return false
	}

	// If relPath starts with "..", it means abs is outside baseDir
	// For example:
	//   base=/app/workspace, abs=/app/workspace/file -> relPath=file (OK)
	//   base=/app/workspace, abs=/app/secrets -> relPath=../secrets (BLOCKED)
	if strings.HasPrefix(relPath, "..") {
		return false
	}

	return true
}
