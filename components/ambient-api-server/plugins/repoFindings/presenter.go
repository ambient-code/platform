package repoFindings

import (
	"time"

	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
)

type RepoFindingAPI struct {
	ID        *string    `json:"id,omitempty"`
	Kind      *string    `json:"kind,omitempty"`
	Href      *string    `json:"href,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`

	IntelligenceID string   `json:"intelligence_id"`
	FilePath       string   `json:"file_path"`
	Category       string   `json:"category"`
	Status         string   `json:"status"`
	Title          string   `json:"title"`
	Body           string   `json:"body"`
	Severity       *string  `json:"severity,omitempty"`
	Confidence     *float64 `json:"confidence,omitempty"`
	SourceType     string   `json:"source_type"`
	SourceRef      *string  `json:"source_ref,omitempty"`
	SessionID      *string  `json:"session_id,omitempty"`
	AgentID        *string  `json:"agent_id,omitempty"`
	ResolvedBy     *string  `json:"resolved_by,omitempty"`
	ResolvedReason *string  `json:"resolved_reason,omitempty"`
}

type RepoFindingListAPI struct {
	Kind  string           `json:"kind"`
	Page  int32            `json:"page"`
	Size  int32            `json:"size"`
	Total int32            `json:"total"`
	Items []RepoFindingAPI `json:"items"`
}

func ptrTime(v time.Time) *time.Time { return &v }

func ConvertRepoFinding(a RepoFindingAPI) *RepoFinding {
	rf := &RepoFinding{}
	if a.ID != nil {
		rf.ID = *a.ID
	}
	rf.IntelligenceID = a.IntelligenceID
	rf.FilePath = a.FilePath
	rf.Category = a.Category
	rf.Status = a.Status
	rf.Title = a.Title
	rf.Body = a.Body
	rf.Severity = a.Severity
	rf.Confidence = a.Confidence
	rf.SourceType = a.SourceType
	rf.SourceRef = a.SourceRef
	rf.SessionID = a.SessionID
	rf.AgentID = a.AgentID
	rf.ResolvedBy = a.ResolvedBy
	rf.ResolvedReason = a.ResolvedReason
	if a.CreatedAt != nil {
		rf.CreatedAt = *a.CreatedAt
	}
	if a.UpdatedAt != nil {
		rf.UpdatedAt = *a.UpdatedAt
	}
	return rf
}

func PresentRepoFinding(rf *RepoFinding) RepoFindingAPI {
	ref := presenters.PresentReference(rf.ID, rf)
	return RepoFindingAPI{
		ID:        ref.Id,
		Kind:      ref.Kind,
		Href:      ref.Href,
		CreatedAt: ptrTime(rf.CreatedAt),
		UpdatedAt: ptrTime(rf.UpdatedAt),

		IntelligenceID: rf.IntelligenceID,
		FilePath:       rf.FilePath,
		Category:       rf.Category,
		Status:         rf.Status,
		Title:          rf.Title,
		Body:           rf.Body,
		Severity:       rf.Severity,
		Confidence:     rf.Confidence,
		SourceType:     rf.SourceType,
		SourceRef:      rf.SourceRef,
		SessionID:      rf.SessionID,
		AgentID:        rf.AgentID,
		ResolvedBy:     rf.ResolvedBy,
		ResolvedReason: rf.ResolvedReason,
	}
}
