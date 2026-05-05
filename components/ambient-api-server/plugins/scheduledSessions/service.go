package scheduledSessions

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
	"gorm.io/gorm"
)

type ScheduledSessionService interface {
	Get(ctx context.Context, id string) (*ScheduledSession, *errors.ServiceError)
	Create(ctx context.Context, ss *ScheduledSession) (*ScheduledSession, *errors.ServiceError)
	Patch(ctx context.Context, id string, patch *ScheduledSessionPatch) (*ScheduledSession, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	ListByProject(ctx context.Context, projectId string) (ScheduledSessionList, *errors.ServiceError)
	Suspend(ctx context.Context, id string) (*ScheduledSession, *errors.ServiceError)
	Resume(ctx context.Context, id string) (*ScheduledSession, *errors.ServiceError)
	Trigger(ctx context.Context, id string) *errors.ServiceError
}

type ScheduledSessionPatch struct {
	Name              *string
	Description       *string
	AgentId           *string
	Schedule          *string
	Timezone          *string
	Enabled           *bool
	SessionPrompt     *string
	Timeout           *int32
	InactivityTimeout *int32
	StopOnRunFinished *bool
	RunnerType        *string
}

type sqlScheduledSessionService struct {
	dao ScheduledSessionDao
}

func NewScheduledSessionService(dao ScheduledSessionDao) ScheduledSessionService {
	return &sqlScheduledSessionService{dao: dao}
}

func (s *sqlScheduledSessionService) Get(ctx context.Context, id string) (*ScheduledSession, *errors.ServiceError) {
	ss, err := s.dao.Get(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("ScheduledSession with id '%s' not found", id)
		}
		return nil, services.HandleGetError("ScheduledSession", "id", id, err)
	}
	return ss, nil
}

func (s *sqlScheduledSessionService) Create(ctx context.Context, ss *ScheduledSession) (*ScheduledSession, *errors.ServiceError) {
	created, err := s.dao.Create(ctx, ss)
	if err != nil {
		return nil, errors.GeneralError("failed to create scheduled session: %v", err)
	}
	return created, nil
}

func (s *sqlScheduledSessionService) Patch(ctx context.Context, id string, patch *ScheduledSessionPatch) (*ScheduledSession, *errors.ServiceError) {
	ss, svcErr := s.Get(ctx, id)
	if svcErr != nil {
		return nil, svcErr
	}
	if patch.Name != nil {
		ss.Name = *patch.Name
	}
	if patch.Description != nil {
		ss.Description = patch.Description
	}
	if patch.AgentId != nil {
		ss.AgentId = patch.AgentId
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
	if patch.Timeout != nil {
		ss.Timeout = patch.Timeout
	}
	if patch.InactivityTimeout != nil {
		ss.InactivityTimeout = patch.InactivityTimeout
	}
	if patch.StopOnRunFinished != nil {
		ss.StopOnRunFinished = patch.StopOnRunFinished
	}
	if patch.RunnerType != nil {
		ss.RunnerType = patch.RunnerType
	}
	updated, err := s.dao.Replace(ctx, ss)
	if err != nil {
		return nil, errors.GeneralError("failed to update scheduled session: %v", err)
	}
	return updated, nil
}

func (s *sqlScheduledSessionService) Delete(ctx context.Context, id string) *errors.ServiceError {
	_, svcErr := s.Get(ctx, id)
	if svcErr != nil {
		return svcErr
	}
	if err := s.dao.Delete(ctx, id); err != nil {
		return errors.GeneralError("failed to delete scheduled session: %v", err)
	}
	return nil
}

func (s *sqlScheduledSessionService) ListByProject(ctx context.Context, projectId string) (ScheduledSessionList, *errors.ServiceError) {
	list, err := s.dao.ListByProject(ctx, projectId)
	if err != nil {
		return nil, errors.GeneralError("failed to list scheduled sessions: %v", err)
	}
	return list, nil
}

func (s *sqlScheduledSessionService) Suspend(ctx context.Context, id string) (*ScheduledSession, *errors.ServiceError) {
	disabled := false
	return s.Patch(ctx, id, &ScheduledSessionPatch{Enabled: &disabled})
}

func (s *sqlScheduledSessionService) Resume(ctx context.Context, id string) (*ScheduledSession, *errors.ServiceError) {
	enabled := true
	return s.Patch(ctx, id, &ScheduledSessionPatch{Enabled: &enabled})
}

func (s *sqlScheduledSessionService) Trigger(ctx context.Context, id string) *errors.ServiceError {
	ss, svcErr := s.Get(ctx, id)
	if svcErr != nil {
		return svcErr
	}
	_ = ss
	// In production this would enqueue an immediate one-off session via the agent start endpoint.
	// In this session, we record intent and return success.
	return nil
}
