package types

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSessionBuilder_ValidSession(t *testing.T) {
	s, err := NewSessionBuilder().
		Name("test-session").
		Prompt("analyze this").
		RepoURL("https://github.com/foo/bar").
		WorkflowID("wf-123").
		AssignedUserID("user-1").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "test-session" {
		t.Errorf("got Name=%q, want %q", s.Name, "test-session")
	}
	if s.Prompt != "analyze this" {
		t.Errorf("got Prompt=%q, want %q", s.Prompt, "analyze this")
	}
	if s.RepoURL != "https://github.com/foo/bar" {
		t.Errorf("got RepoURL=%q, want %q", s.RepoURL, "https://github.com/foo/bar")
	}
	if s.WorkflowID != "wf-123" {
		t.Errorf("got WorkflowID=%q, want %q", s.WorkflowID, "wf-123")
	}
}

func TestSessionBuilder_MissingName(t *testing.T) {
	_, err := NewSessionBuilder().
		Prompt("analyze this").
		Build()
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("error should mention 'name is required', got: %v", err)
	}
}

func TestAgentBuilder_ValidAgent(t *testing.T) {
	a, err := NewAgentBuilder().
		Name("test-agent").
		ProjectID("proj-1").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Name != "test-agent" {
		t.Errorf("got Name=%q, want %q", a.Name, "test-agent")
	}
	if a.ProjectID != "proj-1" {
		t.Errorf("got ProjectID=%q, want %q", a.ProjectID, "proj-1")
	}
}

func TestAgentBuilder_MissingName(t *testing.T) {
	_, err := NewAgentBuilder().Prompt("do stuff").Build()
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestProjectBuilder_ValidProject(t *testing.T) {
	p, err := NewProjectBuilder().
		Name("my-project").
		DisplayName("My Project").
		Description("A test project").
		Labels("env=dev").
		Annotations("note=test").
		Status("active").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "my-project" {
		t.Errorf("got Name=%q, want %q", p.Name, "my-project")
	}
	if p.DisplayName != "My Project" {
		t.Errorf("got DisplayName=%q, want %q", p.DisplayName, "My Project")
	}
}

func TestProjectBuilder_MissingName(t *testing.T) {
	_, err := NewProjectBuilder().Description("no name").Build()
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestProjectSettingsBuilder_Valid(t *testing.T) {
	ps, err := NewProjectSettingsBuilder().
		ProjectID("proj-123").
		GroupAccess("admin,dev").
		RunnerSecrets("secret-ref").
		Repositories("repo1,repo2").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ps.ProjectID != "proj-123" {
		t.Errorf("got ProjectID=%q, want %q", ps.ProjectID, "proj-123")
	}
}

func TestProjectSettingsBuilder_MissingProjectID(t *testing.T) {
	_, err := NewProjectSettingsBuilder().GroupAccess("admin").Build()
	if err == nil {
		t.Fatal("expected error for missing project_id")
	}
	if !strings.Contains(err.Error(), "project_id is required") {
		t.Errorf("error should mention 'project_id is required', got: %v", err)
	}
}

func TestUserBuilder_Valid(t *testing.T) {
	u, err := NewUserBuilder().
		Name("Alice").
		Username("alice").
		Groups("admin,dev").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Groups != "admin,dev" {
		t.Errorf("got Groups=%q, want %q", u.Groups, "admin,dev")
	}
}

func TestUserBuilder_MissingBothRequired(t *testing.T) {
	_, err := NewUserBuilder().Groups("admin").Build()
	if err == nil {
		t.Fatal("expected error for missing name and username")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("error should mention 'name is required', got: %v", err)
	}
	if !strings.Contains(err.Error(), "username is required") {
		t.Errorf("error should mention 'username is required', got: %v", err)
	}
}

func TestWorkflowBuilder_NewFields(t *testing.T) {
	w, err := NewWorkflowBuilder().
		Name("ci-workflow").
		ProjectID("proj-1").
		Branch("main").
		Path("/workflows/ci").
		AgentID("agent-1").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Branch != "main" {
		t.Errorf("got Branch=%q, want %q", w.Branch, "main")
	}
	if w.Path != "/workflows/ci" {
		t.Errorf("got Path=%q, want %q", w.Path, "/workflows/ci")
	}
	if w.ProjectID != "proj-1" {
		t.Errorf("got ProjectID=%q, want %q", w.ProjectID, "proj-1")
	}
}

func TestWorkflowSkillBuilder_Valid(t *testing.T) {
	ws, err := NewWorkflowSkillBuilder().
		WorkflowID("wf-1").
		SkillID("sk-1").
		Position(1).
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.WorkflowID != "wf-1" {
		t.Errorf("got WorkflowID=%q, want %q", ws.WorkflowID, "wf-1")
	}
	if ws.Position != 1 {
		t.Errorf("got Position=%d, want %d", ws.Position, 1)
	}
}

func TestWorkflowTaskBuilder_Valid(t *testing.T) {
	wt, err := NewWorkflowTaskBuilder().
		WorkflowID("wf-1").
		TaskID("task-1").
		Position(2).
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wt.TaskID != "task-1" {
		t.Errorf("got TaskID=%q, want %q", wt.TaskID, "task-1")
	}
}

func TestListOptions_Defaults(t *testing.T) {
	opts := NewListOptions().Build()
	if opts.Page != 1 {
		t.Errorf("got Page=%d, want %d", opts.Page, 1)
	}
	if opts.Size != 100 {
		t.Errorf("got Size=%d, want %d", opts.Size, 100)
	}
}

func TestListOptions_MaxSize(t *testing.T) {
	opts := NewListOptions().Size(999999).Build()
	if opts.Size != 65500 {
		t.Errorf("got Size=%d, want %d (capped at max)", opts.Size, 65500)
	}
}

func TestListOptions_AllFields(t *testing.T) {
	opts := NewListOptions().
		Page(3).
		Size(50).
		Search("name like 'test%'").
		OrderBy("created_at desc").
		Fields("id,name,status").
		Build()
	if opts.Page != 3 {
		t.Errorf("got Page=%d, want %d", opts.Page, 3)
	}
	if opts.Size != 50 {
		t.Errorf("got Size=%d, want %d", opts.Size, 50)
	}
	if opts.Search != "name like 'test%'" {
		t.Errorf("got Search=%q, want %q", opts.Search, "name like 'test%'")
	}
	if opts.OrderBy != "created_at desc" {
		t.Errorf("got OrderBy=%q, want %q", opts.OrderBy, "created_at desc")
	}
	if opts.Fields != "id,name,status" {
		t.Errorf("got Fields=%q, want %q", opts.Fields, "id,name,status")
	}
}

func TestPatchBuilder_SetsOnlySpecifiedFields(t *testing.T) {
	patch := NewSessionPatchBuilder().
		Prompt("updated prompt").
		Build()
	if len(patch) != 1 {
		t.Errorf("got %d fields, want 1", len(patch))
	}
	if patch["prompt"] != "updated prompt" {
		t.Errorf("got prompt=%v, want %q", patch["prompt"], "updated prompt")
	}
	if _, ok := patch["name"]; ok {
		t.Error("name should not be in patch when not set")
	}
}

func TestProjectPatchBuilder_AllFields(t *testing.T) {
	patch := NewProjectPatchBuilder().
		Name("renamed").
		DisplayName("Renamed").
		Description("new desc").
		Labels("env=prod").
		Annotations("a=b").
		Status("archived").
		Build()
	if len(patch) != 6 {
		t.Errorf("got %d fields, want 6", len(patch))
	}
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name   string
		err    APIError
		expect string
	}{
		{
			name:   "with status code",
			err:    APIError{StatusCode: 404, Code: "NOT_FOUND", Reason: "session not found"},
			expect: "ambient API error 404: NOT_FOUND — session not found",
		},
		{
			name:   "without status code",
			err:    APIError{Code: "VALIDATION", Reason: "invalid input"},
			expect: "ambient API error: VALIDATION — invalid input",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expect {
				t.Errorf("got %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestSessionBuilder_WP4Fields(t *testing.T) {
	s, err := NewSessionBuilder().
		Name("wp4-session").
		Prompt("test prompt").
		Interactive(true).
		LlmModel("claude-4-opus").
		LlmTemperature(0.7).
		LlmMaxTokens(4096).
		Repos(`[{"url":"https://github.com/org/repo"}]`).
		Labels("env=dev,team=platform").
		Annotations("note=wp4-test").
		ProjectID("proj-1").
		ParentSessionID("parent-123").
		BotAccountName("bot-1").
		ResourceOverrides(`{"cpu":"2","memory":"4Gi"}`).
		EnvironmentVariables(`{"DEBUG":"true"}`).
		Timeout(3600).
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.Interactive {
		t.Error("expected Interactive=true")
	}
	if s.LlmTemperature != 0.7 {
		t.Errorf("got LlmTemperature=%f, want 0.7", s.LlmTemperature)
	}
	if s.LlmMaxTokens != 4096 {
		t.Errorf("got LlmMaxTokens=%d, want 4096", s.LlmMaxTokens)
	}
	if s.LlmModel != "claude-4-opus" {
		t.Errorf("got LlmModel=%q, want %q", s.LlmModel, "claude-4-opus")
	}
	if s.Timeout != 3600 {
		t.Errorf("got Timeout=%d, want 3600", s.Timeout)
	}
	if s.ProjectID != "proj-1" {
		t.Errorf("got ProjectID=%q, want %q", s.ProjectID, "proj-1")
	}
	if s.BotAccountName != "bot-1" {
		t.Errorf("got BotAccountName=%q, want %q", s.BotAccountName, "bot-1")
	}
}

func TestSessionBuilder_ReadOnlyFieldsNotOnBuilder(t *testing.T) {
	s, err := NewSessionBuilder().
		Name("readonly-test").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Phase != "" {
		t.Error("Phase should be zero-value (not settable via builder)")
	}
	if s.KubeCrName != "" {
		t.Error("KubeCrName should be zero-value")
	}
	if s.KubeCrUid != "" {
		t.Error("KubeCrUid should be zero-value")
	}
	if s.KubeNamespace != "" {
		t.Error("KubeNamespace should be zero-value")
	}
	if s.CompletionTime != nil {
		t.Error("CompletionTime should be nil")
	}
	if s.StartTime != nil {
		t.Error("StartTime should be nil")
	}
	if s.SdkRestartCount != 0 {
		t.Error("SdkRestartCount should be zero")
	}
	if s.SdkSessionID != "" {
		t.Error("SdkSessionID should be zero-value")
	}
	if s.Conditions != "" {
		t.Error("Conditions should be zero-value")
	}
	if s.ReconciledRepos != "" {
		t.Error("ReconciledRepos should be zero-value")
	}
	if s.ReconciledWorkflow != "" {
		t.Error("ReconciledWorkflow should be zero-value")
	}
}

func TestSessionPatchBuilder_WP4Fields(t *testing.T) {
	patch := NewSessionPatchBuilder().
		Interactive(true).
		LlmTemperature(0.9).
		LlmMaxTokens(8192).
		LlmModel("claude-4-sonnet").
		Timeout(7200).
		Build()
	if patch["interactive"] != true {
		t.Error("expected interactive=true in patch")
	}
	if patch["llm_temperature"] != 0.9 {
		t.Errorf("got llm_temperature=%v, want 0.9", patch["llm_temperature"])
	}
	if patch["llm_max_tokens"] != 8192 {
		t.Errorf("got llm_max_tokens=%v, want 8192", patch["llm_max_tokens"])
	}
	if patch["timeout"] != 7200 {
		t.Errorf("got timeout=%v, want 7200", patch["timeout"])
	}
}

func TestSessionJSON_RoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	original := Session{
		ObjectReference: ObjectReference{
			ID:        "sess-123",
			Kind:      "Session",
			CreatedAt: &now,
		},
		Name:   "test-session",
		Prompt: "analyze",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Session
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID: got %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Name != original.Name {
		t.Errorf("Name: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.CreatedAt == nil || !decoded.CreatedAt.Equal(*original.CreatedAt) {
		t.Errorf("CreatedAt: got %v, want %v", decoded.CreatedAt, original.CreatedAt)
	}
}

func TestSessionJSON_WP4FullRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	start := now.Add(-time.Hour)
	original := Session{
		ObjectReference: ObjectReference{
			ID:        "sess-wp4",
			Kind:      "Session",
			CreatedAt: &now,
		},
		Name:                 "wp4-full",
		Prompt:               "analyze code",
		Interactive:          true,
		LlmModel:             "claude-4-opus",
		LlmTemperature:       0.7,
		LlmMaxTokens:         4096,
		Timeout:              3600,
		ProjectID:            "proj-1",
		Phase:                "running",
		StartTime:            &start,
		KubeCrName:           "sess-wp4-cr",
		KubeNamespace:        "ambient-code",
		SdkRestartCount:      2,
		SdkSessionID:         "sdk-abc",
		ReconciledRepos:      `["repo1"]`,
		ReconciledWorkflow:   "wf-reconciled",
		Conditions:           "Ready",
		Labels:               "env=dev",
		Repos:                `[{"url":"https://github.com/org/repo"}]`,
		ResourceOverrides:    `{"cpu":"2"}`,
		EnvironmentVariables: `{"DEBUG":"1"}`,
		BotAccountName:       "bot-1",
		ParentSessionID:      "parent-sess",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Session
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !decoded.Interactive {
		t.Error("Interactive should be true")
	}
	if decoded.LlmTemperature != 0.7 {
		t.Errorf("LlmTemperature: got %f, want 0.7", decoded.LlmTemperature)
	}
	if decoded.LlmMaxTokens != 4096 {
		t.Errorf("LlmMaxTokens: got %d, want 4096", decoded.LlmMaxTokens)
	}
	if decoded.Phase != "running" {
		t.Errorf("Phase: got %q, want %q", decoded.Phase, "running")
	}
	if decoded.StartTime == nil || !decoded.StartTime.Equal(start) {
		t.Error("StartTime should round-trip")
	}
	if decoded.KubeCrName != "sess-wp4-cr" {
		t.Errorf("KubeCrName: got %q, want %q", decoded.KubeCrName, "sess-wp4-cr")
	}
	if decoded.SdkRestartCount != 2 {
		t.Errorf("SdkRestartCount: got %d, want 2", decoded.SdkRestartCount)
	}
	if decoded.BotAccountName != "bot-1" {
		t.Errorf("BotAccountName: got %q, want %q", decoded.BotAccountName, "bot-1")
	}
}

func TestSessionStatusPatchBuilder_AllFields(t *testing.T) {
	now := time.Now().UTC()
	patch := NewSessionStatusPatchBuilder().
		Phase("Running").
		StartTime(&now).
		SdkSessionID("sdk-123").
		SdkRestartCount(2).
		Conditions(`[{"type":"Ready","status":"True"}]`).
		KubeCrUid("uid-abc").
		KubeNamespace("ambient-code").
		ReconciledRepos(`["repo1","repo2"]`).
		ReconciledWorkflow(`{"id":"wf-1"}`).
		Build()
	if patch["phase"] != "Running" {
		t.Errorf("phase: got %v, want %q", patch["phase"], "Running")
	}
	if patch["sdk_restart_count"] != 2 {
		t.Errorf("sdk_restart_count: got %v, want 2", patch["sdk_restart_count"])
	}
	if patch["kube_cr_uid"] != "uid-abc" {
		t.Errorf("kube_cr_uid: got %v, want %q", patch["kube_cr_uid"], "uid-abc")
	}
	if patch["kube_namespace"] != "ambient-code" {
		t.Errorf("kube_namespace: got %v, want %q", patch["kube_namespace"], "ambient-code")
	}
	if len(patch) != 9 {
		t.Errorf("expected 9 fields in status patch, got %d", len(patch))
	}
}

func TestSessionStatusPatchBuilder_SparseUpdate(t *testing.T) {
	patch := NewSessionStatusPatchBuilder().
		Phase("Completed").
		Build()
	if len(patch) != 1 {
		t.Errorf("expected 1 field in sparse status patch, got %d", len(patch))
	}
	if patch["phase"] != "Completed" {
		t.Errorf("phase: got %v, want %q", patch["phase"], "Completed")
	}
	if _, ok := patch["kube_namespace"]; ok {
		t.Error("kube_namespace should not be in sparse patch")
	}
}

func TestSessionStatusPatchBuilder_CompletionTime(t *testing.T) {
	now := time.Now().UTC()
	patch := NewSessionStatusPatchBuilder().
		CompletionTime(&now).
		Build()
	if got, ok := patch["completion_time"].(*time.Time); !ok || !got.Equal(now) {
		t.Errorf("completion_time should round-trip")
	}
}

func TestProjectJSON_RoundTrip(t *testing.T) {
	original := Project{
		ObjectReference: ObjectReference{ID: "proj-1", Kind: "Project"},
		Name:            "my-project",
		DisplayName:     "My Project",
		Description:     "desc",
		Labels:          "env=dev",
		Status:          "active",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Project
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.DisplayName != "My Project" {
		t.Errorf("DisplayName: got %q, want %q", decoded.DisplayName, "My Project")
	}
}

func TestSessionListJSON(t *testing.T) {
	raw := `{"kind":"SessionList","page":1,"size":100,"total":2,"items":[{"id":"s1","name":"a"},{"id":"s2","name":"b"}]}`
	var list SessionList
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if list.GetTotal() != 2 {
		t.Errorf("total: got %d, want 2", list.GetTotal())
	}
	if list.GetPage() != 1 {
		t.Errorf("page: got %d, want 1", list.GetPage())
	}
	if len(list.GetItems()) != 2 {
		t.Errorf("items: got %d, want 2", len(list.GetItems()))
	}
	if list.Items[0].Name != "a" {
		t.Errorf("first item name: got %q, want %q", list.Items[0].Name, "a")
	}
}

func TestPermissionBuilder_Valid(t *testing.T) {
	p, err := NewPermissionBuilder().
		SubjectType("user").
		SubjectName("alice").
		Role("admin").
		ProjectID("proj-1").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.SubjectType != "user" {
		t.Errorf("got SubjectType=%q, want %q", p.SubjectType, "user")
	}
	if p.SubjectName != "alice" {
		t.Errorf("got SubjectName=%q, want %q", p.SubjectName, "alice")
	}
	if p.Role != "admin" {
		t.Errorf("got Role=%q, want %q", p.Role, "admin")
	}
}

func TestPermissionBuilder_MissingRequired(t *testing.T) {
	_, err := NewPermissionBuilder().
		SubjectType("user").
		Build()
	if err == nil {
		t.Fatal("expected error for missing required fields")
	}
	if !strings.Contains(err.Error(), "role is required") {
		t.Errorf("error should mention 'role is required', got: %v", err)
	}
	if !strings.Contains(err.Error(), "subject_name is required") {
		t.Errorf("error should mention 'subject_name is required', got: %v", err)
	}
}

func TestPermissionJSON_RoundTrip(t *testing.T) {
	original := Permission{
		ObjectReference: ObjectReference{ID: "perm-1", Kind: "Permission"},
		SubjectType:     "user",
		SubjectName:     "alice",
		Role:            "admin",
		ProjectID:       "proj-1",
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded Permission
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.SubjectType != "user" {
		t.Errorf("SubjectType: got %q, want %q", decoded.SubjectType, "user")
	}
	if decoded.Role != "admin" {
		t.Errorf("Role: got %q, want %q", decoded.Role, "admin")
	}
}

func TestPermissionPatchBuilder(t *testing.T) {
	patch := NewPermissionPatchBuilder().
		Role("view").
		SubjectName("bob").
		Build()
	if patch["role"] != "view" {
		t.Errorf("role: got %v, want %q", patch["role"], "view")
	}
	if patch["subject_name"] != "bob" {
		t.Errorf("subject_name: got %v, want %q", patch["subject_name"], "bob")
	}
	if len(patch) != 2 {
		t.Errorf("expected 2 fields, got %d", len(patch))
	}
}

func TestRepositoryRefBuilder_Valid(t *testing.T) {
	r, err := NewRepositoryRefBuilder().
		Name("my-repo").
		URL("https://github.com/org/repo").
		Branch("main").
		Provider("github").
		Owner("org").
		RepoName("repo").
		ProjectID("proj-1").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Name != "my-repo" {
		t.Errorf("got Name=%q, want %q", r.Name, "my-repo")
	}
	if r.URL != "https://github.com/org/repo" {
		t.Errorf("got URL=%q, want %q", r.URL, "https://github.com/org/repo")
	}
	if r.Branch != "main" {
		t.Errorf("got Branch=%q, want %q", r.Branch, "main")
	}
	if r.Provider != "github" {
		t.Errorf("got Provider=%q, want %q", r.Provider, "github")
	}
}

func TestRepositoryRefBuilder_MissingRequired(t *testing.T) {
	_, err := NewRepositoryRefBuilder().
		Branch("main").
		Build()
	if err == nil {
		t.Fatal("expected error for missing required fields")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("error should mention 'name is required', got: %v", err)
	}
	if !strings.Contains(err.Error(), "url is required") {
		t.Errorf("error should mention 'url is required', got: %v", err)
	}
}

func TestRepositoryRefJSON_RoundTrip(t *testing.T) {
	original := RepositoryRef{
		ObjectReference: ObjectReference{ID: "ref-1", Kind: "RepositoryRef"},
		Name:            "my-repo",
		URL:             "https://github.com/org/repo",
		Branch:          "main",
		Provider:        "github",
		Owner:           "org",
		RepoName:        "repo",
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded RepositoryRef
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Name != "my-repo" {
		t.Errorf("Name: got %q, want %q", decoded.Name, "my-repo")
	}
	if decoded.Provider != "github" {
		t.Errorf("Provider: got %q, want %q", decoded.Provider, "github")
	}
}

func TestRepositoryRefPatchBuilder(t *testing.T) {
	patch := NewRepositoryRefPatchBuilder().
		Branch("develop").
		URL("https://github.com/org/other").
		Build()
	if patch["branch"] != "develop" {
		t.Errorf("branch: got %v, want %q", patch["branch"], "develop")
	}
	if len(patch) != 2 {
		t.Errorf("expected 2 fields, got %d", len(patch))
	}
}

func TestPermissionListJSON(t *testing.T) {
	raw := `{"kind":"PermissionList","page":1,"size":100,"total":1,"items":[{"id":"p1","role":"admin","subject_type":"user","subject_name":"alice"}]}`
	var list PermissionList
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if list.GetTotal() != 1 {
		t.Errorf("total: got %d, want 1", list.GetTotal())
	}
	if list.Items[0].Role != "admin" {
		t.Errorf("first item role: got %q, want %q", list.Items[0].Role, "admin")
	}
}

func TestRepositoryRefListJSON(t *testing.T) {
	raw := `{"kind":"RepositoryRefList","page":1,"size":100,"total":2,"items":[{"id":"r1","name":"repo-a"},{"id":"r2","name":"repo-b"}]}`
	var list RepositoryRefList
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if list.GetTotal() != 2 {
		t.Errorf("total: got %d, want 2", list.GetTotal())
	}
	if len(list.GetItems()) != 2 {
		t.Errorf("items: got %d, want 2", len(list.GetItems()))
	}
	if list.Items[0].Name != "repo-a" {
		t.Errorf("first item name: got %q, want %q", list.Items[0].Name, "repo-a")
	}
}

func TestProjectKeyBuilder_Valid(t *testing.T) {
	pk, err := NewProjectKeyBuilder().
		Name("my-api-key").
		ProjectID("proj-1").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pk.Name != "my-api-key" {
		t.Errorf("got Name=%q, want %q", pk.Name, "my-api-key")
	}
	if pk.ProjectID != "proj-1" {
		t.Errorf("got ProjectID=%q, want %q", pk.ProjectID, "proj-1")
	}
}

func TestProjectKeyBuilder_MissingRequired(t *testing.T) {
	_, err := NewProjectKeyBuilder().
		ProjectID("proj-1").
		Build()
	if err == nil {
		t.Fatal("expected error for missing required field")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("error should mention 'name is required', got: %v", err)
	}
}

func TestProjectKeyJSON_RoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	original := ProjectKey{
		ObjectReference: ObjectReference{ID: "pk-1", Kind: "ProjectKey", CreatedAt: &now},
		Name:            "my-api-key",
		KeyPrefix:       "ak_12345",
		PlaintextKey:    "ak_the-full-key",
		ProjectID:       "proj-1",
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded ProjectKey
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Name != "my-api-key" {
		t.Errorf("Name: got %q, want %q", decoded.Name, "my-api-key")
	}
	if decoded.KeyPrefix != "ak_12345" {
		t.Errorf("KeyPrefix: got %q, want %q", decoded.KeyPrefix, "ak_12345")
	}
	if decoded.PlaintextKey != "ak_the-full-key" {
		t.Errorf("PlaintextKey: got %q, want %q", decoded.PlaintextKey, "ak_the-full-key")
	}
}

func TestProjectKeyJSON_WithExpiresAt(t *testing.T) {
	expires := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	original := ProjectKey{
		ObjectReference: ObjectReference{ID: "pk-2", Kind: "ProjectKey"},
		Name:            "expiring-key",
		ExpiresAt:       &expires,
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded ProjectKey
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.ExpiresAt == nil || !decoded.ExpiresAt.Equal(expires) {
		t.Errorf("ExpiresAt: got %v, want %v", decoded.ExpiresAt, expires)
	}
}

func TestProjectKeyPatchBuilder(t *testing.T) {
	patch := NewProjectKeyPatchBuilder().
		Name("renamed-key").
		Build()
	if patch["name"] != "renamed-key" {
		t.Errorf("name: got %v, want %q", patch["name"], "renamed-key")
	}
	if len(patch) != 1 {
		t.Errorf("expected 1 field, got %d", len(patch))
	}
}

func TestProjectKeyListJSON(t *testing.T) {
	raw := `{"kind":"ProjectKeyList","page":1,"size":100,"total":2,"items":[{"id":"pk1","name":"key-a","key_prefix":"ak_12345"},{"id":"pk2","name":"key-b","key_prefix":"ak_67890"}]}`
	var list ProjectKeyList
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if list.GetTotal() != 2 {
		t.Errorf("total: got %d, want 2", list.GetTotal())
	}
	if len(list.GetItems()) != 2 {
		t.Errorf("items: got %d, want 2", len(list.GetItems()))
	}
	if list.Items[0].KeyPrefix != "ak_12345" {
		t.Errorf("first item key_prefix: got %q, want %q", list.Items[0].KeyPrefix, "ak_12345")
	}
}
