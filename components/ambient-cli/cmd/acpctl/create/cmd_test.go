package create

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ambient-code/platform/components/ambient-cli/internal/testhelper"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

func TestCreateProject_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		srv.RespondJSON(t, w, http.StatusCreated, &types.Project{
			ObjectReference: types.ObjectReference{ID: "p-new"},
			Name:            "my-project",
			DisplayName:     "My Project",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "project", "--name", "my-project", "--display-name", "My Project")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "project/p-new") {
		t.Errorf("expected 'project/p-new created', got: %s", result.Stdout)
	}
}

func TestCreateProject_MissingName(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "project")
	if result.Err == nil {
		t.Fatal("expected error for missing --name")
	}
	if !strings.Contains(result.Err.Error(), "--name is required") {
		t.Errorf("expected '--name is required', got: %v", result.Err)
	}
}

func TestCreateProject_JSON(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusCreated, &types.Project{
			ObjectReference: types.ObjectReference{ID: "p-json"},
			Name:            "json-project",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "project", "--name", "json-project", "-o", "json")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, `"json-project"`) {
		t.Errorf("expected JSON with 'json-project', got: %s", result.Stdout)
	}
}

func TestCreateAgent_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects/"+testhelper.TestProject+"/agents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		srv.RespondJSON(t, w, http.StatusCreated, &types.Agent{
			ObjectReference: types.ObjectReference{ID: "a-new"},
			Name:            "overlord",
			ProjectID:       "my-project",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "agent",
		"--name", "overlord",
		"--project-id", "my-project",
		"--owner-user-id", "user-1",
		"--prompt", "You coordinate the fleet",
		"--repo-url", "https://github.com/org/repo",
		"--model", "sonnet",
	)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "agent/a-new") {
		t.Errorf("expected 'agent/a-new created', got: %s", result.Stdout)
	}
}

func TestCreateAgent_MissingName(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "agent", "--project-id", "p1", "--owner-user-id", "u1")
	if result.Err == nil {
		t.Fatal("expected error for missing --name")
	}
	if !strings.Contains(result.Err.Error(), "--name is required") {
		t.Errorf("expected '--name is required', got: %v", result.Err)
	}
}

func TestCreateAgent_MissingProjectID(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "agent", "--name", "x", "--owner-user-id", "u1")
	if result.Err == nil {
		t.Fatal("expected error for missing --project-id")
	}
	if !strings.Contains(result.Err.Error(), "--project-id is required") {
		t.Errorf("expected '--project-id is required', got: %v", result.Err)
	}
}

func TestCreateAgent_MissingOwnerUserID(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "agent", "--name", "x", "--project-id", "p1")
	if result.Err == nil {
		t.Fatal("expected error for missing --owner-user-id")
	}
	if !strings.Contains(result.Err.Error(), "--owner-user-id is required") {
		t.Errorf("expected '--owner-user-id is required', got: %v", result.Err)
	}
}

func TestCreateSession_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		srv.RespondJSON(t, w, http.StatusCreated, &types.Session{
			ObjectReference: types.ObjectReference{ID: "s-new"},
			Name:            "my-session",
			ProjectID:       testhelper.TestProject,
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "session",
		"--name", "my-session",
		"--prompt", "Implement the feature",
		"--repo-url", "https://github.com/org/repo",
		"--model", "sonnet",
	)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "session/s-new") {
		t.Errorf("expected 'session/s-new created', got: %s", result.Stdout)
	}
}

func TestCreateSession_MissingName(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "session")
	if result.Err == nil {
		t.Fatal("expected error for missing --name")
	}
	if !strings.Contains(result.Err.Error(), "--name is required") {
		t.Errorf("expected '--name is required', got: %v", result.Err)
	}
}

func TestCreateRole_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/roles", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		srv.RespondJSON(t, w, http.StatusCreated, &types.Role{
			ObjectReference: types.ObjectReference{ID: "r-new"},
			Name:            "agent:runner",
			DisplayName:     "Agent Runner",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "role",
		"--name", "agent:runner",
		"--display-name", "Agent Runner",
		"--description", "Minimum viable pod credential",
	)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "role/r-new") {
		t.Errorf("expected 'role/r-new created', got: %s", result.Stdout)
	}
}

func TestCreateRole_MissingName(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "role")
	if result.Err == nil {
		t.Fatal("expected error for missing --name")
	}
}

func TestCreateRoleBinding_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/role_bindings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		srv.RespondJSON(t, w, http.StatusCreated, &types.RoleBinding{
			ObjectReference: types.ObjectReference{ID: "rb-new"},
			UserID:          "user-1",
			RoleID:          "r-1",
			Scope:           "project",
			ScopeID:         "my-project",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "role-binding",
		"--user-id", "user-1",
		"--role-id", "r-1",
		"--scope", "project",
		"--scope-id", "my-project",
	)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "role-binding/rb-new") {
		t.Errorf("expected 'role-binding/rb-new created', got: %s", result.Stdout)
	}
}

func TestCreateRoleBinding_MissingUserID(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "role-binding", "--role-id", "r1", "--scope", "global")
	if result.Err == nil {
		t.Fatal("expected error for missing --user-id")
	}
	if !strings.Contains(result.Err.Error(), "--user-id is required") {
		t.Errorf("expected '--user-id is required', got: %v", result.Err)
	}
}

func TestCreateRoleBinding_MissingScope(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "role-binding", "--user-id", "u1", "--role-id", "r1")
	if result.Err == nil {
		t.Fatal("expected error for missing --scope")
	}
	if !strings.Contains(result.Err.Error(), "--scope is required") {
		t.Errorf("expected '--scope is required', got: %v", result.Err)
	}
}

func TestCreateRoleBinding_Aliases(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/role_bindings", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusCreated, &types.RoleBinding{
			ObjectReference: types.ObjectReference{ID: "rb-1"},
			UserID:          "u1", RoleID: "r1", Scope: "global",
		})
	})

	for _, alias := range []string{"role-binding", "rolebinding", "rb"} {
		testhelper.Configure(t, srv.URL)
		result := testhelper.Run(t, Cmd, alias, "--user-id", "u1", "--role-id", "r1", "--scope", "global")
		if result.Err != nil {
			t.Errorf("alias %q: unexpected error: %v", alias, result.Err)
		}
	}
}

func TestCreateUnknownResource(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "foobar")
	if result.Err == nil {
		t.Fatal("expected error for unknown resource type")
	}
	if !strings.Contains(result.Err.Error(), "unknown resource type") {
		t.Errorf("expected 'unknown resource type', got: %v", result.Err)
	}
}
