package repoFindings

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type RepoFindingDao interface {
	Get(ctx context.Context, id string) (*RepoFinding, error)
	Create(ctx context.Context, rf *RepoFinding) (*RepoFinding, error)
	Replace(ctx context.Context, rf *RepoFinding) (*RepoFinding, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (RepoFindingList, error)
	All(ctx context.Context) (RepoFindingList, error)
	LookupProjectID(ctx context.Context, intelligenceID string) (string, error)
}

var _ RepoFindingDao = &sqlRepoFindingDao{}

type sqlRepoFindingDao struct {
	sessionFactory *db.SessionFactory
}

func NewRepoFindingDao(sessionFactory *db.SessionFactory) RepoFindingDao {
	return &sqlRepoFindingDao{sessionFactory: sessionFactory}
}

func (d *sqlRepoFindingDao) Get(ctx context.Context, id string) (*RepoFinding, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var rf RepoFinding
	if err := g2.Take(&rf, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &rf, nil
}

func (d *sqlRepoFindingDao) Create(ctx context.Context, rf *RepoFinding) (*RepoFinding, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(rf).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return rf, nil
}

func (d *sqlRepoFindingDao) Replace(ctx context.Context, rf *RepoFinding) (*RepoFinding, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(rf).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return rf, nil
}

func (d *sqlRepoFindingDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&RepoFinding{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlRepoFindingDao) FindByIDs(ctx context.Context, ids []string) (RepoFindingList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	items := RepoFindingList{}
	if err := g2.Where("id in (?)", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (d *sqlRepoFindingDao) All(ctx context.Context) (RepoFindingList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	items := RepoFindingList{}
	if err := g2.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (d *sqlRepoFindingDao) LookupProjectID(ctx context.Context, intelligenceID string) (string, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var projectID string
	err := g2.Table("repo_intelligences").
		Select("project_id").
		Where("id = ?", intelligenceID).
		Take(&projectID).Error
	return projectID, err
}
