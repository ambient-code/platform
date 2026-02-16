package permissions

import (
	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertPermission(permission openapi.Permission) *Permission {
	c := &Permission{
		Meta: api.Meta{
			ID: util.NilToEmptyString(permission.Id),
		},
	}
	c.SubjectType = permission.SubjectType
	c.SubjectName = permission.SubjectName
	c.Role = permission.Role
	c.ProjectId = permission.ProjectId

	if permission.CreatedAt != nil {
		c.CreatedAt = *permission.CreatedAt
		c.UpdatedAt = *permission.UpdatedAt
	}

	return c
}

func PresentPermission(permission *Permission) openapi.Permission {
	reference := presenters.PresentReference(permission.ID, permission)
	return openapi.Permission{
		Id:          reference.Id,
		Kind:        reference.Kind,
		Href:        reference.Href,
		CreatedAt:   openapi.PtrTime(permission.CreatedAt),
		UpdatedAt:   openapi.PtrTime(permission.UpdatedAt),
		SubjectType: permission.SubjectType,
		SubjectName: permission.SubjectName,
		Role:        permission.Role,
		ProjectId:   permission.ProjectId,
	}
}
