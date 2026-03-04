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

// --- FlagsFromManifest ---

func TestFlagsFromManifest_SkipsDefaultAndUnavailable(t *testing.T) {
	manifest := &types.ModelManifest{
		DefaultModel: "claude-sonnet-4-5",
		ProviderDefaults: map[string]string{
			"anthropic": "claude-sonnet-4-5",
			"google":    "gemini-2.5-flash",
		},
		Models: []types.ModelEntry{
			{ID: "claude-sonnet-4-5", Label: "Sonnet 4.5", Provider: "anthropic", Available: true},
			{ID: "claude-opus-4-6", Label: "Opus 4.6", Provider: "anthropic", Available: true},
			{ID: "claude-opus-4-1", Label: "Opus 4.1", Provider: "anthropic", Available: false},
			{ID: "gemini-2.5-flash", Label: "Gemini 2.5 Flash", Provider: "google", Available: true},
			{ID: "gemini-2.5-pro", Label: "Gemini 2.5 Pro", Provider: "google", Available: true},
		},
	}

	flags := FlagsFromManifest(manifest)

	// Should skip: claude-sonnet-4-5 (global default + anthropic default),
	//              gemini-2.5-flash (google default),
	//              claude-opus-4-1 (unavailable)
	// Should include: claude-opus-4-6, gemini-2.5-pro
	if len(flags) != 2 {
		t.Fatalf("expected 2 flags, got %d: %v", len(flags), flags)
	}

	names := map[string]bool{}
	for _, f := range flags {
		names[f.Name] = true
	}
	if !names["model.claude-opus-4-6.enabled"] {
		t.Error("expected model.claude-opus-4-6.enabled")
	}
	if !names["model.gemini-2.5-pro.enabled"] {
		t.Error("expected model.gemini-2.5-pro.enabled")
	}
	if names["model.claude-sonnet-4-5.enabled"] {
		t.Error("global default should be skipped")
	}
	if names["model.gemini-2.5-flash.enabled"] {
		t.Error("provider default should be skipped")
	}
}

func TestFlagsFromManifest_EmptyManifest(t *testing.T) {
	manifest := &types.ModelManifest{DefaultModel: "x", Models: nil}
	flags := FlagsFromManifest(manifest)
	if len(flags) != 0 {
		t.Errorf("expected 0 flags, got %d", len(flags))
	}
}

// --- FlagsFromConfig ---

func TestFlagsFromConfig_LoadsValidFile(t *testing.T) {
	config := FlagsConfig{
		Flags: []FlagSpec{
			{Name: "framework.langgraph.enabled", Description: "Enable LangGraph"},
			{Name: "feature.dark-mode", Description: "Dark mode", Tags: []FlagTag{{Type: "scope", Value: "workspace"}}},
		},
	}
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "flags.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	flags, err := FlagsFromConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flags) != 2 {
		t.Fatalf("expected 2 flags, got %d", len(flags))
	}
	if flags[0].Name != "framework.langgraph.enabled" {
		t.Errorf("expected framework.langgraph.enabled, got %s", flags[0].Name)
	}
}

func TestFlagsFromConfig_MissingFileReturnsNil(t *testing.T) {
	flags, err := FlagsFromConfig("/nonexistent/flags.json")
	if err != nil {
		t.Fatalf("missing file should not error, got: %v", err)
	}
	if flags != nil {
		t.Errorf("expected nil, got %v", flags)
	}
}

func TestFlagsFromConfig_SanitizesNewlines(t *testing.T) {
	raw := `{
		"flags": [{
			"name": "flag.with\nnewline",
			"description": "desc\rwith\r\nCRLF",
			"tags": [{"type": "scope\n", "value": "work\rspace"}]
		}]
	}`

	path := filepath.Join(t.TempDir(), "flags.json")
	if err := os.WriteFile(path, []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}

	flags, err := FlagsFromConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flags) != 1 {
		t.Fatalf("expected 1 flag, got %d", len(flags))
	}

	f := flags[0]
	if strings.ContainsAny(f.Name, "\n\r") {
		t.Errorf("name should not contain newlines, got %q", f.Name)
	}
	if strings.ContainsAny(f.Description, "\n\r") {
		t.Errorf("description should not contain newlines, got %q", f.Description)
	}
	if strings.ContainsAny(f.Tags[0].Type, "\n\r") {
		t.Errorf("tag type should not contain newlines, got %q", f.Tags[0].Type)
	}
	if strings.ContainsAny(f.Tags[0].Value, "\n\r") {
		t.Errorf("tag value should not contain newlines, got %q", f.Tags[0].Value)
	}
}

func TestFlagsFromConfig_InvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "flags.json")
	if err := os.WriteFile(path, []byte("{bad"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := FlagsFromConfig(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// --- SyncFlags ---

func TestSyncFlags_SkipsWhenEnvNotSet(t *testing.T) {
	t.Setenv("UNLEASH_ADMIN_URL", "")
	t.Setenv("UNLEASH_ADMIN_TOKEN", "")

	flags := FlagsFromManifest(&types.ModelManifest{
		DefaultModel: "claude-sonnet-4-5",
		Models: []types.ModelEntry{
			{ID: "claude-sonnet-4-5", Label: "Sonnet 4.5", Available: true},
			{ID: "claude-opus-4-6", Label: "Opus 4.6", Available: true},
		},
	})

	err := SyncFlags(context.Background(), flags)
	if err != nil {
		t.Errorf("expected nil error when env not set, got: %v", err)
	}
}

func TestSyncFlags_CreatesNewFlag(t *testing.T) {
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
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/features/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/features") {
			createCalled = true
			json.NewDecoder(r.Body).Decode(&createBody)
			w.WriteHeader(http.StatusCreated)
			return
		}
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/tags") {
			tagCalled = true
			w.WriteHeader(http.StatusCreated)
			return
		}
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

	flags := []FlagSpec{
		{
			Name:        "model.claude-opus-4-6.enabled",
			Description: "Enable Opus 4.6",
			Tags:        []FlagTag{{Type: "scope", Value: "workspace"}},
		},
	}

	err := SyncFlags(context.Background(), flags)
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

func TestSyncFlags_NoTagsSkipsTagCall(t *testing.T) {
	tagCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/features/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/features") {
			w.WriteHeader(http.StatusCreated)
			return
		}
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/tags") {
			tagCalled = true
			w.WriteHeader(http.StatusCreated)
			return
		}
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/strategies") {
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("UNLEASH_ADMIN_URL", server.URL)
	t.Setenv("UNLEASH_ADMIN_TOKEN", "test-token")

	flags := []FlagSpec{
		{Name: "framework.xyz.enabled", Description: "XYZ framework"},
	}

	err := SyncFlags(context.Background(), flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tagCalled {
		t.Error("tag API should not be called for flags with no tags")
	}
}

func TestSyncFlags_HandlesConflict(t *testing.T) {
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
			w.WriteHeader(http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("UNLEASH_ADMIN_URL", server.URL)
	t.Setenv("UNLEASH_ADMIN_TOKEN", "test-token")

	flags := []FlagSpec{
		{Name: "test.flag", Description: "test", Tags: []FlagTag{{Type: "scope", Value: "workspace"}}},
	}

	err := SyncFlags(context.Background(), flags)
	if err != nil {
		t.Errorf("conflict should not cause error, got: %v", err)
	}
}

func TestSyncFlags_ReturnsErrorOnCreateFailure(t *testing.T) {
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

	flags := []FlagSpec{
		{Name: "test.flag", Description: "test", Tags: []FlagTag{{Type: "scope", Value: "workspace"}}},
	}

	err := SyncFlags(context.Background(), flags)
	if err == nil {
		t.Error("expected error on create failure")
	}
	if !strings.Contains(err.Error(), "1 errors occurred") {
		t.Errorf("expected error count message, got: %v", err)
	}
}

func TestSyncModelFlagsFromFile(t *testing.T) {
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

// --- collectTagTypes ---

func TestCollectTagTypes(t *testing.T) {
	flags := []FlagSpec{
		{Name: "a", Tags: []FlagTag{{Type: "scope", Value: "workspace"}}},
		{Name: "b", Tags: []FlagTag{{Type: "scope", Value: "global"}, {Type: "env", Value: "prod"}}},
		{Name: "c"},
	}

	tagTypes := collectTagTypes(flags)
	if len(tagTypes) != 2 {
		t.Fatalf("expected 2 unique tag types, got %d: %v", len(tagTypes), tagTypes)
	}

	found := map[string]bool{}
	for _, tt := range tagTypes {
		found[tt] = true
	}
	if !found["scope"] {
		t.Error("expected 'scope' in tag types")
	}
	if !found["env"] {
		t.Error("expected 'env' in tag types")
	}
}
