package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	openapi "github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient/platform/components/ambient-control-plane/internal/informer"
	"github.com/ambient/platform/components/ambient-control-plane/internal/process"
	"github.com/rs/zerolog"
)

func findTestPorts(t *testing.T, count int) (int, int) {
	t.Helper()
	start := 19300
	for s := start; s < 19900; s++ {
		allFree := true
		for p := s; p < s+count; p++ {
			ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
			if err != nil {
				allFree = false
				break
			}
			ln.Close()
		}
		if allFree {
			return s, s + count - 1
		}
	}
	t.Fatal("could not find free ports")
	return 0, 0
}

type fakeAPIServer struct {
	mu       sync.Mutex
	patches  []capturedPatch
	sessions map[string]openapi.Session
	server   *httptest.Server
}

type capturedPatch struct {
	SessionID string
	Phase     string
	Body      map[string]interface{}
}

func newFakeAPIServer(t *testing.T) *fakeAPIServer {
	t.Helper()
	f := &fakeAPIServer{
		sessions: make(map[string]openapi.Session),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/ambient-api-server/v1/sessions/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			parts := splitPath(r.URL.Path)
			if len(parts) < 6 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			sessionID := parts[4]

			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)

			f.mu.Lock()
			phase, _ := body["phase"].(string)
			f.patches = append(f.patches, capturedPatch{
				SessionID: sessionID,
				Phase:     phase,
				Body:      body,
			})
			f.mu.Unlock()

			now := time.Now()
			resp := openapi.Session{
				Name: "test",
			}
			resp.SetId(sessionID)
			resp.SetUpdatedAt(now)
			if phase != "" {
				resp.SetPhase(phase)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	f.server = httptest.NewServer(mux)
	return f
}

func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func (f *fakeAPIServer) getPatches() []capturedPatch {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]capturedPatch, len(f.patches))
	copy(result, f.patches)
	return result
}

func (f *fakeAPIServer) close() {
	f.server.Close()
}

func (f *fakeAPIServer) buildClient() *openapi.APIClient {
	cfg := openapi.NewConfiguration()
	cfg.Servers = openapi.ServerConfigurations{
		{URL: f.server.URL, Description: "test"},
	}
	cfg.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	return openapi.NewAPIClient(cfg)
}

func testLocalReconciler(t *testing.T) (*LocalSessionReconciler, *process.Manager, *fakeAPIServer) {
	t.Helper()
	portStart, portEnd := findTestPorts(t, 10)
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	mgr := process.NewManager(process.ManagerConfig{
		WorkspaceRoot: t.TempDir(),
		RunnerCommand: "sleep 30",
		PortStart:     portStart,
		PortEnd:       portEnd,
		MaxSessions:   10,
	}, logger)

	fakeAPI := newFakeAPIServer(t)
	client := fakeAPI.buildClient()

	rec := NewLocalSessionReconciler(client, mgr, logger)
	return rec, mgr, fakeAPI
}

func makeSession(id, name, phase string) openapi.Session {
	s := openapi.Session{Name: name}
	if id != "" {
		s.SetId(id)
	}
	if phase != "" {
		s.SetPhase(phase)
	}
	return s
}

func makeKubeSession(id, name, phase, crName string) openapi.Session {
	s := makeSession(id, name, phase)
	s.SetKubeCrName(crName)
	return s
}

func TestLocalReconcilerResource(t *testing.T) {
	rec, _, fakeAPI := testLocalReconciler(t)
	defer fakeAPI.close()

	if rec.Resource() != "sessions" {
		t.Errorf("expected resource 'sessions', got %q", rec.Resource())
	}
}

func TestLocalReconcilerSkipsK8sSession(t *testing.T) {
	rec, _, fakeAPI := testLocalReconciler(t)
	defer fakeAPI.close()

	session := makeKubeSession("k8s-session", "test", PhasePending, "my-cr-name")

	err := rec.Reconcile(context.Background(), informer.ResourceEvent{
		Type:   informer.EventAdded,
		Object: session,
	})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	patches := fakeAPI.getPatches()
	if len(patches) != 0 {
		t.Errorf("expected 0 patches for K8s session, got %d", len(patches))
	}
}

func TestLocalReconcilerTypeAssertionFailure(t *testing.T) {
	rec, _, fakeAPI := testLocalReconciler(t)
	defer fakeAPI.close()

	err := rec.Reconcile(context.Background(), informer.ResourceEvent{
		Type:   informer.EventAdded,
		Object: "not-a-session",
	})
	if err != nil {
		t.Fatalf("expected nil error for type assertion failure, got %v", err)
	}
}

func TestLocalReconcilerSkipsTerminalPhase(t *testing.T) {
	rec, _, fakeAPI := testLocalReconciler(t)
	defer fakeAPI.close()

	for _, phase := range TerminalPhases {
		t.Run(phase, func(t *testing.T) {
			session := makeSession("terminal-"+phase, "test", phase)
			err := rec.Reconcile(context.Background(), informer.ResourceEvent{
				Type:   informer.EventAdded,
				Object: session,
			})
			if err != nil {
				t.Fatalf("reconcile: %v", err)
			}
		})
	}

	patches := fakeAPI.getPatches()
	if len(patches) != 0 {
		t.Errorf("expected 0 patches for terminal sessions, got %d", len(patches))
	}
}

func TestLocalReconcilerSpawnsPendingSession(t *testing.T) {
	portStart, portEnd := findTestPorts(t, 10)
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	tmpDir := t.TempDir()

	mgr := process.NewManager(process.ManagerConfig{
		WorkspaceRoot: tmpDir,
		RunnerCommand: "sleep 30",
		PortStart:     portStart,
		PortEnd:       portEnd,
		MaxSessions:   10,
	}, logger)

	fakeAPI := newFakeAPIServer(t)
	defer fakeAPI.close()
	client := fakeAPI.buildClient()
	rec := NewLocalSessionReconciler(client, mgr, logger)

	session := makeSession("spawn-test", "My Session", PhasePending)

	err := rec.Reconcile(context.Background(), informer.ResourceEvent{
		Type:   informer.EventAdded,
		Object: session,
	})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	_, exists := mgr.GetProcess("spawn-test")
	if !exists {
		t.Error("expected process to be spawned")
	}

	time.Sleep(200 * time.Millisecond)

	patches := fakeAPI.getPatches()
	if len(patches) == 0 {
		t.Fatal("expected at least 1 patch (Creating)")
	}

	if patches[0].Phase != PhaseCreating {
		t.Errorf("expected first patch phase 'Creating', got %q", patches[0].Phase)
	}

	mgr.Shutdown(context.Background())
}

func TestLocalReconcilerSpawnsEmptyPhaseSession(t *testing.T) {
	portStart, portEnd := findTestPorts(t, 10)
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	mgr := process.NewManager(process.ManagerConfig{
		WorkspaceRoot: t.TempDir(),
		RunnerCommand: "sleep 30",
		PortStart:     portStart,
		PortEnd:       portEnd,
		MaxSessions:   10,
	}, logger)

	fakeAPI := newFakeAPIServer(t)
	defer fakeAPI.close()
	client := fakeAPI.buildClient()
	rec := NewLocalSessionReconciler(client, mgr, logger)

	session := makeSession("empty-phase", "Test Session", "")

	err := rec.Reconcile(context.Background(), informer.ResourceEvent{
		Type:   informer.EventAdded,
		Object: session,
	})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	_, exists := mgr.GetProcess("empty-phase")
	if !exists {
		t.Error("expected process to be spawned for empty phase session")
	}

	mgr.Shutdown(context.Background())
}

func TestLocalReconcilerIdempotentSpawn(t *testing.T) {
	portStart, portEnd := findTestPorts(t, 10)
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	mgr := process.NewManager(process.ManagerConfig{
		WorkspaceRoot: t.TempDir(),
		RunnerCommand: "sleep 30",
		PortStart:     portStart,
		PortEnd:       portEnd,
		MaxSessions:   10,
	}, logger)

	fakeAPI := newFakeAPIServer(t)
	defer fakeAPI.close()
	client := fakeAPI.buildClient()
	rec := NewLocalSessionReconciler(client, mgr, logger)

	session := makeSession("idem-session", "Test", PhasePending)

	rec.Reconcile(context.Background(), informer.ResourceEvent{
		Type:   informer.EventAdded,
		Object: session,
	})

	rec.Reconcile(context.Background(), informer.ResourceEvent{
		Type:   informer.EventAdded,
		Object: session,
	})

	time.Sleep(200 * time.Millisecond)

	patches := fakeAPI.getPatches()
	creatingCount := 0
	for _, p := range patches {
		if p.Phase == PhaseCreating {
			creatingCount++
		}
	}

	if creatingCount != 1 {
		t.Errorf("expected exactly 1 Creating patch (idempotent), got %d", creatingCount)
	}

	mgr.Shutdown(context.Background())
}

func TestLocalReconcilerStopSession(t *testing.T) {
	portStart, portEnd := findTestPorts(t, 10)
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	mgr := process.NewManager(process.ManagerConfig{
		WorkspaceRoot: t.TempDir(),
		RunnerCommand: "sleep 30",
		PortStart:     portStart,
		PortEnd:       portEnd,
		MaxSessions:   10,
	}, logger)

	fakeAPI := newFakeAPIServer(t)
	defer fakeAPI.close()
	client := fakeAPI.buildClient()
	rec := NewLocalSessionReconciler(client, mgr, logger)

	ctx := context.Background()
	session := makeSession("stop-session", "Test", PhasePending)
	rec.Reconcile(ctx, informer.ResourceEvent{
		Type:   informer.EventAdded,
		Object: session,
	})

	_, exists := mgr.GetProcess("stop-session")
	if !exists {
		t.Fatal("expected process to be running before stop")
	}

	stoppingSession := makeSession("stop-session", "Test", PhaseStopping)
	now := time.Now().Add(1 * time.Second)
	stoppingSession.SetUpdatedAt(now)

	err := rec.Reconcile(ctx, informer.ResourceEvent{
		Type:   informer.EventModified,
		Object: stoppingSession,
	})
	if err != nil {
		t.Fatalf("reconcile stopping: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	patches := fakeAPI.getPatches()
	foundStopped := false
	for _, p := range patches {
		if p.Phase == PhaseStopped {
			foundStopped = true
		}
	}

	if !foundStopped {
		t.Error("expected Stopped patch after stopping")
	}
}

func TestLocalReconcilerDeleteKillsProcess(t *testing.T) {
	portStart, portEnd := findTestPorts(t, 10)
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	mgr := process.NewManager(process.ManagerConfig{
		WorkspaceRoot: t.TempDir(),
		RunnerCommand: "sleep 30",
		PortStart:     portStart,
		PortEnd:       portEnd,
		MaxSessions:   10,
	}, logger)

	fakeAPI := newFakeAPIServer(t)
	defer fakeAPI.close()
	client := fakeAPI.buildClient()
	rec := NewLocalSessionReconciler(client, mgr, logger)

	ctx := context.Background()
	session := makeSession("delete-session", "Test", PhasePending)
	rec.Reconcile(ctx, informer.ResourceEvent{
		Type:   informer.EventAdded,
		Object: session,
	})

	_, exists := mgr.GetProcess("delete-session")
	if !exists {
		t.Fatal("expected process to be running before delete")
	}

	err := rec.Reconcile(ctx, informer.ResourceEvent{
		Type:   informer.EventDeleted,
		Object: session,
	})
	if err != nil {
		t.Fatalf("reconcile delete: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	_, exists = mgr.GetProcess("delete-session")
	if exists {
		t.Error("expected process to be gone after delete")
	}
}

func TestLocalReconcilerHandleProcessExit(t *testing.T) {
	rec, _, fakeAPI := testLocalReconciler(t)
	defer fakeAPI.close()

	rec.HandleProcessExit(process.ProcessExitEvent{
		SessionID:  "exit-session",
		ExitCode:   0,
		StderrTail: "",
		Duration:   5 * time.Second,
	})

	time.Sleep(200 * time.Millisecond)

	patches := fakeAPI.getPatches()
	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Phase != PhaseCompleted {
		t.Errorf("expected phase Completed for exit code 0, got %q", patches[0].Phase)
	}
}

func TestLocalReconcilerHandleProcessExitFailure(t *testing.T) {
	rec, _, fakeAPI := testLocalReconciler(t)
	defer fakeAPI.close()

	rec.HandleProcessExit(process.ProcessExitEvent{
		SessionID:  "fail-session",
		ExitCode:   1,
		StderrTail: "error occurred",
		Duration:   3 * time.Second,
	})

	time.Sleep(200 * time.Millisecond)

	patches := fakeAPI.getPatches()
	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Phase != PhaseFailed {
		t.Errorf("expected phase Failed for exit code 1, got %q", patches[0].Phase)
	}
}

func TestLocalReconcilerWritebackEchoPrevention(t *testing.T) {
	portStart, portEnd := findTestPorts(t, 10)
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	mgr := process.NewManager(process.ManagerConfig{
		WorkspaceRoot: t.TempDir(),
		RunnerCommand: "sleep 30",
		PortStart:     portStart,
		PortEnd:       portEnd,
		MaxSessions:   10,
	}, logger)

	fakeAPI := newFakeAPIServer(t)
	defer fakeAPI.close()
	client := fakeAPI.buildClient()
	rec := NewLocalSessionReconciler(client, mgr, logger)

	now := time.Now().Truncate(time.Microsecond)
	rec.lastWritebackAt.Store("echo-session", now)

	session := makeSession("echo-session", "Test", PhaseRunning)
	session.SetUpdatedAt(now)

	err := rec.Reconcile(context.Background(), informer.ResourceEvent{
		Type:   informer.EventModified,
		Object: session,
	})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	patches := fakeAPI.getPatches()
	if len(patches) != 0 {
		t.Errorf("expected 0 patches (writeback echo), got %d", len(patches))
	}

	mgr.Shutdown(context.Background())
}

func TestLocalReconcilerBuildSessionEnv(t *testing.T) {
	rec, _, fakeAPI := testLocalReconciler(t)
	defer fakeAPI.close()

	session := openapi.Session{Name: "My Test Session"}
	session.SetId("env-session")
	prompt := "do something"
	session.Prompt = &prompt
	interactive := true
	session.Interactive = &interactive
	model := "claude-opus-4"
	session.LlmModel = &model
	temp := 0.7
	session.LlmTemperature = &temp
	maxTokens := int32(4000)
	session.LlmMaxTokens = &maxTokens
	timeout := int32(300)
	session.Timeout = &timeout
	repos := `[{"url":"https://github.com/test/repo"}]`
	session.Repos = &repos
	projectID := "my-project"
	session.ProjectId = &projectID

	env := rec.buildSessionEnv(session)

	expected := map[string]string{
		"AGENTIC_SESSION_NAME": "My Test Session",
		"BOSS_AGENT_NAME":      "my-test-session",
		"INITIAL_PROMPT":       "do something",
		"INTERACTIVE":          "true",
		"LLM_MODEL":            "claude-opus-4",
		"LLM_TEMPERATURE":      "0.7",
		"LLM_MAX_TOKENS":       "4000",
		"SESSION_TIMEOUT":      "300",
		"REPOS_JSON":           repos,
		"PROJECT_NAME":         "my-project",
	}

	for k, v := range expected {
		got, ok := env[k]
		if !ok {
			t.Errorf("missing env var %q", k)
			continue
		}
		if got != v {
			t.Errorf("env[%q] = %q, want %q", k, got, v)
		}
	}
}

func TestLocalReconcilerReapLoop(t *testing.T) {
	rec, mgr, fakeAPI := testLocalReconciler(t)
	defer fakeAPI.close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go rec.ReapLoop(ctx)

	mgr.ExitCh <- process.ProcessExitEvent{
		SessionID:  "reap-session",
		ExitCode:   0,
		StderrTail: "",
		Duration:   2 * time.Second,
	}

	time.Sleep(500 * time.Millisecond)

	patches := fakeAPI.getPatches()
	if len(patches) != 1 {
		t.Fatalf("expected 1 patch from ReapLoop, got %d", len(patches))
	}
	if patches[0].Phase != PhaseCompleted {
		t.Errorf("expected Completed, got %q", patches[0].Phase)
	}
	if patches[0].SessionID != "reap-session" {
		t.Errorf("expected session reap-session, got %q", patches[0].SessionID)
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"My Session", "my-session"},
		{"hello-world", "hello-world"},
		{"CamelCase", "camelcase"},
		{"with_underscores", "with-underscores"},
		{"special!chars@here", "specialcharshere"},
		{"", ""},
		{"already-slugified", "already-slugified"},
		{"MiXeD CaSe_stuff", "mixed-case-stuff"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := slugify(tt.input)
			if got != tt.expected {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBuildConditions(t *testing.T) {
	tests := []struct {
		phase  string
		reason string
		check  func(t *testing.T, result string)
	}{
		{
			PhaseCreating, "",
			func(t *testing.T, result string) {
				if result == "" {
					t.Error("expected non-empty conditions")
				}
				var conditions []map[string]string
				json.Unmarshal([]byte(result), &conditions)
				for _, c := range conditions {
					if c["type"] == "ProcessSpawned" && c["status"] != "True" {
						t.Error("expected ProcessSpawned=True for Creating")
					}
					if c["type"] == "HealthCheckPassed" && c["status"] != "False" {
						t.Error("expected HealthCheckPassed=False for Creating")
					}
				}
			},
		},
		{
			PhaseRunning, "",
			func(t *testing.T, result string) {
				var conditions []map[string]string
				json.Unmarshal([]byte(result), &conditions)
				for _, c := range conditions {
					if c["type"] == "HealthCheckPassed" && c["status"] != "True" {
						t.Error("expected HealthCheckPassed=True for Running")
					}
					if c["type"] == "RunnerStarted" && c["status"] != "True" {
						t.Error("expected RunnerStarted=True for Running")
					}
				}
			},
		},
		{
			PhaseFailed, "ProcessSpawnFailed: error",
			func(t *testing.T, result string) {
				var conditions []map[string]string
				json.Unmarshal([]byte(result), &conditions)
				for _, c := range conditions {
					if c["type"] == "ProcessSpawned" && c["status"] != "False" {
						t.Error("expected ProcessSpawned=False for spawn failure")
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.phase, func(t *testing.T) {
			result := buildConditions(tt.phase, tt.reason)
			tt.check(t, result)
		})
	}
}

func TestIsTerminalPhase(t *testing.T) {
	tests := []struct {
		phase    string
		terminal bool
	}{
		{PhasePending, false},
		{PhaseCreating, false},
		{PhaseRunning, false},
		{PhaseStopping, false},
		{PhaseStopped, true},
		{PhaseCompleted, true},
		{PhaseFailed, true},
		{"", false},
		{"Unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.phase, func(t *testing.T) {
			got := isTerminalPhase(tt.phase)
			if got != tt.terminal {
				t.Errorf("isTerminalPhase(%q) = %v, want %v", tt.phase, got, tt.terminal)
			}
		})
	}
}
