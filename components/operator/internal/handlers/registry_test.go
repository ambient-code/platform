package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"ambient-code-operator/internal/config"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// testRegistryJSON returns a sample agent registry JSON for testing.
func testRegistryJSON() string {
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
				StateDir:               ".claude",
				StateSyncImage:         "quay.io/ambient_code/state_sync:latest",
				Persistence:            "s3",
				WorkspaceSize:          "10Gi",
				TerminationGracePeriod: 60,
				Seed:                   SeedSpec{CloneRepos: true, HydrateState: true},
			},
			Auth: AuthSpec{
				RequiredSecretKeys: []string{"ANTHROPIC_API_KEY"},
				SecretKeyLogic:     "any",
				VertexSupported:    true,
			},
			Provider:    "anthropic",
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
				Persistence: "none",
				Seed:        SeedSpec{CloneRepos: false, HydrateState: false},
			},
			Auth: AuthSpec{
				RequiredSecretKeys: []string{"GEMINI_API_KEY"},
				SecretKeyLogic:     "any",
				VertexSupported:    true,
			},
			Provider:    "google",
			FeatureGate: "runner.gemini-cli.enabled",
		},
		{
			ID:          "lightweight-runner",
			DisplayName: "Lightweight Runner",
			Description: "Minimal runner with no persistence and no seeding",
			Framework:   "lightweight",
			Container: ContainerSpec{
				Image: "quay.io/ambient_code/ambient_runner:light",
				Port:  8002,
			},
			Sandbox: SandboxSpec{
				Persistence: "none",
				Seed:        SeedSpec{CloneRepos: false, HydrateState: false},
			},
		},
		{
			ID:          "seed-only-runner",
			DisplayName: "Seed Only Runner",
			Description: "Runner that clones repos but has no persistence",
			Framework:   "seed-only",
			Container: ContainerSpec{
				Image: "quay.io/ambient_code/ambient_runner:seed",
				Port:  8003,
			},
			Sandbox: SandboxSpec{
				Persistence: "none",
				Seed:        SeedSpec{CloneRepos: true, HydrateState: false},
			},
		},
	}
	data, _ := json.Marshal(entries)
	return string(data)
}

// setupRegistryConfigMap creates a fake K8s client with the registry ConfigMap and
// clears the in-memory cache so tests get fresh data.
func setupRegistryConfigMap(t *testing.T, namespace string) {
	t.Helper()

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agentRegistryConfigMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			agentRegistryDataKey: testRegistryJSON(),
		},
	}
	config.K8sClient = fake.NewSimpleClientset(cm)

	// Clear registry cache so each test starts fresh
	runtimeRegistryCacheMu.Lock()
	runtimeRegistryCache = nil
	runtimeRegistryCacheMu.Unlock()
}

// --- Registry loading tests ---

func TestLoadRuntimeRegistry_Success(t *testing.T) {
	setupRegistryConfigMap(t, "ambient-system")
	// loadRuntimeRegistry reads from config.LoadConfig().Namespace which reads NAMESPACE env
	t.Setenv("NAMESPACE", "ambient-system")

	entries, err := loadRuntimeRegistry()
	if err != nil {
		t.Fatalf("loadRuntimeRegistry failed: %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("Expected 4 entries, got %d", len(entries))
	}
}

func TestLoadRuntimeRegistry_CachesResults(t *testing.T) {
	setupRegistryConfigMap(t, "ambient-system")
	t.Setenv("NAMESPACE", "ambient-system")

	first, err := loadRuntimeRegistry()
	if err != nil {
		t.Fatalf("First load failed: %v", err)
	}

	// Delete the ConfigMap — cached result should still be returned
	_ = config.K8sClient.CoreV1().ConfigMaps("ambient-system").Delete(
		context.Background(), agentRegistryConfigMapName, metav1.DeleteOptions{},
	)

	second, err := loadRuntimeRegistry()
	if err != nil {
		t.Fatalf("Second load (from cache) failed: %v", err)
	}
	if len(first) != len(second) {
		t.Errorf("Cache should return same data: first=%d, second=%d", len(first), len(second))
	}
}

// --- getRuntimeSpec tests ---

func TestGetRuntimeSpec_KnownType(t *testing.T) {
	setupRegistryConfigMap(t, "ambient-system")
	t.Setenv("NAMESPACE", "ambient-system")

	rt := getRuntimeSpec("claude-agent-sdk")
	if rt == nil {
		t.Fatal("Expected non-nil runtime for claude-agent-sdk")
	}
	if rt.ID != "claude-agent-sdk" {
		t.Errorf("Expected ID 'claude-agent-sdk', got %q", rt.ID)
	}
	if rt.Container.Port != 8001 {
		t.Errorf("Expected port 8001, got %d", rt.Container.Port)
	}
	if rt.Framework != "claude-agent-sdk" {
		t.Errorf("Expected framework 'claude-agent-sdk', got %q", rt.Framework)
	}
}

func TestGetRuntimeSpec_UnknownType(t *testing.T) {
	setupRegistryConfigMap(t, "ambient-system")
	t.Setenv("NAMESPACE", "ambient-system")

	rt := getRuntimeSpec("nonexistent-runner")
	if rt != nil {
		t.Errorf("Expected nil for unknown runner type, got %+v", rt)
	}
}

func TestGetRuntimeSpec_GeminiPort(t *testing.T) {
	setupRegistryConfigMap(t, "ambient-system")
	t.Setenv("NAMESPACE", "ambient-system")

	rt := getRuntimeSpec("gemini-cli")
	if rt == nil {
		t.Fatal("Expected non-nil runtime for gemini-cli")
	}
	if rt.Container.Port != 9090 {
		t.Errorf("Expected port 9090 for gemini-cli, got %d", rt.Container.Port)
	}
}

// --- Conditional pod component decision tests ---
// These test the same decision logic used in handleAgenticSessionEvent

func TestConditionalComponents_FullSandbox(t *testing.T) {
	// persistence=s3, seed both true → includes init container, sidecar, workspace volume
	setupRegistryConfigMap(t, "ambient-system")
	t.Setenv("NAMESPACE", "ambient-system")

	runtime := getRuntimeSpec("claude-agent-sdk")
	if runtime == nil {
		t.Fatal("Expected non-nil runtime for claude-agent-sdk")
	}

	needsWorkspace := runtime.Sandbox.Persistence != "none" || runtime.Sandbox.Seed.CloneRepos
	needsInitContainer := runtime.Sandbox.Seed.CloneRepos || runtime.Sandbox.Seed.HydrateState
	needsStateSyncSidecar := runtime.Sandbox.Persistence != "none"

	if !needsWorkspace {
		t.Error("Full sandbox (persistence=s3, cloneRepos=true) should need workspace volume")
	}
	if !needsInitContainer {
		t.Error("Full sandbox (cloneRepos=true, hydrateState=true) should need init container")
	}
	if !needsStateSyncSidecar {
		t.Error("Full sandbox (persistence=s3) should need state-sync sidecar")
	}
}

func TestConditionalComponents_NoSandbox(t *testing.T) {
	// persistence=none, seed both false → NO init container, NO sidecar, NO workspace volume
	setupRegistryConfigMap(t, "ambient-system")
	t.Setenv("NAMESPACE", "ambient-system")

	runtime := getRuntimeSpec("lightweight-runner")
	if runtime == nil {
		t.Fatal("Expected non-nil runtime for lightweight-runner")
	}

	needsWorkspace := runtime.Sandbox.Persistence != "none" || runtime.Sandbox.Seed.CloneRepos
	needsInitContainer := runtime.Sandbox.Seed.CloneRepos || runtime.Sandbox.Seed.HydrateState
	needsStateSyncSidecar := runtime.Sandbox.Persistence != "none"

	if needsWorkspace {
		t.Error("Lightweight (persistence=none, cloneRepos=false) should NOT need workspace volume")
	}
	if needsInitContainer {
		t.Error("Lightweight (cloneRepos=false, hydrateState=false) should NOT need init container")
	}
	if needsStateSyncSidecar {
		t.Error("Lightweight (persistence=none) should NOT need state-sync sidecar")
	}
}

func TestConditionalComponents_CloneReposOnly(t *testing.T) {
	// persistence=none, cloneRepos=true → includes init container and workspace volume, NO sidecar
	setupRegistryConfigMap(t, "ambient-system")
	t.Setenv("NAMESPACE", "ambient-system")

	runtime := getRuntimeSpec("seed-only-runner")
	if runtime == nil {
		t.Fatal("Expected non-nil runtime for seed-only-runner")
	}

	needsWorkspace := runtime.Sandbox.Persistence != "none" || runtime.Sandbox.Seed.CloneRepos
	needsInitContainer := runtime.Sandbox.Seed.CloneRepos || runtime.Sandbox.Seed.HydrateState
	needsStateSyncSidecar := runtime.Sandbox.Persistence != "none"

	if !needsWorkspace {
		t.Error("Seed-only (cloneRepos=true) should need workspace volume even with persistence=none")
	}
	if !needsInitContainer {
		t.Error("Seed-only (cloneRepos=true) should need init container")
	}
	if needsStateSyncSidecar {
		t.Error("Seed-only (persistence=none) should NOT need state-sync sidecar")
	}
}

func TestContainerImageFromRegistry(t *testing.T) {
	// Container image from registry overrides env var default
	setupRegistryConfigMap(t, "ambient-system")
	t.Setenv("NAMESPACE", "ambient-system")
	t.Setenv("AMBIENT_CODE_RUNNER_IMAGE", "quay.io/ambient_code/ambient_runner:fallback")

	runtime := getRuntimeSpec("claude-agent-sdk")
	if runtime == nil {
		t.Fatal("Expected non-nil runtime")
	}

	// Simulate the override logic from sessions.go
	envImage := "quay.io/ambient_code/ambient_runner:fallback"
	runnerImage := envImage
	if runtime.Container.Image != "" {
		runnerImage = runtime.Container.Image
	}

	expected := "quay.io/ambient_code/ambient_runner:latest"
	if runnerImage != expected {
		t.Errorf("Registry image should override env var: expected %q, got %q", expected, runnerImage)
	}
}

func TestFallbackWhenRegistryReturnsNil(t *testing.T) {
	// When registry returns nil for unknown runner type, env var defaults are used
	setupRegistryConfigMap(t, "ambient-system")
	t.Setenv("NAMESPACE", "ambient-system")

	runtime := getRuntimeSpec("nonexistent-runner")
	if runtime != nil {
		t.Fatal("Expected nil for unknown runner type")
	}

	// Simulate fallback logic from sessions.go
	envImage := "quay.io/ambient_code/ambient_runner:env-fallback"
	runnerImage := envImage
	if runtime != nil && runtime.Container.Image != "" {
		runnerImage = runtime.Container.Image
	}

	if runnerImage != envImage {
		t.Errorf("Should fall back to env var when runtime is nil: expected %q, got %q", envImage, runnerImage)
	}

	// Port fallback
	runnerPort := int32(8001) // default
	if runtime != nil && runtime.Container.Port > 0 {
		runnerPort = int32(runtime.Container.Port)
	}
	if runnerPort != 8001 {
		t.Errorf("Should fall back to default port 8001, got %d", runnerPort)
	}

	// Workspace/init/sidecar should default to true when runtime is nil
	needsWorkspace := true
	needsInitContainer := true
	needsStateSyncSidecar := true
	if runtime != nil {
		needsWorkspace = runtime.Sandbox.Persistence != "none" || runtime.Sandbox.Seed.CloneRepos
		needsInitContainer = runtime.Sandbox.Seed.CloneRepos || runtime.Sandbox.Seed.HydrateState
		needsStateSyncSidecar = runtime.Sandbox.Persistence != "none"
	}
	if !needsWorkspace || !needsInitContainer || !needsStateSyncSidecar {
		t.Error("All pod components should default to true when runtime is nil")
	}
}

func TestPortFromRegistryUsedInService(t *testing.T) {
	// Port from registry is used in Service spec (not hardcoded 8001)
	setupRegistryConfigMap(t, "ambient-system")
	t.Setenv("NAMESPACE", "ambient-system")

	runtime := getRuntimeSpec("gemini-cli")
	if runtime == nil {
		t.Fatal("Expected non-nil runtime for gemini-cli")
	}

	// Simulate the port resolution from sessions.go
	runnerPort := int32(8001)
	if runtime != nil && runtime.Container.Port > 0 {
		runnerPort = int32(runtime.Container.Port)
	}

	if runnerPort != 9090 {
		t.Errorf("Service should use port from registry: expected 9090, got %d", runnerPort)
	}
}

func TestGetRunnerPort_FromService(t *testing.T) {
	// getRunnerPort reads port from Service "agui" port
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "session-test-session",
			Namespace: "test-ns",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Name: "agui", Port: 9090, Protocol: corev1.ProtocolTCP},
			},
		},
	}
	config.K8sClient = fake.NewSimpleClientset(svc)

	port := getRunnerPort("test-ns", "test-session")
	if port != 9090 {
		t.Errorf("Expected port 9090 from service, got %d", port)
	}
}

func TestGetRunnerPort_FallbackWhenServiceNotFound(t *testing.T) {
	config.K8sClient = fake.NewSimpleClientset()

	port := getRunnerPort("test-ns", "nonexistent-session")
	if port != 8001 {
		t.Errorf("Expected fallback port 8001, got %d", port)
	}
}

func TestGetRunnerPort_FallbackWhenNoAguiPort(t *testing.T) {
	// Service exists but has no "agui" named port
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "session-test-session",
			Namespace: "test-ns",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Name: "other", Port: 3000, Protocol: corev1.ProtocolTCP},
			},
		},
	}
	config.K8sClient = fake.NewSimpleClientset(svc)

	port := getRunnerPort("test-ns", "test-session")
	if port != 8001 {
		t.Errorf("Expected fallback port 8001 when no 'agui' port, got %d", port)
	}
}

func TestRegistryAuth_RequiredSecretKeys(t *testing.T) {
	setupRegistryConfigMap(t, "ambient-system")
	t.Setenv("NAMESPACE", "ambient-system")

	rt := getRuntimeSpec("claude-agent-sdk")
	if rt == nil {
		t.Fatal("Expected non-nil runtime")
	}
	if len(rt.Auth.RequiredSecretKeys) != 1 || rt.Auth.RequiredSecretKeys[0] != "ANTHROPIC_API_KEY" {
		t.Errorf("Expected [ANTHROPIC_API_KEY], got %v", rt.Auth.RequiredSecretKeys)
	}

	rtGemini := getRuntimeSpec("gemini-cli")
	if rtGemini == nil {
		t.Fatal("Expected non-nil runtime for gemini-cli")
	}
	if len(rtGemini.Auth.RequiredSecretKeys) != 1 || rtGemini.Auth.RequiredSecretKeys[0] != "GEMINI_API_KEY" {
		t.Errorf("Expected [GEMINI_API_KEY], got %v", rtGemini.Auth.RequiredSecretKeys)
	}
}

func TestRegistryResourceOverrides(t *testing.T) {
	setupRegistryConfigMap(t, "ambient-system")
	t.Setenv("NAMESPACE", "ambient-system")

	rt := getRuntimeSpec("claude-agent-sdk")
	if rt == nil {
		t.Fatal("Expected non-nil runtime")
	}
	if rt.Container.Resources == nil {
		t.Fatal("Expected non-nil resources for claude-agent-sdk")
	}
	if rt.Container.Resources.Requests["cpu"] != "500m" {
		t.Errorf("Expected cpu request '500m', got %q", rt.Container.Resources.Requests["cpu"])
	}
	if rt.Container.Resources.Limits["memory"] != "4Gi" {
		t.Errorf("Expected memory limit '4Gi', got %q", rt.Container.Resources.Limits["memory"])
	}
}
