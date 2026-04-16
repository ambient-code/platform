package handlers

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// --- Feature flag name ---

const correctionsFeatureFlag = "learning-agent-loop"

// --- Types ---

// CorrectionEvent represents a single correction event in the buffer.
type CorrectionEvent struct {
	SessionName    string    `json:"sessionName"`
	CorrectionType string    `json:"correctionType"`
	AgentAction    string    `json:"agentAction"`
	UserCorrection string    `json:"userCorrection"`
	Target         string    `json:"target,omitempty"`
	Source         string    `json:"source"`
	Timestamp      string    `json:"timestamp,omitempty"`
	ReceivedAt     time.Time `json:"receivedAt"`
}

// CorrectionRequest is the JSON body for POST /corrections.
type CorrectionRequest struct {
	SessionName    string `json:"sessionName" binding:"required"`
	CorrectionType string `json:"correctionType" binding:"required"`
	AgentAction    string `json:"agentAction" binding:"required"`
	UserCorrection string `json:"userCorrection" binding:"required"`
	Target         string `json:"target"`
	Source         string `json:"source" binding:"required"`
	Timestamp      string `json:"timestamp"`
}

// --- Allowed enum values ---

var allowedCorrectionTypes = map[string]bool{
	"incomplete":   true,
	"incorrect":    true,
	"out_of_scope": true,
	"style":        true,
}

var allowedCorrectionSources = map[string]bool{
	"human":  true,
	"rubric": true,
	"ui":     true,
}

// --- Per-project buffer ---

const (
	maxEventsPerProject = 10000
	eventTTL            = 24 * time.Hour
)

// projectBuffer is a goroutine-safe FIFO buffer for a single project.
type projectBuffer struct {
	mu     sync.RWMutex
	events []CorrectionEvent
}

// append adds an event, evicting the oldest if the buffer is full.
func (b *projectBuffer) append(event CorrectionEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.events) >= maxEventsPerProject {
		// FIFO eviction: drop the oldest event
		b.events = b.events[1:]
	}
	b.events = append(b.events, event)
}

// list returns non-expired events, optionally filtered by session and target.
func (b *projectBuffer) list(session, target string) []CorrectionEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	cutoff := time.Now().Add(-eventTTL)
	result := make([]CorrectionEvent, 0)
	for _, e := range b.events {
		if e.ReceivedAt.Before(cutoff) {
			continue
		}
		if session != "" && e.SessionName != session {
			continue
		}
		if target != "" && e.Target != target {
			continue
		}
		result = append(result, e)
	}
	return result
}

// summary returns correction counts grouped by target, optionally filtered.
func (b *projectBuffer) summary(target string) map[string]int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	cutoff := time.Now().Add(-eventTTL)
	counts := make(map[string]int)
	for _, e := range b.events {
		if e.ReceivedAt.Before(cutoff) {
			continue
		}
		if target != "" && e.Target != target {
			continue
		}
		key := e.Target
		if key == "" {
			key = "(none)"
		}
		counts[key]++
	}
	return counts
}

// --- Global buffer registry ---

var (
	buffersMu sync.RWMutex
	buffers   = make(map[string]*projectBuffer)
)

func getProjectBuffer(project string) *projectBuffer {
	buffersMu.RLock()
	buf, ok := buffers[project]
	buffersMu.RUnlock()
	if ok {
		return buf
	}

	buffersMu.Lock()
	defer buffersMu.Unlock()
	// Double-check after acquiring write lock
	if buf, ok = buffers[project]; ok {
		return buf
	}
	buf = &projectBuffer{}
	buffers[project] = buf
	return buf
}

// ResetCorrectionsBuffers clears all buffers. Exported for tests only.
func ResetCorrectionsBuffers() {
	buffersMu.Lock()
	defer buffersMu.Unlock()
	buffers = make(map[string]*projectBuffer)
}

// isCorrectionsEnabled checks workspace ConfigMap override first, then
// falls back to the Unleash SDK. This mirrors the pattern used by
// isRunnerEnabledWithOverrides in runner_types.go.
func isCorrectionsEnabled(c *gin.Context) bool {
	namespace := sanitizeParam(c.Param("projectName"))
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		return false
	}

	// Check workspace ConfigMap override first
	overrides, err := getWorkspaceOverrides(c.Request.Context(), reqK8s, namespace)
	if err == nil && overrides != nil {
		if val, exists := overrides[correctionsFeatureFlag]; exists {
			return val == "true"
		}
	}

	// Fall back to Unleash SDK
	return FeatureEnabledForRequest(c, correctionsFeatureFlag)
}

// --- Handlers ---

// PostCorrection handles POST /api/projects/:projectName/corrections
func PostCorrection(c *gin.Context) {
	project := sanitizeParam(c.Param("projectName"))

	// Feature flag gate
	if !isCorrectionsEnabled(c) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Feature not enabled"})
		return
	}

	var req CorrectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Validate correctionType enum
	if !allowedCorrectionTypes[req.CorrectionType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid correctionType. Must be one of: incomplete, incorrect, out_of_scope, style",
		})
		return
	}

	// Validate source enum
	if !allowedCorrectionSources[req.Source] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid source. Must be one of: human, rubric, ui",
		})
		return
	}

	event := CorrectionEvent{
		SessionName:    req.SessionName,
		CorrectionType: req.CorrectionType,
		AgentAction:    req.AgentAction,
		UserCorrection: req.UserCorrection,
		Target:         req.Target,
		Source:         req.Source,
		Timestamp:      req.Timestamp,
		ReceivedAt:     time.Now(),
	}

	buf := getProjectBuffer(project)
	buf.append(event)

	log.Printf("Correction received: project=%s session=%s type=%s target=%s source=%s",
		project, req.SessionName, req.CorrectionType, req.Target, req.Source)

	c.JSON(http.StatusCreated, gin.H{"message": "Correction recorded"})
}

// ListCorrections handles GET /api/projects/:projectName/corrections
func ListCorrections(c *gin.Context) {
	// Feature flag gate
	if !isCorrectionsEnabled(c) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Feature not enabled"})
		return
	}

	project := sanitizeParam(c.Param("projectName"))
	session := c.Query("session")
	target := c.Query("target")

	buf := getProjectBuffer(project)
	events := buf.list(session, target)

	c.JSON(http.StatusOK, gin.H{"corrections": events})
}

// GetCorrectionsSummary handles GET /api/projects/:projectName/corrections/summary
func GetCorrectionsSummary(c *gin.Context) {
	// Feature flag gate
	if !isCorrectionsEnabled(c) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Feature not enabled"})
		return
	}

	project := sanitizeParam(c.Param("projectName"))
	target := c.Query("target")

	buf := getProjectBuffer(project)
	counts := buf.summary(target)

	c.JSON(http.StatusOK, gin.H{"summary": counts})
}
