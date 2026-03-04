package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"ambient-code-operator/internal/config"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	agentRegistryConfigMapName = "ambient-agent-registry"
	agentRegistryDataKey       = "agent-registry.json"
	agentRegistryCacheTTL      = 60 * time.Second

	// Persistence mode constants for SandboxSpec.Persistence.
	persistenceNone = "none"
	persistenceS3   = "s3"

	// defaultRunnerPort is the fallback AG-UI server port.
	defaultRunnerPort = 8001
)

// AgentRuntimeSpec — parsed from registry ConfigMap JSON.
// NOTE: These types are duplicated in components/backend/handlers/runner_types.go.
// Keep both in sync when modifying the schema.
type AgentRuntimeSpec struct {
	ID          string        `json:"id"`
	DisplayName string        `json:"displayName"`
	Description string        `json:"description"`
	Framework   string        `json:"framework"`
	Provider    string        `json:"provider"`
	Container   ContainerSpec `json:"container"`
	Sandbox     SandboxSpec   `json:"sandbox"`
	Auth        AuthSpec      `json:"auth"`
	FeatureGate string        `json:"featureGate"`
}

// ContainerSpec defines the runner container configuration.
type ContainerSpec struct {
	Image     string            `json:"image"`
	Port      int               `json:"port"`
	Env       map[string]string `json:"env"`
	Resources *ResourcesSpec    `json:"resources,omitempty"`
}

// ResourcesSpec defines Kubernetes resource requests and limits.
type ResourcesSpec struct {
	Requests map[string]string `json:"requests,omitempty"`
	Limits   map[string]string `json:"limits,omitempty"`
}

// SandboxSpec defines sandbox (pod-level) configuration.
type SandboxSpec struct {
	StateDir               string   `json:"stateDir,omitempty"`
	StateSyncImage         string   `json:"stateSyncImage,omitempty"`
	Persistence            string   `json:"persistence"`
	WorkspaceSize          string   `json:"workspaceSize,omitempty"`
	TerminationGracePeriod int      `json:"terminationGracePeriod,omitempty"`
	Seed                   SeedSpec `json:"seed"`
}

// SeedSpec defines init container seeding behavior.
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

// In-memory cache for the agent runtime registry.
var (
	runtimeRegistryCache     []AgentRuntimeSpec
	runtimeRegistryCacheMu   sync.RWMutex
	runtimeRegistryCacheTime time.Time
)

// defaultRegistryPath is where the agent-registry ConfigMap is mounted.
const defaultRegistryPath = "/config/registry/agent-registry.json"

// registryFilePath returns the filesystem path to the agent registry JSON.
func registryFilePath() string {
	if p := os.Getenv("AGENT_REGISTRY_PATH"); p != "" {
		return p
	}
	return defaultRegistryPath
}

// loadRuntimeRegistry reads and parses the agent registry from the mounted ConfigMap file.
// Results are cached in-memory with a 60s TTL.
func loadRuntimeRegistry() ([]AgentRuntimeSpec, error) {
	runtimeRegistryCacheMu.RLock()
	if time.Since(runtimeRegistryCacheTime) < agentRegistryCacheTTL && runtimeRegistryCache != nil {
		defer runtimeRegistryCacheMu.RUnlock()
		return runtimeRegistryCache, nil
	}
	runtimeRegistryCacheMu.RUnlock()

	data, err := os.ReadFile(registryFilePath())
	if err != nil {
		// On read failure, return stale cache if available
		runtimeRegistryCacheMu.RLock()
		if runtimeRegistryCache != nil {
			defer runtimeRegistryCacheMu.RUnlock()
			log.Printf("Warning: failed to refresh agent registry, using stale cache: %v", err)
			return runtimeRegistryCache, nil
		}
		runtimeRegistryCacheMu.RUnlock()
		return nil, fmt.Errorf("failed to read agent registry from %s: %w", registryFilePath(), err)
	}

	var entries []AgentRuntimeSpec
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse agent registry JSON: %w", err)
	}

	runtimeRegistryCacheMu.Lock()
	runtimeRegistryCache = entries
	runtimeRegistryCacheTime = time.Now()
	runtimeRegistryCacheMu.Unlock()

	log.Printf("Loaded %d runtime entries from agent registry ConfigMap", len(entries))
	return entries, nil
}

// getRunnerPort looks up the runner port for a session by reading its Service spec.
// Falls back to 8001 if the service or port cannot be determined.
func getRunnerPort(namespace, sessionName string) int32 {
	svcName := fmt.Sprintf("session-%s", sessionName)
	svc, err := config.K8sClient.CoreV1().Services(namespace).Get(
		context.Background(), svcName, v1.GetOptions{},
	)
	if err != nil {
		return defaultRunnerPort
	}
	for _, port := range svc.Spec.Ports {
		if port.Name == "agui" {
			return port.Port
		}
	}
	return defaultRunnerPort
}

// getRuntimeSpec looks up a runtime by ID from the registry.
// Returns nil if the registry is unavailable or the ID is not found.
func getRuntimeSpec(runnerTypeID string) *AgentRuntimeSpec {
	entries, err := loadRuntimeRegistry()
	if err != nil {
		log.Printf("Warning: failed to load agent registry: %v (using fallback defaults)", err)
		return nil
	}
	for i := range entries {
		if entries[i].ID == runnerTypeID {
			spec := entries[i]
			return &spec
		}
	}
	log.Printf("Warning: runner type %q not found in registry (using fallback defaults)", runnerTypeID)
	return nil
}
