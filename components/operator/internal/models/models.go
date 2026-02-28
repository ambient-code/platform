// Package models provides model manifest loading and Vertex AI model ID resolution.
// Types are defined locally to avoid importing the backend types package.
package models

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	// DefaultManifestPath is where the ambient-models ConfigMap is mounted.
	DefaultManifestPath = "/config/models.json"
)

// ManifestPath returns the filesystem path to the models manifest.
// Defaults to DefaultManifestPath; override via MODELS_MANIFEST_PATH env var.
func ManifestPath() string {
	if p := os.Getenv("MODELS_MANIFEST_PATH"); p != "" {
		return p
	}
	return DefaultManifestPath
}

// ModelEntry represents a single model in the manifest.
type ModelEntry struct {
	ID        string `json:"id"`
	VertexID  string `json:"vertexId"`
	Available bool   `json:"available"`
}

// ModelManifest represents the model manifest structure.
type ModelManifest struct {
	Version      int          `json:"version"`
	DefaultModel string       `json:"defaultModel"`
	Models       []ModelEntry `json:"models"`
}

// LoadManifest reads the model manifest from the given path on the filesystem
// (mounted ConfigMap). No K8s API call required — the kubelet syncs the
// ConfigMap volume automatically.
func LoadManifest(path string) (*ModelManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest %s: %w", path, err)
	}

	var manifest ModelManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	return &manifest, nil
}

// ResolveVertexID finds a model by ID and returns its Vertex AI model ID.
// Returns an empty string if the model is not found or not available.
func ResolveVertexID(manifest *ModelManifest, modelID string) string {
	for _, m := range manifest.Models {
		if m.ID == modelID && m.Available {
			return m.VertexID
		}
	}
	return ""
}
