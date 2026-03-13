package sessionCheckIns

import (
	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertSessionCheckIn(sessionCheckIn openapi.SessionCheckIn) *SessionCheckIn {
	c := &SessionCheckIn{
		Meta: api.Meta{
			ID: util.NilToEmptyString(sessionCheckIn.Id),
		},
	}
	c.SessionId = sessionCheckIn.SessionId
	c.AgentId = sessionCheckIn.AgentId
	c.Summary = sessionCheckIn.Summary
	c.Branch = sessionCheckIn.Branch
	c.Worktree = sessionCheckIn.Worktree
	c.Pr = sessionCheckIn.Pr
	c.Phase = sessionCheckIn.Phase
	if sessionCheckIn.TestCount != nil {
		c.TestCount = openapi.PtrInt(int(*sessionCheckIn.TestCount))
	}
	c.NextSteps = sessionCheckIn.NextSteps

	if sessionCheckIn.CreatedAt != nil {
		c.CreatedAt = *sessionCheckIn.CreatedAt
		c.UpdatedAt = *sessionCheckIn.UpdatedAt
	}

	return c
}

func PresentSessionCheckIn(sessionCheckIn *SessionCheckIn) openapi.SessionCheckIn {
	reference := presenters.PresentReference(sessionCheckIn.ID, sessionCheckIn)
	return openapi.SessionCheckIn{
		Id:        reference.Id,
		Kind:      reference.Kind,
		Href:      reference.Href,
		CreatedAt: openapi.PtrTime(sessionCheckIn.CreatedAt),
		UpdatedAt: openapi.PtrTime(sessionCheckIn.UpdatedAt),
		SessionId: sessionCheckIn.SessionId,
		AgentId:   sessionCheckIn.AgentId,
		Summary:   sessionCheckIn.Summary,
		Branch:    sessionCheckIn.Branch,
		Worktree:  sessionCheckIn.Worktree,
		Pr:        sessionCheckIn.Pr,
		Phase:     sessionCheckIn.Phase,
		TestCount: func() *int32 {
			if sessionCheckIn.TestCount != nil {
				return openapi.PtrInt32(int32(*sessionCheckIn.TestCount))
			}
			return nil
		}(),
		NextSteps: sessionCheckIn.NextSteps,
	}
}
