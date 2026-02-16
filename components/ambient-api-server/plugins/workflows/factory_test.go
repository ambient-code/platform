package workflows_test

import (
	"context"
	"fmt"

	"github.com/ambient/platform/components/ambient-api-server/plugins/agents"
	"github.com/ambient/platform/components/ambient-api-server/plugins/workflows"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newAgent() (*agents.Agent, error) {
	agentService := agents.Service(&environments.Environment().Services)
	result, svcErr := agentService.Create(context.Background(), &agents.Agent{
		Name:    "test-agent",
		RepoUrl: stringPtr("test-repo_url"),
		Prompt:  stringPtr("test-prompt"),
	})
	if svcErr != nil {
		return nil, fmt.Errorf("agents.Create: %s", svcErr.Error())
	}
	return result, nil
}

func newWorkflow(id string) (*workflows.Workflow, error) {
	workflowService := workflows.Service(&environments.Environment().Services)

	agent, err := newAgent()
	if err != nil {
		return nil, err
	}

	wf, svcErr := workflowService.Create(context.Background(), &workflows.Workflow{
		Name:    "test-name",
		RepoUrl: stringPtr("test-repo_url"),
		Prompt:  stringPtr("test-prompt"),
		AgentId: stringPtr(agent.ID),
	})
	if svcErr != nil {
		return nil, fmt.Errorf("workflows.Create: %s", svcErr.Error())
	}
	return wf, nil
}

func newWorkflowList(namePrefix string, count int) ([]*workflows.Workflow, error) {
	var items []*workflows.Workflow
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newWorkflow(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}
func stringPtr(s string) *string { return &s }
