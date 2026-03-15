package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestE2E_GetSessionTranscript(t *testing.T) {
	exportData := map[string]interface{}{
		"sessionId":   "session-123",
		"projectName": "test-project",
		"aguiEvents":  []interface{}{},
	}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/export") {
			t.Errorf("Expected path to contain /export, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(exportData)
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/session-123/transcript", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Expected JSON response: %v", err)
	}
	if resp["sessionId"] != "session-123" {
		t.Errorf("Expected sessionId session-123, got %v", resp["sessionId"])
	}
}

func TestE2E_GetSessionLogs(t *testing.T) {
	logOutput := "2026-01-29T10:00:00Z Starting session\n2026-01-29T10:00:01Z Running task\n"
	tailLinesReceived := ""
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/logs") {
			t.Errorf("Expected path to contain /logs, got %s", r.URL.Path)
		}
		tailLinesReceived = r.URL.Query().Get("tailLines")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(logOutput))
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/session-123/logs?tailLines=500", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	if tailLinesReceived != "500" {
		t.Errorf("Expected tailLines=500 forwarded, got %q", tailLinesReceived)
	}

	if !strings.Contains(w.Body.String(), "Starting session") {
		t.Errorf("Expected log output in response, got %s", w.Body.String())
	}
}

func TestE2E_GetSessionMetrics(t *testing.T) {
	metricsData := map[string]interface{}{
		"sessionId":       "session-123",
		"phase":           "Running",
		"durationSeconds": 120,
	}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/metrics") {
			t.Errorf("Expected path to contain /metrics, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(metricsData)
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/session-123/metrics", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Expected JSON response: %v", err)
	}
	if resp["sessionId"] != "session-123" {
		t.Errorf("Expected sessionId session-123, got %v", resp["sessionId"])
	}
}

func TestE2E_GetSessionLogs_InvalidSessionID(t *testing.T) {
	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/INVALID_ID/logs", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid session ID, got %d", w.Code)
	}
}

func TestE2E_GetSessionTranscript_BackendError(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/session-123/transcript", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestE2E_GetSessionLogs_QueryParamForwarding(t *testing.T) {
	queryReceived := ""
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queryReceived = r.URL.RawQuery
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/session-123/logs?tailLines=100&container=runner", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !strings.Contains(queryReceived, "tailLines=100") {
		t.Errorf("Expected tailLines forwarded, got query: %s", queryReceived)
	}
	if !strings.Contains(queryReceived, "container=runner") {
		t.Errorf("Expected container forwarded, got query: %s", queryReceived)
	}
}

func TestE2E_ListSessions_LabelSelector(t *testing.T) {
	queryReceived := ""
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queryReceived = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"items": []interface{}{}})
	}))
	defer backend.Close()

	originalURL := BackendURL
	BackendURL = backend.URL
	defer func() { BackendURL = originalURL }()

	router := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions?labelSelector=env%3Dprod", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Ambient-Project", "test-project")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !strings.Contains(queryReceived, "labelSelector=env%3Dprod") {
		t.Errorf("Expected labelSelector forwarded, got query: %s", queryReceived)
	}
}

func TestFilterUserLabels(t *testing.T) {
	input := map[string]interface{}{
		"env":                          "staging",
		"team":                         "platform",
		"app.kubernetes.io/managed-by": "helm",
		"vteam.ambient-code/session":   "abc",
		"ambient-code.io/runner-sa":    "my-sa",
		"not-a-string":                 12345,
	}

	result := filterUserLabels(input)

	if result["env"] != "staging" {
		t.Errorf("Expected env=staging, got %q", result["env"])
	}
	if result["team"] != "platform" {
		t.Errorf("Expected team=platform, got %q", result["team"])
	}
	if _, ok := result["app.kubernetes.io/managed-by"]; ok {
		t.Error("Expected K8s label to be filtered")
	}
	if _, ok := result["vteam.ambient-code/session"]; ok {
		t.Error("Expected vteam label to be filtered")
	}
	if _, ok := result["ambient-code.io/runner-sa"]; ok {
		t.Error("Expected ambient-code label to be filtered")
	}
	if _, ok := result["not-a-string"]; ok {
		t.Error("Expected non-string value to be filtered")
	}
}

func TestExtractRepos(t *testing.T) {
	tests := []struct {
		name     string
		input    []interface{}
		expected int
		firstURL string
	}{
		{
			name: "Flat format",
			input: []interface{}{
				map[string]interface{}{
					"url":    "https://github.com/org/repo",
					"branch": "main",
				},
			},
			expected: 1,
			firstURL: "https://github.com/org/repo",
		},
		{
			name: "Nested input format",
			input: []interface{}{
				map[string]interface{}{
					"input": map[string]interface{}{
						"url":    "https://github.com/org/repo2",
						"branch": "dev",
					},
				},
			},
			expected: 1,
			firstURL: "https://github.com/org/repo2",
		},
		{
			name:     "Empty repos",
			input:    []interface{}{},
			expected: 0,
		},
		{
			name: "Invalid entry skipped",
			input: []interface{}{
				"not-a-map",
				map[string]interface{}{
					"url": "https://github.com/org/valid",
				},
			},
			expected: 1,
			firstURL: "https://github.com/org/valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRepos(tt.input)
			if len(result) != tt.expected {
				t.Errorf("Expected %d repos, got %d", tt.expected, len(result))
			}
			if tt.expected > 0 && result[0].URL != tt.firstURL {
				t.Errorf("Expected first URL %q, got %q", tt.firstURL, result[0].URL)
			}
		})
	}
}
