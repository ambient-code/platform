package models

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifest(t *testing.T) {
	validJSON := `{
		"version": 1,
		"defaultModel": "claude-sonnet-4-5",
		"models": [
			{"id": "claude-sonnet-4-5", "vertexId": "claude-sonnet-4-5@20250929", "available": true},
			{"id": "claude-opus-4-6", "vertexId": "claude-opus-4-6@default", "available": true}
		]
	}`

	writeManifest := func(t *testing.T, content string) string {
		t.Helper()
		dir := t.TempDir()
		path := filepath.Join(dir, "models.json")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test manifest: %v", err)
		}
		return path
	}

	t.Run("valid JSON parses correctly", func(t *testing.T) {
		path := writeManifest(t, validJSON)

		manifest, err := LoadManifest(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if manifest.Version != 1 {
			t.Errorf("expected version 1, got %d", manifest.Version)
		}
		if manifest.DefaultModel != "claude-sonnet-4-5" {
			t.Errorf("expected defaultModel claude-sonnet-4-5, got %s", manifest.DefaultModel)
		}
		if len(manifest.Models) != 2 {
			t.Errorf("expected 2 models, got %d", len(manifest.Models))
		}
	})

	t.Run("malformed JSON returns error", func(t *testing.T) {
		path := writeManifest(t, "{invalid json")

		_, err := LoadManifest(path)
		if err == nil {
			t.Fatal("expected error for malformed JSON")
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nonexistent.json")

		_, err := LoadManifest(path)
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})
}

func TestResolveVertexID(t *testing.T) {
	manifest := &ModelManifest{
		Version:      1,
		DefaultModel: "claude-sonnet-4-5",
		Models: []ModelEntry{
			{ID: "claude-sonnet-4-5", VertexID: "claude-sonnet-4-5@20250929", Available: true},
			{ID: "claude-opus-4-6", VertexID: "claude-opus-4-6@default", Available: true},
		},
	}

	t.Run("returns correct vertexId for known model", func(t *testing.T) {
		result := ResolveVertexID(manifest, "claude-sonnet-4-5")
		if result != "claude-sonnet-4-5@20250929" {
			t.Errorf("expected claude-sonnet-4-5@20250929, got %s", result)
		}
	})

	t.Run("returns correct vertexId for second model", func(t *testing.T) {
		result := ResolveVertexID(manifest, "claude-opus-4-6")
		if result != "claude-opus-4-6@default" {
			t.Errorf("expected claude-opus-4-6@default, got %s", result)
		}
	})

	t.Run("returns empty string for unknown model", func(t *testing.T) {
		result := ResolveVertexID(manifest, "nonexistent-model")
		if result != "" {
			t.Errorf("expected empty string, got %s", result)
		}
	})

	t.Run("returns empty string for unavailable model", func(t *testing.T) {
		m := &ModelManifest{
			Version:      1,
			DefaultModel: "claude-sonnet-4-5",
			Models: []ModelEntry{
				{ID: "claude-sonnet-4-5", VertexID: "claude-sonnet-4-5@20250929", Available: true},
				{ID: "claude-opus-4-6", VertexID: "claude-opus-4-6@default", Available: false},
			},
		}
		result := ResolveVertexID(m, "claude-opus-4-6")
		if result != "" {
			t.Errorf("expected empty string for unavailable model, got %s", result)
		}
		// available model should still resolve
		result = ResolveVertexID(m, "claude-sonnet-4-5")
		if result != "claude-sonnet-4-5@20250929" {
			t.Errorf("expected claude-sonnet-4-5@20250929, got %s", result)
		}
	})
}
