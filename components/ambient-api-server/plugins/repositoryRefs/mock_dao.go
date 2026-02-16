package repositoryRefs

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ RepositoryRefDao = &repositoryRefDaoMock{}

type repositoryRefDaoMock struct {
	repositoryRefs RepositoryRefList
}

func NewMockRepositoryRefDao() *repositoryRefDaoMock {
	return &repositoryRefDaoMock{}
}

func (d *repositoryRefDaoMock) Get(ctx context.Context, id string) (*RepositoryRef, error) {
	for _, repositoryRef := range d.repositoryRefs {
		if repositoryRef.ID == id {
			return repositoryRef, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *repositoryRefDaoMock) Create(ctx context.Context, repositoryRef *RepositoryRef) (*RepositoryRef, error) {
	d.repositoryRefs = append(d.repositoryRefs, repositoryRef)
	return repositoryRef, nil
}

func (d *repositoryRefDaoMock) Replace(ctx context.Context, repositoryRef *RepositoryRef) (*RepositoryRef, error) {
	return nil, errors.NotImplemented("RepositoryRef").AsError()
}

func (d *repositoryRefDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("RepositoryRef").AsError()
}

func (d *repositoryRefDaoMock) FindByIDs(ctx context.Context, ids []string) (RepositoryRefList, error) {
	return nil, errors.NotImplemented("RepositoryRef").AsError()
}

func (d *repositoryRefDaoMock) All(ctx context.Context) (RepositoryRefList, error) {
	return d.repositoryRefs, nil
}
