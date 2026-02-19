package agents

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const agentsLockType db.LockType = "agents"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type AgentService interface {
	Get(ctx context.Context, id string) (*Agent, *errors.ServiceError)
	Create(ctx context.Context, agent *Agent) (*Agent, *errors.ServiceError)
	Replace(ctx context.Context, agent *Agent) (*Agent, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (AgentList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (AgentList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewAgentService(lockFactory db.LockFactory, agentDao AgentDao, events services.EventService) AgentService {
	return &sqlAgentService{
		lockFactory: lockFactory,
		agentDao:    agentDao,
		events:      events,
	}
}

var _ AgentService = &sqlAgentService{}

type sqlAgentService struct {
	lockFactory db.LockFactory
	agentDao    AgentDao
	events      services.EventService
}

func (s *sqlAgentService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)

	agent, err := s.agentDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this agent: %s", agent.ID)

	return nil
}

func (s *sqlAgentService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)
	logger.Infof("This agent has been deleted: %s", id)
	return nil
}

func (s *sqlAgentService) Get(ctx context.Context, id string) (*Agent, *errors.ServiceError) {
	agent, err := s.agentDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("Agent", "id", id, err)
	}
	return agent, nil
}

func (s *sqlAgentService) Create(ctx context.Context, agent *Agent) (*Agent, *errors.ServiceError) {
	agent, err := s.agentDao.Create(ctx, agent)
	if err != nil {
		return nil, services.HandleCreateError("Agent", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Agents",
		SourceID:  agent.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("Agent", evErr)
	}

	return agent, nil
}

func (s *sqlAgentService) Replace(ctx context.Context, agent *Agent) (*Agent, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, agent.ID, agentsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, agent.ID, agentsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("Agent", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	agent, err := s.agentDao.Replace(ctx, agent)
	if err != nil {
		return nil, services.HandleUpdateError("Agent", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Agents",
		SourceID:  agent.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("Agent", evErr)
	}

	return agent, nil
}

func (s *sqlAgentService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.agentDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("Agent", errors.GeneralError("Unable to delete agent: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Agents",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("Agent", evErr)
	}

	return nil
}

func (s *sqlAgentService) FindByIDs(ctx context.Context, ids []string) (AgentList, *errors.ServiceError) {
	agents, err := s.agentDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all agents: %s", err)
	}
	return agents, nil
}

func (s *sqlAgentService) All(ctx context.Context) (AgentList, *errors.ServiceError) {
	agents, err := s.agentDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all agents: %s", err)
	}
	return agents, nil
}
