package agentMessages

import (
	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertAgentMessage(agentMessage openapi.AgentMessage) *AgentMessage {
	c := &AgentMessage{
		Meta: api.Meta{
			ID: util.NilToEmptyString(agentMessage.Id),
		},
	}
	c.RecipientAgentId = agentMessage.RecipientAgentId
	c.SenderAgentId = agentMessage.SenderAgentId
	c.SenderUserId = agentMessage.SenderUserId
	c.SenderName = agentMessage.SenderName
	c.Body = agentMessage.Body
	c.Read = agentMessage.Read

	if agentMessage.CreatedAt != nil {
		c.CreatedAt = *agentMessage.CreatedAt
		c.UpdatedAt = *agentMessage.UpdatedAt
	}

	return c
}

func PresentAgentMessage(agentMessage *AgentMessage) openapi.AgentMessage {
	reference := presenters.PresentReference(agentMessage.ID, agentMessage)
	return openapi.AgentMessage{
		Id:               reference.Id,
		Kind:             reference.Kind,
		Href:             reference.Href,
		CreatedAt:        openapi.PtrTime(agentMessage.CreatedAt),
		UpdatedAt:        openapi.PtrTime(agentMessage.UpdatedAt),
		RecipientAgentId: agentMessage.RecipientAgentId,
		SenderAgentId:    agentMessage.SenderAgentId,
		SenderUserId:     agentMessage.SenderUserId,
		SenderName:       agentMessage.SenderName,
		Body:             agentMessage.Body,
		Read:             agentMessage.Read,
	}
}
