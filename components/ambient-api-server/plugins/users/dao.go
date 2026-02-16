package users

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type UserDao interface {
	Get(ctx context.Context, id string) (*User, error)
	Create(ctx context.Context, user *User) (*User, error)
	Replace(ctx context.Context, user *User) (*User, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (UserList, error)
	All(ctx context.Context) (UserList, error)
}

var _ UserDao = &sqlUserDao{}

type sqlUserDao struct {
	sessionFactory *db.SessionFactory
}

func NewUserDao(sessionFactory *db.SessionFactory) UserDao {
	return &sqlUserDao{sessionFactory: sessionFactory}
}

func (d *sqlUserDao) Get(ctx context.Context, id string) (*User, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var user User
	if err := g2.Take(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *sqlUserDao) Create(ctx context.Context, user *User) (*User, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(user).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return user, nil
}

func (d *sqlUserDao) Replace(ctx context.Context, user *User) (*User, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(user).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return user, nil
}

func (d *sqlUserDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&User{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlUserDao) FindByIDs(ctx context.Context, ids []string) (UserList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	users := UserList{}
	if err := g2.Where("id in (?)", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (d *sqlUserDao) All(ctx context.Context) (UserList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	users := UserList{}
	if err := g2.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
