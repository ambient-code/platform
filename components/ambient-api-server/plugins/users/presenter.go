package users

import (
	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertUser(user openapi.User) *User {
	c := &User{
		Meta: api.Meta{
			ID: util.NilToEmptyString(user.Id),
		},
	}
	c.Username = user.Username
	c.Name = user.Name
	c.Email = user.Email

	if user.CreatedAt != nil {
		c.CreatedAt = *user.CreatedAt
		c.UpdatedAt = *user.UpdatedAt
	}

	return c
}

func PresentUser(user *User) openapi.User {
	reference := presenters.PresentReference(user.ID, user)
	return openapi.User{
		Id:        reference.Id,
		Kind:      reference.Kind,
		Href:      reference.Href,
		CreatedAt: openapi.PtrTime(user.CreatedAt),
		UpdatedAt: openapi.PtrTime(user.UpdatedAt),
		Username:  user.Username,
		Name:      user.Name,
		Email:     user.Email,
	}
}
