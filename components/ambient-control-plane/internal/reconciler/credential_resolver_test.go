package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/rs/zerolog"
)

// mockAPIServer creates a test HTTP server that returns canned responses
// for role binding list and credential get endpoints.
type mockData struct {
	roleBindings []types.RoleBinding
	credentials  map[string]types.Credential // keyed by ID
}

func newMockAPIServer(data mockData) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		page := r.URL.Query().Get("page")

		if strings.HasPrefix(r.URL.Path, "/api/ambient/v1/role_bindings") && r.Method == "GET" {
			search := r.URL.Query().Get("search")
			filtered := filterBindings(data.roleBindings, search)
			// Only return items on page 1; subsequent pages return empty to stop pagination
			items := filtered
			if page != "" && page != "1" {
				items = nil
			}
			resp := map[string]interface{}{
				"kind":  "RoleBindingList",
				"page":  1,
				"size":  len(items),
				"total": len(filtered),
				"items": items,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/api/ambient/v1/credentials/") && r.Method == "GET" {
			parts := strings.Split(r.URL.Path, "/")
			credID := parts[len(parts)-1]
			if cred, ok := data.credentials[credID]; ok {
				json.NewEncoder(w).Encode(cred)
				return
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
			return
		}

		if strings.HasPrefix(r.URL.Path, "/api/ambient/v1/credentials") && r.Method == "GET" {
			var creds []types.Credential
			for _, c := range data.credentials {
				creds = append(creds, c)
			}
			items := creds
			if page != "" && page != "1" {
				items = nil
			}
			resp := map[string]interface{}{
				"kind":  "CredentialList",
				"page":  1,
				"size":  len(items),
				"total": len(creds),
				"items": items,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

// filterBindings does a basic TSL-like filter matching for test purposes.
func filterBindings(bindings []types.RoleBinding, search string) []types.RoleBinding {
	if search == "" {
		return bindings
	}
	var result []types.RoleBinding
	for _, b := range bindings {
		if matchesSearch(b, search) {
			result = append(result, b)
		}
	}
	return result
}

// matchesSearch does simplified TSL matching sufficient for test queries.
func matchesSearch(b types.RoleBinding, search string) bool {
	if strings.Contains(search, "scope = 'credential'") && b.Scope != "credential" {
		return false
	}
	for _, part := range strings.Split(search, " and ") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "project_id = '") {
			val := extractTSLValue(part)
			if b.ProjectID == nil || *b.ProjectID != val {
				return false
			}
		}
		if strings.HasPrefix(part, "agent_id = '") {
			val := extractTSLValue(part)
			if b.AgentID == nil || *b.AgentID != val {
				return false
			}
		}
	}
	return true
}

func extractTSLValue(part string) string {
	start := strings.Index(part, "'")
	end := strings.LastIndex(part, "'")
	if start >= 0 && end > start {
		return part[start+1 : end]
	}
	return ""
}

func strPtr(s string) *string        { return &s }
func timePtr(t time.Time) *time.Time { return &t }

func newTestReconciler(logger zerolog.Logger) *SimpleKubeReconciler {
	return &SimpleKubeReconciler{
		cfg:    KubeReconcilerConfig{},
		logger: logger,
	}
}

func newSDKClient(t *testing.T, serverURL string) *sdkclient.Client {
	t.Helper()
	c, err := sdkclient.NewClient(serverURL, "test-token-must-be-at-least-20-chars-long", "test-project")
	if err != nil {
		t.Fatalf("failed to create SDK client: %v", err)
	}
	return c
}

func TestResolveCredentialIDs_AgentLevelOverridesProject(t *testing.T) {
	credA := "cred-a"
	credB := "cred-b"
	projectID := "proj-1"
	agentID := "agent-1"

	server := newMockAPIServer(mockData{
		roleBindings: []types.RoleBinding{
			{
				ObjectReference: types.ObjectReference{ID: "rb-1", CreatedAt: timePtr(time.Now().Add(-2 * time.Hour))},
				Scope:           "credential",
				CredentialID:    &credA,
				ProjectID:       &projectID,
			},
			{
				ObjectReference: types.ObjectReference{ID: "rb-2", CreatedAt: timePtr(time.Now().Add(-1 * time.Hour))},
				Scope:           "credential",
				CredentialID:    &credB,
				ProjectID:       &projectID,
				AgentID:         &agentID,
			},
		},
		credentials: map[string]types.Credential{
			credA: {ObjectReference: types.ObjectReference{ID: credA}, Provider: "github", Name: "cred-a"},
			credB: {ObjectReference: types.ObjectReference{ID: credB}, Provider: "github", Name: "cred-b"},
		},
	})
	defer server.Close()

	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := newTestReconciler(logger)
	sdk := newSDKClient(t, server.URL)

	result, err := r.resolveCredentialIDs(context.Background(), sdk, projectID, agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["github"] != credB {
		t.Errorf("expected agent-level credential %s, got %s", credB, result["github"])
	}
}

func TestResolveCredentialIDs_ProjectLevelFallback(t *testing.T) {
	credA := "cred-a"
	projectID := "proj-1"
	agentID := "agent-1"

	server := newMockAPIServer(mockData{
		roleBindings: []types.RoleBinding{
			{
				ObjectReference: types.ObjectReference{ID: "rb-1", CreatedAt: timePtr(time.Now())},
				Scope:           "credential",
				CredentialID:    &credA,
				ProjectID:       &projectID,
				// AgentID is nil → project-level binding
			},
		},
		credentials: map[string]types.Credential{
			credA: {ObjectReference: types.ObjectReference{ID: credA}, Provider: "github", Name: "cred-a"},
		},
	})
	defer server.Close()

	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := newTestReconciler(logger)
	sdk := newSDKClient(t, server.URL)

	result, err := r.resolveCredentialIDs(context.Background(), sdk, projectID, agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["github"] != credA {
		t.Errorf("expected project-level credential %s, got %s", credA, result["github"])
	}
}

func TestResolveCredentialIDs_GlobalFallback(t *testing.T) {
	credA := "cred-global"

	server := newMockAPIServer(mockData{
		roleBindings: []types.RoleBinding{
			{
				ObjectReference: types.ObjectReference{ID: "rb-global", CreatedAt: timePtr(time.Now())},
				Scope:           "credential",
				CredentialID:    &credA,
				// ProjectID and AgentID nil → global
			},
		},
		credentials: map[string]types.Credential{
			credA: {ObjectReference: types.ObjectReference{ID: credA}, Provider: "jira", Name: "global-jira"},
		},
	})
	defer server.Close()

	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := newTestReconciler(logger)
	sdk := newSDKClient(t, server.URL)

	result, err := r.resolveCredentialIDs(context.Background(), sdk, "some-project", "some-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["jira"] != credA {
		t.Errorf("expected global credential %s, got %s", credA, result["jira"])
	}
}

func TestResolveCredentialIDs_NoBindingNoInjection(t *testing.T) {
	server := newMockAPIServer(mockData{
		roleBindings: []types.RoleBinding{},
		credentials:  map[string]types.Credential{},
	})
	defer server.Close()

	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := newTestReconciler(logger)
	sdk := newSDKClient(t, server.URL)

	result, err := r.resolveCredentialIDs(context.Background(), sdk, "no-bindings-project", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestResolveCredentialIDs_MultipleProviders(t *testing.T) {
	credGH := "cred-github"
	credJira := "cred-jira"
	projectID := "proj-multi"
	agentID := "agent-multi"

	server := newMockAPIServer(mockData{
		roleBindings: []types.RoleBinding{
			{
				ObjectReference: types.ObjectReference{ID: "rb-gh", CreatedAt: timePtr(time.Now())},
				Scope:           "credential",
				CredentialID:    &credGH,
				ProjectID:       &projectID,
			},
			{
				ObjectReference: types.ObjectReference{ID: "rb-jira", CreatedAt: timePtr(time.Now())},
				Scope:           "credential",
				CredentialID:    &credJira,
				ProjectID:       &projectID,
				AgentID:         &agentID,
			},
		},
		credentials: map[string]types.Credential{
			credGH:   {ObjectReference: types.ObjectReference{ID: credGH}, Provider: "github", Name: "gh"},
			credJira: {ObjectReference: types.ObjectReference{ID: credJira}, Provider: "jira", Name: "jira"},
		},
	})
	defer server.Close()

	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := newTestReconciler(logger)
	sdk := newSDKClient(t, server.URL)

	result, err := r.resolveCredentialIDs(context.Background(), sdk, projectID, agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["github"] != credGH {
		t.Errorf("expected github=%s, got %s", credGH, result["github"])
	}
	if result["jira"] != credJira {
		t.Errorf("expected jira=%s, got %s", credJira, result["jira"])
	}
	if len(result) != 2 {
		t.Errorf("expected 2 providers, got %d: %v", len(result), result)
	}
}

func TestResolveCredentialIDs_DuplicateSameScopeEarliestWins(t *testing.T) {
	credOld := "cred-old"
	credNew := "cred-new"
	projectID := "proj-dup"

	server := newMockAPIServer(mockData{
		roleBindings: []types.RoleBinding{
			{
				ObjectReference: types.ObjectReference{ID: "rb-new", CreatedAt: timePtr(time.Now())},
				Scope:           "credential",
				CredentialID:    &credNew,
				ProjectID:       &projectID,
			},
			{
				ObjectReference: types.ObjectReference{ID: "rb-old", CreatedAt: timePtr(time.Now().Add(-24 * time.Hour))},
				Scope:           "credential",
				CredentialID:    &credOld,
				ProjectID:       &projectID,
			},
		},
		credentials: map[string]types.Credential{
			credOld: {ObjectReference: types.ObjectReference{ID: credOld}, Provider: "github", Name: "old-gh"},
			credNew: {ObjectReference: types.ObjectReference{ID: credNew}, Provider: "github", Name: "new-gh"},
		},
	})
	defer server.Close()

	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := newTestReconciler(logger)
	sdk := newSDKClient(t, server.URL)

	result, err := r.resolveCredentialIDs(context.Background(), sdk, projectID, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["github"] != credOld {
		t.Errorf("expected earliest credential %s to win, got %s", credOld, result["github"])
	}
}

func TestResolveCredentialIDs_AgentOverridesGlobal(t *testing.T) {
	credGlobal := "cred-global"
	credAgent := "cred-agent"
	projectID := "proj-1"
	agentID := "agent-1"

	server := newMockAPIServer(mockData{
		roleBindings: []types.RoleBinding{
			{
				ObjectReference: types.ObjectReference{ID: "rb-global", CreatedAt: timePtr(time.Now())},
				Scope:           "credential",
				CredentialID:    &credGlobal,
			},
			{
				ObjectReference: types.ObjectReference{ID: "rb-agent", CreatedAt: timePtr(time.Now())},
				Scope:           "credential",
				CredentialID:    &credAgent,
				ProjectID:       &projectID,
				AgentID:         &agentID,
			},
		},
		credentials: map[string]types.Credential{
			credGlobal: {ObjectReference: types.ObjectReference{ID: credGlobal}, Provider: "github", Name: "global"},
			credAgent:  {ObjectReference: types.ObjectReference{ID: credAgent}, Provider: "github", Name: "agent"},
		},
	})
	defer server.Close()

	logger := zerolog.New(zerolog.NewTestWriter(t))
	r := newTestReconciler(logger)
	sdk := newSDKClient(t, server.URL)

	result, err := r.resolveCredentialIDs(context.Background(), sdk, projectID, agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["github"] != credAgent {
		t.Errorf("expected agent-level %s to override global, got %s", credAgent, result["github"])
	}
}

// Suppress unused import warnings
var _ = fmt.Sprintf
