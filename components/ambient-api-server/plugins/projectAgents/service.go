package projectAgents

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const projectAgentsLockType db.LockType = "project_agents"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type ProjectAgentService interface {
	Get(ctx context.Context, id string) (*ProjectAgent, *errors.ServiceError)
	Create(ctx context.Context, projectAgent *ProjectAgent) (*ProjectAgent, *errors.ServiceError)
	Replace(ctx context.Context, projectAgent *ProjectAgent) (*ProjectAgent, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (ProjectAgentList, *errors.ServiceError)
	AllByProjectID(ctx context.Context, projectID string) (ProjectAgentList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (ProjectAgentList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewProjectAgentService(lockFactory db.LockFactory, projectAgentDao ProjectAgentDao, events services.EventService) ProjectAgentService {
	return &sqlProjectAgentService{
		lockFactory:     lockFactory,
		projectAgentDao: projectAgentDao,
		events:          events,
	}
}

var _ ProjectAgentService = &sqlProjectAgentService{}

type sqlProjectAgentService struct {
	lockFactory     db.LockFactory
	projectAgentDao ProjectAgentDao
	events          services.EventService
}

func (s *sqlProjectAgentService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)

	projectAgent, err := s.projectAgentDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this projectAgent: %s", projectAgent.ID)

	return nil
}

func (s *sqlProjectAgentService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)
	logger.Infof("This projectAgent has been deleted: %s", id)
	return nil
}

func (s *sqlProjectAgentService) Get(ctx context.Context, id string) (*ProjectAgent, *errors.ServiceError) {
	projectAgent, err := s.projectAgentDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("ProjectAgent", "id", id, err)
	}
	return projectAgent, nil
}

func (s *sqlProjectAgentService) Create(ctx context.Context, projectAgent *ProjectAgent) (*ProjectAgent, *errors.ServiceError) {
	projectAgent, err := s.projectAgentDao.Create(ctx, projectAgent)
	if err != nil {
		return nil, services.HandleCreateError("ProjectAgent", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "ProjectAgents",
		SourceID:  projectAgent.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("ProjectAgent", evErr)
	}

	return projectAgent, nil
}

func (s *sqlProjectAgentService) Replace(ctx context.Context, projectAgent *ProjectAgent) (*ProjectAgent, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, projectAgent.ID, projectAgentsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, projectAgent.ID, projectAgentsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("ProjectAgent", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	projectAgent, err := s.projectAgentDao.Replace(ctx, projectAgent)
	if err != nil {
		return nil, services.HandleUpdateError("ProjectAgent", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "ProjectAgents",
		SourceID:  projectAgent.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("ProjectAgent", evErr)
	}

	return projectAgent, nil
}

func (s *sqlProjectAgentService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.projectAgentDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("ProjectAgent", errors.GeneralError("Unable to delete projectAgent: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "ProjectAgents",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("ProjectAgent", evErr)
	}

	return nil
}

func (s *sqlProjectAgentService) FindByIDs(ctx context.Context, ids []string) (ProjectAgentList, *errors.ServiceError) {
	projectAgents, err := s.projectAgentDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all projectAgents: %s", err)
	}
	return projectAgents, nil
}

func (s *sqlProjectAgentService) All(ctx context.Context) (ProjectAgentList, *errors.ServiceError) {
	projectAgents, err := s.projectAgentDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all projectAgents: %s", err)
	}
	return projectAgents, nil
}

func (s *sqlProjectAgentService) AllByProjectID(ctx context.Context, projectID string) (ProjectAgentList, *errors.ServiceError) {
	projectAgents, err := s.projectAgentDao.AllByProjectID(ctx, projectID)
	if err != nil {
		return nil, errors.GeneralError("Unable to get projectAgents for project %s: %s", projectID, err)
	}
	return projectAgents, nil
}
