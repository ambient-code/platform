package tasks

import (
	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertTask(task openapi.Task) *Task {
	c := &Task{
		Meta: api.Meta{
			ID: util.NilToEmptyString(task.Id),
		},
	}
	c.Name = task.Name
	c.RepoUrl = task.RepoUrl
	c.Prompt = task.Prompt
	c.ProjectId = task.ProjectId

	if task.CreatedAt != nil {
		c.CreatedAt = *task.CreatedAt
		c.UpdatedAt = *task.UpdatedAt
	}

	return c
}

func PresentTask(task *Task) openapi.Task {
	reference := presenters.PresentReference(task.ID, task)
	return openapi.Task{
		Id:        reference.Id,
		Kind:      reference.Kind,
		Href:      reference.Href,
		CreatedAt: openapi.PtrTime(task.CreatedAt),
		UpdatedAt: openapi.PtrTime(task.UpdatedAt),
		Name:      task.Name,
		RepoUrl:   task.RepoUrl,
		Prompt:    task.Prompt,
		ProjectId: task.ProjectId,
	}
}
