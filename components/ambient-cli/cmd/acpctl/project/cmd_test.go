package project

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ambient-code/platform/components/ambient-cli/internal/testhelper"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

func TestProjectCurrent_NoProject(t *testing.T) {
	t.Setenv("AMBIENT_PROJECT", "")
	dir := t.TempDir()
	t.Setenv("AMBIENT_CONFIG", dir+"/config.json")

	result := testhelper.Run(t, Cmd, "current")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, "No project context set") {
		t.Errorf("expected 'No project context set', got: %s", result.Stdout)
	}
}

func TestProjectCurrent_WithProject(t *testing.T) {
	t.Setenv("AMBIENT_PROJECT", "my-project")
	result := testhelper.Run(t, Cmd, "current")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, "my-project") {
		t.Errorf("expected 'my-project' in output, got: %s", result.Stdout)
	}
}

func TestProjectSet_ValidProject(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects/new-project", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.Project{
			ObjectReference: types.ObjectReference{ID: "p1"},
			Name:            "new-project",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "set", "new-project")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "new-project") {
		t.Errorf("expected 'new-project' in output, got: %s", result.Stdout)
	}
}

func TestProjectSet_Shorthand(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects/my-proj", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.Project{
			ObjectReference: types.ObjectReference{ID: "p1"},
			Name:            "my-proj",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "my-proj")
	if result.Err != nil {
		t.Fatalf("shorthand: unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "my-proj") {
		t.Errorf("expected 'my-proj' in output, got: %s", result.Stdout)
	}
}

func TestProjectSet_NotFound(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects/nonexistent", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusNotFound, &types.APIError{
			Code:   "not_found",
			Reason: "project not found",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "set", "nonexistent")
	if result.Err == nil {
		t.Fatal("expected error for non-existent project")
	}
	if !strings.Contains(result.Err.Error(), "nonexistent") {
		t.Errorf("expected project name in error, got: %v", result.Err)
	}
}

func TestProjectSet_InvalidName_Uppercase(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "set", "MyProject")
	if result.Err == nil {
		t.Fatal("expected error for uppercase project name")
	}
	if !strings.Contains(result.Err.Error(), "invalid project name") {
		t.Errorf("expected 'invalid project name', got: %v", result.Err)
	}
}

func TestProjectSet_InvalidName_Spaces(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "set", "my project")
	if result.Err == nil {
		t.Fatal("expected error for project name with spaces")
	}
}

func TestProjectSet_InvalidName_LeadingHyphen(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "set", "-bad-name")
	if result.Err == nil {
		t.Fatal("expected error for leading hyphen in project name")
	}
}

func TestProjectList(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.ProjectList{
			Items: []types.Project{
				{ObjectReference: types.ObjectReference{ID: "p1"}, Name: "alpha", Description: "Alpha project"},
				{ObjectReference: types.ObjectReference{ID: "p2"}, Name: "beta", Description: "Beta project"},
			},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "list")
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

func TestProjectList_Empty(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.ProjectList{Items: []types.Project{}})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "list")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, "No projects found") {
		t.Errorf("expected 'No projects found', got: %s", result.Stdout)
	}
}
