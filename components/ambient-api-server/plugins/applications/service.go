package applications

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const applicationsLockType db.LockType = "applications"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type ApplicationService interface {
	Get(ctx context.Context, id string) (*Application, *errors.ServiceError)
	Create(ctx context.Context, application *Application) (*Application, *errors.ServiceError)
	Replace(ctx context.Context, application *Application) (*Application, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (ApplicationList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (ApplicationList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewApplicationService(lockFactory db.LockFactory, applicationDao ApplicationDao, events services.EventService) ApplicationService {
	return &sqlApplicationService{
		lockFactory:    lockFactory,
		applicationDao: applicationDao,
		events:         events,
	}
}

var _ ApplicationService = &sqlApplicationService{}

type sqlApplicationService struct {
	lockFactory    db.LockFactory
	applicationDao ApplicationDao
	events         services.EventService
}

func (s *sqlApplicationService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)

	application, err := s.applicationDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this application: %s", application.ID)

	return nil
}

func (s *sqlApplicationService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)
	logger.Infof("This application has been deleted: %s", id)
	return nil
}

func (s *sqlApplicationService) Get(ctx context.Context, id string) (*Application, *errors.ServiceError) {
	application, err := s.applicationDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("Application", "id", id, err)
	}
	return application, nil
}

func (s *sqlApplicationService) Create(ctx context.Context, application *Application) (*Application, *errors.ServiceError) {
	application, err := s.applicationDao.Create(ctx, application)
	if err != nil {
		return nil, services.HandleCreateError("Application", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Applications",
		SourceID:  application.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("Application", evErr)
	}

	return application, nil
}

func (s *sqlApplicationService) Replace(ctx context.Context, application *Application) (*Application, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, application.ID, applicationsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, application.ID, applicationsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("Application", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	application, err := s.applicationDao.Replace(ctx, application)
	if err != nil {
		return nil, services.HandleUpdateError("Application", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Applications",
		SourceID:  application.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("Application", evErr)
	}

	return application, nil
}

func (s *sqlApplicationService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.applicationDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("Application", errors.GeneralError("Unable to delete application: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Applications",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("Application", evErr)
	}

	return nil
}

func (s *sqlApplicationService) FindByIDs(ctx context.Context, ids []string) (ApplicationList, *errors.ServiceError) {
	applications, err := s.applicationDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all applications: %s", err)
	}
	return applications, nil
}

func (s *sqlApplicationService) All(ctx context.Context) (ApplicationList, *errors.ServiceError) {
	applications, err := s.applicationDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all applications: %s", err)
	}
	return applications, nil
}
