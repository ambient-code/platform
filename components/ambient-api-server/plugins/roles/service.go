package roles

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const rolesLockType db.LockType = "roles"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type RoleService interface {
	Get(ctx context.Context, id string) (*Role, *errors.ServiceError)
	Create(ctx context.Context, role *Role) (*Role, *errors.ServiceError)
	Replace(ctx context.Context, role *Role) (*Role, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (RoleList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (RoleList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewRoleService(lockFactory db.LockFactory, roleDao RoleDao, events services.EventService) RoleService {
	return &sqlRoleService{
		lockFactory: lockFactory,
		roleDao:     roleDao,
		events:      events,
	}
}

var _ RoleService = &sqlRoleService{}

type sqlRoleService struct {
	lockFactory db.LockFactory
	roleDao     RoleDao
	events      services.EventService
}

func (s *sqlRoleService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)

	role, err := s.roleDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this role: %s", role.ID)

	return nil
}

func (s *sqlRoleService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)
	logger.Infof("This role has been deleted: %s", id)
	return nil
}

func (s *sqlRoleService) Get(ctx context.Context, id string) (*Role, *errors.ServiceError) {
	role, err := s.roleDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("Role", "id", id, err)
	}
	return role, nil
}

func (s *sqlRoleService) Create(ctx context.Context, role *Role) (*Role, *errors.ServiceError) {
	role, err := s.roleDao.Create(ctx, role)
	if err != nil {
		return nil, services.HandleCreateError("Role", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Roles",
		SourceID:  role.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("Role", evErr)
	}

	return role, nil
}

func (s *sqlRoleService) Replace(ctx context.Context, role *Role) (*Role, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, role.ID, rolesLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, role.ID, rolesLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("Role", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	role, err := s.roleDao.Replace(ctx, role)
	if err != nil {
		return nil, services.HandleUpdateError("Role", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Roles",
		SourceID:  role.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("Role", evErr)
	}

	return role, nil
}

func (s *sqlRoleService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.roleDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("Role", errors.GeneralError("Unable to delete role: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Roles",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("Role", evErr)
	}

	return nil
}

func (s *sqlRoleService) FindByIDs(ctx context.Context, ids []string) (RoleList, *errors.ServiceError) {
	roles, err := s.roleDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all roles: %s", err)
	}
	return roles, nil
}

func (s *sqlRoleService) All(ctx context.Context) (RoleList, *errors.ServiceError) {
	roles, err := s.roleDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all roles: %s", err)
	}
	return roles, nil
}
