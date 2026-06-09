package gitlab

import (
	"fmt"
	"strings"
	"testing"
)

func TestRedactToken_BearerJWT(t *testing.T) {
	jwt := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpM"
	input := "Authorization: Bearer " + jwt
	result := RedactToken(input)
	if strings.Contains(result, jwt) {
		t.Errorf("JWT token was not fully redacted: %s", result)
	}
	if !strings.Contains(result, "Bearer "+TokenRedactionPlaceholder) {
		t.Errorf("expected Bearer [REDACTED], got: %s", result)
	}
}

func TestRedactToken_BearerBase64(t *testing.T) {
	base64Token := "dGhpcyBpcyBhIHRva2VuIHdpdGggcGFkZGluZw=="
	input := "Authorization: Bearer " + base64Token
	result := RedactToken(input)
	if strings.Contains(result, base64Token) {
		t.Errorf("base64 token was not fully redacted: %s", result)
	}
	if !strings.Contains(result, "Bearer "+TokenRedactionPlaceholder) {
		t.Errorf("expected Bearer [REDACTED], got: %s", result)
	}
}

func TestRedactToken_BearerSimple(t *testing.T) {
	input := "Authorization: Bearer ghp_abc123def456"
	result := RedactToken(input)
	if strings.Contains(result, "ghp_abc123def456") {
		t.Errorf("simple token was not redacted: %s", result)
	}
}

func TestSanitizeErrorMessage_RedactsTokenInError(t *testing.T) {
	token := "ghp_secrettoken123"
	err := fmt.Errorf("failed to fetch: Bearer %s was rejected", token)
	result := SanitizeErrorMessage(err)
	if strings.Contains(result, token) {
		t.Errorf("token leaked in sanitized error message: %s", result)
	}
}

func TestSanitizeErrorMessage_NilError(t *testing.T) {
	result := SanitizeErrorMessage(nil)
	if result != "" {
		t.Errorf("expected empty string for nil error, got: %s", result)
	}
}
