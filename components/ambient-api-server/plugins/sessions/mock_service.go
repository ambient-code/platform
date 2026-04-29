package sessions

import (
	"context"
	"sync"
	"time"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

// InMemorySessionService is a zero-dependency service for unit tests.
// It stores state in a map and never touches the database.
type InMemorySessionService struct {
	mu   sync.RWMutex
	data map[string]*Session
}

var _ SessionService = &InMemorySessionService{}

func NewInMemorySessionService() *InMemorySessionService {
	return &InMemorySessionService{data: make(map[string]*Session)}
}

func (s *InMemorySessionService) Get(_ context.Context, id string) (*Session, *errors.ServiceError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.data[id]
	if !ok {
		return nil, errors.NotFound("Session with id '%s' not found", id)
	}
	cp := *sess
	return &cp, nil
}

func (s *InMemorySessionService) Create(_ context.Context, session *Session) (*Session, *errors.ServiceError) {
	session.ID = api.NewID()
	now := time.Now()
	session.CreatedAt = now
	session.UpdatedAt = now
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *session
	s.data[session.ID] = &cp
	return &cp, nil
}

func (s *InMemorySessionService) Replace(_ context.Context, session *Session) (*Session, *errors.ServiceError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[session.ID]; !ok {
		return nil, errors.NotFound("Session with id '%s' not found", session.ID)
	}
	session.UpdatedAt = time.Now()
	cp := *session
	s.data[session.ID] = &cp
	return &cp, nil
}

func (s *InMemorySessionService) Delete(_ context.Context, id string) *errors.ServiceError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[id]; !ok {
		return errors.NotFound("Session with id '%s' not found", id)
	}
	delete(s.data, id)
	return nil
}

func (s *InMemorySessionService) All(_ context.Context) (SessionList, *errors.ServiceError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make(SessionList, 0, len(s.data))
	for _, sess := range s.data {
		cp := *sess
		list = append(list, &cp)
	}
	return list, nil
}

func (s *InMemorySessionService) AllByProjectId(_ context.Context, projectId string) (SessionList, *errors.ServiceError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list SessionList
	for _, sess := range s.data {
		if sess.ProjectId != nil && *sess.ProjectId == projectId {
			cp := *sess
			list = append(list, &cp)
		}
	}
	return list, nil
}

func (s *InMemorySessionService) FindByIDs(_ context.Context, ids []string) (SessionList, *errors.ServiceError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	var list SessionList
	for _, sess := range s.data {
		if idSet[sess.ID] {
			cp := *sess
			list = append(list, &cp)
		}
	}
	return list, nil
}

func (s *InMemorySessionService) UpdateStatus(_ context.Context, id string, patch *SessionStatusPatchRequest) (*Session, *errors.ServiceError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.data[id]
	if !ok {
		return nil, errors.NotFound("Session with id '%s' not found", id)
	}
	if patch.Phase != nil {
		sess.Phase = patch.Phase
	}
	if patch.StartTime != nil {
		sess.StartTime = patch.StartTime
	}
	if patch.CompletionTime != nil {
		sess.CompletionTime = patch.CompletionTime
	}
	if patch.SdkSessionId != nil {
		sess.SdkSessionId = patch.SdkSessionId
	}
	if patch.SdkRestartCount != nil {
		sess.SdkRestartCount = patch.SdkRestartCount
	}
	if patch.Conditions != nil {
		sess.Conditions = patch.Conditions
	}
	if patch.ReconciledRepos != nil {
		sess.ReconciledRepos = patch.ReconciledRepos
	}
	if patch.ReconciledWorkflow != nil {
		sess.ReconciledWorkflow = patch.ReconciledWorkflow
	}
	if patch.KubeCrUid != nil {
		sess.KubeCrUid = patch.KubeCrUid
	}
	if patch.KubeNamespace != nil {
		sess.KubeNamespace = patch.KubeNamespace
	}
	sess.UpdatedAt = time.Now()
	cp := *sess
	return &cp, nil
}

func (s *InMemorySessionService) Start(_ context.Context, id string) (*Session, *errors.ServiceError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.data[id]
	if !ok {
		return nil, errors.NotFound("Session with id '%s' not found", id)
	}
	phase := "Running"
	sess.Phase = &phase
	cp := *sess
	return &cp, nil
}

func (s *InMemorySessionService) Stop(_ context.Context, id string) (*Session, *errors.ServiceError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.data[id]
	if !ok {
		return nil, errors.NotFound("Session with id '%s' not found", id)
	}
	phase := "Stopped"
	sess.Phase = &phase
	cp := *sess
	return &cp, nil
}

func (s *InMemorySessionService) ActiveByAgentID(_ context.Context, agentID string) (*Session, *errors.ServiceError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	activePhases := map[string]bool{"Pending": true, "Creating": true, "Running": true}
	var newest *Session
	for _, sess := range s.data {
		if sess.AgentId != nil && *sess.AgentId == agentID &&
			sess.Phase != nil && activePhases[*sess.Phase] {
			if newest == nil || sess.CreatedAt.After(newest.CreatedAt) {
				newest = sess
			}
		}
	}
	if newest != nil {
		cp := *newest
		return &cp, nil
	}
	return nil, errors.NotFound("no active session for agent '%s'", agentID)
}

func (s *InMemorySessionService) OnUpsert(_ context.Context, _ string) error { return nil }
func (s *InMemorySessionService) OnDelete(_ context.Context, _ string) error { return nil }
