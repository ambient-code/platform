package reconciler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ambient-code/platform/components/ambient-control-plane/internal/informer"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/rs/zerolog"
)

type SessionTally struct {
	TotalSessions   int            `json:"total_sessions"`
	SessionsByPhase map[string]int `json:"sessions_by_phase"`
	SessionsByUser  map[string]int `json:"sessions_by_user"`
	LastUpdated     time.Time      `json:"last_updated"`
}

type SessionTallyReconciler struct {
	logger zerolog.Logger
	tally  SessionTally
	mu     sync.RWMutex
}

func NewSessionTallyReconciler(logger zerolog.Logger) *SessionTallyReconciler {
	return &SessionTallyReconciler{
		logger: logger.With().Str("reconciler", "session-tally").Logger(),
		tally: SessionTally{
			SessionsByPhase: make(map[string]int),
			SessionsByUser:  make(map[string]int),
		},
	}
}

func (r *SessionTallyReconciler) Resource() string {
	return "sessions"
}

func (r *SessionTallyReconciler) Reconcile(ctx context.Context, event informer.ResourceEvent) error {
	if event.Object.Session == nil {
		r.logger.Warn().Msg("expected session object in session event")
		return nil
	}
	session := *event.Object.Session

	r.logger.Debug().
		Str("event", string(event.Type)).
		Str("session_id", session.ID).
		Str("phase", session.Phase).
		Str("user", session.CreatedByUserID).
		Msg("tally reconciler: processing session event")

	r.mu.Lock()
	defer r.mu.Unlock()

	switch event.Type {
	case informer.EventAdded:
		r.handleSessionAdded(session)
	case informer.EventModified:
		r.handleSessionModified(session)
	case informer.EventDeleted:
		r.handleSessionDeleted(session)
	}

	r.tally.LastUpdated = time.Now()
	r.logCurrentTally()
	return nil
}

func (r *SessionTallyReconciler) handleSessionAdded(session types.Session) {
	r.tally.TotalSessions++

	if session.Phase != "" {
		r.tally.SessionsByPhase[session.Phase]++
	}

	if session.CreatedByUserID != "" {
		r.tally.SessionsByUser[session.CreatedByUserID]++
	}

	r.logger.Info().
		Str("session_id", session.ID).
		Str("phase", session.Phase).
		Str("user", session.CreatedByUserID).
		Int("total_sessions", r.tally.TotalSessions).
		Msg("session added to tally")
}

func (r *SessionTallyReconciler) handleSessionModified(session types.Session) {
	r.logger.Debug().
		Str("session_id", session.ID).
		Str("phase", session.Phase).
		Msg("session modified - tally unchanged (only tracks adds/deletes)")
}

func (r *SessionTallyReconciler) handleSessionDeleted(session types.Session) {
	r.tally.TotalSessions--

	if session.Phase != "" && r.tally.SessionsByPhase[session.Phase] > 0 {
		r.tally.SessionsByPhase[session.Phase]--
		if r.tally.SessionsByPhase[session.Phase] == 0 {
			delete(r.tally.SessionsByPhase, session.Phase)
		}
	}

	if session.CreatedByUserID != "" && r.tally.SessionsByUser[session.CreatedByUserID] > 0 {
		r.tally.SessionsByUser[session.CreatedByUserID]--
		if r.tally.SessionsByUser[session.CreatedByUserID] == 0 {
			delete(r.tally.SessionsByUser, session.CreatedByUserID)
		}
	}

	r.logger.Info().
		Str("session_id", session.ID).
		Str("phase", session.Phase).
		Str("user", session.CreatedByUserID).
		Int("total_sessions", r.tally.TotalSessions).
		Msg("session removed from tally")
}

func (r *SessionTallyReconciler) logCurrentTally() {
	logEvent := r.logger.Info().
		Int("total_sessions", r.tally.TotalSessions).
		Time("last_updated", r.tally.LastUpdated)

	if len(r.tally.SessionsByPhase) > 0 {
		phaseStr := r.formatMap(r.tally.SessionsByPhase)
		logEvent = logEvent.Str("sessions_by_phase", phaseStr)
	}

	if len(r.tally.SessionsByUser) > 0 {
		userStr := r.formatMap(r.tally.SessionsByUser)
		logEvent = logEvent.Str("sessions_by_user", userStr)
	}

	logEvent.Msg("current session tally")
}

func (r *SessionTallyReconciler) formatMap(m map[string]int) string {
	if len(m) == 0 {
		return "{}"
	}

	result := "{"
	first := true
	for k, v := range m {
		if !first {
			result += ", "
		}
		result += fmt.Sprintf("%s:%d", k, v)
		first = false
	}
	result += "}"
	return result
}

func (r *SessionTallyReconciler) GetCurrentTally() SessionTally {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tally := SessionTally{
		TotalSessions:   r.tally.TotalSessions,
		SessionsByPhase: make(map[string]int),
		SessionsByUser:  make(map[string]int),
		LastUpdated:     r.tally.LastUpdated,
	}

	for k, v := range r.tally.SessionsByPhase {
		tally.SessionsByPhase[k] = v
	}
	for k, v := range r.tally.SessionsByUser {
		tally.SessionsByUser[k] = v
	}

	return tally
}

func (r *SessionTallyReconciler) ResetTally() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tally.TotalSessions = 0
	r.tally.SessionsByPhase = make(map[string]int)
	r.tally.SessionsByUser = make(map[string]int)
	r.tally.LastUpdated = time.Now()

	r.logger.Info().Msg("session tally reset to zero")
}
