package sessions

import (
	"context"
	"sync"
	"time"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

type InMemorySessionService struct {
	mu   sync.RWMutex
	data map[string]*Session
}

var _ SessionService = &InMemorySessionService{}

func NewInMemorySessionService() *InMemorySessionService {
	return &InMemorySessionService{
		data: make(map[string]*Session),
	}
}

func (s *InMemorySessionService) Get(_ context.Context, id string) (*Session, *errors.ServiceError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ss, ok := s.data[id]
	if !ok {
		return nil, errors.NotFound("Session with id '%s' not found", id)
	}
	cp := *ss
	return &cp, nil
}

func (s *InMemorySessionService) Create(_ context.Context, session *Session) (*Session, *errors.ServiceError) {
	session.ID = api.NewID()
	now := time.Now()
	session.CreatedAt = now
	session.UpdatedAt = now
	session.KubeCrName = &session.ID
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
	var list SessionList
	for _, ss := range s.data {
		cp := *ss
		list = append(list, &cp)
	}
	return list, nil
}

func (s *InMemorySessionService) AllByProjectId(_ context.Context, projectId string) (SessionList, *errors.ServiceError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list SessionList
	for _, ss := range s.data {
		if ss.ProjectId != nil && *ss.ProjectId == projectId {
			cp := *ss
			list = append(list, &cp)
		}
	}
	return list, nil
}

func (s *InMemorySessionService) UpdateStatus(_ context.Context, id string, patch *SessionStatusPatchRequest) (*Session, *errors.ServiceError) {
	if patch == nil {
		return nil, errors.Validation("patch request must not be nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ss, ok := s.data[id]
	if !ok {
		return nil, errors.NotFound("Session with id '%s' not found", id)
	}
	if patch.Phase != nil {
		ss.Phase = patch.Phase
	}
	ss.UpdatedAt = time.Now()
	cp := *ss
	return &cp, nil
}

func (s *InMemorySessionService) Start(_ context.Context, id string) (*Session, *errors.ServiceError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ss, ok := s.data[id]
	if !ok {
		return nil, errors.NotFound("Session with id '%s' not found", id)
	}
	phase := "running"
	ss.Phase = &phase
	ss.UpdatedAt = time.Now()
	cp := *ss
	return &cp, nil
}

func (s *InMemorySessionService) Stop(_ context.Context, id string) (*Session, *errors.ServiceError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ss, ok := s.data[id]
	if !ok {
		return nil, errors.NotFound("Session with id '%s' not found", id)
	}
	phase := "stopped"
	ss.Phase = &phase
	ss.UpdatedAt = time.Now()
	cp := *ss
	return &cp, nil
}

func (s *InMemorySessionService) ActiveByAgentID(_ context.Context, agentID string) (*Session, *errors.ServiceError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ss := range s.data {
		if ss.AgentId != nil && *ss.AgentId == agentID && ss.Phase != nil && *ss.Phase == "running" {
			cp := *ss
			return &cp, nil
		}
	}
	return nil, errors.NotFound("no active session for agent '%s'", agentID)
}

func (s *InMemorySessionService) FindByIDs(_ context.Context, ids []string) (SessionList, *errors.ServiceError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list SessionList
	for _, id := range ids {
		if ss, ok := s.data[id]; ok {
			cp := *ss
			list = append(list, &cp)
		}
	}
	return list, nil
}

func (s *InMemorySessionService) OnUpsert(_ context.Context, _ string) error { return nil }
func (s *InMemorySessionService) OnDelete(_ context.Context, _ string) error { return nil }
