package workflowSkills

import (
	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertWorkflowSkill(workflowSkill openapi.WorkflowSkill) *WorkflowSkill {
	c := &WorkflowSkill{
		Meta: api.Meta{
			ID: util.NilToEmptyString(workflowSkill.Id),
		},
	}
	c.WorkflowId = workflowSkill.WorkflowId
	c.SkillId = workflowSkill.SkillId
	c.Position = int(workflowSkill.Position)

	if workflowSkill.CreatedAt != nil {
		c.CreatedAt = *workflowSkill.CreatedAt
		c.UpdatedAt = *workflowSkill.UpdatedAt
	}

	return c
}

func PresentWorkflowSkill(workflowSkill *WorkflowSkill) openapi.WorkflowSkill {
	reference := presenters.PresentReference(workflowSkill.ID, workflowSkill)
	return openapi.WorkflowSkill{
		Id:         reference.Id,
		Kind:       reference.Kind,
		Href:       reference.Href,
		CreatedAt:  openapi.PtrTime(workflowSkill.CreatedAt),
		UpdatedAt:  openapi.PtrTime(workflowSkill.UpdatedAt),
		WorkflowId: workflowSkill.WorkflowId,
		SkillId:    workflowSkill.SkillId,
		Position:   int32(workflowSkill.Position),
	}
}
