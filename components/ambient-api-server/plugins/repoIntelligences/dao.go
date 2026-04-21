package repoIntelligences

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type RepoIntelligenceDao interface {
	Get(ctx context.Context, id string) (*RepoIntelligence, error)
	Create(ctx context.Context, ri *RepoIntelligence) (*RepoIntelligence, error)
	Replace(ctx context.Context, ri *RepoIntelligence) (*RepoIntelligence, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (RepoIntelligenceList, error)
	All(ctx context.Context) (RepoIntelligenceList, error)
	GetByProjectAndRepo(ctx context.Context, projectID, repoURL string) (*RepoIntelligence, error)
}

var _ RepoIntelligenceDao = &sqlRepoIntelligenceDao{}

type sqlRepoIntelligenceDao struct {
	sessionFactory *db.SessionFactory
}

func NewRepoIntelligenceDao(sessionFactory *db.SessionFactory) RepoIntelligenceDao {
	return &sqlRepoIntelligenceDao{sessionFactory: sessionFactory}
}

func (d *sqlRepoIntelligenceDao) Get(ctx context.Context, id string) (*RepoIntelligence, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var ri RepoIntelligence
	if err := g2.Take(&ri, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &ri, nil
}

func (d *sqlRepoIntelligenceDao) Create(ctx context.Context, ri *RepoIntelligence) (*RepoIntelligence, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(ri).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return ri, nil
}

func (d *sqlRepoIntelligenceDao) Replace(ctx context.Context, ri *RepoIntelligence) (*RepoIntelligence, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(ri).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return ri, nil
}

func (d *sqlRepoIntelligenceDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)

	// Cascade soft-delete child findings to prevent orphans.
	if err := g2.Exec(
		"UPDATE repo_findings SET deleted_at = NOW() WHERE intelligence_id = ? AND deleted_at IS NULL", id,
	).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}

	if err := g2.Omit(clause.Associations).Delete(&RepoIntelligence{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlRepoIntelligenceDao) FindByIDs(ctx context.Context, ids []string) (RepoIntelligenceList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	items := RepoIntelligenceList{}
	if err := g2.Where("id in (?)", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (d *sqlRepoIntelligenceDao) All(ctx context.Context) (RepoIntelligenceList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	items := RepoIntelligenceList{}
	if err := g2.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (d *sqlRepoIntelligenceDao) GetByProjectAndRepo(ctx context.Context, projectID, repoURL string) (*RepoIntelligence, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var ri RepoIntelligence
	if err := g2.Where("project_id = ? AND repo_url = ?", projectID, repoURL).Take(&ri).Error; err != nil {
		return nil, err
	}
	return &ri, nil
}
