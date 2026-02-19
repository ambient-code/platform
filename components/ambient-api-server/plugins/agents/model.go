package agents

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type Agent struct {
	api.Meta
	Name      string  `json:"name"`
	RepoUrl   *string `json:"repo_url"`
	Prompt    *string `json:"prompt"`
	ProjectId *string `json:"project_id"`
}

type AgentList []*Agent
type AgentIndex map[string]*Agent

func (l AgentList) Index() AgentIndex {
	index := AgentIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *Agent) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	return nil
}

type AgentPatchRequest struct {
	Name      *string `json:"name,omitempty"`
	RepoUrl   *string `json:"repo_url,omitempty"`
	Prompt    *string `json:"prompt,omitempty"`
	ProjectId *string `json:"project_id,omitempty"`
}
