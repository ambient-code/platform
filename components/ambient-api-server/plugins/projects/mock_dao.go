package projects

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ ProjectDao = &projectDaoMock{}

type projectDaoMock struct {
	projects ProjectList
}

func NewMockProjectDao() *projectDaoMock {
	return &projectDaoMock{}
}

func (d *projectDaoMock) Get(ctx context.Context, id string) (*Project, error) {
	for _, project := range d.projects {
		if project.ID == id {
			return project, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *projectDaoMock) Create(ctx context.Context, project *Project) (*Project, error) {
	d.projects = append(d.projects, project)
	return project, nil
}

func (d *projectDaoMock) Replace(ctx context.Context, project *Project) (*Project, error) {
	return nil, errors.NotImplemented("Project").AsError()
}

func (d *projectDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("Project").AsError()
}

func (d *projectDaoMock) FindByIDs(ctx context.Context, ids []string) (ProjectList, error) {
	return nil, errors.NotImplemented("Project").AsError()
}

func (d *projectDaoMock) All(ctx context.Context) (ProjectList, error) {
	return d.projects, nil
}
