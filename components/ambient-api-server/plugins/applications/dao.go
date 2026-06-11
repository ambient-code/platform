package applications

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type ApplicationDao interface {
	Get(ctx context.Context, id string) (*Application, error)
	Create(ctx context.Context, application *Application) (*Application, error)
	Replace(ctx context.Context, application *Application) (*Application, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (ApplicationList, error)
	All(ctx context.Context) (ApplicationList, error)
}

var _ ApplicationDao = &sqlApplicationDao{}

type sqlApplicationDao struct {
	sessionFactory *db.SessionFactory
}

func NewApplicationDao(sessionFactory *db.SessionFactory) ApplicationDao {
	return &sqlApplicationDao{sessionFactory: sessionFactory}
}

func (d *sqlApplicationDao) Get(ctx context.Context, id string) (*Application, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var application Application
	if err := g2.Take(&application, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &application, nil
}

func (d *sqlApplicationDao) Create(ctx context.Context, application *Application) (*Application, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(application).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return application, nil
}

func (d *sqlApplicationDao) Replace(ctx context.Context, application *Application) (*Application, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(application).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return application, nil
}

func (d *sqlApplicationDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&Application{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlApplicationDao) FindByIDs(ctx context.Context, ids []string) (ApplicationList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	applications := ApplicationList{}
	if err := g2.Where("id in (?)", ids).Find(&applications).Error; err != nil {
		return nil, err
	}
	return applications, nil
}

func (d *sqlApplicationDao) All(ctx context.Context) (ApplicationList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	applications := ApplicationList{}
	if err := g2.Find(&applications).Error; err != nil {
		return nil, err
	}
	return applications, nil
}
