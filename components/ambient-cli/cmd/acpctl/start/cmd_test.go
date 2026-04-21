package start

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ambient-code/platform/components/ambient-cli/internal/testhelper"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

func TestStart_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects/proj-1/agents/pa-1/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		srv.RespondJSON(t, w, http.StatusCreated, &types.StartResponse{
			Session: &types.Session{
				ObjectReference: types.ObjectReference{ID: "s1"},
				Name:            "my-session",
				Phase:           "running",
			},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "pa-1", "--project-id", "proj-1")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "started") {
		t.Errorf("expected 'started' in output, got: %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "running") {
		t.Errorf("expected 'running' phase in output, got: %s", result.Stdout)
	}
}

func TestStart_NotFound(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects/proj-1/agents/missing/start", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusNotFound, &types.APIError{
			Code:   "not_found",
			Reason: "project-agent not found",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "missing", "--project-id", "proj-1")
	if result.Err == nil {
		t.Fatal("expected error for missing project-agent")
	}
	if !strings.Contains(result.Err.Error(), "start agent") {
		t.Errorf("expected 'start agent' in error, got: %v", result.Err)
	}
}

func TestStart_RequiresArg(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd)
	if result.Err == nil {
		t.Fatal("expected error for missing project-agent ID argument")
	}
}

func TestStart_RequiresProjectID(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "pa-1")
	if result.Err == nil {
		t.Fatal("expected error for missing --project-id")
	}
	if !strings.Contains(result.Err.Error(), "--project-id is required") {
		t.Errorf("expected '--project-id is required', got: %v", result.Err)
	}
}

func TestStart_OutputContainsID(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/projects/proj-1/agents/pa-abc/start", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusCreated, &types.StartResponse{
			Session: &types.Session{
				ObjectReference: types.ObjectReference{ID: "abc-123"},
				Phase:           "pending",
			},
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "pa-abc", "--project-id", "proj-1")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, "abc-123") {
		t.Errorf("expected session ID in output, got: %s", result.Stdout)
	}
}
