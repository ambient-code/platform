package users_test

import (
	"context"
	"fmt"

	"github.com/ambient/platform/components/ambient-api-server/plugins/users"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newUser(id string) (*users.User, error) {
	userService := users.Service(&environments.Environment().Services)

	user := &users.User{
		Username: "test-username",
		Name:     "test-name",
	}

	sub, err := userService.Create(context.Background(), user)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func newUserList(namePrefix string, count int) ([]*users.User, error) {
	var items []*users.User
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newUser(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}
