package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

// sampleRegistryJSON returns a test agent registry JSON with 2 runtimes.
func sampleRegistryJSON() string {
	entries := []AgentRuntimeSpec{
		{
			ID:          "claude-agent-sdk",
			DisplayName: "Claude Code",
			Description: "Anthropic Claude with full coding capabilities",
			Framework:   "claude-agent-sdk",
			Provider:    "anthropic",
			Container: ContainerSpec{
				Image: "quay.io/ambient_code/ambient_runner:latest",
				Port:  8001,
				Env: map[string]string{
					"RUNNER_TYPE":      "claude-agent-sdk",
					"RUNNER_STATE_DIR": ".claude",
				},
				Resources: &ResourcesSpec{
					Requests: map[string]string{"cpu": "500m", "memory": "512Mi"},
					Limits:   map[string]string{"cpu": "2", "memory": "4Gi"},
				},
			},
			Sandbox: SandboxSpec{
				StateDir:    ".claude",
				Persistence: "s3",
				Seed:        SeedSpec{CloneRepos: true, HydrateState: true},
			},
			Auth: AuthSpec{
				RequiredSecretKeys: []string{"ANTHROPIC_API_KEY"},
				SecretKeyLogic:     "any",
				VertexSupported:    true,
			},
			FeatureGate: "",
		},
		{
			ID:          "gemini-cli",
			DisplayName: "Gemini CLI",
			Description: "Google Gemini coding agent",
			Framework:   "gemini-cli",
			Provider:    "google",
			Container: ContainerSpec{
				Image: "quay.io/ambient_code/ambient_runner:latest",
				Port:  9090,
				Env: map[string]string{
					"RUNNER_TYPE":      "gemini-cli",
					"RUNNER_STATE_DIR": ".gemini",
				},
			},
			Sandbox: SandboxSpec{
				StateDir:    ".gemini",
				Persistence: "s3",
				Seed:        SeedSpec{CloneRepos: true, HydrateState: true},
			},
			Auth: AuthSpec{
				RequiredSecretKeys: []string{"GEMINI_API_KEY", "GOOGLE_API_KEY"},
				SecretKeyLogic:     "any",
				VertexSupported:    true,
			},
			FeatureGate: "runner.gemini-cli.enabled",
		},
	}
	data, _ := json.Marshal(entries)
	return string(data)
}

// setupRegistryForTest installs a fake K8s client with the registry ConfigMap
// and clears the in-memory cache.
func setupRegistryForTest(t *testing.T) {
	t.Helper()

	// Write registry JSON to a temp file and point env var to it
	dir := t.TempDir()
	path := filepath.Join(dir, "agent-registry.json")
	if err := os.WriteFile(path, []byte(sampleRegistryJSON()), 0644); err != nil {
		t.Fatalf("Failed to write test registry: %v", err)
	}
	t.Setenv("AGENT_REGISTRY_PATH", path)

	// Set up fake K8s clients for auth and workspace overrides
	K8sClientMw = fake.NewSimpleClientset()
	DynamicClient = dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())

	if Namespace == "" {
		Namespace = "test-ns"
	}

	// Clear the in-memory cache
	registryCacheMu.Lock()
	registryCache = nil
	registryCacheTime = time.Time{}
	registryCacheMu.Unlock()
}

// --- GetRuntime tests ---

func TestGetRuntime_KnownID(t *testing.T) {
	setupRegistryForTest(t)

	rt, err := GetRuntime("claude-agent-sdk")
	if err != nil {
		t.Fatalf("GetRuntime failed: %v", err)
	}
	if rt.ID != "claude-agent-sdk" {
		t.Errorf("Expected ID 'claude-agent-sdk', got %q", rt.ID)
	}
	if rt.Framework != "claude-agent-sdk" {
		t.Errorf("Expected framework 'claude-agent-sdk', got %q", rt.Framework)
	}
	if rt.DisplayName != "Claude Code" {
		t.Errorf("Expected displayName 'Claude Code', got %q", rt.DisplayName)
	}
	if rt.Container.Port != 8001 {
		t.Errorf("Expected port 8001, got %d", rt.Container.Port)
	}
}

func TestGetRuntime_UnknownID(t *testing.T) {
	setupRegistryForTest(t)

	rt, err := GetRuntime("nonexistent-runner")
	if err == nil {
		t.Fatal("Expected error for unknown runner type")
	}
	if rt != nil {
		t.Errorf("Expected nil runtime for unknown runner type, got %+v", rt)
	}
}

func TestGetRuntime_FullFields(t *testing.T) {
	setupRegistryForTest(t)

	rt, err := GetRuntime("claude-agent-sdk")
	if err != nil {
		t.Fatalf("GetRuntime failed: %v", err)
	}

	if rt.Framework != "claude-agent-sdk" {
		t.Errorf("Framework: expected 'claude-agent-sdk', got %q", rt.Framework)
	}
	if rt.Provider != "anthropic" {
		t.Errorf("Provider: expected 'anthropic', got %q", rt.Provider)
	}
	if len(rt.Auth.RequiredSecretKeys) != 1 || rt.Auth.RequiredSecretKeys[0] != "ANTHROPIC_API_KEY" {
		t.Errorf("Auth.RequiredSecretKeys: expected [ANTHROPIC_API_KEY], got %v", rt.Auth.RequiredSecretKeys)
	}
	if rt.Auth.SecretKeyLogic != "any" {
		t.Errorf("Auth.SecretKeyLogic: expected 'any', got %q", rt.Auth.SecretKeyLogic)
	}
	if !rt.Auth.VertexSupported {
		t.Error("Auth.VertexSupported: expected true")
	}
	if rt.FeatureGate != "" {
		t.Errorf("FeatureGate: expected empty string, got %q", rt.FeatureGate)
	}
}

func TestGetRuntime_GeminiFields(t *testing.T) {
	setupRegistryForTest(t)

	rt, err := GetRuntime("gemini-cli")
	if err != nil {
		t.Fatalf("GetRuntime failed: %v", err)
	}

	if rt.Framework != "gemini-cli" {
		t.Errorf("Framework: expected 'gemini-cli', got %q", rt.Framework)
	}
	if rt.Provider != "google" {
		t.Errorf("Provider: expected 'google', got %q", rt.Provider)
	}
	if rt.FeatureGate != "runner.gemini-cli.enabled" {
		t.Errorf("FeatureGate: expected 'runner.gemini-cli.enabled', got %q", rt.FeatureGate)
	}
	if len(rt.Auth.RequiredSecretKeys) != 2 {
		t.Errorf("Expected 2 required secret keys, got %d", len(rt.Auth.RequiredSecretKeys))
	}
	if rt.Container.Port != 9090 {
		t.Errorf("Container.Port: expected 9090, got %d", rt.Container.Port)
	}
}

// --- GetRuntimePort tests ---

func TestGetRuntimePort_KnownType(t *testing.T) {
	setupRegistryForTest(t)

	port := GetRuntimePort("claude-agent-sdk")
	if port != 8001 {
		t.Errorf("Expected port 8001 for claude-agent-sdk, got %d", port)
	}
}

func TestGetRuntimePort_GeminiPort(t *testing.T) {
	setupRegistryForTest(t)

	port := GetRuntimePort("gemini-cli")
	if port != 9090 {
		t.Errorf("Expected port 9090 for gemini-cli, got %d", port)
	}
}

func TestGetRuntimePort_FallbackForUnknown(t *testing.T) {
	setupRegistryForTest(t)

	port := GetRuntimePort("nonexistent-runner")
	if port != DefaultRunnerPort {
		t.Errorf("Expected default port %d for unknown runner, got %d", DefaultRunnerPort, port)
	}
}

// --- getRequiredSecretKeys tests ---

func TestGetRequiredSecretKeys_Claude(t *testing.T) {
	setupRegistryForTest(t)

	keys := getRequiredSecretKeys("claude-agent-sdk")
	if len(keys) != 1 || keys[0] != "ANTHROPIC_API_KEY" {
		t.Errorf("Expected [ANTHROPIC_API_KEY], got %v", keys)
	}
}

func TestGetRequiredSecretKeys_Unknown(t *testing.T) {
	setupRegistryForTest(t)

	keys := getRequiredSecretKeys("nonexistent")
	if keys != nil {
		t.Errorf("Expected nil for unknown runner type, got %v", keys)
	}
}

// --- getContainerEnvVars tests ---

func TestGetContainerEnvVars_KnownType(t *testing.T) {
	setupRegistryForTest(t)

	envVars := getContainerEnvVars("claude-agent-sdk")
	if envVars["RUNNER_TYPE"] != "claude-agent-sdk" {
		t.Errorf("Expected RUNNER_TYPE=claude-agent-sdk, got %q", envVars["RUNNER_TYPE"])
	}
	if envVars["RUNNER_STATE_DIR"] != ".claude" {
		t.Errorf("Expected RUNNER_STATE_DIR=.claude, got %q", envVars["RUNNER_STATE_DIR"])
	}
}

func TestGetContainerEnvVars_UnknownFallback(t *testing.T) {
	setupRegistryForTest(t)

	envVars := getContainerEnvVars("nonexistent")
	if envVars["RUNNER_TYPE"] != "nonexistent" {
		t.Errorf("Fallback should set RUNNER_TYPE to the ID: expected 'nonexistent', got %q", envVars["RUNNER_TYPE"])
	}
}

// --- GetRunnerTypes handler test ---

func TestGetRunnerTypes_ReturnsProvider(t *testing.T) {
	setupRegistryForTest(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/projects/test-project/runner-types", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	c.Request = req
	c.Params = gin.Params{{Key: "projectName", Value: "test-project"}}

	GetRunnerTypes(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []RunnerTypeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Only claude-agent-sdk should be returned (empty featureGate = always enabled)
	if len(resp) != 1 {
		t.Fatalf("Expected 1 runner type (only ungated), got %d", len(resp))
	}

	claude := resp[0]
	if claude.ID != "claude-agent-sdk" {
		t.Fatalf("Expected claude-agent-sdk, got %q", claude.ID)
	}
	if claude.Provider != "anthropic" {
		t.Errorf("Provider: expected 'anthropic', got %q", claude.Provider)
	}
	if claude.Framework != "claude-agent-sdk" {
		t.Errorf("Framework: expected 'claude-agent-sdk', got %q", claude.Framework)
	}
	if claude.Auth.SecretKeyLogic != "any" {
		t.Errorf("Auth.SecretKeyLogic: expected 'any', got %q", claude.Auth.SecretKeyLogic)
	}
}

func TestGetRunnerTypes_GatedRunnersFiltered(t *testing.T) {
	setupRegistryForTest(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/projects/test-project/runner-types", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	c.Request = req
	c.Params = gin.Params{{Key: "projectName", Value: "test-project"}}

	GetRunnerTypes(c)

	var resp []RunnerTypeResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	for _, rt := range resp {
		if rt.ID == "gemini-cli" {
			t.Error("gemini-cli should be filtered out when its feature gate is disabled")
		}
	}
}

func TestIsRunnerEnabled_EmptyGate(t *testing.T) {
	setupRegistryForTest(t)

	if !isRunnerEnabled("claude-agent-sdk") {
		t.Error("claude-agent-sdk with empty featureGate should be enabled")
	}
}

func TestIsRunnerEnabled_NonEmptyGate_Disabled(t *testing.T) {
	setupRegistryForTest(t)

	if isRunnerEnabled("gemini-cli") {
		t.Error("gemini-cli should be disabled when Unleash is not configured")
	}
}

// --- isRunnerEnabledWithOverrides tests ---

func TestIsRunnerEnabledWithOverrides_OverrideTrue(t *testing.T) {
	overrides := map[string]string{"runner.gemini-cli.enabled": "true"}
	if !isRunnerEnabledWithOverrides("runner.gemini-cli.enabled", overrides) {
		t.Error("expected enabled when override is true")
	}
}

func TestIsRunnerEnabledWithOverrides_OverrideFalse(t *testing.T) {
	overrides := map[string]string{"runner.gemini-cli.enabled": "false"}
	if isRunnerEnabledWithOverrides("runner.gemini-cli.enabled", overrides) {
		t.Error("expected disabled when override is false")
	}
}

func TestIsRunnerEnabledWithOverrides_NoOverrideFallsThrough(t *testing.T) {
	overrides := map[string]string{"other.flag": "true"}
	// Without Unleash configured, FeatureEnabled returns false
	if isRunnerEnabledWithOverrides("runner.gemini-cli.enabled", overrides) {
		t.Error("expected disabled when no override and Unleash not configured")
	}
}

func TestIsRunnerEnabledWithOverrides_NilOverrides(t *testing.T) {
	// Without Unleash configured, FeatureEnabled returns false
	if isRunnerEnabledWithOverrides("runner.gemini-cli.enabled", nil) {
		t.Error("expected disabled with nil overrides and Unleash not configured")
	}
}

// --- GetRunnerTypesGlobal tests ---

func TestGetRunnerTypesGlobal_ReturnsUngatedRunners(t *testing.T) {
	setupRegistryForTest(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/runner-types", nil)

	GetRunnerTypesGlobal(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []RunnerTypeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Only ungated runners returned (gemini-cli gated, disabled without Unleash)
	if len(resp) != 1 {
		t.Fatalf("Expected 1 runner type, got %d", len(resp))
	}
	if resp[0].ID != "claude-agent-sdk" {
		t.Errorf("Expected claude-agent-sdk, got %q", resp[0].ID)
	}
	if resp[0].Provider != "anthropic" {
		t.Errorf("Expected provider anthropic, got %q", resp[0].Provider)
	}
}

func TestGetRunnerTypesGlobal_NoAuthRequired(t *testing.T) {
	setupRegistryForTest(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// No auth header
	c.Request = httptest.NewRequest(http.MethodGet, "/api/runner-types", nil)

	GetRunnerTypesGlobal(c)

	// Should succeed without auth
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 without auth, got %d", w.Code)
	}
}
