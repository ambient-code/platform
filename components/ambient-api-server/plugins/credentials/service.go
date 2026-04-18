package credentials

import (
	"context"
	"fmt"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const credentialsLockType db.LockType = "credentials"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type CredentialService interface {
	Get(ctx context.Context, id string) (*Credential, *errors.ServiceError)
	Create(ctx context.Context, credential *Credential) (*Credential, *errors.ServiceError)
	Replace(ctx context.Context, credential *Credential) (*Credential, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (CredentialList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (CredentialList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewCredentialService(lockFactory db.LockFactory, credentialDao CredentialDao, events services.EventService) CredentialService {
	return &sqlCredentialService{
		lockFactory:   lockFactory,
		credentialDao: credentialDao,
		events:        events,
	}
}

var _ CredentialService = &sqlCredentialService{}

type sqlCredentialService struct {
	lockFactory   db.LockFactory
	credentialDao CredentialDao
	events        services.EventService
}

func (s *sqlCredentialService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)

	credential, err := s.credentialDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this credential: %s", credential.ID)

	return nil
}

func (s *sqlCredentialService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)
	logger.Infof("This credential has been deleted: %s", id)
	return nil
}

func (s *sqlCredentialService) Get(ctx context.Context, id string) (*Credential, *errors.ServiceError) {
	credential, err := s.credentialDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("Credential", "id", id, err)
	}
	return credential, nil
}

func (s *sqlCredentialService) Create(ctx context.Context, credential *Credential) (*Credential, *errors.ServiceError) {
	credential, err := s.credentialDao.Create(ctx, credential)
	if err != nil {
		return nil, services.HandleCreateError("Credential", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Credentials",
		SourceID:  credential.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("Credential", evErr)
	}

	return credential, nil
}

func (s *sqlCredentialService) Replace(ctx context.Context, credential *Credential) (*Credential, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, credential.ID, credentialsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, credential.ID, credentialsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("Credential", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	credential, err := s.credentialDao.Replace(ctx, credential)
	if err != nil {
		return nil, services.HandleUpdateError("Credential", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Credentials",
		SourceID:  credential.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("Credential", evErr)
	}

	return credential, nil
}

func (s *sqlCredentialService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.credentialDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("Credential", errors.GeneralError("Unable to delete credential: %s", err))
	}

	if _, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Credentials",
		SourceID:  id,
		EventType: api.DeleteEventType,
	}); evErr != nil {
		logger.NewLogger(ctx).Warning(fmt.Sprintf("Credential %s deleted but event creation failed: %v", id, evErr))
	}

	return nil
}

func (s *sqlCredentialService) FindByIDs(ctx context.Context, ids []string) (CredentialList, *errors.ServiceError) {
	credentials, err := s.credentialDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all credentials: %s", err)
	}
	return credentials, nil
}

func (s *sqlCredentialService) All(ctx context.Context) (CredentialList, *errors.ServiceError) {
	credentials, err := s.credentialDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all credentials: %s", err)
	}
	return credentials, nil
}
