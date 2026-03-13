package roleBindings

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ RoleBindingDao = &roleBindingDaoMock{}

type roleBindingDaoMock struct {
	roleBindings RoleBindingList
}

func NewMockRoleBindingDao() *roleBindingDaoMock {
	return &roleBindingDaoMock{}
}

func (d *roleBindingDaoMock) Get(ctx context.Context, id string) (*RoleBinding, error) {
	for _, roleBinding := range d.roleBindings {
		if roleBinding.ID == id {
			return roleBinding, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *roleBindingDaoMock) Create(ctx context.Context, roleBinding *RoleBinding) (*RoleBinding, error) {
	d.roleBindings = append(d.roleBindings, roleBinding)
	return roleBinding, nil
}

func (d *roleBindingDaoMock) Replace(ctx context.Context, roleBinding *RoleBinding) (*RoleBinding, error) {
	return nil, errors.NotImplemented("RoleBinding").AsError()
}

func (d *roleBindingDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("RoleBinding").AsError()
}

func (d *roleBindingDaoMock) FindByIDs(ctx context.Context, ids []string) (RoleBindingList, error) {
	return nil, errors.NotImplemented("RoleBinding").AsError()
}

func (d *roleBindingDaoMock) All(ctx context.Context) (RoleBindingList, error) {
	return d.roleBindings, nil
}
