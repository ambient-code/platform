package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

const (
	testToken   = "sha256~test-token-for-unit-tests-only"
	testProject = "test-project"
)

func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	c, err := NewClient(server.URL, testToken, testProject)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func marshalJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return b
}

func TestNewClient_MissingToken(t *testing.T) {
	_, err := NewClient("http://localhost:8080", "", testProject)
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestNewClient_ShortToken(t *testing.T) {
	_, err := NewClient("http://localhost:8080", "tooshort", testProject)
	if err == nil {
		t.Fatal("expected error for short token")
	}
}

func TestNewClient_PlaceholderToken(t *testing.T) {
	_, err := NewClient("http://localhost:8080", "YOUR_TOKEN_HERE", testProject)
	if err == nil {
		t.Fatal("expected error for placeholder token")
	}
}

func TestNewClient_MissingProject(t *testing.T) {
	_, err := NewClient("http://localhost:8080", testToken, "")
	if err == nil {
		t.Fatal("expected error for empty project")
	}
}

func TestNewClient_InvalidURL(t *testing.T) {
	_, err := NewClient("ftp://bad-scheme.io", testToken, testProject)
	if err == nil {
		t.Fatal("expected error for invalid URL scheme")
	}
}

func TestNewClient_PlaceholderURL(t *testing.T) {
	_, err := NewClient("http://example.com", testToken, testProject)
	if err == nil {
		t.Fatal("expected error for placeholder URL")
	}
}

func TestSessionsList(t *testing.T) {
	want := &types.SessionList{
		ListMeta: types.ListMeta{Kind: "SessionList", Page: 1, Size: 10, Total: 1},
		Items: []types.Session{
			{ObjectReference: types.ObjectReference{ID: "sess-1"}, Name: "my-session"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sessions") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer "+testToken {
			t.Errorf("missing or wrong Authorization header")
		}
		if r.Header.Get("X-Ambient-Project") != testProject {
			t.Errorf("missing or wrong X-Ambient-Project header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(marshalJSON(t, want))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Sessions().List(context.Background(), &types.ListOptions{})
	if err != nil {
		t.Fatalf("Sessions().List: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].ID != "sess-1" {
		t.Errorf("unexpected items: %+v", got.Items)
	}
	if got.Total != 1 {
		t.Errorf("expected total 1, got %d", got.Total)
	}
}

func TestSessionsGet(t *testing.T) {
	want := &types.Session{
		ObjectReference: types.ObjectReference{ID: "sess-abc"},
		Name:            "target-session",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sessions/sess-abc") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(marshalJSON(t, want))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Sessions().Get(context.Background(), "sess-abc")
	if err != nil {
		t.Fatalf("Sessions().Get: %v", err)
	}
	if got.ID != "sess-abc" || got.Name != "target-session" {
		t.Errorf("unexpected session: %+v", got)
	}
}

func TestSessionsCreate(t *testing.T) {
	input := &types.Session{Name: "new-session"}
	want := &types.Session{
		ObjectReference: types.ObjectReference{ID: "sess-new"},
		Name:            "new-session",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sessions") {
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
	got, err := c.Sessions().Create(context.Background(), input)
	if err != nil {
		t.Fatalf("Sessions().Create: %v", err)
	}
	if got.ID != "sess-new" || got.Name != "new-session" {
		t.Errorf("unexpected session: %+v", got)
	}
}

func TestSessionsDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sessions/sess-del") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Sessions().Delete(context.Background(), "sess-del"); err != nil {
		t.Fatalf("Sessions().Delete: %v", err)
	}
}

func TestSessionsStart(t *testing.T) {
	want := &types.Session{
		ObjectReference: types.ObjectReference{ID: "sess-start"},
		Name:            "start-me",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sessions/sess-start/start") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(marshalJSON(t, want))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Sessions().Start(context.Background(), "sess-start")
	if err != nil {
		t.Fatalf("Sessions().Start: %v", err)
	}
	if got.ID != "sess-start" {
		t.Errorf("unexpected session: %+v", got)
	}
}

func TestSessionsStop(t *testing.T) {
	want := &types.Session{
		ObjectReference: types.ObjectReference{ID: "sess-stop"},
		Name:            "stop-me",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sessions/sess-stop/stop") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(marshalJSON(t, want))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.Sessions().Stop(context.Background(), "sess-stop")
	if err != nil {
		t.Fatalf("Sessions().Stop: %v", err)
	}
	if got.ID != "sess-stop" {
		t.Errorf("unexpected session: %+v", got)
	}
}

func TestSessionsAPIError(t *testing.T) {
	apiErr := &types.APIError{
		Code:   "not_found",
		Reason: "session does not exist",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(marshalJSON(t, apiErr))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Sessions().Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ambientErr *types.APIError
	if !asAPIError(err, &ambientErr) {
		t.Fatalf("expected *types.APIError, got %T: %v", err, err)
	}
	if ambientErr.Code != "not_found" {
		t.Errorf("expected code 'not_found', got %q", ambientErr.Code)
	}
	if ambientErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", ambientErr.StatusCode)
	}
}

func TestSessionsUnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Sessions().Get(context.Background(), "any")
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	var ambientErr *types.APIError
	if !asAPIError(err, &ambientErr) {
		t.Fatalf("expected *types.APIError, got %T: %v", err, err)
	}
	if ambientErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", ambientErr.StatusCode)
	}
}

func TestProjectSettingsList(t *testing.T) {
	want := &types.ProjectSettingsList{
		ListMeta: types.ListMeta{Kind: "ProjectSettingsList", Page: 1, Size: 10, Total: 1},
		Items: []types.ProjectSettings{
			{ObjectReference: types.ObjectReference{ID: "ps-1"}, ProjectID: "proj-1"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/project_settings") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(marshalJSON(t, want))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.ProjectSettings().List(context.Background(), &types.ListOptions{})
	if err != nil {
		t.Fatalf("ProjectSettings().List: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].ID != "ps-1" {
		t.Errorf("unexpected items: %+v", got.Items)
	}
}

func TestProjectSettingsGet(t *testing.T) {
	want := &types.ProjectSettings{
		ObjectReference: types.ObjectReference{ID: "ps-abc"},
		ProjectID:       "proj-abc",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/project_settings/ps-abc") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(marshalJSON(t, want))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.ProjectSettings().Get(context.Background(), "ps-abc")
	if err != nil {
		t.Fatalf("ProjectSettings().Get: %v", err)
	}
	if got.ID != "ps-abc" || got.ProjectID != "proj-abc" {
		t.Errorf("unexpected settings: %+v", got)
	}
}

func TestProjectSettingsCreate(t *testing.T) {
	input := &types.ProjectSettings{ProjectID: "proj-new"}
	want := &types.ProjectSettings{
		ObjectReference: types.ObjectReference{ID: "ps-new"},
		ProjectID:       "proj-new",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(marshalJSON(t, want))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.ProjectSettings().Create(context.Background(), input)
	if err != nil {
		t.Fatalf("ProjectSettings().Create: %v", err)
	}
	if got.ID != "ps-new" {
		t.Errorf("unexpected settings: %+v", got)
	}
}

func TestProjectSettingsDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/project_settings/ps-del") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.ProjectSettings().Delete(context.Background(), "ps-del"); err != nil {
		t.Fatalf("ProjectSettings().Delete: %v", err)
	}
}

func TestListOptionsQueryParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("page") != "2" {
			t.Errorf("expected page=2, got %q", q.Get("page"))
		}
		if q.Get("size") != "25" {
			t.Errorf("expected size=25, got %q", q.Get("size"))
		}
		if q.Get("search") != "foo" {
			t.Errorf("expected search=foo, got %q", q.Get("search"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(marshalJSON(t, &types.SessionList{}))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	opts := &types.ListOptions{Page: 2, Size: 25, Search: "foo"}
	_, err := c.Sessions().List(context.Background(), opts)
	if err != nil {
		t.Fatalf("Sessions().List with opts: %v", err)
	}
}

func asAPIError(err error, target **types.APIError) bool {
	return errors.As(err, target)
}
