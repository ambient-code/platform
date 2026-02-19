package workflows

import (
	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertWorkflow(workflow openapi.Workflow) *Workflow {
	c := &Workflow{
		Meta: api.Meta{
			ID: util.NilToEmptyString(workflow.Id),
		},
	}
	c.Name = workflow.Name
	c.RepoUrl = workflow.RepoUrl
	c.Prompt = workflow.Prompt
	c.AgentId = workflow.AgentId
	c.ProjectId = workflow.ProjectId
	c.Branch = workflow.Branch
	c.Path = workflow.Path

	if workflow.CreatedAt != nil {
		c.CreatedAt = *workflow.CreatedAt
		c.UpdatedAt = *workflow.UpdatedAt
	}

	return c
}

func PresentWorkflow(workflow *Workflow) openapi.Workflow {
	reference := presenters.PresentReference(workflow.ID, workflow)
	return openapi.Workflow{
		Id:        reference.Id,
		Kind:      reference.Kind,
		Href:      reference.Href,
		CreatedAt: openapi.PtrTime(workflow.CreatedAt),
		UpdatedAt: openapi.PtrTime(workflow.UpdatedAt),
		Name:      workflow.Name,
		RepoUrl:   workflow.RepoUrl,
		Prompt:    workflow.Prompt,
		AgentId:   workflow.AgentId,
		ProjectId: workflow.ProjectId,
		Branch:    workflow.Branch,
		Path:      workflow.Path,
	}
}
