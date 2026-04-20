package agents

import (
	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
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
	c.ProjectId = agent.ProjectId
	c.Name = agent.Name
	c.Prompt = agent.Prompt
	c.Labels = agent.Labels
	c.Annotations = agent.Annotations

	if agent.CreatedAt != nil {
		c.CreatedAt = *agent.CreatedAt
	}
	if agent.UpdatedAt != nil {
		c.UpdatedAt = *agent.UpdatedAt
	}

	return c
}

func PresentAgent(agent *Agent) openapi.Agent {
	reference := presenters.PresentReference(agent.ID, agent)
	return openapi.Agent{
		Id:               reference.Id,
		Kind:             reference.Kind,
		Href:             reference.Href,
		CreatedAt:        openapi.PtrTime(agent.CreatedAt),
		UpdatedAt:        openapi.PtrTime(agent.UpdatedAt),
		ProjectId:        agent.ProjectId,
		Name:             agent.Name,
		Prompt:           agent.Prompt,
		CurrentSessionId: agent.CurrentSessionId,
		Labels:           agent.Labels,
		Annotations:      agent.Annotations,
	}
}
