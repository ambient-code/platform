package client

import (
	"log/slog"
	"strings"
	"testing"
)

func TestSecureToken_IsValid_OpenShift(t *testing.T) {
	token := SecureToken("sha256~abcdefghijklmnopqrstuvwxyz1234567890")
	if err := token.IsValid(); err != nil {
		t.Errorf("expected valid OpenShift token, got: %v", err)
	}
}

func TestSecureToken_IsValid_JWT(t *testing.T) {
	token := SecureToken("eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.signature_data_here")
	if err := token.IsValid(); err != nil {
		t.Errorf("expected valid JWT token, got: %v", err)
	}
}

func TestSecureToken_IsValid_GitHub(t *testing.T) {
	token := SecureToken("ghp_abcdefghijklmnopqrstuvwxyz1234567890abcd")
	if err := token.IsValid(); err != nil {
		t.Errorf("expected valid GitHub token, got: %v", err)
	}
}

func TestSecureToken_Invalid_Empty(t *testing.T) {
	token := SecureToken("")
	if err := token.IsValid(); err == nil {
		t.Error("expected error for empty token")
	}
}

func TestSecureToken_Invalid_TooShort(t *testing.T) {
	token := SecureToken("abc")
	if err := token.IsValid(); err == nil {
		t.Error("expected error for short token")
	}
}

func TestSecureToken_Invalid_Placeholder(t *testing.T) {
	placeholders := []string{"YOUR_TOKEN_HERE", "token", "password", "secret", "placeholder"}
	for _, p := range placeholders {
		token := SecureToken(p)
		if err := token.IsValid(); err == nil {
			t.Errorf("expected error for placeholder %q", p)
		}
	}
}

func TestSecureToken_Invalid_ShortOpenShift(t *testing.T) {
	token := SecureToken("sha256~short")
	if err := token.IsValid(); err == nil {
		t.Error("expected error for short OpenShift token")
	}
}

func TestSecureToken_Invalid_ShortGitHub(t *testing.T) {
	token := SecureToken("ghp_short")
	if err := token.IsValid(); err == nil {
		t.Error("expected error for short GitHub token")
	}
}

func TestSecureToken_LogValue_Normal(t *testing.T) {
	token := SecureToken("sha256~abcdefghijklmnopqrstuvwxyz")
	logVal := token.LogValue()
	str := logVal.String()
	if !strings.HasPrefix(str, "sha256") {
		t.Errorf("expected log value to start with partial token, got: %s", str)
	}
	if !strings.Contains(str, "***") {
		t.Errorf("expected log value to contain ***, got: %s", str)
	}
	if strings.Contains(str, "abcdefghijklmnopqrstuvwxyz") {
		t.Error("log value should NOT contain full token")
	}
}

func TestSecureToken_LogValue_Empty(t *testing.T) {
	token := SecureToken("")
	logVal := token.LogValue()
	if logVal.String() != "[EMPTY]" {
		t.Errorf("expected [EMPTY], got: %s", logVal.String())
	}
}

func TestSecureToken_LogValue_TooShort(t *testing.T) {
	token := SecureToken("abc")
	logVal := token.LogValue()
	if logVal.String() != "[TOO_SHORT]" {
		t.Errorf("expected [TOO_SHORT], got: %s", logVal.String())
	}
}

func TestSanitizeLogAttrs_TokenRedaction(t *testing.T) {
	tests := []struct {
		key    string
		value  string
		expect string
	}{
		{"token", "secret-value", "[REDACTED]"},
		{"Token", "secret-value", "[REDACTED]"},
		{"TOKEN", "secret-value", "[REDACTED]"},
		{"password", "secret-value", "[REDACTED]"},
		{"secret", "secret-value", "[REDACTED]"},
		{"api_key", "secret-value", "[REDACTED]"},
		{"authorization", "secret-value", "[REDACTED]"},
		{"auth_token", "secret-value", "[REDACTED]"},
		{"db_password", "secret-value", "[REDACTED]"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			attr := slog.String(tt.key, tt.value)
			result := sanitizeLogAttrs(nil, attr)
			if result.Value.String() != tt.expect {
				t.Errorf("key=%q: got %q, want %q", tt.key, result.Value.String(), tt.expect)
			}
		})
	}
}

func TestSanitizeLogAttrs_BearerRedaction(t *testing.T) {
	attr := slog.String("header", "Bearer eyJhbGciOiJSUzI1NiJ9.xxx.yyy")
	result := sanitizeLogAttrs(nil, attr)
	if result.Value.String() != "[REDACTED_BEARER]" {
		t.Errorf("got %q, want [REDACTED_BEARER]", result.Value.String())
	}
}

func TestSanitizeLogAttrs_SHA256Redaction(t *testing.T) {
	attr := slog.String("value", "sha256~abcdefghijklmnop")
	result := sanitizeLogAttrs(nil, attr)
	if result.Value.String() != "[REDACTED_SHA256_TOKEN]" {
		t.Errorf("got %q, want [REDACTED_SHA256_TOKEN]", result.Value.String())
	}
}

func TestSanitizeLogAttrs_JWTRedaction(t *testing.T) {
	jwt := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0In0.long_signature_data"
	attr := slog.String("data", jwt)
	result := sanitizeLogAttrs(nil, attr)
	if result.Value.String() != "[REDACTED_JWT]" {
		t.Errorf("got %q, want [REDACTED_JWT]", result.Value.String())
	}
}

func TestSanitizeLogAttrs_SafeValue(t *testing.T) {
	attr := slog.String("name", "my-session")
	result := sanitizeLogAttrs(nil, attr)
	if result.Value.String() != "my-session" {
		t.Errorf("safe value should not be redacted, got %q", result.Value.String())
	}
}

func TestNewClient_ValidInputs(t *testing.T) {
	c, err := NewClient(
		"https://api.example-platform.com",
		"sha256~abcdefghijklmnopqrstuvwxyz1234567890",
		"my-project",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.baseURL != "https://api.example-platform.com" {
		t.Errorf("got baseURL=%q", c.baseURL)
	}
	if c.project != "my-project" {
		t.Errorf("got project=%q", c.project)
	}
	if c.basePath != "/api/ambient-api-server/v1" {
		t.Errorf("got basePath=%q", c.basePath)
	}
}

func TestNewClient_TrailingSlash(t *testing.T) {
	c, err := NewClient(
		"https://api.example-platform.com/",
		"sha256~abcdefghijklmnopqrstuvwxyz1234567890",
		"my-project",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.baseURL != "https://api.example-platform.com" {
		t.Errorf("trailing slash not stripped: got %q", c.baseURL)
	}
}

func TestNewClient_InvalidToken(t *testing.T) {
	_, err := NewClient("https://api.test.com", "bad", "project")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestNewClient_EmptyProject(t *testing.T) {
	_, err := NewClient("https://api.test.com", "sha256~abcdefghijklmnopqrstuvwxyz1234567890", "")
	if err == nil {
		t.Fatal("expected error for empty project")
	}
}

func TestNewClient_WithBasePath(t *testing.T) {
	c, err := NewClient(
		"https://api.test.com",
		"sha256~abcdefghijklmnopqrstuvwxyz1234567890",
		"project",
		WithBasePath("/custom/v2"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.basePath != "/custom/v2" {
		t.Errorf("got basePath=%q, want %q", c.basePath, "/custom/v2")
	}
}

func TestSessionAPI_WP6ActionMethods(t *testing.T) {
	c, err := NewClient(
		"https://api.test.com",
		"sha256~abcdefghijklmnopqrstuvwxyz1234567890",
		"project",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	api := c.Sessions()
	_ = api.Start
	_ = api.Stop
	_ = api.UpdateStatus
	_ = api.Create
	_ = api.Get
	_ = api.List
	_ = api.Update
	_ = api.ListAll
}
