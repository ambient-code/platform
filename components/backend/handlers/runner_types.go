package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// DefaultRunnerType is the default runner type when none is specified.
const DefaultRunnerType = "claude-agent-sdk"

// DefaultRunnerPort is used when a runtime's container port is not set.
const DefaultRunnerPort = 8001

// AgentRuntimeSpec is parsed from the agent registry ConfigMap JSON.
// NOTE: These types are duplicated in components/operator/internal/handlers/registry.go.
// Keep both in sync when modifying the schema.
// It is the single source of truth for runtime configuration.
type AgentRuntimeSpec struct {
	ID          string        `json:"id"`
	DisplayName string        `json:"displayName"`
	Description string        `json:"description"`
	Framework   string        `json:"framework"`
	Container   ContainerSpec `json:"container"`
	Sandbox     SandboxSpec   `json:"sandbox"`
	Auth        AuthSpec      `json:"auth"`
	Provider    string        `json:"provider"`
	FeatureGate string        `json:"featureGate"`
}

// ContainerSpec defines the runner container configuration.
type ContainerSpec struct {
	Image     string            `json:"image"`
	Port      int               `json:"port"`
	Env       map[string]string `json:"env"`
	Resources *ResourcesSpec    `json:"resources,omitempty"`
}

// ResourcesSpec defines container resource requests and limits.
type ResourcesSpec struct {
	Requests map[string]string `json:"requests,omitempty"`
	Limits   map[string]string `json:"limits,omitempty"`
}

// SandboxSpec defines sandbox/pod configuration for the runner.
type SandboxSpec struct {
	StateDir               string   `json:"stateDir,omitempty"`
	StateSyncImage         string   `json:"stateSyncImage,omitempty"`
	Persistence            string   `json:"persistence"`
	WorkspaceSize          string   `json:"workspaceSize,omitempty"`
	TerminationGracePeriod int      `json:"terminationGracePeriod,omitempty"`
	Seed                   SeedSpec `json:"seed"`
}

// SeedSpec controls init container behavior.
type SeedSpec struct {
	CloneRepos   bool `json:"cloneRepos"`
	HydrateState bool `json:"hydrateState"`
}

// AuthSpec defines authentication requirements for a runner.
type AuthSpec struct {
	RequiredSecretKeys []string `json:"requiredSecretKeys"`
	SecretKeyLogic     string   `json:"secretKeyLogic"`
	VertexSupported    bool     `json:"vertexSupported"`
}

// RunnerTypeResponse is the public API shape returned to the frontend.
// FeatureGate is intentionally excluded — gated runners are already filtered
// out by the handler, so the frontend never needs to see the gate name.
type RunnerTypeResponse struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"displayName"`
	Description string   `json:"description"`
	Framework   string   `json:"framework"`
	Provider    string   `json:"provider"`
	Auth        AuthSpec `json:"auth"`
}

// In-memory cache for the agent registry (ConfigMap content changes rarely).
var (
	registryCache     []AgentRuntimeSpec
	registryCacheMu   sync.RWMutex
	registryCacheTime time.Time
)

const registryCacheTTL = 60 * time.Second

// defaultRegistryPath is where the agent-registry ConfigMap is mounted.
const defaultRegistryPath = "/config/registry/agent-registry.json"

// registryPath returns the filesystem path to the agent registry JSON.
func registryPath() string {
	if p := os.Getenv("AGENT_REGISTRY_PATH"); p != "" {
		return p
	}
	return defaultRegistryPath
}

// loadAgentRegistry reads and parses the agent registry from the mounted ConfigMap file.
// Results are cached in-memory with a TTL since the file content rarely changes.
func loadAgentRegistry() ([]AgentRuntimeSpec, error) {
	registryCacheMu.RLock()
	if time.Since(registryCacheTime) < registryCacheTTL && registryCache != nil {
		defer registryCacheMu.RUnlock()
		return registryCache, nil
	}
	registryCacheMu.RUnlock()

	data, err := os.ReadFile(registryPath())
	if err != nil {
		// On read failure, return stale cache if available
		registryCacheMu.RLock()
		if registryCache != nil {
			defer registryCacheMu.RUnlock()
			log.Printf("Warning: failed to refresh agent registry, using stale cache: %v", err)
			return registryCache, nil
		}
		registryCacheMu.RUnlock()
		return nil, fmt.Errorf("failed to read agent registry from %s: %w", registryPath(), err)
	}

	var entries []AgentRuntimeSpec
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse agent registry JSON: %w", err)
	}

	registryCacheMu.Lock()
	registryCache = entries
	registryCacheTime = time.Now()
	registryCacheMu.Unlock()

	return entries, nil
}

// GetRuntime returns the AgentRuntimeSpec for a given runner type ID.
// Returns an error if the registry cannot be loaded or the runtime is not found.
func GetRuntime(runnerTypeID string) (*AgentRuntimeSpec, error) {
	entries, err := loadAgentRegistry()
	if err != nil {
		return nil, err
	}
	for i := range entries {
		if entries[i].ID == runnerTypeID {
			copy := entries[i]
			return &copy, nil
		}
	}
	return nil, fmt.Errorf("unknown runner type %q", runnerTypeID)
}

// GetRuntimePort returns the container port for a given runner type.
// Falls back to DefaultRunnerPort if the runtime is not found or port is 0.
func GetRuntimePort(runnerTypeID string) int {
	rt, err := GetRuntime(runnerTypeID)
	if err != nil || rt.Container.Port == 0 {
		return DefaultRunnerPort
	}
	return rt.Container.Port
}

// getRequiredSecretKeys returns the requiredSecretKeys for a given runner type
// from the agent registry. Returns nil if the runner is not found or has no keys.
func getRequiredSecretKeys(runnerTypeID string) []string {
	rt, err := GetRuntime(runnerTypeID)
	if err != nil {
		return nil
	}
	return rt.Auth.RequiredSecretKeys
}

// getContainerEnvVars returns the env vars from the registry for a given runner type.
// These are injected into the CRD's environmentVariables during session creation.
func getContainerEnvVars(runnerTypeID string) map[string]string {
	rt, err := GetRuntime(runnerTypeID)
	if err != nil {
		// Fallback: at minimum set RUNNER_TYPE
		return map[string]string{"RUNNER_TYPE": runnerTypeID}
	}
	if len(rt.Container.Env) > 0 {
		// Return a copy to prevent mutation
		envCopy := make(map[string]string, len(rt.Container.Env))
		for k, v := range rt.Container.Env {
			envCopy[k] = v
		}
		return envCopy
	}
	return map[string]string{"RUNNER_TYPE": runnerTypeID}
}

// isRunnerEnabled checks if a runner type is enabled via feature flags.
// Runtimes with an empty featureGate are always enabled.
func isRunnerEnabled(runnerID string) bool {
	rt, err := GetRuntime(runnerID)
	if err != nil {
		// Registry unavailable — fail-open for the default runner to prevent
		// blocking all session creation during cold start or transient outages.
		if runnerID == DefaultRunnerType {
			return true
		}
		flag := "runner." + runnerID + ".enabled"
		return FeatureEnabled(flag)
	}
	if rt.FeatureGate == "" {
		return true
	}
	return FeatureEnabled(rt.FeatureGate)
}

// isRunnerEnabledWithOverrides checks workspace ConfigMap overrides first,
// then falls back to the Unleash SDK for global state.
func isRunnerEnabledWithOverrides(flagName string, overrides map[string]string) bool {
	if overrides != nil {
		if val, exists := overrides[flagName]; exists {
			return val == "true"
		}
	}
	return FeatureEnabled(flagName)
}

// GetRunnerTypesGlobal handles GET /api/runner-types (no auth, no workspace overrides).
// Used by admin pages that need to list all runner types regardless of workspace.
func GetRunnerTypesGlobal(c *gin.Context) {
	entries, err := loadAgentRegistry()
	if err != nil {
		log.Printf("Failed to load agent registry: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load runner types"})
		return
	}

	resp := make([]RunnerTypeResponse, 0, len(entries))
	for _, e := range entries {
		if e.FeatureGate != "" && !FeatureEnabled(e.FeatureGate) {
			continue
		}
		resp = append(resp, RunnerTypeResponse{
			ID:          e.ID,
			DisplayName: e.DisplayName,
			Description: e.Description,
			Framework:   e.Framework,
			Provider:    e.Provider,
			Auth:        e.Auth,
		})
	}

	c.JSON(http.StatusOK, resp)
}

// GetRunnerTypes handles GET /api/projects/:projectName/runner-types and returns
// the list of available runner types. Runners gated by feature flags are filtered
// out, respecting workspace-scoped overrides in the feature-flag-overrides ConfigMap.
func GetRunnerTypes(c *gin.Context) {
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token required"})
		c.Abort()
		return
	}

	ctx := c.Request.Context()
	namespace := sanitizeParam(c.Param("projectName"))

	entries, err := loadAgentRegistry()
	if err != nil {
		log.Printf("Failed to load agent registry: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load runner types"})
		return
	}

	// Load workspace overrides for feature gate evaluation
	overrides, err := getWorkspaceOverrides(ctx, reqK8s, namespace)
	if err != nil {
		log.Printf("WARNING: failed to read workspace overrides for runner types in %s: %v", namespace, err)
	}

	resp := make([]RunnerTypeResponse, 0, len(entries))
	for _, e := range entries {
		if e.FeatureGate != "" && !isRunnerEnabledWithOverrides(e.FeatureGate, overrides) {
			continue
		}
		resp = append(resp, RunnerTypeResponse{
			ID:          e.ID,
			DisplayName: e.DisplayName,
			Description: e.Description,
			Framework:   e.Framework,
			Provider:    e.Provider,
			Auth:        e.Auth,
		})
	}

	c.JSON(http.StatusOK, resp)
}
