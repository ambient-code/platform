package scheduledSessions

import (
	"context"
	"sync"
	"time"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

// InMemoryScheduledSessionService is a zero-dependency service for tests and local dev.
// It stores state in a map and never touches the database.
type InMemoryScheduledSessionService struct {
	mu   sync.RWMutex
	data map[string]*ScheduledSession
}

var _ ScheduledSessionService = &InMemoryScheduledSessionService{}

func NewInMemoryService() *InMemoryScheduledSessionService {
	return &InMemoryScheduledSessionService{
		data: make(map[string]*ScheduledSession),
	}
}

func (s *InMemoryScheduledSessionService) Get(_ context.Context, id string) (*ScheduledSession, *errors.ServiceError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ss, ok := s.data[id]
	if !ok {
		return nil, errors.NotFound("ScheduledSession with id '%s' not found", id)
	}
	cp := *ss
	return &cp, nil
}

func (s *InMemoryScheduledSessionService) Create(_ context.Context, ss *ScheduledSession) (*ScheduledSession, *errors.ServiceError) {
	ss.ID = api.NewID()
	now := time.Now()
	ss.CreatedAt = now
	ss.UpdatedAt = now
	if ss.Timezone == "" {
		ss.Timezone = "UTC"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *ss
	s.data[ss.ID] = &cp
	return &cp, nil
}

func (s *InMemoryScheduledSessionService) Patch(_ context.Context, id string, patch *ScheduledSessionPatch) (*ScheduledSession, *errors.ServiceError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ss, ok := s.data[id]
	if !ok {
		return nil, errors.NotFound("ScheduledSession with id '%s' not found", id)
	}
	if patch.Name != nil {
		ss.Name = *patch.Name
	}
	if patch.Description != nil {
		ss.Description = patch.Description
	}
	if patch.Schedule != nil {
		ss.Schedule = *patch.Schedule
	}
	if patch.Timezone != nil {
		ss.Timezone = *patch.Timezone
	}
	if patch.Enabled != nil {
		ss.Enabled = *patch.Enabled
	}
	if patch.SessionPrompt != nil {
		ss.SessionPrompt = patch.SessionPrompt
	}
	ss.UpdatedAt = time.Now()
	cp := *ss
	return &cp, nil
}

func (s *InMemoryScheduledSessionService) Delete(_ context.Context, id string) *errors.ServiceError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[id]; !ok {
		return errors.NotFound("ScheduledSession with id '%s' not found", id)
	}
	delete(s.data, id)
	return nil
}

func (s *InMemoryScheduledSessionService) ListByProject(_ context.Context, projectId string) (ScheduledSessionList, *errors.ServiceError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list ScheduledSessionList
	for _, ss := range s.data {
		if ss.ProjectId == projectId {
			cp := *ss
			list = append(list, &cp)
		}
	}
	return list, nil
}

func (s *InMemoryScheduledSessionService) Suspend(ctx context.Context, id string) (*ScheduledSession, *errors.ServiceError) {
	disabled := false
	return s.Patch(ctx, id, &ScheduledSessionPatch{Enabled: &disabled})
}

func (s *InMemoryScheduledSessionService) Resume(ctx context.Context, id string) (*ScheduledSession, *errors.ServiceError) {
	enabled := true
	return s.Patch(ctx, id, &ScheduledSessionPatch{Enabled: &enabled})
}

func (s *InMemoryScheduledSessionService) Trigger(ctx context.Context, id string) *errors.ServiceError {
	_, err := s.Get(ctx, id)
	return err
}
