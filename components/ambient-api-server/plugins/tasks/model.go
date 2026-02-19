package tasks

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type Task struct {
	api.Meta
	Name      string  `json:"name"`
	RepoUrl   *string `json:"repo_url"`
	Prompt    *string `json:"prompt"`
	ProjectId *string `json:"project_id"`
}

type TaskList []*Task
type TaskIndex map[string]*Task

func (l TaskList) Index() TaskIndex {
	index := TaskIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *Task) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	return nil
}

type TaskPatchRequest struct {
	Name      *string `json:"name,omitempty"`
	RepoUrl   *string `json:"repo_url,omitempty"`
	Prompt    *string `json:"prompt,omitempty"`
	ProjectId *string `json:"project_id,omitempty"`
}
