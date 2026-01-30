package webhook

import (
	"testing"
	"time"
)

// TestKeywordDetector_AmberDetection tests @amber keyword detection
func TestKeywordDetector_AmberDetection(t *testing.T) {
	detector := NewKeywordDetector()

	testCases := []struct {
		name     string
		text     string
		expected bool
	}{
		// Valid @amber mentions
		{
			name:     "simple amber mention",
			text:     "@amber review this PR",
			expected: true,
		},
		{
			name:     "amber at start",
			text:     "@amber please help",
			expected: true,
		},
		{
			name:     "amber in middle",
			text:     "Hey @amber can you review?",
			expected: true,
		},
		{
			name:     "amber at end",
			text:     "Please review @amber",
			expected: true,
		},
		{
			name:     "amber after newline",
			text:     "Some text\n@amber review",
			expected: true,
		},
		{
			name:     "amber after punctuation",
			text:     "Hello! @amber please help",
			expected: true,
		},
		{
			name:     "amber with emoji",
			text:     "👋 @amber review this",
			expected: true,
		},

		// Invalid - not @amber
		{
			name:     "no amber mention",
			text:     "This is a regular comment",
			expected: false,
		},
		{
			name:     "amber without @",
			text:     "amber review this",
			expected: false,
		},
		{
			name:     "partial match - @ambers",
			text:     "@ambers review this",
			expected: false,
		},
		{
			name:     "partial match - @amber-bot",
			text:     "@amber-bot review this",
			expected: false,
		},
		{
			name:     "amber in middle of word",
			text:     "test@amber.com",
			expected: false,
		},
		{
			name:     "amber in URL",
			text:     "https://github.com/@amber/repo",
			expected: false,
		},
		{
			name:     "different mention",
			text:     "@github-bot review",
			expected: false,
		},

		// Edge cases
		{
			name:     "empty string",
			text:     "",
			expected: false,
		},
		{
			name:     "just @amber",
			text:     "@amber",
			expected: true,
		},
		{
			name:     "multiple spaces before amber",
			text:     "    @amber",
			expected: true,
		},
		{
			name:     "tab before amber",
			text:     "\t@amber",
			expected: true,
		},
		{
			name:     "@amber followed by punctuation",
			text:     "@amber!",
			expected: true,
		},
		{
			name:     "@amber followed by comma",
			text:     "@amber, please review",
			expected: true,
		},
		{
			name:     "multiple amber mentions",
			text:     "@amber review and @amber check tests",
			expected: true,
		},
		{
			name:     "case sensitive - @Amber",
			text:     "@Amber review",
			expected: false, // Should be case-sensitive
		},
		{
			name:     "case sensitive - @AMBER",
			text:     "@AMBER review",
			expected: false, // Should be case-sensitive
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.DetectKeyword(tc.text)
			if result != tc.expected {
				t.Errorf("Expected %v for text %q, got %v", tc.expected, tc.text, result)
			}
		})
	}
}

// TestKeywordDetector_MultilineText tests detection in multiline comments
func TestKeywordDetector_MultilineText(t *testing.T) {
	detector := NewKeywordDetector()

	testCases := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name: "amber in multiline comment",
			text: `This is a long comment
with multiple lines.

@amber please review the security aspects of this PR.

Thanks!`,
			expected: true,
		},
		{
			name: "amber at start of line",
			text: `Line 1
@amber review
Line 3`,
			expected: true,
		},
		{
			name: "no amber in multiline",
			text: `This is a long comment
with multiple lines
but no keyword mention`,
			expected: false,
		},
		{
			name: "multiple mentions in multiline",
			text: `@amber check the frontend
and also
@amber review the backend`,
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.DetectKeyword(tc.text)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for multiline text", tc.expected, result)
			}
		})
	}
}

// TestKeywordDetector_RealWorldExamples tests with actual GitHub comment patterns
func TestKeywordDetector_RealWorldExamples(t *testing.T) {
	detector := NewKeywordDetector()

	testCases := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "code review request",
			text:     "@amber can you review the changes in src/auth.go?",
			expected: true,
		},
		{
			name:     "bug investigation",
			text:     "Found a bug in the webhook handler. @amber help investigate",
			expected: true,
		},
		{
			name:     "test request",
			text:     "@amber run the test suite and report results",
			expected: true,
		},
		{
			name:     "documentation request",
			text:     "@amber update the README with deployment instructions",
			expected: true,
		},
		{
			name:     "regular comment",
			text:     "LGTM! Approving this PR.",
			expected: false,
		},
		{
			name:     "quoted amber mention",
			text:     "As @someone said, `amber` is the keyword",
			expected: false, // Not a mention, just quoted text
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.DetectKeyword(tc.text)
			if result != tc.expected {
				t.Errorf("Expected %v for %q, got %v", tc.expected, tc.text, result)
			}
		})
	}
}

// TestKeywordDetector_Performance tests detection performance
func TestKeywordDetector_Performance(t *testing.T) {
	detector := NewKeywordDetector()

	// Create a very long comment (10KB)
	longText := ""
	for i := 0; i < 1000; i++ {
		longText += "This is a line of text without the keyword. "
	}
	longText += "@amber review this"

	// Should find keyword quickly even in long text
	start := time.Now()
	result := detector.DetectKeyword(longText)
	duration := time.Since(start)

	if !result {
		t.Error("Expected to find @amber in long text")
	}

	if duration > 10*time.Millisecond {
		t.Errorf("Keyword detection too slow (%v) for long text", duration)
	}
}
