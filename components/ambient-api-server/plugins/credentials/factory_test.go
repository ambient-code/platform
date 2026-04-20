package credentials_test

import (
	"context"
	"fmt"

	"github.com/ambient-code/platform/components/ambient-api-server/plugins/credentials"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newCredential(name string) (*credentials.Credential, error) {
	credentialService := credentials.Service(&environments.Environment().Services)

	credential := &credentials.Credential{
		ProjectID:   "test-project",
		Name:        name,
		Description: stringPtr("test-description"),
		Provider:    "test-provider",
		Token:       stringPtr("test-token"),
		Url:         stringPtr("test-url"),
		Email:       stringPtr("test-email"),
		Labels:      stringPtr("test-labels"),
		Annotations: stringPtr("test-annotations"),
	}

	sub, err := credentialService.Create(context.Background(), credential)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func newCredentialList(namePrefix string, count int) ([]*credentials.Credential, error) {
	var items []*credentials.Credential
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newCredential(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}
func stringPtr(s string) *string { return &s }
