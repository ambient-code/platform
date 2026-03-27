package agents

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type Agent struct {
	api.Meta
	ProjectId        string  `json:"project_id"          gorm:"not null;index"`
	Name             string  `json:"name"                gorm:"not null"`
	Prompt           *string `json:"prompt"              gorm:"type:text"`
	CurrentSessionId *string `json:"current_session_id"`
	Labels           *string `json:"labels"`
	Annotations      *string `json:"annotations"`
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
