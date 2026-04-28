package scheduledSessions

import (
	"time"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type ScheduledSession struct {
	api.Meta
	Name          string     `json:"name"`
	Description   *string    `json:"description,omitempty"`
	ProjectId     string     `json:"project_id"`
	AgentId       string     `json:"agent_id"`
	Schedule      string     `json:"schedule"`
	Timezone      string     `json:"timezone"`
	Enabled       bool       `json:"enabled"`
	SessionPrompt *string    `json:"session_prompt,omitempty"`
	LastRunAt     *time.Time `json:"last_run_at,omitempty"`
	NextRunAt     *time.Time `json:"next_run_at,omitempty"`
}

type ScheduledSessionList []*ScheduledSession

func (l ScheduledSessionList) Index() map[string]*ScheduledSession {
	idx := make(map[string]*ScheduledSession, len(l))
	for _, s := range l {
		idx[s.ID] = s
	}
	return idx
}

func (s *ScheduledSession) BeforeCreate(tx *gorm.DB) error {
	s.ID = api.NewID()
	if s.Timezone == "" {
		s.Timezone = "UTC"
	}
	return nil
}
