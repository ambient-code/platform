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
	DefaultManifestPath = "/config/models/models.json"
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
	namespace := sanitizeParam(c.Param("projectName"))
	providerFilter := sanitizeParam(c.Query("provider"))

	manifest, err := LoadManifest(ManifestPath())
	if err != nil {
		log.Printf("WARNING: failed to load model manifest from disk: %v", err)
		manifest = cachedManifest.Load()
		if manifest == nil {
			log.Printf("ERROR: no model manifest available (file unreadable, no cache)")
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Model manifest unavailable"})
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

	// Resolve which model ID is the "default" for this request.
	// When filtering by provider, use the provider-specific default.
	effectiveDefault := manifest.DefaultModel
	if providerFilter != "" {
		if pd, ok := manifest.ProviderDefaults[providerFilter]; ok {
			effectiveDefault = pd
		}
	}

	models := make([]types.Model, 0)
	for _, entry := range manifest.Models {
		if !entry.Available {
			continue
		}

		// Filter by provider if specified
		if providerFilter != "" && entry.Provider != providerFilter {
			continue
		}

		isDefault := entry.ID == effectiveDefault
		flagName := fmt.Sprintf("model.%s.enabled", entry.ID)

		// Default model is always included
		if isDefault {
			models = append(models, types.Model{
				ID: entry.ID, Label: entry.Label, Provider: entry.Provider,
				IsDefault: true,
			})
			continue
		}

		// Check workspace override first, then fall back to Unleash
		if isModelEnabledWithOverrides(flagName, overrides) {
			models = append(models, types.Model{
				ID: entry.ID, Label: entry.Label, Provider: entry.Provider,
				IsDefault: false,
			})
		}
	}

	responseDefault := effectiveDefault
	if len(models) == 0 {
		log.Printf("WARNING: no models passed filtering for provider=%q in namespace %s", providerFilter, namespace)
		responseDefault = ""
	}

	c.JSON(http.StatusOK, types.ListModelsResponse{
		Models:       models,
		DefaultModel: responseDefault,
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
// given workspace namespace. All models (Claude and Gemini) are validated
// against models.json. Returns true if the model exists, is available, and
// is enabled (checking workspace overrides first, then Unleash).
// When requiredProvider is non-empty, the model's provider must match
// (prevents using a Gemini model with a Claude runner, for example).
// The default model always returns true. Fails open when no manifest has
// ever been loaded (cold start).
func isModelAvailable(ctx context.Context, k8sClient kubernetes.Interface, modelID, requiredProvider, namespace string) bool {
	if modelID == "" {
		return true // Empty model will use default
	}

	manifest, err := LoadManifest(ManifestPath())
	if err != nil {
		log.Printf("WARNING: failed to load model manifest for validation: %v", err)
		manifest = cachedManifest.Load()
		if manifest == nil {
			// When we know the runner's provider, reject unknown models rather
			// than allowing a cross-provider mismatch through to the runner.
			// Fail-open only when both manifest and registry are unavailable
			// (requiredProvider == "") to avoid blocking cold starts.
			if requiredProvider != "" {
				log.Printf("WARNING: no manifest available, rejecting model %q (provider=%q)", modelID, requiredProvider)
				return false
			}
			log.Printf("WARNING: no manifest or registry available, allowing model %q", modelID)
			return true
		}
	} else {
		cachedManifest.Store(manifest)
	}

	for _, entry := range manifest.Models {
		if entry.ID == modelID {
			if !entry.Available {
				return false
			}
			// Provider mismatch check applies to ALL models, including defaults
			if requiredProvider != "" && entry.Provider != requiredProvider {
				log.Printf("Model %q has provider %q but runner requires %q", modelID, entry.Provider, requiredProvider)
				return false
			}
			// Default models (global and per-provider) are always enabled
			// (skip feature flag check) but must still pass provider matching above
			if modelID == manifest.DefaultModel {
				return true
			}
			for provider, pd := range manifest.ProviderDefaults {
				if modelID == pd && (requiredProvider == "" || provider == requiredProvider) {
					return true
				}
			}
			flagName := fmt.Sprintf("model.%s.enabled", entry.ID)
			overrides, oErr := getWorkspaceOverrides(ctx, k8sClient, namespace)
			if oErr != nil {
				log.Printf("WARNING: failed to read workspace overrides for %s: %v", namespace, oErr)
			}
			return isModelEnabledWithOverrides(flagName, overrides)
		}
	}

	log.Printf("WARNING: model %q not found in manifest, rejecting", modelID)
	return false
}
