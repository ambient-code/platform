package informer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	openapi "github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/rs/zerolog"
)

func strPtr(s string) *string { return &s }

func timePtr(t time.Time) *time.Time { return &t }

func makeSession(id, name string, updatedAt time.Time) openapi.Session {
	s := *openapi.NewSession(name)
	s.SetId(id)
	s.SetUpdatedAt(updatedAt)
	return s
}

type fakeAPIHandler struct {
	mu              sync.Mutex
	sessions        *openapi.SessionList
	workflows       *openapi.WorkflowList
	tasks           *openapi.TaskList
	projects        *openapi.ProjectList
	projectSettings *openapi.ProjectSettingsList
}

func (h *fakeAPIHandler) setSessions(list *openapi.SessionList) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sessions = list
}

func (h *fakeAPIHandler) setWorkflows(list *openapi.WorkflowList) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.workflows = list
}

func (h *fakeAPIHandler) setTasks(list *openapi.TaskList) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.tasks = list
}

func (h *fakeAPIHandler) setProjects(list *openapi.ProjectList) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.projects = list
}

func (h *fakeAPIHandler) setProjectSettings(list *openapi.ProjectSettingsList) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.projectSettings = list
}

func (h *fakeAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	switch r.URL.Path {
	case "/api/ambient-api-server/v1/sessions":
		if h.sessions != nil {
			json.NewEncoder(w).Encode(h.sessions)
		} else {
			json.NewEncoder(w).Encode(openapi.NewSessionList("SessionList", 1, 0, 0, []openapi.Session{}))
		}
	case "/api/ambient-api-server/v1/workflows":
		if h.workflows != nil {
			json.NewEncoder(w).Encode(h.workflows)
		} else {
			json.NewEncoder(w).Encode(openapi.NewWorkflowList("WorkflowList", 1, 0, 0, []openapi.Workflow{}))
		}
	case "/api/ambient-api-server/v1/tasks":
		if h.tasks != nil {
			json.NewEncoder(w).Encode(h.tasks)
		} else {
			json.NewEncoder(w).Encode(openapi.NewTaskList("TaskList", 1, 0, 0, []openapi.Task{}))
		}
	case "/api/ambient-api-server/v1/projects":
		if h.projects != nil {
			json.NewEncoder(w).Encode(h.projects)
		} else {
			json.NewEncoder(w).Encode(openapi.NewProjectList("ProjectList", 1, 0, 0, []openapi.Project{}))
		}
	case "/api/ambient-api-server/v1/project_settings":
		if h.projectSettings != nil {
			json.NewEncoder(w).Encode(h.projectSettings)
		} else {
			json.NewEncoder(w).Encode(openapi.NewProjectSettingsList("ProjectSettingsList", 1, 0, 0, []openapi.ProjectSettings{}))
		}
	default:
		http.NotFound(w, r)
	}
}

func newTestInformer(serverURL string) *Informer {
	cfg := openapi.NewConfiguration()
	cfg.Servers = openapi.ServerConfigurations{
		{URL: serverURL},
	}
	client := openapi.NewAPIClient(cfg)

	return New(client, time.Second, zerolog.Nop())
}

type eventCollector struct {
	mu     sync.Mutex
	events []ResourceEvent
}

func (c *eventCollector) handler(ctx context.Context, event ResourceEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, event)
	return nil
}

func (c *eventCollector) getEvents() []ResourceEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]ResourceEvent, len(c.events))
	copy(result, c.events)
	return result
}

func (c *eventCollector) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = nil
}

func countEventsByType(events []ResourceEvent, eventType EventType) int {
	count := 0
	for _, e := range events {
		if e.Type == eventType {
			count++
		}
	}
	return count
}

func TestSyncSessions_EmptyCache_AllAdded(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	t1 := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	sessions := []openapi.Session{
		makeSession("s1", "session-1", t1),
		makeSession("s2", "session-2", t1),
		makeSession("s3", "session-3", t1),
	}
	handler.setSessions(openapi.NewSessionList("SessionList", 1, 3, 3, sessions))

	inf := newTestInformer(server.URL)
	collector := &eventCollector{}
	inf.RegisterHandler("sessions", collector.handler)

	if err := inf.syncSessions(context.Background(), "test"); err != nil {
		t.Fatalf("syncSessions failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 3 {
		t.Fatalf("expected 3 ADDED events, got %d", len(events))
	}
	for _, e := range events {
		if e.Type != EventAdded {
			t.Errorf("expected ADDED event, got %s", e.Type)
		}
		if e.Resource != "sessions" {
			t.Errorf("expected resource 'sessions', got %q", e.Resource)
		}
	}
}

func TestSyncSessions_UnchangedTimestamp_NoEvent(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	t1 := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	sessions := []openapi.Session{
		makeSession("s1", "session-1", t1),
	}
	handler.setSessions(openapi.NewSessionList("SessionList", 1, 1, 1, sessions))

	inf := newTestInformer(server.URL)
	collector := &eventCollector{}
	inf.RegisterHandler("sessions", collector.handler)

	if err := inf.syncSessions(context.Background(), "test"); err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	collector.reset()
	if err := inf.syncSessions(context.Background(), "test"); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events on unchanged data, got %d", len(events))
	}
}

func TestSyncSessions_ChangedTimestamp_Modified(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	t1 := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 2, 14, 13, 0, 0, 0, time.UTC)

	handler.setSessions(openapi.NewSessionList("SessionList", 1, 1, 1, []openapi.Session{
		makeSession("s1", "session-1", t1),
	}))

	inf := newTestInformer(server.URL)
	collector := &eventCollector{}
	inf.RegisterHandler("sessions", collector.handler)

	if err := inf.syncSessions(context.Background(), "test"); err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	collector.reset()
	handler.setSessions(openapi.NewSessionList("SessionList", 1, 1, 1, []openapi.Session{
		makeSession("s1", "session-1", t2),
	}))

	if err := inf.syncSessions(context.Background(), "test"); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 MODIFIED event, got %d", len(events))
	}
	if events[0].Type != EventModified {
		t.Errorf("expected MODIFIED, got %s", events[0].Type)
	}
	if events[0].OldObject == nil {
		t.Error("expected OldObject to be set on MODIFIED event")
	}
}

func TestSyncSessions_MissingFromList_Deleted(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	t1 := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)

	handler.setSessions(openapi.NewSessionList("SessionList", 1, 2, 2, []openapi.Session{
		makeSession("s1", "session-1", t1),
		makeSession("s2", "session-2", t1),
	}))

	inf := newTestInformer(server.URL)
	collector := &eventCollector{}
	inf.RegisterHandler("sessions", collector.handler)

	if err := inf.syncSessions(context.Background(), "test"); err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	collector.reset()
	handler.setSessions(openapi.NewSessionList("SessionList", 1, 1, 1, []openapi.Session{
		makeSession("s1", "session-1", t1),
	}))

	if err := inf.syncSessions(context.Background(), "test"); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 DELETED event, got %d", len(events))
	}
	if events[0].Type != EventDeleted {
		t.Errorf("expected DELETED, got %s", events[0].Type)
	}

	s, ok := events[0].Object.(openapi.Session)
	if !ok {
		t.Fatal("expected Object to be openapi.Session")
	}
	if s.GetId() != "s2" {
		t.Errorf("expected deleted session id 's2', got %q", s.GetId())
	}
}

func TestSyncSessions_MixedEvents(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	t1 := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 2, 14, 13, 0, 0, 0, time.UTC)

	handler.setSessions(openapi.NewSessionList("SessionList", 1, 2, 2, []openapi.Session{
		makeSession("s1", "session-1", t1),
		makeSession("s2", "session-2", t1),
	}))

	inf := newTestInformer(server.URL)
	collector := &eventCollector{}
	inf.RegisterHandler("sessions", collector.handler)

	if err := inf.syncSessions(context.Background(), "test"); err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	collector.reset()
	handler.setSessions(openapi.NewSessionList("SessionList", 1, 2, 2, []openapi.Session{
		makeSession("s1", "session-1", t2),
		makeSession("s3", "session-3", t1),
	}))

	if err := inf.syncSessions(context.Background(), "test"); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	events := collector.getEvents()

	added := countEventsByType(events, EventAdded)
	modified := countEventsByType(events, EventModified)
	deleted := countEventsByType(events, EventDeleted)

	if added != 1 {
		t.Errorf("expected 1 ADDED, got %d", added)
	}
	if modified != 1 {
		t.Errorf("expected 1 MODIFIED, got %d", modified)
	}
	if deleted != 1 {
		t.Errorf("expected 1 DELETED, got %d", deleted)
	}
}

func TestSyncWorkflows_DiffLogic(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	t1 := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)

	wf := *openapi.NewWorkflow("workflow-1")
	wf.SetId("w1")
	wf.SetUpdatedAt(t1)

	handler.setWorkflows(openapi.NewWorkflowList("WorkflowList", 1, 1, 1, []openapi.Workflow{wf}))

	inf := newTestInformer(server.URL)
	collector := &eventCollector{}
	inf.RegisterHandler("workflows", collector.handler)

	if err := inf.syncWorkflows(context.Background(), "test"); err != nil {
		t.Fatalf("syncWorkflows failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 ADDED event, got %d", len(events))
	}
	if events[0].Type != EventAdded {
		t.Errorf("expected ADDED, got %s", events[0].Type)
	}
	if events[0].Resource != "workflows" {
		t.Errorf("expected resource 'workflows', got %q", events[0].Resource)
	}
}

func TestSyncTasks_DiffLogic(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	t1 := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)

	task := *openapi.NewTask("task-1")
	task.SetId("t1")
	task.SetUpdatedAt(t1)

	handler.setTasks(openapi.NewTaskList("TaskList", 1, 1, 1, []openapi.Task{task}))

	inf := newTestInformer(server.URL)
	collector := &eventCollector{}
	inf.RegisterHandler("tasks", collector.handler)

	if err := inf.syncTasks(context.Background(), "test"); err != nil {
		t.Fatalf("syncTasks failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 ADDED event, got %d", len(events))
	}
	if events[0].Type != EventAdded {
		t.Errorf("expected ADDED, got %s", events[0].Type)
	}
	if events[0].Resource != "tasks" {
		t.Errorf("expected resource 'tasks', got %q", events[0].Resource)
	}
}

func TestDispatch_HandlerError_ContinuesProcessing(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	inf := newTestInformer(server.URL)

	failCount := 0
	successCount := 0
	var mu sync.Mutex

	inf.RegisterHandler("sessions", func(ctx context.Context, event ResourceEvent) error {
		mu.Lock()
		failCount++
		mu.Unlock()
		return context.DeadlineExceeded
	})
	inf.RegisterHandler("sessions", func(ctx context.Context, event ResourceEvent) error {
		mu.Lock()
		successCount++
		mu.Unlock()
		return nil
	})

	t1 := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	handler.setSessions(openapi.NewSessionList("SessionList", 1, 1, 1, []openapi.Session{
		makeSession("s1", "session-1", t1),
	}))

	if err := inf.syncSessions(context.Background(), "test"); err != nil {
		t.Fatalf("syncSessions failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if failCount != 1 {
		t.Errorf("expected failing handler called 1 time, got %d", failCount)
	}
	if successCount != 1 {
		t.Errorf("expected success handler called 1 time, got %d", successCount)
	}
}

func TestRegisterHandler_MultipleResources(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	inf := newTestInformer(server.URL)

	sessionCollector := &eventCollector{}
	workflowCollector := &eventCollector{}

	inf.RegisterHandler("sessions", sessionCollector.handler)
	inf.RegisterHandler("workflows", workflowCollector.handler)

	t1 := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	handler.setSessions(openapi.NewSessionList("SessionList", 1, 1, 1, []openapi.Session{
		makeSession("s1", "session-1", t1),
	}))
	wf := *openapi.NewWorkflow("wf-1")
	wf.SetId("w1")
	wf.SetUpdatedAt(t1)
	handler.setWorkflows(openapi.NewWorkflowList("WorkflowList", 1, 1, 1, []openapi.Workflow{wf}))

	if err := inf.syncAll(context.Background()); err != nil {
		t.Fatalf("syncAll failed: %v", err)
	}

	sessionEvents := sessionCollector.getEvents()
	workflowEvents := workflowCollector.getEvents()

	if len(sessionEvents) != 1 {
		t.Errorf("expected 1 session event, got %d", len(sessionEvents))
	}
	if len(workflowEvents) != 1 {
		t.Errorf("expected 1 workflow event, got %d", len(workflowEvents))
	}
}

func TestSyncAll_ErrorInSessionsStopsEarly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/ambient-api-server/v1/sessions" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		switch r.URL.Path {
		case "/api/ambient-api-server/v1/workflows":
			json.NewEncoder(w).Encode(openapi.NewWorkflowList("WorkflowList", 1, 0, 0, []openapi.Workflow{}))
		case "/api/ambient-api-server/v1/tasks":
			json.NewEncoder(w).Encode(openapi.NewTaskList("TaskList", 1, 0, 0, []openapi.Task{}))
		case "/api/ambient-api-server/v1/projects":
			json.NewEncoder(w).Encode(openapi.NewProjectList("ProjectList", 1, 0, 0, []openapi.Project{}))
		case "/api/ambient-api-server/v1/project_settings":
			json.NewEncoder(w).Encode(openapi.NewProjectSettingsList("ProjectSettingsList", 1, 0, 0, []openapi.ProjectSettings{}))
		}
	}))
	defer server.Close()

	inf := newTestInformer(server.URL)

	err := inf.syncAll(context.Background())
	if err == nil {
		t.Fatal("expected error from syncAll when sessions endpoint fails")
	}
}

func TestSyncSessions_EmptyList_NoEvents(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	handler.setSessions(openapi.NewSessionList("SessionList", 1, 0, 0, []openapi.Session{}))

	inf := newTestInformer(server.URL)
	collector := &eventCollector{}
	inf.RegisterHandler("sessions", collector.handler)

	if err := inf.syncSessions(context.Background(), "test"); err != nil {
		t.Fatalf("syncSessions failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events for empty list, got %d", len(events))
	}
}

func TestSyncSessions_CacheStateAfterSync(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	t1 := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	handler.setSessions(openapi.NewSessionList("SessionList", 1, 2, 2, []openapi.Session{
		makeSession("s1", "session-1", t1),
		makeSession("s2", "session-2", t1),
	}))

	inf := newTestInformer(server.URL)
	inf.RegisterHandler("sessions", func(ctx context.Context, event ResourceEvent) error { return nil })

	if err := inf.syncSessions(context.Background(), "test"); err != nil {
		t.Fatalf("syncSessions failed: %v", err)
	}

	if len(inf.sessionCache) != 2 {
		t.Errorf("expected 2 items in cache, got %d", len(inf.sessionCache))
	}
	if _, ok := inf.sessionCache["s1"]; !ok {
		t.Error("expected 's1' in cache")
	}
	if _, ok := inf.sessionCache["s2"]; !ok {
		t.Error("expected 's2' in cache")
	}
}

type paginatedSessionHandler struct {
	mu          sync.Mutex
	allSessions []openapi.Session
	pageSize    int32
	callLog     []int32
}

func (h *paginatedSessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	switch r.URL.Path {
	case "/api/ambient-api-server/v1/sessions":
		pageStr := r.URL.Query().Get("page")
		sizeStr := r.URL.Query().Get("size")

		page := int32(1)
		size := h.pageSize
		if pageStr != "" {
			if v, err := strconv.Atoi(pageStr); err == nil {
				page = int32(v)
			}
		}
		if sizeStr != "" {
			if v, err := strconv.Atoi(sizeStr); err == nil {
				size = int32(v)
			}
		}

		h.callLog = append(h.callLog, page)

		total := int32(len(h.allSessions))
		start := (page - 1) * size
		end := start + size
		if start > total {
			start = total
		}
		if end > total {
			end = total
		}

		items := h.allSessions[start:end]
		list := openapi.NewSessionList("SessionList", page, int32(len(items)), total, items)
		json.NewEncoder(w).Encode(list)

	case "/api/ambient-api-server/v1/workflows":
		json.NewEncoder(w).Encode(openapi.NewWorkflowList("WorkflowList", 1, 0, 0, []openapi.Workflow{}))
	case "/api/ambient-api-server/v1/tasks":
		json.NewEncoder(w).Encode(openapi.NewTaskList("TaskList", 1, 0, 0, []openapi.Task{}))
	case "/api/ambient-api-server/v1/projects":
		json.NewEncoder(w).Encode(openapi.NewProjectList("ProjectList", 1, 0, 0, []openapi.Project{}))
	case "/api/ambient-api-server/v1/project_settings":
		json.NewEncoder(w).Encode(openapi.NewProjectSettingsList("ProjectSettingsList", 1, 0, 0, []openapi.ProjectSettings{}))
	default:
		http.NotFound(w, r)
	}
}

func generateSessions(count int) []openapi.Session {
	t1 := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	sessions := make([]openapi.Session, count)
	for i := 0; i < count; i++ {
		sessions[i] = makeSession(fmt.Sprintf("s%d", i+1), fmt.Sprintf("session-%d", i+1), t1)
	}
	return sessions
}

func TestPagination_SinglePage(t *testing.T) {
	ph := &paginatedSessionHandler{
		allSessions: generateSessions(3),
		pageSize:    100,
	}
	server := httptest.NewServer(ph)
	defer server.Close()

	inf := newTestInformer(server.URL)
	inf.pageSize = 100
	collector := &eventCollector{}
	inf.RegisterHandler("sessions", collector.handler)

	if err := inf.syncSessions(context.Background(), "test"); err != nil {
		t.Fatalf("syncSessions failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 3 {
		t.Errorf("expected 3 ADDED events, got %d", len(events))
	}

	ph.mu.Lock()
	if len(ph.callLog) != 1 {
		t.Errorf("expected 1 page request, got %d: %v", len(ph.callLog), ph.callLog)
	}
	ph.mu.Unlock()
}

func TestPagination_MultiplePages(t *testing.T) {
	ph := &paginatedSessionHandler{
		allSessions: generateSessions(250),
		pageSize:    100,
	}
	server := httptest.NewServer(ph)
	defer server.Close()

	inf := newTestInformer(server.URL)
	inf.pageSize = 100
	collector := &eventCollector{}
	inf.RegisterHandler("sessions", collector.handler)

	if err := inf.syncSessions(context.Background(), "test"); err != nil {
		t.Fatalf("syncSessions failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 250 {
		t.Errorf("expected 250 ADDED events, got %d", len(events))
	}

	ph.mu.Lock()
	if len(ph.callLog) != 3 {
		t.Errorf("expected 3 page requests (100+100+50), got %d: %v", len(ph.callLog), ph.callLog)
	}
	for i, page := range ph.callLog {
		if page != int32(i+1) {
			t.Errorf("expected page %d at index %d, got %d", i+1, i, page)
		}
	}
	ph.mu.Unlock()
}

func TestPagination_ExactBoundary(t *testing.T) {
	ph := &paginatedSessionHandler{
		allSessions: generateSessions(200),
		pageSize:    100,
	}
	server := httptest.NewServer(ph)
	defer server.Close()

	inf := newTestInformer(server.URL)
	inf.pageSize = 100
	collector := &eventCollector{}
	inf.RegisterHandler("sessions", collector.handler)

	if err := inf.syncSessions(context.Background(), "test"); err != nil {
		t.Fatalf("syncSessions failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 200 {
		t.Errorf("expected 200 ADDED events, got %d", len(events))
	}

	ph.mu.Lock()
	if len(ph.callLog) != 2 {
		t.Errorf("expected exactly 2 page requests (100+100), got %d: %v", len(ph.callLog), ph.callLog)
	}
	ph.mu.Unlock()
}

func TestPagination_EmptyResponse(t *testing.T) {
	ph := &paginatedSessionHandler{
		allSessions: []openapi.Session{},
		pageSize:    100,
	}
	server := httptest.NewServer(ph)
	defer server.Close()

	inf := newTestInformer(server.URL)
	inf.pageSize = 100
	collector := &eventCollector{}
	inf.RegisterHandler("sessions", collector.handler)

	if err := inf.syncSessions(context.Background(), "test"); err != nil {
		t.Fatalf("syncSessions failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}

	ph.mu.Lock()
	if len(ph.callLog) != 1 {
		t.Errorf("expected 1 page request (empty response terminates), got %d", len(ph.callLog))
	}
	ph.mu.Unlock()
}

func TestPagination_SmallPageSize(t *testing.T) {
	ph := &paginatedSessionHandler{
		allSessions: generateSessions(10),
		pageSize:    3,
	}
	server := httptest.NewServer(ph)
	defer server.Close()

	inf := newTestInformer(server.URL)
	inf.pageSize = 3
	collector := &eventCollector{}
	inf.RegisterHandler("sessions", collector.handler)

	if err := inf.syncSessions(context.Background(), "test"); err != nil {
		t.Fatalf("syncSessions failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 10 {
		t.Errorf("expected 10 ADDED events, got %d", len(events))
	}

	ph.mu.Lock()
	if len(ph.callLog) != 4 {
		t.Errorf("expected 4 page requests (3+3+3+1), got %d: %v", len(ph.callLog), ph.callLog)
	}
	ph.mu.Unlock()
}

func makeProject(id, name string, updatedAt time.Time) openapi.Project {
	p := *openapi.NewProject(name)
	p.SetId(id)
	p.SetUpdatedAt(updatedAt)
	return p
}

func makeProjectSettings(id, projectID string, updatedAt time.Time) openapi.ProjectSettings {
	ps := *openapi.NewProjectSettings(projectID)
	ps.SetId(id)
	ps.SetUpdatedAt(updatedAt)
	return ps
}

func TestSyncProjects_EmptyCache_AllAdded(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	t1 := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	projects := []openapi.Project{
		makeProject("p1", "project-alpha", t1),
		makeProject("p2", "project-beta", t1),
	}
	handler.setProjects(openapi.NewProjectList("ProjectList", 1, 2, 2, projects))

	inf := newTestInformer(server.URL)
	collector := &eventCollector{}
	inf.RegisterHandler("projects", collector.handler)

	if err := inf.syncProjects(context.Background(), "test"); err != nil {
		t.Fatalf("syncProjects failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 2 {
		t.Fatalf("expected 2 ADDED events, got %d", len(events))
	}
	for _, e := range events {
		if e.Type != EventAdded {
			t.Errorf("expected ADDED event, got %s", e.Type)
		}
		if e.Resource != "projects" {
			t.Errorf("expected resource 'projects', got %q", e.Resource)
		}
	}
}

func TestSyncProjects_UnchangedTimestamp_NoEvent(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	t1 := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	handler.setProjects(openapi.NewProjectList("ProjectList", 1, 1, 1, []openapi.Project{
		makeProject("p1", "project-alpha", t1),
	}))

	inf := newTestInformer(server.URL)
	collector := &eventCollector{}
	inf.RegisterHandler("projects", collector.handler)

	if err := inf.syncProjects(context.Background(), "test"); err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	collector.reset()
	if err := inf.syncProjects(context.Background(), "test"); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events on unchanged data, got %d", len(events))
	}
}

func TestSyncProjects_ChangedTimestamp_Modified(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	t1 := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 2, 15, 13, 0, 0, 0, time.UTC)

	handler.setProjects(openapi.NewProjectList("ProjectList", 1, 1, 1, []openapi.Project{
		makeProject("p1", "project-alpha", t1),
	}))

	inf := newTestInformer(server.URL)
	collector := &eventCollector{}
	inf.RegisterHandler("projects", collector.handler)

	if err := inf.syncProjects(context.Background(), "test"); err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	collector.reset()
	handler.setProjects(openapi.NewProjectList("ProjectList", 1, 1, 1, []openapi.Project{
		makeProject("p1", "project-alpha", t2),
	}))

	if err := inf.syncProjects(context.Background(), "test"); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 MODIFIED event, got %d", len(events))
	}
	if events[0].Type != EventModified {
		t.Errorf("expected MODIFIED, got %s", events[0].Type)
	}
	if events[0].OldObject == nil {
		t.Error("expected OldObject to be set on MODIFIED event")
	}
}

func TestSyncProjects_MissingFromList_Deleted(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	t1 := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)

	handler.setProjects(openapi.NewProjectList("ProjectList", 1, 2, 2, []openapi.Project{
		makeProject("p1", "project-alpha", t1),
		makeProject("p2", "project-beta", t1),
	}))

	inf := newTestInformer(server.URL)
	collector := &eventCollector{}
	inf.RegisterHandler("projects", collector.handler)

	if err := inf.syncProjects(context.Background(), "test"); err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	collector.reset()
	handler.setProjects(openapi.NewProjectList("ProjectList", 1, 1, 1, []openapi.Project{
		makeProject("p1", "project-alpha", t1),
	}))

	if err := inf.syncProjects(context.Background(), "test"); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 DELETED event, got %d", len(events))
	}
	if events[0].Type != EventDeleted {
		t.Errorf("expected DELETED, got %s", events[0].Type)
	}

	p, ok := events[0].Object.(openapi.Project)
	if !ok {
		t.Fatal("expected Object to be openapi.Project")
	}
	if p.GetId() != "p2" {
		t.Errorf("expected deleted project id 'p2', got %q", p.GetId())
	}
}

func TestSyncProjects_CacheStateAfterSync(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	t1 := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	handler.setProjects(openapi.NewProjectList("ProjectList", 1, 2, 2, []openapi.Project{
		makeProject("p1", "project-alpha", t1),
		makeProject("p2", "project-beta", t1),
	}))

	inf := newTestInformer(server.URL)
	inf.RegisterHandler("projects", func(ctx context.Context, event ResourceEvent) error { return nil })

	if err := inf.syncProjects(context.Background(), "test"); err != nil {
		t.Fatalf("syncProjects failed: %v", err)
	}

	if len(inf.projectCache) != 2 {
		t.Errorf("expected 2 items in cache, got %d", len(inf.projectCache))
	}
	if _, ok := inf.projectCache["p1"]; !ok {
		t.Error("expected 'p1' in cache")
	}
	if _, ok := inf.projectCache["p2"]; !ok {
		t.Error("expected 'p2' in cache")
	}
}

func TestSyncProjectSettings_EmptyCache_AllAdded(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	t1 := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	settings := []openapi.ProjectSettings{
		makeProjectSettings("ps1", "proj-alpha", t1),
		makeProjectSettings("ps2", "proj-beta", t1),
	}
	handler.setProjectSettings(openapi.NewProjectSettingsList("ProjectSettingsList", 1, 2, 2, settings))

	inf := newTestInformer(server.URL)
	collector := &eventCollector{}
	inf.RegisterHandler("project_settings", collector.handler)

	if err := inf.syncProjectSettings(context.Background(), "test"); err != nil {
		t.Fatalf("syncProjectSettings failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 2 {
		t.Fatalf("expected 2 ADDED events, got %d", len(events))
	}
	for _, e := range events {
		if e.Type != EventAdded {
			t.Errorf("expected ADDED event, got %s", e.Type)
		}
		if e.Resource != "project_settings" {
			t.Errorf("expected resource 'project_settings', got %q", e.Resource)
		}
	}
}

func TestSyncProjectSettings_UnchangedTimestamp_NoEvent(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	t1 := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	handler.setProjectSettings(openapi.NewProjectSettingsList("ProjectSettingsList", 1, 1, 1, []openapi.ProjectSettings{
		makeProjectSettings("ps1", "proj-alpha", t1),
	}))

	inf := newTestInformer(server.URL)
	collector := &eventCollector{}
	inf.RegisterHandler("project_settings", collector.handler)

	if err := inf.syncProjectSettings(context.Background(), "test"); err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	collector.reset()
	if err := inf.syncProjectSettings(context.Background(), "test"); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events on unchanged data, got %d", len(events))
	}
}

func TestSyncProjectSettings_ChangedTimestamp_Modified(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	t1 := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 2, 15, 13, 0, 0, 0, time.UTC)

	handler.setProjectSettings(openapi.NewProjectSettingsList("ProjectSettingsList", 1, 1, 1, []openapi.ProjectSettings{
		makeProjectSettings("ps1", "proj-alpha", t1),
	}))

	inf := newTestInformer(server.URL)
	collector := &eventCollector{}
	inf.RegisterHandler("project_settings", collector.handler)

	if err := inf.syncProjectSettings(context.Background(), "test"); err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	collector.reset()
	handler.setProjectSettings(openapi.NewProjectSettingsList("ProjectSettingsList", 1, 1, 1, []openapi.ProjectSettings{
		makeProjectSettings("ps1", "proj-alpha", t2),
	}))

	if err := inf.syncProjectSettings(context.Background(), "test"); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 MODIFIED event, got %d", len(events))
	}
	if events[0].Type != EventModified {
		t.Errorf("expected MODIFIED, got %s", events[0].Type)
	}
	if events[0].OldObject == nil {
		t.Error("expected OldObject to be set on MODIFIED event")
	}
}

func TestSyncProjectSettings_MissingFromList_Deleted(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	t1 := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)

	handler.setProjectSettings(openapi.NewProjectSettingsList("ProjectSettingsList", 1, 2, 2, []openapi.ProjectSettings{
		makeProjectSettings("ps1", "proj-alpha", t1),
		makeProjectSettings("ps2", "proj-beta", t1),
	}))

	inf := newTestInformer(server.URL)
	collector := &eventCollector{}
	inf.RegisterHandler("project_settings", collector.handler)

	if err := inf.syncProjectSettings(context.Background(), "test"); err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	collector.reset()
	handler.setProjectSettings(openapi.NewProjectSettingsList("ProjectSettingsList", 1, 1, 1, []openapi.ProjectSettings{
		makeProjectSettings("ps1", "proj-alpha", t1),
	}))

	if err := inf.syncProjectSettings(context.Background(), "test"); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	events := collector.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 DELETED event, got %d", len(events))
	}
	if events[0].Type != EventDeleted {
		t.Errorf("expected DELETED, got %s", events[0].Type)
	}

	ps, ok := events[0].Object.(openapi.ProjectSettings)
	if !ok {
		t.Fatal("expected Object to be openapi.ProjectSettings")
	}
	if ps.GetId() != "ps2" {
		t.Errorf("expected deleted settings id 'ps2', got %q", ps.GetId())
	}
}

func TestSyncProjectSettings_CacheStateAfterSync(t *testing.T) {
	handler := &fakeAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	t1 := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	handler.setProjectSettings(openapi.NewProjectSettingsList("ProjectSettingsList", 1, 2, 2, []openapi.ProjectSettings{
		makeProjectSettings("ps1", "proj-alpha", t1),
		makeProjectSettings("ps2", "proj-beta", t1),
	}))

	inf := newTestInformer(server.URL)
	inf.RegisterHandler("project_settings", func(ctx context.Context, event ResourceEvent) error { return nil })

	if err := inf.syncProjectSettings(context.Background(), "test"); err != nil {
		t.Fatalf("syncProjectSettings failed: %v", err)
	}

	if len(inf.projectSettingsCache) != 2 {
		t.Errorf("expected 2 items in cache, got %d", len(inf.projectSettingsCache))
	}
	if _, ok := inf.projectSettingsCache["ps1"]; !ok {
		t.Error("expected 'ps1' in cache")
	}
	if _, ok := inf.projectSettingsCache["ps2"]; !ok {
		t.Error("expected 'ps2' in cache")
	}
}
