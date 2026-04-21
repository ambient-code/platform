package repoEvents

import (
	"context"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/logger"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

type RepoEventService interface {
	Get(ctx context.Context, id string) (*RepoEvent, *errors.ServiceError)
	Create(ctx context.Context, re *RepoEvent) (*RepoEvent, *errors.ServiceError)
	All(ctx context.Context) (RepoEventList, *errors.ServiceError)
	FindByIDs(ctx context.Context, ids []string) (RepoEventList, *errors.ServiceError)

	OnUpsert(ctx context.Context, id string) error
}

func NewRepoEventService(dao RepoEventDao) RepoEventService {
	return &sqlRepoEventService{dao: dao}
}

var _ RepoEventService = &sqlRepoEventService{}

type sqlRepoEventService struct {
	dao RepoEventDao
}

func (s *sqlRepoEventService) OnUpsert(ctx context.Context, id string) error {
	log := logger.NewLogger(ctx)
	log.Infof("RepoEvent recorded: %s", id)
	return nil
}

func (s *sqlRepoEventService) Get(ctx context.Context, id string) (*RepoEvent, *errors.ServiceError) {
	re, err := s.dao.Get(ctx, id)
	if err != nil {
		return nil, services.HandleGetError("RepoEvent", "id", id, err)
	}
	return re, nil
}

func (s *sqlRepoEventService) Create(ctx context.Context, re *RepoEvent) (*RepoEvent, *errors.ServiceError) {
	re, err := s.dao.Create(ctx, re)
	if err != nil {
		return nil, services.HandleCreateError("RepoEvent", err)
	}
	return re, nil
}

func (s *sqlRepoEventService) FindByIDs(ctx context.Context, ids []string) (RepoEventList, *errors.ServiceError) {
	items, err := s.dao.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.GeneralError("unable to find repo events: %s", err)
	}
	return items, nil
}

func (s *sqlRepoEventService) All(ctx context.Context) (RepoEventList, *errors.ServiceError) {
	items, err := s.dao.All(ctx)
	if err != nil {
		return nil, errors.GeneralError("unable to get all repo events: %s", err)
	}
	return items, nil
}
