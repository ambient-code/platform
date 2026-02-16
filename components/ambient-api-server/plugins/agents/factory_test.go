package agents_test

import (
	"context"
	"fmt"

	"github.com/ambient/platform/components/ambient-api-server/plugins/agents"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newAgent(id string) (*agents.Agent, error) {
	agentService := agents.Service(&environments.Environment().Services)

	agent := &agents.Agent{
		Name:    "test-name",
		RepoUrl: stringPtr("test-repo_url"),
		Prompt:  stringPtr("test-prompt"),
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
