package projectKeys

import (
	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertProjectKey(projectKey openapi.ProjectKey) *ProjectKey {
	c := &ProjectKey{
		Meta: api.Meta{
			ID: util.NilToEmptyString(projectKey.Id),
		},
	}
	c.Name = projectKey.Name
	c.ProjectId = projectKey.ProjectId
	c.ExpiresAt = projectKey.ExpiresAt

	if projectKey.CreatedAt != nil {
		c.CreatedAt = *projectKey.CreatedAt
		c.UpdatedAt = *projectKey.UpdatedAt
	}

	return c
}

func PresentProjectKey(projectKey *ProjectKey) openapi.ProjectKey {
	reference := presenters.PresentReference(projectKey.ID, projectKey)
	result := openapi.ProjectKey{
		Id:        reference.Id,
		Kind:      reference.Kind,
		Href:      reference.Href,
		CreatedAt: openapi.PtrTime(projectKey.CreatedAt),
		UpdatedAt: openapi.PtrTime(projectKey.UpdatedAt),
		Name:      projectKey.Name,
		KeyPrefix: openapi.PtrString(projectKey.KeyPrefix),
		ProjectId: projectKey.ProjectId,
		ExpiresAt: projectKey.ExpiresAt,
	}

	if projectKey.PlaintextKey != "" {
		result.PlaintextKey = openapi.PtrString(projectKey.PlaintextKey)
	}

	return result
}
