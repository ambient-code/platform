package tasks

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const tasksLockType db.LockType = "tasks"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type TaskService interface {
	Get(ctx context.Context, id string) (*Task, *errors.ServiceError)
	Create(ctx context.Context, task *Task) (*Task, *errors.ServiceError)
	Replace(ctx context.Context, task *Task) (*Task, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (TaskList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (TaskList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewTaskService(lockFactory db.LockFactory, taskDao TaskDao, events services.EventService) TaskService {
	return &sqlTaskService{
		lockFactory: lockFactory,
		taskDao:     taskDao,
		events:      events,
	}
}

var _ TaskService = &sqlTaskService{}

type sqlTaskService struct {
	lockFactory db.LockFactory
	taskDao     TaskDao
	events      services.EventService
}

func (s *sqlTaskService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)

	task, err := s.taskDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this task: %s", task.ID)

	return nil
}

func (s *sqlTaskService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)
	logger.Infof("This task has been deleted: %s", id)
	return nil
}

func (s *sqlTaskService) Get(ctx context.Context, id string) (*Task, *errors.ServiceError) {
	task, err := s.taskDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("Task", "id", id, err)
	}
	return task, nil
}

func (s *sqlTaskService) Create(ctx context.Context, task *Task) (*Task, *errors.ServiceError) {
	task, err := s.taskDao.Create(ctx, task)
	if err != nil {
		return nil, services.HandleCreateError("Task", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Tasks",
		SourceID:  task.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("Task", evErr)
	}

	return task, nil
}

func (s *sqlTaskService) Replace(ctx context.Context, task *Task) (*Task, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, task.ID, tasksLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, task.ID, tasksLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("Task", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	task, err := s.taskDao.Replace(ctx, task)
	if err != nil {
		return nil, services.HandleUpdateError("Task", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Tasks",
		SourceID:  task.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("Task", evErr)
	}

	return task, nil
}

func (s *sqlTaskService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.taskDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("Task", errors.GeneralError("Unable to delete task: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Tasks",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("Task", evErr)
	}

	return nil
}

func (s *sqlTaskService) FindByIDs(ctx context.Context, ids []string) (TaskList, *errors.ServiceError) {
	tasks, err := s.taskDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all tasks: %s", err)
	}
	return tasks, nil
}

func (s *sqlTaskService) All(ctx context.Context) (TaskList, *errors.ServiceError) {
	tasks, err := s.taskDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all tasks: %s", err)
	}
	return tasks, nil
}
