package permissions

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type PermissionDao interface {
	Get(ctx context.Context, id string) (*Permission, error)
	Create(ctx context.Context, permission *Permission) (*Permission, error)
	Replace(ctx context.Context, permission *Permission) (*Permission, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (PermissionList, error)
	All(ctx context.Context) (PermissionList, error)
}

var _ PermissionDao = &sqlPermissionDao{}

type sqlPermissionDao struct {
	sessionFactory *db.SessionFactory
}

func NewPermissionDao(sessionFactory *db.SessionFactory) PermissionDao {
	return &sqlPermissionDao{sessionFactory: sessionFactory}
}

func (d *sqlPermissionDao) Get(ctx context.Context, id string) (*Permission, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var permission Permission
	if err := g2.Take(&permission, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &permission, nil
}

func (d *sqlPermissionDao) Create(ctx context.Context, permission *Permission) (*Permission, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(permission).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return permission, nil
}

func (d *sqlPermissionDao) Replace(ctx context.Context, permission *Permission) (*Permission, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(permission).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return permission, nil
}

func (d *sqlPermissionDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&Permission{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlPermissionDao) FindByIDs(ctx context.Context, ids []string) (PermissionList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	permissions := PermissionList{}
	if err := g2.Where("id in (?)", ids).Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

func (d *sqlPermissionDao) All(ctx context.Context) (PermissionList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	permissions := PermissionList{}
	if err := g2.Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}
