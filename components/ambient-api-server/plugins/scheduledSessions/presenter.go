package scheduledSessions

import (
	"fmt"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
)

const basePath = "/api/ambient/v1/projects/%s/scheduled-sessions/%s"

func PresentScheduledSession(ss *ScheduledSession) openapi.ScheduledSession {
	kind := "ScheduledSession"
	href := fmt.Sprintf(basePath, ss.ProjectId, ss.ID)
	enabled := ss.Enabled
	return openapi.ScheduledSession{
		Id:            &ss.ID,
		Kind:          &kind,
		Href:          &href,
		CreatedAt:     &ss.CreatedAt,
		UpdatedAt:     &ss.UpdatedAt,
		Name:          ss.Name,
		Description:   ss.Description,
		ProjectId:     ss.ProjectId,
		AgentId:       ss.AgentId,
		Schedule:      ss.Schedule,
		Timezone:      &ss.Timezone,
		Enabled:       &enabled,
		SessionPrompt: ss.SessionPrompt,
		LastRunAt:     ss.LastRunAt,
		NextRunAt:     ss.NextRunAt,
	}
}

func ConvertScheduledSession(in openapi.ScheduledSession) *ScheduledSession {
	ss := &ScheduledSession{
		Name:          in.Name,
		ProjectId:     in.ProjectId,
		AgentId:       in.AgentId,
		Schedule:      in.Schedule,
		SessionPrompt: in.SessionPrompt,
		Description:   in.Description,
	}
	if in.Timezone != nil {
		ss.Timezone = *in.Timezone
	}
	if in.Enabled != nil {
		ss.Enabled = *in.Enabled
	}
	return ss
}
