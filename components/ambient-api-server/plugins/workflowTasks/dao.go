package workflowTasks

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type WorkflowTaskDao interface {
	Get(ctx context.Context, id string) (*WorkflowTask, error)
	Create(ctx context.Context, workflowTask *WorkflowTask) (*WorkflowTask, error)
	Replace(ctx context.Context, workflowTask *WorkflowTask) (*WorkflowTask, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (WorkflowTaskList, error)
	All(ctx context.Context) (WorkflowTaskList, error)
}

var _ WorkflowTaskDao = &sqlWorkflowTaskDao{}

type sqlWorkflowTaskDao struct {
	sessionFactory *db.SessionFactory
}

func NewWorkflowTaskDao(sessionFactory *db.SessionFactory) WorkflowTaskDao {
	return &sqlWorkflowTaskDao{sessionFactory: sessionFactory}
}

func (d *sqlWorkflowTaskDao) Get(ctx context.Context, id string) (*WorkflowTask, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var workflowTask WorkflowTask
	if err := g2.Take(&workflowTask, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &workflowTask, nil
}

func (d *sqlWorkflowTaskDao) Create(ctx context.Context, workflowTask *WorkflowTask) (*WorkflowTask, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(workflowTask).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return workflowTask, nil
}

func (d *sqlWorkflowTaskDao) Replace(ctx context.Context, workflowTask *WorkflowTask) (*WorkflowTask, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(workflowTask).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return workflowTask, nil
}

func (d *sqlWorkflowTaskDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&WorkflowTask{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlWorkflowTaskDao) FindByIDs(ctx context.Context, ids []string) (WorkflowTaskList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	workflowTasks := WorkflowTaskList{}
	if err := g2.Where("id in (?)", ids).Find(&workflowTasks).Error; err != nil {
		return nil, err
	}
	return workflowTasks, nil
}

func (d *sqlWorkflowTaskDao) All(ctx context.Context) (WorkflowTaskList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	workflowTasks := WorkflowTaskList{}
	if err := g2.Find(&workflowTasks).Error; err != nil {
		return nil, err
	}
	return workflowTasks, nil
}
