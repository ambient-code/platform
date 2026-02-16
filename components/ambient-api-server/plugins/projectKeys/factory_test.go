package projectKeys_test

import (
	"context"
	"fmt"

	"github.com/ambient/platform/components/ambient-api-server/plugins/projectKeys"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newProjectKey(name string) (*projectKeys.ProjectKey, error) {
	projectKeyService := projectKeys.Service(&environments.Environment().Services)

	projectKey := &projectKeys.ProjectKey{
		Name:      name,
		ProjectId: stringPtr("test-project"),
	}

	created, err := projectKeyService.Create(context.Background(), projectKey)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func newProjectKeyList(namePrefix string, count int) ([]*projectKeys.ProjectKey, error) {
	var items []*projectKeys.ProjectKey
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newProjectKey(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}

func stringPtr(s string) *string { return &s }
