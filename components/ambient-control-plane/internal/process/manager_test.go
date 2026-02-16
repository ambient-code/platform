package process

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func testManager(t *testing.T, portStart, portEnd int) *Manager {
	t.Helper()
	tmpDir := t.TempDir()
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	return NewManager(ManagerConfig{
		WorkspaceRoot: tmpDir,
		RunnerCommand: "sleep 30",
		PortStart:     portStart,
		PortEnd:       portEnd,
		MaxSessions:   10,
	}, logger)
}

func findAvailablePorts(t *testing.T, count int) (int, int) {
	t.Helper()
	start := 19100
	for s := start; s < 19900; s++ {
		allFree := true
		for p := s; p < s+count; p++ {
			if !isPortAvailable(p) {
				allFree = false
				break
			}
		}
		if allFree {
			return s, s + count - 1
		}
	}
	t.Fatal("could not find enough free ports")
	return 0, 0
}

func TestPortAllocation(t *testing.T) {
	start, end := findAvailablePorts(t, 3)
	pp := NewPortPool(start, end)

	p1, err := pp.Allocate("session-a")
	if err != nil {
		t.Fatalf("allocate 1: %v", err)
	}
	if p1 != start {
		t.Errorf("expected port %d, got %d", start, p1)
	}

	p2, err := pp.Allocate("session-b")
	if err != nil {
		t.Fatalf("allocate 2: %v", err)
	}
	if p2 != start+1 {
		t.Errorf("expected port %d, got %d", start+1, p2)
	}

	p3, err := pp.Allocate("session-c")
	if err != nil {
		t.Fatalf("allocate 3: %v", err)
	}
	if p3 != start+2 {
		t.Errorf("expected port %d, got %d", start+2, p3)
	}

	_, err = pp.Allocate("session-d")
	if err == nil {
		t.Fatal("expected error on 4th allocation, got nil")
	}
}

func TestPortRelease(t *testing.T) {
	start, end := findAvailablePorts(t, 2)
	pp := NewPortPool(start, end)

	p1, _ := pp.Allocate("session-a")
	pp.Allocate("session-b")

	pp.Release("session-a")

	p3, err := pp.Allocate("session-c")
	if err != nil {
		t.Fatalf("allocate after release: %v", err)
	}
	if p3 != p1 {
		t.Errorf("expected released port %d, got %d", p1, p3)
	}
}

func TestConcurrentPortAccess(t *testing.T) {
	start, end := findAvailablePorts(t, 10)
	pp := NewPortPool(start, end)

	var wg sync.WaitGroup
	results := make(chan int, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			port, err := pp.Allocate(fmt.Sprintf("session-%d", id))
			if err != nil {
				errors <- err
				return
			}
			results <- port
		}(i)
	}

	wg.Wait()
	close(results)
	close(errors)

	for err := range errors {
		t.Fatalf("concurrent allocation error: %v", err)
	}

	ports := make(map[int]bool)
	for port := range results {
		if ports[port] {
			t.Errorf("duplicate port %d", port)
		}
		ports[port] = true
	}
	if len(ports) != 10 {
		t.Errorf("expected 10 unique ports, got %d", len(ports))
	}
}

func TestIdempotentAllocation(t *testing.T) {
	start, end := findAvailablePorts(t, 3)
	pp := NewPortPool(start, end)

	p1, err := pp.Allocate("session-a")
	if err != nil {
		t.Fatalf("first allocate: %v", err)
	}

	p2, err := pp.Allocate("session-a")
	if err != nil {
		t.Fatalf("second allocate: %v", err)
	}

	if p1 != p2 {
		t.Errorf("idempotent allocate: expected %d, got %d", p1, p2)
	}
}

func TestSpawnAndExit(t *testing.T) {
	start, end := findAvailablePorts(t, 3)
	tmpDir := t.TempDir()
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	m := NewManager(ManagerConfig{
		WorkspaceRoot: tmpDir,
		RunnerCommand: "echo hello",
		PortStart:     start,
		PortEnd:       end,
		MaxSessions:   10,
	}, logger)

	ctx := context.Background()
	rp, err := m.Spawn(ctx, "test-session", map[string]string{
		"INITIAL_PROMPT": "test",
		"INTERACTIVE":    "false",
	})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	select {
	case evt := <-m.ExitCh:
		if evt.SessionID != "test-session" {
			t.Errorf("expected session test-session, got %s", evt.SessionID)
		}
		if evt.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", evt.ExitCode)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for process exit")
	}

	stdoutLog := filepath.Join(rp.WorkspacePath, "stdout.log")
	data, err := os.ReadFile(stdoutLog)
	if err != nil {
		t.Fatalf("reading stdout log: %v", err)
	}
	if len(data) == 0 {
		t.Error("stdout log is empty, expected output")
	}
}

func TestSpawnFailedProcess(t *testing.T) {
	start, end := findAvailablePorts(t, 3)
	tmpDir := t.TempDir()
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	m := NewManager(ManagerConfig{
		WorkspaceRoot: tmpDir,
		RunnerCommand: "false",
		PortStart:     start,
		PortEnd:       end,
		MaxSessions:   10,
	}, logger)

	ctx := context.Background()
	_, err := m.Spawn(ctx, "fail-session", nil)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	select {
	case evt := <-m.ExitCh:
		if evt.ExitCode == 0 {
			t.Error("expected non-zero exit code")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for exit")
	}
}

func TestSpawnNonexistentCommand(t *testing.T) {
	start, end := findAvailablePorts(t, 3)
	tmpDir := t.TempDir()
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	m := NewManager(ManagerConfig{
		WorkspaceRoot: tmpDir,
		RunnerCommand: "/nonexistent/binary",
		PortStart:     start,
		PortEnd:       end,
		MaxSessions:   10,
	}, logger)

	ctx := context.Background()
	_, err := m.Spawn(ctx, "no-binary", nil)
	if err == nil {
		t.Fatal("expected error spawning nonexistent binary")
	}
}

func TestMaxSessions(t *testing.T) {
	start, end := findAvailablePorts(t, 5)
	tmpDir := t.TempDir()
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	m := NewManager(ManagerConfig{
		WorkspaceRoot: tmpDir,
		RunnerCommand: "sleep 30",
		PortStart:     start,
		PortEnd:       end,
		MaxSessions:   2,
	}, logger)

	ctx := context.Background()
	_, err := m.Spawn(ctx, "s1", nil)
	if err != nil {
		t.Fatalf("spawn 1: %v", err)
	}
	_, err = m.Spawn(ctx, "s2", nil)
	if err != nil {
		t.Fatalf("spawn 2: %v", err)
	}
	_, err = m.Spawn(ctx, "s3", nil)
	if err == nil {
		t.Fatal("expected max sessions error")
	}

	m.Shutdown(ctx)
}

func TestEnvironmentInheritance(t *testing.T) {
	start, end := findAvailablePorts(t, 3)
	tmpDir := t.TempDir()
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	m := NewManager(ManagerConfig{
		WorkspaceRoot: tmpDir,
		RunnerCommand: "env",
		PortStart:     start,
		PortEnd:       end,
		MaxSessions:   10,
		BossURL:       "http://localhost:8899",
		BossSpace:     "test-space",
	}, logger)

	ctx := context.Background()
	rp, err := m.Spawn(ctx, "env-test", map[string]string{
		"LLM_MODEL":       "opus",
		"INTERACTIVE":     "true",
		"INITIAL_PROMPT":  "hello",
		"LLM_TEMPERATURE": "0.7",
		"LLM_MAX_TOKENS":  "4000",
	})
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	select {
	case <-m.ExitCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}

	data, err := os.ReadFile(filepath.Join(rp.WorkspacePath, "stdout.log"))
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	output := string(data)

	for _, expected := range []string{
		"SESSION_ID=env-test",
		fmt.Sprintf("AGUI_PORT=%d", rp.Port),
		"LLM_MODEL=opus",
		"INTERACTIVE=true",
		"BOSS_URL=http://localhost:8899",
		"BOSS_SPACE=test-space",
	} {
		found := false
		for _, line := range splitLines(output) {
			if line == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected env var %q in output", expected)
		}
	}
}

func splitLines(s string) []string {
	var lines []string
	for _, l := range filepath.SplitList(s) {
		lines = append(lines, l)
	}
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	return lines
}

func TestEnvAllowlist(t *testing.T) {
	tests := []struct {
		key     string
		allowed bool
	}{
		{"HOME", true},
		{"PATH", true},
		{"USER", true},
		{"SHELL", true},
		{"LANG", true},
		{"LC_ALL", true},
		{"TMPDIR", true},
		{"GOPATH", true},
		{"GOROOT", true},
		{"SSH_AUTH_SOCK", true},
		{"ANTHROPIC_API_KEY", true},
		{"CLAUDE_CODE_USE_VERTEX", true},
		{"GOOGLE_APPLICATION_CREDENTIALS", true},
		{"GITHUB_TOKEN", true},
		{"GITLAB_TOKEN", true},
		{"GIT_AUTHOR_NAME", true},
		{"RANDOM_VAR", false},
		{"DATABASE_URL", false},
		{"AWS_SECRET_ACCESS_KEY", false},
		{"SOME_PASSWORD", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := isAllowedEnv(tt.key)
			if got != tt.allowed {
				t.Errorf("isAllowedEnv(%q) = %v, want %v", tt.key, got, tt.allowed)
			}
		})
	}
}

func TestGracefulShutdown(t *testing.T) {
	start, end := findAvailablePorts(t, 5)
	tmpDir := t.TempDir()
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	m := NewManager(ManagerConfig{
		WorkspaceRoot: tmpDir,
		RunnerCommand: "sleep 30",
		PortStart:     start,
		PortEnd:       end,
		MaxSessions:   10,
	}, logger)

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, err := m.Spawn(ctx, fmt.Sprintf("shutdown-%d", i), nil)
		if err != nil {
			t.Fatalf("spawn %d: %v", i, err)
		}
	}

	done := make(chan struct{})
	go func() {
		m.Shutdown(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("shutdown timed out")
	}

	m.mu.RLock()
	remaining := len(m.sessions)
	m.mu.RUnlock()
	if remaining != 0 {
		t.Errorf("expected 0 sessions after shutdown, got %d", remaining)
	}
}

func TestStopProcess(t *testing.T) {
	start, end := findAvailablePorts(t, 3)
	tmpDir := t.TempDir()
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	m := NewManager(ManagerConfig{
		WorkspaceRoot: tmpDir,
		RunnerCommand: "sleep 30",
		PortStart:     start,
		PortEnd:       end,
		MaxSessions:   10,
	}, logger)

	ctx := context.Background()
	_, err := m.Spawn(ctx, "stop-test", nil)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	err = m.Stop("stop-test")
	if err != nil {
		t.Fatalf("stop: %v", err)
	}

	select {
	case evt := <-m.ExitCh:
		if evt.SessionID != "stop-test" {
			t.Errorf("expected session stop-test, got %s", evt.SessionID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for exit event")
	}
}

func TestIdempotentSpawn(t *testing.T) {
	start, end := findAvailablePorts(t, 3)
	tmpDir := t.TempDir()
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	m := NewManager(ManagerConfig{
		WorkspaceRoot: tmpDir,
		RunnerCommand: "sleep 30",
		PortStart:     start,
		PortEnd:       end,
		MaxSessions:   10,
	}, logger)

	ctx := context.Background()
	rp1, err := m.Spawn(ctx, "idem-test", nil)
	if err != nil {
		t.Fatalf("spawn 1: %v", err)
	}

	rp2, err := m.Spawn(ctx, "idem-test", nil)
	if err != nil {
		t.Fatalf("spawn 2: %v", err)
	}

	if rp1 != rp2 {
		t.Error("expected same RunnerProcess for idempotent spawn")
	}
	if rp1.Port != rp2.Port {
		t.Errorf("expected same port, got %d and %d", rp1.Port, rp2.Port)
	}

	m.Shutdown(ctx)
}

func TestRingWriter(t *testing.T) {
	rw := newRingWriter(3)
	rw.Write([]byte("line1\nline2\nline3\nline4\nline5\n"))

	result := rw.String()
	expected := "line3\nline4\nline5"
	if result != expected {
		t.Errorf("ring writer: expected %q, got %q", expected, result)
	}
}

func TestRingWriterPartialLines(t *testing.T) {
	rw := newRingWriter(3)
	rw.Write([]byte("partial"))
	rw.Write([]byte(" line\n"))
	rw.Write([]byte("second\n"))

	result := rw.String()
	expected := "partial line\nsecond"
	if result != expected {
		t.Errorf("ring writer partial: expected %q, got %q", expected, result)
	}
}

func TestProcessExitEvent(t *testing.T) {
	start, end := findAvailablePorts(t, 3)
	tmpDir := t.TempDir()
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	scriptPath := filepath.Join(tmpDir, "fail.sh")
	os.WriteFile(scriptPath, []byte("#!/bin/sh\necho error-output >&2\nexit 42\n"), 0o755)

	m := NewManager(ManagerConfig{
		WorkspaceRoot: tmpDir,
		RunnerCommand: scriptPath,
		PortStart:     start,
		PortEnd:       end,
		MaxSessions:   10,
	}, logger)

	ctx := context.Background()
	_, err := m.Spawn(ctx, "exit-test", nil)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	select {
	case evt := <-m.ExitCh:
		if evt.ExitCode != 42 {
			t.Errorf("expected exit code 42, got %d", evt.ExitCode)
		}
		if evt.SessionID != "exit-test" {
			t.Errorf("expected session exit-test, got %s", evt.SessionID)
		}
		if evt.StderrTail == "" {
			t.Error("expected stderr tail to be non-empty")
		}
		if evt.Duration <= 0 {
			t.Error("expected positive duration")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}
