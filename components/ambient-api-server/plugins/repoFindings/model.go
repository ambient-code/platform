package repoFindings

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type RepoFinding struct {
	api.Meta

	// Parent
	IntelligenceID string `json:"intelligence_id" gorm:"not null;index"`

	// Scope
	FilePath string `json:"file_path" gorm:"not null;index"`
	Category string `json:"category"  gorm:"not null;index"`
	Status   string `json:"status"    gorm:"not null;default:'active';index"`

	// Content
	Title      string   `json:"title"      gorm:"not null"`
	Body       string   `json:"body"       gorm:"type:text;not null"`
	Severity   *string  `json:"severity,omitempty"`
	Confidence *float64 `json:"confidence,omitempty"`

	// Provenance
	SourceType string  `json:"source_type" gorm:"not null"`
	SourceRef  *string `json:"source_ref,omitempty"`
	SessionID  *string `json:"session_id,omitempty"  gorm:"index"`
	AgentID    *string `json:"agent_id,omitempty"`

	// Resolution
	ResolvedBy     *string `json:"resolved_by,omitempty"`
	ResolvedReason *string `json:"resolved_reason,omitempty"`
}

type RepoFindingList []*RepoFinding
type RepoFindingIndex map[string]*RepoFinding

func (l RepoFindingList) Index() RepoFindingIndex {
	index := RepoFindingIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *RepoFinding) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	if d.Status == "" {
		d.Status = "active"
	}
	return nil
}

type RepoFindingPatchRequest struct {
	Status         *string `json:"status,omitempty"`
	Severity       *string `json:"severity,omitempty"`
	ResolvedBy     *string `json:"resolved_by,omitempty"`
	ResolvedReason *string `json:"resolved_reason,omitempty"`
	Title          *string `json:"title,omitempty"`
	Body           *string `json:"body,omitempty"`
}
