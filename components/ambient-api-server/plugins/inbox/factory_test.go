package inbox_test

import (
	"context"
	"fmt"

	"github.com/ambient-code/platform/components/ambient-api-server/plugins/inbox"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newInboxMessage(agentID string) (*inbox.InboxMessage, error) { //nolint:unused
	inboxMessageService := inbox.Service(&environments.Environment().Services)

	inboxMessage := &inbox.InboxMessage{
		AgentId: agentID,
		Body:    "test-body",
	}

	sub, err := inboxMessageService.Create(context.Background(), inboxMessage)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func newInboxMessageList(namePrefix string, count int) ([]*inbox.InboxMessage, error) { //nolint:unused
	var items []*inbox.InboxMessage
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newInboxMessage(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}
