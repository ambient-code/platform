package repoIntelligences

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"

	"github.com/ambient-code/platform/components/ambient-api-server/plugins/repoEvents"
)

const repoIntelligencesLockType db.LockType = "repo_intelligences"

type RepoIntelligenceService interface {
	Get(ctx context.Context, id string) (*RepoIntelligence, *errors.ServiceError)
	Create(ctx context.Context, ri *RepoIntelligence) (*RepoIntelligence, *errors.ServiceError)
	Replace(ctx context.Context, ri *RepoIntelligence) (*RepoIntelligence, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (RepoIntelligenceList, *errors.ServiceError)
	FindByIDs(ctx context.Context, ids []string) (RepoIntelligenceList, *errors.ServiceError)
	GetByProjectAndRepo(ctx context.Context, projectID, repoURL string) (*RepoIntelligence, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewRepoIntelligenceService(lockFactory db.LockFactory, dao RepoIntelligenceDao, events services.EventService, auditSvc repoEvents.RepoEventService) RepoIntelligenceService {
	return &sqlRepoIntelligenceService{
		lockFactory: lockFactory,
		dao:         dao,
		events:      events,
		auditSvc:    auditSvc,
	}
}

var _ RepoIntelligenceService = &sqlRepoIntelligenceService{}

type sqlRepoIntelligenceService struct {
	lockFactory db.LockFactory
	dao         RepoIntelligenceDao
	events      services.EventService
	auditSvc    repoEvents.RepoEventService
}

func (s *sqlRepoIntelligenceService) logAuditEvent(ctx context.Context, ri *RepoIntelligence, action string) {
	if s.auditSvc == nil {
		return
	}
	actorType := "system"
	actorID := "api-server"
	if ri.AnalyzedBySessionID != nil {
		actorType = "session"
		actorID = *ri.AnalyzedBySessionID
	}
	_, _ = s.auditSvc.Create(ctx, &repoEvents.RepoEvent{
		ResourceType: "intelligence",
		ResourceID:   ri.ID,
		Action:       action,
		ActorType:    actorType,
		ActorID:      actorID,
		ProjectID:    ri.ProjectID,
	})
}

func (s *sqlRepoIntelligenceService) OnUpsert(ctx context.Context, id string) error {
	log := logger.NewLogger(ctx)
	log.Infof("RepoIntelligence upserted: %s", id)
	return nil
}

func (s *sqlRepoIntelligenceService) OnDelete(ctx context.Context, id string) error {
	log := logger.NewLogger(ctx)
	log.Infof("RepoIntelligence deleted: %s", id)
	return nil
}

func (s *sqlRepoIntelligenceService) Get(ctx context.Context, id string) (*RepoIntelligence, *errors.ServiceError) {
	ri, err := s.dao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("RepoIntelligence", "id", id, err)
	}
	return ri, nil
}

func (s *sqlRepoIntelligenceService) Create(ctx context.Context, ri *RepoIntelligence) (*RepoIntelligence, *errors.ServiceError) {
	ri, err := s.dao.Create(ctx, ri)
	if err != nil {
		return nil, services.HandleCreateError("RepoIntelligence", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "RepoIntelligences",
		SourceID:  ri.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("RepoIntelligence", evErr)
	}

	s.logAuditEvent(ctx, ri, "created")

	return ri, nil
}

func (s *sqlRepoIntelligenceService) Replace(ctx context.Context, ri *RepoIntelligence) (*RepoIntelligence, *errors.ServiceError) {
	lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, ri.ID, repoIntelligencesLockType)
	if err != nil {
		return nil, errors.DatabaseAdvisoryLock(err)
	}
	defer s.lockFactory.Unlock(ctx, lockOwnerID)

	ri, err = s.dao.Replace(ctx, ri)
	if err != nil {
		return nil, services.HandleUpdateError("RepoIntelligence", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "RepoIntelligences",
		SourceID:  ri.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("RepoIntelligence", evErr)
	}

	s.logAuditEvent(ctx, ri, "updated")

	return ri, nil
}

func (s *sqlRepoIntelligenceService) Delete(ctx context.Context, id string) *errors.ServiceError {
	// Fetch before delete to get project_id for audit
	ri, getErr := s.dao.Get(ctx, id)
	if getErr != nil {
		return services.HandleDeleteError("RepoIntelligence", errors.GeneralError("unable to delete repo intelligence: %s", getErr))
	}

	if err := s.dao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("RepoIntelligence", errors.GeneralError("unable to delete repo intelligence: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "RepoIntelligences",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("RepoIntelligence", evErr)
	}

	s.logAuditEvent(ctx, ri, "deleted")

	return nil
}

func (s *sqlRepoIntelligenceService) FindByIDs(ctx context.Context, ids []string) (RepoIntelligenceList, *errors.ServiceError) {
	items, err := s.dao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("unable to find repo intelligences: %s", err)
	}
	return items, nil
}

func (s *sqlRepoIntelligenceService) All(ctx context.Context) (RepoIntelligenceList, *errors.ServiceError) {
	items, err := s.dao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("unable to get all repo intelligences: %s", err)
	}
	return items, nil
}

func (s *sqlRepoIntelligenceService) GetByProjectAndRepo(ctx context.Context, projectID, repoURL string) (*RepoIntelligence, *errors.ServiceError) {
	ri, err := s.dao.GetByProjectAndRepo(ctx, projectID, repoURL)
	if err != nil {
		return nil, services.HandleGetError("RepoIntelligence", "project_id+repo_url", projectID+"/"+repoURL, err)
	}
	return ri, nil
}
