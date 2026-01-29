package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestGetTimeoutFromEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue time.Duration
		expected     time.Duration
	}{
		{
			name:         "Default when env not set",
			envValue:     "",
			defaultValue: 30 * time.Second,
			expected:     30 * time.Second,
		},
		{
			name:         "Parse seconds",
			envValue:     "60s",
			defaultValue: 30 * time.Second,
			expected:     60 * time.Second,
		},
		{
			name:         "Parse minutes",
			envValue:     "2m",
			defaultValue: 30 * time.Second,
			expected:     2 * time.Minute,
		},
		{
			name:         "Invalid duration falls back to default",
			envValue:     "invalid",
			defaultValue: 30 * time.Second,
			expected:     30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv("TEST_TIMEOUT", tt.envValue)
				result := getTimeoutFromEnv("TEST_TIMEOUT", tt.defaultValue)
				if result != tt.expected {
					t.Errorf("getTimeoutFromEnv() = %v, want %v", result, tt.expected)
				}
			} else {
				result := getTimeoutFromEnv("NONEXISTENT_ENV_VAR", tt.defaultValue)
				if result != tt.expected {
					t.Errorf("getTimeoutFromEnv() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestProxyRequest_BackendUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Save original and set a non-existent backend
	originalURL := BackendURL
	BackendURL = "http://localhost:99999" // Non-existent port
	defer func() { BackendURL = originalURL }()

	// Set a short timeout for testing
	originalClient := HTTPClient
	HTTPClient = &http.Client{Timeout: 100 * time.Millisecond}
	defer func() { HTTPClient = originalClient }()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	c.Set(ContextKeyToken, "test-token")
	c.Set(ContextKeyProject, "test-project")

	resp, err := ProxyRequest(c, http.MethodGet, "/api/projects/test/sessions", nil)

	if err == nil {
		resp.Body.Close()
		t.Error("Expected error for unavailable backend")
	}
}

func TestProxyAndRespond_BackendUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Save original and set a non-existent backend
	originalURL := BackendURL
	BackendURL = "http://localhost:99999" // Non-existent port
	defer func() { BackendURL = originalURL }()

	// Set a short timeout for testing
	originalClient := HTTPClient
	HTTPClient = &http.Client{Timeout: 100 * time.Millisecond}
	defer func() { HTTPClient = originalClient }()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	c.Set(ContextKeyToken, "test-token")
	c.Set(ContextKeyProject, "test-project")

	ProxyAndRespond(c, http.MethodGet, "/api/projects/test/sessions", nil)

	if w.Code != http.StatusBadGateway {
		t.Errorf("Expected status %d, got %d", http.StatusBadGateway, w.Code)
	}

	// Verify response doesn't contain internal error details
	body := w.Body.String()
	if containsAny(body, "localhost:99999", "connection refused", "dial tcp") {
		t.Errorf("Response should not contain internal error details: %s", body)
	}
}

func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if len(sub) > 0 && len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

func TestGetProject_WrongType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	// Set wrong type in context
	c.Set(ContextKeyProject, 12345) // int instead of string

	// Should return empty string without panicking
	project := GetProject(c)
	if project != "" {
		t.Errorf("Expected empty string for wrong type, got %q", project)
	}
}

func TestGetToken_WrongType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	// Set wrong type in context
	c.Set(ContextKeyToken, []byte("token-bytes")) // []byte instead of string

	// Should return empty string without panicking
	token := GetToken(c)
	if token != "" {
		t.Errorf("Expected empty string for wrong type, got %q", token)
	}
}

func TestGetProject_NotSet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	// Don't set anything in context
	project := GetProject(c)
	if project != "" {
		t.Errorf("Expected empty string when not set, got %q", project)
	}
}

func TestGetToken_NotSet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	// Don't set anything in context
	token := GetToken(c)
	if token != "" {
		t.Errorf("Expected empty string when not set, got %q", token)
	}
}
