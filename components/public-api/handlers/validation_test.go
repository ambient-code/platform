package handlers

import "testing"

func TestIsValidKubernetesName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid names
		{"simple lowercase", "myproject", true},
		{"with hyphens", "my-project", true},
		{"with numbers", "project123", true},
		{"starts with number", "123project", true},
		{"single char", "a", true},
		{"max length", "a234567890123456789012345678901234567890123456789012345678901ab", true}, // 63 chars

		// Invalid names
		{"empty string", "", false},
		{"too long", "a2345678901234567890123456789012345678901234567890123456789012345", false}, // 65 chars
		{"uppercase", "MyProject", false},
		{"underscore", "my_project", false},
		{"dot", "my.project", false},
		{"starts with hyphen", "-myproject", false},
		{"ends with hyphen", "myproject-", false},
		{"space", "my project", false},
		{"special chars", "my@project", false},
		{"path traversal", "../etc/passwd", false},
		{"slash", "my/project", false},
		{"newline", "my\nproject", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidKubernetesName(tt.input)
			if result != tt.expected {
				t.Errorf("IsValidKubernetesName(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateSessionID(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		expected  bool
	}{
		{"valid session id", "session-123", true},
		{"valid uuid-like", "a1b2c3d4", true},
		{"path traversal attempt", "../../etc/passwd", false},
		{"url encoded traversal", "%2e%2e%2fpasswd", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateSessionID(tt.sessionID)
			if result != tt.expected {
				t.Errorf("ValidateSessionID(%q) = %v, want %v", tt.sessionID, result, tt.expected)
			}
		})
	}
}

func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		expected    bool
	}{
		{"valid project", "my-project", true},
		{"valid namespace", "kube-system", true},
		{"uppercase invalid", "My-Project", false},
		{"special chars", "my$project", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateProjectName(tt.projectName)
			if result != tt.expected {
				t.Errorf("ValidateProjectName(%q) = %v, want %v", tt.projectName, result, tt.expected)
			}
		})
	}
}
