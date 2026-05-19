package scheduledSessions_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	. "github.com/ambient-code/platform/components/ambient-api-server/plugins/scheduledSessions"
)

// ---------------------------------------------------------------------------
// Test harness
// ---------------------------------------------------------------------------

func setupRouter(svc ScheduledSessionService) *mux.Router {
	r := mux.NewRouter()
	h := NewScheduledSessionHandler(svc)

	sub := r.PathPrefix("/api/ambient/v1/projects/{project_id}/scheduled-sessions").Subrouter()
	sub.HandleFunc("", h.List).Methods(http.MethodGet)
	sub.HandleFunc("", h.Create).Methods(http.MethodPost)
	sub.HandleFunc("/{id}", h.Get).Methods(http.MethodGet)
	sub.HandleFunc("/{id}", h.Patch).Methods(http.MethodPatch)
	sub.HandleFunc("/{id}", h.Delete).Methods(http.MethodDelete)
	sub.HandleFunc("/{id}/suspend", h.Suspend).Methods(http.MethodPost)
	sub.HandleFunc("/{id}/resume", h.Resume).Methods(http.MethodPost)
	sub.HandleFunc("/{id}/trigger", h.Trigger).Methods(http.MethodPost)
	sub.HandleFunc("/{id}/runs", h.Runs).Methods(http.MethodGet)
	return r
}

func newSS(t *testing.T, svc ScheduledSessionService, projectId string) openapi.ScheduledSession {
	t.Helper()
	agentId := "agent-123"
	ss, err := svc.Create(context.Background(), &ScheduledSession{
		Name:      "daily-run",
		ProjectId: projectId,
		AgentId:   &agentId,
		Schedule:  "0 9 * * 1-5",
	})
	if err != nil {
		t.Fatalf("failed to seed scheduled session: %v", err)
	}
	return PresentScheduledSession(ss)
}

func jsonBody(t *testing.T, v interface{}) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewBuffer(b)
}

func decodeJSON(t *testing.T, body []byte, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(body, v); err != nil {
		t.Fatalf("json decode: %v — body: %s", err, body)
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestList_Empty(t *testing.T) {
	svc := NewInMemoryService()
	router := setupRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/ambient/v1/projects/proj-1/scheduled-sessions", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var list openapi.ScheduledSessionList
	decodeJSON(t, rr.Body.Bytes(), &list)
	if list.Total != 0 {
		t.Errorf("expected 0 items, got %d", list.Total)
	}
}

func TestCreate_Success(t *testing.T) {
	svc := NewInMemoryService()
	router := setupRouter(svc)

	agentId := "agent-abc"
	body := openapi.ScheduledSession{
		Name:          "nightly",
		AgentId:       &agentId,
		Schedule:      "0 22 * * *",
		SessionPrompt: strPtr("run nightly analysis"),
	}
	req := httptest.NewRequest(http.MethodPost,
		"/api/ambient/v1/projects/proj-1/scheduled-sessions",
		jsonBody(t, body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body)
	}
	var ss openapi.ScheduledSession
	decodeJSON(t, rr.Body.Bytes(), &ss)
	if ss.Id == nil || *ss.Id == "" {
		t.Error("expected non-empty id")
	}
	if ss.Name != "nightly" {
		t.Errorf("name mismatch: %s", ss.Name)
	}
	if *ss.Kind != "ScheduledSession" {
		t.Errorf("kind mismatch: %s", *ss.Kind)
	}
	if ss.ProjectId != "proj-1" {
		t.Errorf("project_id mismatch: %s", ss.ProjectId)
	}
}

func TestCreate_MissingRequiredFields(t *testing.T) {
	svc := NewInMemoryService()
	router := setupRouter(svc)

	cases := []struct {
		name string
		body openapi.ScheduledSession
	}{
		{"missing name", openapi.ScheduledSession{Schedule: "* * * * *"}},
		{"missing schedule", openapi.ScheduledSession{Name: "x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost,
				"/api/ambient/v1/projects/proj-1/scheduled-sessions",
				jsonBody(t, tc.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", rr.Code)
			}
		})
	}
}

func TestCreate_RejectsClientSuppliedId(t *testing.T) {
	svc := NewInMemoryService()
	router := setupRouter(svc)

	id := "client-provided-id"
	body := openapi.ScheduledSession{
		Id:       &id,
		Name:     "x",
		Schedule: "* * * * *",
	}
	req := httptest.NewRequest(http.MethodPost,
		"/api/ambient/v1/projects/proj-1/scheduled-sessions",
		jsonBody(t, body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for client-supplied id, got %d", rr.Code)
	}
}

func TestGet_Found(t *testing.T) {
	svc := NewInMemoryService()
	router := setupRouter(svc)

	ss := newSS(t, svc, "proj-1")

	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/projects/proj-1/scheduled-sessions/%s", *ss.Id), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var got openapi.ScheduledSession
	decodeJSON(t, rr.Body.Bytes(), &got)
	if *got.Id != *ss.Id {
		t.Errorf("id mismatch: got %s want %s", *got.Id, *ss.Id)
	}
}

func TestGet_NotFound(t *testing.T) {
	svc := NewInMemoryService()
	router := setupRouter(svc)

	req := httptest.NewRequest(http.MethodGet,
		"/api/ambient/v1/projects/proj-1/scheduled-sessions/nonexistent", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestPatch_UpdateFields(t *testing.T) {
	svc := NewInMemoryService()
	router := setupRouter(svc)

	ss := newSS(t, svc, "proj-1")
	newSched := "0 6 * * *"
	patch := openapi.ScheduledSessionPatchRequest{Schedule: &newSched}

	req := httptest.NewRequest(http.MethodPatch,
		fmt.Sprintf("/api/ambient/v1/projects/proj-1/scheduled-sessions/%s", *ss.Id),
		jsonBody(t, patch))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var updated openapi.ScheduledSession
	decodeJSON(t, rr.Body.Bytes(), &updated)
	if updated.Schedule != newSched {
		t.Errorf("schedule not updated: got %s want %s", updated.Schedule, newSched)
	}
}

func TestDelete_Success(t *testing.T) {
	svc := NewInMemoryService()
	router := setupRouter(svc)

	ss := newSS(t, svc, "proj-1")

	req := httptest.NewRequest(http.MethodDelete,
		fmt.Sprintf("/api/ambient/v1/projects/proj-1/scheduled-sessions/%s", *ss.Id), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}

	// Verify it's gone
	req2 := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/projects/proj-1/scheduled-sessions/%s", *ss.Id), nil)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", rr2.Code)
	}
}

func TestDelete_NotFound(t *testing.T) {
	svc := NewInMemoryService()
	router := setupRouter(svc)

	req := httptest.NewRequest(http.MethodDelete,
		"/api/ambient/v1/projects/proj-1/scheduled-sessions/nonexistent", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestSuspend_Resume(t *testing.T) {
	svc := NewInMemoryService()
	router := setupRouter(svc)

	// Create enabled=true by default
	ss := newSS(t, svc, "proj-1")
	enabled := true
	_, _ = svc.Patch(context.Background(), *ss.Id, &ScheduledSessionPatch{Enabled: &enabled})

	// Suspend
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/projects/proj-1/scheduled-sessions/%s/suspend", *ss.Id), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("suspend: expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var suspended openapi.ScheduledSession
	decodeJSON(t, rr.Body.Bytes(), &suspended)
	if *suspended.Enabled {
		t.Error("expected enabled=false after suspend")
	}

	// Resume
	req2 := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/projects/proj-1/scheduled-sessions/%s/resume", *ss.Id), nil)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("resume: expected 200, got %d", rr2.Code)
	}
	var resumed openapi.ScheduledSession
	decodeJSON(t, rr2.Body.Bytes(), &resumed)
	if !*resumed.Enabled {
		t.Error("expected enabled=true after resume")
	}
}

func TestTrigger_Success(t *testing.T) {
	svc := NewInMemoryService()
	router := setupRouter(svc)

	ss := newSS(t, svc, "proj-1")

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/projects/proj-1/scheduled-sessions/%s/trigger", *ss.Id), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
}

func TestTrigger_NotFound(t *testing.T) {
	svc := NewInMemoryService()
	router := setupRouter(svc)

	req := httptest.NewRequest(http.MethodPost,
		"/api/ambient/v1/projects/proj-1/scheduled-sessions/bad-id/trigger", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestRuns_ReturnsEmptyList(t *testing.T) {
	svc := NewInMemoryService()
	router := setupRouter(svc)

	ss := newSS(t, svc, "proj-1")

	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/projects/proj-1/scheduled-sessions/%s/runs", *ss.Id), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var result map[string]interface{}
	decodeJSON(t, rr.Body.Bytes(), &result)
	if result["kind"] != "SessionList" {
		t.Errorf("expected kind=SessionList, got %v", result["kind"])
	}
}

func TestList_ProjectIsolation(t *testing.T) {
	svc := NewInMemoryService()
	router := setupRouter(svc)

	// Create sessions in two different projects
	_ = newSS(t, svc, "proj-A")
	_ = newSS(t, svc, "proj-A")
	_ = newSS(t, svc, "proj-B")

	req := httptest.NewRequest(http.MethodGet,
		"/api/ambient/v1/projects/proj-A/scheduled-sessions", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var list openapi.ScheduledSessionList
	decodeJSON(t, rr.Body.Bytes(), &list)
	if list.Total != 2 {
		t.Errorf("expected 2 items for proj-A, got %d", list.Total)
	}
}

func TestFullCRUDLifecycle(t *testing.T) {
	svc := NewInMemoryService()
	router := setupRouter(svc)
	projectId := "lifecycle-proj"

	// Create
	agentId := "agent-1"
	body := openapi.ScheduledSession{
		Name:     "lifecycle-test",
		AgentId:  &agentId,
		Schedule: "*/5 * * * *",
	}
	createReq := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/projects/%s/scheduled-sessions", projectId),
		jsonBody(t, body))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", createRR.Code, createRR.Body)
	}
	var created openapi.ScheduledSession
	decodeJSON(t, createRR.Body.Bytes(), &created)
	id := *created.Id

	// List — should contain 1
	listReq := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/projects/%s/scheduled-sessions", projectId), nil)
	listRR := httptest.NewRecorder()
	router.ServeHTTP(listRR, listReq)
	var list openapi.ScheduledSessionList
	decodeJSON(t, listRR.Body.Bytes(), &list)
	if list.Total != 1 {
		t.Errorf("expected 1 after create, got %d", list.Total)
	}

	// Get
	getReq := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/projects/%s/scheduled-sessions/%s", projectId, id), nil)
	getRR := httptest.NewRecorder()
	router.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", getRR.Code)
	}

	// Patch
	newName := "lifecycle-test-updated"
	patchReq := httptest.NewRequest(http.MethodPatch,
		fmt.Sprintf("/api/ambient/v1/projects/%s/scheduled-sessions/%s", projectId, id),
		jsonBody(t, openapi.ScheduledSessionPatchRequest{Name: &newName}))
	patchReq.Header.Set("Content-Type", "application/json")
	patchRR := httptest.NewRecorder()
	router.ServeHTTP(patchRR, patchReq)
	if patchRR.Code != http.StatusOK {
		t.Fatalf("patch: expected 200, got %d", patchRR.Code)
	}
	var patched openapi.ScheduledSession
	decodeJSON(t, patchRR.Body.Bytes(), &patched)
	if patched.Name != newName {
		t.Errorf("name not updated: got %s", patched.Name)
	}

	// Suspend → Resume
	suspendReq := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/projects/%s/scheduled-sessions/%s/suspend", projectId, id), nil)
	suspendRR := httptest.NewRecorder()
	router.ServeHTTP(suspendRR, suspendReq)
	if suspendRR.Code != http.StatusOK {
		t.Fatalf("suspend: expected 200, got %d", suspendRR.Code)
	}

	resumeReq := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/projects/%s/scheduled-sessions/%s/resume", projectId, id), nil)
	resumeRR := httptest.NewRecorder()
	router.ServeHTTP(resumeRR, resumeReq)
	if resumeRR.Code != http.StatusOK {
		t.Fatalf("resume: expected 200, got %d", resumeRR.Code)
	}

	// Trigger
	triggerReq := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/ambient/v1/projects/%s/scheduled-sessions/%s/trigger", projectId, id), nil)
	triggerRR := httptest.NewRecorder()
	router.ServeHTTP(triggerRR, triggerReq)
	if triggerRR.Code != http.StatusOK {
		t.Fatalf("trigger: expected 200, got %d", triggerRR.Code)
	}

	// Runs
	runsReq := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/projects/%s/scheduled-sessions/%s/runs", projectId, id), nil)
	runsRR := httptest.NewRecorder()
	router.ServeHTTP(runsRR, runsReq)
	if runsRR.Code != http.StatusOK {
		t.Fatalf("runs: expected 200, got %d", runsRR.Code)
	}

	// Delete
	delReq := httptest.NewRequest(http.MethodDelete,
		fmt.Sprintf("/api/ambient/v1/projects/%s/scheduled-sessions/%s", projectId, id), nil)
	delRR := httptest.NewRecorder()
	router.ServeHTTP(delRR, delReq)
	if delRR.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", delRR.Code)
	}

	// List — should be 0 again
	listReq2 := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/ambient/v1/projects/%s/scheduled-sessions", projectId), nil)
	listRR2 := httptest.NewRecorder()
	router.ServeHTTP(listRR2, listReq2)
	var list2 openapi.ScheduledSessionList
	decodeJSON(t, listRR2.Body.Bytes(), &list2)
	if list2.Total != 0 {
		t.Errorf("expected 0 after delete, got %d", list2.Total)
	}
}

func TestCreate_WithoutAgentId(t *testing.T) {
	svc := NewInMemoryService()
	router := setupRouter(svc)

	body := openapi.ScheduledSession{
		Name:     "no-agent",
		Schedule: "0 9 * * *",
	}
	req := httptest.NewRequest(http.MethodPost,
		"/api/ambient/v1/projects/proj-1/scheduled-sessions",
		jsonBody(t, body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body)
	}
	var ss openapi.ScheduledSession
	decodeJSON(t, rr.Body.Bytes(), &ss)
	if ss.AgentId != nil {
		t.Errorf("expected nil agent_id, got %s", *ss.AgentId)
	}
}

func TestCreate_WithExecutionFields(t *testing.T) {
	svc := NewInMemoryService()
	router := setupRouter(svc)

	timeout := int32(3600)
	inactivityTimeout := int32(600)
	stopOnRunFinished := true
	runnerType := "claude-code"
	agentId := "agent-exec"
	body := openapi.ScheduledSession{
		Name:              "exec-fields",
		AgentId:           &agentId,
		Schedule:          "0 12 * * *",
		Timeout:           &timeout,
		InactivityTimeout: &inactivityTimeout,
		StopOnRunFinished: &stopOnRunFinished,
		RunnerType:        &runnerType,
	}
	req := httptest.NewRequest(http.MethodPost,
		"/api/ambient/v1/projects/proj-1/scheduled-sessions",
		jsonBody(t, body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body)
	}
	var ss openapi.ScheduledSession
	decodeJSON(t, rr.Body.Bytes(), &ss)
	if ss.Timeout == nil || *ss.Timeout != 3600 {
		t.Errorf("timeout mismatch: got %v", ss.Timeout)
	}
	if ss.InactivityTimeout == nil || *ss.InactivityTimeout != 600 {
		t.Errorf("inactivity_timeout mismatch: got %v", ss.InactivityTimeout)
	}
	if ss.StopOnRunFinished == nil || !*ss.StopOnRunFinished {
		t.Errorf("stop_on_run_finished mismatch: got %v", ss.StopOnRunFinished)
	}
	if ss.RunnerType == nil || *ss.RunnerType != "claude-code" {
		t.Errorf("runner_type mismatch: got %v", ss.RunnerType)
	}
}

func TestPatch_ExecutionFields(t *testing.T) {
	svc := NewInMemoryService()
	router := setupRouter(svc)

	ss := newSS(t, svc, "proj-1")

	timeout := int32(7200)
	runnerType := "custom-runner"
	patch := openapi.ScheduledSessionPatchRequest{
		Timeout:    &timeout,
		RunnerType: &runnerType,
	}
	req := httptest.NewRequest(http.MethodPatch,
		fmt.Sprintf("/api/ambient/v1/projects/proj-1/scheduled-sessions/%s", *ss.Id),
		jsonBody(t, patch))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var updated openapi.ScheduledSession
	decodeJSON(t, rr.Body.Bytes(), &updated)
	if updated.Timeout == nil || *updated.Timeout != 7200 {
		t.Errorf("timeout not patched: got %v", updated.Timeout)
	}
	if updated.RunnerType == nil || *updated.RunnerType != "custom-runner" {
		t.Errorf("runner_type not patched: got %v", updated.RunnerType)
	}
	if updated.Schedule != "0 9 * * 1-5" {
		t.Errorf("schedule should be unchanged: got %s", updated.Schedule)
	}
}

func strPtr(s string) *string { return &s }
