package repoEvents

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type RepoEventDao interface {
	Get(ctx context.Context, id string) (*RepoEvent, error)
	Create(ctx context.Context, re *RepoEvent) (*RepoEvent, error)
	FindByIDs(ctx context.Context, ids []string) (RepoEventList, error)
	All(ctx context.Context) (RepoEventList, error)
}

var _ RepoEventDao = &sqlRepoEventDao{}

type sqlRepoEventDao struct {
	sessionFactory *db.SessionFactory
}

func NewRepoEventDao(sessionFactory *db.SessionFactory) RepoEventDao {
	return &sqlRepoEventDao{sessionFactory: sessionFactory}
}

func (d *sqlRepoEventDao) Get(ctx context.Context, id string) (*RepoEvent, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var re RepoEvent
	if err := g2.Take(&re, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &re, nil
}

func (d *sqlRepoEventDao) Create(ctx context.Context, re *RepoEvent) (*RepoEvent, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(re).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return re, nil
}

func (d *sqlRepoEventDao) FindByIDs(ctx context.Context, ids []string) (RepoEventList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	items := RepoEventList{}
	if err := g2.Where("id in (?)", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (d *sqlRepoEventDao) All(ctx context.Context) (RepoEventList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	items := RepoEventList{}
	if err := g2.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
