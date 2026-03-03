package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			DefaultModel: "claude-sonnet-4-5",
			Models: []ModelOption{
				{Value: "claude-sonnet-4-5", Label: "Claude Sonnet 4.5"},
				{Value: "claude-opus-4-6", Label: "Claude Opus 4.6"},
			},
			FeatureGate: "",
		},
		{
			ID:          "gemini-cli",
			DisplayName: "Gemini CLI",
			Description: "Google Gemini coding agent",
			Framework:   "gemini-cli",
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
			DefaultModel: "gemini-2.5-flash",
			Models: []ModelOption{
				{Value: "gemini-2.5-flash", Label: "Gemini 2.5 Flash"},
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

	// Ensure Namespace is set for the test
	if Namespace == "" {
		Namespace = "test-ns"
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agentRegistryConfigMapName,
			Namespace: Namespace,
		},
		Data: map[string]string{
			agentRegistryDataKey: sampleRegistryJSON(),
		},
	}
	K8sClientMw = fake.NewSimpleClientset(cm)

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

	// Framework
	if rt.Framework != "claude-agent-sdk" {
		t.Errorf("Framework: expected 'claude-agent-sdk', got %q", rt.Framework)
	}

	// Auth
	if len(rt.Auth.RequiredSecretKeys) != 1 || rt.Auth.RequiredSecretKeys[0] != "ANTHROPIC_API_KEY" {
		t.Errorf("Auth.RequiredSecretKeys: expected [ANTHROPIC_API_KEY], got %v", rt.Auth.RequiredSecretKeys)
	}
	if rt.Auth.SecretKeyLogic != "any" {
		t.Errorf("Auth.SecretKeyLogic: expected 'any', got %q", rt.Auth.SecretKeyLogic)
	}
	if !rt.Auth.VertexSupported {
		t.Error("Auth.VertexSupported: expected true")
	}

	// FeatureGate
	if rt.FeatureGate != "" {
		t.Errorf("FeatureGate: expected empty string, got %q", rt.FeatureGate)
	}

	// Models
	if len(rt.Models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(rt.Models))
	}
	if rt.DefaultModel != "claude-sonnet-4-5" {
		t.Errorf("DefaultModel: expected 'claude-sonnet-4-5', got %q", rt.DefaultModel)
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

func TestGetRunnerTypes_ReturnsFullFields(t *testing.T) {
	setupRegistryForTest(t)

	// Without Unleash initialized, FeatureEnabled returns false.
	// isRunnerEnabled returns true for runtimes with empty featureGate,
	// and false for runtimes with a non-empty featureGate.
	// In our test data: claude (featureGate="") -> enabled, gemini (featureGate="runner.gemini-cli.enabled") -> disabled.

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/runner-types", nil)

	GetRunnerTypes(c)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []RunnerTypeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Only claude-agent-sdk should be returned (empty featureGate = always enabled)
	// gemini-cli has featureGate="runner.gemini-cli.enabled" which is disabled without Unleash
	if len(resp) != 1 {
		t.Fatalf("Expected 1 runner type (only ungated), got %d", len(resp))
	}

	claude := resp[0]
	if claude.ID != "claude-agent-sdk" {
		t.Fatalf("Expected claude-agent-sdk, got %q", claude.ID)
	}

	// Verify full AgentRuntimeSpec fields are in the response
	if claude.Framework != "claude-agent-sdk" {
		t.Errorf("Framework: expected 'claude-agent-sdk', got %q", claude.Framework)
	}
	if claude.Auth.SecretKeyLogic != "any" {
		t.Errorf("Auth.SecretKeyLogic: expected 'any', got %q", claude.Auth.SecretKeyLogic)
	}
	if claude.Auth.VertexSupported != true {
		t.Error("Auth.VertexSupported: expected true")
	}
	if len(claude.Auth.RequiredSecretKeys) != 1 || claude.Auth.RequiredSecretKeys[0] != "ANTHROPIC_API_KEY" {
		t.Errorf("Auth.RequiredSecretKeys: expected [ANTHROPIC_API_KEY], got %v", claude.Auth.RequiredSecretKeys)
	}
	if claude.FeatureGate != "" {
		t.Errorf("FeatureGate: expected empty, got %q", claude.FeatureGate)
	}
	if claude.DefaultModel != "claude-sonnet-4-5" {
		t.Errorf("DefaultModel: expected 'claude-sonnet-4-5', got %q", claude.DefaultModel)
	}
	if len(claude.Models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(claude.Models))
	}
}

func TestGetRunnerTypes_GatedRunnersFiltered(t *testing.T) {
	setupRegistryForTest(t)

	// Without Unleash, gated runners should be filtered out
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/runner-types", nil)

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

	// Runtimes with empty featureGate should always be enabled
	if !isRunnerEnabled("claude-agent-sdk") {
		t.Error("claude-agent-sdk with empty featureGate should be enabled")
	}
}

func TestIsRunnerEnabled_NonEmptyGate_Disabled(t *testing.T) {
	setupRegistryForTest(t)

	// Without Unleash, non-empty featureGate should be disabled
	if isRunnerEnabled("gemini-cli") {
		t.Error("gemini-cli should be disabled when Unleash is not configured")
	}
}
