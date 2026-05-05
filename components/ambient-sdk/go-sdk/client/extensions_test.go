package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

// ---------------------------------------------------------------------------
// ScheduledSession extensions
// ---------------------------------------------------------------------------

func TestScheduledSessionListByProject(t *testing.T) {
	want := &types.ScheduledSessionList{
		ListMeta: types.ListMeta{Kind: "ScheduledSessionList", Page: 1, Size: 10, Total: 1},
		Items:    []types.ScheduledSession{{ObjectReference: types.ObjectReference{ID: "ss-1"}, Name: "nightly"}},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/projects/proj-a/scheduled-sessions") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(marshalJSON(t, want))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.ScheduledSessions().ListByProject(context.Background(), "proj-a", &types.ListOptions{})
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].ID != "ss-1" {
		t.Errorf("unexpected items: %+v", got.Items)
	}
}

func TestScheduledSessionGetByProject(t *testing.T) {
	want := &types.ScheduledSession{
		ObjectReference: types.ObjectReference{ID: "ss-abc"},
		Name:            "daily-build",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/projects/proj-a/scheduled-sessions/ss-abc") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(marshalJSON(t, want))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.ScheduledSessions().GetByProject(context.Background(), "proj-a", "ss-abc")
	if err != nil {
		t.Fatalf("GetByProject: %v", err)
	}
	if got.ID != "ss-abc" || got.Name != "daily-build" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestScheduledSessionCreateInProject(t *testing.T) {
	want := &types.ScheduledSession{
		ObjectReference: types.ObjectReference{ID: "ss-new"},
		Name:            "new-schedule",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/projects/proj-a/scheduled-sessions") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type: application/json")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(marshalJSON(t, want))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.ScheduledSessions().CreateInProject(context.Background(), "proj-a", &types.ScheduledSession{Name: "new-schedule"})
	if err != nil {
		t.Fatalf("CreateInProject: %v", err)
	}
	if got.ID != "ss-new" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestScheduledSessionUpdateInProject(t *testing.T) {
	want := &types.ScheduledSession{
		ObjectReference: types.ObjectReference{ID: "ss-upd"},
		Name:            "updated",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/projects/proj-a/scheduled-sessions/ss-upd") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var patch map[string]any
		if err := json.Unmarshal(body, &patch); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if patch["name"] != "updated" {
			t.Errorf("expected name=updated in patch, got %v", patch["name"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(marshalJSON(t, want))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.ScheduledSessions().UpdateInProject(context.Background(), "proj-a", "ss-upd", map[string]any{"name": "updated"})
	if err != nil {
		t.Fatalf("UpdateInProject: %v", err)
	}
	if got.ID != "ss-upd" || got.Name != "updated" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestScheduledSessionDeleteInProject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/projects/proj-a/scheduled-sessions/ss-del") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.ScheduledSessions().DeleteInProject(context.Background(), "proj-a", "ss-del"); err != nil {
		t.Fatalf("DeleteInProject: %v", err)
	}
}

func TestScheduledSessionSuspend(t *testing.T) {
	want := &types.ScheduledSession{
		ObjectReference: types.ObjectReference{ID: "ss-sus"},
		Enabled:         false,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/scheduled-sessions/ss-sus/suspend") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(marshalJSON(t, want))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.ScheduledSessions().Suspend(context.Background(), "proj-a", "ss-sus")
	if err != nil {
		t.Fatalf("Suspend: %v", err)
	}
	if got.ID != "ss-sus" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestScheduledSessionResume(t *testing.T) {
	want := &types.ScheduledSession{
		ObjectReference: types.ObjectReference{ID: "ss-res"},
		Enabled:         true,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/scheduled-sessions/ss-res/resume") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(marshalJSON(t, want))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.ScheduledSessions().Resume(context.Background(), "proj-a", "ss-res")
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if got.ID != "ss-res" || !got.Enabled {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestScheduledSessionTrigger(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/scheduled-sessions/ss-trig/trigger") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.ScheduledSessions().Trigger(context.Background(), "proj-a", "ss-trig"); err != nil {
		t.Fatalf("Trigger: %v", err)
	}
}

func TestScheduledSessionGetByName(t *testing.T) {
	want := &types.ScheduledSessionList{
		ListMeta: types.ListMeta{Kind: "ScheduledSessionList", Page: 1, Size: 10, Total: 1},
		Items:    []types.ScheduledSession{{ObjectReference: types.ObjectReference{ID: "ss-named"}, Name: "nightly-build"}},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("search") == "" {
			t.Error("expected search query param for GetByName")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(marshalJSON(t, want))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.ScheduledSessions().GetByName(context.Background(), "proj-a", "nightly-build")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if got.ID != "ss-named" || got.Name != "nightly-build" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestScheduledSessionGetByName_NotFound(t *testing.T) {
	want := &types.ScheduledSessionList{
		ListMeta: types.ListMeta{Kind: "ScheduledSessionList", Page: 1, Size: 10, Total: 0},
		Items:    []types.ScheduledSession{},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(marshalJSON(t, want))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.ScheduledSessions().GetByName(context.Background(), "proj-a", "nonexistent")
	if err == nil {
		t.Fatal("expected error for not-found name")
	}
}

// ---------------------------------------------------------------------------
// Agent StartInProject (multi-status)
// ---------------------------------------------------------------------------

func TestAgentStartInProject(t *testing.T) {
	want := &types.StartResponse{
		Session:        &types.Session{ObjectReference: types.ObjectReference{ID: "sess-started"}},
		StartingPrompt: "do the thing",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/projects/proj-a/agents/agent-1/start") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req types.StartRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if req.Prompt != "do the thing" {
			t.Errorf("expected prompt 'do the thing', got %q", req.Prompt)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(marshalJSON(t, want))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Agents().StartInProject(context.Background(), "proj-a", "agent-1", "do the thing")
	if err != nil {
		t.Fatalf("StartInProject: %v", err)
	}
	if got.Session == nil || got.Session.ID != "sess-started" {
		t.Errorf("unexpected result: %+v", got)
	}
}

// ---------------------------------------------------------------------------
// Credential GetToken
// ---------------------------------------------------------------------------

func TestCredentialGetToken(t *testing.T) {
	want := &types.CredentialTokenResponse{
		CredentialID: "cred-1",
		Provider:     "github",
		Token:        "ghp_test123",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/credentials/cred-1/token") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(marshalJSON(t, want))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Credentials().GetToken(context.Background(), "cred-1")
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if got.CredentialID != "cred-1" || got.Provider != "github" || got.Token != "ghp_test123" {
		t.Errorf("unexpected result: %+v", got)
	}
}

// ---------------------------------------------------------------------------
// doMultiStatus
// ---------------------------------------------------------------------------

func TestDoMultiStatus_AcceptsMultipleStatuses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"test"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	var result map[string]string
	err := c.doMultiStatus(context.Background(), http.MethodPost, "/test", nil, &result, http.StatusOK, http.StatusCreated)
	if err != nil {
		t.Fatalf("doMultiStatus should accept 201 when 200,201 expected: %v", err)
	}
	if result["id"] != "test" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestDoMultiStatus_RejectsUnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.doMultiStatus(context.Background(), http.MethodPost, "/test", nil, nil, http.StatusOK, http.StatusCreated)
	if err == nil {
		t.Fatal("expected error for 403 when 200,201 expected")
	}
}

// ---------------------------------------------------------------------------
// Project() accessor
// ---------------------------------------------------------------------------

func TestClientProjectAccessor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if c.Project() != testProject {
		t.Errorf("expected project %q, got %q", testProject, c.Project())
	}
}
