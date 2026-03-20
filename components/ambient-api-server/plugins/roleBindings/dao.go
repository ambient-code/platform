package roleBindings

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type RoleBindingDao interface {
	Get(ctx context.Context, id string) (*RoleBinding, error)
	Create(ctx context.Context, roleBinding *RoleBinding) (*RoleBinding, error)
	Replace(ctx context.Context, roleBinding *RoleBinding) (*RoleBinding, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (RoleBindingList, error)
	All(ctx context.Context) (RoleBindingList, error)
}

var _ RoleBindingDao = &sqlRoleBindingDao{}

type sqlRoleBindingDao struct {
	sessionFactory *db.SessionFactory
}

func NewRoleBindingDao(sessionFactory *db.SessionFactory) RoleBindingDao {
	return &sqlRoleBindingDao{sessionFactory: sessionFactory}
}

func (d *sqlRoleBindingDao) Get(ctx context.Context, id string) (*RoleBinding, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var roleBinding RoleBinding
	if err := g2.Take(&roleBinding, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &roleBinding, nil
}

func (d *sqlRoleBindingDao) Create(ctx context.Context, roleBinding *RoleBinding) (*RoleBinding, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(roleBinding).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return roleBinding, nil
}

func (d *sqlRoleBindingDao) Replace(ctx context.Context, roleBinding *RoleBinding) (*RoleBinding, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(roleBinding).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return roleBinding, nil
}

func (d *sqlRoleBindingDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&RoleBinding{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlRoleBindingDao) FindByIDs(ctx context.Context, ids []string) (RoleBindingList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	roleBindings := RoleBindingList{}
	if err := g2.Where("id in (?)", ids).Find(&roleBindings).Error; err != nil {
		return nil, err
	}
	return roleBindings, nil
}

func (d *sqlRoleBindingDao) All(ctx context.Context) (RoleBindingList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	roleBindings := RoleBindingList{}
	if err := g2.Find(&roleBindings).Error; err != nil {
		return nil, err
	}
	return roleBindings, nil
}
