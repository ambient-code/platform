package workflowSkills

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ WorkflowSkillDao = &workflowSkillDaoMock{}

type workflowSkillDaoMock struct {
	workflowSkills WorkflowSkillList
}

func NewMockWorkflowSkillDao() *workflowSkillDaoMock {
	return &workflowSkillDaoMock{}
}

func (d *workflowSkillDaoMock) Get(ctx context.Context, id string) (*WorkflowSkill, error) {
	for _, workflowSkill := range d.workflowSkills {
		if workflowSkill.ID == id {
			return workflowSkill, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *workflowSkillDaoMock) Create(ctx context.Context, workflowSkill *WorkflowSkill) (*WorkflowSkill, error) {
	d.workflowSkills = append(d.workflowSkills, workflowSkill)
	return workflowSkill, nil
}

func (d *workflowSkillDaoMock) Replace(ctx context.Context, workflowSkill *WorkflowSkill) (*WorkflowSkill, error) {
	return nil, errors.NotImplemented("WorkflowSkill").AsError()
}

func (d *workflowSkillDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("WorkflowSkill").AsError()
}

func (d *workflowSkillDaoMock) FindByIDs(ctx context.Context, ids []string) (WorkflowSkillList, error) {
	return nil, errors.NotImplemented("WorkflowSkill").AsError()
}

func (d *workflowSkillDaoMock) All(ctx context.Context) (WorkflowSkillList, error) {
	return d.workflowSkills, nil
}
