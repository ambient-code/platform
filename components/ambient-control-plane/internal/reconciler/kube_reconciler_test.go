package reconciler

import (
	"encoding/json"
	"testing"
)

// noSessionEnv is a convenience alias for tests that don't need session env vars.
var noSessionEnv = map[string]string{}

func TestBuildCredentialSidecars_NoCredentials(t *testing.T) {
	r := &SimpleKubeReconciler{cfg: KubeReconcilerConfig{}}
	sidecars, urls, _ := r.buildCredentialSidecars("test-session", "test-namespace", map[string]string{}, noSessionEnv)
	if len(sidecars) != 0 {
		t.Errorf("expected 0 sidecars, got %d", len(sidecars))
	}
	if len(urls) != 0 {
		t.Errorf("expected 0 urls, got %d", len(urls))
	}
}

func TestBuildCredentialSidecars_NoImageConfigured(t *testing.T) {
	r := &SimpleKubeReconciler{cfg: KubeReconcilerConfig{}}
	credentialIDs := map[string]string{"github": "cred-123"}
	sidecars, urls, _ := r.buildCredentialSidecars("test-session", "test-namespace", credentialIDs, noSessionEnv)
	if len(sidecars) != 0 {
		t.Errorf("expected 0 sidecars (no image configured), got %d", len(sidecars))
	}
	if len(urls) != 0 {
		t.Errorf("expected 0 urls, got %d", len(urls))
	}
}

func TestBuildCredentialSidecars_GitHubSidecar(t *testing.T) {
	r := &SimpleKubeReconciler{
		cfg: KubeReconcilerConfig{
			GitHubMCPImage:   "ghcr.io/github/github-mcp-server:latest",
			MCPAPIServerURL:  "http://api.svc:8000",
			CPTokenURL:       "http://cp.svc:8080",
			CPTokenPublicKey: "test-key",
		},
	}
	r.logger = r.logger.With().Logger()

	credentialIDs := map[string]string{"github": "cred-123"}
	sidecars, urls, _ := r.buildCredentialSidecars("test-session", "test-namespace", credentialIDs, noSessionEnv)

	if len(sidecars) != 1 {
		t.Fatalf("expected 1 sidecar, got %d", len(sidecars))
	}
	if len(urls) != 1 {
		t.Fatalf("expected 1 url, got %d", len(urls))
	}

	url, ok := urls["github"]
	if !ok {
		t.Fatal("expected github url")
	}
	if url != "http://localhost:8091" {
		t.Errorf("expected http://localhost:8091, got %s", url)
	}

	sidecar := sidecars[0].(map[string]interface{})
	if sidecar["name"] != "credential-github" {
		t.Errorf("expected container name credential-github, got %s", sidecar["name"])
	}
	if sidecar["image"] != "ghcr.io/github/github-mcp-server:latest" {
		t.Errorf("unexpected image: %s", sidecar["image"])
	}

	ports := sidecar["ports"].([]interface{})
	port := ports[0].(map[string]interface{})
	if port["containerPort"] != int64(8091) {
		t.Errorf("expected port 8091, got %v", port["containerPort"])
	}

	secCtx := sidecar["securityContext"].(map[string]interface{})
	if secCtx["allowPrivilegeEscalation"] != false {
		t.Error("expected allowPrivilegeEscalation=false")
	}
}

func TestBuildCredentialSidecars_MultipleSidecars(t *testing.T) {
	r := &SimpleKubeReconciler{
		cfg: KubeReconcilerConfig{
			GitHubMCPImage:   "github-mcp:latest",
			JiraMCPImage:     "jira-mcp:latest",
			K8sMCPImage:      "k8s-mcp:latest",
			GoogleMCPImage:   "google-mcp:latest",
			MCPAPIServerURL:  "http://api.svc:8000",
			CPTokenURL:       "http://cp.svc:8080",
			CPTokenPublicKey: "test-key",
		},
	}
	r.logger = r.logger.With().Logger()

	credentialIDs := map[string]string{
		"github":     "cred-1",
		"jira":       "cred-2",
		"kubeconfig": "cred-3",
		"google":     "cred-4",
	}
	sidecars, urls, _ := r.buildCredentialSidecars("test-session", "test-namespace", credentialIDs, noSessionEnv)

	if len(sidecars) != 4 {
		t.Fatalf("expected 4 sidecars, got %d", len(sidecars))
	}
	if len(urls) != 4 {
		t.Fatalf("expected 4 urls, got %d", len(urls))
	}

	expectedPorts := map[string]string{
		"github":     "http://localhost:8091",
		"jira":       "http://localhost:8092",
		"kubeconfig": "http://localhost:8093",
		"google":     "http://localhost:8094",
	}
	for provider, expectedURL := range expectedPorts {
		if urls[provider] != expectedURL {
			t.Errorf("provider %s: expected %s, got %s", provider, expectedURL, urls[provider])
		}
	}
}

func TestBuildCredentialSidecars_UnknownProvider(t *testing.T) {
	r := &SimpleKubeReconciler{cfg: KubeReconcilerConfig{}}
	r.logger = r.logger.With().Logger()

	credentialIDs := map[string]string{"unknown-provider": "cred-999"}
	sidecars, urls, _ := r.buildCredentialSidecars("test-session", "test-namespace", credentialIDs, noSessionEnv)

	if len(sidecars) != 0 {
		t.Errorf("expected 0 sidecars for unknown provider, got %d", len(sidecars))
	}
	if len(urls) != 0 {
		t.Errorf("expected 0 urls for unknown provider, got %d", len(urls))
	}
}

func TestBuildCredentialSidecars_LocalImagePullPolicy(t *testing.T) {
	r := &SimpleKubeReconciler{
		cfg: KubeReconcilerConfig{
			GitHubMCPImage: "localhost/github-mcp:latest",
		},
	}
	r.logger = r.logger.With().Logger()

	credentialIDs := map[string]string{"github": "cred-123"}
	sidecars, _, _ := r.buildCredentialSidecars("test-session", "test-namespace", credentialIDs, noSessionEnv)

	if len(sidecars) != 1 {
		t.Fatalf("expected 1 sidecar, got %d", len(sidecars))
	}

	sidecar := sidecars[0].(map[string]interface{})
	if sidecar["imagePullPolicy"] != "IfNotPresent" {
		t.Errorf("expected IfNotPresent for localhost image, got %s", sidecar["imagePullPolicy"])
	}
}

func TestCredentialMCPURLsJSON(t *testing.T) {
	urls := map[string]string{
		"github": "http://localhost:8091",
		"jira":   "http://localhost:8092",
	}
	raw, err := json.Marshal(urls)
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["github"] != "http://localhost:8091" {
		t.Error("round-trip failed for github")
	}
	if parsed["jira"] != "http://localhost:8092" {
		t.Error("round-trip failed for jira")
	}
}

// ---------------------------------------------------------------------------
// Tests for JIRA_READ_ONLY_MODE propagation (Issue #1506)
// ---------------------------------------------------------------------------

func findEnvVar(env []interface{}, name string) (string, bool) {
	for _, item := range env {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if m["name"] == name {
			val, _ := m["value"].(string)
			return val, true
		}
	}
	return "", false
}

func TestBuildCredentialSidecars_JiraReadOnlyMode_SetToFalse(t *testing.T) {
	// When jira-write is enabled, JIRA_READ_ONLY_MODE=false must be passed to
	// the Jira sidecar so mcp-atlassian exposes write tools.
	r := &SimpleKubeReconciler{
		cfg: KubeReconcilerConfig{
			JiraMCPImage:     "jira-mcp:latest",
			MCPAPIServerURL:  "http://api.svc:8000",
			CPTokenURL:       "http://cp.svc:8080",
			CPTokenPublicKey: "test-key",
		},
	}
	r.logger = r.logger.With().Logger()

	credentialIDs := map[string]string{"jira": "cred-42"}
	sessionEnv := map[string]string{"JIRA_READ_ONLY_MODE": "false"}

	sidecars, _, _ := r.buildCredentialSidecars("test-session", "test-ns", credentialIDs, sessionEnv)

	if len(sidecars) != 1 {
		t.Fatalf("expected 1 sidecar, got %d", len(sidecars))
	}

	sidecar := sidecars[0].(map[string]interface{})
	env := sidecar["env"].([]interface{})

	val, found := findEnvVar(env, "JIRA_READ_ONLY_MODE")
	if !found {
		t.Fatal("JIRA_READ_ONLY_MODE not found in Jira sidecar env")
	}
	if val != "false" {
		t.Errorf("expected JIRA_READ_ONLY_MODE=false, got %q", val)
	}
}

func TestBuildCredentialSidecars_JiraReadOnlyMode_NotSetByDefault(t *testing.T) {
	// When jira-write is not enabled, JIRA_READ_ONLY_MODE must NOT be injected
	// (mcp-atlassian defaults to read-only, which is the safe default).
	r := &SimpleKubeReconciler{
		cfg: KubeReconcilerConfig{
			JiraMCPImage:     "jira-mcp:latest",
			MCPAPIServerURL:  "http://api.svc:8000",
			CPTokenURL:       "http://cp.svc:8080",
			CPTokenPublicKey: "test-key",
		},
	}
	r.logger = r.logger.With().Logger()

	credentialIDs := map[string]string{"jira": "cred-42"}

	sidecars, _, _ := r.buildCredentialSidecars("test-session", "test-ns", credentialIDs, noSessionEnv)

	if len(sidecars) != 1 {
		t.Fatalf("expected 1 sidecar, got %d", len(sidecars))
	}

	sidecar := sidecars[0].(map[string]interface{})
	env := sidecar["env"].([]interface{})

	_, found := findEnvVar(env, "JIRA_READ_ONLY_MODE")
	if found {
		t.Error("JIRA_READ_ONLY_MODE should not be present when jira-write is disabled")
	}
}

func TestBuildCredentialSidecars_JiraReadOnly_NotPropagatedToGitHub(t *testing.T) {
	// JIRA_READ_ONLY_MODE must only be injected into the Jira sidecar, never
	// into other provider sidecars (e.g. GitHub).
	r := &SimpleKubeReconciler{
		cfg: KubeReconcilerConfig{
			GitHubMCPImage:   "github-mcp:latest",
			JiraMCPImage:     "jira-mcp:latest",
			MCPAPIServerURL:  "http://api.svc:8000",
			CPTokenURL:       "http://cp.svc:8080",
			CPTokenPublicKey: "test-key",
		},
	}
	r.logger = r.logger.With().Logger()

	credentialIDs := map[string]string{"github": "cred-1", "jira": "cred-2"}
	sessionEnv := map[string]string{"JIRA_READ_ONLY_MODE": "false"}

	sidecars, _, _ := r.buildCredentialSidecars("test-session", "test-ns", credentialIDs, sessionEnv)

	if len(sidecars) != 2 {
		t.Fatalf("expected 2 sidecars, got %d", len(sidecars))
	}

	for _, item := range sidecars {
		sidecar := item.(map[string]interface{})
		name := sidecar["name"].(string)
		if name == "credential-github" {
			env := sidecar["env"].([]interface{})
			if _, found := findEnvVar(env, "JIRA_READ_ONLY_MODE"); found {
				t.Errorf("JIRA_READ_ONLY_MODE must not be injected into GitHub sidecar")
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Tests for parseSessionEnvVars
// ---------------------------------------------------------------------------

func TestParseSessionEnvVars_ValidJSON(t *testing.T) {
	raw := `{"JIRA_READ_ONLY_MODE":"false","FOO":"bar"}`
	got := parseSessionEnvVars(raw)
	if got["JIRA_READ_ONLY_MODE"] != "false" {
		t.Errorf("expected false, got %q", got["JIRA_READ_ONLY_MODE"])
	}
	if got["FOO"] != "bar" {
		t.Errorf("expected bar, got %q", got["FOO"])
	}
}

func TestParseSessionEnvVars_Empty(t *testing.T) {
	got := parseSessionEnvVars("")
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestParseSessionEnvVars_InvalidJSON(t *testing.T) {
	got := parseSessionEnvVars("not-json")
	if len(got) != 0 {
		t.Errorf("expected empty map on parse error, got %v", got)
	}
}

func TestParseSessionEnvVars_EmptyObject(t *testing.T) {
	got := parseSessionEnvVars("{}")
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}
