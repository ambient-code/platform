package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupTestRouter creates a test router with the same middleware as production
func setupTestRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	v1 := r.Group("/v1")
	v1.Use(AuthMiddleware())
	v1.Use(LoggingMiddleware())
	{
		v1.GET("/sessions", ListSessions)
		v1.POST("/sessions", CreateSession)
		v1.GET("/sessions/:id", GetSession)
		v1.DELETE("/sessions/:id", DeleteSession)
		v1.POST("/sessions/:id/runs", CreateSessionRun)
	}

	return r
}

func TestE2E_TokenForwarding(t *testing.T) {
	// Start mock backend that verifies token forwarding
	tokenReceived := ""
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenReceived = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"items": []interface{}{}})
	}))
	defer backend.Close()

	// Configure handler to use mock backend
	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	// Make request with test token
	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	req.Header.Set("Authorization", "Bearer test-token-12345")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	// Verify token was forwarded correctly
	if tokenReceived != "Bearer test-token-12345" {
		t.Errorf("Token not forwarded correctly, got %q", tokenReceived)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestE2E_CreateSession(t *testing.T) {
	// Start mock backend
	requestBody := ""
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Verify path contains project
		if !strings.Contains(r.URL.Path, "/test-project/") {
			t.Errorf("Expected path to contain project, got %s", r.URL.Path)
		}

		// Read body
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		requestBody = string(buf[:n])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"name": "session-123"})
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions",
		strings.NewReader(`{"task": "Fix the bug"}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify request body was transformed correctly
	if !strings.Contains(requestBody, "prompt") {
		t.Errorf("Expected request body to contain 'prompt', got %s", requestBody)
	}
}

func TestE2E_BackendReturns500(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Database connection failed"})
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	// Should forward 500 status
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	// Should forward error message
	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["error"] != "Database connection failed" {
		t.Errorf("Expected forwarded error message, got %v", response)
	}
}

func TestE2E_BackendReturns404(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Session not found"})
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/test-session", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestE2E_InvalidSessionID(t *testing.T) {
	router := setupTestRouter()

	tests := []struct {
		name      string
		sessionID string
	}{
		{"uppercase", "Session-123"},
		{"underscore", "session_123"},
		{"special chars", "session@123"},
		{"starts with hyphen", "-session"},
		{"ends with hyphen", "session-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+tt.sessionID, nil)
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("X-Ambient-Project", "test-project")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for invalid session ID %q, got %d", tt.sessionID, w.Code)
			}
		})
	}
}

func TestE2E_InvalidProjectName(t *testing.T) {
	router := setupTestRouter()
	w := httptest.NewRecorder()

	// Use invalid project name with uppercase
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "INVALID_PROJECT")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid project name, got %d", w.Code)
	}
}

func TestE2E_ProjectMismatchAttack(t *testing.T) {
	// This test verifies that if an attacker provides a forged token
	// with a different project than the header, the request is rejected

	router := setupTestRouter()
	w := httptest.NewRecorder()

	// Create a valid-looking JWT with a different project in the sub claim
	// Header says "my-project" but token claims "attacker-project"
	// JWT payload: {"sub": "system:serviceaccount:attacker-project:my-sa"}
	// Base64 of that payload
	forgedToken := "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJzeXN0ZW06c2VydmljZWFjY291bnQ6YXR0YWNrZXItcHJvamVjdDpteS1zYSJ9.signature"

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+forgedToken)
	req.Header.Set("X-Ambient-Project", "my-project") // Different from token!
	router.ServeHTTP(w, req)

	// Should reject due to project mismatch
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for project mismatch, got %d: %s", w.Code, w.Body.String())
	}
}

func TestE2E_CreateSessionRun(t *testing.T) {
	tests := []struct {
		name           string
		sessionID      string
		requestBody    string
		backendStatus  int
		backendResp    string
		expectedStatus int
		expectRunID    bool
	}{
		{
			name:           "Successful run creation",
			sessionID:      "test-session",
			requestBody:    `{"prompt": "Fix the authentication bug"}`,
			backendStatus:  http.StatusOK,
			backendResp:    `{"runId": "run-abc123", "threadId": "test-session"}`,
			expectedStatus: http.StatusOK,
			expectRunID:    true,
		},
		{
			name:           "Missing prompt returns 400",
			sessionID:      "test-session",
			requestBody:    `{}`,
			backendStatus:  http.StatusOK,
			backendResp:    `{}`,
			expectedStatus: http.StatusBadRequest,
			expectRunID:    false,
		},
		{
			name:           "Backend error forwarded",
			sessionID:      "test-session",
			requestBody:    `{"prompt": "Do something"}`,
			backendStatus:  http.StatusBadRequest,
			backendResp:    `{"error": "Session not running"}`,
			expectedStatus: http.StatusBadRequest,
			expectRunID:    false,
		},
		{
			name:           "Invalid session ID returns 400",
			sessionID:      "INVALID-SESSION",
			requestBody:    `{"prompt": "Do something"}`,
			backendStatus:  http.StatusOK,
			backendResp:    `{}`,
			expectedStatus: http.StatusBadRequest,
			expectRunID:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify path contains agui/run for valid requests
				if tt.expectedStatus != http.StatusBadRequest || tt.backendStatus != http.StatusOK {
					if !strings.Contains(r.URL.Path, "agui/run") {
						t.Errorf("Expected path to contain agui/run, got %s", r.URL.Path)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.backendStatus)
				w.Write([]byte(tt.backendResp))
			}))
			defer backend.Close()

			originalURL := BackendURL
			BackendURL = backend.URL
			defer func() { BackendURL = originalURL }()

			router := setupTestRouter()
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+tt.sessionID+"/runs",
				strings.NewReader(tt.requestBody))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("X-Ambient-Project", "test-project")
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectRunID {
				var resp map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				if resp["runId"] == "" {
					t.Errorf("Expected runId in response, got %v", resp)
				}
				if resp["threadId"] == "" {
					t.Errorf("Expected threadId in response, got %v", resp)
				}
			}
		})
	}
}

func TestE2E_CreateSessionRun_MessageFormat(t *testing.T) {
	// Verify the AG-UI message format sent to the backend
	var capturedBody map[string]interface{}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"runId": "run-xyz", "threadId": "my-session"}`))
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/my-session/runs",
		strings.NewReader(`{"prompt": "Add unit tests"}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify AG-UI format: threadId and messages array
	if capturedBody["threadId"] != "my-session" {
		t.Errorf("Expected threadId=my-session, got %v", capturedBody["threadId"])
	}
	messages, ok := capturedBody["messages"].([]interface{})
	if !ok || len(messages) != 1 {
		t.Fatalf("Expected 1 message in AG-UI format, got %v", capturedBody["messages"])
	}
	msg, ok := messages[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected message to be an object, got %T", messages[0])
	}
	if msg["role"] != "user" {
		t.Errorf("Expected role=user, got %v", msg["role"])
	}
	if msg["content"] != "Add unit tests" {
		t.Errorf("Expected content='Add unit tests', got %v", msg["content"])
	}
	if msg["id"] == "" {
		t.Error("Expected non-empty message id")
	}
}

func TestE2E_DeleteSession(t *testing.T) {
	deleted := false
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/v1/sessions/test-session", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	if !deleted {
		t.Error("Expected backend delete to be called")
	}
}
