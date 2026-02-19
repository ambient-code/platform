package workflowSkills

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const workflowSkillsLockType db.LockType = "workflow_skills"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type WorkflowSkillService interface {
	Get(ctx context.Context, id string) (*WorkflowSkill, *errors.ServiceError)
	Create(ctx context.Context, workflowSkill *WorkflowSkill) (*WorkflowSkill, *errors.ServiceError)
	Replace(ctx context.Context, workflowSkill *WorkflowSkill) (*WorkflowSkill, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (WorkflowSkillList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (WorkflowSkillList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewWorkflowSkillService(lockFactory db.LockFactory, workflowSkillDao WorkflowSkillDao, events services.EventService) WorkflowSkillService {
	return &sqlWorkflowSkillService{
		lockFactory:      lockFactory,
		workflowSkillDao: workflowSkillDao,
		events:           events,
	}
}

var _ WorkflowSkillService = &sqlWorkflowSkillService{}

type sqlWorkflowSkillService struct {
	lockFactory      db.LockFactory
	workflowSkillDao WorkflowSkillDao
	events           services.EventService
}

func (s *sqlWorkflowSkillService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)

	workflowSkill, err := s.workflowSkillDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this workflowSkill: %s", workflowSkill.ID)

	return nil
}

func (s *sqlWorkflowSkillService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)
	logger.Infof("This workflowSkill has been deleted: %s", id)
	return nil
}

func (s *sqlWorkflowSkillService) Get(ctx context.Context, id string) (*WorkflowSkill, *errors.ServiceError) {
	workflowSkill, err := s.workflowSkillDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("WorkflowSkill", "id", id, err)
	}
	return workflowSkill, nil
}

func (s *sqlWorkflowSkillService) Create(ctx context.Context, workflowSkill *WorkflowSkill) (*WorkflowSkill, *errors.ServiceError) {
	workflowSkill, err := s.workflowSkillDao.Create(ctx, workflowSkill)
	if err != nil {
		return nil, services.HandleCreateError("WorkflowSkill", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "WorkflowSkills",
		SourceID:  workflowSkill.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("WorkflowSkill", evErr)
	}

	return workflowSkill, nil
}

func (s *sqlWorkflowSkillService) Replace(ctx context.Context, workflowSkill *WorkflowSkill) (*WorkflowSkill, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, workflowSkill.ID, workflowSkillsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, workflowSkill.ID, workflowSkillsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("WorkflowSkill", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	workflowSkill, err := s.workflowSkillDao.Replace(ctx, workflowSkill)
	if err != nil {
		return nil, services.HandleUpdateError("WorkflowSkill", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "WorkflowSkills",
		SourceID:  workflowSkill.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("WorkflowSkill", evErr)
	}

	return workflowSkill, nil
}

func (s *sqlWorkflowSkillService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.workflowSkillDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("WorkflowSkill", errors.GeneralError("Unable to delete workflowSkill: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "WorkflowSkills",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("WorkflowSkill", evErr)
	}

	return nil
}

func (s *sqlWorkflowSkillService) FindByIDs(ctx context.Context, ids []string) (WorkflowSkillList, *errors.ServiceError) {
	workflowSkills, err := s.workflowSkillDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all workflowSkills: %s", err)
	}
	return workflowSkills, nil
}

func (s *sqlWorkflowSkillService) All(ctx context.Context) (WorkflowSkillList, *errors.ServiceError) {
	workflowSkills, err := s.workflowSkillDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all workflowSkills: %s", err)
	}
	return workflowSkills, nil
}
