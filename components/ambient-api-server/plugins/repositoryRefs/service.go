package repositoryRefs

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const repositoryRefsLockType db.LockType = "repository_refs"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type RepositoryRefService interface {
	Get(ctx context.Context, id string) (*RepositoryRef, *errors.ServiceError)
	Create(ctx context.Context, repositoryRef *RepositoryRef) (*RepositoryRef, *errors.ServiceError)
	Replace(ctx context.Context, repositoryRef *RepositoryRef) (*RepositoryRef, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (RepositoryRefList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (RepositoryRefList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewRepositoryRefService(lockFactory db.LockFactory, repositoryRefDao RepositoryRefDao, events services.EventService) RepositoryRefService {
	return &sqlRepositoryRefService{
		lockFactory:      lockFactory,
		repositoryRefDao: repositoryRefDao,
		events:           events,
	}
}

var _ RepositoryRefService = &sqlRepositoryRefService{}

type sqlRepositoryRefService struct {
	lockFactory      db.LockFactory
	repositoryRefDao RepositoryRefDao
	events           services.EventService
}

func (s *sqlRepositoryRefService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)

	repositoryRef, err := s.repositoryRefDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this repositoryRef: %s", repositoryRef.ID)

	return nil
}

func (s *sqlRepositoryRefService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)
	logger.Infof("This repositoryRef has been deleted: %s", id)
	return nil
}

func (s *sqlRepositoryRefService) Get(ctx context.Context, id string) (*RepositoryRef, *errors.ServiceError) {
	repositoryRef, err := s.repositoryRefDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("RepositoryRef", "id", id, err)
	}
	return repositoryRef, nil
}

func (s *sqlRepositoryRefService) Create(ctx context.Context, repositoryRef *RepositoryRef) (*RepositoryRef, *errors.ServiceError) {
	repositoryRef, err := s.repositoryRefDao.Create(ctx, repositoryRef)
	if err != nil {
		return nil, services.HandleCreateError("RepositoryRef", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "RepositoryRefs",
		SourceID:  repositoryRef.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("RepositoryRef", evErr)
	}

	return repositoryRef, nil
}

func (s *sqlRepositoryRefService) Replace(ctx context.Context, repositoryRef *RepositoryRef) (*RepositoryRef, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, repositoryRef.ID, repositoryRefsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, repositoryRef.ID, repositoryRefsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("RepositoryRef", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	repositoryRef, err := s.repositoryRefDao.Replace(ctx, repositoryRef)
	if err != nil {
		return nil, services.HandleUpdateError("RepositoryRef", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "RepositoryRefs",
		SourceID:  repositoryRef.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("RepositoryRef", evErr)
	}

	return repositoryRef, nil
}

func (s *sqlRepositoryRefService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.repositoryRefDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("RepositoryRef", errors.GeneralError("Unable to delete repositoryRef: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "RepositoryRefs",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("RepositoryRef", evErr)
	}

	return nil
}

func (s *sqlRepositoryRefService) FindByIDs(ctx context.Context, ids []string) (RepositoryRefList, *errors.ServiceError) {
	repositoryRefs, err := s.repositoryRefDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all repositoryRefs: %s", err)
	}
	return repositoryRefs, nil
}

func (s *sqlRepositoryRefService) All(ctx context.Context) (RepositoryRefList, *errors.ServiceError) {
	repositoryRefs, err := s.repositoryRefDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all repositoryRefs: %s", err)
	}
	return repositoryRefs, nil
}
