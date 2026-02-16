package sessions_test

import (
	"context"
	"fmt"

	"github.com/ambient/platform/components/ambient-api-server/plugins/agents"
	"github.com/ambient/platform/components/ambient-api-server/plugins/sessions"
	"github.com/ambient/platform/components/ambient-api-server/plugins/users"
	"github.com/ambient/platform/components/ambient-api-server/plugins/workflows"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newUser(name string) (*users.User, error) {
	userService := users.Service(&environments.Environment().Services)
	user := &users.User{
		Username: name,
		Name:     name,
	}
	result, svcErr := userService.Create(context.Background(), user)
	if svcErr != nil {
		return nil, fmt.Errorf("users.Create: %s", svcErr.Error())
	}
	return result, nil
}

func newParentWorkflow() (*workflows.Workflow, error) {
	agentService := agents.Service(&environments.Environment().Services)
	agent, agentErr := agentService.Create(context.Background(), &agents.Agent{
		Name:    "test-agent",
		RepoUrl: stringPtr("test-repo_url"),
		Prompt:  stringPtr("test-prompt"),
	})
	if agentErr != nil {
		return nil, fmt.Errorf("agents.Create: %s", agentErr.Error())
	}

	workflowService := workflows.Service(&environments.Environment().Services)
	wf, wfErr := workflowService.Create(context.Background(), &workflows.Workflow{
		Name:    "test-workflow",
		RepoUrl: stringPtr("test-repo_url"),
		Prompt:  stringPtr("test-prompt"),
		AgentId: stringPtr(agent.ID),
	})
	if wfErr != nil {
		return nil, fmt.Errorf("workflows.Create: %s", wfErr.Error())
	}
	return wf, nil
}

func newSession(id string) (*sessions.Session, error) {
	sessionService := sessions.Service(&environments.Environment().Services)

	creator, err := newUser("test-creator")
	if err != nil {
		return nil, fmt.Errorf("newUser(creator): %w", err)
	}
	assignee, err := newUser("test-assignee")
	if err != nil {
		return nil, fmt.Errorf("newUser(assignee): %w", err)
	}
	wf, err := newParentWorkflow()
	if err != nil {
		return nil, fmt.Errorf("newParentWorkflow: %w", err)
	}

	session := &sessions.Session{
		Name:            "test-name",
		RepoUrl:         stringPtr("test-repo_url"),
		Prompt:          stringPtr("test-prompt"),
		CreatedByUserId: stringPtr(creator.ID),
		AssignedUserId:  stringPtr(assignee.ID),
		WorkflowId:      stringPtr(wf.ID),
	}

	sub, svcErr := sessionService.Create(context.Background(), session)
	if svcErr != nil {
		return nil, fmt.Errorf("sessionService.Create: %s", svcErr.Error())
	}

	return sub, nil
}

func newSessionList(namePrefix string, count int) ([]*sessions.Session, error) {
	var items []*sessions.Session
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newSession(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}
func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool       { return &b }
func int32Ptr(i int32) *int32    { return &i }
func float64Ptr(f float64) *float64 { return &f }
