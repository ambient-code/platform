package agents

import (
	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertAgent(agent openapi.Agent) *Agent {
	c := &Agent{
		Meta: api.Meta{
			ID: util.NilToEmptyString(agent.Id),
		},
	}
	c.Name = agent.Name
	c.RepoUrl = agent.RepoUrl
	c.Prompt = agent.Prompt
	c.ProjectId = agent.ProjectId

	if agent.CreatedAt != nil {
		c.CreatedAt = *agent.CreatedAt
		c.UpdatedAt = *agent.UpdatedAt
	}

	return c
}

func PresentAgent(agent *Agent) openapi.Agent {
	reference := presenters.PresentReference(agent.ID, agent)
	return openapi.Agent{
		Id:        reference.Id,
		Kind:      reference.Kind,
		Href:      reference.Href,
		CreatedAt: openapi.PtrTime(agent.CreatedAt),
		UpdatedAt: openapi.PtrTime(agent.UpdatedAt),
		Name:      agent.Name,
		RepoUrl:   agent.RepoUrl,
		Prompt:    agent.Prompt,
		ProjectId: agent.ProjectId,
	}
}
