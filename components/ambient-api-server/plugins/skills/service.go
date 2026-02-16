package skills

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const skillsLockType db.LockType = "skills"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type SkillService interface {
	Get(ctx context.Context, id string) (*Skill, *errors.ServiceError)
	Create(ctx context.Context, skill *Skill) (*Skill, *errors.ServiceError)
	Replace(ctx context.Context, skill *Skill) (*Skill, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (SkillList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (SkillList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewSkillService(lockFactory db.LockFactory, skillDao SkillDao, events services.EventService) SkillService {
	return &sqlSkillService{
		lockFactory: lockFactory,
		skillDao:    skillDao,
		events:      events,
	}
}

var _ SkillService = &sqlSkillService{}

type sqlSkillService struct {
	lockFactory db.LockFactory
	skillDao    SkillDao
	events      services.EventService
}

func (s *sqlSkillService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)

	skill, err := s.skillDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this skill: %s", skill.ID)

	return nil
}

func (s *sqlSkillService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewOCMLogger(ctx)
	logger.Infof("This skill has been deleted: %s", id)
	return nil
}

func (s *sqlSkillService) Get(ctx context.Context, id string) (*Skill, *errors.ServiceError) {
	skill, err := s.skillDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("Skill", "id", id, err)
	}
	return skill, nil
}

func (s *sqlSkillService) Create(ctx context.Context, skill *Skill) (*Skill, *errors.ServiceError) {
	skill, err := s.skillDao.Create(ctx, skill)
	if err != nil {
		return nil, services.HandleCreateError("Skill", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Skills",
		SourceID:  skill.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("Skill", evErr)
	}

	return skill, nil
}

func (s *sqlSkillService) Replace(ctx context.Context, skill *Skill) (*Skill, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, skill.ID, skillsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, skill.ID, skillsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("Skill", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	skill, err := s.skillDao.Replace(ctx, skill)
	if err != nil {
		return nil, services.HandleUpdateError("Skill", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Skills",
		SourceID:  skill.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("Skill", evErr)
	}

	return skill, nil
}

func (s *sqlSkillService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.skillDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("Skill", errors.GeneralError("Unable to delete skill: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Skills",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("Skill", evErr)
	}

	return nil
}

func (s *sqlSkillService) FindByIDs(ctx context.Context, ids []string) (SkillList, *errors.ServiceError) {
	skills, err := s.skillDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all skills: %s", err)
	}
	return skills, nil
}

func (s *sqlSkillService) All(ctx context.Context) (SkillList, *errors.ServiceError) {
	skills, err := s.skillDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all skills: %s", err)
	}
	return skills, nil
}
