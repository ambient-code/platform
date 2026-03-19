package roles

import (
	"encoding/json"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertRole(role openapi.Role) *Role {
	c := &Role{
		Meta: api.Meta{
			ID: util.NilToEmptyString(role.Id),
		},
	}
	c.Name = role.Name
	c.DisplayName = role.DisplayName
	c.Description = role.Description
	if role.Permissions != nil {
		_ = json.Unmarshal([]byte(*role.Permissions), &c.Permissions)
	}
	if role.BuiltIn != nil {
		c.BuiltIn = *role.BuiltIn
	}

	if role.CreatedAt != nil {
		c.CreatedAt = *role.CreatedAt
	}
	if role.UpdatedAt != nil {
		c.UpdatedAt = *role.UpdatedAt
	}

	return c
}

func PresentRole(role *Role) openapi.Role {
	reference := presenters.PresentReference(role.ID, role)
	var permsStr *string
	if len(role.Permissions) > 0 {
		b, _ := json.Marshal(role.Permissions)
		s := string(b)
		permsStr = &s
	}
	return openapi.Role{
		Id:          reference.Id,
		Kind:        reference.Kind,
		Href:        reference.Href,
		CreatedAt:   openapi.PtrTime(role.CreatedAt),
		UpdatedAt:   openapi.PtrTime(role.UpdatedAt),
		Name:        role.Name,
		DisplayName: role.DisplayName,
		Description: role.Description,
		Permissions: permsStr,
		BuiltIn:     openapi.PtrBool(role.BuiltIn),
	}
}
