package get

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/internal/testhelper"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

func makeTime(s string) *time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return &t
}

func TestGetProjects_List(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		srv.RespondJSON(t, w, http.StatusOK, &types.ProjectList{
			ListMeta: types.ListMeta{Total: 2},
			Items: []types.Project{
				{ObjectReference: types.ObjectReference{ID: "p1", CreatedAt: makeTime("2026-01-01T00:00:00Z")}, Name: "alpha", DisplayName: "Alpha"},
				{ObjectReference: types.ObjectReference{ID: "p2", CreatedAt: makeTime("2026-01-02T00:00:00Z")}, Name: "beta", DisplayName: "Beta"},
			},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "projects")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "alpha") {
		t.Errorf("expected 'alpha' in output, got: %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "beta") {
		t.Errorf("expected 'beta' in output, got: %s", result.Stdout)
	}
}

func TestGetProjects_Single(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects/alpha", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.Project{
			ObjectReference: types.ObjectReference{ID: "p1"},
			Name:            "alpha",
			DisplayName:     "Alpha Project",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "project", "alpha")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, "alpha") {
		t.Errorf("expected 'alpha' in output, got: %s", result.Stdout)
	}
}

func TestGetProjects_JSON(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.ProjectList{
			Items: []types.Project{
				{ObjectReference: types.ObjectReference{ID: "p1"}, Name: "alpha"},
			},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "projects", "-o", "json")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, `"alpha"`) {
		t.Errorf("expected JSON with 'alpha', got: %s", result.Stdout)
	}
}

func TestGetSessions_List(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/sessions", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.SessionList{
			ListMeta: types.ListMeta{Total: 1},
			Items: []types.Session{
				{
					ObjectReference: types.ObjectReference{ID: "s1", CreatedAt: makeTime("2026-01-01T00:00:00Z")},
					Name:            "my-session",
					Phase:           "running",
					LlmModel:        "sonnet",
					ProjectID:       testhelper.TestProject,
				},
			},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "sessions")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "my-session") {
		t.Errorf("expected 'my-session' in output, got: %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "running") {
		t.Errorf("expected 'running' phase in output, got: %s", result.Stdout)
	}
}

func TestGetSessions_Single(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/sessions/s1", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.Session{
			ObjectReference: types.ObjectReference{ID: "s1"},
			Name:            "my-session",
			Phase:           "pending",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "session", "s1")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, "my-session") {
		t.Errorf("expected 'my-session' in output, got: %s", result.Stdout)
	}
}

func TestGetSessions_JSON(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/sessions", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.SessionList{
			Items: []types.Session{
				{ObjectReference: types.ObjectReference{ID: "s1"}, Name: "sess-json", Phase: "completed"},
			},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "sessions", "-o", "json")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, `"sess-json"`) {
		t.Errorf("expected JSON with 'sess-json', got: %s", result.Stdout)
	}
}

func TestGetAgents_List(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.AgentList{
			ListMeta: types.ListMeta{Total: 1},
			Items: []types.Agent{
				{
					ObjectReference: types.ObjectReference{ID: "a1", CreatedAt: makeTime("2026-01-01T00:00:00Z")},
					Name:            "overlord",
					DisplayName:     "Overlord",
					ProjectID:       testhelper.TestProject,
				},
			},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "agents")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "overlord") {
		t.Errorf("expected 'overlord' in output, got: %s", result.Stdout)
	}
}

func TestGetAgents_Single(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents/a1", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.Agent{
			ObjectReference: types.ObjectReference{ID: "a1"},
			Name:            "overlord",
			ProjectID:       testhelper.TestProject,
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "agent", "a1")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, "overlord") {
		t.Errorf("expected 'overlord' in output, got: %s", result.Stdout)
	}
}

func TestGetAgents_JSON(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.AgentList{
			Items: []types.Agent{
				{ObjectReference: types.ObjectReference{ID: "a1"}, Name: "api-agent"},
			},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "agents", "-o", "json")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, `"api-agent"`) {
		t.Errorf("expected JSON with 'api-agent', got: %s", result.Stdout)
	}
}

func TestGetUsers_List(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/users", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.UserList{
			Items: []types.User{
				{ObjectReference: types.ObjectReference{ID: "u1"}, Username: "alice", Name: "Alice"},
			},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "users")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, "alice") {
		t.Errorf("expected 'alice' in output, got: %s", result.Stdout)
	}
}

func TestGetUnknownResource(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "foobar")
	if result.Err == nil {
		t.Fatal("expected error for unknown resource type")
	}
	if !strings.Contains(result.Err.Error(), "unknown resource type") {
		t.Errorf("expected 'unknown resource type' error, got: %v", result.Err)
	}
}

func TestGetSessions_Aliases(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/sessions", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.SessionList{})
	})

	for _, alias := range []string{"session", "sessions", "sess"} {
		testhelper.Configure(t, srv.URL)
		result := testhelper.Run(t, Cmd, alias)
		if result.Err != nil {
			t.Errorf("alias %q: unexpected error: %v", alias, result.Err)
		}
	}
}

func TestGetProjects_Aliases(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.ProjectList{})
	})

	for _, alias := range []string{"project", "projects", "proj"} {
		testhelper.Configure(t, srv.URL)
		result := testhelper.Run(t, Cmd, alias)
		if result.Err != nil {
			t.Errorf("alias %q: unexpected error: %v", alias, result.Err)
		}
	}
}
