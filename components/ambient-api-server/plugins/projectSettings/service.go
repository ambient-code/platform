package projectSettings

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const projectSettingsLockType db.LockType = "project_settings"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type ProjectSettingsService interface {
	Get(ctx context.Context, id string) (*ProjectSettings, *errors.ServiceError)
	Create(ctx context.Context, ps *ProjectSettings) (*ProjectSettings, *errors.ServiceError)
	Replace(ctx context.Context, ps *ProjectSettings) (*ProjectSettings, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (ProjectSettingsList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (ProjectSettingsList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewProjectSettingsService(lockFactory db.LockFactory, dao ProjectSettingsDao, events services.EventService) ProjectSettingsService {
	return &sqlProjectSettingsService{
		lockFactory: lockFactory,
		dao:         dao,
		events:      events,
	}
}

var _ ProjectSettingsService = &sqlProjectSettingsService{}

type sqlProjectSettingsService struct {
	lockFactory db.LockFactory
	dao         ProjectSettingsDao
	events      services.EventService
}

func (s *sqlProjectSettingsService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)

	ps, err := s.dao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this project settings: %s", ps.ID)

	return nil
}

func (s *sqlProjectSettingsService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)
	logger.Infof("This project settings has been deleted: %s", id)
	return nil
}

func (s *sqlProjectSettingsService) Get(ctx context.Context, id string) (*ProjectSettings, *errors.ServiceError) {
	ps, err := s.dao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("ProjectSettings", "id", id, err)
	}
	return ps, nil
}

func (s *sqlProjectSettingsService) Create(ctx context.Context, ps *ProjectSettings) (*ProjectSettings, *errors.ServiceError) {
	ps, err := s.dao.Create(ctx, ps)
	if err != nil {
		return nil, services.HandleCreateError("ProjectSettings", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "ProjectSettings",
		SourceID:  ps.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("ProjectSettings", evErr)
	}

	return ps, nil
}

func (s *sqlProjectSettingsService) Replace(ctx context.Context, ps *ProjectSettings) (*ProjectSettings, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, ps.ID, projectSettingsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, ps.ID, projectSettingsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("ProjectSettings", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	ps, err := s.dao.Replace(ctx, ps)
	if err != nil {
		return nil, services.HandleUpdateError("ProjectSettings", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "ProjectSettings",
		SourceID:  ps.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("ProjectSettings", evErr)
	}

	return ps, nil
}

func (s *sqlProjectSettingsService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.dao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("ProjectSettings", errors.GeneralError("Unable to delete project settings: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "ProjectSettings",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("ProjectSettings", evErr)
	}

	return nil
}

func (s *sqlProjectSettingsService) FindByIDs(ctx context.Context, ids []string) (ProjectSettingsList, *errors.ServiceError) {
	list, err := s.dao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all project settings: %s", err)
	}
	return list, nil
}

func (s *sqlProjectSettingsService) All(ctx context.Context) (ProjectSettingsList, *errors.ServiceError) {
	list, err := s.dao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all project settings: %s", err)
	}
	return list, nil
}
