package agents_test

import (
	"context"
	"fmt"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient-code/platform/components/ambient-api-server/plugins/agents"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newAgent(name string) (*agents.Agent, error) {
	agentService := agents.Service(&environments.Environment().Services)

	agent := &agents.Agent{
		ProjectId: "test-project_id",
		Name:      name,
	}

	sub, err := agentService.Create(context.Background(), agent)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func newAgentWithProject(name string, projectID string) (*agents.Agent, error) {
	agentService := agents.Service(&environments.Environment().Services)

	agent := &agents.Agent{
		ProjectId: projectID,
		Name:      name,
	}

	sub, err := agentService.Create(context.Background(), agent)
	if err != nil {
		return nil, fmt.Errorf("agents.Create: %s", err.Error())
	}

	return sub, nil
}

func newAgentList(namePrefix string, count int) ([]*agents.Agent, error) {
	var items []*agents.Agent
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newAgent(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}

func presentAgent(agent *agents.Agent) openapi.Agent {
	reference := presenters.PresentReference(agent.ID, agent)
	return openapi.Agent{
		Id:             reference.Id,
		Kind:           reference.Kind,
		Href:           reference.Href,
		CreatedAt:      openapi.PtrTime(agent.CreatedAt),
		UpdatedAt:      openapi.PtrTime(agent.UpdatedAt),
		ProjectId:      agent.ProjectId,
		ParentAgentId:  agent.ParentAgentId,
		OwnerUserId:    openapi.PtrString(agent.OwnerUserId),
		Name:           agent.Name,
		DisplayName:    agent.DisplayName,
		Description:    agent.Description,
		Prompt:         agent.Prompt,
		RepoUrl:        agent.RepoUrl,
		WorkflowId:     agent.WorkflowId,
		LlmModel:       openapi.PtrString(agent.LlmModel),
		LlmTemperature: openapi.PtrFloat64(agent.LlmTemperature),
		LlmMaxTokens:   openapi.PtrInt32(agent.LlmMaxTokens),
	}
}

func stringPtr(s string) *string { return &s }

var (
	_ = newAgent
	_ = newAgentWithProject
	_ = newAgentList
	_ = presentAgent
	_ = stringPtr
)
