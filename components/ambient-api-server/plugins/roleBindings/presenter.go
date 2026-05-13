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
	c.UserId = roleBinding.UserId
	c.ProjectId = roleBinding.ProjectId
	c.AgentId = roleBinding.AgentId
	c.SessionId = roleBinding.SessionId
	c.CredentialId = roleBinding.CredentialId

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
		UserId:       roleBinding.UserId,
		ProjectId:    roleBinding.ProjectId,
		AgentId:      roleBinding.AgentId,
		SessionId:    roleBinding.SessionId,
		CredentialId: roleBinding.CredentialId,
	}
}
