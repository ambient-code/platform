package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestE2E_PatchSession_Stop(t *testing.T) {
	methodReceived := ""
	pathReceived := ""
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methodReceived = r.Method
		pathReceived = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":              "session-123",
				"creationTimestamp": "2026-01-29T10:00:00Z",
			},
			"spec": map[string]interface{}{
				"initialPrompt": "Fix the bug",
			},
			"status": map[string]interface{}{
				"phase": "Completed",
			},
		})
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/v1/sessions/session-123",
		strings.NewReader(`{"stopped": true}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	if methodReceived != http.MethodPost {
		t.Errorf("Expected POST to backend, got %s", methodReceived)
	}
	if !strings.HasSuffix(pathReceived, "/stop") {
		t.Errorf("Expected path ending in /stop, got %s", pathReceived)
	}
}

func TestE2E_PatchSession_Start(t *testing.T) {
	pathReceived := ""
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathReceived = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"metadata": map[string]interface{}{
				"name": "session-123",
			},
			"status": map[string]interface{}{
				"phase": "Running",
			},
		})
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/v1/sessions/session-123",
		strings.NewReader(`{"stopped": false}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.HasSuffix(pathReceived, "/start") {
		t.Errorf("Expected path ending in /start, got %s", pathReceived)
	}
}

func TestE2E_PatchSession_Update(t *testing.T) {
	methodReceived := ""
	var receivedBody map[string]interface{}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methodReceived = r.Method
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"metadata": map[string]interface{}{
				"name": "session-123",
			},
			"spec": map[string]interface{}{
				"displayName": "New Name",
				"timeout":     float64(900),
			},
			"status": map[string]interface{}{
				"phase": "Pending",
			},
		})
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/v1/sessions/session-123",
		strings.NewReader(`{"displayName": "New Name", "timeout": 900}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	if methodReceived != http.MethodPut {
		t.Errorf("Expected PUT, got %s", methodReceived)
	}

	// Verify the backend received the correct fields
	if receivedBody["displayName"] != "New Name" {
		t.Errorf("Expected displayName 'New Name', got %v", receivedBody["displayName"])
	}
	if receivedBody["timeout"] != float64(900) {
		t.Errorf("Expected timeout 900, got %v", receivedBody["timeout"])
	}
}

func TestE2E_PatchSession_Labels(t *testing.T) {
	var receivedBody map[string]interface{}
	patchReceived := false
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPatch {
			patchReceived = true
			json.NewDecoder(r.Body).Decode(&receivedBody)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message":     "Session patched successfully",
				"annotations": map[string]string{"env": "prod"},
			})
		} else if r.Method == http.MethodGet {
			// Follow-up GET to return full session DTO
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":              "session-123",
					"creationTimestamp": "2026-03-04T10:00:00Z",
					"annotations":       map[string]interface{}{"env": "prod"},
				},
				"spec":   map[string]interface{}{"initialPrompt": "test"},
				"status": map[string]interface{}{"phase": "Running"},
			})
		}
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/v1/sessions/session-123",
		strings.NewReader(`{"labels": {"env": "prod"}, "removeLabels": ["old-label"]}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	if !patchReceived {
		t.Errorf("Expected PATCH to be sent to backend")
	}

	// Verify transformation to annotation format
	metadata, ok := receivedBody["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected metadata in request body")
	}
	annotations, ok := metadata["annotations"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected annotations in metadata")
	}
	if annotations["env"] != "prod" {
		t.Errorf("Expected annotation env=prod, got %v", annotations["env"])
	}
	if annotations["old-label"] != nil {
		t.Errorf("Expected annotation old-label=nil, got %v", annotations["old-label"])
	}

	// Verify response is a full session DTO (from follow-up GET)
	var respBody map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &respBody)
	if respBody["id"] != "session-123" {
		t.Errorf("Expected session id in response, got %v", respBody["id"])
	}
}

func TestE2E_PatchSession_ReservedLabelPrefix(t *testing.T) {
	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/v1/sessions/session-123",
		strings.NewReader(`{"labels": {"ambient-code.io/evil": "injected"}}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for reserved label prefix, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "reserved prefix") {
		t.Errorf("Expected error about reserved prefix, got: %s", body)
	}
}

func TestE2E_PatchSession_EmptyBody(t *testing.T) {
	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/v1/sessions/session-123",
		strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty body, got %d: %s", w.Code, w.Body.String())
	}
}

func TestE2E_PatchSession_MixedCategories(t *testing.T) {
	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/v1/sessions/session-123",
		strings.NewReader(`{"stopped": true, "displayName": "New Name"}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for mixed categories, got %d: %s", w.Code, w.Body.String())
	}
}
