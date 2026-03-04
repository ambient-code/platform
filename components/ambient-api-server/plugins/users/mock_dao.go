package users

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ UserDao = &userDaoMock{}

type userDaoMock struct {
	users UserList
}

func NewMockUserDao() *userDaoMock {
	return &userDaoMock{}
}

func (d *userDaoMock) Get(ctx context.Context, id string) (*User, error) {
	for _, user := range d.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *userDaoMock) Create(ctx context.Context, user *User) (*User, error) {
	d.users = append(d.users, user)
	return user, nil
}

func (d *userDaoMock) Replace(ctx context.Context, user *User) (*User, error) {
	return nil, errors.NotImplemented("User").AsError()
}

func (d *userDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("User").AsError()
}

func (d *userDaoMock) FindByIDs(ctx context.Context, ids []string) (UserList, error) {
	return nil, errors.NotImplemented("User").AsError()
}

func (d *userDaoMock) All(ctx context.Context) (UserList, error) {
	return d.users, nil
}
