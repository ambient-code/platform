package roleBindings

import (
	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertRoleBinding(roleBinding openapi.RoleBinding) *RoleBinding {
	c := &RoleBinding{
		Meta: api.Meta{
			ID: util.NilToEmptyString(roleBinding.Id),
		},
	}
	c.RoleId = roleBinding.RoleId
	c.Scope = roleBinding.Scope
	c.UserId = roleBinding.UserId.Get()
	c.ProjectId = roleBinding.ProjectId.Get()
	c.AgentId = roleBinding.AgentId.Get()
	c.SessionId = roleBinding.SessionId.Get()
	c.CredentialId = roleBinding.CredentialId.Get()

	if roleBinding.CreatedAt != nil {
		c.CreatedAt = *roleBinding.CreatedAt
		c.UpdatedAt = *roleBinding.UpdatedAt
	}

	return c
}

func PresentRoleBinding(roleBinding *RoleBinding) openapi.RoleBinding {
	reference := presenters.PresentReference(roleBinding.ID, roleBinding)
	return openapi.RoleBinding{
		Id:           reference.Id,
		Kind:         reference.Kind,
		Href:         reference.Href,
		CreatedAt:    openapi.PtrTime(roleBinding.CreatedAt),
		UpdatedAt:    openapi.PtrTime(roleBinding.UpdatedAt),
		RoleId:       roleBinding.RoleId,
		Scope:        roleBinding.Scope,
		UserId:       *openapi.NewNullableString(roleBinding.UserId),
		ProjectId:    *openapi.NewNullableString(roleBinding.ProjectId),
		AgentId:      *openapi.NewNullableString(roleBinding.AgentId),
		SessionId:    *openapi.NewNullableString(roleBinding.SessionId),
		CredentialId: *openapi.NewNullableString(roleBinding.CredentialId),
	}
}
