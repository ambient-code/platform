package workflows

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type Workflow struct {
	api.Meta
	Name      string  `json:"name"`
	RepoUrl   *string `json:"repo_url"`
	Prompt    *string `json:"prompt"`
	AgentId   *string `json:"agent_id"`
	ProjectId *string `json:"project_id"`
	Branch    *string `json:"branch"`
	Path      *string `json:"path"`
}

type WorkflowList []*Workflow
type WorkflowIndex map[string]*Workflow

func (l WorkflowList) Index() WorkflowIndex {
	index := WorkflowIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *Workflow) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	return nil
}

type WorkflowPatchRequest struct {
	Name      *string `json:"name,omitempty"`
	RepoUrl   *string `json:"repo_url,omitempty"`
	Prompt    *string `json:"prompt,omitempty"`
	AgentId   *string `json:"agent_id,omitempty"`
	ProjectId *string `json:"project_id,omitempty"`
	Branch    *string `json:"branch,omitempty"`
	Path      *string `json:"path,omitempty"`
}
