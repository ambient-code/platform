package projectAgents

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ ProjectAgentDao = &projectAgentDaoMock{}

type projectAgentDaoMock struct {
	projectAgents ProjectAgentList
}

func NewMockProjectAgentDao() *projectAgentDaoMock {
	return &projectAgentDaoMock{}
}

func (d *projectAgentDaoMock) Get(ctx context.Context, id string) (*ProjectAgent, error) {
	for _, projectAgent := range d.projectAgents {
		if projectAgent.ID == id {
			return projectAgent, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *projectAgentDaoMock) Create(ctx context.Context, projectAgent *ProjectAgent) (*ProjectAgent, error) {
	d.projectAgents = append(d.projectAgents, projectAgent)
	return projectAgent, nil
}

func (d *projectAgentDaoMock) Replace(ctx context.Context, projectAgent *ProjectAgent) (*ProjectAgent, error) {
	return nil, errors.NotImplemented("ProjectAgent").AsError()
}

func (d *projectAgentDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("ProjectAgent").AsError()
}

func (d *projectAgentDaoMock) FindByIDs(ctx context.Context, ids []string) (ProjectAgentList, error) {
	return nil, errors.NotImplemented("ProjectAgent").AsError()
}

func (d *projectAgentDaoMock) All(ctx context.Context) (ProjectAgentList, error) {
	return d.projectAgents, nil
}

func (d *projectAgentDaoMock) AllByProjectID(ctx context.Context, projectID string) (ProjectAgentList, error) {
	var result ProjectAgentList
	for _, pa := range d.projectAgents {
		if pa.ProjectId == projectID {
			result = append(result, pa)
		}
	}
	return result, nil
}
