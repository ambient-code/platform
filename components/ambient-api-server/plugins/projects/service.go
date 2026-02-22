package projects

import (
	"context"
	"regexp"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const projectsLockType db.LockType = "projects"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
	projectNameRegex        = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)
)

const projectNameMaxLength = 63

func ValidateProjectName(name string) *errors.ServiceError {
	if len(name) < 2 || len(name) > projectNameMaxLength {
		return errors.Validation("project name must be between 2 and %d characters", projectNameMaxLength)
	}
	if !projectNameRegex.MatchString(name) {
		return errors.Validation("project name must match DNS-1123 label format: lowercase alphanumeric and hyphens, must start with a letter and end with an alphanumeric character")
	}
	return nil
}

type ProjectService interface {
	Get(ctx context.Context, id string) (*Project, *errors.ServiceError)
	Create(ctx context.Context, project *Project) (*Project, *errors.ServiceError)
	Replace(ctx context.Context, project *Project) (*Project, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (ProjectList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (ProjectList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewProjectService(lockFactory db.LockFactory, projectDao ProjectDao, events services.EventService) ProjectService {
	return &sqlProjectService{
		lockFactory: lockFactory,
		projectDao:  projectDao,
		events:      events,
	}
}

var _ ProjectService = &sqlProjectService{}

type sqlProjectService struct {
	lockFactory db.LockFactory
	projectDao  ProjectDao
	events      services.EventService
}

func (s *sqlProjectService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)

	project, err := s.projectDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this project: %s", project.ID)

	return nil
}

func (s *sqlProjectService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)
	logger.Infof("This project has been deleted: %s", id)
	return nil
}

func (s *sqlProjectService) Get(ctx context.Context, id string) (*Project, *errors.ServiceError) {
	project, err := s.projectDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("Project", "id", id, err)
	}
	return project, nil
}

func (s *sqlProjectService) Create(ctx context.Context, project *Project) (*Project, *errors.ServiceError) {
	if svcErr := ValidateProjectName(project.Name); svcErr != nil {
		return nil, svcErr
	}

	project, err := s.projectDao.Create(ctx, project)
	if err != nil {
		return nil, services.HandleCreateError("Project", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Projects",
		SourceID:  project.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("Project", evErr)
	}

	return project, nil
}

func (s *sqlProjectService) Replace(ctx context.Context, project *Project) (*Project, *errors.ServiceError) {
	if svcErr := ValidateProjectName(project.Name); svcErr != nil {
		return nil, svcErr
	}

	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, project.ID, projectsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, project.ID, projectsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("Project", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	project, err := s.projectDao.Replace(ctx, project)
	if err != nil {
		return nil, services.HandleUpdateError("Project", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Projects",
		SourceID:  project.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("Project", evErr)
	}

	return project, nil
}

func (s *sqlProjectService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.projectDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("Project", errors.GeneralError("Unable to delete project: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "Projects",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("Project", evErr)
	}

	return nil
}

func (s *sqlProjectService) FindByIDs(ctx context.Context, ids []string) (ProjectList, *errors.ServiceError) {
	projects, err := s.projectDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all projects: %s", err)
	}
	return projects, nil
}

func (s *sqlProjectService) All(ctx context.Context) (ProjectList, *errors.ServiceError) {
	projects, err := s.projectDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all projects: %s", err)
	}
	return projects, nil
}
