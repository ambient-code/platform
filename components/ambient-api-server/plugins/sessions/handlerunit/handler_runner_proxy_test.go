package handlerunit_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/ambient-code/platform/components/ambient-api-server/plugins/sessions"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func seedWithRunner(t *testing.T, svc SessionService) *Session {
	t.Helper()
	proj := "proj-runner"
	sess, err := svc.Create(t.Context(), &Session{
		Name:      "runner-session",
		ProjectId: &proj,
	})
	if err != nil {
		t.Fatalf("seed runner session: %v", err)
	}
	// Populate KubeCrName + KubeNamespace so runnerBaseURL returns a non-empty URL.
	crName := "test-runner"
	ns := "test-ns"
	sess.KubeCrName = &crName
	sess.KubeNamespace = &ns
	updated, err := svc.Replace(t.Context(), sess)
	if err != nil {
		t.Fatalf("set runner fields: %v", err)
	}
	return updated
}

// ---------------------------------------------------------------------------
// Workspace list
// ---------------------------------------------------------------------------

func TestWorkspaceList_NoRunner_ReturnsEmptyStub(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedRunnerlessSession(t, svc)

	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/workspace", sess.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	if rr.Body.String() != `{"files":[]}` {
		t.Errorf("unexpected body: %s", rr.Body)
	}
}

func TestWorkspaceList_SessionNotFound_Returns404(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/ambient/v1/sessions/bad-id/workspace", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Workspace file GET/PUT/DELETE
// ---------------------------------------------------------------------------

func TestWorkspaceFile_NoRunner_Returns503(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedRunnerlessSession(t, svc)

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method,
			fmt.Sprintf("/api/ambient/v1/sessions/%s/workspace/src/main.go", sess.ID), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("method %s: expected 503, got %d", method, rr.Code)
		}
	}
}

func TestWorkspaceFile_SessionNotFound_Returns404(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)

	req := httptest.NewRequest(http.MethodGet,
		"/api/ambient/v1/sessions/bad-id/workspace/foo.txt", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Files list
// ---------------------------------------------------------------------------

func TestFilesList_NoRunner_ReturnsEmptyStub(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedRunnerlessSession(t, svc)

	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/files", sess.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	if rr.Body.String() != `{"files":[]}` {
		t.Errorf("unexpected body: %s", rr.Body)
	}
}

// ---------------------------------------------------------------------------
// Files file PUT/DELETE
// ---------------------------------------------------------------------------

func TestFilesFile_NoRunner_Returns503(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedRunnerlessSession(t, svc)

	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method,
			fmt.Sprintf("/api/ambient/v1/sessions/%s/files/upload/doc.pdf", sess.ID), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("method %s: expected 503, got %d", method, rr.Code)
		}
	}
}

// ---------------------------------------------------------------------------
// Git status
// ---------------------------------------------------------------------------

func TestGitStatus_NoRunner_ReturnsEmptyStub(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedRunnerlessSession(t, svc)

	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/git/status", sess.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	if rr.Body.String() != `{"modified":[],"staged":[],"untracked":[]}` {
		t.Errorf("unexpected body: %s", rr.Body)
	}
}

func TestGitStatus_SessionNotFound_Returns404(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/ambient/v1/sessions/bad-id/git/status", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Git configure-remote
// ---------------------------------------------------------------------------

func TestGitConfigureRemote_NoRunner_Returns503(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedRunnerlessSession(t, svc)

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/git/configure-remote", sess.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Git branches
// ---------------------------------------------------------------------------

func TestGitBranches_NoRunner_ReturnsEmptyArray(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedRunnerlessSession(t, svc)

	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/git/branches", sess.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	if rr.Body.String() != `[]` {
		t.Errorf("unexpected body: %s", rr.Body)
	}
}

// ---------------------------------------------------------------------------
// Repos status
// ---------------------------------------------------------------------------

func TestReposStatus_NoRunner_ReturnsEmptyArray(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedRunnerlessSession(t, svc)

	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/repos/status", sess.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	if rr.Body.String() != `[]` {
		t.Errorf("unexpected body: %s", rr.Body)
	}
}

// ---------------------------------------------------------------------------
// Pod events (always stub)
// ---------------------------------------------------------------------------

func TestPodEvents_ReturnsEmptyArray(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)
	sess := seedRunnerlessSession(t, svc)

	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/pod-events", sess.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	if rr.Body.String() != `[]` {
		t.Errorf("unexpected body: %s", rr.Body)
	}
}

func TestPodEvents_SessionNotFound_Returns404(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupFullRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/ambient/v1/sessions/bad-id/pod-events", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}
