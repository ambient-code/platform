package repoIntelligences

import (
	"time"

	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
)

// RepoIntelligenceAPI is the API representation returned to clients.
// Defined here rather than in the generated openapi package so the plugin
// is self-contained and does not require running `make generate`.
type RepoIntelligenceAPI struct {
	ID        *string    `json:"id,omitempty"`
	Kind      *string    `json:"kind,omitempty"`
	Href      *string    `json:"href,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`

	ProjectID  string `json:"project_id"`
	RepoURL    string `json:"repo_url"`
	RepoBranch string `json:"repo_branch"`

	Summary      string  `json:"summary"`
	Language     string  `json:"language"`
	Framework    *string `json:"framework,omitempty"`
	BuildSystem  *string `json:"build_system,omitempty"`
	TestStrategy *string `json:"test_strategy,omitempty"`
	Architecture *string `json:"architecture,omitempty"`
	Conventions  *string `json:"conventions,omitempty"`
	Dependencies *string `json:"dependencies,omitempty"`
	Caveats      *string `json:"caveats,omitempty"`

	AnalyzedBySessionID *string    `json:"analyzed_by_session_id,omitempty"`
	AnalyzedByAgentID   *string    `json:"analyzed_by_agent_id,omitempty"`
	AnalyzedAt          *time.Time `json:"analyzed_at,omitempty"`
	Confidence          *float64   `json:"confidence,omitempty"`
	Version             int        `json:"version"`
}

type RepoIntelligenceListAPI struct {
	Kind  string                `json:"kind"`
	Page  int32                 `json:"page"`
	Size  int32                 `json:"size"`
	Total int32                 `json:"total"`
	Items []RepoIntelligenceAPI `json:"items"`
}

func ptrString(v string) *string   { return &v }
func ptrTime(v time.Time) *time.Time { return &v }

func ConvertRepoIntelligence(a RepoIntelligenceAPI) *RepoIntelligence {
	ri := &RepoIntelligence{}
	if a.ID != nil {
		ri.ID = *a.ID
	}
	ri.ProjectID = a.ProjectID
	ri.RepoURL = a.RepoURL
	ri.RepoBranch = a.RepoBranch
	ri.Summary = a.Summary
	ri.Language = a.Language
	ri.Framework = a.Framework
	ri.BuildSystem = a.BuildSystem
	ri.TestStrategy = a.TestStrategy
	ri.Architecture = a.Architecture
	ri.Conventions = a.Conventions
	ri.Dependencies = a.Dependencies
	ri.Caveats = a.Caveats
	ri.AnalyzedBySessionID = a.AnalyzedBySessionID
	ri.AnalyzedByAgentID = a.AnalyzedByAgentID
	ri.AnalyzedAt = a.AnalyzedAt
	ri.Confidence = a.Confidence
	if a.Version > 0 {
		ri.Version = a.Version
	}
	if a.CreatedAt != nil {
		ri.CreatedAt = *a.CreatedAt
	}
	if a.UpdatedAt != nil {
		ri.UpdatedAt = *a.UpdatedAt
	}
	return ri
}

func PresentRepoIntelligence(ri *RepoIntelligence) RepoIntelligenceAPI {
	ref := presenters.PresentReference(ri.ID, ri)
	return RepoIntelligenceAPI{
		ID:        ref.Id,
		Kind:      ref.Kind,
		Href:      ref.Href,
		CreatedAt: ptrTime(ri.CreatedAt),
		UpdatedAt: ptrTime(ri.UpdatedAt),

		ProjectID:  ri.ProjectID,
		RepoURL:    ri.RepoURL,
		RepoBranch: ri.RepoBranch,

		Summary:      ri.Summary,
		Language:     ri.Language,
		Framework:    ri.Framework,
		BuildSystem:  ri.BuildSystem,
		TestStrategy: ri.TestStrategy,
		Architecture: ri.Architecture,
		Conventions:  ri.Conventions,
		Dependencies: ri.Dependencies,
		Caveats:      ri.Caveats,

		AnalyzedBySessionID: ri.AnalyzedBySessionID,
		AnalyzedByAgentID:   ri.AnalyzedByAgentID,
		AnalyzedAt:          ri.AnalyzedAt,
		Confidence:          ri.Confidence,
		Version:             ri.Version,
	}
}
