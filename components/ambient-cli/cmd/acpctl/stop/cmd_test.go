package stop

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ambient-code/platform/components/ambient-cli/internal/testhelper"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

func TestStop_Success(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/sessions/s1/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		srv.RespondJSON(t, w, http.StatusOK, &types.Session{
			ObjectReference: types.ObjectReference{ID: "s1"},
			Name:            "my-session",
			Phase:           "completed",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "s1")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v\nstdout: %s\nstderr: %s", result.Err, result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "stopped") {
		t.Errorf("expected 'stopped' in output, got: %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "completed") {
		t.Errorf("expected 'completed' phase in output, got: %s", result.Stdout)
	}
}

func TestStop_NotFound(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/sessions/missing/stop", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusNotFound, &types.APIError{
			Code:   "not_found",
			Reason: "session not found",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "missing")
	if result.Err == nil {
		t.Fatal("expected error for missing session")
	}
	if !strings.Contains(result.Err.Error(), "stop session") {
		t.Errorf("expected 'stop session' in error, got: %v", result.Err)
	}
}

func TestStop_RequiresArg(t *testing.T) {
	srv := testhelper.NewServer(t)
	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd)
	if result.Err == nil {
		t.Fatal("expected error for missing session ID argument")
	}
}

func TestStop_OutputContainsID(t *testing.T) {
	srv := testhelper.NewServer(t)
	srv.Handle("/api/ambient/v1/sessions/abc-123/stop", func(w http.ResponseWriter, r *http.Request) {
		srv.RespondJSON(t, w, http.StatusOK, &types.Session{
			ObjectReference: types.ObjectReference{ID: "abc-123"},
			Phase:           "failed",
		})
	})

	testhelper.Configure(t, srv.URL)
	result := testhelper.Run(t, Cmd, "abc-123")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !strings.Contains(result.Stdout, "abc-123") {
		t.Errorf("expected session ID in output, got: %s", result.Stdout)
	}
}
