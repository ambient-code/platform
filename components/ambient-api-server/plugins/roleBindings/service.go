package roleBindings

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const roleBindingsLockType db.LockType = "role_bindings"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type RoleBindingService interface {
	Get(ctx context.Context, id string) (*RoleBinding, *errors.ServiceError)
	Create(ctx context.Context, roleBinding *RoleBinding) (*RoleBinding, *errors.ServiceError)
	Replace(ctx context.Context, roleBinding *RoleBinding) (*RoleBinding, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (RoleBindingList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (RoleBindingList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewRoleBindingService(lockFactory db.LockFactory, roleBindingDao RoleBindingDao, events services.EventService) RoleBindingService {
	return &sqlRoleBindingService{
		lockFactory:    lockFactory,
		roleBindingDao: roleBindingDao,
		events:         events,
	}
}

var _ RoleBindingService = &sqlRoleBindingService{}

type sqlRoleBindingService struct {
	lockFactory    db.LockFactory
	roleBindingDao RoleBindingDao
	events         services.EventService
}

func (s *sqlRoleBindingService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)

	roleBinding, err := s.roleBindingDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this roleBinding: %s", roleBinding.ID)

	return nil
}

func (s *sqlRoleBindingService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)
	logger.Infof("This roleBinding has been deleted: %s", id)
	return nil
}

func (s *sqlRoleBindingService) Get(ctx context.Context, id string) (*RoleBinding, *errors.ServiceError) {
	roleBinding, err := s.roleBindingDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("RoleBinding", "id", id, err)
	}
	return roleBinding, nil
}

func (s *sqlRoleBindingService) Create(ctx context.Context, roleBinding *RoleBinding) (*RoleBinding, *errors.ServiceError) {
	roleBinding, err := s.roleBindingDao.Create(ctx, roleBinding)
	if err != nil {
		return nil, services.HandleCreateError("RoleBinding", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "RoleBindings",
		SourceID:  roleBinding.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("RoleBinding", evErr)
	}

	return roleBinding, nil
}

func (s *sqlRoleBindingService) Replace(ctx context.Context, roleBinding *RoleBinding) (*RoleBinding, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, roleBinding.ID, roleBindingsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, roleBinding.ID, roleBindingsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("RoleBinding", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	roleBinding, err := s.roleBindingDao.Replace(ctx, roleBinding)
	if err != nil {
		return nil, services.HandleUpdateError("RoleBinding", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "RoleBindings",
		SourceID:  roleBinding.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("RoleBinding", evErr)
	}

	return roleBinding, nil
}

func (s *sqlRoleBindingService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.roleBindingDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("RoleBinding", errors.GeneralError("Unable to delete roleBinding: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "RoleBindings",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("RoleBinding", evErr)
	}

	return nil
}

func (s *sqlRoleBindingService) FindByIDs(ctx context.Context, ids []string) (RoleBindingList, *errors.ServiceError) {
	roleBindings, err := s.roleBindingDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all roleBindings: %s", err)
	}
	return roleBindings, nil
}

func (s *sqlRoleBindingService) All(ctx context.Context) (RoleBindingList, *errors.ServiceError) {
	roleBindings, err := s.roleBindingDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all roleBindings: %s", err)
	}
	return roleBindings, nil
}
