package gitlab

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
)

// TokenRedactionPlaceholder is used to replace sensitive tokens in logs
const TokenRedactionPlaceholder = "[REDACTED]"

var (
	gitlabPATPattern = regexp.MustCompile(`glpat-[a-zA-Z0-9_-]+`)
	gitlabCIPattern  = regexp.MustCompile(`gitlab-ci-token:\s*[a-zA-Z0-9_-]+`)
	bearerPattern    = regexp.MustCompile(`Bearer\s+\S+`)
	oauthURLPattern  = regexp.MustCompile(`oauth2:[^@]+@`)
	tokenURLPattern  = regexp.MustCompile(`://[^:]+:[^@]+@`)
)

// RedactToken removes sensitive token information from a string
func RedactToken(s string) string {
	s = gitlabPATPattern.ReplaceAllString(s, TokenRedactionPlaceholder)
	s = gitlabCIPattern.ReplaceAllString(s, "gitlab-ci-token: "+TokenRedactionPlaceholder)
	s = bearerPattern.ReplaceAllString(s, "Bearer "+TokenRedactionPlaceholder)
	s = oauthURLPattern.ReplaceAllString(s, "oauth2:"+TokenRedactionPlaceholder+"@")
	s = tokenURLPattern.ReplaceAllString(s, "://"+TokenRedactionPlaceholder+":"+TokenRedactionPlaceholder+"@")

	return s
}

// LogInfo logs an informational message with token redaction
func LogInfo(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	redacted := RedactToken(message)
	log.Printf("[GitLab] INFO: %s", redacted)
}

// LogWarning logs a warning message with token redaction
func LogWarning(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	redacted := RedactToken(message)
	log.Printf("[GitLab] WARNING: %s", redacted)
}

// LogError logs an error message with token redaction
func LogError(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	redacted := RedactToken(message)
	log.Printf("[GitLab] ERROR: %s", redacted)
}

// RedactURL removes sensitive information from a Git URL
// Handles both GitLab (oauth2:token@) and GitHub (x-access-token:token@) formats
func RedactURL(gitURL string) string {
	// Parse the URL properly instead of string splitting
	parsedURL, err := url.Parse(gitURL)
	if err != nil {
		// If parsing fails, fall back to regex-based redaction
		return RedactToken(gitURL)
	}

	// Check if URL contains user info (credentials)
	if parsedURL.User != nil {
		// Redact the entire userinfo part (handles oauth2:token, x-access-token:token, etc.)
		parsedURL.User = url.User(TokenRedactionPlaceholder)
	}

	return parsedURL.String()
}

// SanitizeErrorMessage removes sensitive information from error messages
func SanitizeErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	message := err.Error()
	return RedactToken(message)
}
