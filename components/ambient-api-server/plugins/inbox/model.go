package inbox

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type InboxMessage struct {
	api.Meta
	ProjectAgentId     string  `json:"project_agent_id"`
	FromProjectAgentId *string `json:"from_project_agent_id"`
	FromName           *string `json:"from_name"`
	Body               string  `json:"body"`
	Read               *bool   `json:"read"`
}

type InboxMessageList []*InboxMessage
type InboxMessageIndex map[string]*InboxMessage

func (l InboxMessageList) Index() InboxMessageIndex {
	index := InboxMessageIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *InboxMessage) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	return nil
}

type InboxMessagePatchRequest struct {
	ProjectAgentId     *string `json:"project_agent_id,omitempty"`
	FromProjectAgentId *string `json:"from_project_agent_id,omitempty"`
	FromName           *string `json:"from_name,omitempty"`
	Body               *string `json:"body,omitempty"`
	Read               *bool   `json:"read,omitempty"`
}
