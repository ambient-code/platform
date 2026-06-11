package applications

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ ApplicationDao = &applicationDaoMock{}

type applicationDaoMock struct {
	applications ApplicationList
}

func NewMockApplicationDao() *applicationDaoMock {
	return &applicationDaoMock{}
}

func (d *applicationDaoMock) Get(ctx context.Context, id string) (*Application, error) {
	for _, application := range d.applications {
		if application.ID == id {
			return application, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *applicationDaoMock) Create(ctx context.Context, application *Application) (*Application, error) {
	d.applications = append(d.applications, application)
	return application, nil
}

func (d *applicationDaoMock) Replace(ctx context.Context, application *Application) (*Application, error) {
	return nil, errors.NotImplemented("Application").AsError()
}

func (d *applicationDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("Application").AsError()
}

func (d *applicationDaoMock) FindByIDs(ctx context.Context, ids []string) (ApplicationList, error) {
	return nil, errors.NotImplemented("Application").AsError()
}

func (d *applicationDaoMock) All(ctx context.Context) (ApplicationList, error) {
	return d.applications, nil
}
