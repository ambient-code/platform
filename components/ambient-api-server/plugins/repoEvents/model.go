package repoEvents

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type RepoEvent struct {
	api.Meta

	// What changed
	ResourceType string `json:"resource_type" gorm:"not null;index"`
	ResourceID   string `json:"resource_id"   gorm:"not null;index"`
	Action       string `json:"action"        gorm:"not null"`

	// Who
	ActorType string `json:"actor_type" gorm:"not null"`
	ActorID   string `json:"actor_id"   gorm:"not null"`

	// Context
	ProjectID string  `json:"project_id" gorm:"not null;index"`
	Reason    *string `json:"reason,omitempty"`
	Diff      *string `json:"diff,omitempty" gorm:"type:text"`
}

type RepoEventList []*RepoEvent
type RepoEventIndex map[string]*RepoEvent

func (l RepoEventList) Index() RepoEventIndex {
	index := RepoEventIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *RepoEvent) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	return nil
}
