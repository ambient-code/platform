package inbox

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type InboxMessage struct {
	api.Meta
	AgentId     string  `json:"agent_id"`
	FromAgentId *string `json:"from_agent_id"`
	FromName    *string `json:"from_name"`
	Body        string  `json:"body"`
	Read        *bool   `json:"read"`
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
	Read *bool `json:"read,omitempty"`
}
