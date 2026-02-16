package repositoryRefs

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type RepositoryRefDao interface {
	Get(ctx context.Context, id string) (*RepositoryRef, error)
	Create(ctx context.Context, repositoryRef *RepositoryRef) (*RepositoryRef, error)
	Replace(ctx context.Context, repositoryRef *RepositoryRef) (*RepositoryRef, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (RepositoryRefList, error)
	All(ctx context.Context) (RepositoryRefList, error)
}

var _ RepositoryRefDao = &sqlRepositoryRefDao{}

type sqlRepositoryRefDao struct {
	sessionFactory *db.SessionFactory
}

func NewRepositoryRefDao(sessionFactory *db.SessionFactory) RepositoryRefDao {
	return &sqlRepositoryRefDao{sessionFactory: sessionFactory}
}

func (d *sqlRepositoryRefDao) Get(ctx context.Context, id string) (*RepositoryRef, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var repositoryRef RepositoryRef
	if err := g2.Take(&repositoryRef, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &repositoryRef, nil
}

func (d *sqlRepositoryRefDao) Create(ctx context.Context, repositoryRef *RepositoryRef) (*RepositoryRef, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(repositoryRef).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return repositoryRef, nil
}

func (d *sqlRepositoryRefDao) Replace(ctx context.Context, repositoryRef *RepositoryRef) (*RepositoryRef, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(repositoryRef).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return repositoryRef, nil
}

func (d *sqlRepositoryRefDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&RepositoryRef{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlRepositoryRefDao) FindByIDs(ctx context.Context, ids []string) (RepositoryRefList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	repositoryRefs := RepositoryRefList{}
	if err := g2.Where("id in (?)", ids).Find(&repositoryRefs).Error; err != nil {
		return nil, err
	}
	return repositoryRefs, nil
}

func (d *sqlRepositoryRefDao) All(ctx context.Context) (RepositoryRefList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	repositoryRefs := RepositoryRefList{}
	if err := g2.Find(&repositoryRefs).Error; err != nil {
		return nil, err
	}
	return repositoryRefs, nil
}
