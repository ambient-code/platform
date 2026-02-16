package workflowTasks

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type WorkflowTask struct {
	api.Meta
	WorkflowId string `json:"workflow_id"`
	TaskId     string `json:"task_id"`
	Position   int    `json:"position"`
}

type WorkflowTaskList []*WorkflowTask
type WorkflowTaskIndex map[string]*WorkflowTask

func (l WorkflowTaskList) Index() WorkflowTaskIndex {
	index := WorkflowTaskIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *WorkflowTask) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	return nil
}

type WorkflowTaskPatchRequest struct {
	WorkflowId *string `json:"workflow_id,omitempty"`
	TaskId     *string `json:"task_id,omitempty"`
	Position   *int    `json:"position,omitempty"`
}
