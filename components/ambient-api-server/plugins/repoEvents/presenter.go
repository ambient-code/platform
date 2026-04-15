package repoEvents

import (
	"time"

	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
)

type RepoEventAPI struct {
	ID        *string    `json:"id,omitempty"`
	Kind      *string    `json:"kind,omitempty"`
	Href      *string    `json:"href,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`

	ResourceType string  `json:"resource_type"`
	ResourceID   string  `json:"resource_id"`
	Action       string  `json:"action"`
	ActorType    string  `json:"actor_type"`
	ActorID      string  `json:"actor_id"`
	ProjectID    string  `json:"project_id"`
	Reason       *string `json:"reason,omitempty"`
	Diff         *string `json:"diff,omitempty"`
}

type RepoEventListAPI struct {
	Kind  string         `json:"kind"`
	Page  int32          `json:"page"`
	Size  int32          `json:"size"`
	Total int32          `json:"total"`
	Items []RepoEventAPI `json:"items"`
}

func ptrTime(v time.Time) *time.Time { return &v }

func PresentRepoEvent(re *RepoEvent) RepoEventAPI {
	ref := presenters.PresentReference(re.ID, re)
	return RepoEventAPI{
		ID:        ref.Id,
		Kind:      ref.Kind,
		Href:      ref.Href,
		CreatedAt: ptrTime(re.CreatedAt),
		UpdatedAt: ptrTime(re.UpdatedAt),

		ResourceType: re.ResourceType,
		ResourceID:   re.ResourceID,
		Action:       re.Action,
		ActorType:    re.ActorType,
		ActorID:      re.ActorID,
		ProjectID:    re.ProjectID,
		Reason:       re.Reason,
		Diff:         re.Diff,
	}
}
