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
	c.ParentAgentId = agent.ParentAgentId
	c.OwnerUserId = agent.OwnerUserId
	c.Name = agent.Name
	c.DisplayName = agent.DisplayName
	c.Description = agent.Description
	c.Prompt = agent.Prompt
	c.RepoUrl = agent.RepoUrl
	c.WorkflowId = agent.WorkflowId
	if agent.LlmModel != nil {
		c.LlmModel = *agent.LlmModel
	}
	if agent.LlmTemperature != nil {
		c.LlmTemperature = *agent.LlmTemperature
	}
	if agent.LlmMaxTokens != nil {
		c.LlmMaxTokens = *agent.LlmMaxTokens
	}
	c.BotAccountName = agent.BotAccountName
	c.ResourceOverrides = agent.ResourceOverrides
	c.EnvironmentVariables = agent.EnvironmentVariables
	c.Labels = agent.Labels
	c.Annotations = agent.Annotations
	c.CurrentSessionId = agent.CurrentSessionId

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
		Id:                   reference.Id,
		Kind:                 reference.Kind,
		Href:                 reference.Href,
		CreatedAt:            openapi.PtrTime(agent.CreatedAt),
		UpdatedAt:            openapi.PtrTime(agent.UpdatedAt),
		ProjectId:            agent.ProjectId,
		ParentAgentId:        agent.ParentAgentId,
		OwnerUserId:          agent.OwnerUserId,
		Name:                 agent.Name,
		DisplayName:          agent.DisplayName,
		Description:          agent.Description,
		Prompt:               agent.Prompt,
		RepoUrl:              agent.RepoUrl,
		WorkflowId:           agent.WorkflowId,
		LlmModel:             openapi.PtrString(agent.LlmModel),
		LlmTemperature:       openapi.PtrFloat64(agent.LlmTemperature),
		LlmMaxTokens:         openapi.PtrInt32(agent.LlmMaxTokens),
		BotAccountName:       agent.BotAccountName,
		ResourceOverrides:    agent.ResourceOverrides,
		EnvironmentVariables: agent.EnvironmentVariables,
		Labels:               agent.Labels,
		Annotations:          agent.Annotations,
		CurrentSessionId:     agent.CurrentSessionId,
	}
}
