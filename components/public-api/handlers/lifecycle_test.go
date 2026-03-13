package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ambient-code-public-api/types"
)

func makeSessionBackend(t *testing.T, phase string, lifecyclePath string, lifecycleStatus int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// GET session (for phase check)
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/agentic-sessions/") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":              "test-session",
					"creationTimestamp": "2026-01-29T10:00:00Z",
				},
				"spec": map[string]interface{}{
					"prompt": "Test task",
				},
				"status": map[string]interface{}{
					"phase": phase,
				},
			})
			return
		}

		// POST lifecycle action
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, lifecyclePath) {
			w.WriteHeader(lifecycleStatus)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":              "test-session",
					"creationTimestamp": "2026-01-29T10:00:00Z",
				},
				"spec": map[string]interface{}{
					"prompt": "Test task",
				},
				"status": map[string]interface{}{
					"phase": "Running",
				},
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Not found"})
	}))
}

func TestE2E_StartSession(t *testing.T) {
	backend := makeSessionBackend(t, "Completed", "/start", http.StatusOK)
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/test-session/start", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d: %s", w.Code, w.Body.String())
	}

	var response types.SessionResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.ID != "test-session" {
		t.Errorf("Expected ID 'test-session', got %q", response.ID)
	}
}

func TestE2E_StartSession_AlreadyRunning(t *testing.T) {
	backend := makeSessionBackend(t, "Running", "/start", http.StatusOK)
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/test-session/start", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestE2E_StartSession_AlreadyPending(t *testing.T) {
	backend := makeSessionBackend(t, "Pending", "/start", http.StatusOK)
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/test-session/start", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422 for pending session, got %d", w.Code)
	}
}

func TestE2E_StopSession(t *testing.T) {
	backend := makeSessionBackend(t, "Running", "/stop", http.StatusOK)
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/test-session/stop", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d: %s", w.Code, w.Body.String())
	}
}

func TestE2E_StopSession_AlreadyCompleted(t *testing.T) {
	backend := makeSessionBackend(t, "Completed", "/stop", http.StatusOK)
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/test-session/stop", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestE2E_StopSession_AlreadyFailed(t *testing.T) {
	backend := makeSessionBackend(t, "Failed", "/stop", http.StatusOK)
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/test-session/stop", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422 for failed session, got %d", w.Code)
	}
}

func TestE2E_InterruptSession(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/agui/interrupt") {
			t.Errorf("Expected path to contain /agui/interrupt, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/test-session/interrupt", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response types.MessageResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Message != "Interrupt signal sent" {
		t.Errorf("Expected message 'Interrupt signal sent', got %q", response.Message)
	}
}

func TestE2E_InterruptSession_BackendError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": "Runner unavailable"})
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/test-session/interrupt", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("Expected status 502, got %d", w.Code)
	}
}

func TestE2E_StartSession_InvalidSessionID(t *testing.T) {
	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/INVALID/start", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestE2E_StopSession_BackendReturns404(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/nonexistent/stop", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}
