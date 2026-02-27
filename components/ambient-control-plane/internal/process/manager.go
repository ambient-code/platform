package process

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
)

var envAllowlistPrefixes = []string{
	"ANTHROPIC_",
	"CLAUDE_",
	"GOOGLE_",
	"GITHUB_",
	"GITLAB_",
	"GIT_",
}

var envAllowlistExact = []string{
	"HOME",
	"PATH",
	"USER",
	"SHELL",
	"LANG",
	"LC_ALL",
	"TMPDIR",
	"GOPATH",
	"GOROOT",
	"SSH_AUTH_SOCK",
}

type ProcessExitEvent struct {
	SessionID  string
	ExitCode   int
	StderrTail string
	Duration   time.Duration
}

type RunnerProcess struct {
	SessionID     string
	Port          int
	Cmd           *exec.Cmd
	Cancel        context.CancelFunc
	StartedAt     time.Time
	WorkspacePath string
	ExitCh        chan struct{}
	exitCode      *int
	mu            sync.Mutex
}

func (rp *RunnerProcess) ExitCode() *int {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if rp.exitCode == nil {
		return nil
	}
	v := *rp.exitCode
	return &v
}

type Manager struct {
	mu            sync.RWMutex
	sessions      map[string]*RunnerProcess
	portPool      *PortPool
	workspaceRoot string
	runnerCmd     string
	logger        zerolog.Logger
	maxSessions   int
	ExitCh        chan ProcessExitEvent
	bossURL       string
	bossSpace     string
}

type ManagerConfig struct {
	WorkspaceRoot string
	RunnerCommand string
	PortStart     int
	PortEnd       int
	MaxSessions   int
	BossURL       string
	BossSpace     string
}

func NewManager(cfg ManagerConfig, logger zerolog.Logger) *Manager {
	return &Manager{
		sessions:      make(map[string]*RunnerProcess),
		portPool:      NewPortPool(cfg.PortStart, cfg.PortEnd),
		workspaceRoot: cfg.WorkspaceRoot,
		runnerCmd:     cfg.RunnerCommand,
		logger:        logger.With().Str("component", "process-manager").Logger(),
		maxSessions:   cfg.MaxSessions,
		ExitCh:        make(chan ProcessExitEvent, cfg.MaxSessions),
		bossURL:       cfg.BossURL,
		bossSpace:     cfg.BossSpace,
	}
}

func (m *Manager) Spawn(ctx context.Context, sessionID string, env map[string]string) (*RunnerProcess, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.sessions[sessionID]; ok {
		return existing, nil
	}

	if len(m.sessions) >= m.maxSessions {
		return nil, fmt.Errorf("max sessions reached (%d)", m.maxSessions)
	}

	port, err := m.portPool.Allocate(sessionID)
	if err != nil {
		return nil, fmt.Errorf("allocating port: %w", err)
	}

	workspace := filepath.Join(m.workspaceRoot, sessionID)
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		m.portPool.Release(sessionID)
		return nil, fmt.Errorf("creating workspace %s: %w", workspace, err)
	}

	stdoutPath := filepath.Join(workspace, "stdout.log")
	stderrPath := filepath.Join(workspace, "stderr.log")
	stdoutFile, err := os.OpenFile(stdoutPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		m.portPool.Release(sessionID)
		return nil, fmt.Errorf("creating stdout log: %w", err)
	}
	stderrFile, err := os.OpenFile(stderrPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		stdoutFile.Close()
		m.portPool.Release(sessionID)
		return nil, fmt.Errorf("creating stderr log: %w", err)
	}

	stderrRing := newRingWriter(50)

	procCtx, cancel := context.WithCancel(ctx)

	envSlice := m.buildEnv(sessionID, port, workspace, env)

	parts := splitCommand(m.runnerCmd)
	cmd := exec.CommandContext(procCtx, parts[0], parts[1:]...)
	cmd.Dir = workspace
	cmd.Env = envSlice
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	loggerStdout := m.logger.With().Str("session_id", sessionID).Str("stream", "stdout").Logger()
	loggerStderr := m.logger.With().Str("session_id", sessionID).Str("stream", "stderr").Logger()

	cmd.Stdout = io.MultiWriter(stdoutFile, &zerologLineWriter{logger: loggerStdout})
	cmd.Stderr = io.MultiWriter(stderrFile, stderrRing, &zerologLineWriter{logger: loggerStderr})

	if err := cmd.Start(); err != nil {
		cancel()
		stdoutFile.Close()
		stderrFile.Close()
		m.portPool.Release(sessionID)
		return nil, fmt.Errorf("starting runner process: %w", err)
	}

	rp := &RunnerProcess{
		SessionID:     sessionID,
		Port:          port,
		Cmd:           cmd,
		Cancel:        cancel,
		StartedAt:     time.Now(),
		WorkspacePath: workspace,
		ExitCh:        make(chan struct{}),
	}
	m.sessions[sessionID] = rp

	m.logger.Info().
		Str("session_id", sessionID).
		Int("port", port).
		Int("pid", cmd.Process.Pid).
		Str("workspace", workspace).
		Msg("spawned runner process")

	go m.waitForExit(rp, stderrRing, stdoutFile, stderrFile)

	return rp, nil
}

func (m *Manager) Kill(sessionID string) error {
	m.mu.RLock()
	rp, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}

	return m.killProcess(rp)
}

func (m *Manager) Stop(sessionID string) error {
	m.mu.RLock()
	rp, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}

	if rp.Cmd.Process == nil {
		return nil
	}

	pgid, err := syscall.Getpgid(rp.Cmd.Process.Pid)
	if err != nil {
		return m.killProcess(rp)
	}

	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
		return m.killProcess(rp)
	}

	select {
	case <-rp.ExitCh:
		return nil
	case <-time.After(10 * time.Second):
		m.logger.Warn().
			Str("session_id", sessionID).
			Msg("process did not exit after SIGTERM, sending SIGKILL")
		return m.killProcess(rp)
	}
}

func (m *Manager) killProcess(rp *RunnerProcess) error {
	if rp.Cmd.Process == nil {
		return nil
	}
	pgid, err := syscall.Getpgid(rp.Cmd.Process.Pid)
	if err != nil {
		rp.Cancel()
		return nil
	}
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
	<-rp.ExitCh
	return nil
}

func (m *Manager) GetProcess(sessionID string) (*RunnerProcess, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rp, ok := m.sessions[sessionID]
	return rp, ok
}

func (m *Manager) Shutdown(ctx context.Context) {
	m.mu.RLock()
	sessions := make([]*RunnerProcess, 0, len(m.sessions))
	for _, rp := range m.sessions {
		sessions = append(sessions, rp)
	}
	m.mu.RUnlock()

	if len(sessions) == 0 {
		return
	}

	m.logger.Info().Int("count", len(sessions)).Msg("shutting down all runner processes")

	for _, rp := range sessions {
		if rp.Cmd.Process != nil {
			pgid, err := syscall.Getpgid(rp.Cmd.Process.Pid)
			if err == nil {
				_ = syscall.Kill(-pgid, syscall.SIGTERM)
			}
		}
	}

	done := make(chan struct{})
	go func() {
		for _, rp := range sessions {
			<-rp.ExitCh
		}
		close(done)
	}()

	deadline := time.After(10 * time.Second)
	if d, ok := ctx.Deadline(); ok {
		deadline = time.After(time.Until(d))
	}

	select {
	case <-done:
		m.logger.Info().Msg("all processes exited gracefully")
	case <-ctx.Done():
		m.logger.Warn().Msg("context cancelled, sending SIGKILL to remaining processes")
		for _, rp := range sessions {
			select {
			case <-rp.ExitCh:
			default:
				if rp.Cmd.Process != nil {
					pgid, err := syscall.Getpgid(rp.Cmd.Process.Pid)
					if err == nil {
						_ = syscall.Kill(-pgid, syscall.SIGKILL)
					}
				}
			}
		}
		for _, rp := range sessions {
			<-rp.ExitCh
		}
	case <-deadline:
		m.logger.Warn().Msg("shutdown deadline reached, sending SIGKILL to remaining processes")
		for _, rp := range sessions {
			select {
			case <-rp.ExitCh:
			default:
				if rp.Cmd.Process != nil {
					pgid, err := syscall.Getpgid(rp.Cmd.Process.Pid)
					if err == nil {
						_ = syscall.Kill(-pgid, syscall.SIGKILL)
					}
				}
			}
		}
		for _, rp := range sessions {
			<-rp.ExitCh
		}
	}
}

func (m *Manager) cleanup(sessionID string) {
	m.mu.Lock()
	delete(m.sessions, sessionID)
	m.mu.Unlock()
	m.portPool.Release(sessionID)
}

func (m *Manager) waitForExit(rp *RunnerProcess, stderrRing *ringWriter, stdoutFile, stderrFile *os.File) {
	err := rp.Cmd.Wait()
	duration := time.Since(rp.StartedAt)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	rp.mu.Lock()
	rp.exitCode = &exitCode
	rp.mu.Unlock()

	stdoutFile.Close()
	stderrFile.Close()

	m.logger.Info().
		Str("session_id", rp.SessionID).
		Int("exit_code", exitCode).
		Dur("duration", duration).
		Msg("runner process exited")

	close(rp.ExitCh)

	m.cleanup(rp.SessionID)

	m.ExitCh <- ProcessExitEvent{
		SessionID:  rp.SessionID,
		ExitCode:   exitCode,
		StderrTail: stderrRing.String(),
		Duration:   duration,
	}
}

func (m *Manager) buildEnv(sessionID string, port int, workspace string, sessionEnv map[string]string) []string {
	allowed := m.filteredHostEnv()

	allowed = append(allowed, fmt.Sprintf("SESSION_ID=%s", sessionID))
	allowed = append(allowed, fmt.Sprintf("AGUI_PORT=%d", port))
	allowed = append(allowed, fmt.Sprintf("WORKSPACE_PATH=%s", workspace))

	if m.bossURL != "" {
		allowed = append(allowed, fmt.Sprintf("BOSS_URL=%s", m.bossURL))
	}
	if m.bossSpace != "" {
		allowed = append(allowed, fmt.Sprintf("BOSS_SPACE=%s", m.bossSpace))
	}

	for k, v := range sessionEnv {
		allowed = append(allowed, fmt.Sprintf("%s=%s", k, v))
	}

	return allowed
}

func (m *Manager) filteredHostEnv() []string {
	var result []string
	for _, env := range os.Environ() {
		key, _, ok := strings.Cut(env, "=")
		if !ok {
			continue
		}
		if isAllowedEnv(key) {
			result = append(result, env)
		}
	}
	return result
}

func isAllowedEnv(key string) bool {
	for _, exact := range envAllowlistExact {
		if key == exact {
			return true
		}
	}
	for _, prefix := range envAllowlistPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

type PortPool struct {
	mu    sync.Mutex
	start int
	end   int
	used  map[int]string
	byID  map[string]int
}

func NewPortPool(start, end int) *PortPool {
	return &PortPool{
		start: start,
		end:   end,
		used:  make(map[int]string),
		byID:  make(map[string]int),
	}
}

func (pp *PortPool) Allocate(sessionID string) (int, error) {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	if port, ok := pp.byID[sessionID]; ok {
		return port, nil
	}

	for port := pp.start; port <= pp.end; port++ {
		if _, inUse := pp.used[port]; inUse {
			continue
		}
		if !isPortAvailable(port) {
			continue
		}
		pp.used[port] = sessionID
		pp.byID[sessionID] = port
		return port, nil
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", pp.start, pp.end)
}

func (pp *PortPool) Release(sessionID string) {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	port, ok := pp.byID[sessionID]
	if !ok {
		return
	}
	delete(pp.used, port)
	delete(pp.byID, sessionID)
}

func (pp *PortPool) PortForSession(sessionID string) (int, bool) {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	port, ok := pp.byID[sessionID]
	return port, ok
}

func splitCommand(cmd string) []string {
	var parts []string
	var current []byte
	var inQuote byte

	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		if inQuote != 0 {
			if c == inQuote {
				inQuote = 0
			} else {
				current = append(current, c)
			}
		} else if c == '"' || c == '\'' {
			inQuote = c
		} else if c == ' ' || c == '\t' {
			if len(current) > 0 {
				parts = append(parts, string(current))
				current = current[:0]
			}
		} else {
			current = append(current, c)
		}
	}
	if len(current) > 0 {
		parts = append(parts, string(current))
	}
	if len(parts) == 0 {
		return strings.Fields(cmd)
	}
	return parts
}

func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

type ringWriter struct {
	mu    sync.Mutex
	lines []string
	max   int
	pos   int
	full  bool
	buf   []byte
}

func newRingWriter(maxLines int) *ringWriter {
	return &ringWriter{
		lines: make([]string, maxLines),
		max:   maxLines,
	}
}

func (rw *ringWriter) Write(p []byte) (n int, err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	rw.buf = append(rw.buf, p...)
	for {
		idx := -1
		for i, b := range rw.buf {
			if b == '\n' {
				idx = i
				break
			}
		}
		if idx < 0 {
			break
		}
		line := string(rw.buf[:idx])
		rw.buf = rw.buf[idx+1:]
		rw.lines[rw.pos] = line
		rw.pos = (rw.pos + 1) % rw.max
		if rw.pos == 0 {
			rw.full = true
		}
	}
	return len(p), nil
}

func (rw *ringWriter) String() string {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	var result []string
	if rw.full {
		for i := 0; i < rw.max; i++ {
			idx := (rw.pos + i) % rw.max
			if rw.lines[idx] != "" {
				result = append(result, rw.lines[idx])
			}
		}
	} else {
		for i := 0; i < rw.pos; i++ {
			if rw.lines[i] != "" {
				result = append(result, rw.lines[i])
			}
		}
	}
	if len(rw.buf) > 0 {
		result = append(result, string(rw.buf))
	}
	return strings.Join(result, "\n")
}

type zerologLineWriter struct {
	logger zerolog.Logger
	buf    []byte
}

func (w *zerologLineWriter) Write(p []byte) (n int, err error) {
	w.buf = append(w.buf, p...)
	for {
		idx := -1
		for i, b := range w.buf {
			if b == '\n' {
				idx = i
				break
			}
		}
		if idx < 0 {
			break
		}
		line := string(w.buf[:idx])
		w.buf = w.buf[idx+1:]
		w.logger.Debug().Msg(line)
	}
	return len(p), nil
}
