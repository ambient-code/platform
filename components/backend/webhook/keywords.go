package webhook

import (
	"regexp"
	"strings"
)

const (
	// AmberKeywordPattern is the regex pattern for detecting @amber mentions (FR-026)
	// Matches @amber at the start of a line or after whitespace
	// \b ensures word boundary to avoid matching "@amberlight" or similar
	AmberKeywordPattern = `(?:^|\s)@amber\b`
)

var (
	// amberKeywordRegex is the compiled regex for @amber keyword detection
	amberKeywordRegex = regexp.MustCompile(AmberKeywordPattern)
)

// KeywordDetector detects trigger keywords in GitHub comment bodies
type KeywordDetector struct {
	pattern *regexp.Regexp
}

// NewKeywordDetector creates a new keyword detector with the @amber pattern (FR-013)
func NewKeywordDetector() *KeywordDetector {
	return &KeywordDetector{
		pattern: amberKeywordRegex,
	}
}

// DetectKeyword checks if the comment body contains the @amber keyword
// Returns true if the keyword is detected, false otherwise
//
// The detection is case-insensitive and requires @amber to be:
// - At the start of the comment, OR
// - After whitespace (space, tab, newline)
//
// Examples that MATCH:
// - "@amber review this PR"
// - "Please @amber analyze this"
// - "Line 1\n@amber fix the bug"
//
// Examples that DO NOT match:
// - "email@amber.com" (no whitespace before @)
// - "@amberlight" (no word boundary after)
// - "Contact @amber123" (no word boundary after)
func (kd *KeywordDetector) DetectKeyword(commentBody string) bool {
	// Case-insensitive search
	lowerBody := strings.ToLower(commentBody)
	return kd.pattern.MatchString(lowerBody)
}

// ExtractCommand extracts the full command following the @amber keyword
// This returns the text after @amber for context in session creation
//
// Example: "@amber review this PR" -> "review this PR"
func (kd *KeywordDetector) ExtractCommand(commentBody string) string {
	lowerBody := strings.ToLower(commentBody)

	// Find the @amber keyword
	match := kd.pattern.FindStringIndex(lowerBody)
	if match == nil {
		return ""
	}

	// Get the position after @amber
	startPos := match[1]

	// Trim @amber prefix from the original body (preserving case)
	// Find @amber in the original body at the same position
	amberPos := strings.Index(strings.ToLower(commentBody), "@amber")
	if amberPos == -1 {
		return ""
	}

	// Extract everything after "@amber "
	afterAmber := commentBody[amberPos+len("@amber"):]
	afterAmber = strings.TrimSpace(afterAmber)

	return afterAmber
}

// IsReviewRequest checks if the comment is explicitly requesting a code review
// This is a helper for Phase 1A to identify review-specific requests
func (kd *KeywordDetector) IsReviewRequest(commentBody string) bool {
	if !kd.DetectKeyword(commentBody) {
		return false
	}

	lowerBody := strings.ToLower(commentBody)
	reviewKeywords := []string{"review", "check", "analyze", "examine", "look at"}

	for _, keyword := range reviewKeywords {
		if strings.Contains(lowerBody, keyword) {
			return true
		}
	}

	// Default to true if @amber is mentioned in a PR context
	return true
}
