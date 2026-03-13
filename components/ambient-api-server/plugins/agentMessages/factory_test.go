package agentMessages_test

import (
	"context"
	"fmt"

	"github.com/ambient-code/platform/components/ambient-api-server/plugins/agentMessages"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newAgentMessage(id string) (*agentMessages.AgentMessage, error) {
	agentMessageService := agentMessages.Service(&environments.Environment().Services)

	agentMessage := &agentMessages.AgentMessage{
		RecipientAgentId: "test-recipient_agent_id",
		SenderAgentId:    stringPtr("test-sender_agent_id"),
		SenderUserId:     stringPtr("test-sender_user_id"),
		SenderName:       stringPtr("test-sender_name"),
		Body:             stringPtr("test-body"),
		Read:             boolPtr(true),
	}

	sub, err := agentMessageService.Create(context.Background(), agentMessage)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func newAgentMessageList(namePrefix string, count int) ([]*agentMessages.AgentMessage, error) {
	var items []*agentMessages.AgentMessage
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newAgentMessage(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}
func stringPtr(s string) *string { return &s }
func boolPtr(b bool) *bool       { return &b }
