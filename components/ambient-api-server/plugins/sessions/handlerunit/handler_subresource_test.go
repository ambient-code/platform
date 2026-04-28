package handlerunit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	. "github.com/ambient-code/platform/components/ambient-api-server/plugins/sessions"
)

// ---------------------------------------------------------------------------
// Minimal harness — no DB, no rh-trex-ai env, no sqlmock.
// ---------------------------------------------------------------------------

func setupSessionRouter(svc SessionService) *mux.Router {
	return setupFullRouter(svc)
}

// setupFullRouter builds a mux with all session sub-resource routes including AGUI.
func setupFullRouter(svc SessionService) *mux.Router {
	r := mux.NewRouter()
	h := NewSessionHandler(svc, nil, nil)

	base := "/api/ambient/v1/sessions"
	r.HandleFunc(base, h.List).Methods(http.MethodGet)
	r.HandleFunc(base, h.Create).Methods(http.MethodPost)
	r.HandleFunc(base+"/{id}", h.Get).Methods(http.MethodGet)
	r.HandleFunc(base+"/{id}", h.Patch).Methods(http.MethodPatch)
	r.HandleFunc(base+"/{id}", h.Delete).Methods(http.MethodDelete)
	r.HandleFunc(base+"/{id}/start", h.Start).Methods(http.MethodPost)
	r.HandleFunc(base+"/{id}/stop", h.Stop).Methods(http.MethodPost)
	r.HandleFunc(base+"/{id}/clone", h.Clone).Methods(http.MethodPost)
	r.HandleFunc(base+"/{id}/repos", h.AddRepo).Methods(http.MethodPost)
	r.HandleFunc(base+"/{id}/repos/{repoName}", h.RemoveRepo).Methods(http.MethodDelete)
	r.HandleFunc(base+"/{id}/workflow", h.SetWorkflow).Methods(http.MethodPost)
	r.HandleFunc(base+"/{id}/model", h.SetModel).Methods(http.MethodPost)
	r.HandleFunc(base+"/{id}/agui/events", h.AGUIEvents).Methods(http.MethodGet)
	r.HandleFunc(base+"/{id}/agui/run", h.AGUIRun).Methods(http.MethodPost)
	r.HandleFunc(base+"/{id}/agui/interrupt", h.AGUIInterrupt).Methods(http.MethodPost)
	r.HandleFunc(base+"/{id}/agui/feedback", h.AGUIFeedback).Methods(http.MethodPost)
	r.HandleFunc(base+"/{id}/agui/tasks", h.AGUITasks).Methods(http.MethodGet)
	r.HandleFunc(base+"/{id}/agui/tasks/{taskId}/stop", h.AGUITaskStop).Methods(http.MethodPost)
	r.HandleFunc(base+"/{id}/agui/tasks/{taskId}/output", h.AGUITaskOutput).Methods(http.MethodGet)
	r.HandleFunc(base+"/{id}/agui/capabilities", h.AGUICapabilities).Methods(http.MethodGet)
	r.HandleFunc(base+"/{id}/mcp/status", h.MCPStatus).Methods(http.MethodGet)
	// Workspace file proxy
	r.HandleFunc(base+"/{id}/workspace", h.WorkspaceList).Methods(http.MethodGet)
	r.HandleFunc(base+"/{id}/workspace/{path:.*}", h.WorkspaceFile).Methods(http.MethodGet, http.MethodPut, http.MethodDelete)
	// Pre-upload file proxy
	r.HandleFunc(base+"/{id}/files", h.FilesList).Methods(http.MethodGet)
	r.HandleFunc(base+"/{id}/files/{path:.*}", h.FilesFile).Methods(http.MethodPut, http.MethodDelete)
	// Git proxy
	r.HandleFunc(base+"/{id}/git/status", h.GitStatus).Methods(http.MethodGet)
	r.HandleFunc(base+"/{id}/git/configure-remote", h.GitConfigureRemote).Methods(http.MethodPost)
	r.HandleFunc(base+"/{id}/git/branches", h.GitBranches).Methods(http.MethodGet)
	// Repos status proxy
	r.HandleFunc(base+"/{id}/repos/status", h.ReposStatus).Methods(http.MethodGet)
	// Pod events
	r.HandleFunc(base+"/{id}/pod-events", h.PodEvents).Methods(http.MethodGet)
	// Operational sub-resources
	r.HandleFunc(base+"/{id}/displayname", h.PatchDisplayName).Methods(http.MethodPatch)
	r.HandleFunc(base+"/{id}/workflow/metadata", h.WorkflowMetadata).Methods(http.MethodGet)
	r.HandleFunc(base+"/{id}/oauth/{provider}/url", h.OAuthProviderURL).Methods(http.MethodGet)
	r.HandleFunc(base+"/{id}/export", h.ExportSession).Methods(http.MethodGet)
	return r
}

func seedSession(t *testing.T, svc SessionService) *Session {
	t.Helper()
	proj := "proj-1"
	sess, err := svc.Create(context.Background(), &Session{
		Name:      "test-session",
		ProjectId: &proj,
	})
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
	return sess
}

func jsonReq(t *testing.T, method, url string, v interface{}) *http.Request {
	t.Helper()
	var body *bytes.Buffer
	if v != nil {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.NewBuffer(b)
	} else {
		body = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, url, body)
	if v != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func decodeSession(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("json decode: %v — body: %s", err, body)
	}
	return m
}

// ---------------------------------------------------------------------------
// Clone
// ---------------------------------------------------------------------------

func TestClone_Success(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupSessionRouter(svc)

	src := seedSession(t, svc)

	req := jsonReq(t, http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/clone", src.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("clone: expected 201, got %d: %s", rr.Code, rr.Body)
	}
	m := decodeSession(t, rr.Body.Bytes())
	cloneID, _ := m["id"].(string)
	if cloneID == "" || cloneID == src.ID {
		t.Error("expected a new non-empty id different from source")
	}
	// parent_session_id should point back to source
	if m["parent_session_id"] != src.ID {
		t.Errorf("expected parent_session_id=%s, got %v", src.ID, m["parent_session_id"])
	}
	// name should be "<original>-clone"
	if m["name"] != src.Name+"-clone" {
		t.Errorf("expected name=%s-clone, got %v", src.Name, m["name"])
	}
}

func TestClone_NotFound(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupSessionRouter(svc)

	req := jsonReq(t, http.MethodPost, "/api/ambient/v1/sessions/nonexistent/clone", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// AddRepo
// ---------------------------------------------------------------------------

func TestAddRepo_Success(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupSessionRouter(svc)

	src := seedSession(t, svc)

	body := AddRepoRequest{URL: "https://github.com/org/my-repo.git", Branch: "develop"}
	req := jsonReq(t, http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/repos", src.ID), body)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("add repo: expected 200, got %d: %s", rr.Code, rr.Body)
	}
	m := decodeSession(t, rr.Body.Bytes())
	reposRaw, _ := m["repos"].(string)
	if reposRaw == "" {
		t.Fatal("expected repos field to be set")
	}
	var repos []RepoEntry
	if err := json.Unmarshal([]byte(reposRaw), &repos); err != nil {
		t.Fatalf("repos json: %v", err)
	}
	if len(repos) != 1 {
		t.Errorf("expected 1 repo, got %d", len(repos))
	}
	if repos[0].URL != body.URL {
		t.Errorf("repo url mismatch: %s", repos[0].URL)
	}
	if repos[0].Branch != "develop" {
		t.Errorf("repo branch mismatch: %s", repos[0].Branch)
	}
	if repos[0].Name != "my-repo" {
		t.Errorf("repo name mismatch: %s", repos[0].Name)
	}
}

func TestAddRepo_DefaultBranch(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupSessionRouter(svc)
	src := seedSession(t, svc)

	body := AddRepoRequest{URL: "https://github.com/org/repo.git"}
	req := jsonReq(t, http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/repos", src.ID), body)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	m := decodeSession(t, rr.Body.Bytes())
	var repos []RepoEntry
	json.Unmarshal([]byte(m["repos"].(string)), &repos)
	if repos[0].Branch != "main" {
		t.Errorf("expected default branch=main, got %s", repos[0].Branch)
	}
}

func TestAddRepo_MissingURL(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupSessionRouter(svc)
	src := seedSession(t, svc)

	body := AddRepoRequest{}
	req := jsonReq(t, http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/repos", src.ID), body)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestAddRepo_Accumulates(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupSessionRouter(svc)
	src := seedSession(t, svc)

	for i, url := range []string{
		"https://github.com/org/repo-a.git",
		"https://github.com/org/repo-b.git",
	} {
		req := jsonReq(t, http.MethodPost,
			fmt.Sprintf("/api/ambient/v1/sessions/%s/repos", src.ID),
			AddRepoRequest{URL: url})
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("add repo %d: expected 200, got %d", i, rr.Code)
		}
	}

	sess, err := svc.Get(context.Background(), src.ID)
	if err != nil {
		t.Fatal(err)
	}
	var repos []RepoEntry
	json.Unmarshal([]byte(*sess.Repos), &repos)
	if len(repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(repos))
	}
}

// ---------------------------------------------------------------------------
// RemoveRepo
// ---------------------------------------------------------------------------

func TestRemoveRepo_Success(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupSessionRouter(svc)
	src := seedSession(t, svc)

	// Add two repos
	for _, url := range []string{
		"https://github.com/org/keep.git",
		"https://github.com/org/remove-me.git",
	} {
		req := jsonReq(t, http.MethodPost,
			fmt.Sprintf("/api/ambient/v1/sessions/%s/repos", src.ID),
			AddRepoRequest{URL: url})
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("setup add repo: %d", rr.Code)
		}
	}

	// Remove one
	req := httptest.NewRequest(http.MethodDelete,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/repos/remove-me", src.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("remove repo: expected 200, got %d: %s", rr.Code, rr.Body)
	}
	sess, _ := svc.Get(context.Background(), src.ID)
	var repos []RepoEntry
	json.Unmarshal([]byte(*sess.Repos), &repos)
	if len(repos) != 1 {
		t.Errorf("expected 1 repo after removal, got %d", len(repos))
	}
	if repos[0].Name != "keep" {
		t.Errorf("wrong repo remaining: %s", repos[0].Name)
	}
}

// ---------------------------------------------------------------------------
// SetWorkflow
// ---------------------------------------------------------------------------

func TestSetWorkflow_Success(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupSessionRouter(svc)
	src := seedSession(t, svc)

	body := SetWorkflowRequest{GitURL: "https://github.com/org/workflow.git", Branch: "feature"}
	req := jsonReq(t, http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/workflow", src.ID), body)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("set workflow: expected 200, got %d: %s", rr.Code, rr.Body)
	}
	sess, _ := svc.Get(context.Background(), src.ID)
	if sess.WorkflowId == nil || *sess.WorkflowId == "" {
		t.Fatal("expected workflow_id to be set")
	}
	var wf SetWorkflowRequest
	if err := json.Unmarshal([]byte(*sess.WorkflowId), &wf); err != nil {
		t.Fatalf("workflow json: %v", err)
	}
	if wf.GitURL != body.GitURL {
		t.Errorf("git_url mismatch: %s", wf.GitURL)
	}
	if wf.Branch != "feature" {
		t.Errorf("branch mismatch: %s", wf.Branch)
	}
}

func TestSetWorkflow_DefaultBranch(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupSessionRouter(svc)
	src := seedSession(t, svc)

	body := SetWorkflowRequest{GitURL: "https://github.com/org/workflow.git"}
	req := jsonReq(t, http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/workflow", src.ID), body)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	sess, _ := svc.Get(context.Background(), src.ID)
	var wf SetWorkflowRequest
	json.Unmarshal([]byte(*sess.WorkflowId), &wf)
	if wf.Branch != "main" {
		t.Errorf("expected default branch=main, got %s", wf.Branch)
	}
}

func TestSetWorkflow_MissingGitURL(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupSessionRouter(svc)
	src := seedSession(t, svc)

	body := SetWorkflowRequest{}
	req := jsonReq(t, http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/workflow", src.ID), body)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// SetModel
// ---------------------------------------------------------------------------

func TestSetModel_Success(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupSessionRouter(svc)
	src := seedSession(t, svc)

	body := SetModelRequest{Model: "claude-opus-4-7"}
	req := jsonReq(t, http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/model", src.ID), body)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("set model: expected 200, got %d: %s", rr.Code, rr.Body)
	}
	m := decodeSession(t, rr.Body.Bytes())
	if m["llm_model"] != "claude-opus-4-7" {
		t.Errorf("expected llm_model=claude-opus-4-7, got %v", m["llm_model"])
	}
}

func TestSetModel_MissingModel(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupSessionRouter(svc)
	src := seedSession(t, svc)

	body := SetModelRequest{}
	req := jsonReq(t, http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/sessions/%s/model", src.ID), body)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestSetModel_NotFound(t *testing.T) {
	svc := NewInMemorySessionService()
	router := setupSessionRouter(svc)

	body := SetModelRequest{Model: "claude-sonnet-4-6"}
	req := jsonReq(t, http.MethodPost, "/api/ambient/v1/sessions/bad-id/model", body)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}
