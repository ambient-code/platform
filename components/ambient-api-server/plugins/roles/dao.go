package roles

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type RoleDao interface {
	Get(ctx context.Context, id string) (*Role, error)
	Create(ctx context.Context, role *Role) (*Role, error)
	Replace(ctx context.Context, role *Role) (*Role, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (RoleList, error)
	All(ctx context.Context) (RoleList, error)
}

var _ RoleDao = &sqlRoleDao{}

type sqlRoleDao struct {
	sessionFactory *db.SessionFactory
}

func NewRoleDao(sessionFactory *db.SessionFactory) RoleDao {
	return &sqlRoleDao{sessionFactory: sessionFactory}
}

func (d *sqlRoleDao) Get(ctx context.Context, id string) (*Role, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var role Role
	if err := g2.Take(&role, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (d *sqlRoleDao) Create(ctx context.Context, role *Role) (*Role, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(role).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return role, nil
}

func (d *sqlRoleDao) Replace(ctx context.Context, role *Role) (*Role, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(role).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return role, nil
}

func (d *sqlRoleDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&Role{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlRoleDao) FindByIDs(ctx context.Context, ids []string) (RoleList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	roles := RoleList{}
	if err := g2.Where("id in (?)", ids).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (d *sqlRoleDao) All(ctx context.Context) (RoleList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	roles := RoleList{}
	if err := g2.Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}
