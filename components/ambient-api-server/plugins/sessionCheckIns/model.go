package sessionCheckIns

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type SessionCheckIn struct {
	api.Meta
	SessionId string   `json:"session_id" gorm:"not null;index"`
	AgentId   string   `json:"agent_id"   gorm:"not null;index"`
	Summary   *string  `json:"summary"`
	Branch    *string  `json:"branch"`
	Worktree  *string  `json:"worktree"`
	Pr        *string  `json:"pr"`
	Phase     *string  `json:"phase"`
	TestCount *int     `json:"test_count"`
	NextSteps *string  `json:"next_steps"`
	Items     []string `json:"items"      gorm:"type:text;serializer:json"`
	Questions []string `json:"questions"  gorm:"type:text;serializer:json"`
	Blockers  []string `json:"blockers"   gorm:"type:text;serializer:json"`
}

type SessionCheckInList []*SessionCheckIn
type SessionCheckInIndex map[string]*SessionCheckIn

func (l SessionCheckInList) Index() SessionCheckInIndex {
	index := SessionCheckInIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *SessionCheckIn) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	return nil
}

type SessionCheckInPatchRequest struct {
	Summary   *string  `json:"summary,omitempty"`
	Branch    *string  `json:"branch,omitempty"`
	Worktree  *string  `json:"worktree,omitempty"`
	Pr        *string  `json:"pr,omitempty"`
	Phase     *string  `json:"phase,omitempty"`
	TestCount *int     `json:"test_count,omitempty"`
	NextSteps *string  `json:"next_steps,omitempty"`
	Items     []string `json:"items,omitempty"`
	Questions []string `json:"questions,omitempty"`
	Blockers  []string `json:"blockers,omitempty"`
}
