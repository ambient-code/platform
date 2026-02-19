package tasks

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type TaskDao interface {
	Get(ctx context.Context, id string) (*Task, error)
	Create(ctx context.Context, task *Task) (*Task, error)
	Replace(ctx context.Context, task *Task) (*Task, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (TaskList, error)
	All(ctx context.Context) (TaskList, error)
}

var _ TaskDao = &sqlTaskDao{}

type sqlTaskDao struct {
	sessionFactory *db.SessionFactory
}

func NewTaskDao(sessionFactory *db.SessionFactory) TaskDao {
	return &sqlTaskDao{sessionFactory: sessionFactory}
}

func (d *sqlTaskDao) Get(ctx context.Context, id string) (*Task, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var task Task
	if err := g2.Take(&task, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (d *sqlTaskDao) Create(ctx context.Context, task *Task) (*Task, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(task).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return task, nil
}

func (d *sqlTaskDao) Replace(ctx context.Context, task *Task) (*Task, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(task).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return task, nil
}

func (d *sqlTaskDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&Task{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlTaskDao) FindByIDs(ctx context.Context, ids []string) (TaskList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	tasks := TaskList{}
	if err := g2.Where("id in (?)", ids).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (d *sqlTaskDao) All(ctx context.Context) (TaskList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	tasks := TaskList{}
	if err := g2.Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}
