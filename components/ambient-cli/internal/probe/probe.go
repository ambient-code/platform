package probe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

// Log is a printf-style logger injected by the caller.
type Log func(format string, args ...interface{})

// State carries all mutable context across steps.
type State struct {
	Client          *sdkclient.Client
	BaseURL         string
	Token           string
	Project         string
	Log             Log
	pf              *portForward

	ProjectResource *types.Project
	Agent           *types.Agent
	SessionID       string
	IgnitionPrompt  string
}

// Step is a named unit of work.
type Step struct {
	Name string
	Fn   func(ctx context.Context, s *State) error
}

// Steps returns the ordered probe sequence.
func Steps() []Step {
	return []Step{
		{"check API reachability", stepCheckReachability},
		{"create project", stepCreateProject},
		{"create agent", stepCreateAgent},
		{"post blackboard document", stepPostBlackboard},
		{"get project snapshot (before ignite)", stepSnapshotBefore},
		{"ignite agent", stepIgniteAgent},
		{"wait for session Running", stepWaitRunning},
		{"get project snapshot (after ignite)", stepSnapshotAfter},
		{"wait for ignition run, then send message and stream response", stepSendAndStream},
		{"get project snapshot (final)", stepSnapshotFinal},
		{"cleanup", stepCleanup},
	}
}

// Run executes steps in sequence, stopping on the first error.
// Cleanup always runs even on failure when a session or project was created.
func Run(ctx context.Context, s *State) error {
	steps := Steps()
	cleanupIdx := len(steps) - 1

	for i, step := range steps {
		s.Log("── step %d/%d: %s", i+1, len(steps), step.Name)
		if err := step.Fn(ctx, s); err != nil {
			s.Log("   FAIL: %v", err)
			if i < cleanupIdx && (s.SessionID != "" || s.Agent != nil || s.ProjectResource != nil) {
				s.Log("── running cleanup after failure")
				_ = stepCleanup(ctx, s)
			}
			return fmt.Errorf("step %q: %w", step.Name, err)
		}
		s.Log("   ok")
	}
	return nil
}

// ── Steps ─────────────────────────────────────────────────────────────────────

func stepCheckReachability(ctx context.Context, s *State) error {
	pf, err := ensureAPIReachable(s.BaseURL, "ambient-code")
	if err != nil {
		return err
	}
	if pf != nil {
		s.pf = pf
		s.Log("   port-forward established → %s", s.BaseURL)
	} else {
		s.Log("   API already reachable → %s", s.BaseURL)
	}
	return nil
}

func stepCreateProject(ctx context.Context, s *State) error {
	runID := fmt.Sprintf("%d", time.Now().Unix()%100000)
	name := "probe-" + runID
	s.Log("   POST /projects  name=%s", name)
	proj, err := s.Client.Projects().Create(ctx, &types.Project{
		Name:        name,
		Description: "Single-agent API probe",
	})
	if err != nil {
		return err
	}
	s.ProjectResource = proj
	s.Log("   project id=%s name=%s", proj.ID, proj.Name)
	return nil
}

func stepCreateAgent(ctx context.Context, s *State) error {
	s.Log("   POST /agents  project=%s", s.ProjectResource.Name)
	a, err := s.Client.Agents().Create(ctx, &types.Agent{
		Name:        "assistant",
		Prompt:      "You are a helpful assistant. Respond concisely.",
		OwnerUserID: "dev-user",
	})
	if err != nil {
		return err
	}
	s.Agent = a
	s.Log("   agent id=%s", a.ID)
	return nil
}

func stepPostBlackboard(ctx context.Context, s *State) error {
	s.Log("   PUT /projects/%s/documents/blackboard", s.ProjectResource.Name)
	content := fmt.Sprintf("# Probe Blackboard\n\nProject: %s\nAgent: %s (%s)\n\nThis is a connectivity probe.\n",
		s.ProjectResource.Name, s.Agent.Name, s.Agent.ID)
	_, err := s.Client.ProjectDocuments().Create(ctx, &types.ProjectDocument{
		ProjectID: s.ProjectResource.Name,
		Slug:      "blackboard",
		Title:     "Blackboard",
		Content:   content,
	})
	return err
}

func stepSnapshotBefore(ctx context.Context, s *State) error {
	return printSnapshot(ctx, s, "before ignite")
}

func stepIgniteAgent(ctx context.Context, s *State) error {
	s.Log("   POST /agents/%s/ignite", s.Agent.ID)
	sub := newHTTPClient(s.BaseURL, s.Token, s.Project)
	ig, err := sub.ignite(ctx, s.Agent.ID)
	if err != nil {
		return err
	}
	s.SessionID = ig.Session.ID
	s.IgnitionPrompt = ig.IgnitionPrompt
	s.Log("   session id=%s", s.SessionID)
	s.Log("   ignition prompt length=%d chars", len(s.IgnitionPrompt))
	return nil
}

func stepWaitRunning(ctx context.Context, s *State) error {
	s.Log("   polling session phase (target: Running)…")
	deadline := time.Now().Add(2 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			sess, err := s.Client.Sessions().Get(ctx, s.SessionID)
			if err != nil {
				s.Log("   poll error: %v (retrying)", err)
				continue
			}
			s.Log("   phase=%s", sess.Phase)
			switch sess.Phase {
			case "Running":
				return nil
			case "Failed", "Stopped":
				return fmt.Errorf("session reached terminal phase %q before Running", sess.Phase)
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timed out waiting for Running (last phase: %s)", sess.Phase)
			}
		}
	}
}

func stepSnapshotAfter(ctx context.Context, s *State) error {
	return printSnapshot(ctx, s, "after ignite")
}


func stepSnapshotFinal(ctx context.Context, s *State) error {
	return printSnapshot(ctx, s, "final")
}

func stepCleanup(ctx context.Context, s *State) error {
	var errs []string

	if s.SessionID != "" {
		s.Log("   POST /sessions/%s/stop", s.SessionID)
		if _, err := s.Client.Sessions().Stop(ctx, s.SessionID); err != nil {
			errs = append(errs, "stop session: "+err.Error())
		}
		s.Log("   DELETE /sessions/%s", s.SessionID)
		if err := s.Client.Sessions().Delete(ctx, s.SessionID); err != nil {
			errs = append(errs, "delete session: "+err.Error())
		}
		s.SessionID = ""
	}

	if s.Agent != nil {
		s.Log("   DELETE /agents/%s", s.Agent.ID)
		if err := s.Client.Agents().Delete(ctx, s.Agent.ID); err != nil {
			errs = append(errs, "delete agent: "+err.Error())
		}
		s.Agent = nil
	}

	if s.ProjectResource != nil {
		s.Log("   DELETE /projects/%s", s.ProjectResource.ID)
		if err := s.Client.Projects().Delete(ctx, s.ProjectResource.ID); err != nil {
			errs = append(errs, "delete project: "+err.Error())
		}
		s.ProjectResource = nil
	}

	if s.pf != nil {
		s.pf.stop()
		s.pf = nil
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func printSnapshot(ctx context.Context, s *State, label string) error {
	s.Log("   GET /projects/%s/home/snapshot (%s)", s.ProjectResource.Name, label)
	sub := newHTTPClient(s.BaseURL, s.Token, s.Project)
	snap, err := sub.snapshot(ctx, s.ProjectResource.Name)
	if err != nil {
		return err
	}
	for _, row := range snap.Agents {
		phase := "—"
		summary := "—"
		if row.CheckIn != nil {
			var ci sessionCheckIn
			if err := json.Unmarshal(row.CheckIn, &ci); err == nil {
				phase = ci.Phase
				summary = ci.Summary
			}
		}
		s.Log("   agent=%-12s  phase=%-12s  summary=%s", row.Agent.Name, phase, firstN(summary, 60))
	}
	return nil
}

func firstN(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

// ── Thin HTTP client for endpoints not in the SDK ─────────────────────────────

type httpClient struct {
	cli     *http.Client
	baseURL string
	token   string
	project string
}

func newHTTPClient(baseURL, token, project string) *httpClient {
	return &httpClient{cli: &http.Client{}, baseURL: baseURL, token: token, project: project}
}

func (h *httpClient) do(ctx context.Context, method, path string, body interface{}, result interface{}, wantStatus int) error {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, h.baseURL+"/api/ambient/v1"+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+h.token)
	req.Header.Set("X-Ambient-Project", h.project)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := h.cli.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		var apiErr map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return fmt.Errorf("HTTP %d: %v", resp.StatusCode, apiErr)
	}
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode: %w", err)
		}
	}
	return nil
}

type igniteResponse struct {
	Session        types.Session `json:"session"`
	IgnitionPrompt string        `json:"ignition_prompt"`
}

func (h *httpClient) ignite(ctx context.Context, agentID string) (*igniteResponse, error) {
	var result igniteResponse
	err := h.do(ctx, http.MethodPost, "/agents/"+agentID+"/ignite", map[string]any{}, &result, http.StatusCreated)
	return &result, err
}

type sessionCheckIn struct {
	Phase   string `json:"phase"`
	Summary string `json:"summary"`
}

type agentCheckInRow struct {
	Agent   types.Agent     `json:"agent"`
	CheckIn json.RawMessage `json:"check_in,omitempty"`
}

type projectSnapshot struct {
	Kind      string            `json:"kind"`
	ProjectID string            `json:"project_id"`
	Agents    []agentCheckInRow `json:"agents"`
}

func (h *httpClient) snapshot(ctx context.Context, projectID string) (*projectSnapshot, error) {
	var result projectSnapshot
	err := h.do(ctx, http.MethodGet, "/projects/"+projectID+"/home/snapshot", nil, &result, http.StatusOK)
	return &result, err
}

// ── Port-forward ──────────────────────────────────────────────────────────────

type portForward struct {
	cmd *exec.Cmd
}

func (pf *portForward) stop() {
	if pf != nil && pf.cmd != nil && pf.cmd.Process != nil {
		_ = pf.cmd.Process.Kill()
		_ = pf.cmd.Wait()
	}
}

func ensureAPIReachable(apiURL, namespace string) (*portForward, error) {
	host, port, err := parseHostPort(apiURL)
	if err != nil {
		return nil, fmt.Errorf("parse API URL: %w", err)
	}
	if tcpReachable(host, port, 500*time.Millisecond) {
		return nil, nil
	}
	if !isLocalhost(host) {
		return nil, fmt.Errorf("API at %s:%s is not reachable", host, port)
	}
	cmd := exec.Command("kubectl", "port-forward", "svc/ambient-api-server", port+":8000", "-n", namespace)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start port-forward: %w", err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if tcpReachable(host, port, 300*time.Millisecond) {
			return &portForward{cmd: cmd}, nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	_ = cmd.Process.Kill()
	return nil, fmt.Errorf("port-forward timed out after 15s")
}

func parseHostPort(apiURL string) (host, port string, err error) {
	u, err := url.Parse(apiURL)
	if err != nil {
		return "", "", err
	}
	host = u.Hostname()
	port = u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return host, port, nil
}

func isLocalhost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func tcpReachable(host, port string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

var _ = sdkclient.NewClient
