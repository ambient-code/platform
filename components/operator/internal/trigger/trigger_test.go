package trigger

import (
	"testing"
)

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal string is lowercased and cleaned",
			input:    "DailyReport",
			expected: "dailyreport",
		},
		{
			name:     "special characters replaced with hyphens",
			input:    "my task!here",
			expected: "my-task-here",
		},
		{
			name:     "consecutive special chars produce single hyphen",
			input:    "hello!!!world",
			expected: "hello-world",
		},
		{
			name:     "trailing hyphens trimmed",
			input:    "hello!",
			expected: "hello",
		},
		{
			name:     "string over 40 chars truncated",
			input:    "abcdefghijklmnopqrstuvwxyz1234567890abcdefghijklmnop",
			expected: "abcdefghijklmnopqrstuvwxyz1234567890abcd",
		},
		{
			name:     "empty string returns run",
			input:    "",
			expected: "run",
		},
		{
			name:     "all special chars returns run",
			input:    "!!!@@@###",
			expected: "run",
		},
		{
			name:     "mixed case lowercased",
			input:    "MyDailyTask",
			expected: "mydailytask",
		},
		{
			name:     "spaces replaced with hyphens",
			input:    "daily jira summary",
			expected: "daily-jira-summary",
		},
		{
			name:     "leading special chars omitted",
			input:    "  hello",
			expected: "hello",
		},
		{
			name:     "digits preserved",
			input:    "task123",
			expected: "task123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeName_ScheduledSessionUUID(t *testing.T) {
	// Scheduled session names contain UUIDs (e.g. "schedule-cb9522da-ce4d-4bc8-8cc8-5647407288f5").
	// Without sanitization, "session-" + name + "-" + timestamp = 64 chars, exceeding
	// the 63-char K8s Service name limit.  sanitizeName must cap at 40 chars.
	input := "schedule-cb9522da-ce4d-4bc8-8cc8-5647407288f5"
	result := sanitizeName(input)
	if len(result) > 40 {
		t.Errorf("sanitizeName(%q) length = %d, want <= 40", input, len(result))
	}
	// session-{sanitized}-{10-digit-ts} must fit in 63 chars
	svcName := "session-" + result + "-1775052000"
	if len(svcName) > 63 {
		t.Errorf("derived Service name %q length = %d, exceeds 63-char K8s limit", svcName, len(svcName))
	}
}

func TestSanitizeName_TruncationPreservesValidSuffix(t *testing.T) {
	// Verify that truncation to 40 chars does not leave a trailing hyphen
	input := "abcdefghijklmnopqrstuvwxyz1234567890abcd!"
	result := sanitizeName(input)
	if len(result) > 40 {
		t.Errorf("sanitizeName(%q) length = %d, want <= 40", input, len(result))
	}
	if result[len(result)-1] == '-' {
		t.Errorf("sanitizeName(%q) ends with hyphen: %q", input, result)
	}
}
