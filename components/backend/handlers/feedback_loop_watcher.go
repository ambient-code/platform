package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// IsFeedbackLoopEnabled is a package-level var so tests can override the
// feature flag check. In production it delegates to FeatureEnabled().
// Uses correctionsFeatureFlag (defined in corrections.go) — all learning
// agent loop features share one flag.
var IsFeedbackLoopEnabled = func() bool {
	return FeatureEnabled(correctionsFeatureFlag)
}

// CorrectionNotification is the payload passed to the watcher when a correction
// is logged via the corrections pipeline (spec 003).
type CorrectionNotification struct {
	Project        string    `json:"project"`
	TargetType     string    `json:"target_type"` // "workflow" or "repo"
	TargetRepoURL  string    `json:"target_repo_url"`
	TargetBranch   string    `json:"target_branch"`
	TargetPath     string    `json:"target_path"`
	CorrectionType string    `json:"correction_type"` // "incomplete", "incorrect", "out_of_scope", "style"
	Source         string    `json:"source"`          // "human" or "rubric"
	AgentAction    string    `json:"agent_action"`
	UserCorrection string    `json:"user_correction"`
	SessionName    string    `json:"session_name"`
	TraceID        string    `json:"trace_id"`
	Timestamp      time.Time `json:"timestamp"`
}

// bufferedCorrection stores a correction in the per-target buffer.
type bufferedCorrection struct {
	CorrectionNotification
	ReceivedAt time.Time
}

// targetBuffer holds buffered corrections for a single target key.
type targetBuffer struct {
	Corrections []bufferedCorrection
}

// FeedbackLoopWatcher evaluates corrections against per-project thresholds
// and creates improvement sessions when thresholds are crossed.
//
// The watcher maintains an in-memory buffer of recent corrections per target
// for fast evaluation. Deduplication state and history are persisted in a
// ConfigMap so that backend restarts do not create duplicate sessions.
//
// NOTE (v1): The in-memory correction buffer is lost on backend restart.
// This means corrections logged before a restart will not count toward the
// threshold after restart. This is acceptable for v1 because:
//  1. The weekly GHA sweep catches anything the real-time path misses.
//  2. Persisting the full correction buffer in ConfigMap would add write
//     amplification on every correction log (acceptable trade-off for v2).
type FeedbackLoopWatcher struct {
	mu      sync.Mutex
	buffers map[string]*targetBuffer // key: "project|targetKey"
}

// NewFeedbackLoopWatcher creates a new watcher instance.
func NewFeedbackLoopWatcher() *FeedbackLoopWatcher {
	return &FeedbackLoopWatcher{
		buffers: make(map[string]*targetBuffer),
	}
}

// NotifyCorrection is called asynchronously when a correction is logged.
// It buffers the correction and evaluates whether the threshold has been crossed.
// Returns true if an improvement session was triggered (for testing).
//
// This function MUST NOT add latency to the correction logging path (NFR-002).
// Callers should invoke it in a goroutine.
func (w *FeedbackLoopWatcher) NotifyCorrection(ctx context.Context, n CorrectionNotification) bool {
	// Gate behind feature flag
	if !IsFeedbackLoopEnabled() {
		return false
	}

	// Load project config
	config, err := loadFeedbackLoopConfig(ctx, n.Project)
	if err != nil {
		log.Printf("feedback-loop: failed to load config for %s: %v", n.Project, err)
		return false
	}

	if !config.AutoTriggerEnabled {
		return false
	}

	targetKey := groupCorrectionKey(n.TargetType, n.TargetRepoURL, n.TargetBranch, n.TargetPath)
	bufferKey := fmt.Sprintf("%s|%s", n.Project, targetKey)

	w.mu.Lock()
	defer w.mu.Unlock()

	buf, ok := w.buffers[bufferKey]
	if !ok {
		buf = &targetBuffer{}
		w.buffers[bufferKey] = buf
	}

	// Add the correction
	buf.Corrections = append(buf.Corrections, bufferedCorrection{
		CorrectionNotification: n,
		ReceivedAt:             time.Now(),
	})

	// Prune corrections outside the time window
	cutoff := time.Now().Add(-time.Duration(config.TimeWindowHours) * time.Hour)
	pruned := buf.Corrections[:0]
	for _, c := range buf.Corrections {
		if c.Timestamp.After(cutoff) {
			pruned = append(pruned, c)
		}
	}
	buf.Corrections = pruned

	// Check threshold
	if len(buf.Corrections) < config.MinCorrections {
		return false
	}

	// Check deduplication: was an improvement session already created for this target
	// within the time window? Checks the persisted history in ConfigMap.
	if w.isDuplicate(ctx, n.Project, targetKey, cutoff) {
		return false
	}

	// Threshold crossed -- create improvement session
	group := w.buildGroupFromBuffer(buf, n)
	sessionName, err := w.createImprovementSession(ctx, n.Project, group)
	if err != nil {
		log.Printf("feedback-loop: failed to create improvement session for %s in %s: %v", targetKey, n.Project, err)
		return false
	}

	// Record in history for deduplication and the history endpoint
	traceIDs := make([]string, len(buf.Corrections))
	for i, c := range buf.Corrections {
		traceIDs[i] = c.TraceID
	}

	entry := FeedbackLoopHistoryEntry{
		SessionName:   sessionName,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		Source:        "event-driven",
		TargetType:    n.TargetType,
		TargetRepoURL: n.TargetRepoURL,
		TargetBranch:  n.TargetBranch,
		TargetPath:    n.TargetPath,
		CorrectionIDs: traceIDs,
	}
	if err := appendFeedbackLoopHistory(ctx, n.Project, entry); err != nil {
		log.Printf("feedback-loop: failed to record history for %s: %v", n.Project, err)
	}

	// Clear the buffer for this target to prevent re-triggering
	buf.Corrections = nil

	log.Printf("feedback-loop: triggered improvement session %s for target %s in project %s", sessionName, targetKey, n.Project)
	return true
}

// isDuplicate checks whether an improvement session was already created for
// this target within the time window by reading the persisted history.
func (w *FeedbackLoopWatcher) isDuplicate(ctx context.Context, project, targetKey string, cutoff time.Time) bool {
	entries, err := loadFeedbackLoopHistory(ctx, project)
	if err != nil {
		log.Printf("feedback-loop: failed to load history for dedup check in %s: %v", project, err)
		// Fail open: prefer creating a possible duplicate over silently dropping
		return false
	}

	for _, entry := range entries {
		entryKey := groupCorrectionKey(entry.TargetType, entry.TargetRepoURL, entry.TargetBranch, entry.TargetPath)
		if entryKey != targetKey {
			continue
		}
		createdAt, err := time.Parse(time.RFC3339, entry.CreatedAt)
		if err != nil {
			continue
		}
		if createdAt.After(cutoff) {
			return true
		}
	}

	return false
}

// buildGroupFromBuffer constructs a correctionGroup from the buffered corrections.
func (w *FeedbackLoopWatcher) buildGroupFromBuffer(buf *targetBuffer, n CorrectionNotification) correctionGroup {
	typeCounts := map[string]int{}
	sourceCounts := map[string]int{}
	details := make([]correctionDetail, len(buf.Corrections))

	for i, c := range buf.Corrections {
		typeCounts[c.CorrectionType]++
		sourceCounts[c.Source]++
		details[i] = correctionDetail{
			CorrectionType: c.CorrectionType,
			Source:         c.Source,
			AgentAction:    c.AgentAction,
			UserCorrection: c.UserCorrection,
			SessionName:    c.SessionName,
			TraceID:        c.TraceID,
		}
	}

	return correctionGroup{
		TargetType:           n.TargetType,
		TargetRepoURL:        n.TargetRepoURL,
		TargetBranch:         n.TargetBranch,
		TargetPath:           n.TargetPath,
		Corrections:          details,
		TotalCount:           len(buf.Corrections),
		CorrectionTypeCounts: typeCounts,
		SourceCounts:         sourceCounts,
	}
}

// createImprovementSession creates a new AgenticSession CR for the improvement.
// Returns the session name on success.
func (w *FeedbackLoopWatcher) createImprovementSession(ctx context.Context, project string, group correctionGroup) (string, error) {
	prompt := buildImprovementPrompt(group)
	displayName := buildSessionDisplayName(group.TargetType, group.TargetRepoURL, group.TargetPath)

	labels := map[string]interface{}{
		"feedback-loop": "true",
		"source":        "event-driven",
		"target-type":   group.TargetType,
	}

	spec := map[string]interface{}{
		"initialPrompt": prompt,
		"displayName":   displayName,
		"timeout":       300,
		"llmSettings": map[string]interface{}{
			"model":       "claude-sonnet-4-6",
			"temperature": 0.7,
			"maxTokens":   4000,
		},
		"environmentVariables": map[string]interface{}{
			"LANGFUSE_MASK_MESSAGES": "false",
		},
	}

	// Add repo if available
	if group.TargetRepoURL != "" && strings.HasPrefix(group.TargetRepoURL, "http") {
		repo := map[string]interface{}{
			"url":      group.TargetRepoURL,
			"autoPush": true,
		}
		if group.TargetBranch != "" {
			repo["branch"] = group.TargetBranch
		}
		spec["repos"] = []interface{}{repo}
	}

	sessionName := fmt.Sprintf("feedback-%s", time.Now().UTC().Format("20060102-150405"))

	sessionObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      sessionName,
				"namespace": project,
				"labels":    labels,
			},
			"spec": spec,
		},
	}

	gvr := GetAgenticSessionV1Alpha1Resource()
	_, err := DynamicClient.Resource(gvr).Namespace(project).Create(ctx, sessionObj, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create improvement session CR: %w", err)
	}

	return sessionName, nil
}

// buildSessionDisplayName constructs a human-readable display name for an
// improvement session. Matches the naming convention from
// scripts/feedback-loop/query_corrections.py.
func buildSessionDisplayName(targetType, repoURL, targetPath string) string {
	repoShort := repoShortName(repoURL)
	if targetType == "workflow" {
		pathShort := ""
		if targetPath != "" {
			parts := strings.Split(strings.TrimRight(targetPath, "/"), "/")
			pathShort = parts[len(parts)-1]
		}
		name := "Feedback Loop: " + repoShort
		if pathShort != "" {
			name += " (" + pathShort + ")"
		}
		return name
	}
	return "Feedback Loop: " + repoShort + " (repo)"
}
