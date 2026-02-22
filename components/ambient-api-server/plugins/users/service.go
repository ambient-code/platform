package users

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const usersLockType db.LockType = "users"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type UserService interface {
	Get(ctx context.Context, id string) (*User, *errors.ServiceError)
	Create(ctx context.Context, user *User) (*User, *errors.ServiceError)
	Replace(ctx context.Context, user *User) (*User, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (UserList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (UserList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewUserService(lockFactory db.LockFactory, userDao UserDao, events services.EventService) UserService {
	return &sqlUserService{
		lockFactory: lockFactory,
		userDao:     userDao,
		events:      events,
	}
}

var _ UserService = &sqlUserService{}

type sqlUserService struct {
	lockFactory db.LockFactory
	userDao     UserDao
	events      services.EventService
}

func (s *sqlUserService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)

	user, err := s.userDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this user: %s", user.ID)

	return nil
}

func (s *sqlUserService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)
	logger.Infof("This user has been deleted: %s", id)
	return nil
}

func (s *sqlUserService) Get(ctx context.Context, id string) (*User, *errors.ServiceError) {
	user, err := s.userDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("User", "id", id, err)
	}
	return user, nil
}

func (s *sqlUserService) Create(ctx context.Context, user *User) (*User, *errors.ServiceError) {
	user, err := s.userDao.Create(ctx, user)
	if err != nil {
		return nil, services.HandleCreateError("User", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Users",
		SourceID:  user.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("User", evErr)
	}

	return user, nil
}

func (s *sqlUserService) Replace(ctx context.Context, user *User) (*User, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, user.ID, usersLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, user.ID, usersLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("User", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	user, err := s.userDao.Replace(ctx, user)
	if err != nil {
		return nil, services.HandleUpdateError("User", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Users",
		SourceID:  user.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("User", evErr)
	}

	return user, nil
}

func (s *sqlUserService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.userDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("User", errors.GeneralError("Unable to delete user: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Users",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("User", evErr)
	}

	return nil
}

func (s *sqlUserService) FindByIDs(ctx context.Context, ids []string) (UserList, *errors.ServiceError) {
	users, err := s.userDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all users: %s", err)
	}
	return users, nil
}

func (s *sqlUserService) All(ctx context.Context) (UserList, *errors.ServiceError) {
	users, err := s.userDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all users: %s", err)
	}
	return users, nil
}
