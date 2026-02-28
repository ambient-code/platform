package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"ambient-code-backend/featureflags"
	"ambient-code-backend/types"

	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes"
)

// cachedManifest stores the last successfully loaded manifest so that
// transient file-read errors fall back to the previous good version
// instead of the hardcoded default (which bypasses feature flags).
var cachedManifest atomic.Pointer[types.ModelManifest]

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

// ListModelsForProject returns available models for a specific workspace.
// Checks workspace-scoped feature flag overrides (ConfigMap) first, then falls
// back to Unleash global state.
//
// Auth: ValidateProjectContext() middleware on the route verifies user access
// to the project namespace. The GetK8sClientsForRequest nil-check below is
// defense-in-depth per backend-development.md patterns.
func ListModelsForProject(c *gin.Context) {
	// Defense-in-depth: verify user token even though ValidateProjectContext()
	// middleware already gates this route.
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token required"})
		c.Abort()
		return
	}

	ctx := c.Request.Context()
	namespace := c.Param("projectName")

	manifest, err := LoadManifest(ManifestPath())
	if err != nil {
		log.Printf("WARNING: failed to load model manifest: %v", err)
		manifest = cachedManifest.Load()
		if manifest == nil {
			log.Printf("WARNING: no cached manifest available, using hardcoded defaults")
			c.JSON(http.StatusOK, defaultModelsResponse())
			return
		}
	} else {
		cachedManifest.Store(manifest)
	}

	// Load workspace overrides using the user-scoped client for RBAC enforcement,
	// matching the pattern in featureflags_admin.go.
	overrides, err := getWorkspaceOverrides(ctx, reqK8s, namespace)
	if err != nil {
		log.Printf("WARNING: failed to read workspace overrides for %s: %v", namespace, err)
		// Continue without overrides
	}

	var models []types.Model
	for _, entry := range manifest.Models {
		if !entry.Available {
			continue
		}

		isDefault := entry.ID == manifest.DefaultModel
		flagName := fmt.Sprintf("model.%s.enabled", entry.ID)

		// Default model is always included
		if isDefault {
			models = append(models, types.Model{
				ID: entry.ID, Label: entry.Label, Provider: entry.Provider,
				Tier: entry.Tier, IsDefault: true,
			})
			continue
		}

		// Check workspace override first, then fall back to Unleash
		if isModelEnabledWithOverrides(flagName, overrides) {
			models = append(models, types.Model{
				ID: entry.ID, Label: entry.Label, Provider: entry.Provider,
				Tier: entry.Tier, IsDefault: false,
			})
		}
	}

	if len(models) == 0 {
		log.Printf("WARNING: no models passed filtering, using defaults")
		c.JSON(http.StatusOK, defaultModelsResponse())
		return
	}

	c.JSON(http.StatusOK, types.ListModelsResponse{
		Models:       models,
		DefaultModel: manifest.DefaultModel,
	})
}

// isModelEnabledWithOverrides checks workspace ConfigMap overrides first,
// then falls back to the Unleash SDK for global state.
func isModelEnabledWithOverrides(flagName string, overrides map[string]string) bool {
	if overrides != nil {
		if val, exists := overrides[flagName]; exists {
			return val == "true"
		}
	}
	return featureflags.IsModelEnabled(flagName)
}

// LoadManifest reads the model manifest from the given path on the filesystem
// (mounted ConfigMap). No K8s API call required — the kubelet syncs the
// ConfigMap volume automatically.
func LoadManifest(path string) (*types.ModelManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest %s: %w", path, err)
	}

	var manifest types.ModelManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	return &manifest, nil
}

// isModelAvailable checks if a model is available for session creation in the
// given workspace namespace. Returns true if the model exists, is available,
// and is enabled (checking workspace overrides first, then Unleash).
// The default model always returns true. Returns true only when no manifest
// has ever been loaded (fail-open on cold start).
func isModelAvailable(ctx context.Context, k8sClient kubernetes.Interface, modelID, namespace string) bool {
	if modelID == "" {
		return true // Empty model will use default
	}

	manifest, err := LoadManifest(ManifestPath())
	if err != nil {
		log.Printf("WARNING: failed to load model manifest for validation: %v", err)
		manifest = cachedManifest.Load()
		if manifest == nil {
			// No manifest has ever been loaded — fail-open to avoid
			// blocking session creation on first startup.
			log.Printf("WARNING: no cached manifest available, allowing model %q", modelID)
			return true
		}
	} else {
		cachedManifest.Store(manifest)
	}

	// Default model is always available
	if modelID == manifest.DefaultModel {
		return true
	}

	for _, entry := range manifest.Models {
		if entry.ID == modelID {
			if !entry.Available {
				return false
			}
			flagName := fmt.Sprintf("model.%s.enabled", entry.ID)
			// Check workspace overrides first, then fall back to Unleash
			overrides, err := getWorkspaceOverrides(ctx, k8sClient, namespace)
			if err != nil {
				log.Printf("WARNING: failed to read workspace overrides for %s: %v", namespace, err)
			}
			return isModelEnabledWithOverrides(flagName, overrides)
		}
	}

	// Model not found in manifest — reject it. The manifest is authoritative;
	// unknown model IDs should not be accepted.
	log.Printf("WARNING: model %q not found in manifest, rejecting", modelID)
	return false
}

// defaultModelsResponse returns a hardcoded ListModelsResponse as a fallback
// when the model manifest file is unavailable or malformed.
// Keep in sync with components/manifests/base/models.json (available: true entries).
func defaultModelsResponse() types.ListModelsResponse {
	return types.ListModelsResponse{
		DefaultModel: "claude-sonnet-4-5",
		Models: []types.Model{
			{ID: "claude-sonnet-4-5", Label: "Claude Sonnet 4.5", Provider: "anthropic", Tier: "standard", IsDefault: true},
			{ID: "claude-sonnet-4-6", Label: "Claude Sonnet 4.6", Provider: "anthropic", Tier: "standard", IsDefault: false},
			{ID: "claude-opus-4-6", Label: "Claude Opus 4.6", Provider: "anthropic", Tier: "premium", IsDefault: false},
			{ID: "claude-opus-4-5", Label: "Claude Opus 4.5", Provider: "anthropic", Tier: "premium", IsDefault: false},
			{ID: "claude-haiku-4-5", Label: "Claude Haiku 4.5", Provider: "anthropic", Tier: "standard", IsDefault: false},
		},
	}
}
