package workflows

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type WorkflowDao interface {
	Get(ctx context.Context, id string) (*Workflow, error)
	Create(ctx context.Context, workflow *Workflow) (*Workflow, error)
	Replace(ctx context.Context, workflow *Workflow) (*Workflow, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (WorkflowList, error)
	All(ctx context.Context) (WorkflowList, error)
}

var _ WorkflowDao = &sqlWorkflowDao{}

type sqlWorkflowDao struct {
	sessionFactory *db.SessionFactory
}

func NewWorkflowDao(sessionFactory *db.SessionFactory) WorkflowDao {
	return &sqlWorkflowDao{sessionFactory: sessionFactory}
}

func (d *sqlWorkflowDao) Get(ctx context.Context, id string) (*Workflow, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var workflow Workflow
	if err := g2.Take(&workflow, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &workflow, nil
}

func (d *sqlWorkflowDao) Create(ctx context.Context, workflow *Workflow) (*Workflow, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(workflow).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return workflow, nil
}

func (d *sqlWorkflowDao) Replace(ctx context.Context, workflow *Workflow) (*Workflow, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(workflow).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return workflow, nil
}

func (d *sqlWorkflowDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&Workflow{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlWorkflowDao) FindByIDs(ctx context.Context, ids []string) (WorkflowList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	workflows := WorkflowList{}
	if err := g2.Where("id in (?)", ids).Find(&workflows).Error; err != nil {
		return nil, err
	}
	return workflows, nil
}

func (d *sqlWorkflowDao) All(ctx context.Context) (WorkflowList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	workflows := WorkflowList{}
	if err := g2.Find(&workflows).Error; err != nil {
		return nil, err
	}
	return workflows, nil
}
