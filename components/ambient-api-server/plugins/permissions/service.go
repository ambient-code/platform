package permissions

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const permissionsLockType db.LockType = "permissions"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type PermissionService interface {
	Get(ctx context.Context, id string) (*Permission, *errors.ServiceError)
	Create(ctx context.Context, permission *Permission) (*Permission, *errors.ServiceError)
	Replace(ctx context.Context, permission *Permission) (*Permission, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (PermissionList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (PermissionList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewPermissionService(lockFactory db.LockFactory, permissionDao PermissionDao, events services.EventService) PermissionService {
	return &sqlPermissionService{
		lockFactory:   lockFactory,
		permissionDao: permissionDao,
		events:        events,
	}
}

var _ PermissionService = &sqlPermissionService{}

type sqlPermissionService struct {
	lockFactory   db.LockFactory
	permissionDao PermissionDao
	events        services.EventService
}

func (s *sqlPermissionService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)

	permission, err := s.permissionDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this permission: %s", permission.ID)

	return nil
}

func (s *sqlPermissionService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)
	logger.Infof("This permission has been deleted: %s", id)
	return nil
}

func (s *sqlPermissionService) Get(ctx context.Context, id string) (*Permission, *errors.ServiceError) {
	permission, err := s.permissionDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("Permission", "id", id, err)
	}
	return permission, nil
}

func (s *sqlPermissionService) Create(ctx context.Context, permission *Permission) (*Permission, *errors.ServiceError) {
	permission, err := s.permissionDao.Create(ctx, permission)
	if err != nil {
		return nil, services.HandleCreateError("Permission", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Permissions",
		SourceID:  permission.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("Permission", evErr)
	}

	return permission, nil
}

func (s *sqlPermissionService) Replace(ctx context.Context, permission *Permission) (*Permission, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, permission.ID, permissionsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, permission.ID, permissionsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("Permission", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	permission, err := s.permissionDao.Replace(ctx, permission)
	if err != nil {
		return nil, services.HandleUpdateError("Permission", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Permissions",
		SourceID:  permission.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("Permission", evErr)
	}

	return permission, nil
}

func (s *sqlPermissionService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.permissionDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("Permission", errors.GeneralError("Unable to delete permission: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Permissions",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("Permission", evErr)
	}

	return nil
}

func (s *sqlPermissionService) FindByIDs(ctx context.Context, ids []string) (PermissionList, *errors.ServiceError) {
	permissions, err := s.permissionDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all permissions: %s", err)
	}
	return permissions, nil
}

func (s *sqlPermissionService) All(ctx context.Context) (PermissionList, *errors.ServiceError) {
	permissions, err := s.permissionDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all permissions: %s", err)
	}
	return permissions, nil
}
