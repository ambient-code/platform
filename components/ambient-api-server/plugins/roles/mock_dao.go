package roles

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ RoleDao = &roleDaoMock{}

type roleDaoMock struct {
	roles RoleList
}

func NewMockRoleDao() *roleDaoMock {
	return &roleDaoMock{}
}

func (d *roleDaoMock) Get(ctx context.Context, id string) (*Role, error) {
	for _, role := range d.roles {
		if role.ID == id {
			return role, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *roleDaoMock) Create(ctx context.Context, role *Role) (*Role, error) {
	d.roles = append(d.roles, role)
	return role, nil
}

func (d *roleDaoMock) Replace(ctx context.Context, role *Role) (*Role, error) {
	return nil, errors.NotImplemented("Role").AsError()
}

func (d *roleDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("Role").AsError()
}

func (d *roleDaoMock) FindByIDs(ctx context.Context, ids []string) (RoleList, error) {
	return nil, errors.NotImplemented("Role").AsError()
}

func (d *roleDaoMock) All(ctx context.Context) (RoleList, error) {
	return d.roles, nil
}
