package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// LearningSummary represents aggregated learning metrics for a project.
type LearningSummary struct {
	TotalCorrections    int            `json:"totalCorrections"`
	CorrectionsByType   map[string]int `json:"correctionsByType"`
	ImprovementSessions int            `json:"improvementSessions"`
	MemoriesCreated     int            `json:"memoriesCreated"`
	MemoryCitations     int            `json:"memoryCitations"`
}

// TimelineEntry represents a single event in the learning timeline.
type TimelineEntry struct {
	ID                 string `json:"id"`
	Timestamp          string `json:"timestamp"`
	EventType          string `json:"eventType"`
	Summary            string `json:"summary"`
	CorrectionType     string `json:"correctionType,omitempty"`
	ImprovementSession string `json:"improvementSession,omitempty"`
	MemoryID           string `json:"memoryId,omitempty"`
}

// TimelineResponse wraps timeline entries with pagination metadata.
type TimelineResponse struct {
	Items      []TimelineEntry `json:"items"`
	TotalCount int             `json:"totalCount"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
}

// GetLearningSummary returns aggregated learning metrics for a project.
// GET /api/projects/:projectName/learning/summary
//
// This endpoint returns correction counts, improvement session counts,
// and memory creation counts. Data is sourced from the corrections
// pipeline (spec 003) and project memory store (spec 002).
//
// Until those specs are implemented, this returns zero-value metrics.
func GetLearningSummary(c *gin.Context) {
	_, dynClient := GetK8sClientsForRequest(c)
	if dynClient == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// projectName is validated by ValidateProjectContext middleware
	// _ = c.Param("projectName")

	// TODO(spec-002, spec-003): Query actual data from corrections pipeline
	// and project memory store once those specs are implemented.
	// For now, return empty/zero-value summary.
	summary := LearningSummary{
		TotalCorrections:    0,
		CorrectionsByType:   map[string]int{},
		ImprovementSessions: 0,
		MemoriesCreated:     0,
		MemoryCitations:     0,
	}

	c.JSON(http.StatusOK, summary)
}

// GetLearningTimeline returns a paginated, reverse-chronological list of
// correction events for a project.
// GET /api/projects/:projectName/learning/timeline?page=1&pageSize=20
//
// Until specs 002/003 are implemented, this returns an empty list.
func GetLearningTimeline(c *gin.Context) {
	_, dynClient := GetK8sClientsForRequest(c)
	if dynClient == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Parse pagination params with defaults
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// TODO(spec-002, spec-003): Query actual timeline events from
	// corrections pipeline once implemented.
	response := TimelineResponse{
		Items:      []TimelineEntry{},
		TotalCount: 0,
		Page:       page,
		PageSize:   pageSize,
	}

	c.JSON(http.StatusOK, response)
}
