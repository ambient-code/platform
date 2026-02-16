package projectKeys

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const projectKeysLockType db.LockType = "project_keys"

type ProjectKeyService interface {
	Get(ctx context.Context, id string) (*ProjectKey, *errors.ServiceError)
	Create(ctx context.Context, projectKey *ProjectKey) (*ProjectKey, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (ProjectKeyList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (ProjectKeyList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewProjectKeyService(lockFactory db.LockFactory, projectKeyDao ProjectKeyDao, events services.EventService) ProjectKeyService {
	return &sqlProjectKeyService{
		lockFactory:   lockFactory,
		projectKeyDao: projectKeyDao,
		events:        events,
	}
}

var _ ProjectKeyService = &sqlProjectKeyService{}

type sqlProjectKeyService struct {
	lockFactory   db.LockFactory
	projectKeyDao ProjectKeyDao
	events        services.EventService
}

func (s *sqlProjectKeyService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)

	projectKey, err := s.projectKeyDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Project key created: %s (prefix: %s)", projectKey.ID, projectKey.KeyPrefix)

	return nil
}

func (s *sqlProjectKeyService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)
	logger.Infof("Project key revoked: %s", id)
	return nil
}

func (s *sqlProjectKeyService) Get(ctx context.Context, id string) (*ProjectKey, *errors.ServiceError) {
	projectKey, err := s.projectKeyDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("ProjectKey", "id", id, err)
	}
	return projectKey, nil
}

func (s *sqlProjectKeyService) Create(ctx context.Context, projectKey *ProjectKey) (*ProjectKey, *errors.ServiceError) {
	projectKey, err := s.projectKeyDao.Create(ctx, projectKey)
	if err != nil {
		return nil, services.HandleCreateError("ProjectKey", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "ProjectKeys",
		SourceID:  projectKey.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("ProjectKey", evErr)
	}

	return projectKey, nil
}

func (s *sqlProjectKeyService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.projectKeyDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("ProjectKey", errors.GeneralError("Unable to delete projectKey: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "ProjectKeys",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("ProjectKey", evErr)
	}

	return nil
}

func (s *sqlProjectKeyService) FindByIDs(ctx context.Context, ids []string) (ProjectKeyList, *errors.ServiceError) {
	projectKeys, err := s.projectKeyDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all projectKeys: %s", err)
	}
	return projectKeys, nil
}

func (s *sqlProjectKeyService) All(ctx context.Context) (ProjectKeyList, *errors.ServiceError) {
	projectKeys, err := s.projectKeyDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all projectKeys: %s", err)
	}
	return projectKeys, nil
}
