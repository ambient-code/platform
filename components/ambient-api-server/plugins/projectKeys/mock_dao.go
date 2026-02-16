package projectKeys

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ ProjectKeyDao = &projectKeyDaoMock{}

type projectKeyDaoMock struct {
	projectKeys ProjectKeyList
}

func NewMockProjectKeyDao() *projectKeyDaoMock {
	return &projectKeyDaoMock{}
}

func (d *projectKeyDaoMock) Get(ctx context.Context, id string) (*ProjectKey, error) {
	for _, projectKey := range d.projectKeys {
		if projectKey.ID == id {
			return projectKey, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *projectKeyDaoMock) Create(ctx context.Context, projectKey *ProjectKey) (*ProjectKey, error) {
	d.projectKeys = append(d.projectKeys, projectKey)
	return projectKey, nil
}

func (d *projectKeyDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("ProjectKey").AsError()
}

func (d *projectKeyDaoMock) FindByIDs(ctx context.Context, ids []string) (ProjectKeyList, error) {
	return nil, errors.NotImplemented("ProjectKey").AsError()
}

func (d *projectKeyDaoMock) All(ctx context.Context) (ProjectKeyList, error) {
	return d.projectKeys, nil
}
