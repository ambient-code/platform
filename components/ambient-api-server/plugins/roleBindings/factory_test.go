package roleBindings_test

import (
	"context"
	"fmt"

	"github.com/ambient-code/platform/components/ambient-api-server/plugins/roleBindings"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newRoleBinding(id string) (*roleBindings.RoleBinding, error) {
	roleBindingService := roleBindings.Service(&environments.Environment().Services)

	roleBinding := &roleBindings.RoleBinding{
		UserId:  id,
		RoleId:  "test-role_id",
		Scope:   "test-scope",
		ScopeId: stringPtr("test-scope_id"),
	}

	sub, err := roleBindingService.Create(context.Background(), roleBinding)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func newRoleBindingList(namePrefix string, count int) ([]*roleBindings.RoleBinding, error) {
	var items []*roleBindings.RoleBinding
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newRoleBinding(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}
func stringPtr(s string) *string { return &s }
