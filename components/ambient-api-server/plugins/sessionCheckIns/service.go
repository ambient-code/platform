package sessionCheckIns

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const sessionCheckInsLockType db.LockType = "session_check_ins"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type SessionCheckInService interface {
	Get(ctx context.Context, id string) (*SessionCheckIn, *errors.ServiceError)
	Create(ctx context.Context, sessionCheckIn *SessionCheckIn) (*SessionCheckIn, *errors.ServiceError)
	Replace(ctx context.Context, sessionCheckIn *SessionCheckIn) (*SessionCheckIn, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (SessionCheckInList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (SessionCheckInList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewSessionCheckInService(lockFactory db.LockFactory, sessionCheckInDao SessionCheckInDao, events services.EventService) SessionCheckInService {
	return &sqlSessionCheckInService{
		lockFactory:       lockFactory,
		sessionCheckInDao: sessionCheckInDao,
		events:            events,
	}
}

var _ SessionCheckInService = &sqlSessionCheckInService{}

type sqlSessionCheckInService struct {
	lockFactory       db.LockFactory
	sessionCheckInDao SessionCheckInDao
	events            services.EventService
}

func (s *sqlSessionCheckInService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)

	sessionCheckIn, err := s.sessionCheckInDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this sessionCheckIn: %s", sessionCheckIn.ID)

	return nil
}

func (s *sqlSessionCheckInService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)
	logger.Infof("This sessionCheckIn has been deleted: %s", id)
	return nil
}

func (s *sqlSessionCheckInService) Get(ctx context.Context, id string) (*SessionCheckIn, *errors.ServiceError) {
	sessionCheckIn, err := s.sessionCheckInDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("SessionCheckIn", "id", id, err)
	}
	return sessionCheckIn, nil
}

func (s *sqlSessionCheckInService) Create(ctx context.Context, sessionCheckIn *SessionCheckIn) (*SessionCheckIn, *errors.ServiceError) {
	sessionCheckIn, err := s.sessionCheckInDao.Create(ctx, sessionCheckIn)
	if err != nil {
		return nil, services.HandleCreateError("SessionCheckIn", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "SessionCheckIns",
		SourceID:  sessionCheckIn.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("SessionCheckIn", evErr)
	}

	return sessionCheckIn, nil
}

func (s *sqlSessionCheckInService) Replace(ctx context.Context, sessionCheckIn *SessionCheckIn) (*SessionCheckIn, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, sessionCheckIn.ID, sessionCheckInsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, sessionCheckIn.ID, sessionCheckInsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("SessionCheckIn", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	sessionCheckIn, err := s.sessionCheckInDao.Replace(ctx, sessionCheckIn)
	if err != nil {
		return nil, services.HandleUpdateError("SessionCheckIn", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "SessionCheckIns",
		SourceID:  sessionCheckIn.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("SessionCheckIn", evErr)
	}

	return sessionCheckIn, nil
}

func (s *sqlSessionCheckInService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.sessionCheckInDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("SessionCheckIn", errors.GeneralError("Unable to delete sessionCheckIn: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "SessionCheckIns",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("SessionCheckIn", evErr)
	}

	return nil
}

func (s *sqlSessionCheckInService) FindByIDs(ctx context.Context, ids []string) (SessionCheckInList, *errors.ServiceError) {
	sessionCheckIns, err := s.sessionCheckInDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all sessionCheckIns: %s", err)
	}
	return sessionCheckIns, nil
}

func (s *sqlSessionCheckInService) All(ctx context.Context) (SessionCheckInList, *errors.ServiceError) {
	sessionCheckIns, err := s.sessionCheckInDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all sessionCheckIns: %s", err)
	}
	return sessionCheckIns, nil
}
