package delete

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ambient-code/platform/components/ambient-cli/internal/testhelper"
)

func TestDeleteProject_Success(t *testing.T) {
	deleted := false
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects/my-project", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "project", "my-project", "--yes")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !deleted {
		t.Error("expected DELETE request to be made")
	}
	if !strings.Contains(result.Stdout, "deleted") {
		t.Errorf("expected 'deleted' in output, got: %s", result.Stdout)
	}
}

func TestDeleteProject_Aliases(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects/p1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	for _, alias := range []string{"project", "projects", "proj"} {
		testhelper.Configure(t, srv.URL)
		result := testhelper.Run(t, Cmd, alias, "p1", "--yes")
		if result.Err != nil {
			t.Errorf("alias %q: unexpected error: %v", alias, result.Err)
		}
	}
}

func TestDeleteSession_Success(t *testing.T) {
	deleted := false
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/sessions/s1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "session", "s1", "--yes")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !deleted {
		t.Error("expected DELETE request to be made")
	}
	if !strings.Contains(result.Stdout, "deleted") {
		t.Errorf("expected 'deleted' in output, got: %s", result.Stdout)
	}
}

func TestDeleteSession_Aliases(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/sessions/s1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	for _, alias := range []string{"session", "sessions", "sess"} {
		testhelper.Configure(t, srv.URL)
		result := testhelper.Run(t, Cmd, alias, "s1", "--yes")
		if result.Err != nil {
			t.Errorf("alias %q: unexpected error: %v", alias, result.Err)
		}
	}
}

func TestDeleteProjectSettings_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/project_settings/ps1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "project-settings", "ps1", "--yes")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "deleted") {
		t.Errorf("expected 'deleted' in output, got: %s", result.Stdout)
	}
}

func TestDeleteUnknownResource(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "foobar", "x", "--yes")
	if result.Err == nil {
		t.Fatal("expected error for unknown resource type")
	}
	if !strings.Contains(result.Err.Error(), "unknown") {
		t.Errorf("expected 'unknown' in error, got: %v", result.Err)
	}
}

func TestDeleteAbortedWithoutYes(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)

	result := testhelper.Run(t, Cmd, "project", "my-project")
	if result.Err == nil && strings.Contains(result.Stdout, "deleted") {
		t.Fatal("expected abort or error without --yes and no stdin")
	}
}
