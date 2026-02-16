package repositoryRefs

import (
	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertRepositoryRef(repositoryRef openapi.RepositoryRef) *RepositoryRef {
	c := &RepositoryRef{
		Meta: api.Meta{
			ID: util.NilToEmptyString(repositoryRef.Id),
		},
	}
	c.Name = repositoryRef.Name
	c.Url = repositoryRef.Url
	c.Branch = repositoryRef.Branch
	c.Provider = repositoryRef.Provider
	c.Owner = repositoryRef.Owner
	c.RepoName = repositoryRef.RepoName
	c.ProjectId = repositoryRef.ProjectId

	if repositoryRef.CreatedAt != nil {
		c.CreatedAt = *repositoryRef.CreatedAt
		c.UpdatedAt = *repositoryRef.UpdatedAt
	}

	return c
}

func PresentRepositoryRef(repositoryRef *RepositoryRef) openapi.RepositoryRef {
	reference := presenters.PresentReference(repositoryRef.ID, repositoryRef)
	return openapi.RepositoryRef{
		Id:        reference.Id,
		Kind:      reference.Kind,
		Href:      reference.Href,
		CreatedAt: openapi.PtrTime(repositoryRef.CreatedAt),
		UpdatedAt: openapi.PtrTime(repositoryRef.UpdatedAt),
		Name:      repositoryRef.Name,
		Url:       repositoryRef.Url,
		Branch:    repositoryRef.Branch,
		Provider:  repositoryRef.Provider,
		Owner:     repositoryRef.Owner,
		RepoName:  repositoryRef.RepoName,
		ProjectId: repositoryRef.ProjectId,
	}
}
