package handlers

import (
	"regexp"
)

// kubernetesNameRegex matches valid Kubernetes resource names.
// Names must:
// - Start with a lowercase letter or digit
// - Contain only lowercase letters, digits, or hyphens
// - End with a lowercase letter or digit
// - Be at most 63 characters
var kubernetesNameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// IsValidKubernetesName validates that a name conforms to Kubernetes naming conventions.
// This prevents path injection attacks and ensures names are valid K8s resource names.
func IsValidKubernetesName(name string) bool {
	if name == "" || len(name) > 63 {
		return false
	}
	return kubernetesNameRegex.MatchString(name)
}

// ValidateSessionID validates a session ID parameter.
// Returns true if the session ID is a valid Kubernetes resource name.
func ValidateSessionID(sessionID string) bool {
	return IsValidKubernetesName(sessionID)
}

// ValidateProjectName validates a project name parameter.
// Returns true if the project name is a valid Kubernetes namespace name.
func ValidateProjectName(project string) bool {
	return IsValidKubernetesName(project)
}
