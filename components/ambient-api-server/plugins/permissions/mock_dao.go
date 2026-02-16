package permissions

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ PermissionDao = &permissionDaoMock{}

type permissionDaoMock struct {
	permissions PermissionList
}

func NewMockPermissionDao() *permissionDaoMock {
	return &permissionDaoMock{}
}

func (d *permissionDaoMock) Get(ctx context.Context, id string) (*Permission, error) {
	for _, permission := range d.permissions {
		if permission.ID == id {
			return permission, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *permissionDaoMock) Create(ctx context.Context, permission *Permission) (*Permission, error) {
	d.permissions = append(d.permissions, permission)
	return permission, nil
}

func (d *permissionDaoMock) Replace(ctx context.Context, permission *Permission) (*Permission, error) {
	return nil, errors.NotImplemented("Permission").AsError()
}

func (d *permissionDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("Permission").AsError()
}

func (d *permissionDaoMock) FindByIDs(ctx context.Context, ids []string) (PermissionList, error) {
	return nil, errors.NotImplemented("Permission").AsError()
}

func (d *permissionDaoMock) All(ctx context.Context) (PermissionList, error) {
	return d.permissions, nil
}
