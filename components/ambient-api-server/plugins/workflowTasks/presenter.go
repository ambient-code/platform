package workflowTasks

import (
	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertWorkflowTask(workflowTask openapi.WorkflowTask) *WorkflowTask {
	c := &WorkflowTask{
		Meta: api.Meta{
			ID: util.NilToEmptyString(workflowTask.Id),
		},
	}
	c.WorkflowId = workflowTask.WorkflowId
	c.TaskId = workflowTask.TaskId
	c.Position = int(workflowTask.Position)

	if workflowTask.CreatedAt != nil {
		c.CreatedAt = *workflowTask.CreatedAt
		c.UpdatedAt = *workflowTask.UpdatedAt
	}

	return c
}

func PresentWorkflowTask(workflowTask *WorkflowTask) openapi.WorkflowTask {
	reference := presenters.PresentReference(workflowTask.ID, workflowTask)
	return openapi.WorkflowTask{
		Id:         reference.Id,
		Kind:       reference.Kind,
		Href:       reference.Href,
		CreatedAt:  openapi.PtrTime(workflowTask.CreatedAt),
		UpdatedAt:  openapi.PtrTime(workflowTask.UpdatedAt),
		WorkflowId: workflowTask.WorkflowId,
		TaskId:     workflowTask.TaskId,
		Position:   int32(workflowTask.Position),
	}
}
