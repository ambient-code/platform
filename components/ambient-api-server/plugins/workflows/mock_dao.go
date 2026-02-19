package workflows

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ WorkflowDao = &workflowDaoMock{}

type workflowDaoMock struct {
	workflows WorkflowList
}

func NewMockWorkflowDao() *workflowDaoMock {
	return &workflowDaoMock{}
}

func (d *workflowDaoMock) Get(ctx context.Context, id string) (*Workflow, error) {
	for _, workflow := range d.workflows {
		if workflow.ID == id {
			return workflow, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *workflowDaoMock) Create(ctx context.Context, workflow *Workflow) (*Workflow, error) {
	d.workflows = append(d.workflows, workflow)
	return workflow, nil
}

func (d *workflowDaoMock) Replace(ctx context.Context, workflow *Workflow) (*Workflow, error) {
	return nil, errors.NotImplemented("Workflow").AsError()
}

func (d *workflowDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("Workflow").AsError()
}

func (d *workflowDaoMock) FindByIDs(ctx context.Context, ids []string) (WorkflowList, error) {
	return nil, errors.NotImplemented("Workflow").AsError()
}

func (d *workflowDaoMock) All(ctx context.Context) (WorkflowList, error) {
	return d.workflows, nil
}
