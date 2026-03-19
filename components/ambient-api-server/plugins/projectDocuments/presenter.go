package projectDocuments

import (
	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertProjectDocument(projectDocument openapi.ProjectDocument) *ProjectDocument {
	c := &ProjectDocument{
		Meta: api.Meta{
			ID: util.NilToEmptyString(projectDocument.Id),
		},
	}
	c.ProjectId = projectDocument.ProjectId
	c.Slug = projectDocument.Slug
	c.Title = projectDocument.Title
	c.Content = projectDocument.Content

	if projectDocument.CreatedAt != nil {
		c.CreatedAt = *projectDocument.CreatedAt
		c.UpdatedAt = *projectDocument.UpdatedAt
	}

	return c
}

func PresentProjectDocument(projectDocument *ProjectDocument) openapi.ProjectDocument {
	reference := presenters.PresentReference(projectDocument.ID, projectDocument)
	return openapi.ProjectDocument{
		Id:        reference.Id,
		Kind:      reference.Kind,
		Href:      reference.Href,
		CreatedAt: openapi.PtrTime(projectDocument.CreatedAt),
		UpdatedAt: openapi.PtrTime(projectDocument.UpdatedAt),
		ProjectId: projectDocument.ProjectId,
		Slug:      projectDocument.Slug,
		Title:     projectDocument.Title,
		Content:   projectDocument.Content,
	}
}
