package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestExtractToken(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		forwardedToken string
		expectedToken  string
	}{
		{
			name:          "Bearer token from Authorization header",
			authHeader:    "Bearer test-token-123",
			expectedToken: "test-token-123",
		},
		{
			name:           "Token from X-Forwarded-Access-Token",
			forwardedToken: "forwarded-token-456",
			expectedToken:  "forwarded-token-456",
		},
		{
			name:           "Authorization header takes precedence",
			authHeader:     "Bearer auth-token",
			forwardedToken: "forwarded-token",
			expectedToken:  "auth-token",
		},
		{
			name:          "No token",
			expectedToken: "",
		},
		{
			name:          "Invalid Authorization header format",
			authHeader:    "Basic dXNlcjpwYXNz",
			expectedToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

			if tt.authHeader != "" {
				c.Request.Header.Set("Authorization", tt.authHeader)
			}
			if tt.forwardedToken != "" {
				c.Request.Header.Set("X-Forwarded-Access-Token", tt.forwardedToken)
			}

			token := extractToken(c)
			if token != tt.expectedToken {
				t.Errorf("extractToken() = %q, want %q", token, tt.expectedToken)
			}
		})
	}
}

func TestExtractProject(t *testing.T) {
	tests := []struct {
		name            string
		projectHeader   string
		token           string
		expectedProject string
	}{
		{
			name:            "Project from header only",
			projectHeader:   "my-project",
			expectedProject: "my-project",
		},
		{
			name:            "Header matches token project",
			projectHeader:   "token-project",
			token:           "eyJhbGciOiJSUzI1NiIsImtpZCI6IiJ9.eyJzdWIiOiJzeXN0ZW06c2VydmljZWFjY291bnQ6dG9rZW4tcHJvamVjdDp0ZXN0LXNhIn0.signature",
			expectedProject: "token-project",
		},
		{
			name:            "Header and token mismatch returns empty (security)",
			projectHeader:   "header-project",
			token:           "eyJhbGciOiJSUzI1NiIsImtpZCI6IiJ9.eyJzdWIiOiJzeXN0ZW06c2VydmljZWFjY291bnQ6dG9rZW4tcHJvamVjdDp0ZXN0LXNhIn0.signature",
			expectedProject: "", // Security: mismatch should fail
		},
		{
			name:            "Project from ServiceAccount token",
			token:           "eyJhbGciOiJSUzI1NiIsImtpZCI6IiJ9.eyJzdWIiOiJzeXN0ZW06c2VydmljZWFjY291bnQ6bXktbmFtZXNwYWNlOnRlc3Qtc2EifQ.signature",
			expectedProject: "my-namespace",
		},
		{
			name:            "No project available",
			expectedProject: "",
		},
		{
			name:            "Header only with non-SA token",
			projectHeader:   "my-project",
			token:           "not-a-jwt-token",
			expectedProject: "my-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

			if tt.projectHeader != "" {
				c.Request.Header.Set("X-Ambient-Project", tt.projectHeader)
			}

			project := extractProject(c, tt.token)
			if project != tt.expectedProject {
				t.Errorf("extractProject() = %q, want %q", project, tt.expectedProject)
			}
		})
	}
}

func TestExtractJWTSubject(t *testing.T) {
	tests := []struct {
		name            string
		token           string
		expectedSubject string
	}{
		{
			name:            "Valid ServiceAccount JWT",
			token:           "eyJhbGciOiJSUzI1NiIsImtpZCI6IiJ9.eyJzdWIiOiJzeXN0ZW06c2VydmljZWFjY291bnQ6dGVzdC1uczp0ZXN0LXNhIn0.signature",
			expectedSubject: "system:serviceaccount:test-ns:test-sa",
		},
		{
			name:            "Invalid token format",
			token:           "not-a-jwt",
			expectedSubject: "",
		},
		{
			name:            "Empty token",
			token:           "",
			expectedSubject: "",
		},
		{
			name:            "Token with invalid base64",
			token:           "header.!!!invalid!!!.signature",
			expectedSubject: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subject := extractJWTSubject(tt.token)
			if subject != tt.expectedSubject {
				t.Errorf("extractJWTSubject() = %q, want %q", subject, tt.expectedSubject)
			}
		})
	}
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)

	handler := AuthMiddleware()
	handler(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_NoProject(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	c.Request.Header.Set("Authorization", "Bearer some-token-without-project-info")

	handler := AuthMiddleware()
	handler(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestRedactSensitiveParams(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		rawQuery string
		expected string
	}{
		{
			name:     "No query params",
			path:     "/v1/sessions",
			rawQuery: "",
			expected: "/v1/sessions",
		},
		{
			name:     "No sensitive params",
			path:     "/v1/sessions",
			rawQuery: "limit=10&offset=0",
			expected: "/v1/sessions?limit=10&offset=0",
		},
		{
			name:     "Redact token param",
			path:     "/v1/sessions",
			rawQuery: "token=secret123",
			expected: "/v1/sessions?token=[REDACTED]",
		},
		{
			name:     "Redact token in middle",
			path:     "/v1/sessions",
			rawQuery: "limit=10&token=secret123&offset=0",
			expected: "/v1/sessions?limit=10&token=[REDACTED]&offset=0",
		},
		{
			name:     "Redact access_token",
			path:     "/oauth/callback",
			rawQuery: "access_token=abc123&state=xyz",
			expected: "/oauth/callback?access_token=[REDACTED]&state=xyz",
		},
		{
			name:     "Redact api_key",
			path:     "/v1/sessions",
			rawQuery: "api_key=key123",
			expected: "/v1/sessions?api_key=[REDACTED]",
		},
		{
			name:     "Redact multiple sensitive params",
			path:     "/v1/sessions",
			rawQuery: "token=abc&api_key=def",
			expected: "/v1/sessions?token=[REDACTED]&api_key=[REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactSensitiveParams(tt.path, tt.rawQuery)
			if result != tt.expected {
				t.Errorf("redactSensitiveParams() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRedactQueryParam(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		param    string
		expected string
	}{
		{
			name:     "Param at end",
			query:    "foo=bar&token=secret",
			param:    "token",
			expected: "foo=bar&token=[REDACTED]",
		},
		{
			name:     "Param at start",
			query:    "token=secret&foo=bar",
			param:    "token",
			expected: "token=[REDACTED]&foo=bar",
		},
		{
			name:     "Param in middle",
			query:    "a=1&token=secret&b=2",
			param:    "token",
			expected: "a=1&token=[REDACTED]&b=2",
		},
		{
			name:     "Param not found",
			query:    "foo=bar",
			param:    "token",
			expected: "foo=bar",
		},
		{
			name:     "Only param",
			query:    "token=secret",
			param:    "token",
			expected: "token=[REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactQueryParam(tt.query, tt.param)
			if result != tt.expected {
				t.Errorf("redactQueryParam() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestAuthMiddleware_ValidRequest(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	c.Request.Header.Set("Authorization", "Bearer test-token")
	c.Request.Header.Set("X-Ambient-Project", "my-project")

	handler := AuthMiddleware()
	handler(c)

	// Middleware should not abort
	if c.IsAborted() {
		t.Error("Expected middleware to not abort")
	}

	// Check context values
	if GetToken(c) != "test-token" {
		t.Errorf("Expected token 'test-token', got %q", GetToken(c))
	}
	if GetProject(c) != "my-project" {
		t.Errorf("Expected project 'my-project', got %q", GetProject(c))
	}
}
