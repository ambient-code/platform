package sessions

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const sessionsLockType db.LockType = "sessions"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type SessionService interface {
	Get(ctx context.Context, id string) (*Session, *errors.ServiceError)
	Create(ctx context.Context, session *Session) (*Session, *errors.ServiceError)
	Replace(ctx context.Context, session *Session) (*Session, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (SessionList, *errors.ServiceError)
	UpdateStatus(ctx context.Context, id string, patch *SessionStatusPatchRequest) (*Session, *errors.ServiceError)
	Start(ctx context.Context, id string) (*Session, *errors.ServiceError)
	Stop(ctx context.Context, id string) (*Session, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (SessionList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewSessionService(lockFactory db.LockFactory, sessionDao SessionDao, events services.EventService) SessionService {
	return &sqlSessionService{
		lockFactory: lockFactory,
		sessionDao:  sessionDao,
		events:      events,
	}
}

var _ SessionService = &sqlSessionService{}

type sqlSessionService struct {
	lockFactory db.LockFactory
	sessionDao  SessionDao
	events      services.EventService
}

func (s *sqlSessionService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)

	session, err := s.sessionDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this session: %s", session.ID)

	return nil
}

func (s *sqlSessionService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)
	logger.Infof("This session has been deleted: %s", id)
	return nil
}

func (s *sqlSessionService) Get(ctx context.Context, id string) (*Session, *errors.ServiceError) {
	session, err := s.sessionDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("Session", "id", id, err)
	}
	return session, nil
}

func (s *sqlSessionService) Create(ctx context.Context, session *Session) (*Session, *errors.ServiceError) {
	session, err := s.sessionDao.Create(ctx, session)
	if err != nil {
		return nil, services.HandleCreateError("Session", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Sessions",
		SourceID:  session.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("Session", evErr)
	}

	return session, nil
}

func (s *sqlSessionService) Replace(ctx context.Context, session *Session) (*Session, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, session.ID, sessionsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, session.ID, sessionsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("Session", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	session, err := s.sessionDao.Replace(ctx, session)
	if err != nil {
		return nil, services.HandleUpdateError("Session", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Sessions",
		SourceID:  session.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("Session", evErr)
	}

	return session, nil
}

func (s *sqlSessionService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.sessionDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("Session", errors.GeneralError("Unable to delete session: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Sessions",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("Session", evErr)
	}

	return nil
}

func (s *sqlSessionService) FindByIDs(ctx context.Context, ids []string) (SessionList, *errors.ServiceError) {
	sessions, err := s.sessionDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all sessions: %s", err)
	}
	return sessions, nil
}

func (s *sqlSessionService) All(ctx context.Context) (SessionList, *errors.ServiceError) {
	sessions, err := s.sessionDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all sessions: %s", err)
	}
	return sessions, nil
}

func (s *sqlSessionService) UpdateStatus(ctx context.Context, id string, patch *SessionStatusPatchRequest) (*Session, *errors.ServiceError) {
	session, err := s.sessionDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("Session", "id", id, err)
	}

	if patch.Phase != nil {
		session.Phase = patch.Phase
	}
	if patch.StartTime != nil {
		session.StartTime = patch.StartTime
	}
	if patch.CompletionTime != nil {
		session.CompletionTime = patch.CompletionTime
	}
	if patch.SdkSessionId != nil {
		session.SdkSessionId = patch.SdkSessionId
	}
	if patch.SdkRestartCount != nil {
		session.SdkRestartCount = patch.SdkRestartCount
	}
	if patch.Conditions != nil {
		session.Conditions = patch.Conditions
	}
	if patch.ReconciledRepos != nil {
		session.ReconciledRepos = patch.ReconciledRepos
	}
	if patch.ReconciledWorkflow != nil {
		session.ReconciledWorkflow = patch.ReconciledWorkflow
	}
	if patch.KubeCrUid != nil {
		session.KubeCrUid = patch.KubeCrUid
	}
	if patch.KubeNamespace != nil {
		session.KubeNamespace = patch.KubeNamespace
	}

	session, err = s.sessionDao.Replace(ctx, session)
	if err != nil {
		return nil, services.HandleUpdateError("Session", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Sessions",
		SourceID:  session.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("Session", evErr)
	}

	return session, nil
}

func (s *sqlSessionService) Start(ctx context.Context, id string) (*Session, *errors.ServiceError) {
	session, err := s.sessionDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("Session", "id", id, err)
	}

	currentPhase := ""
	if session.Phase != nil {
		currentPhase = *session.Phase
	}

	if currentPhase != "" && currentPhase != "Stopped" && currentPhase != "Failed" && currentPhase != "Completed" {
		return nil, errors.Conflict("cannot start session in phase %q; must be empty, Stopped, Failed, or Completed", currentPhase)
	}

	pending := "Pending"
	session.Phase = &pending
	interactive := true
	session.Interactive = &interactive

	session, err = s.sessionDao.Replace(ctx, session)
	if err != nil {
		return nil, services.HandleUpdateError("Session", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Sessions",
		SourceID:  session.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("Session", evErr)
	}

	return session, nil
}

func (s *sqlSessionService) Stop(ctx context.Context, id string) (*Session, *errors.ServiceError) {
	session, err := s.sessionDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("Session", "id", id, err)
	}

	currentPhase := ""
	if session.Phase != nil {
		currentPhase = *session.Phase
	}

	if currentPhase != "Running" && currentPhase != "Creating" && currentPhase != "Pending" {
		return nil, errors.Conflict("cannot stop session in phase %q; must be Running, Creating, or Pending", currentPhase)
	}

	stopping := "Stopping"
	session.Phase = &stopping

	session, err = s.sessionDao.Replace(ctx, session)
	if err != nil {
		return nil, services.HandleUpdateError("Session", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Sessions",
		SourceID:  session.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("Session", evErr)
	}

	return session, nil
}
