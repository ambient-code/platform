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
	c.ProjectAgentId = inboxMessage.ProjectAgentId
	c.FromProjectAgentId = inboxMessage.FromProjectAgentId
	c.FromName = inboxMessage.FromName
	c.Body = inboxMessage.Body
	c.Read = inboxMessage.Read

	if inboxMessage.CreatedAt != nil {
		c.CreatedAt = *inboxMessage.CreatedAt
		c.UpdatedAt = *inboxMessage.UpdatedAt
	}

	return c
}

func PresentInboxMessage(inboxMessage *InboxMessage) openapi.InboxMessage {
	reference := presenters.PresentReference(inboxMessage.ID, inboxMessage)
	return openapi.InboxMessage{
		Id:                 reference.Id,
		Kind:               reference.Kind,
		Href:               reference.Href,
		CreatedAt:          openapi.PtrTime(inboxMessage.CreatedAt),
		UpdatedAt:          openapi.PtrTime(inboxMessage.UpdatedAt),
		ProjectAgentId:     inboxMessage.ProjectAgentId,
		FromProjectAgentId: inboxMessage.FromProjectAgentId,
		FromName:           inboxMessage.FromName,
		Body:               inboxMessage.Body,
		Read:               inboxMessage.Read,
	}
}
