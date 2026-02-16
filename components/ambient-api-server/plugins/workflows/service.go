package workflows

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const workflowsLockType db.LockType = "workflows"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type WorkflowService interface {
	Get(ctx context.Context, id string) (*Workflow, *errors.ServiceError)
	Create(ctx context.Context, workflow *Workflow) (*Workflow, *errors.ServiceError)
	Replace(ctx context.Context, workflow *Workflow) (*Workflow, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (WorkflowList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (WorkflowList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewWorkflowService(lockFactory db.LockFactory, workflowDao WorkflowDao, events services.EventService) WorkflowService {
	return &sqlWorkflowService{
		lockFactory: lockFactory,
		workflowDao: workflowDao,
		events:      events,
	}
}

var _ WorkflowService = &sqlWorkflowService{}

type sqlWorkflowService struct {
	lockFactory db.LockFactory
	workflowDao WorkflowDao
	events      services.EventService
}

func (s *sqlWorkflowService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)

	workflow, err := s.workflowDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this workflow: %s", workflow.ID)

	return nil
}

func (s *sqlWorkflowService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)
	logger.Infof("This workflow has been deleted: %s", id)
	return nil
}

func (s *sqlWorkflowService) Get(ctx context.Context, id string) (*Workflow, *errors.ServiceError) {
	workflow, err := s.workflowDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("Workflow", "id", id, err)
	}
	return workflow, nil
}

func (s *sqlWorkflowService) Create(ctx context.Context, workflow *Workflow) (*Workflow, *errors.ServiceError) {
	workflow, err := s.workflowDao.Create(ctx, workflow)
	if err != nil {
		return nil, services.HandleCreateError("Workflow", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Workflows",
		SourceID:  workflow.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("Workflow", evErr)
	}

	return workflow, nil
}

func (s *sqlWorkflowService) Replace(ctx context.Context, workflow *Workflow) (*Workflow, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, workflow.ID, workflowsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, workflow.ID, workflowsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("Workflow", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	workflow, err := s.workflowDao.Replace(ctx, workflow)
	if err != nil {
		return nil, services.HandleUpdateError("Workflow", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Workflows",
		SourceID:  workflow.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("Workflow", evErr)
	}

	return workflow, nil
}

func (s *sqlWorkflowService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.workflowDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("Workflow", errors.GeneralError("Unable to delete workflow: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Workflows",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("Workflow", evErr)
	}

	return nil
}

func (s *sqlWorkflowService) FindByIDs(ctx context.Context, ids []string) (WorkflowList, *errors.ServiceError) {
	workflows, err := s.workflowDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all workflows: %s", err)
	}
	return workflows, nil
}

func (s *sqlWorkflowService) All(ctx context.Context) (WorkflowList, *errors.ServiceError) {
	workflows, err := s.workflowDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all workflows: %s", err)
	}
	return workflows, nil
}
