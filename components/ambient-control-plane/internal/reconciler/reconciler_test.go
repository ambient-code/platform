package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	openapi "github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient/platform/components/ambient-control-plane/internal/informer"
	"github.com/ambient/platform/components/ambient-control-plane/internal/kubeclient"
	"github.com/rs/zerolog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	clienttesting "k8s.io/client-go/testing"
)

func strPtr(s string) *string { return &s }

func newTestSessionReconciler() *SessionReconciler {
	return &SessionReconciler{
		logger: zerolog.Nop(),
	}
}

func buildCR(name string, spec map[string]interface{}) *unstructured.Unstructured {
	obj := map[string]interface{}{
		"apiVersion": "vteam.ambient-code/v1alpha1",
		"kind":       "AgenticSession",
		"metadata":   map[string]interface{}{"name": name},
	}
	if spec != nil {
		obj["spec"] = spec
	}
	return &unstructured.Unstructured{Object: obj}
}

func findDiff(diffs []FieldDiff, field string) *FieldDiff {
	for i := range diffs {
		if diffs[i].Field == field {
			return &diffs[i]
		}
	}
	return nil
}

func TestCompareSessionToCR_NameMatch(t *testing.T) {
	r := newTestSessionReconciler()
	session := *openapi.NewSession("test-session")
	session.SetId("sess-001")
	cr := buildCR("test-session", map[string]interface{}{
		"displayName": "test-session",
	})

	diffs := r.compareSessionToCR(session, cr)
	if d := findDiff(diffs, "name"); d != nil {
		t.Errorf("expected no 'name' diff when names match, got %+v", d)
	}
}

func TestCompareSessionToCR_NameMismatch(t *testing.T) {
	r := newTestSessionReconciler()
	session := *openapi.NewSession("api-name")
	session.SetId("sess-001")
	cr := buildCR("k8s-name", map[string]interface{}{
		"displayName": "api-name",
	})

	diffs := r.compareSessionToCR(session, cr)
	d := findDiff(diffs, "name")
	if d == nil {
		t.Fatal("expected 'name' diff when names differ")
	}
	if d.Category != "identity" {
		t.Errorf("expected category 'identity', got %q", d.Category)
	}
	if d.APIValue != "api-name" {
		t.Errorf("expected APIValue 'api-name', got %q", d.APIValue)
	}
	if d.K8sValue != "k8s-name" {
		t.Errorf("expected K8sValue 'k8s-name', got %q", d.K8sValue)
	}
}

func TestCompareSessionToCR_DisplayNameMapping(t *testing.T) {
	r := newTestSessionReconciler()
	session := *openapi.NewSession("My Session")
	session.SetId("sess-001")
	cr := buildCR("my-session", map[string]interface{}{
		"displayName": "Different Display Name",
	})

	diffs := r.compareSessionToCR(session, cr)
	d := findDiff(diffs, "name↔displayName")
	if d == nil {
		t.Fatal("expected 'name↔displayName' diff when display names differ")
	}
	if d.Category != "field-mapping" {
		t.Errorf("expected category 'field-mapping', got %q", d.Category)
	}
}

func TestCompareSessionToCR_PromptMapping(t *testing.T) {
	tests := []struct {
		name      string
		apiPrompt *string
		crPrompt  string
		wantDiff  bool
	}{
		{
			name:      "matching prompts",
			apiPrompt: strPtr("analyze this"),
			crPrompt:  "analyze this",
			wantDiff:  false,
		},
		{
			name:      "different prompts",
			apiPrompt: strPtr("analyze this"),
			crPrompt:  "do something else",
			wantDiff:  true,
		},
		{
			name:      "nil api prompt vs empty cr prompt",
			apiPrompt: nil,
			crPrompt:  "",
			wantDiff:  false,
		},
		{
			name:      "nil api prompt vs non-empty cr prompt",
			apiPrompt: nil,
			crPrompt:  "some prompt",
			wantDiff:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTestSessionReconciler()
			session := *openapi.NewSession("test")
			session.SetId("sess-001")
			session.Prompt = tt.apiPrompt
			cr := buildCR("test", map[string]interface{}{
				"displayName":   "test",
				"initialPrompt": tt.crPrompt,
			})

			diffs := r.compareSessionToCR(session, cr)
			d := findDiff(diffs, "prompt↔initialPrompt")
			if tt.wantDiff && d == nil {
				t.Error("expected prompt diff but got none")
			}
			if !tt.wantDiff && d != nil {
				t.Errorf("expected no prompt diff but got %+v", d)
			}
		})
	}
}

func TestCompareSessionToCR_RepoURLContained(t *testing.T) {
	r := newTestSessionReconciler()
	session := *openapi.NewSession("test")
	session.SetId("sess-001")
	session.SetRepoUrl("https://github.com/foo/bar")
	cr := buildCR("test", map[string]interface{}{
		"displayName": "test",
		"repos": []interface{}{
			map[string]interface{}{"url": "https://github.com/foo/bar"},
			map[string]interface{}{"url": "https://github.com/baz/qux"},
		},
	})

	diffs := r.compareSessionToCR(session, cr)
	if d := findDiff(diffs, "repo_url↔repos"); d != nil {
		t.Errorf("expected no repo diff when URL is in repos list, got %+v", d)
	}
}

func TestCompareSessionToCR_RepoURLMissing(t *testing.T) {
	r := newTestSessionReconciler()
	session := *openapi.NewSession("test")
	session.SetId("sess-001")
	session.SetRepoUrl("https://github.com/missing/repo")
	cr := buildCR("test", map[string]interface{}{
		"displayName": "test",
		"repos": []interface{}{
			map[string]interface{}{"url": "https://github.com/other/repo"},
		},
	})

	diffs := r.compareSessionToCR(session, cr)
	d := findDiff(diffs, "repo_url↔repos")
	if d == nil {
		t.Fatal("expected repo diff when URL is not in repos list")
	}
	if d.Category != "structural" {
		t.Errorf("expected category 'structural', got %q", d.Category)
	}
}

func TestCompareSessionToCR_WorkflowIDMapping(t *testing.T) {
	r := newTestSessionReconciler()
	session := *openapi.NewSession("test")
	session.SetId("sess-001")
	session.SetWorkflowId("wf-123")
	cr := buildCR("test", map[string]interface{}{
		"displayName": "test",
		"activeWorkflow": map[string]interface{}{
			"gitUrl": "https://github.com/workflows/wf-123",
		},
	})

	diffs := r.compareSessionToCR(session, cr)
	d := findDiff(diffs, "workflow_id↔activeWorkflow")
	if d == nil {
		t.Fatal("expected workflow diff (always reports mapping)")
	}
	if d.Category != "field-mapping" {
		t.Errorf("expected category 'field-mapping', got %q", d.Category)
	}
}

func TestCompareSessionToCR_WorkflowIDNoK8sEquiv(t *testing.T) {
	r := newTestSessionReconciler()
	session := *openapi.NewSession("test")
	session.SetId("sess-001")
	session.SetWorkflowId("wf-123")
	cr := buildCR("test", map[string]interface{}{
		"displayName": "test",
	})

	diffs := r.compareSessionToCR(session, cr)
	d := findDiff(diffs, "workflow_id↔activeWorkflow")
	if d == nil {
		t.Fatal("expected workflow diff when K8s has no activeWorkflow")
	}
	if d.K8sValue != "(not set)" {
		t.Errorf("expected K8sValue '(not set)', got %q", d.K8sValue)
	}
}

func TestFindAPIOnlyFields(t *testing.T) {
	r := newTestSessionReconciler()
	session := *openapi.NewSession("test")
	session.SetId("sess-001")
	session.SetCreatedByUserId("user-abc")
	session.SetAssignedUserId("user-xyz")

	diffs := r.findAPIOnlyFields(session)

	fieldNames := map[string]bool{}
	for _, d := range diffs {
		fieldNames[d.Field] = true
		if d.Category != "api-only" && d.Category != "identity-mapping" {
			t.Errorf("expected category 'api-only' or 'identity-mapping' for field %q, got %q", d.Field, d.Category)
		}
	}

	for _, expected := range []string{"created_by_user_id", "assigned_user_id", "id"} {
		if !fieldNames[expected] {
			t.Errorf("expected field %q in API-only diffs", expected)
		}
	}
}

func TestFindK8sOnlyFields(t *testing.T) {
	r := newTestSessionReconciler()
	cr := buildCR("test", map[string]interface{}{
		"interactive": true,
		"llmSettings": map[string]interface{}{
			"model":       "claude-sonnet-4-20250514",
			"temperature": float64(0.7),
			"maxTokens":   float64(4096),
		},
		"timeout": int64(3600),
		"userContext": map[string]interface{}{
			"userId":      "user-1",
			"displayName": "Test User",
			"groups":      []interface{}{"admin"},
		},
		"resourceOverrides": map[string]interface{}{
			"cpu":           "2",
			"memory":        "4Gi",
			"storageClass":  "gp3",
			"priorityClass": "high",
		},
		"environmentVariables": map[string]interface{}{
			"FOO": "bar",
		},
		"project": "my-project",
	})
	cr.Object["status"] = map[string]interface{}{
		"phase": "Running",
	}

	diffs := r.findK8sOnlyFields(cr)

	expectedFields := []string{
		"spec.interactive",
		"spec.llmSettings",
		"spec.timeout",
		"spec.userContext",
		"spec.resourceOverrides",
		"spec.environmentVariables",
		"spec.project",
		"status",
	}

	fieldNames := map[string]bool{}
	for _, d := range diffs {
		fieldNames[d.Field] = true
		if d.Category != "k8s-only" {
			t.Errorf("expected category 'k8s-only' for field %q, got %q", d.Field, d.Category)
		}
	}

	for _, expected := range expectedFields {
		if !fieldNames[expected] {
			t.Errorf("expected field %q in K8s-only diffs", expected)
		}
	}

	if len(diffs) != len(expectedFields) {
		t.Errorf("expected %d K8s-only diffs, got %d", len(expectedFields), len(diffs))
	}
}

func TestPtrStr_NilAndNonNil(t *testing.T) {
	tests := []struct {
		name string
		in   *string
		want string
	}{
		{"nil", nil, ""},
		{"empty", strPtr(""), ""},
		{"non-empty", strPtr("hello"), "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ptrStr(tt.in); got != tt.want {
				t.Errorf("ptrStr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTruncate_ShortAndLong(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"shorter than max", "abc", 5, "abc"},
		{"equal to max", "abcde", 5, "abcde"},
		{"longer than max", "abcdef", 3, "abc..."},
		{"empty string", "", 5, ""},
		{"max zero", "abc", 0, "..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncate(tt.s, tt.max); got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}

func TestExtractRepoURLs(t *testing.T) {
	tests := []struct {
		name string
		in   []interface{}
		want []string
	}{
		{
			name: "valid repos",
			in: []interface{}{
				map[string]interface{}{"url": "https://github.com/a/b"},
				map[string]interface{}{"url": "https://github.com/c/d"},
			},
			want: []string{"https://github.com/a/b", "https://github.com/c/d"},
		},
		{
			name: "mixed valid and invalid",
			in: []interface{}{
				map[string]interface{}{"url": "https://github.com/a/b"},
				map[string]interface{}{"not-url": "something"},
				"not a map",
				map[string]interface{}{"url": 42},
			},
			want: []string{"https://github.com/a/b"},
		},
		{
			name: "nil input",
			in:   nil,
			want: nil,
		},
		{
			name: "empty input",
			in:   []interface{}{},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRepoURLs(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("extractRepoURLs() returned %d URLs, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractRepoURLs()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestContainsString_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		s     string
		want  bool
	}{
		{"exact match", []string{"a", "b", "c"}, "b", true},
		{"case insensitive", []string{"HTTPS://GITHUB.COM/FOO/BAR"}, "https://github.com/foo/bar", true},
		{"not found", []string{"a", "b"}, "c", false},
		{"empty slice", []string{}, "a", false},
		{"nil slice", nil, "a", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsString(tt.slice, tt.s); got != tt.want {
				t.Errorf("containsString(%v, %q) = %v, want %v", tt.slice, tt.s, got, tt.want)
			}
		})
	}
}

func float64Ptr(f float64) *float64 { return &f }
func int32Ptr(i int32) *int32       { return &i }
func boolPtr(b bool) *bool          { return &b }

func newFakeKubeClient(namespace string, objects ...runtime.Object) *kubeclient.KubeClient {
	sch := runtime.NewScheme()
	sch.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "vteam.ambient-code", Version: "v1alpha1", Kind: "AgenticSession"},
		&unstructured.Unstructured{},
	)
	sch.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "vteam.ambient-code", Version: "v1alpha1", Kind: "AgenticSessionList"},
		&unstructured.UnstructuredList{},
	)
	fakeClient := dynamicfake.NewSimpleDynamicClient(sch, objects...)
	return kubeclient.NewFromDynamic(fakeClient, namespace, zerolog.Nop())
}

func TestSessionToUnstructured_BasicFields(t *testing.T) {
	session := *openapi.NewSession("my-session")
	session.SetId("ksuid-001")
	session.SetKubeCrName("ksuid-001")
	session.SetPrompt("analyze the code")
	session.SetInteractive(true)
	session.SetTimeout(600)
	session.SetProjectId("proj-1")
	session.SetCreatedByUserId("user-abc")

	cr := SessionToUnstructured(session, "ambient-code")

	if cr.GetName() != "ksuid-001" {
		t.Errorf("expected name 'ksuid-001', got %q", cr.GetName())
	}
	if cr.GetNamespace() != "ambient-code" {
		t.Errorf("expected namespace 'ambient-code', got %q", cr.GetNamespace())
	}
	if cr.GetKind() != "AgenticSession" {
		t.Errorf("expected kind 'AgenticSession', got %q", cr.GetKind())
	}

	displayName, _, _ := unstructured.NestedString(cr.Object, "spec", "displayName")
	if displayName != "my-session" {
		t.Errorf("expected displayName 'my-session', got %q", displayName)
	}

	prompt, _, _ := unstructured.NestedString(cr.Object, "spec", "initialPrompt")
	if prompt != "analyze the code" {
		t.Errorf("expected prompt 'analyze the code', got %q", prompt)
	}

	interactive, found, _ := unstructured.NestedBool(cr.Object, "spec", "interactive")
	if !found || !interactive {
		t.Error("expected spec.interactive to be true")
	}

	timeout, found, _ := unstructured.NestedInt64(cr.Object, "spec", "timeout")
	if !found || timeout != 600 {
		t.Errorf("expected timeout 600, got %d (found=%v)", timeout, found)
	}

	project, _, _ := unstructured.NestedString(cr.Object, "spec", "project")
	if project != "proj-1" {
		t.Errorf("expected project 'proj-1', got %q", project)
	}

	userId, _, _ := unstructured.NestedString(cr.Object, "spec", "userContext", "userId")
	if userId != "user-abc" {
		t.Errorf("expected userContext.userId 'user-abc', got %q", userId)
	}
}

func TestSessionToUnstructured_LLMSettings(t *testing.T) {
	session := *openapi.NewSession("llm-test")
	session.SetId("ksuid-002")
	session.SetKubeCrName("ksuid-002")
	session.LlmModel = strPtr("claude-3-7-sonnet")
	session.LlmTemperature = float64Ptr(0.5)
	session.LlmMaxTokens = int32Ptr(8000)

	cr := SessionToUnstructured(session, "test-ns")

	model, _, _ := unstructured.NestedString(cr.Object, "spec", "llmSettings", "model")
	if model != "claude-3-7-sonnet" {
		t.Errorf("expected model 'claude-3-7-sonnet', got %q", model)
	}

	temp, found, _ := unstructured.NestedFloat64(cr.Object, "spec", "llmSettings", "temperature")
	if !found || temp != 0.5 {
		t.Errorf("expected temperature 0.5, got %f", temp)
	}

	maxTokens, found, _ := unstructured.NestedInt64(cr.Object, "spec", "llmSettings", "maxTokens")
	if !found || maxTokens != 8000 {
		t.Errorf("expected maxTokens 8000, got %d", maxTokens)
	}
}

func TestSessionToUnstructured_ReposFromJSON(t *testing.T) {
	session := *openapi.NewSession("repos-test")
	session.SetId("ksuid-003")
	session.SetKubeCrName("ksuid-003")
	reposJSON := `[{"url":"https://github.com/foo/bar","branch":"develop"},{"url":"https://github.com/baz/qux"}]`
	session.SetRepos(reposJSON)

	cr := SessionToUnstructured(session, "test-ns")

	repos, found, _ := unstructured.NestedSlice(cr.Object, "spec", "repos")
	if !found || len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d (found=%v)", len(repos), found)
	}

	firstRepo, ok := repos[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected repo to be a map")
	}
	if firstRepo["url"] != "https://github.com/foo/bar" {
		t.Errorf("expected first repo URL 'https://github.com/foo/bar', got %v", firstRepo["url"])
	}
	if firstRepo["branch"] != "develop" {
		t.Errorf("expected first repo branch 'develop', got %v", firstRepo["branch"])
	}
}

func TestSessionToUnstructured_LegacyRepoURL(t *testing.T) {
	session := *openapi.NewSession("legacy-repo")
	session.SetId("ksuid-004")
	session.SetKubeCrName("ksuid-004")
	session.SetRepoUrl("https://github.com/legacy/repo")

	cr := SessionToUnstructured(session, "test-ns")

	repos, found, _ := unstructured.NestedSlice(cr.Object, "spec", "repos")
	if !found || len(repos) != 1 {
		t.Fatalf("expected 1 repo from legacy URL, got %d (found=%v)", len(repos), found)
	}

	repo, ok := repos[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected repo to be a map")
	}
	if repo["url"] != "https://github.com/legacy/repo" {
		t.Errorf("expected URL 'https://github.com/legacy/repo', got %v", repo["url"])
	}
	if repo["branch"] != "ambient/ksuid-004" {
		t.Errorf("expected auto-generated branch 'ambient/ksuid-004', got %v", repo["branch"])
	}
}

func TestSessionToUnstructured_BotAccount(t *testing.T) {
	session := *openapi.NewSession("bot-test")
	session.SetId("ksuid-005")
	session.SetKubeCrName("ksuid-005")
	session.SetBotAccountName("deploy-bot")

	cr := SessionToUnstructured(session, "test-ns")

	botName, _, _ := unstructured.NestedString(cr.Object, "spec", "botAccount", "name")
	if botName != "deploy-bot" {
		t.Errorf("expected bot name 'deploy-bot', got %q", botName)
	}
}

func TestSessionToUnstructured_ResourceOverrides(t *testing.T) {
	session := *openapi.NewSession("overrides-test")
	session.SetId("ksuid-006")
	session.SetKubeCrName("ksuid-006")
	overridesJSON := `{"cpu":"4","memory":"8Gi","storageClass":"gp3"}`
	session.SetResourceOverrides(overridesJSON)

	cr := SessionToUnstructured(session, "test-ns")

	cpu, _, _ := unstructured.NestedString(cr.Object, "spec", "resourceOverrides", "cpu")
	if cpu != "4" {
		t.Errorf("expected cpu '4', got %q", cpu)
	}
	mem, _, _ := unstructured.NestedString(cr.Object, "spec", "resourceOverrides", "memory")
	if mem != "8Gi" {
		t.Errorf("expected memory '8Gi', got %q", mem)
	}
}

func TestSessionToUnstructured_EnvironmentVariables(t *testing.T) {
	session := *openapi.NewSession("env-test")
	session.SetId("ksuid-007")
	session.SetKubeCrName("ksuid-007")
	envJSON := `{"API_KEY":"redacted","DEBUG":"true"}`
	session.SetEnvironmentVariables(envJSON)

	cr := SessionToUnstructured(session, "test-ns")

	envMap, found, _ := unstructured.NestedMap(cr.Object, "spec", "environmentVariables")
	if !found {
		t.Fatal("expected environmentVariables to be set")
	}
	if envMap["DEBUG"] != "true" {
		t.Errorf("expected DEBUG='true', got %v", envMap["DEBUG"])
	}
}

func TestSessionToUnstructured_LabelsAndAnnotations(t *testing.T) {
	session := *openapi.NewSession("labels-test")
	session.SetId("ksuid-008")
	session.SetKubeCrName("ksuid-008")
	session.SetLabels(`{"team":"platform","env":"dev"}`)
	session.SetAnnotations(`{"note":"test session"}`)

	cr := SessionToUnstructured(session, "test-ns")

	labels := cr.GetLabels()
	if labels["team"] != "platform" {
		t.Errorf("expected label team=platform, got %q", labels["team"])
	}
	if labels["env"] != "dev" {
		t.Errorf("expected label env=dev, got %q", labels["env"])
	}

	annotations := cr.GetAnnotations()
	if annotations["note"] != "test session" {
		t.Errorf("expected annotation note='test session', got %q", annotations["note"])
	}
}

func TestSessionToUnstructured_FallbackToID(t *testing.T) {
	session := *openapi.NewSession("fallback-test")
	session.SetId("ksuid-fallback")

	cr := SessionToUnstructured(session, "test-ns")
	if cr.GetName() != "ksuid-fallback" {
		t.Errorf("expected CR name to fall back to session ID, got %q", cr.GetName())
	}
}

func TestSessionToUnstructured_MinimalSession(t *testing.T) {
	session := *openapi.NewSession("minimal")
	session.SetId("ksuid-min")

	cr := SessionToUnstructured(session, "test-ns")

	spec, found, _ := unstructured.NestedMap(cr.Object, "spec")
	if !found {
		t.Fatal("expected spec to exist")
	}
	if spec["displayName"] != "minimal" {
		t.Errorf("expected displayName 'minimal', got %v", spec["displayName"])
	}
	if _, hasPrompt := spec["initialPrompt"]; hasPrompt {
		t.Error("expected no initialPrompt for minimal session")
	}
	if _, hasLLM := spec["llmSettings"]; hasLLM {
		t.Error("expected no llmSettings for minimal session")
	}
}

func TestBuildSpec_JSONRoundTrip(t *testing.T) {
	session := *openapi.NewSession("json-test")
	session.SetId("ksuid-json")
	session.SetPrompt("do stuff")
	session.SetInteractive(false)
	session.SetTimeout(300)
	session.LlmModel = strPtr("claude-3-7-sonnet")
	session.LlmTemperature = float64Ptr(0.7)
	session.LlmMaxTokens = int32Ptr(4000)

	spec := buildSpec(session)

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}

	var roundTripped map[string]interface{}
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	if roundTripped["displayName"] != "json-test" {
		t.Errorf("expected displayName 'json-test', got %v", roundTripped["displayName"])
	}
	if roundTripped["initialPrompt"] != "do stuff" {
		t.Errorf("expected initialPrompt 'do stuff', got %v", roundTripped["initialPrompt"])
	}
}

func TestReconcile_AddedEvent_CreatesCR(t *testing.T) {
	kube := newFakeKubeClient("test-ns")
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("new-session")
	session.SetId("ksuid-new")
	session.SetKubeCrName("ksuid-new")
	session.SetPrompt("create this")

	event := informer.ResourceEvent{
		Type:     informer.EventAdded,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	cr, err := kube.GetAgenticSession(context.Background(), "ksuid-new")
	if err != nil {
		t.Fatalf("CR not found after ADDED event: %v", err)
	}

	displayName, _, _ := unstructured.NestedString(cr.Object, "spec", "displayName")
	if displayName != "new-session" {
		t.Errorf("expected displayName 'new-session', got %q", displayName)
	}

	prompt, _, _ := unstructured.NestedString(cr.Object, "spec", "initialPrompt")
	if prompt != "create this" {
		t.Errorf("expected prompt 'create this', got %q", prompt)
	}
}

func TestReconcile_ModifiedEvent_UpdatesCR(t *testing.T) {
	existingCR := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "ksuid-mod",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"displayName":   "old-name",
				"initialPrompt": "old prompt",
			},
		},
	}

	kube := newFakeKubeClient("test-ns", existingCR)
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("updated-name")
	session.SetId("ksuid-mod")
	session.SetKubeCrName("ksuid-mod")
	session.SetPrompt("new prompt")
	session.SetTimeout(900)

	event := informer.ResourceEvent{
		Type:     informer.EventModified,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	cr, err := kube.GetAgenticSession(context.Background(), "ksuid-mod")
	if err != nil {
		t.Fatalf("CR not found: %v", err)
	}

	displayName, _, _ := unstructured.NestedString(cr.Object, "spec", "displayName")
	if displayName != "updated-name" {
		t.Errorf("expected displayName 'updated-name', got %q", displayName)
	}

	prompt, _, _ := unstructured.NestedString(cr.Object, "spec", "initialPrompt")
	if prompt != "new prompt" {
		t.Errorf("expected prompt 'new prompt', got %q", prompt)
	}

	timeout, _, _ := unstructured.NestedInt64(cr.Object, "spec", "timeout")
	if timeout != 900 {
		t.Errorf("expected timeout 900, got %d", timeout)
	}
}

func TestReconcile_DeletedEvent_RemovesCR(t *testing.T) {
	existingCR := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "ksuid-del",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"displayName": "to-delete",
			},
		},
	}

	kube := newFakeKubeClient("test-ns", existingCR)
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("to-delete")
	session.SetId("ksuid-del")
	session.SetKubeCrName("ksuid-del")

	event := informer.ResourceEvent{
		Type:     informer.EventDeleted,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	_, err = kube.GetAgenticSession(context.Background(), "ksuid-del")
	if err == nil {
		t.Fatal("expected CR to be deleted")
	}
}

func TestReconcile_DeletedEvent_AlreadyGone(t *testing.T) {
	kube := newFakeKubeClient("test-ns")
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("ghost")
	session.SetId("ksuid-ghost")
	session.SetKubeCrName("ksuid-ghost")

	event := informer.ResourceEvent{
		Type:     informer.EventDeleted,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("expected no error for already-gone CR, got: %v", err)
	}
}

func TestReconcile_AddedEvent_CRAlreadyExists(t *testing.T) {
	existingCR := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "ksuid-exists",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"displayName": "old",
			},
		},
	}

	kube := newFakeKubeClient("test-ns", existingCR)
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("updated-via-add")
	session.SetId("ksuid-exists")
	session.SetKubeCrName("ksuid-exists")
	session.SetPrompt("from add event")

	event := informer.ResourceEvent{
		Type:     informer.EventAdded,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	cr, _ := kube.GetAgenticSession(context.Background(), "ksuid-exists")
	displayName, _, _ := unstructured.NestedString(cr.Object, "spec", "displayName")
	if displayName != "updated-via-add" {
		t.Errorf("expected displayName 'updated-via-add', got %q", displayName)
	}
}

func TestReconcile_ModifiedEvent_CRMissing_Creates(t *testing.T) {
	kube := newFakeKubeClient("test-ns")
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("missing-cr")
	session.SetId("ksuid-missing")
	session.SetKubeCrName("ksuid-missing")

	event := informer.ResourceEvent{
		Type:     informer.EventModified,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	_, err = kube.GetAgenticSession(context.Background(), "ksuid-missing")
	if err != nil {
		t.Fatalf("expected CR to be created on MODIFIED when missing: %v", err)
	}
}

func TestReconcile_FullLifecycle_CreateUpdateDelete(t *testing.T) {
	kube := newFakeKubeClient("test-ns")
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("lifecycle")
	session.SetId("ksuid-life")
	session.SetKubeCrName("ksuid-life")
	session.SetPrompt("initial")

	err := r.Reconcile(context.Background(), informer.ResourceEvent{
		Type: informer.EventAdded, Resource: "sessions", Object: session,
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	cr, _ := kube.GetAgenticSession(context.Background(), "ksuid-life")
	prompt, _, _ := unstructured.NestedString(cr.Object, "spec", "initialPrompt")
	if prompt != "initial" {
		t.Errorf("expected prompt 'initial', got %q", prompt)
	}

	session.SetPrompt("updated")
	session.SetTimeout(1200)
	err = r.Reconcile(context.Background(), informer.ResourceEvent{
		Type: informer.EventModified, Resource: "sessions", Object: session,
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	cr, _ = kube.GetAgenticSession(context.Background(), "ksuid-life")
	prompt, _, _ = unstructured.NestedString(cr.Object, "spec", "initialPrompt")
	if prompt != "updated" {
		t.Errorf("expected prompt 'updated', got %q", prompt)
	}
	timeout, _, _ := unstructured.NestedInt64(cr.Object, "spec", "timeout")
	if timeout != 1200 {
		t.Errorf("expected timeout 1200, got %d", timeout)
	}

	err = r.Reconcile(context.Background(), informer.ResourceEvent{
		Type: informer.EventDeleted, Resource: "sessions", Object: session,
	})
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, err = kube.GetAgenticSession(context.Background(), "ksuid-life")
	if err == nil {
		t.Fatal("expected CR to be gone after delete")
	}
}

func TestCRStatusToStatusPatch_MetadataFields(t *testing.T) {
	cr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "test-cr",
				"namespace": "ambient-code",
				"uid":       "abc-def-123",
			},
		},
	}

	patch := CRStatusToStatusPatch(cr)

	uid, ok := patch.GetKubeCrUidOk()
	if !ok || *uid != "abc-def-123" {
		t.Errorf("expected kube_cr_uid 'abc-def-123', got %v", uid)
	}

	ns, ok := patch.GetKubeNamespaceOk()
	if !ok || *ns != "ambient-code" {
		t.Errorf("expected kube_namespace 'ambient-code', got %v", ns)
	}
}

func TestCRStatusToStatusPatch_PhaseAndTimestamps(t *testing.T) {
	startTime := "2026-02-15T10:00:00Z"
	completionTime := "2026-02-15T11:30:00Z"
	cr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "test-cr",
				"namespace": "test-ns",
				"uid":       "uid-phase",
			},
			"status": map[string]interface{}{
				"phase":          "Succeeded",
				"startTime":      startTime,
				"completionTime": completionTime,
			},
		},
	}

	patch := CRStatusToStatusPatch(cr)

	if patch.GetPhase() != "Succeeded" {
		t.Errorf("expected phase 'Succeeded', got %q", patch.GetPhase())
	}

	expectedStart, _ := time.Parse(time.RFC3339, startTime)
	if !patch.GetStartTime().Equal(expectedStart) {
		t.Errorf("expected start_time %v, got %v", expectedStart, patch.GetStartTime())
	}

	expectedCompletion, _ := time.Parse(time.RFC3339, completionTime)
	if !patch.GetCompletionTime().Equal(expectedCompletion) {
		t.Errorf("expected completion_time %v, got %v", expectedCompletion, patch.GetCompletionTime())
	}
}

func TestCRStatusToStatusPatch_SDKFields(t *testing.T) {
	cr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name": "test-cr",
				"uid":  "uid-sdk",
			},
			"status": map[string]interface{}{
				"sdkSessionId":    "sdk-sess-456",
				"sdkRestartCount": int64(3),
			},
		},
	}

	patch := CRStatusToStatusPatch(cr)

	if patch.GetSdkSessionId() != "sdk-sess-456" {
		t.Errorf("expected sdk_session_id 'sdk-sess-456', got %q", patch.GetSdkSessionId())
	}
	if patch.GetSdkRestartCount() != 3 {
		t.Errorf("expected sdk_restart_count 3, got %d", patch.GetSdkRestartCount())
	}
}

func TestCRStatusToStatusPatch_Conditions(t *testing.T) {
	cr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name": "test-cr",
				"uid":  "uid-cond",
			},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Ready",
						"status": "True",
						"reason": "AllGood",
					},
				},
			},
		},
	}

	patch := CRStatusToStatusPatch(cr)

	conditionsJSON := patch.GetConditions()
	if conditionsJSON == "" {
		t.Fatal("expected conditions to be set")
	}

	var conditions []map[string]interface{}
	if err := json.Unmarshal([]byte(conditionsJSON), &conditions); err != nil {
		t.Fatalf("conditions JSON unmarshal failed: %v", err)
	}
	if len(conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(conditions))
	}
	if conditions[0]["type"] != "Ready" {
		t.Errorf("expected condition type 'Ready', got %v", conditions[0]["type"])
	}
}

func TestCRStatusToStatusPatch_ReconciledRepos(t *testing.T) {
	cr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name": "test-cr",
				"uid":  "uid-repos",
			},
			"status": map[string]interface{}{
				"reconciledRepos": []interface{}{
					map[string]interface{}{
						"url":    "https://github.com/foo/bar",
						"commit": "abc123",
					},
				},
			},
		},
	}

	patch := CRStatusToStatusPatch(cr)

	reposJSON := patch.GetReconciledRepos()
	if reposJSON == "" {
		t.Fatal("expected reconciled_repos to be set")
	}

	var repos []map[string]interface{}
	if err := json.Unmarshal([]byte(reposJSON), &repos); err != nil {
		t.Fatalf("repos JSON unmarshal failed: %v", err)
	}
	if repos[0]["url"] != "https://github.com/foo/bar" {
		t.Errorf("expected repo URL 'https://github.com/foo/bar', got %v", repos[0]["url"])
	}
}

func TestCRStatusToStatusPatch_ReconciledWorkflow(t *testing.T) {
	cr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name": "test-cr",
				"uid":  "uid-wf",
			},
			"status": map[string]interface{}{
				"reconciledWorkflow": map[string]interface{}{
					"gitUrl": "https://github.com/wf/repo",
					"branch": "main",
					"path":   "workflows/deploy.yaml",
				},
			},
		},
	}

	patch := CRStatusToStatusPatch(cr)

	wfJSON := patch.GetReconciledWorkflow()
	if wfJSON == "" {
		t.Fatal("expected reconciled_workflow to be set")
	}

	var wf map[string]interface{}
	if err := json.Unmarshal([]byte(wfJSON), &wf); err != nil {
		t.Fatalf("workflow JSON unmarshal failed: %v", err)
	}
	if wf["gitUrl"] != "https://github.com/wf/repo" {
		t.Errorf("expected gitUrl 'https://github.com/wf/repo', got %v", wf["gitUrl"])
	}
}

func TestCRStatusToStatusPatch_NoStatus(t *testing.T) {
	cr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "test-cr",
				"namespace": "test-ns",
				"uid":       "uid-no-status",
			},
		},
	}

	patch := CRStatusToStatusPatch(cr)

	if patch.GetKubeCrUid() != "uid-no-status" {
		t.Errorf("expected uid 'uid-no-status', got %q", patch.GetKubeCrUid())
	}
	if patch.GetKubeNamespace() != "test-ns" {
		t.Errorf("expected namespace 'test-ns', got %q", patch.GetKubeNamespace())
	}
	if patch.HasPhase() {
		t.Error("expected no phase when status is absent")
	}
	if patch.HasStartTime() {
		t.Error("expected no start_time when status is absent")
	}
	if patch.HasConditions() {
		t.Error("expected no conditions when status is absent")
	}
}

func TestCRStatusToStatusPatch_FullStatus(t *testing.T) {
	cr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "full-status-cr",
				"namespace": "ambient-code",
				"uid":       "uid-full",
			},
			"status": map[string]interface{}{
				"phase":           "Running",
				"startTime":       "2026-02-15T08:00:00Z",
				"sdkSessionId":    "sdk-001",
				"sdkRestartCount": int64(1),
				"conditions": []interface{}{
					map[string]interface{}{"type": "Ready", "status": "True"},
				},
				"reconciledRepos": []interface{}{
					map[string]interface{}{"url": "https://github.com/test/repo"},
				},
				"reconciledWorkflow": map[string]interface{}{
					"gitUrl": "https://github.com/wf/test",
				},
			},
		},
	}

	patch := CRStatusToStatusPatch(cr)

	if patch.GetKubeCrUid() != "uid-full" {
		t.Errorf("uid mismatch: got %q", patch.GetKubeCrUid())
	}
	if patch.GetKubeNamespace() != "ambient-code" {
		t.Errorf("namespace mismatch: got %q", patch.GetKubeNamespace())
	}
	if patch.GetPhase() != "Running" {
		t.Errorf("phase mismatch: got %q", patch.GetPhase())
	}
	if !patch.HasStartTime() {
		t.Error("expected start_time to be set")
	}
	if patch.GetSdkSessionId() != "sdk-001" {
		t.Errorf("sdk_session_id mismatch: got %q", patch.GetSdkSessionId())
	}
	if patch.GetSdkRestartCount() != 1 {
		t.Errorf("sdk_restart_count mismatch: got %d", patch.GetSdkRestartCount())
	}
	if !patch.HasConditions() {
		t.Error("expected conditions to be set")
	}
	if !patch.HasReconciledRepos() {
		t.Error("expected reconciled_repos to be set")
	}
	if !patch.HasReconciledWorkflow() {
		t.Error("expected reconciled_workflow to be set")
	}
}

func TestWriteStatusToAPI_NilClientSkips(t *testing.T) {
	kube := newFakeKubeClient("test-ns")
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("nil-client-test")
	session.SetId("ksuid-nil-client")
	session.SetKubeCrName("ksuid-nil-client")
	session.SetPrompt("test")

	event := informer.ResourceEvent{
		Type:     informer.EventAdded,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile with nil client should succeed: %v", err)
	}

	cr, err := kube.GetAgenticSession(context.Background(), "ksuid-nil-client")
	if err != nil {
		t.Fatalf("CR should exist: %v", err)
	}
	if cr.GetName() != "ksuid-nil-client" {
		t.Errorf("expected CR name 'ksuid-nil-client', got %q", cr.GetName())
	}
}

func TestCRStatusToStatusPatch_EmptyUID(t *testing.T) {
	cr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name": "no-uid-cr",
			},
		},
	}

	patch := CRStatusToStatusPatch(cr)

	if patch.HasKubeCrUid() {
		t.Error("expected no kube_cr_uid when UID is empty")
	}
	if patch.HasKubeNamespace() {
		t.Error("expected no kube_namespace when namespace is empty")
	}
}

func TestCRStatusToStatusPatch_InvalidTimestampIgnored(t *testing.T) {
	cr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name": "bad-time-cr",
				"uid":  "uid-bad-time",
			},
			"status": map[string]interface{}{
				"phase":     "Running",
				"startTime": "not-a-valid-timestamp",
			},
		},
	}

	patch := CRStatusToStatusPatch(cr)

	if patch.GetPhase() != "Running" {
		t.Errorf("expected phase 'Running', got %q", patch.GetPhase())
	}
	if patch.HasStartTime() {
		t.Error("expected no start_time when timestamp is invalid")
	}
}

func TestCRStatusToStatusPatch_UIDFromTypedField(t *testing.T) {
	cr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "typed-uid-cr",
				"namespace": "prod-ns",
				"uid":       "typed-uid-value",
			},
		},
	}
	cr.SetUID(types.UID("explicitly-set-uid"))

	patch := CRStatusToStatusPatch(cr)

	if patch.GetKubeCrUid() != "explicitly-set-uid" {
		t.Errorf("expected uid 'explicitly-set-uid', got %q", patch.GetKubeCrUid())
	}
}

var agenticSessionGR = schema.GroupResource{
	Group:    "vteam.ambient-code",
	Resource: "agenticsessions",
}

func newFakeKubeClientWithReactors(namespace string, objects ...runtime.Object) (*dynamicfake.FakeDynamicClient, *kubeclient.KubeClient) {
	sch := runtime.NewScheme()
	sch.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "vteam.ambient-code", Version: "v1alpha1", Kind: "AgenticSession"},
		&unstructured.Unstructured{},
	)
	sch.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "vteam.ambient-code", Version: "v1alpha1", Kind: "AgenticSessionList"},
		&unstructured.UnstructuredList{},
	)
	fakeClient := dynamicfake.NewSimpleDynamicClient(sch, objects...)
	kube := kubeclient.NewFromDynamic(fakeClient, namespace, zerolog.Nop())
	return fakeClient, kube
}

func TestReconcile_AddedEvent_GetReturnsUnexpectedError(t *testing.T) {
	fake, kube := newFakeKubeClientWithReactors("test-ns")
	fake.PrependReactor("get", "agenticsessions",
		func(action clienttesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewInternalError(fmt.Errorf("etcd unavailable"))
		},
	)
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("get-error-test")
	session.SetId("ksuid-get-err")
	session.SetKubeCrName("ksuid-get-err")

	event := informer.ResourceEvent{
		Type:     informer.EventAdded,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err == nil {
		t.Fatal("expected error when Get returns InternalError")
	}
	if !apierrors.IsInternalError(err) {
		t.Errorf("expected InternalError, got: %v", err)
	}
}

func TestReconcile_AddedEvent_CreateReturnsAlreadyExists(t *testing.T) {
	fake, kube := newFakeKubeClientWithReactors("test-ns")
	fake.PrependReactor("create", "agenticsessions",
		func(action clienttesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewAlreadyExists(agenticSessionGR, "ksuid-already")
		},
	)
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("already-exists-test")
	session.SetId("ksuid-already")
	session.SetKubeCrName("ksuid-already")

	event := informer.ResourceEvent{
		Type:     informer.EventAdded,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err == nil {
		t.Fatal("expected error when Create returns AlreadyExists")
	}
	if !apierrors.IsAlreadyExists(err) {
		t.Errorf("expected AlreadyExists error, got: %v", err)
	}
}

func TestReconcile_ModifiedEvent_UpdateReturnsConflict(t *testing.T) {
	existingCR := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "ksuid-conflict",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"displayName": "old-name",
			},
		},
	}

	fake, kube := newFakeKubeClientWithReactors("test-ns", existingCR)
	fake.PrependReactor("update", "agenticsessions",
		func(action clienttesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewConflict(agenticSessionGR, "ksuid-conflict",
				fmt.Errorf("the object has been modified"))
		},
	)
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("conflict-test")
	session.SetId("ksuid-conflict")
	session.SetKubeCrName("ksuid-conflict")
	session.SetPrompt("updated prompt")

	event := informer.ResourceEvent{
		Type:     informer.EventModified,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err == nil {
		t.Fatal("expected error when Update returns Conflict")
	}
	if !apierrors.IsConflict(err) {
		t.Errorf("expected Conflict error, got: %v", err)
	}
}

func TestReconcile_ModifiedEvent_GetReturnsUnexpectedError(t *testing.T) {
	fake, kube := newFakeKubeClientWithReactors("test-ns")
	fake.PrependReactor("get", "agenticsessions",
		func(action clienttesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewInternalError(fmt.Errorf("storage backend down"))
		},
	)
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("mod-get-error")
	session.SetId("ksuid-mod-get-err")
	session.SetKubeCrName("ksuid-mod-get-err")

	event := informer.ResourceEvent{
		Type:     informer.EventModified,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err == nil {
		t.Fatal("expected error when Get returns InternalError on MODIFIED")
	}
	if !apierrors.IsInternalError(err) {
		t.Errorf("expected InternalError, got: %v", err)
	}
}

func TestReconcile_DeletedEvent_DeleteReturnsInternalError(t *testing.T) {
	existingCR := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "ksuid-del-err",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"displayName": "to-delete",
			},
		},
	}

	fake, kube := newFakeKubeClientWithReactors("test-ns", existingCR)
	fake.PrependReactor("delete", "agenticsessions",
		func(action clienttesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewInternalError(fmt.Errorf("etcd timeout"))
		},
	)
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("del-error-test")
	session.SetId("ksuid-del-err")
	session.SetKubeCrName("ksuid-del-err")

	event := informer.ResourceEvent{
		Type:     informer.EventDeleted,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err == nil {
		t.Fatal("expected error when Delete returns InternalError")
	}
	if !apierrors.IsInternalError(err) {
		t.Errorf("expected InternalError, got: %v", err)
	}
}

func TestReconcile_InvalidObjectType(t *testing.T) {
	kube := newFakeKubeClient("test-ns")
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	event := informer.ResourceEvent{
		Type:     informer.EventAdded,
		Resource: "sessions",
		Object:   "not-a-session",
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Errorf("expected nil error for invalid type assertion (graceful skip), got: %v", err)
	}
}

func TestReconcile_AddedEvent_NoCRName(t *testing.T) {
	kube := newFakeKubeClient("test-ns")
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("no-id-session")

	event := informer.ResourceEvent{
		Type:     informer.EventAdded,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err == nil {
		t.Fatal("expected error when session has no kube_cr_name and no id")
	}
}

func TestReconcile_ModifiedEvent_NoCRName(t *testing.T) {
	kube := newFakeKubeClient("test-ns")
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("no-id-modified")

	event := informer.ResourceEvent{
		Type:     informer.EventModified,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err == nil {
		t.Fatal("expected error when modified session has no kube_cr_name and no id")
	}
}

func TestReconcile_DeletedEvent_NoCRName(t *testing.T) {
	kube := newFakeKubeClient("test-ns")
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("no-id-deleted")

	event := informer.ResourceEvent{
		Type:     informer.EventDeleted,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Errorf("expected nil error for deleted with no CR name (graceful skip), got: %v", err)
	}
}

func TestReconcile_ModifiedEvent_CreateFailsWhenCRMissing(t *testing.T) {
	fake, kube := newFakeKubeClientWithReactors("test-ns")
	fake.PrependReactor("create", "agenticsessions",
		func(action clienttesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewForbidden(agenticSessionGR, "ksuid-forbidden",
				fmt.Errorf("insufficient permissions"))
		},
	)
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("forbidden-create")
	session.SetId("ksuid-forbidden")
	session.SetKubeCrName("ksuid-forbidden")

	event := informer.ResourceEvent{
		Type:     informer.EventModified,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err == nil {
		t.Fatal("expected error when create-on-missing returns Forbidden")
	}
	if !apierrors.IsForbidden(err) {
		t.Errorf("expected Forbidden error, got: %v", err)
	}
}

func TestReconcile_UnknownEventType(t *testing.T) {
	kube := newFakeKubeClient("test-ns")
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("unknown-event")
	session.SetId("ksuid-unknown")

	event := informer.ResourceEvent{
		Type:     informer.EventType("UNKNOWN"),
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Errorf("expected nil for unknown event type (graceful skip), got: %v", err)
	}
}

func TestReconcile_AddedEvent_UpdateFailsWhenCRExists(t *testing.T) {
	existingCR := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "ksuid-upd-fail",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"displayName": "old",
			},
		},
	}

	fake, kube := newFakeKubeClientWithReactors("test-ns", existingCR)
	fake.PrependReactor("update", "agenticsessions",
		func(action clienttesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewServiceUnavailable("API server shutting down")
		},
	)
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("update-fail-on-add")
	session.SetId("ksuid-upd-fail")
	session.SetKubeCrName("ksuid-upd-fail")

	event := informer.ResourceEvent{
		Type:     informer.EventAdded,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err == nil {
		t.Fatal("expected error when update-on-existing-add returns ServiceUnavailable")
	}
	if !apierrors.IsServiceUnavailable(err) {
		t.Errorf("expected ServiceUnavailable error, got: %v", err)
	}
}

func TestReconcile_AddedEvent_MixedCaseKSUID_LowercasedCRName(t *testing.T) {
	kube := newFakeKubeClient("test-ns")
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("mixed-case-test")
	session.SetId("39fXvXTvj2VZAGvyv1tSidPcAJs")
	session.SetKubeCrName("39fXvXTvj2VZAGvyv1tSidPcAJs")
	session.SetPrompt("test dns-1123 compliance")

	event := informer.ResourceEvent{
		Type:     informer.EventAdded,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	cr, err := kube.GetAgenticSession(context.Background(), "39fxvxtvj2vzagvyv1tsidpcajs")
	if err != nil {
		t.Fatalf("CR not found with lowercased name: %v", err)
	}
	if cr.GetName() != "39fxvxtvj2vzagvyv1tsidpcajs" {
		t.Errorf("expected lowercased CR name, got %q", cr.GetName())
	}
}

func TestSessionToUnstructured_MixedCaseKSUID_LowercasedName(t *testing.T) {
	session := *openapi.NewSession("uppercase-id-test")
	session.SetId("39fXyFEvrhDEKpGZeJQTFPVkkBf")
	session.SetKubeCrName("39fXyFEvrhDEKpGZeJQTFPVkkBf")

	cr := SessionToUnstructured(session, "test-ns")
	if cr.GetName() != "39fxyfevrhdekpgzejqtfpvkkbf" {
		t.Errorf("expected lowercased name '39fxyfevrhdekpgzejqtfpvkkbf', got %q", cr.GetName())
	}
}

func TestIsWritebackEcho_MatchingTimestamp_ReturnsTrue(t *testing.T) {
	kube := newFakeKubeClient("test-ns")
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	ts := time.Date(2026, 2, 15, 12, 30, 0, 0, time.UTC)
	r.lastWritebackAt.Store("sess-echo", ts)

	session := *openapi.NewSession("echo-test")
	session.SetId("sess-echo")
	session.SetUpdatedAt(ts)

	if !r.isWritebackEcho(session) {
		t.Error("expected isWritebackEcho to return true when timestamps match")
	}
}

func TestIsWritebackEcho_MicrosecondTruncation_ReturnsTrue(t *testing.T) {
	kube := newFakeKubeClient("test-ns")
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	tsStored := time.Date(2026, 2, 15, 3, 39, 2, 392227000, time.UTC)
	r.lastWritebackAt.Store("sess-trunc", tsStored)

	tsFromJSON := time.Date(2026, 2, 15, 3, 39, 2, 392227000, time.UTC)
	session := *openapi.NewSession("trunc-test")
	session.SetId("sess-trunc")
	session.SetUpdatedAt(tsFromJSON)

	if !r.isWritebackEcho(session) {
		t.Error("expected isWritebackEcho to return true when microsecond-truncated timestamps match")
	}
}

func TestIsWritebackEcho_NanosecondDifference_StillMatches(t *testing.T) {
	kube := newFakeKubeClient("test-ns")
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	tsStored := time.Date(2026, 2, 15, 3, 39, 2, 392227000, time.UTC)
	r.lastWritebackAt.Store("sess-nano", tsStored)

	tsWithExtraNanos := time.Date(2026, 2, 15, 3, 39, 2, 392227670, time.UTC)
	session := *openapi.NewSession("nano-test")
	session.SetId("sess-nano")
	session.SetUpdatedAt(tsWithExtraNanos)

	if !r.isWritebackEcho(session) {
		t.Error("expected isWritebackEcho to match despite nanosecond-level difference (PostgreSQL stores microseconds)")
	}
}

func TestIsWritebackEcho_DifferentTimestamp_ReturnsFalse(t *testing.T) {
	kube := newFakeKubeClient("test-ns")
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	ts := time.Date(2026, 2, 15, 12, 30, 0, 0, time.UTC)
	r.lastWritebackAt.Store("sess-echo", ts)

	session := *openapi.NewSession("echo-test")
	session.SetId("sess-echo")
	session.SetUpdatedAt(ts.Add(5 * time.Second))

	if r.isWritebackEcho(session) {
		t.Error("expected isWritebackEcho to return false when timestamps differ")
	}
}

func TestIsWritebackEcho_NoEntry_ReturnsFalse(t *testing.T) {
	kube := newFakeKubeClient("test-ns")
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("no-entry")
	session.SetId("sess-no-entry")
	session.SetUpdatedAt(time.Now())

	if r.isWritebackEcho(session) {
		t.Error("expected isWritebackEcho to return false when no entry exists")
	}
}

func TestIsWritebackEcho_NoSessionID_ReturnsFalse(t *testing.T) {
	kube := newFakeKubeClient("test-ns")
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	session := *openapi.NewSession("no-id")

	if r.isWritebackEcho(session) {
		t.Error("expected isWritebackEcho to return false when session has no ID")
	}
}

func TestReconcile_ModifiedEvent_WritebackEcho_SkipsUpdate(t *testing.T) {
	existingCR := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "ksuid-wb-echo",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"displayName":   "original",
				"initialPrompt": "original prompt",
			},
		},
	}

	kube := newFakeKubeClient("test-ns", existingCR)
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	ts := time.Date(2026, 2, 15, 12, 30, 0, 0, time.UTC)
	r.lastWritebackAt.Store("ksuid-wb-echo", ts)

	session := *openapi.NewSession("original")
	session.SetId("ksuid-wb-echo")
	session.SetKubeCrName("ksuid-wb-echo")
	session.SetPrompt("original prompt")
	session.SetUpdatedAt(ts)

	event := informer.ResourceEvent{
		Type:     informer.EventModified,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	cr, err := kube.GetAgenticSession(context.Background(), "ksuid-wb-echo")
	if err != nil {
		t.Fatalf("CR should still exist: %v", err)
	}
	displayName, _, _ := unstructured.NestedString(cr.Object, "spec", "displayName")
	if displayName != "original" {
		t.Errorf("expected displayName to remain 'original' (skipped), got %q", displayName)
	}
}

func TestReconcile_ModifiedEvent_RealChange_ProcessesNormally(t *testing.T) {
	existingCR := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "ksuid-real-mod",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"displayName":   "old-name",
				"initialPrompt": "old prompt",
			},
		},
	}

	kube := newFakeKubeClient("test-ns", existingCR)
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	wbTs := time.Date(2026, 2, 15, 12, 30, 0, 0, time.UTC)
	r.lastWritebackAt.Store("ksuid-real-mod", wbTs)

	session := *openapi.NewSession("new-name")
	session.SetId("ksuid-real-mod")
	session.SetKubeCrName("ksuid-real-mod")
	session.SetPrompt("new prompt")
	session.SetUpdatedAt(wbTs.Add(10 * time.Second))

	event := informer.ResourceEvent{
		Type:     informer.EventModified,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	cr, err := kube.GetAgenticSession(context.Background(), "ksuid-real-mod")
	if err != nil {
		t.Fatalf("CR not found: %v", err)
	}
	displayName, _, _ := unstructured.NestedString(cr.Object, "spec", "displayName")
	if displayName != "new-name" {
		t.Errorf("expected displayName 'new-name', got %q", displayName)
	}
}

func TestReconcile_DeletedEvent_CleansUpWritebackEntry(t *testing.T) {
	existingCR := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "ksuid-del-wb",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{
				"displayName": "to-delete",
			},
		},
	}

	kube := newFakeKubeClient("test-ns", existingCR)
	r := NewSessionReconciler(nil, kube, zerolog.Nop())

	r.lastWritebackAt.Store("ksuid-del-wb", time.Now())

	session := *openapi.NewSession("to-delete")
	session.SetId("ksuid-del-wb")
	session.SetKubeCrName("ksuid-del-wb")

	event := informer.ResourceEvent{
		Type:     informer.EventDeleted,
		Resource: "sessions",
		Object:   session,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	if _, loaded := r.lastWritebackAt.Load("ksuid-del-wb"); loaded {
		t.Error("expected lastWritebackAt entry to be cleaned up after delete")
	}
}

func TestConditionTypeConstants_MatchOperator(t *testing.T) {
	operatorConditions := []string{
		"Ready", "SecretsReady", "PodCreated", "PodScheduled",
		"RunnerStarted", "ReposReconciled", "WorkflowReconciled", "Reconciled",
	}

	cpConditions := []string{
		ConditionReady, ConditionSecretsReady, ConditionPodCreated, ConditionPodScheduled,
		ConditionRunnerStarted, ConditionReposReconciled, ConditionWorkflowReconciled, ConditionReconciled,
	}

	if len(cpConditions) != len(operatorConditions) {
		t.Fatalf("expected %d condition types, got %d", len(operatorConditions), len(cpConditions))
	}
	for i, expected := range operatorConditions {
		if cpConditions[i] != expected {
			t.Errorf("condition[%d]: expected %q, got %q", i, expected, cpConditions[i])
		}
	}
}

func TestAllConditionTypes_CompleteAndOrdered(t *testing.T) {
	if len(AllConditionTypes) != 8 {
		t.Fatalf("expected 8 condition types, got %d", len(AllConditionTypes))
	}
	if AllConditionTypes[0] != ConditionReady {
		t.Errorf("expected first condition to be Ready, got %q", AllConditionTypes[0])
	}
	if AllConditionTypes[7] != ConditionReconciled {
		t.Errorf("expected last condition to be Reconciled, got %q", AllConditionTypes[7])
	}
}

func TestPhaseConstants_MatchOperator(t *testing.T) {
	operatorPhases := []string{
		"Pending", "Creating", "Running", "Stopping", "Stopped", "Completed", "Failed",
	}

	cpPhases := []string{
		PhasePending, PhaseCreating, PhaseRunning, PhaseStopping,
		PhaseStopped, PhaseCompleted, PhaseFailed,
	}

	if len(cpPhases) != len(operatorPhases) {
		t.Fatalf("expected %d phases, got %d", len(operatorPhases), len(cpPhases))
	}
	for i, expected := range operatorPhases {
		if cpPhases[i] != expected {
			t.Errorf("phase[%d]: expected %q, got %q", i, expected, cpPhases[i])
		}
	}
}

func TestAllPhases_CompleteAndOrdered(t *testing.T) {
	if len(AllPhases) != 7 {
		t.Fatalf("expected 7 phases, got %d", len(AllPhases))
	}
	if AllPhases[0] != PhasePending {
		t.Errorf("expected first phase to be Pending, got %q", AllPhases[0])
	}
	if AllPhases[6] != PhaseFailed {
		t.Errorf("expected last phase to be Failed, got %q", AllPhases[6])
	}
}

func TestTerminalPhases_Correct(t *testing.T) {
	if len(TerminalPhases) != 3 {
		t.Fatalf("expected 3 terminal phases, got %d", len(TerminalPhases))
	}
	expected := map[string]bool{"Stopped": true, "Completed": true, "Failed": true}
	for _, p := range TerminalPhases {
		if !expected[p] {
			t.Errorf("unexpected terminal phase: %q", p)
		}
	}
}

func TestAutoBranchName_FromKubeCrName(t *testing.T) {
	session := *openapi.NewSession("test")
	session.SetKubeCrName("2hFxAbCdEfGhIjKlMnOpQrStUvW")
	session.SetId("should-not-use-this")

	branch := autoBranchName(session)
	if branch != "ambient/2hfxabcdefghijklmnopqrstuvw" {
		t.Errorf("expected 'ambient/2hfxabcdefghijklmnopqrstuvw', got %q", branch)
	}
}

func TestAutoBranchName_FallbackToId(t *testing.T) {
	session := *openapi.NewSession("test")
	session.SetId("KSUID-UPPER-CASE")

	branch := autoBranchName(session)
	if branch != "ambient/ksuid-upper-case" {
		t.Errorf("expected 'ambient/ksuid-upper-case', got %q", branch)
	}
}

func TestAutoBranchName_NoIdOrCrName(t *testing.T) {
	session := *openapi.NewSession("test")

	branch := autoBranchName(session)
	if branch != "ambient/session" {
		t.Errorf("expected 'ambient/session', got %q", branch)
	}
}

func TestBuildSpec_LegacyRepoURL_AutoBranch(t *testing.T) {
	session := *openapi.NewSession("legacy-auto-branch")
	session.SetId("ksuid-auto-001")
	session.SetKubeCrName("ksuid-auto-001")
	session.SetRepoUrl("https://github.com/test/repo")

	spec := buildSpec(session)

	repos, ok := spec["repos"].([]interface{})
	if !ok || len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %v", spec["repos"])
	}
	repo := repos[0].(map[string]interface{})
	if repo["branch"] != "ambient/ksuid-auto-001" {
		t.Errorf("expected branch 'ambient/ksuid-auto-001', got %v", repo["branch"])
	}
}

func TestBuildSpec_JSONRepos_BackfillsBranchWhenMissing(t *testing.T) {
	session := *openapi.NewSession("json-auto-branch")
	session.SetId("ksuid-auto-002")
	session.SetKubeCrName("ksuid-auto-002")
	session.SetRepos(`[{"url":"https://github.com/foo/bar","branch":"develop"},{"url":"https://github.com/baz/qux"}]`)

	spec := buildSpec(session)

	repos, ok := spec["repos"].([]interface{})
	if !ok || len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %v", spec["repos"])
	}

	first := repos[0].(map[string]interface{})
	if first["branch"] != "develop" {
		t.Errorf("expected first repo branch 'develop' (explicit), got %v", first["branch"])
	}

	second := repos[1].(map[string]interface{})
	if second["branch"] != "ambient/ksuid-auto-002" {
		t.Errorf("expected second repo branch 'ambient/ksuid-auto-002' (auto-generated), got %v", second["branch"])
	}
}

func TestBuildSpec_JSONRepos_PreservesExplicitBranch(t *testing.T) {
	session := *openapi.NewSession("preserve-branch")
	session.SetId("ksuid-auto-003")
	session.SetRepos(`[{"url":"https://github.com/foo/bar","branch":"feature/my-work"}]`)

	spec := buildSpec(session)

	repos := spec["repos"].([]interface{})
	repo := repos[0].(map[string]interface{})
	if repo["branch"] != "feature/my-work" {
		t.Errorf("expected explicit branch 'feature/my-work' preserved, got %v", repo["branch"])
	}
}

func TestBuildSpec_MixedCaseKSUID_LowercasedInBranch(t *testing.T) {
	session := *openapi.NewSession("mixed-case-branch")
	session.SetId("39fXvXTvj2VZAGvyv1tSidPcAJs")
	session.SetKubeCrName("39fXvXTvj2VZAGvyv1tSidPcAJs")
	session.SetRepoUrl("https://github.com/test/repo")

	spec := buildSpec(session)

	repos := spec["repos"].([]interface{})
	repo := repos[0].(map[string]interface{})
	if repo["branch"] != "ambient/39fxvxtvj2vzagvyv1tsidpcajs" {
		t.Errorf("expected lowercased branch 'ambient/39fxvxtvj2vzagvyv1tsidpcajs', got %v", repo["branch"])
	}
}

func newFakeKubeClientWithNamespaces(namespace string, objects ...runtime.Object) *kubeclient.KubeClient {
	sch := runtime.NewScheme()
	sch.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "vteam.ambient-code", Version: "v1alpha1", Kind: "AgenticSession"},
		&unstructured.Unstructured{},
	)
	sch.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "vteam.ambient-code", Version: "v1alpha1", Kind: "AgenticSessionList"},
		&unstructured.UnstructuredList{},
	)
	sch.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"},
		&unstructured.Unstructured{},
	)
	sch.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "", Version: "v1", Kind: "NamespaceList"},
		&unstructured.UnstructuredList{},
	)
	sch.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"},
		&unstructured.Unstructured{},
	)
	sch.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBindingList"},
		&unstructured.UnstructuredList{},
	)
	fakeClient := dynamicfake.NewSimpleDynamicClient(sch, objects...)
	return kubeclient.NewFromDynamic(fakeClient, namespace, zerolog.Nop())
}

func TestProjectReconciler_Resource(t *testing.T) {
	r := NewProjectReconciler(nil, nil, zerolog.Nop())
	if r.Resource() != "projects" {
		t.Errorf("expected resource 'projects', got %q", r.Resource())
	}
}

func TestProjectSettingsReconciler_Resource(t *testing.T) {
	r := NewProjectSettingsReconciler(nil, nil, zerolog.Nop())
	if r.Resource() != "project_settings" {
		t.Errorf("expected resource 'project_settings', got %q", r.Resource())
	}
}

func TestProjectReconcile_AddedEvent_CreatesNamespace(t *testing.T) {
	kube := newFakeKubeClientWithNamespaces("default")
	r := NewProjectReconciler(nil, kube, zerolog.Nop())

	project := *openapi.NewProject("team-alpha")
	project.SetId("proj-001")

	event := informer.ResourceEvent{
		Type:     informer.EventAdded,
		Resource: "projects",
		Object:   project,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	ns, err := kube.GetNamespace(context.Background(), "team-alpha")
	if err != nil {
		t.Fatalf("namespace not created: %v", err)
	}

	labels := ns.GetLabels()
	if labels[LabelManaged] != "true" {
		t.Errorf("expected label %s=true, got %q", LabelManaged, labels[LabelManaged])
	}
	if labels[LabelProjectID] != "proj-001" {
		t.Errorf("expected label %s=proj-001, got %q", LabelProjectID, labels[LabelProjectID])
	}
	if labels[LabelManagedBy] != "ambient-control-plane" {
		t.Errorf("expected label %s=ambient-control-plane, got %q", LabelManagedBy, labels[LabelManagedBy])
	}
}

func TestProjectReconcile_AddedEvent_ExistingNamespace_ReconcilesLabels(t *testing.T) {
	existingNS := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": "team-beta",
			},
		},
	}

	kube := newFakeKubeClientWithNamespaces("default", existingNS)
	r := NewProjectReconciler(nil, kube, zerolog.Nop())

	project := *openapi.NewProject("team-beta")
	project.SetId("proj-002")

	event := informer.ResourceEvent{
		Type:     informer.EventAdded,
		Resource: "projects",
		Object:   project,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	ns, _ := kube.GetNamespace(context.Background(), "team-beta")
	labels := ns.GetLabels()
	if labels[LabelManaged] != "true" {
		t.Errorf("expected managed label after reconcile, got %q", labels[LabelManaged])
	}
	if labels[LabelProjectID] != "proj-002" {
		t.Errorf("expected project-id label after reconcile, got %q", labels[LabelProjectID])
	}
}

func TestProjectReconcile_ModifiedEvent_ReconcilesLabels(t *testing.T) {
	existingNS := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": "team-gamma",
				"labels": map[string]interface{}{
					LabelManaged:   "true",
					LabelProjectID: "old-id",
					LabelManagedBy: "ambient-control-plane",
				},
			},
		},
	}

	kube := newFakeKubeClientWithNamespaces("default", existingNS)
	r := NewProjectReconciler(nil, kube, zerolog.Nop())

	project := *openapi.NewProject("team-gamma")
	project.SetId("new-id")

	event := informer.ResourceEvent{
		Type:     informer.EventModified,
		Resource: "projects",
		Object:   project,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	ns, _ := kube.GetNamespace(context.Background(), "team-gamma")
	labels := ns.GetLabels()
	if labels[LabelProjectID] != "new-id" {
		t.Errorf("expected project-id updated to 'new-id', got %q", labels[LabelProjectID])
	}
}

func TestProjectReconcile_ModifiedEvent_LabelsAlreadyCorrect_NoUpdate(t *testing.T) {
	existingNS := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": "team-delta",
				"labels": map[string]interface{}{
					LabelManaged:   "true",
					LabelProjectID: "proj-004",
					LabelManagedBy: "ambient-control-plane",
				},
			},
		},
	}

	kube := newFakeKubeClientWithNamespaces("default", existingNS)
	r := NewProjectReconciler(nil, kube, zerolog.Nop())

	project := *openapi.NewProject("team-delta")
	project.SetId("proj-004")

	event := informer.ResourceEvent{
		Type:     informer.EventModified,
		Resource: "projects",
		Object:   project,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}
}

func TestProjectReconcile_DeletedEvent_NamespaceRetained(t *testing.T) {
	existingNS := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": "team-deleted",
				"labels": map[string]interface{}{
					LabelManaged: "true",
				},
			},
		},
	}

	kube := newFakeKubeClientWithNamespaces("default", existingNS)
	r := NewProjectReconciler(nil, kube, zerolog.Nop())

	project := *openapi.NewProject("team-deleted")
	project.SetId("proj-del")

	event := informer.ResourceEvent{
		Type:     informer.EventDeleted,
		Resource: "projects",
		Object:   project,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	_, err = kube.GetNamespace(context.Background(), "team-deleted")
	if err != nil {
		t.Fatalf("namespace should still exist after project deletion: %v", err)
	}
}

func TestProjectReconcile_InvalidObjectType(t *testing.T) {
	r := NewProjectReconciler(nil, nil, zerolog.Nop())

	event := informer.ResourceEvent{
		Type:     informer.EventAdded,
		Resource: "projects",
		Object:   "not-a-project",
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Errorf("expected nil error for invalid type assertion (graceful skip), got: %v", err)
	}
}

func TestProjectReconcile_NoName_ReturnsError(t *testing.T) {
	kube := newFakeKubeClientWithNamespaces("default")
	r := NewProjectReconciler(nil, kube, zerolog.Nop())

	project := *openapi.NewProject("")
	project.SetId("proj-noname")

	event := informer.ResourceEvent{
		Type:     informer.EventAdded,
		Resource: "projects",
		Object:   project,
	}

	err := r.Reconcile(context.Background(), event)
	if err == nil {
		t.Fatal("expected error when project has no name")
	}
}

func TestBuildNamespace_WithProjectID(t *testing.T) {
	ns := buildNamespace("my-ns", "proj-123")

	if ns.GetName() != "my-ns" {
		t.Errorf("expected name 'my-ns', got %q", ns.GetName())
	}
	if ns.GetKind() != "Namespace" {
		t.Errorf("expected kind 'Namespace', got %q", ns.GetKind())
	}

	labels := ns.GetLabels()
	if labels[LabelManaged] != "true" {
		t.Errorf("expected managed=true, got %q", labels[LabelManaged])
	}
	if labels[LabelProjectID] != "proj-123" {
		t.Errorf("expected project-id=proj-123, got %q", labels[LabelProjectID])
	}
	if labels[LabelManagedBy] != "ambient-control-plane" {
		t.Errorf("expected managed-by=ambient-control-plane, got %q", labels[LabelManagedBy])
	}
}

func TestBuildNamespace_EmptyProjectID(t *testing.T) {
	ns := buildNamespace("my-ns", "")

	labels := ns.GetLabels()
	if _, exists := labels[LabelProjectID]; exists {
		t.Error("expected no project-id label when projectID is empty")
	}
	if labels[LabelManaged] != "true" {
		t.Errorf("expected managed=true, got %q", labels[LabelManaged])
	}
}

func TestProjectSettingsReconcile_AddedEvent_CreatesRoleBindings(t *testing.T) {
	kube := newFakeKubeClientWithNamespaces("default")
	r := NewProjectSettingsReconciler(nil, kube, zerolog.Nop())

	ps := *openapi.NewProjectSettings("team-alpha")
	ps.SetId("ps-001")
	ps.SetGroupAccess(`[{"group":"devs","role":"edit"},{"group":"admins","role":"admin"}]`)

	event := informer.ResourceEvent{
		Type:     informer.EventAdded,
		Resource: "project_settings",
		Object:   ps,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	rb1, err := kube.GetRoleBinding(context.Background(), "team-alpha", "ambient-devs-edit")
	if err != nil {
		t.Fatalf("rolebinding ambient-devs-edit not created: %v", err)
	}
	role1, _, _ := unstructured.NestedString(rb1.Object, "roleRef", "name")
	if role1 != "edit" {
		t.Errorf("expected roleRef.name 'edit', got %q", role1)
	}

	rb2, err := kube.GetRoleBinding(context.Background(), "team-alpha", "ambient-admins-admin")
	if err != nil {
		t.Fatalf("rolebinding ambient-admins-admin not created: %v", err)
	}
	role2, _, _ := unstructured.NestedString(rb2.Object, "roleRef", "name")
	if role2 != "admin" {
		t.Errorf("expected roleRef.name 'admin', got %q", role2)
	}
}

func TestProjectSettingsReconcile_ModifiedEvent_UpdatesRoleBindings(t *testing.T) {
	existingRB := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "RoleBinding",
			"metadata": map[string]interface{}{
				"name":      "ambient-devs-edit",
				"namespace": "team-beta",
			},
			"roleRef": map[string]interface{}{
				"apiGroup": "rbac.authorization.k8s.io",
				"kind":     "ClusterRole",
				"name":     "edit",
			},
			"subjects": []interface{}{
				map[string]interface{}{
					"kind":     "Group",
					"name":     "devs",
					"apiGroup": "rbac.authorization.k8s.io",
				},
			},
		},
	}

	kube := newFakeKubeClientWithNamespaces("default", existingRB)
	r := NewProjectSettingsReconciler(nil, kube, zerolog.Nop())

	ps := *openapi.NewProjectSettings("team-beta")
	ps.SetId("ps-002")
	ps.SetGroupAccess(`[{"group":"devs","role":"edit"}]`)

	event := informer.ResourceEvent{
		Type:     informer.EventModified,
		Resource: "project_settings",
		Object:   ps,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	rb, _ := kube.GetRoleBinding(context.Background(), "team-beta", "ambient-devs-edit")
	role, _, _ := unstructured.NestedString(rb.Object, "roleRef", "name")
	if role != "edit" {
		t.Errorf("expected roleRef.name 'edit', got %q", role)
	}
}

func TestProjectSettingsReconcile_DeletedEvent_RoleBindingsRetained(t *testing.T) {
	existingRB := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "RoleBinding",
			"metadata": map[string]interface{}{
				"name":      "ambient-devs-edit",
				"namespace": "team-gamma",
			},
			"roleRef": map[string]interface{}{
				"apiGroup": "rbac.authorization.k8s.io",
				"kind":     "ClusterRole",
				"name":     "edit",
			},
		},
	}

	kube := newFakeKubeClientWithNamespaces("default", existingRB)
	r := NewProjectSettingsReconciler(nil, kube, zerolog.Nop())

	ps := *openapi.NewProjectSettings("team-gamma")
	ps.SetId("ps-003")

	event := informer.ResourceEvent{
		Type:     informer.EventDeleted,
		Resource: "project_settings",
		Object:   ps,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	_, err = kube.GetRoleBinding(context.Background(), "team-gamma", "ambient-devs-edit")
	if err != nil {
		t.Fatalf("rolebinding should still exist after settings deletion: %v", err)
	}
}

func TestProjectSettingsReconcile_EmptyGroupAccess_NoOp(t *testing.T) {
	kube := newFakeKubeClientWithNamespaces("default")
	r := NewProjectSettingsReconciler(nil, kube, zerolog.Nop())

	ps := *openapi.NewProjectSettings("team-empty")
	ps.SetId("ps-004")

	event := informer.ResourceEvent{
		Type:     informer.EventAdded,
		Resource: "project_settings",
		Object:   ps,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}
}

func TestProjectSettingsReconcile_InvalidGroupAccessJSON_NoError(t *testing.T) {
	kube := newFakeKubeClientWithNamespaces("default")
	r := NewProjectSettingsReconciler(nil, kube, zerolog.Nop())

	ps := *openapi.NewProjectSettings("team-invalid")
	ps.SetId("ps-005")
	ps.SetGroupAccess(`not valid json`)

	event := informer.ResourceEvent{
		Type:     informer.EventAdded,
		Resource: "project_settings",
		Object:   ps,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("expected nil error for invalid JSON (graceful skip), got: %v", err)
	}
}

func TestProjectSettingsReconcile_SkipsEmptyGroupOrRole(t *testing.T) {
	kube := newFakeKubeClientWithNamespaces("default")
	r := NewProjectSettingsReconciler(nil, kube, zerolog.Nop())

	ps := *openapi.NewProjectSettings("team-skip")
	ps.SetId("ps-006")
	ps.SetGroupAccess(`[{"group":"","role":"edit"},{"group":"devs","role":""},{"group":"ops","role":"view"}]`)

	event := informer.ResourceEvent{
		Type:     informer.EventAdded,
		Resource: "project_settings",
		Object:   ps,
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	_, err = kube.GetRoleBinding(context.Background(), "team-skip", "ambient-ops-view")
	if err != nil {
		t.Fatalf("expected rolebinding for ops/view, got error: %v", err)
	}
}

func TestProjectSettingsReconcile_InvalidObjectType(t *testing.T) {
	r := NewProjectSettingsReconciler(nil, nil, zerolog.Nop())

	event := informer.ResourceEvent{
		Type:     informer.EventAdded,
		Resource: "project_settings",
		Object:   "not-project-settings",
	}

	err := r.Reconcile(context.Background(), event)
	if err != nil {
		t.Errorf("expected nil error for invalid type assertion (graceful skip), got: %v", err)
	}
}

func TestProjectSettingsReconcile_NoProjectID_ReturnsError(t *testing.T) {
	kube := newFakeKubeClientWithNamespaces("default")
	r := NewProjectSettingsReconciler(nil, kube, zerolog.Nop())

	ps := *openapi.NewProjectSettings("")
	ps.SetId("ps-007")
	ps.SetGroupAccess(`[{"group":"devs","role":"edit"}]`)

	event := informer.ResourceEvent{
		Type:     informer.EventAdded,
		Resource: "project_settings",
		Object:   ps,
	}

	err := r.Reconcile(context.Background(), event)
	if err == nil {
		t.Fatal("expected error when project settings has no project_id")
	}
}

func TestBuildRoleBinding_CorrectStructure(t *testing.T) {
	entry := GroupAccessEntry{Group: "developers", Role: "edit"}
	rb := buildRoleBinding("my-ns", "ambient-developers-edit", entry)

	if rb.GetName() != "ambient-developers-edit" {
		t.Errorf("expected name 'ambient-developers-edit', got %q", rb.GetName())
	}
	if rb.GetNamespace() != "my-ns" {
		t.Errorf("expected namespace 'my-ns', got %q", rb.GetNamespace())
	}

	labels := rb.GetLabels()
	if labels[LabelManaged] != "true" {
		t.Errorf("expected managed=true, got %q", labels[LabelManaged])
	}
	if labels[LabelManagedBy] != "ambient-control-plane" {
		t.Errorf("expected managed-by=ambient-control-plane, got %q", labels[LabelManagedBy])
	}

	roleRefName, _, _ := unstructured.NestedString(rb.Object, "roleRef", "name")
	if roleRefName != "edit" {
		t.Errorf("expected roleRef.name 'edit', got %q", roleRefName)
	}

	roleRefKind, _, _ := unstructured.NestedString(rb.Object, "roleRef", "kind")
	if roleRefKind != "ClusterRole" {
		t.Errorf("expected roleRef.kind 'ClusterRole', got %q", roleRefKind)
	}

	subjects, _, _ := unstructured.NestedSlice(rb.Object, "subjects")
	if len(subjects) != 1 {
		t.Fatalf("expected 1 subject, got %d", len(subjects))
	}

	subject := subjects[0].(map[string]interface{})
	if subject["kind"] != "Group" {
		t.Errorf("expected subject kind 'Group', got %v", subject["kind"])
	}
	if subject["name"] != "developers" {
		t.Errorf("expected subject name 'developers', got %v", subject["name"])
	}
}

func TestLabelConstants(t *testing.T) {
	if LabelManaged != "ambient-code.io/managed" {
		t.Errorf("LabelManaged = %q, want 'ambient-code.io/managed'", LabelManaged)
	}
	if LabelProjectID != "ambient-code.io/project-id" {
		t.Errorf("LabelProjectID = %q, want 'ambient-code.io/project-id'", LabelProjectID)
	}
	if LabelManagedBy != "ambient-code.io/managed-by" {
		t.Errorf("LabelManagedBy = %q, want 'ambient-code.io/managed-by'", LabelManagedBy)
	}
}
