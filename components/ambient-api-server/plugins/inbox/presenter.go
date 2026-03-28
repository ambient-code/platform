package inbox

import (
	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertInboxMessage(inboxMessage openapi.InboxMessage) *InboxMessage {
	c := &InboxMessage{
		Meta: api.Meta{
			ID: util.NilToEmptyString(inboxMessage.Id),
		},
	}
	c.AgentId = inboxMessage.AgentId
	c.FromAgentId = inboxMessage.FromAgentId
	c.FromName = inboxMessage.FromName
	c.Body = inboxMessage.Body
	c.Read = inboxMessage.Read

	if inboxMessage.CreatedAt != nil {
		c.CreatedAt = *inboxMessage.CreatedAt
	}
	if inboxMessage.UpdatedAt != nil {
		c.UpdatedAt = *inboxMessage.UpdatedAt
	}

	return c
}

func PresentInboxMessage(inboxMessage *InboxMessage) openapi.InboxMessage {
	reference := presenters.PresentReference(inboxMessage.ID, inboxMessage)
	return openapi.InboxMessage{
		Id:          reference.Id,
		Kind:        reference.Kind,
		Href:        reference.Href,
		CreatedAt:   openapi.PtrTime(inboxMessage.CreatedAt),
		UpdatedAt:   openapi.PtrTime(inboxMessage.UpdatedAt),
		AgentId:     inboxMessage.AgentId,
		FromAgentId: inboxMessage.FromAgentId,
		FromName:    inboxMessage.FromName,
		Body:        inboxMessage.Body,
		Read:        inboxMessage.Read,
	}
}
