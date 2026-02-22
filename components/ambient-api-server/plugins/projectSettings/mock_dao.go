package projectSettings

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ ProjectSettingsDao = &projectSettingsDaoMock{}

type projectSettingsDaoMock struct {
	items ProjectSettingsList
}

func NewMockProjectSettingsDao() *projectSettingsDaoMock {
	return &projectSettingsDaoMock{}
}

func (d *projectSettingsDaoMock) Get(ctx context.Context, id string) (*ProjectSettings, error) {
	for _, item := range d.items {
		if item.ID == id {
			return item, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *projectSettingsDaoMock) Create(ctx context.Context, ps *ProjectSettings) (*ProjectSettings, error) {
	d.items = append(d.items, ps)
	return ps, nil
}

func (d *projectSettingsDaoMock) Replace(ctx context.Context, ps *ProjectSettings) (*ProjectSettings, error) {
	return nil, errors.NotImplemented("ProjectSettings").AsError()
}

func (d *projectSettingsDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("ProjectSettings").AsError()
}

func (d *projectSettingsDaoMock) FindByIDs(ctx context.Context, ids []string) (ProjectSettingsList, error) {
	return nil, errors.NotImplemented("ProjectSettings").AsError()
}

func (d *projectSettingsDaoMock) All(ctx context.Context) (ProjectSettingsList, error) {
	return d.items, nil
}

func (d *projectSettingsDaoMock) AllByProjectId(ctx context.Context, projectId string) (ProjectSettingsList, error) {
	var filtered ProjectSettingsList
	for _, item := range d.items {
		if item.ProjectId == projectId {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}
