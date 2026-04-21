package repoFindings

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"

	"github.com/ambient-code/platform/components/ambient-api-server/plugins/repoEvents"
)

const repoFindingsLockType db.LockType = "repo_findings"

type RepoFindingService interface {
	Get(ctx context.Context, id string) (*RepoFinding, *errors.ServiceError)
	Create(ctx context.Context, rf *RepoFinding) (*RepoFinding, *errors.ServiceError)
	Replace(ctx context.Context, rf *RepoFinding) (*RepoFinding, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (RepoFindingList, *errors.ServiceError)
	FindByIDs(ctx context.Context, ids []string) (RepoFindingList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
	OnDelete(ctx context.Context, id string) error
}

func NewRepoFindingService(lockFactory db.LockFactory, dao RepoFindingDao, events services.EventService, auditSvc repoEvents.RepoEventService) RepoFindingService {
	return &sqlRepoFindingService{
		lockFactory: lockFactory,
		dao:         dao,
		events:      events,
		auditSvc:    auditSvc,
	}
}

var _ RepoFindingService = &sqlRepoFindingService{}

type sqlRepoFindingService struct {
	lockFactory db.LockFactory
	dao         RepoFindingDao
	events      services.EventService
	auditSvc    repoEvents.RepoEventService
}

func (s *sqlRepoFindingService) logAuditEvent(ctx context.Context, rf *RepoFinding, action string) {
	if s.auditSvc == nil {
		return
	}
	actorType := "system"
	actorID := "api-server"
	if rf.SessionID != nil {
		actorType = "session"
		actorID = *rf.SessionID
	}
	projectID, lookupErr := s.dao.LookupProjectID(ctx, rf.IntelligenceID)
	if lookupErr != nil {
		projectID = rf.IntelligenceID // fallback to intelligence_id if lookup fails
	}
	_, _ = s.auditSvc.Create(ctx, &repoEvents.RepoEvent{
		ResourceType: "finding",
		ResourceID:   rf.ID,
		Action:       action,
		ActorType:    actorType,
		ActorID:      actorID,
		ProjectID:    projectID,
	})
}

func (s *sqlRepoFindingService) OnUpsert(ctx context.Context, id string) error {
	log := logger.NewLogger(ctx)
	log.Infof("RepoFinding upserted: %s", id)
	return nil
}

func (s *sqlRepoFindingService) OnDelete(ctx context.Context, id string) error {
	log := logger.NewLogger(ctx)
	log.Infof("RepoFinding deleted: %s", id)
	return nil
}

func (s *sqlRepoFindingService) Get(ctx context.Context, id string) (*RepoFinding, *errors.ServiceError) {
	rf, err := s.dao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("RepoFinding", "id", id, err)
	}
	return rf, nil
}

func (s *sqlRepoFindingService) Create(ctx context.Context, rf *RepoFinding) (*RepoFinding, *errors.ServiceError) {
	rf, err := s.dao.Create(ctx, rf)
	if err != nil {
		return nil, services.HandleCreateError("RepoFinding", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "RepoFindings",
		SourceID:  rf.ID,
		EventType: api.CreateEventType,
	})
	if evErr != nil {
		return nil, services.HandleCreateError("RepoFinding", evErr)
	}

	s.logAuditEvent(ctx, rf, "created")

	return rf, nil
}

func (s *sqlRepoFindingService) Replace(ctx context.Context, rf *RepoFinding) (*RepoFinding, *errors.ServiceError) {
	lockOwnerID, err := s.lockFactory.NewAdvisoryLock(ctx, rf.ID, repoFindingsLockType)
	if err != nil {
		return nil, errors.DatabaseAdvisoryLock(err)
	}
	defer s.lockFactory.Unlock(ctx, lockOwnerID)

	rf, err = s.dao.Replace(ctx, rf)
	if err != nil {
		return nil, services.HandleUpdateError("RepoFinding", err)
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "RepoFindings",
		SourceID:  rf.ID,
		EventType: api.UpdateEventType,
	})
	if evErr != nil {
		return nil, services.HandleUpdateError("RepoFinding", evErr)
	}

	s.logAuditEvent(ctx, rf, "updated")

	return rf, nil
}

func (s *sqlRepoFindingService) Delete(ctx context.Context, id string) *errors.ServiceError {
	rf, getErr := s.dao.Get(ctx, id)
	if getErr != nil {
		return services.HandleDeleteError("RepoFinding", errors.GeneralError("unable to delete repo finding: %s", getErr))
	}

	if err := s.dao.Delete(ctx, id); err != nil {
		return services.HandleDeleteError("RepoFinding", errors.GeneralError("unable to delete repo finding: %s", err))
	}

	_, evErr := s.events.Create(ctx, &api.Event{
		Source:    "RepoFindings",
		SourceID:  id,
		EventType: api.DeleteEventType,
	})
	if evErr != nil {
		return services.HandleDeleteError("RepoFinding", evErr)
	}

	s.logAuditEvent(ctx, rf, "deleted")

	return nil
}

func (s *sqlRepoFindingService) FindByIDs(ctx context.Context, ids []string) (RepoFindingList, *errors.ServiceError) {
	items, err := s.dao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("unable to find repo findings: %s", err)
	}
	return items, nil
}

func (s *sqlRepoFindingService) All(ctx context.Context) (RepoFindingList, *errors.ServiceError) {
	items, err := s.dao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("unable to get all repo findings: %s", err)
	}
	return items, nil
}
