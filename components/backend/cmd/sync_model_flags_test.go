package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ambient-code-backend/types"
)

func TestParseManifestPath(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "no args returns default",
			args: []string{},
			want: defaultManifestPath,
		},
		{
			name: "unrelated args returns default",
			args: []string{"--verbose", "--port", "8080"},
			want: defaultManifestPath,
		},
		{
			name: "flag with separate value",
			args: []string{"--manifest-path", "/custom/path.json"},
			want: "/custom/path.json",
		},
		{
			name: "flag with equals sign",
			args: []string{"--manifest-path=/custom/path.json"},
			want: "/custom/path.json",
		},
		{
			name: "flag among other args",
			args: []string{"--verbose", "--manifest-path", "/data/models.json", "--port", "8080"},
			want: "/data/models.json",
		},
		{
			name: "flag at end without value returns default",
			args: []string{"--manifest-path"},
			want: defaultManifestPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseManifestPath(tt.args)
			if got != tt.want {
				t.Errorf("ParseManifestPath(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestSyncModelFlags_SkipsWhenEnvNotSet(t *testing.T) {
	// Ensure env vars are not set
	t.Setenv("UNLEASH_ADMIN_URL", "")
	t.Setenv("UNLEASH_ADMIN_TOKEN", "")

	manifest := &types.ModelManifest{
		DefaultModel: "claude-sonnet-4-5",
		Models: []types.ModelEntry{
			{ID: "claude-sonnet-4-5", Label: "Sonnet 4.5", Available: true},
			{ID: "claude-opus-4-6", Label: "Opus 4.6", Available: true},
		},
	}

	err := SyncModelFlags(context.Background(), manifest)
	if err != nil {
		t.Errorf("expected nil error when env not set, got: %v", err)
	}
}

func TestSyncModelFlags_ExcludesDefaultModel(t *testing.T) {
	var flagChecks []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Tag type check
		if strings.Contains(r.URL.Path, "/tag-types/") {
			w.WriteHeader(http.StatusOK)
			return
		}
		// Feature existence check
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/features/") {
			parts := strings.Split(r.URL.Path, "/features/")
			if len(parts) > 1 {
				flagChecks = append(flagChecks, parts[1])
			}
			w.WriteHeader(http.StatusOK) // flag already exists
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("UNLEASH_ADMIN_URL", server.URL)
	t.Setenv("UNLEASH_ADMIN_TOKEN", "test-token")
	t.Setenv("UNLEASH_PROJECT", "default")

	manifest := &types.ModelManifest{
		DefaultModel: "claude-sonnet-4-5",
		Models: []types.ModelEntry{
			{ID: "claude-sonnet-4-5", Label: "Sonnet 4.5", Available: true},
			{ID: "claude-opus-4-6", Label: "Opus 4.6", Available: true},
		},
	}

	err := SyncModelFlags(context.Background(), manifest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only opus should have been checked — sonnet is the default
	if len(flagChecks) != 1 {
		t.Fatalf("expected 1 flag check, got %d: %v", len(flagChecks), flagChecks)
	}
	if flagChecks[0] != "model.claude-opus-4-6.enabled" {
		t.Errorf("expected flag check for model.claude-opus-4-6.enabled, got %s", flagChecks[0])
	}
}

func TestSyncModelFlags_ExcludesUnavailableModels(t *testing.T) {
	var flagChecks []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/tag-types/") {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/features/") {
			parts := strings.Split(r.URL.Path, "/features/")
			if len(parts) > 1 {
				flagChecks = append(flagChecks, parts[1])
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("UNLEASH_ADMIN_URL", server.URL)
	t.Setenv("UNLEASH_ADMIN_TOKEN", "test-token")
	t.Setenv("UNLEASH_PROJECT", "default")

	manifest := &types.ModelManifest{
		DefaultModel: "claude-sonnet-4-5",
		Models: []types.ModelEntry{
			{ID: "claude-sonnet-4-5", Label: "Sonnet 4.5", Available: true},
			{ID: "claude-opus-4-6", Label: "Opus 4.6", Available: true},
			{ID: "claude-opus-4-1", Label: "Opus 4.1", Available: false},
		},
	}

	err := SyncModelFlags(context.Background(), manifest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only opus-4-6 should be checked (sonnet is default, opus-4-1 is unavailable)
	if len(flagChecks) != 1 {
		t.Fatalf("expected 1 flag check, got %d: %v", len(flagChecks), flagChecks)
	}
	if flagChecks[0] != "model.claude-opus-4-6.enabled" {
		t.Errorf("expected flag check for model.claude-opus-4-6.enabled, got %s", flagChecks[0])
	}
}

func TestSyncModelFlags_CreatesNewFlag(t *testing.T) {
	var (
		createCalled   bool
		tagCalled      bool
		strategyCalled bool
		createBody     map[string]any
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/tag-types/") {
			w.WriteHeader(http.StatusOK)
			return
		}
		// Feature existence check — return 404 so it gets created
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/features/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Feature creation
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/features") {
			createCalled = true
			json.NewDecoder(r.Body).Decode(&createBody)
			w.WriteHeader(http.StatusCreated)
			return
		}
		// Tag addition
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/tags") {
			tagCalled = true
			w.WriteHeader(http.StatusCreated)
			return
		}
		// Strategy addition
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/strategies") {
			strategyCalled = true
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("UNLEASH_ADMIN_URL", server.URL)
	t.Setenv("UNLEASH_ADMIN_TOKEN", "test-token")
	t.Setenv("UNLEASH_PROJECT", "default")
	t.Setenv("UNLEASH_ENVIRONMENT", "development")

	manifest := &types.ModelManifest{
		DefaultModel: "claude-sonnet-4-5",
		Models: []types.ModelEntry{
			{ID: "claude-sonnet-4-5", Label: "Sonnet 4.5", Available: true},
			{ID: "claude-opus-4-6", Label: "Opus 4.6", Available: true},
		},
	}

	err := SyncModelFlags(context.Background(), manifest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !createCalled {
		t.Error("expected flag creation API call")
	}
	if !tagCalled {
		t.Error("expected tag API call")
	}
	if !strategyCalled {
		t.Error("expected strategy API call")
	}

	// Verify the flag was created with correct properties
	if createBody["name"] != "model.claude-opus-4-6.enabled" {
		t.Errorf("expected flag name model.claude-opus-4-6.enabled, got %v", createBody["name"])
	}
	if createBody["type"] != "release" {
		t.Errorf("expected type release, got %v", createBody["type"])
	}
	if createBody["enabled"] != false {
		t.Errorf("expected enabled=false, got %v", createBody["enabled"])
	}
}

func TestSyncModelFlags_HandlesConflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/tag-types/") {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/features/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/features") {
			w.WriteHeader(http.StatusConflict) // another instance created it
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("UNLEASH_ADMIN_URL", server.URL)
	t.Setenv("UNLEASH_ADMIN_TOKEN", "test-token")

	manifest := &types.ModelManifest{
		DefaultModel: "claude-sonnet-4-5",
		Models: []types.ModelEntry{
			{ID: "claude-sonnet-4-5", Label: "Sonnet 4.5", Available: true},
			{ID: "claude-opus-4-6", Label: "Opus 4.6", Available: true},
		},
	}

	err := SyncModelFlags(context.Background(), manifest)
	if err != nil {
		t.Errorf("conflict should not cause error, got: %v", err)
	}
}

func TestSyncModelFlags_ReturnsErrorOnCreateFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/tag-types/") {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/features/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/features") {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"internal error"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("UNLEASH_ADMIN_URL", server.URL)
	t.Setenv("UNLEASH_ADMIN_TOKEN", "test-token")

	manifest := &types.ModelManifest{
		DefaultModel: "claude-sonnet-4-5",
		Models: []types.ModelEntry{
			{ID: "claude-sonnet-4-5", Label: "Sonnet 4.5", Available: true},
			{ID: "claude-opus-4-6", Label: "Opus 4.6", Available: true},
		},
	}

	err := SyncModelFlags(context.Background(), manifest)
	if err == nil {
		t.Error("expected error on create failure")
	}
	if !strings.Contains(err.Error(), "1 errors occurred") {
		t.Errorf("expected error count message, got: %v", err)
	}
}

func TestSyncModelFlagsFromFile(t *testing.T) {
	// Ensure Unleash env vars are not set so sync is a no-op
	t.Setenv("UNLEASH_ADMIN_URL", "")
	t.Setenv("UNLEASH_ADMIN_TOKEN", "")

	manifest := types.ModelManifest{
		Version:      1,
		DefaultModel: "claude-sonnet-4-5",
		Models: []types.ModelEntry{
			{ID: "claude-sonnet-4-5", Label: "Sonnet 4.5", Available: true},
		},
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "models.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	err = SyncModelFlagsFromFile(path)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSyncModelFlagsFromFile_FileNotFound(t *testing.T) {
	err := SyncModelFlagsFromFile("/nonexistent/path/models.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestSyncModelFlagsFromFile_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "models.json")
	if err := os.WriteFile(path, []byte("{invalid"), 0644); err != nil {
		t.Fatal(err)
	}

	err := SyncModelFlagsFromFile(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parsing manifest") {
		t.Errorf("expected parsing error, got: %v", err)
	}
}
