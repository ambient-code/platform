package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/ambient/platform/components/ambient-control-plane/internal/informer"
	"github.com/ambient/platform/components/ambient-control-plane/internal/process"
	"github.com/rs/zerolog"
)

type LocalSessionReconciler struct {
	sdk             *sdkclient.Client
	processManager  *process.Manager
	logger          zerolog.Logger
	lastWritebackAt sync.Map
	healthClient    *http.Client
	shutdownCtx     context.Context
	shutdownCancel  context.CancelFunc
}

func NewLocalSessionReconciler(
	sdk *sdkclient.Client,
	pm *process.Manager,
	logger zerolog.Logger,
) *LocalSessionReconciler {
	ctx, cancel := context.WithCancel(context.Background())
	return &LocalSessionReconciler{
		sdk:            sdk,
		processManager: pm,
		logger:         logger.With().Str("reconciler", "local-sessions").Logger(),
		healthClient:   &http.Client{Timeout: 1 * time.Second},
		shutdownCtx:    ctx,
		shutdownCancel: cancel,
	}
}

func (r *LocalSessionReconciler) Resource() string {
	return "sessions"
}

func (r *LocalSessionReconciler) Reconcile(ctx context.Context, event informer.ResourceEvent) error {
	session, ok := event.Object.(types.Session)
	if !ok {
		r.logger.Warn().
			Str("actual_type", fmt.Sprintf("%T", event.Object)).
			Msg("type assertion failed: expected types.Session")
		return nil
	}

	if session.KubeCrName != "" {
		r.logger.Debug().
			Str("session_id", session.ID).
			Str("kube_cr_name", session.KubeCrName).
			Msg("skipping K8s-managed session")
		return nil
	}

	r.logger.Info().
		Str("event", string(event.Type)).
		Str("session_id", session.ID).
		Str("name", session.Name).
		Str("phase", session.Phase).
		Msg("local session event")

	switch event.Type {
	case informer.EventAdded:
		return r.handleAdded(ctx, session)
	case informer.EventModified:
		return r.handleModified(ctx, session)
	case informer.EventDeleted:
		return r.handleDeleted(ctx, session)
	default:
		return nil
	}
}

func (r *LocalSessionReconciler) handleAdded(ctx context.Context, session types.Session) error {
	if isTerminalPhase(session.Phase) {
		r.logger.Debug().Str("session_id", session.ID).Str("phase", session.Phase).Msg("ignoring terminal phase session")
		return nil
	}

	if session.Phase != PhasePending && session.Phase != "" {
		return nil
	}

	if _, exists := r.processManager.GetProcess(session.ID); exists {
		r.logger.Debug().Str("session_id", session.ID).Msg("process already running, skipping")
		return nil
	}

	return r.spawnSession(ctx, session)
}

func (r *LocalSessionReconciler) handleModified(ctx context.Context, session types.Session) error {
	if r.isWritebackEcho(session) {
		r.logger.Debug().Str("session_id", session.ID).Msg("skipping writeback echo")
		return nil
	}

	if session.Phase == PhaseStopping {
		return r.stopSession(ctx, session)
	}

	if session.Phase == PhasePending {
		if _, exists := r.processManager.GetProcess(session.ID); !exists {
			return r.spawnSession(ctx, session)
		}
	}

	return nil
}

func (r *LocalSessionReconciler) handleDeleted(ctx context.Context, session types.Session) error {
	if _, exists := r.processManager.GetProcess(session.ID); exists {
		if err := r.processManager.Kill(session.ID); err != nil {
			r.logger.Warn().Err(err).Str("session_id", session.ID).Msg("failed to kill process on delete")
		}
	}
	r.lastWritebackAt.Delete(session.ID)
	return nil
}

func (r *LocalSessionReconciler) spawnSession(ctx context.Context, session types.Session) error {
	env := r.buildSessionEnv(session)

	rp, err := r.processManager.Spawn(ctx, session.ID, env)
	if err != nil {
		r.logger.Warn().Err(err).Str("session_id", session.ID).Msg("failed to spawn runner")
		r.writePhase(ctx, session.ID, PhaseFailed, "ProcessSpawnFailed: "+err.Error())
		return nil
	}

	r.writePhase(ctx, session.ID, PhaseCreating, "")

	go r.asyncHealthCheck(ctx, rp, session.ID)

	return nil
}

const (
	healthCheckInterval = 500 * time.Millisecond
	healthCheckAttempts = 30
)

func (r *LocalSessionReconciler) asyncHealthCheck(ctx context.Context, rp *process.RunnerProcess, sessionID string) {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for i := 0; i < healthCheckAttempts; i++ {
		select {
		case <-rp.ExitCh:
			r.logger.Warn().Str("session_id", sessionID).Msg("process exited during health check")
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			resp, err := r.healthClient.Get(fmt.Sprintf("http://127.0.0.1:%d/health", rp.Port))
			if err == nil {
				statusOK := resp.StatusCode == http.StatusOK
				_ = resp.Body.Close()
				if statusOK {
					r.logger.Info().Str("session_id", sessionID).Int("port", rp.Port).Msg("health check passed")
					r.writePhaseWithStartTime(ctx, sessionID, PhaseRunning)
					return
				}
			}
		}
	}

	r.logger.Warn().Str("session_id", sessionID).Msg("health check timed out after 15s")
	r.writePhase(ctx, sessionID, PhaseFailed, "HealthCheckTimeout: no response after 15s")
	_ = r.processManager.Kill(sessionID)
}

func (r *LocalSessionReconciler) stopSession(ctx context.Context, session types.Session) error {
	if _, exists := r.processManager.GetProcess(session.ID); !exists {
		r.writePhaseWithCompletionTime(ctx, session.ID, PhaseStopped)
		return nil
	}

	r.writePhase(ctx, session.ID, PhaseStopping, "")

	go func() {
		if err := r.processManager.Stop(session.ID); err != nil {
			r.logger.Warn().Err(err).Str("session_id", session.ID).Msg("error stopping process")
		}
		r.writePhaseWithCompletionTime(r.shutdownCtx, session.ID, PhaseStopped)
	}()

	return nil
}

func (r *LocalSessionReconciler) Close() {
	r.shutdownCancel()
}

func (r *LocalSessionReconciler) HandleProcessExit(event process.ProcessExitEvent) {
	ctx := r.shutdownCtx

	phase := PhaseCompleted
	if event.ExitCode != 0 {
		phase = PhaseFailed
	}

	reason := ""
	if event.ExitCode != 0 {
		reason = fmt.Sprintf("ProcessExited: exit code %d", event.ExitCode)
	}

	now := time.Now()
	patch := types.NewSessionStatusPatchBuilder().
		Phase(phase).
		CompletionTime(&now).
		Conditions(buildConditions(phase, reason)).
		Build()

	r.patchStatus(ctx, event.SessionID, patch)
}

func (r *LocalSessionReconciler) ReapLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-r.processManager.ExitCh:
			if !ok {
				return
			}
			r.HandleProcessExit(event)
		}
	}
}

func (r *LocalSessionReconciler) buildSessionEnv(session types.Session) map[string]string {
	env := map[string]string{}

	if session.Name != "" {
		env["AGENTIC_SESSION_NAME"] = session.Name
		env["BOSS_AGENT_NAME"] = slugify(session.Name)
	}
	if session.Prompt != "" {
		env["INITIAL_PROMPT"] = session.Prompt
	}
	if session.LlmModel != "" {
		env["LLM_MODEL"] = session.LlmModel
	}
	if session.LlmTemperature != 0 {
		env["LLM_TEMPERATURE"] = fmt.Sprintf("%g", session.LlmTemperature)
	}
	if session.LlmMaxTokens != 0 {
		env["LLM_MAX_TOKENS"] = fmt.Sprintf("%d", session.LlmMaxTokens)
	}
	if session.Timeout != 0 {
		env["SESSION_TIMEOUT"] = fmt.Sprintf("%d", session.Timeout)
	}
	if session.Repos != "" {
		env["REPOS_JSON"] = session.Repos
	}
	if session.ProjectID != "" {
		env["PROJECT_NAME"] = session.ProjectID
	}

	return env
}

func (r *LocalSessionReconciler) writePhase(ctx context.Context, sessionID, phase, reason string) {
	patch := types.NewSessionStatusPatchBuilder().
		Phase(phase).
		Conditions(buildConditions(phase, reason)).
		Build()
	r.patchStatus(ctx, sessionID, patch)
}

func (r *LocalSessionReconciler) writePhaseWithStartTime(ctx context.Context, sessionID, phase string) {
	now := time.Now()
	patch := types.NewSessionStatusPatchBuilder().
		Phase(phase).
		StartTime(&now).
		Conditions(buildConditions(phase, "")).
		Build()
	r.patchStatus(ctx, sessionID, patch)
}

func (r *LocalSessionReconciler) writePhaseWithCompletionTime(ctx context.Context, sessionID, phase string) {
	now := time.Now()
	patch := types.NewSessionStatusPatchBuilder().
		Phase(phase).
		CompletionTime(&now).
		Conditions(buildConditions(phase, "")).
		Build()
	r.patchStatus(ctx, sessionID, patch)
}

func (r *LocalSessionReconciler) patchStatus(ctx context.Context, sessionID string, patch map[string]any) {
	if r.sdk == nil {
		return
	}

	response, err := r.sdk.Sessions().UpdateStatus(ctx, sessionID, patch)
	if err != nil {
		r.logger.Warn().Err(err).Str("session_id", sessionID).Msg("failed to write status back to API server")
		return
	}

	if response != nil && response.UpdatedAt != nil {
		r.lastWritebackAt.Store(sessionID, response.UpdatedAt.Truncate(time.Microsecond))
	}

	phase, _ := patch["phase"].(string)
	r.logger.Info().Str("session_id", sessionID).Str("phase", phase).Msg("wrote status to API server")
}

func (r *LocalSessionReconciler) isWritebackEcho(session types.Session) bool {
	if session.ID == "" || session.UpdatedAt == nil {
		return false
	}
	val, ok := r.lastWritebackAt.Load(session.ID)
	if !ok {
		return false
	}
	lastWB, ok2 := val.(time.Time)
	if !ok2 {
		return false
	}
	return session.UpdatedAt.Truncate(time.Microsecond).Equal(lastWB)
}

func isTerminalPhase(phase string) bool {
	for _, tp := range TerminalPhases {
		if phase == tp {
			return true
		}
	}
	return false
}

func slugify(name string) string {
	var result []byte
	for _, c := range []byte(name) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result = append(result, c)
		} else if c >= 'A' && c <= 'Z' {
			result = append(result, c+32)
		} else if c == ' ' || c == '_' {
			result = append(result, '-')
		}
	}
	return string(result)
}

type conditionEntry struct {
	Type   string `json:"type"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func buildConditions(phase, reason string) string {
	processSpawned := "True"
	healthPassed := "Unknown"
	runnerStarted := "Unknown"

	switch phase {
	case PhaseCreating:
		healthPassed = "False"
		runnerStarted = "False"
	case PhaseRunning:
		healthPassed = "True"
		runnerStarted = "True"
	case PhaseFailed:
		if reason != "" {
			if strings.HasPrefix(reason, "Process") {
				processSpawned = "False"
			}
			healthPassed = "False"
			runnerStarted = "False"
		}
	case PhaseStopped:
		healthPassed = "True"
		runnerStarted = "False"
	case PhaseCompleted:
		healthPassed = "True"
		runnerStarted = "False"
	}

	conditions := []conditionEntry{
		{Type: "ProcessSpawned", Status: processSpawned},
		{Type: "HealthCheckPassed", Status: healthPassed},
		{Type: "RunnerStarted", Status: runnerStarted, Reason: reason},
	}

	data, err := json.Marshal(conditions)
	if err != nil {
		return "[]"
	}
	return string(data)
}
