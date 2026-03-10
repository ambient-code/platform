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
