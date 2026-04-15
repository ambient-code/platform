package repoIntelligences

import (
	"time"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type RepoIntelligence struct {
	api.Meta

	// Scoping
	ProjectID  string `json:"project_id"  gorm:"not null;index"`
	RepoURL    string `json:"repo_url"    gorm:"not null;index"`
	RepoBranch string `json:"repo_branch" gorm:"not null;default:'main'"`

	// Content
	Summary      string  `json:"summary"       gorm:"type:text;not null"`
	Language     string  `json:"language"      gorm:"not null"`
	Framework    *string `json:"framework,omitempty"`
	BuildSystem  *string `json:"build_system,omitempty"`
	TestStrategy *string `json:"test_strategy,omitempty" gorm:"type:text"`
	Architecture *string `json:"architecture,omitempty"  gorm:"type:text"`
	Conventions  *string `json:"conventions,omitempty"   gorm:"type:text"`
	Dependencies *string `json:"dependencies,omitempty"  gorm:"type:text"`
	Caveats      *string `json:"caveats,omitempty"       gorm:"type:text"`

	// Metadata
	AnalyzedBySessionID *string    `json:"analyzed_by_session_id,omitempty" gorm:"index"`
	AnalyzedByAgentID   *string    `json:"analyzed_by_agent_id,omitempty"`
	AnalyzedAt          *time.Time `json:"analyzed_at,omitempty"`
	Confidence          *float64   `json:"confidence,omitempty"`
	Version             int        `json:"version" gorm:"not null;default:1"`
}

type RepoIntelligenceList []*RepoIntelligence
type RepoIntelligenceIndex map[string]*RepoIntelligence

func (l RepoIntelligenceList) Index() RepoIntelligenceIndex {
	index := RepoIntelligenceIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *RepoIntelligence) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	if d.Version == 0 {
		d.Version = 1
	}
	return nil
}

type RepoIntelligencePatchRequest struct {
	Summary      *string  `json:"summary,omitempty"`
	Language     *string  `json:"language,omitempty"`
	Framework    *string  `json:"framework,omitempty"`
	BuildSystem  *string  `json:"build_system,omitempty"`
	TestStrategy *string  `json:"test_strategy,omitempty"`
	Architecture *string  `json:"architecture,omitempty"`
	Conventions  *string  `json:"conventions,omitempty"`
	Dependencies *string  `json:"dependencies,omitempty"`
	Caveats      *string  `json:"caveats,omitempty"`
	Confidence   *float64 `json:"confidence,omitempty"`
	RepoBranch   *string  `json:"repo_branch,omitempty"`
}
