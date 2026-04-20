package inbox

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const inboxMessagesLockType db.LockType = "inbox_messages"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type InboxMessageService interface {
	Get(ctx context.Context, id string) (*InboxMessage, *errors.ServiceError)
	Create(ctx context.Context, inboxMessage *InboxMessage) (*InboxMessage, *errors.ServiceError)
	Replace(ctx context.Context, inboxMessage *InboxMessage) (*InboxMessage, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (InboxMessageList, *errors.ServiceError)
	UnreadByAgentID(ctx context.Context, agentID string) (InboxMessageList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (InboxMessageList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewInboxMessageService(lockFactory db.LockFactory, inboxMessageDao InboxMessageDao, events services.EventService, watchSvc InboxWatchService) InboxMessageService {
	return &sqlInboxMessageService{
		lockFactory:     lockFactory,
		inboxMessageDao: inboxMessageDao,
		events:          events,
		watchSvc:        watchSvc,
	}
}

var _ InboxMessageService = &sqlInboxMessageService{}

type sqlInboxMessageService struct {
	lockFactory     db.LockFactory
	inboxMessageDao InboxMessageDao
	events          services.EventService
	watchSvc        InboxWatchService
}

func (s *sqlInboxMessageService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)

	inboxMessage, err := s.inboxMessageDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this inboxMessage: %s", inboxMessage.ID)

	return nil
}

func (s *sqlInboxMessageService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)
	logger.Infof("This inboxMessage has been deleted: %s", id)
	return nil
}

func (s *sqlInboxMessageService) Get(ctx context.Context, id string) (*InboxMessage, *errors.ServiceError) {
	inboxMessage, err := s.inboxMessageDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("InboxMessage", "id", id, err)
	}
	return inboxMessage, nil
}

func (s *sqlInboxMessageService) Create(ctx context.Context, inboxMessage *InboxMessage) (*InboxMessage, *errors.ServiceError) {
	inboxMessage, err := s.inboxMessageDao.Create(ctx, inboxMessage)
	if err != nil {
		return nil, services.HandleCreateError("InboxMessage", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "InboxMessages",
		SourceID:  inboxMessage.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("InboxMessage", evErr)
	}

	if s.watchSvc != nil {
		s.watchSvc.Notify(inboxMessage)
	}

	return inboxMessage, nil
}

func (s *sqlInboxMessageService) Replace(ctx context.Context, inboxMessage *InboxMessage) (*InboxMessage, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, inboxMessage.ID, inboxMessagesLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, inboxMessage.ID, inboxMessagesLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("InboxMessage", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	inboxMessage, err := s.inboxMessageDao.Replace(ctx, inboxMessage)
	if err != nil {
		return nil, services.HandleUpdateError("InboxMessage", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "InboxMessages",
		SourceID:  inboxMessage.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("InboxMessage", evErr)
	}

	return inboxMessage, nil
}

func (s *sqlInboxMessageService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.inboxMessageDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("InboxMessage", errors.GeneralError("Unable to delete inboxMessage: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "InboxMessages",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("InboxMessage", evErr)
	}

	return nil
}

func (s *sqlInboxMessageService) FindByIDs(ctx context.Context, ids []string) (InboxMessageList, *errors.ServiceError) {
	inboxMessages, err := s.inboxMessageDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all inboxMessages: %s", err)
	}
	return inboxMessages, nil
}

func (s *sqlInboxMessageService) All(ctx context.Context) (InboxMessageList, *errors.ServiceError) {
	inboxMessages, err := s.inboxMessageDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all inboxMessages: %s", err)
	}
	return inboxMessages, nil
}

func (s *sqlInboxMessageService) UnreadByAgentID(ctx context.Context, agentID string) (InboxMessageList, *errors.ServiceError) {
	messages, err := s.inboxMessageDao.UnreadByAgentID(ctx, agentID)
	if err != nil {
		return nil, errors.GeneralError("Unable to get unread inbox messages for agent %s: %s", agentID, err)
	}
	return messages, nil
}
