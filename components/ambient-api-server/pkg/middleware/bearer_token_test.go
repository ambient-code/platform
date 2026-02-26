package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
		wantErr   bool
	}{
		{"valid lowercase", "bearer my-token", "my-token", false},
		{"valid titlecase", "Bearer my-token", "my-token", false},
		{"valid uppercase", "BEARER my-token", "my-token", false},
		{"valid mixed case", "BeArEr my-token", "my-token", false},
		{"valid with extra whitespace", "Bearer   my-token  ", "my-token", false},
		{"missing scheme", "my-token", "", true},
		{"empty string", "", "", true},
		{"wrong scheme", "Basic dXNlcjpwYXNz", "", true},
		{"bearer only no token", "Bearer ", "", true},
		{"bearer only no space", "Bearer", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := extractBearerToken(tt.header)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractBearerToken(%q) error = %v, wantErr %v", tt.header, err, tt.wantErr)
				return
			}
			if token != tt.wantToken {
				t.Errorf("extractBearerToken(%q) = %q, want %q", tt.header, token, tt.wantToken)
			}
		})
	}
}

func TestBearerTokenAuth(t *testing.T) {
	const validToken = "test-secret-token"
	handler := BearerTokenAuth(validToken)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		path       string
		authHeader string
		wantCode   int
	}{
		{"valid token", "/api/v1/sessions", "Bearer " + validToken, http.StatusOK},
		{"valid token lowercase", "/api/v1/sessions", "bearer " + validToken, http.StatusOK},
		{"valid token uppercase", "/api/v1/sessions", "BEARER " + validToken, http.StatusOK},
		{"invalid token", "/api/v1/sessions", "Bearer wrong-token", http.StatusUnauthorized},
		{"missing header", "/api/v1/sessions", "", http.StatusUnauthorized},
		{"wrong scheme", "/api/v1/sessions", "Basic dXNlcjpwYXNz", http.StatusUnauthorized},
		{"bypass healthcheck", "/healthcheck", "", http.StatusOK},
		{"bypass health", "/health", "", http.StatusOK},
		{"bypass metrics", "/metrics", "", http.StatusOK},
		{"no bypass for api paths", "/api/v1/users", "", http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.wantCode {
				t.Errorf("path=%q auth=%q: got %d, want %d", tt.path, tt.authHeader, rec.Code, tt.wantCode)
			}
		})
	}
}
