package server

import (
	"testing"
)

func TestSanitizeUserID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "OpenShift kube:admin",
			input:    "kube:admin",
			expected: "kube-admin",
		},
		{
			name:     "OpenShift system user",
			input:    "system:serviceaccount:namespace:sa-name",
			expected: "system-serviceaccount-namespace-sa-name",
		},
		{
			name:     "Email address",
			input:    "user@company.com",
			expected: "user-company.com",
		},
		{
			name:     "LDAP DN",
			input:    "CN=John Doe,OU=Engineering,DC=company,DC=com",
			expected: "CN-John-Doe-OU-Engineering-DC-company-DC-com",
		},
		{
			name:     "Windows domain",
			input:    "DOMAIN\\username",
			expected: "DOMAIN-username",
		},
		{
			name:     "Name with spaces",
			input:    "First Last",
			expected: "First-Last",
		},
		{
			name:     "Multiple consecutive invalid chars",
			input:    "user::admin@@test",
			expected: "user-admin-test",
		},
		{
			name:     "Leading/trailing hyphens removed",
			input:    ":user:",
			expected: "user",
		},
		{
			name:     "Already valid",
			input:    "valid-user_name.123",
			expected: "valid-user_name.123",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Very long username (truncated to 253)",
			input:    "very-long-username-" + string(make([]rune, 280)),
			expected: "", // Will be truncated and sanitized
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeUserID(tt.input)
			if tt.name == "Very long username (truncated to 253)" {
				// Just check length is <= 253
				if len(result) > 253 {
					t.Errorf("sanitizeUserID() length = %d, want <= 253", len(result))
				}
			} else if result != tt.expected {
				t.Errorf("sanitizeUserID(%q) = %q, want %q", tt.input, result, tt.expected)
			}

			// Security check: result should only contain valid chars
			for _, r := range result {
				if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.') {
					t.Errorf("sanitizeUserID(%q) contains invalid character: %q", tt.input, r)
				}
			}
		})
	}
}

// Test that sanitization is deterministic (same input always produces same output)
func TestSanitizeUserIDDeterministic(t *testing.T) {
	inputs := []string{
		"kube:admin",
		"user@email.com",
		"system:serviceaccount:ns:sa",
	}

	for _, input := range inputs {
		first := sanitizeUserID(input)
		for i := 0; i < 10; i++ {
			result := sanitizeUserID(input)
			if result != first {
				t.Errorf("sanitizeUserID() not deterministic: %q != %q", result, first)
			}
		}
	}
}
