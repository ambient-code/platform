package projectSettings

import (
	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertProjectSettings(ps openapi.ProjectSettings) *ProjectSettings {
	c := &ProjectSettings{
		Meta: api.Meta{
			ID: util.NilToEmptyString(ps.Id),
		},
	}
	c.ProjectId = ps.ProjectId
	c.GroupAccess = ps.GroupAccess
	c.Repositories = ps.Repositories

	if ps.CreatedAt != nil {
		c.CreatedAt = *ps.CreatedAt
	}
	if ps.UpdatedAt != nil {
		c.UpdatedAt = *ps.UpdatedAt
	}

	return c
}

func PresentProjectSettings(ps *ProjectSettings) openapi.ProjectSettings {
	reference := presenters.PresentReference(ps.ID, ps)
	return openapi.ProjectSettings{
		Id:           reference.Id,
		Kind:         reference.Kind,
		Href:         reference.Href,
		CreatedAt:    openapi.PtrTime(ps.CreatedAt),
		UpdatedAt:    openapi.PtrTime(ps.UpdatedAt),
		ProjectId:    ps.ProjectId,
		GroupAccess:  ps.GroupAccess,
		Repositories: ps.Repositories,
	}
}
