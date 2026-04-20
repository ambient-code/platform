package credentials

import (
	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertCredential(credential openapi.Credential, projectID string) *Credential {
	c := &Credential{
		Meta: api.Meta{
			ID: util.NilToEmptyString(credential.Id),
		},
		ProjectID: projectID,
	}
	c.Name = credential.Name
	c.Description = credential.Description
	c.Provider = credential.Provider
	c.Token = credential.Token
	c.Url = credential.Url
	c.Email = credential.Email
	c.Labels = credential.Labels
	c.Annotations = credential.Annotations

	if credential.CreatedAt != nil {
		c.CreatedAt = *credential.CreatedAt
	}
	if credential.UpdatedAt != nil {
		c.UpdatedAt = *credential.UpdatedAt
	}

	return c
}

func PresentCredential(credential *Credential) openapi.Credential {
	reference := presenters.PresentReference(credential.ID, credential)
	return openapi.Credential{
		Id:          reference.Id,
		Kind:        reference.Kind,
		Href:        reference.Href,
		CreatedAt:   openapi.PtrTime(credential.CreatedAt),
		UpdatedAt:   openapi.PtrTime(credential.UpdatedAt),
		ProjectId:   credential.ProjectID,
		Name:        credential.Name,
		Description: credential.Description,
		Provider:    credential.Provider,
		Url:         credential.Url,
		Email:       credential.Email,
		Labels:      credential.Labels,
		Annotations: credential.Annotations,
	}
}

func PresentCredentialToken(credential *Credential) openapi.CredentialTokenResponse {
	return openapi.CredentialTokenResponse{
		CredentialId: credential.ID,
		Provider:     credential.Provider,
		Token:        util.NilToEmptyString(credential.Token),
	}
}
