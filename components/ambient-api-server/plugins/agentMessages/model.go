package agentMessages

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type AgentMessage struct {
	api.Meta
	RecipientAgentId string  `json:"recipient_agent_id" gorm:"not null;index"`
	SenderAgentId    *string `json:"sender_agent_id"`
	SenderUserId     *string `json:"sender_user_id"`
	SenderName       *string `json:"sender_name"`
	Body             *string `json:"body"               gorm:"type:text"`
	Read             *bool   `json:"read"               gorm:"default:false"`
}

type AgentMessageList []*AgentMessage
type AgentMessageIndex map[string]*AgentMessage

func (l AgentMessageList) Index() AgentMessageIndex {
	index := AgentMessageIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *AgentMessage) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	return nil
}

type AgentMessagePatchRequest struct {
	Read *bool `json:"read,omitempty"`
}
