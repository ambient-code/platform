// Package handlers provides inactivity timeout detection and auto-stop logic
// for agentic sessions. The operator's monitorPod loop calls shouldAutoStop()
// on each tick to check whether a Running session has exceeded its configured
// inactivity timeout, and triggerInactivityStop() to initiate the shutdown.
package handlers

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"ambient-code-operator/internal/config"
	"ambient-code-operator/internal/types"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	// stopReasonAnnotation is set on the CR before triggering a stop so the
	// Stopping phase handler can distinguish inactivity from user stops.
	stopReasonAnnotation = "ambient-code.io/stop-reason"

	// defaultInactivityTimeoutSec is the fallback when neither the session
	// nor the project specifies an inactivity timeout (24 hours).
	defaultInactivityTimeoutSec = 86400

	// inactivityTimeoutCacheTTL controls how long cached ProjectSettings
	// timeout values are valid before re-fetching from the API server.
	inactivityTimeoutCacheTTL = 5 * time.Minute

	// projectSettingsName is the well-known name for the singleton
	// ProjectSettings CR in each namespace.
	projectSettingsName = "projectsettings"
)

// --- Project-level timeout cache ---

// projectTimeoutCache caches inactivityTimeoutSeconds from ProjectSettings per namespace.
type projectTimeoutCache struct {
	mu      sync.Mutex
	entries map[string]projectTimeoutEntry
}

type projectTimeoutEntry struct {
	timeout   int64
	fetchedAt time.Time
}

var psTimeoutCache = &projectTimeoutCache{
	entries: make(map[string]projectTimeoutEntry),
}

// getProjectInactivityTimeout reads inactivityTimeoutSeconds from the ProjectSettings
// CR in the given namespace. Results are cached for inactivityTimeoutCacheTTL.
// Returns -1 if the field is not set (so the caller should use the session or default value).
func getProjectInactivityTimeout(namespace string) int64 {
	// Check cache under lock
	psTimeoutCache.mu.Lock()
	if entry, ok := psTimeoutCache.entries[namespace]; ok {
		if time.Since(entry.fetchedAt) < inactivityTimeoutCacheTTL {
			psTimeoutCache.mu.Unlock()
			return entry.timeout
		}
	}
	psTimeoutCache.mu.Unlock()

	// Fetch from API server without holding the lock
	gvr := types.GetProjectSettingsResource()
	obj, err := config.DynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), projectSettingsName, v1.GetOptions{})

	var result int64 = -1
	if err == nil {
		if val, found, _ := unstructured.NestedInt64(obj.Object, "spec", "inactivityTimeoutSeconds"); found {
			result = val
		}
	}

	// Update cache under lock
	psTimeoutCache.mu.Lock()
	psTimeoutCache.entries[namespace] = projectTimeoutEntry{timeout: result, fetchedAt: time.Now()}
	psTimeoutCache.mu.Unlock()

	return result
}

// --- Timeout resolution ---

// resolveInactivityTimeout determines the effective inactivity timeout for a session.
// Precedence: session spec > project settings > default (86400s / 24h).
func resolveInactivityTimeout(sessionObj *unstructured.Unstructured) int64 {
	// 1. Check session-level spec.inactivityTimeout
	if val, found, _ := unstructured.NestedInt64(sessionObj.Object, "spec", "inactivityTimeout"); found {
		return val
	}

	// 2. Check project-level ProjectSettings.spec.inactivityTimeoutSeconds
	namespace := sessionObj.GetNamespace()
	if val := getProjectInactivityTimeout(namespace); val >= 0 {
		return val
	}

	// 3. Default
	return defaultInactivityTimeoutSec
}

// --- Auto-stop detection ---

// shouldAutoStop checks whether the session should be auto-stopped due to inactivity.
// Only applies to Running sessions.
func shouldAutoStop(sessionObj *unstructured.Unstructured) bool {
	status, _, _ := unstructured.NestedMap(sessionObj.Object, "status")
	if status == nil {
		return false
	}

	phase, _ := status["phase"].(string)
	if phase != "Running" {
		return false
	}

	timeout := resolveInactivityTimeout(sessionObj)
	if timeout == 0 {
		return false // Disabled
	}

	// Determine last activity time: lastActivityTime > startTime > creationTimestamp
	var lastActivity time.Time

	if lat, ok := status["lastActivityTime"].(string); ok && lat != "" {
		if t, err := time.Parse(time.RFC3339, lat); err == nil {
			lastActivity = t
		}
	}

	if lastActivity.IsZero() {
		if st, ok := status["startTime"].(string); ok && st != "" {
			if t, err := time.Parse(time.RFC3339, st); err == nil {
				lastActivity = t
			}
		}
	}

	if lastActivity.IsZero() {
		ct := sessionObj.GetCreationTimestamp()
		if !ct.IsZero() {
			lastActivity = ct.Time
		}
	}

	if lastActivity.IsZero() {
		return false // No timestamp to compare against
	}

	return time.Since(lastActivity) > time.Duration(timeout)*time.Second
}

// --- Auto-stop trigger ---

// triggerInactivityStop sets the desired-phase annotation to Stopped with a stop-reason
// annotation for inactivity. It re-reads the CR to avoid race conditions.
func triggerInactivityStop(namespace, name string) error {
	gvr := types.GetAgenticSessionResource()

	// Re-read the CR to get fresh state (race condition protection)
	obj, err := config.DynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to re-read session %s/%s: %w", namespace, name, err)
	}

	// Re-check that session is still idle (user may have sent a message)
	if !shouldAutoStop(obj) {
		log.Printf("[Inactivity] Session %s/%s: no longer idle after re-check, skipping auto-stop", namespace, name)
		return nil
	}

	// Set desired-phase and stop-reason annotations
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["ambient-code.io/desired-phase"] = "Stopped"
	annotations[stopReasonAnnotation] = "inactivity"
	obj.SetAnnotations(annotations)

	_, err = config.DynamicClient.Resource(gvr).Namespace(namespace).Update(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to set desired-phase for %s/%s: %w", namespace, name, err)
	}

	log.Printf("[Inactivity] Session %s/%s: set desired-phase=Stopped with reason=inactivity", namespace, name)
	return nil
}
