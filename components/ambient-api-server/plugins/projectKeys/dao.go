package projectKeys

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type ProjectKeyDao interface {
	Get(ctx context.Context, id string) (*ProjectKey, error)
	Create(ctx context.Context, projectKey *ProjectKey) (*ProjectKey, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (ProjectKeyList, error)
	All(ctx context.Context) (ProjectKeyList, error)
}

var _ ProjectKeyDao = &sqlProjectKeyDao{}

type sqlProjectKeyDao struct {
	sessionFactory *db.SessionFactory
}

func NewProjectKeyDao(sessionFactory *db.SessionFactory) ProjectKeyDao {
	return &sqlProjectKeyDao{sessionFactory: sessionFactory}
}

func (d *sqlProjectKeyDao) Get(ctx context.Context, id string) (*ProjectKey, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var projectKey ProjectKey
	if err := g2.Take(&projectKey, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &projectKey, nil
}

func (d *sqlProjectKeyDao) Create(ctx context.Context, projectKey *ProjectKey) (*ProjectKey, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(projectKey).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return projectKey, nil
}

func (d *sqlProjectKeyDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&ProjectKey{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlProjectKeyDao) FindByIDs(ctx context.Context, ids []string) (ProjectKeyList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	projectKeys := ProjectKeyList{}
	if err := g2.Where("id in (?)", ids).Find(&projectKeys).Error; err != nil {
		return nil, err
	}
	return projectKeys, nil
}

func (d *sqlProjectKeyDao) All(ctx context.Context) (ProjectKeyList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	projectKeys := ProjectKeyList{}
	if err := g2.Find(&projectKeys).Error; err != nil {
		return nil, err
	}
	return projectKeys, nil
}
