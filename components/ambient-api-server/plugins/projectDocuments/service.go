package projectDocuments

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

const projectDocumentsLockType db.LockType = "project_documents"

var (
	DisableAdvisoryLock     = false
	UseBlockingAdvisoryLock = true
)

type ProjectDocumentService interface {
	Get(ctx context.Context, id string) (*ProjectDocument, *errors.ServiceError)
	Create(ctx context.Context, projectDocument *ProjectDocument) (*ProjectDocument, *errors.ServiceError)
	Replace(ctx context.Context, projectDocument *ProjectDocument) (*ProjectDocument, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (ProjectDocumentList, *errors.ServiceError)

	FindByIDs(ctx context.Context, ids []string) (ProjectDocumentList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewProjectDocumentService(lockFactory db.LockFactory, projectDocumentDao ProjectDocumentDao, events services.EventService) ProjectDocumentService {
	return &sqlProjectDocumentService{
		lockFactory:        lockFactory,
		projectDocumentDao: projectDocumentDao,
		events:             events,
	}
}

var _ ProjectDocumentService = &sqlProjectDocumentService{}

type sqlProjectDocumentService struct {
	lockFactory        db.LockFactory
	projectDocumentDao ProjectDocumentDao
	events             services.EventService
}

func (s *sqlProjectDocumentService) OnUpsert(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)

	projectDocument, err := s.projectDocumentDao.Get(ctx, id)
	if err != nil {
		return err
	}

	logger.Infof("Do idempotent somethings with this projectDocument: %s", projectDocument.ID)

	return nil
}

func (s *sqlProjectDocumentService) OnDelete(ctx context.Context, id string) error {
	logger := logger.NewLogger(ctx)
	logger.Infof("This projectDocument has been deleted: %s", id)
	return nil
}

func (s *sqlProjectDocumentService) Get(ctx context.Context, id string) (*ProjectDocument, *errors.ServiceError) {
	projectDocument, err := s.projectDocumentDao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("ProjectDocument", "id", id, err)
	}
	return projectDocument, nil
}

func (s *sqlProjectDocumentService) Create(ctx context.Context, projectDocument *ProjectDocument) (*ProjectDocument, *errors.ServiceError) {
	projectDocument, err := s.projectDocumentDao.Create(ctx, projectDocument)
	if err != nil {
		return nil, services.HandleCreateError("ProjectDocument", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "ProjectDocuments",
		SourceID:  projectDocument.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("ProjectDocument", evErr)
	}

	return projectDocument, nil
}

func (s *sqlProjectDocumentService) Replace(ctx context.Context, projectDocument *ProjectDocument) (*ProjectDocument, *errors.ServiceError) {
	if !DisableAdvisoryLock {
		if UseBlockingAdvisoryLock {
			lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, projectDocument.ID, projectDocumentsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		} else {
			lockOwnerID, locked, err := s.lockFactory.NewNonBlockingLock(ctx, projectDocument.ID, projectDocumentsLockType)
			if err != nil {
				return nil, errors.DatabaseAdvisoryLock(err)
			}
			if !locked {
				return nil, services.HandleCreateError("ProjectDocument", errors.New(errors.ErrorConflict, "row locked"))
			}
			defer s.lockFactory.Unlock(ctx, lockOwnerID)
		}
	}

	projectDocument, err := s.projectDocumentDao.Replace(ctx, projectDocument)
	if err != nil {
		return nil, services.HandleUpdateError("ProjectDocument", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "ProjectDocuments",
		SourceID:  projectDocument.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("ProjectDocument", evErr)
	}

	return projectDocument, nil
}

func (s *sqlProjectDocumentService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if err := s.projectDocumentDao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("ProjectDocument", errors.GeneralError("Unable to delete projectDocument: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "ProjectDocuments",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("ProjectDocument", evErr)
	}

	return nil
}

func (s *sqlProjectDocumentService) FindByIDs(ctx context.Context, ids []string) (ProjectDocumentList, *errors.ServiceError) {
	projectDocuments, err := s.projectDocumentDao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all projectDocuments: %s", err)
	}
	return projectDocuments, nil
}

func (s *sqlProjectDocumentService) All(ctx context.Context) (ProjectDocumentList, *errors.ServiceError) {
	projectDocuments, err := s.projectDocumentDao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("Unable to get all projectDocuments: %s", err)
	}
	return projectDocuments, nil
}
