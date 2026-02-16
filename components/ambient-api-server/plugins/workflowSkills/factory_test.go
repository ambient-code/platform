package workflowSkills_test

import (
	"context"
	"fmt"

	"github.com/ambient/platform/components/ambient-api-server/plugins/agents"
	"github.com/ambient/platform/components/ambient-api-server/plugins/skills"
	"github.com/ambient/platform/components/ambient-api-server/plugins/workflowSkills"
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

func newParentSkill() (*skills.Skill, error) {
	skillService := skills.Service(&environments.Environment().Services)
	result, svcErr := skillService.Create(context.Background(), &skills.Skill{
		Name:    "test-skill",
		RepoUrl: stringPtr("test-repo_url"),
		Prompt:  stringPtr("test-prompt"),
	})
	if svcErr != nil {
		return nil, fmt.Errorf("skills.Create: %s", svcErr.Error())
	}
	return result, nil
}

func newWorkflowSkill(id string) (*workflowSkills.WorkflowSkill, error) {
	workflowSkillService := workflowSkills.Service(&environments.Environment().Services)

	wf, err := newParentWorkflow()
	if err != nil {
		return nil, err
	}
	sk, err := newParentSkill()
	if err != nil {
		return nil, err
	}

	result, svcErr := workflowSkillService.Create(context.Background(), &workflowSkills.WorkflowSkill{
		WorkflowId: wf.ID,
		SkillId:    sk.ID,
		Position:   42,
	})
	if svcErr != nil {
		return nil, fmt.Errorf("workflowSkills.Create: %s", svcErr.Error())
	}
	return result, nil
}

func newWorkflowSkillList(namePrefix string, count int) ([]*workflowSkills.WorkflowSkill, error) {
	var items []*workflowSkills.WorkflowSkill
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newWorkflowSkill(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}
