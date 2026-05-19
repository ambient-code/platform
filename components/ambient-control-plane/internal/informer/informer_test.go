package informer

import (
	"testing"
	"time"

	pb "github.com/ambient-code/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func strPtr(s string) *string       { return &s }
func int32Ptr(i int32) *int32       { return &i }
func float64Ptr(f float64) *float64 { return &f }

func TestProtoSessionToSDK_NilReturnsZero(t *testing.T) {
	s := protoSessionToSDK(nil)
	if s.ID != "" || s.Name != "" {
		t.Fatal("nil proto should produce zero-value Session")
	}
}

func TestProtoSessionToSDK_StandaloneSession(t *testing.T) {
	proto := &pb.Session{
		Metadata:  &pb.ObjectReference{Id: "session-standalone"},
		Name:      "no-agent-session",
		Prompt:    strPtr("just do the thing"),
		ProjectId: strPtr("my-project"),
	}

	s := protoSessionToSDK(proto)

	if s.AgentID != "" {
		t.Errorf("AgentID: got %q, want empty string for standalone session", s.AgentID)
	}
	if s.Name != "no-agent-session" {
		t.Errorf("Name: got %q, want %q", s.Name, "no-agent-session")
	}
	if s.Prompt != "just do the thing" {
		t.Errorf("Prompt: got %q, want %q", s.Prompt, "just do the thing")
	}
}

func TestProtoSessionToSDK_AllFieldsMapped(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	later := now.Add(10 * time.Minute)
	created := now.Add(-1 * time.Hour)
	updated := now.Add(-30 * time.Minute)

	proto := &pb.Session{
		Metadata: &pb.ObjectReference{
			Id:        "session-123",
			CreatedAt: timestamppb.New(created),
			UpdatedAt: timestamppb.New(updated),
		},
		Name:                 "test-session",
		Prompt:               strPtr("do something"),
		RepoUrl:              strPtr("https://github.com/example/repo"),
		Repos:                strPtr(`[{"url":"https://github.com/example/repo"}]`),
		LlmModel:             strPtr("claude-sonnet-4-6"),
		LlmTemperature:       float64Ptr(0.7),
		LlmMaxTokens:         int32Ptr(4000),
		Timeout:              int32Ptr(3600),
		ProjectId:            strPtr("my-project"),
		AgentId:              strPtr("agent-456"),
		WorkflowId:           strPtr("workflow-789"),
		BotAccountName:       strPtr("bot-user"),
		Labels:               strPtr(`{"env":"test"}`),
		Annotations:          strPtr(`{"note":"hello"}`),
		ResourceOverrides:    strPtr(`{"cpu":"2"}`),
		EnvironmentVariables: strPtr(`{"FOO":"bar"}`),
		CreatedByUserId:      strPtr("test-creator"),
		AssignedUserId:       strPtr("test-assignee"),
		ParentSessionId:      strPtr("parent-001"),
		Phase:                strPtr("Running"),
		KubeCrName:           strPtr("cr-name"),
		KubeCrUid:            strPtr("cr-uid-abc"),
		KubeNamespace:        strPtr("my-project"),
		SdkSessionId:         strPtr("sdk-sess-id"),
		SdkRestartCount:      int32Ptr(2),
		Conditions:           strPtr("Ready"),
		ReconciledRepos:      strPtr("reconciled"),
		ReconciledWorkflow:   strPtr("wf-reconciled"),
		StartTime:            timestamppb.New(now),
		CompletionTime:       timestamppb.New(later),
	}

	s := protoSessionToSDK(proto)

	checks := []struct {
		field string
		got   any
		want  any
	}{
		{"ID", s.ID, "session-123"},
		{"Name", s.Name, "test-session"},
		{"Prompt", s.Prompt, "do something"},
		{"RepoURL", s.RepoURL, "https://github.com/example/repo"},
		{"Repos", s.Repos, `[{"url":"https://github.com/example/repo"}]`},
		{"LlmModel", s.LlmModel, "claude-sonnet-4-6"},
		{"LlmTemperature", s.LlmTemperature, 0.7},
		{"LlmMaxTokens", s.LlmMaxTokens, 4000},
		{"Timeout", s.Timeout, 3600},
		{"ProjectID", s.ProjectID, "my-project"},
		{"AgentID", s.AgentID, "agent-456"},
		{"WorkflowID", s.WorkflowID, "workflow-789"},
		{"BotAccountName", s.BotAccountName, "bot-user"},
		{"Labels", s.Labels, `{"env":"test"}`},
		{"Annotations", s.Annotations, `{"note":"hello"}`},
		{"ResourceOverrides", s.ResourceOverrides, `{"cpu":"2"}`},
		{"EnvironmentVariables", s.EnvironmentVariables, `{"FOO":"bar"}`},
		{"CreatedByUserID", s.CreatedByUserID, "test-creator"},
		{"AssignedUserID", s.AssignedUserID, "test-assignee"},
		{"ParentSessionID", s.ParentSessionID, "parent-001"},
		{"Phase", s.Phase, "Running"},
		{"KubeCrName", s.KubeCrName, "cr-name"},
		{"KubeCrUid", s.KubeCrUid, "cr-uid-abc"},
		{"KubeNamespace", s.KubeNamespace, "my-project"},
		{"SdkSessionID", s.SdkSessionID, "sdk-sess-id"},
		{"SdkRestartCount", s.SdkRestartCount, 2},
		{"Conditions", s.Conditions, "Ready"},
		{"ReconciledRepos", s.ReconciledRepos, "reconciled"},
		{"ReconciledWorkflow", s.ReconciledWorkflow, "wf-reconciled"},
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.field, c.got, c.want)
		}
	}

	if s.CreatedAt == nil || !s.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt: got %v, want %v", s.CreatedAt, created)
	}
	if s.UpdatedAt == nil || !s.UpdatedAt.Equal(updated) {
		t.Errorf("UpdatedAt: got %v, want %v", s.UpdatedAt, updated)
	}
	if s.StartTime == nil || !s.StartTime.Equal(now) {
		t.Errorf("StartTime: got %v, want %v", s.StartTime, now)
	}
	if s.CompletionTime == nil || !s.CompletionTime.Equal(later) {
		t.Errorf("CompletionTime: got %v, want %v", s.CompletionTime, later)
	}
}
