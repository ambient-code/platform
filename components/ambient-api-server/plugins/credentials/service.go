package credentials

import (
	"context"
	"fmt"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/crypto"
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

func NewCredentialService(lockFactory db.LockFactory, credentialDao CredentialDao, events services.EventService, keyring *crypto.Keyring) CredentialService {
	return &sqlCredentialService{
		lockFactory:   lockFactory,
		credentialDao: credentialDao,
		events:        events,
		keyring:       keyring,
	}
}

var _ CredentialService = &sqlCredentialService{}

type sqlCredentialService struct {
	lockFactory   db.LockFactory
	credentialDao CredentialDao
	events        services.EventService
	keyring       *crypto.Keyring
}

func (s *sqlCredentialService) encryptToken(credential *Credential) *errors.ServiceError {
	if s.keyring == nil || credential.Token == nil || *credential.Token == "" {
		return nil
	}
	ciphertext, err := s.keyring.Encrypt(*credential.Token, credential.ID)
	if err != nil {
		return errors.GeneralError("encrypt credential token: %v", err)
	}
	credential.Token = &ciphertext
	return nil
}

func (s *sqlCredentialService) decryptToken(credential *Credential) *errors.ServiceError {
	if s.keyring == nil || credential.Token == nil || *credential.Token == "" {
		return nil
	}
	if !crypto.IsEncrypted(*credential.Token) {
		return nil
	}
	plaintext, err := s.keyring.Decrypt(*credential.Token, credential.ID)
	if err != nil {
		return errors.GeneralError("decrypt credential token: %v", err)
	}
	credential.Token = &plaintext
	return nil
}

func (s *sqlCredentialService) decryptList(credentials CredentialList) *errors.ServiceError {
	for _, c := range credentials {
		if err := s.decryptToken(c); err != nil {
			return err
		}
	}
	return nil
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
	if svcErr := s.decryptToken(credential); svcErr != nil {
		return nil, svcErr
	}
	return credential, nil
}

func (s *sqlCredentialService) Create(ctx context.Context, credential *Credential) (*Credential, *errors.ServiceError) {
	if credential.ID == "" {
		credential.ID = api.NewID()
	}
	if svcErr := s.encryptToken(credential); svcErr != nil {
		return nil, svcErr
	}
	credential, err := s.credentialDao.Create(ctx, credential)
	if err != nil {
		return nil, services.HandleCreateError("Credential", err)
	}

	if s.events != nil {
		_, evErr := s.events.Create(ctx, &api.Event{
			Source:    "Credentials",
			SourceID:  credential.ID,
			EventType: api.CreateEventType,
		})
		if evErr != nil {
			return nil, services.HandleCreateError("Credential", evErr)
		}
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

	if svcErr := s.encryptToken(credential); svcErr != nil {
		return nil, svcErr
	}
	credential, err := s.credentialDao.Replace(ctx, credential)
	if err != nil {
		return nil, services.HandleUpdateError("Credential", err)
	}

	if s.events != nil {
		_, evErr := s.events.Create(ctx, &api.Event{
			Source:    "Credentials",
			SourceID:  credential.ID,
			EventType: api.UpdateEventType,
		})
		if evErr != nil {
			return nil, services.HandleUpdateError("Credential", evErr)
		}
	}

	return credential, nil
}

func (s *sqlCredentialService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.credentialDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("Credential", errors.GeneralError("Unable to delete credential: %s", err))
	}

	if s.events != nil {
		if _, evErr := s.events.Create(ctx, &api.Event{
			Source:    "Credentials",
			SourceID:  id,
			EventType: api.DeleteEventType,
		}); evErr != nil {
			logger.NewLogger(ctx).Warning(fmt.Sprintf("Credential %s deleted but event creation failed: %v", id, evErr))
		}
	}

	return nil
}

func (s *sqlCredentialService) FindByIDs(ctx context.Context, ids []string) (CredentialList, *errors.ServiceError) {
	credentials, err := s.credentialDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all credentials: %s", err)
	}
	if svcErr := s.decryptList(credentials); svcErr != nil {
		return nil, svcErr
	}
	return credentials, nil
}

func (s *sqlCredentialService) All(ctx context.Context) (CredentialList, *errors.ServiceError) {
	credentials, err := s.credentialDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all credentials: %s", err)
	}
	if svcErr := s.decryptList(credentials); svcErr != nil {
		return nil, svcErr
	}
	return credentials, nil
}
