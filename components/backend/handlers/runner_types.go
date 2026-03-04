package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const agentRegistryConfigMapName = "ambient-agent-registry"
const agentRegistryDataKey = "agent-registry.json"

// DefaultRunnerType is the default runner type when none is specified.
const DefaultRunnerType = "claude-agent-sdk"

// DefaultRunnerPort is used when a runtime's container port is not set.
const DefaultRunnerPort = 8001

// AgentRuntimeSpec is parsed from the agent registry ConfigMap JSON.
// NOTE: These types are duplicated in components/operator/internal/handlers/registry.go.
// Keep both in sync when modifying the schema.
// It is the single source of truth for runtime configuration.
type AgentRuntimeSpec struct {
	ID           string        `json:"id"`
	DisplayName  string        `json:"displayName"`
	Description  string        `json:"description"`
	Framework    string        `json:"framework"`
	Container    ContainerSpec `json:"container"`
	Sandbox      SandboxSpec   `json:"sandbox"`
	Auth         AuthSpec      `json:"auth"`
	DefaultModel string        `json:"defaultModel"`
	Models       []ModelOption `json:"models"`
	FeatureGate  string        `json:"featureGate"`
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

// ModelOption represents a model choice within a runner type.
type ModelOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// RunnerTypeResponse is the public API shape returned to the frontend.
// FeatureGate is intentionally excluded — gated runners are already filtered
// out by the handler, so the frontend never needs to see the gate name.
type RunnerTypeResponse struct {
	ID           string        `json:"id"`
	DisplayName  string        `json:"displayName"`
	Description  string        `json:"description"`
	Framework    string        `json:"framework"`
	DefaultModel string        `json:"defaultModel"`
	Models       []ModelOption `json:"models"`
	Auth         AuthSpec      `json:"auth"`
}

// In-memory cache for the agent registry (ConfigMap content changes rarely).
var (
	registryCache     []AgentRuntimeSpec
	registryCacheMu   sync.RWMutex
	registryCacheTime time.Time
)

const registryCacheTTL = 60 * time.Second

// loadAgentRegistry reads and parses the agent registry ConfigMap using the backend service account.
// Results are cached in-memory with a TTL since the ConfigMap content rarely changes.
func loadAgentRegistry() ([]AgentRuntimeSpec, error) {
	registryCacheMu.RLock()
	if time.Since(registryCacheTime) < registryCacheTTL && registryCache != nil {
		defer registryCacheMu.RUnlock()
		return registryCache, nil
	}
	registryCacheMu.RUnlock()

	if K8sClientMw == nil {
		return nil, fmt.Errorf("backend K8s client not initialized")
	}

	cm, err := K8sClientMw.CoreV1().ConfigMaps(Namespace).Get(
		context.Background(), agentRegistryConfigMapName, v1.GetOptions{},
	)
	if err != nil {
		// On refresh failure, return stale cache if available
		registryCacheMu.RLock()
		if registryCache != nil {
			defer registryCacheMu.RUnlock()
			log.Printf("Warning: failed to refresh agent registry, using stale cache: %v", err)
			return registryCache, nil
		}
		registryCacheMu.RUnlock()
		return nil, fmt.Errorf("failed to read ConfigMap %s: %w", agentRegistryConfigMapName, err)
	}

	rawJSON, ok := cm.Data[agentRegistryDataKey]
	if !ok {
		return nil, fmt.Errorf("ConfigMap %s missing key %q", agentRegistryConfigMapName, agentRegistryDataKey)
	}

	var entries []AgentRuntimeSpec
	if err := json.Unmarshal([]byte(rawJSON), &entries); err != nil {
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

// GetRunnerTypes handles GET /api/runner-types and returns the list of available runner types.
// Runners gated by feature flags are filtered out.
func GetRunnerTypes(c *gin.Context) {
	entries, err := loadAgentRegistry()
	if err != nil {
		log.Printf("Failed to load agent registry: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load runner types"})
		return
	}

	resp := make([]RunnerTypeResponse, 0, len(entries))
	for _, e := range entries {
		// Check feature gate directly instead of calling isRunnerEnabled (which
		// re-loads the registry per entry — N+1 pattern).
		if e.FeatureGate != "" && !FeatureEnabled(e.FeatureGate) {
			continue
		}
		resp = append(resp, RunnerTypeResponse{
			ID:           e.ID,
			DisplayName:  e.DisplayName,
			Description:  e.Description,
			Framework:    e.Framework,
			DefaultModel: e.DefaultModel,
			Models:       e.Models,
			Auth:         e.Auth,
		})
	}

	c.JSON(http.StatusOK, resp)
}
