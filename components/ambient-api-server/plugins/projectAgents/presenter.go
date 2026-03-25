package projectAgents

import (
	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertProjectAgent(projectAgent openapi.ProjectAgent) *ProjectAgent {
	c := &ProjectAgent{
		Meta: api.Meta{
			ID: util.NilToEmptyString(projectAgent.Id),
		},
	}
	c.ProjectId = projectAgent.ProjectId
	c.AgentId = projectAgent.AgentId
	if projectAgent.AgentVersion != nil {
		c.AgentVersion = openapi.PtrInt(int(*projectAgent.AgentVersion))
	}
	c.CurrentSessionId = projectAgent.CurrentSessionId

	if projectAgent.CreatedAt != nil {
		c.CreatedAt = *projectAgent.CreatedAt
		c.UpdatedAt = *projectAgent.UpdatedAt
	}

	return c
}

func PresentProjectAgent(projectAgent *ProjectAgent) openapi.ProjectAgent {
	reference := presenters.PresentReference(projectAgent.ID, projectAgent)
	return openapi.ProjectAgent{
		Id:        reference.Id,
		Kind:      reference.Kind,
		Href:      reference.Href,
		CreatedAt: openapi.PtrTime(projectAgent.CreatedAt),
		UpdatedAt: openapi.PtrTime(projectAgent.UpdatedAt),
		ProjectId: projectAgent.ProjectId,
		AgentId:   projectAgent.AgentId,
		AgentVersion: func() *int32 {
			if projectAgent.AgentVersion != nil {
				return openapi.PtrInt32(int32(*projectAgent.AgentVersion))
			}
			return nil
		}(),
		CurrentSessionId: projectAgent.CurrentSessionId,
	}
}
