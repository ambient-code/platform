package repositoryRefs_test

import (
	"context"
	"fmt"

	"github.com/ambient/platform/components/ambient-api-server/plugins/repositoryRefs"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newRepositoryRef(id string) (*repositoryRefs.RepositoryRef, error) {
	repositoryRefService := repositoryRefs.Service(&environments.Environment().Services)

	repositoryRef := &repositoryRefs.RepositoryRef{
		Name:      "test-name",
		Url:       "test-url",
		Branch:    stringPtr("test-branch"),
		Provider:  stringPtr("test-provider"),
		Owner:     stringPtr("test-owner"),
		RepoName:  stringPtr("test-repo_name"),
		ProjectId: stringPtr("test-project_id"),
	}

	sub, err := repositoryRefService.Create(context.Background(), repositoryRef)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func newRepositoryRefList(namePrefix string, count int) ([]*repositoryRefs.RepositoryRef, error) {
	var items []*repositoryRefs.RepositoryRef
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newRepositoryRef(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}
func stringPtr(s string) *string { return &s }
