package projectDocuments

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ ProjectDocumentDao = &projectDocumentDaoMock{}

type projectDocumentDaoMock struct {
	projectDocuments ProjectDocumentList
}

func NewMockProjectDocumentDao() *projectDocumentDaoMock {
	return &projectDocumentDaoMock{}
}

func (d *projectDocumentDaoMock) Get(ctx context.Context, id string) (*ProjectDocument, error) {
	for _, projectDocument := range d.projectDocuments {
		if projectDocument.ID == id {
			return projectDocument, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *projectDocumentDaoMock) Create(ctx context.Context, projectDocument *ProjectDocument) (*ProjectDocument, error) {
	d.projectDocuments = append(d.projectDocuments, projectDocument)
	return projectDocument, nil
}

func (d *projectDocumentDaoMock) Replace(ctx context.Context, projectDocument *ProjectDocument) (*ProjectDocument, error) {
	return nil, errors.NotImplemented("ProjectDocument").AsError()
}

func (d *projectDocumentDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("ProjectDocument").AsError()
}

func (d *projectDocumentDaoMock) FindByIDs(ctx context.Context, ids []string) (ProjectDocumentList, error) {
	return nil, errors.NotImplemented("ProjectDocument").AsError()
}

func (d *projectDocumentDaoMock) All(ctx context.Context) (ProjectDocumentList, error) {
	return d.projectDocuments, nil
}
