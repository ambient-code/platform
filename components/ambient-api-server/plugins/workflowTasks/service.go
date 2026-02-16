package workflowTasks

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const workflowTasksLockType db.LockType = "workflow_tasks"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type WorkflowTaskService interface {
	Get(ctx context.Context, id string) (*WorkflowTask, *errors.ServiceError)
	Create(ctx context.Context, workflowTask *WorkflowTask) (*WorkflowTask, *errors.ServiceError)
	Replace(ctx context.Context, workflowTask *WorkflowTask) (*WorkflowTask, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (WorkflowTaskList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (WorkflowTaskList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewWorkflowTaskService(lockFactory db.LockFactory, workflowTaskDao WorkflowTaskDao, events services.EventService) WorkflowTaskService {
	return &sqlWorkflowTaskService{
		lockFactory:     lockFactory,
		workflowTaskDao: workflowTaskDao,
		events:          events,
	}
}

var _ WorkflowTaskService = &sqlWorkflowTaskService{}

type sqlWorkflowTaskService struct {
	lockFactory     db.LockFactory
	workflowTaskDao WorkflowTaskDao
	events          services.EventService
}

func (s *sqlWorkflowTaskService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)

	workflowTask, err := s.workflowTaskDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this workflowTask: %s", workflowTask.ID)

	return nil
}

func (s *sqlWorkflowTaskService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)
	logger.Infof("This workflowTask has been deleted: %s", id)
	return nil
}

func (s *sqlWorkflowTaskService) Get(ctx context.Context, id string) (*WorkflowTask, *errors.ServiceError) {
	workflowTask, err := s.workflowTaskDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("WorkflowTask", "id", id, err)
	}
	return workflowTask, nil
}

func (s *sqlWorkflowTaskService) Create(ctx context.Context, workflowTask *WorkflowTask) (*WorkflowTask, *errors.ServiceError) {
	workflowTask, err := s.workflowTaskDao.Create(ctx, workflowTask)
	if err != nil {
		return nil, services.HandleCreateError("WorkflowTask", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "WorkflowTasks",
		SourceID:  workflowTask.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("WorkflowTask", evErr)
	}

	return workflowTask, nil
}

func (s *sqlWorkflowTaskService) Replace(ctx context.Context, workflowTask *WorkflowTask) (*WorkflowTask, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, workflowTask.ID, workflowTasksLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, workflowTask.ID, workflowTasksLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("WorkflowTask", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	workflowTask, err := s.workflowTaskDao.Replace(ctx, workflowTask)
	if err != nil {
		return nil, services.HandleUpdateError("WorkflowTask", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "WorkflowTasks",
		SourceID:  workflowTask.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("WorkflowTask", evErr)
	}

	return workflowTask, nil
}

func (s *sqlWorkflowTaskService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.workflowTaskDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("WorkflowTask", errors.GeneralError("Unable to delete workflowTask: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "WorkflowTasks",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("WorkflowTask", evErr)
	}

	return nil
}

func (s *sqlWorkflowTaskService) FindByIDs(ctx context.Context, ids []string) (WorkflowTaskList, *errors.ServiceError) {
	workflowTasks, err := s.workflowTaskDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all workflowTasks: %s", err)
	}
	return workflowTasks, nil
}

func (s *sqlWorkflowTaskService) All(ctx context.Context) (WorkflowTaskList, *errors.ServiceError) {
	workflowTasks, err := s.workflowTaskDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all workflowTasks: %s", err)
	}
	return workflowTasks, nil
}
