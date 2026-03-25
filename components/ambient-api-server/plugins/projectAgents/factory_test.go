package projectAgents_test

import (
	"context"
	"fmt"

	"github.com/ambient-code/platform/components/ambient-api-server/plugins/projectAgents"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newProjectAgent(id string) (*projectAgents.ProjectAgent, error) {
	projectAgentService := projectAgents.Service(&environments.Environment().Services)

	projectAgent := &projectAgents.ProjectAgent{
		ProjectId:        "test-project_id",
		AgentId:          "test-agent_id",
		AgentVersion:     intPtr(42),
		CurrentSessionId: stringPtr("test-current_session_id"),
	}

	sub, err := projectAgentService.Create(context.Background(), projectAgent)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func newProjectAgentList(namePrefix string, count int) ([]*projectAgents.ProjectAgent, error) {
	var items []*projectAgents.ProjectAgent
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newProjectAgent(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}
func stringPtr(s string) *string { return &s }
func intPtr(i int) *int          { return &i }
