package projectAgents

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type ProjectAgent struct {
	api.Meta
	ProjectId        string  `json:"project_id"`
	AgentId          string  `json:"agent_id"`
	AgentVersion     *int    `json:"agent_version"`
	CurrentSessionId *string `json:"current_session_id"`
}

type ProjectAgentList []*ProjectAgent
type ProjectAgentIndex map[string]*ProjectAgent

func (l ProjectAgentList) Index() ProjectAgentIndex {
	index := ProjectAgentIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *ProjectAgent) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	return nil
}

type ProjectAgentPatchRequest struct {
	ProjectId        *string `json:"project_id,omitempty"`
	AgentId          *string `json:"agent_id,omitempty"`
	AgentVersion     *int    `json:"agent_version,omitempty"`
	CurrentSessionId *string `json:"current_session_id,omitempty"`
}
