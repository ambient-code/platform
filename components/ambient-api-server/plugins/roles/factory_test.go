package roles_test

import (
	"context"
	"fmt"

	"github.com/ambient-code/platform/components/ambient-api-server/plugins/roles"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newRole(id string) (*roles.Role, error) {
	roleService := roles.Service(&environments.Environment().Services)

	role := &roles.Role{
		Name:        id,
		DisplayName: stringPtr("test-display_name"),
		Description: stringPtr("test-description"),
		Permissions: []string{"test-permissions"},
		BuiltIn:     true,
	}

	sub, err := roleService.Create(context.Background(), role)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func newRoleList(namePrefix string, count int) ([]*roles.Role, error) {
	var items []*roles.Role
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newRole(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}
func stringPtr(s string) *string { return &s }
