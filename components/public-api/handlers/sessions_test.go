package handlers

import (
	"net/http/httptest"
	"testing"

	"ambient-code-public-api/types"

	"github.com/gin-gonic/gin"
)

func TestTransformSession(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected types.SessionResponse
	}{
		{
			name: "Full session with metadata and status",
			input: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":              "session-123",
					"creationTimestamp": "2026-01-29T10:00:00Z",
					"annotations": map[string]interface{}{
						"env":                  "staging",
						"app.kubernetes.io/foo": "bar", // should be filtered
					},
				},
				"spec": map[string]interface{}{
					"initialPrompt": "Fix the bug",
					"displayName":   "Bug Fix Session",
					"timeout":       float64(600),
					"llmSettings": map[string]interface{}{
						"model": "claude-sonnet-4",
					},
					"repos": []interface{}{
						map[string]interface{}{
							"url":    "https://github.com/org/repo",
							"branch": "main",
						},
					},
				},
				"status": map[string]interface{}{
					"phase":          "Running",
					"completionTime": "",
				},
			},
			expected: types.SessionResponse{
				ID:          "session-123",
				Status:      "running",
				Task:        "Fix the bug",
				Model:       "claude-sonnet-4",
				DisplayName: "Bug Fix Session",
				Timeout:     600,
				CreatedAt:   "2026-01-29T10:00:00Z",
				Repos:       []types.Repo{{URL: "https://github.com/org/repo", Branch: "main"}},
				Labels:      map[string]string{"env": "staging"},
			},
		},
		{
			name: "Completed session with result",
			input: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":              "session-456",
					"creationTimestamp": "2026-01-29T09:00:00Z",
				},
				"spec": map[string]interface{}{
					"prompt": "Refactor code",
				},
				"status": map[string]interface{}{
					"phase":          "Completed",
					"completionTime": "2026-01-29T09:30:00Z",
					"result":         "Successfully refactored",
				},
			},
			expected: types.SessionResponse{
				ID:          "session-456",
				Status:      "completed",
				Task:        "Refactor code",
				CreatedAt:   "2026-01-29T09:00:00Z",
				CompletedAt: "2026-01-29T09:30:00Z",
				Result:      "Successfully refactored",
			},
		},
		{
			name: "Failed session with error",
			input: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":              "session-789",
					"creationTimestamp": "2026-01-29T08:00:00Z",
				},
				"spec": map[string]interface{}{
					"prompt": "Do something",
				},
				"status": map[string]interface{}{
					"phase": "Failed",
					"error": "Something went wrong",
				},
			},
			expected: types.SessionResponse{
				ID:        "session-789",
				Status:    "failed",
				Task:      "Do something",
				CreatedAt: "2026-01-29T08:00:00Z",
				Error:     "Something went wrong",
			},
		},
		{
			name: "List response format (name at top level)",
			input: map[string]interface{}{
				"name": "session-list-item",
				"spec": map[string]interface{}{
					"prompt": "List item task",
				},
				"status": map[string]interface{}{
					"phase": "Pending",
				},
			},
			expected: types.SessionResponse{
				ID:     "session-list-item",
				Status: "pending",
				Task:   "List item task",
			},
		},
		{
			name:  "Empty session",
			input: map[string]interface{}{},
			expected: types.SessionResponse{
				Status: "pending",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformSession(tt.input)

			if result.ID != tt.expected.ID {
				t.Errorf("ID = %q, want %q", result.ID, tt.expected.ID)
			}
			if result.Status != tt.expected.Status {
				t.Errorf("Status = %q, want %q", result.Status, tt.expected.Status)
			}
			if result.Task != tt.expected.Task {
				t.Errorf("Task = %q, want %q", result.Task, tt.expected.Task)
			}
			if result.Model != tt.expected.Model {
				t.Errorf("Model = %q, want %q", result.Model, tt.expected.Model)
			}
			if result.DisplayName != tt.expected.DisplayName {
				t.Errorf("DisplayName = %q, want %q", result.DisplayName, tt.expected.DisplayName)
			}
			if result.CreatedAt != tt.expected.CreatedAt {
				t.Errorf("CreatedAt = %q, want %q", result.CreatedAt, tt.expected.CreatedAt)
			}
			if result.CompletedAt != tt.expected.CompletedAt {
				t.Errorf("CompletedAt = %q, want %q", result.CompletedAt, tt.expected.CompletedAt)
			}
			if result.Result != tt.expected.Result {
				t.Errorf("Result = %q, want %q", result.Result, tt.expected.Result)
			}
			if result.Error != tt.expected.Error {
				t.Errorf("Error = %q, want %q", result.Error, tt.expected.Error)
			}
			if result.Timeout != tt.expected.Timeout {
				t.Errorf("Timeout = %d, want %d", result.Timeout, tt.expected.Timeout)
			}
			if len(result.Repos) != len(tt.expected.Repos) {
				t.Errorf("Repos len = %d, want %d", len(result.Repos), len(tt.expected.Repos))
			} else {
				for i, r := range result.Repos {
					if r.URL != tt.expected.Repos[i].URL {
						t.Errorf("Repos[%d].URL = %q, want %q", i, r.URL, tt.expected.Repos[i].URL)
					}
					if r.Branch != tt.expected.Repos[i].Branch {
						t.Errorf("Repos[%d].Branch = %q, want %q", i, r.Branch, tt.expected.Repos[i].Branch)
					}
				}
			}
			if len(tt.expected.Labels) > 0 {
				for k, v := range tt.expected.Labels {
					if result.Labels[k] != v {
						t.Errorf("Labels[%q] = %q, want %q", k, result.Labels[k], v)
					}
				}
			}
		})
	}
}

func TestNormalizePhase(t *testing.T) {
	tests := []struct {
		phase    string
		expected string
	}{
		{"Pending", "pending"},
		{"Creating", "pending"},
		{"Initializing", "pending"},
		{"Running", "running"},
		{"Active", "running"},
		{"Completed", "completed"},
		{"Succeeded", "completed"},
		{"Failed", "failed"},
		{"Error", "failed"},
		{"Unknown", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.phase, func(t *testing.T) {
			result := normalizePhase(tt.phase)
			if result != tt.expected {
				t.Errorf("normalizePhase(%q) = %q, want %q", tt.phase, result, tt.expected)
			}
		})
	}
}

func TestForwardErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		statusCode     int
		body           []byte
		expectedStatus int
		expectJSON     bool
	}{
		{
			name:           "Backend returns JSON error",
			statusCode:     500,
			body:           []byte(`{"error": "Backend error message"}`),
			expectedStatus: 500,
			expectJSON:     true,
		},
		{
			name:           "Backend returns 404 JSON",
			statusCode:     404,
			body:           []byte(`{"error": "Session not found"}`),
			expectedStatus: 404,
			expectJSON:     true,
		},
		{
			name:           "Backend returns non-JSON (plain text)",
			statusCode:     502,
			body:           []byte("Bad Gateway"),
			expectedStatus: 502,
			expectJSON:     true, // Should be wrapped in JSON
		},
		{
			name:           "Backend returns malformed JSON",
			statusCode:     500,
			body:           []byte(`{"error": "incomplete`),
			expectedStatus: 500,
			expectJSON:     true, // Should be wrapped in generic JSON
		},
		{
			name:           "Backend returns empty body",
			statusCode:     500,
			body:           []byte{},
			expectedStatus: 500,
			expectJSON:     true, // Should be wrapped in generic JSON
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)

			forwardErrorResponse(c, tt.statusCode, tt.body)

			if w.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.expectedStatus)
			}

			if tt.expectJSON {
				contentType := w.Header().Get("Content-Type")
				if contentType != "application/json; charset=utf-8" {
					t.Errorf("Content-Type = %q, want application/json", contentType)
				}
			}
		})
	}
}

func TestValidateLabelKeys(t *testing.T) {
	tests := []struct {
		name       string
		keys       []string
		wantOK     bool
		wantKey    string
		wantPrefix string
	}{
		{"valid user keys", []string{"env", "team", "custom-label"}, true, "", ""},
		{"app.kubernetes.io prefix rejected", []string{"app.kubernetes.io/name"}, false, "app.kubernetes.io/name", "app.kubernetes.io/"},
		{"vteam.ambient-code prefix rejected", []string{"vteam.ambient-code/session"}, false, "vteam.ambient-code/session", "vteam.ambient-code/"},
		{"ambient-code.io prefix rejected", []string{"ambient-code.io/runner-sa"}, false, "ambient-code.io/runner-sa", "ambient-code.io/"},
		{"mixed valid and reserved", []string{"env", "ambient-code.io/evil"}, false, "ambient-code.io/evil", "ambient-code.io/"},
		{"empty keys list", []string{}, true, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, prefix, ok := validateLabelKeys(tt.keys)
			if ok != tt.wantOK {
				t.Errorf("validateLabelKeys() ok = %v, want %v", ok, tt.wantOK)
			}
			if key != tt.wantKey {
				t.Errorf("validateLabelKeys() key = %q, want %q", key, tt.wantKey)
			}
			if prefix != tt.wantPrefix {
				t.Errorf("validateLabelKeys() prefix = %q, want %q", prefix, tt.wantPrefix)
			}
		})
	}
}

func TestFilterUserLabels_EmptyValues(t *testing.T) {
	input := map[string]interface{}{
		"tag":                   "",
		"env":                   "prod",
		"app.kubernetes.io/foo": "filtered",
	}
	result := filterUserLabels(input)

	if val, exists := result["tag"]; !exists {
		t.Error("expected empty-string label key to be present in result")
	} else if val != "" {
		t.Errorf("expected empty-string label value, got %q", val)
	}
	if result["env"] != "prod" {
		t.Errorf("expected env=prod, got %q", result["env"])
	}
	if _, exists := result["app.kubernetes.io/foo"]; exists {
		t.Error("expected internal label to be filtered out")
	}
}

func TestTransformSession_TypeSafety(t *testing.T) {
	// Test that transformSession handles incorrect types gracefully
	tests := []struct {
		name  string
		input map[string]interface{}
	}{
		{
			name: "Metadata is wrong type",
			input: map[string]interface{}{
				"metadata": "not-a-map",
			},
		},
		{
			name: "Spec is wrong type",
			input: map[string]interface{}{
				"spec": []string{"not", "a", "map"},
			},
		},
		{
			name: "Status is wrong type",
			input: map[string]interface{}{
				"status": 12345,
			},
		},
		{
			name: "Nested fields are wrong types",
			input: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":              12345, // should be string
					"creationTimestamp": true,  // should be string
				},
				"spec": map[string]interface{}{
					"prompt": []byte("bytes"), // should be string
					"model":  nil,
				},
				"status": map[string]interface{}{
					"phase":  map[string]string{}, // should be string
					"result": 99.9,                // should be string
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			result := transformSession(tt.input)
			// Should return valid (though possibly empty) response
			if result.Status == "" {
				result.Status = "pending" // default is applied
			}
			// Just verify no panic occurred
		})
	}
}
