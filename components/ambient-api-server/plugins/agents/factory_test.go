package agents_test

import (
	"context"
	"fmt"

	"github.com/ambient-code/platform/components/ambient-api-server/plugins/agents"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newAgent(id string) (*agents.Agent, error) {
	agentService := agents.Service(&environments.Environment().Services)

	agent := &agents.Agent{
		ProjectId:            "test-project_id",
		ParentAgentId:        stringPtr("test-parent_agent_id"),
		OwnerUserId:          "test-owner_user_id",
		Name:                 "test-name",
		DisplayName:          stringPtr("test-display_name"),
		Description:          stringPtr("test-description"),
		Prompt:               stringPtr("test-prompt"),
		RepoUrl:              stringPtr("test-repo_url"),
		WorkflowId:           stringPtr("test-workflow_id"),
		LlmModel:             "test-llm_model",
		LlmTemperature:       3.14,
		LlmMaxTokens:         42,
		BotAccountName:       stringPtr("test-bot_account_name"),
		ResourceOverrides:    stringPtr("test-resource_overrides"),
		EnvironmentVariables: stringPtr("test-environment_variables"),
		Labels:               stringPtr("test-labels"),
		Annotations:          stringPtr("test-annotations"),
		CurrentSessionId:     stringPtr("test-current_session_id"),
	}

	sub, err := agentService.Create(context.Background(), agent)
	if err != nil {
		return nil, err
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
func stringPtr(s string) *string { return &s }
