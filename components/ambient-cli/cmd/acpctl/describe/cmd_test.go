package describe

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ambient-code/platform/components/ambient-cli/internal/testhelper"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

func TestDescribeSession(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/sessions/s1", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.Session{
			ObjectReference: types.ObjectReference{ID: "s1"},
			Name:            "target-session",
			Phase:           "running",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "session", "s1")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, `"target-session"`) {
		t.Errorf("expected JSON with 'target-session', got: %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, `"running"`) {
		t.Errorf("expected JSON with 'running', got: %s", result.Stdout)
	}
}

func TestDescribeSession_Aliases(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/sessions/s1", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.Session{
			ObjectReference: types.ObjectReference{ID: "s1"},
			Name:            "s",
		})
	})

	for _, alias := range []string{"session", "sessions", "sess"} {
		testhelper.Configure(t, srv.URL)
		result := testhelper.Run(t, Cmd, alias, "s1")
		if result.Err != nil {
			t.Errorf("alias %q: unexpected error: %v", alias, result.Err)
		}
	}
}

func TestDescribeProject(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects/my-proj", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.Project{
			ObjectReference: types.ObjectReference{ID: "p1"},
			Name:            "my-proj",
			Description:     "A test project",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "project", "my-proj")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, `"my-proj"`) {
		t.Errorf("expected JSON with 'my-proj', got: %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, `"A test project"`) {
		t.Errorf("expected description in JSON, got: %s", result.Stdout)
	}
}

func TestDescribeProject_Aliases(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects/p1", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.Project{
			ObjectReference: types.ObjectReference{ID: "p1"},
			Name:            "p1",
		})
	})

	for _, alias := range []string{"project", "projects", "proj"} {
		testhelper.Configure(t, srv.URL)
		result := testhelper.Run(t, Cmd, alias, "p1")
		if result.Err != nil {
			t.Errorf("alias %q: unexpected error: %v", alias, result.Err)
		}
	}
}

func TestDescribeProjectSettings(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/project_settings/ps1", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.ProjectSettings{
			ObjectReference: types.ObjectReference{ID: "ps1"},
			ProjectID:       "my-project",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "project-settings", "ps1")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, `"my-project"`) {
		t.Errorf("expected project_id in JSON, got: %s", result.Stdout)
	}
}

func TestDescribeUser(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/users/u1", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.User{
			ObjectReference: types.ObjectReference{ID: "u1"},
			Username:        "alice",
			Name:            "Alice Smith",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "user", "u1")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, `"alice"`) {
		t.Errorf("expected 'alice' in JSON, got: %s", result.Stdout)
	}
}

func TestDescribeUnknownResource(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "foobar", "x")
	if result.Err == nil {
		t.Fatal("expected error for unknown resource type")
	}
	if !strings.Contains(result.Err.Error(), "unknown resource type") {
		t.Errorf("expected 'unknown resource type', got: %v", result.Err)
	}
}

func TestDescribeOutputIsJSON(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/sessions/s1", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.Session{
			ObjectReference: types.ObjectReference{ID: "s1"},
			Name:            "json-check",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "session", "s1")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.HasPrefix(strings.TrimSpace(result.Stdout), "{") {
		t.Errorf("expected JSON output (starts with '{'), got: %s", result.Stdout)
	}
}
