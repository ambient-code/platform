package tasks

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ TaskDao = &taskDaoMock{}

type taskDaoMock struct {
	tasks TaskList
}

func NewMockTaskDao() *taskDaoMock {
	return &taskDaoMock{}
}

func (d *taskDaoMock) Get(ctx context.Context, id string) (*Task, error) {
	for _, task := range d.tasks {
		if task.ID == id {
			return task, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *taskDaoMock) Create(ctx context.Context, task *Task) (*Task, error) {
	d.tasks = append(d.tasks, task)
	return task, nil
}

func (d *taskDaoMock) Replace(ctx context.Context, task *Task) (*Task, error) {
	return nil, errors.NotImplemented("Task").AsError()
}

func (d *taskDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("Task").AsError()
}

func (d *taskDaoMock) FindByIDs(ctx context.Context, ids []string) (TaskList, error) {
	return nil, errors.NotImplemented("Task").AsError()
}

func (d *taskDaoMock) All(ctx context.Context) (TaskList, error) {
	return d.tasks, nil
}
