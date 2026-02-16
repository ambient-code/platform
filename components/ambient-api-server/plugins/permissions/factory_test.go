package permissions_test

import (
	"context"
	"fmt"

	"github.com/ambient/platform/components/ambient-api-server/plugins/permissions"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newPermission(id string) (*permissions.Permission, error) {
	permissionService := permissions.Service(&environments.Environment().Services)

	permission := &permissions.Permission{
		SubjectType: "user",
		SubjectName: "test-user",
		Role:        "edit",
		ProjectId:   stringPtr("test-project_id"),
	}

	sub, err := permissionService.Create(context.Background(), permission)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func newPermissionList(namePrefix string, count int) ([]*permissions.Permission, error) {
	var items []*permissions.Permission
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newPermission(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}
func stringPtr(s string) *string { return &s }
