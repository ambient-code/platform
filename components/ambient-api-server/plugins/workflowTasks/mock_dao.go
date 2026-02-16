package workflowTasks

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ WorkflowTaskDao = &workflowTaskDaoMock{}

type workflowTaskDaoMock struct {
	workflowTasks WorkflowTaskList
}

func NewMockWorkflowTaskDao() *workflowTaskDaoMock {
	return &workflowTaskDaoMock{}
}

func (d *workflowTaskDaoMock) Get(ctx context.Context, id string) (*WorkflowTask, error) {
	for _, workflowTask := range d.workflowTasks {
		if workflowTask.ID == id {
			return workflowTask, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *workflowTaskDaoMock) Create(ctx context.Context, workflowTask *WorkflowTask) (*WorkflowTask, error) {
	d.workflowTasks = append(d.workflowTasks, workflowTask)
	return workflowTask, nil
}

func (d *workflowTaskDaoMock) Replace(ctx context.Context, workflowTask *WorkflowTask) (*WorkflowTask, error) {
	return nil, errors.NotImplemented("WorkflowTask").AsError()
}

func (d *workflowTaskDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("WorkflowTask").AsError()
}

func (d *workflowTaskDaoMock) FindByIDs(ctx context.Context, ids []string) (WorkflowTaskList, error) {
	return nil, errors.NotImplemented("WorkflowTask").AsError()
}

func (d *workflowTaskDaoMock) All(ctx context.Context) (WorkflowTaskList, error) {
	return d.workflowTasks, nil
}
