package workflowTasks_test

import (
	"context"
	"fmt"

	"github.com/ambient/platform/components/ambient-api-server/plugins/agents"
	"github.com/ambient/platform/components/ambient-api-server/plugins/tasks"
	"github.com/ambient/platform/components/ambient-api-server/plugins/workflowTasks"
	"github.com/ambient/platform/components/ambient-api-server/plugins/workflows"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func stringPtr(s string) *string { return &s }

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

func newParentTask() (*tasks.Task, error) {
	taskService := tasks.Service(&environments.Environment().Services)
	result, svcErr := taskService.Create(context.Background(), &tasks.Task{
		Name:    "test-task",
		RepoUrl: stringPtr("test-repo_url"),
		Prompt:  stringPtr("test-prompt"),
	})
	if svcErr != nil {
		return nil, fmt.Errorf("tasks.Create: %s", svcErr.Error())
	}
	return result, nil
}

func newWorkflowTask(id string) (*workflowTasks.WorkflowTask, error) {
	workflowTaskService := workflowTasks.Service(&environments.Environment().Services)

	wf, err := newParentWorkflow()
	if err != nil {
		return nil, err
	}
	tk, err := newParentTask()
	if err != nil {
		return nil, err
	}

	result, svcErr := workflowTaskService.Create(context.Background(), &workflowTasks.WorkflowTask{
		WorkflowId: wf.ID,
		TaskId:     tk.ID,
		Position:   42,
	})
	if svcErr != nil {
		return nil, fmt.Errorf("workflowTasks.Create: %s", svcErr.Error())
	}
	return result, nil
}

func newWorkflowTaskList(namePrefix string, count int) ([]*workflowTasks.WorkflowTask, error) {
	var items []*workflowTasks.WorkflowTask
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newWorkflowTask(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}
