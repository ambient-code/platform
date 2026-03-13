package agentMessages

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const agentMessagesLockType db.LockType = "agent_messages"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type AgentMessageService interface {
	Get(ctx context.Context, id string) (*AgentMessage, *errors.ServiceError)
	Create(ctx context.Context, agentMessage *AgentMessage) (*AgentMessage, *errors.ServiceError)
	Replace(ctx context.Context, agentMessage *AgentMessage) (*AgentMessage, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (AgentMessageList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (AgentMessageList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewAgentMessageService(lockFactory db.LockFactory, agentMessageDao AgentMessageDao, events services.EventService) AgentMessageService {
	return &sqlAgentMessageService{
		lockFactory:     lockFactory,
		agentMessageDao: agentMessageDao,
		events:          events,
	}
}

var _ AgentMessageService = &sqlAgentMessageService{}

type sqlAgentMessageService struct {
	lockFactory     db.LockFactory
	agentMessageDao AgentMessageDao
	events          services.EventService
}

func (s *sqlAgentMessageService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)

	agentMessage, err := s.agentMessageDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this agentMessage: %s", agentMessage.ID)

	return nil
}

func (s *sqlAgentMessageService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)
	logger.Infof("This agentMessage has been deleted: %s", id)
	return nil
}

func (s *sqlAgentMessageService) Get(ctx context.Context, id string) (*AgentMessage, *errors.ServiceError) {
	agentMessage, err := s.agentMessageDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("AgentMessage", "id", id, err)
	}
	return agentMessage, nil
}

func (s *sqlAgentMessageService) Create(ctx context.Context, agentMessage *AgentMessage) (*AgentMessage, *errors.ServiceError) {
	agentMessage, err := s.agentMessageDao.Create(ctx, agentMessage)
	if err != nil {
		return nil, services.HandleCreateError("AgentMessage", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "AgentMessages",
		SourceID:  agentMessage.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("AgentMessage", evErr)
	}

	return agentMessage, nil
}

func (s *sqlAgentMessageService) Replace(ctx context.Context, agentMessage *AgentMessage) (*AgentMessage, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, agentMessage.ID, agentMessagesLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, agentMessage.ID, agentMessagesLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("AgentMessage", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	agentMessage, err := s.agentMessageDao.Replace(ctx, agentMessage)
	if err != nil {
		return nil, services.HandleUpdateError("AgentMessage", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "AgentMessages",
		SourceID:  agentMessage.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("AgentMessage", evErr)
	}

	return agentMessage, nil
}

func (s *sqlAgentMessageService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.agentMessageDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("AgentMessage", errors.GeneralError("Unable to delete agentMessage: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "AgentMessages",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("AgentMessage", evErr)
	}

	return nil
}

func (s *sqlAgentMessageService) FindByIDs(ctx context.Context, ids []string) (AgentMessageList, *errors.ServiceError) {
	agentMessages, err := s.agentMessageDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all agentMessages: %s", err)
	}
	return agentMessages, nil
}

func (s *sqlAgentMessageService) All(ctx context.Context) (AgentMessageList, *errors.ServiceError) {
	agentMessages, err := s.agentMessageDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all agentMessages: %s", err)
	}
	return agentMessages, nil
}
