// Package handlers provides migration functions for upgrading AgenticSession resources.
package handlers

import (
	"context"
	"fmt"
	"log"

	"ambient-code-operator/internal/config"
	"ambient-code-operator/internal/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	// MigrationAnnotation marks sessions that have been migrated to v2 repo format
	MigrationAnnotation = "ambient-code.io/repos-migrated"
	// MigrationVersion is the current migration version
	MigrationVersion = "v2"

	// Event reasons for migration tracking
	eventReasonMigrationStarted   = "MigrationStarted"
	eventReasonMigrationCompleted = "MigrationCompleted"
	eventReasonMigrationFailed    = "MigrationFailed"
)

// MigrateAllSessions migrates all AgenticSessions from legacy repo format to v2 format.
// This function is idempotent and safe to run multiple times.
//
// Legacy format: repos[].{url, branch}
// New format: repos[].{input: {url, branch}, autoPush: false}
//
// SECURITY MODEL: This migration runs at operator startup using the operator's service
// account privileges. This is intentional and safe because:
//  1. Migration occurs before any user requests are processed (operator startup only)
//  2. The operator is reconciling existing CRs it already has RBAC access to
//  3. No new user data is being created or accessed
//  4. Migration only updates the repo structure format, not repository content
//  5. Active sessions (Running/Creating) are skipped to avoid interfering with in-flight work
//
// Returns nil even if individual migrations fail (failures are logged).
// This allows operator startup to continue and retry failed sessions on next restart.
func MigrateAllSessions() error {
	ctx := context.Background()
	gvr := types.GetAgenticSessionResource()

	// List all AgenticSessions across all namespaces
	log.Println("========================================")
	log.Println("Starting AgenticSession v2 migration...")
	log.Println("========================================")
	sessionList, err := config.DynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list AgenticSessions: %w", err)
	}

	totalSessions := len(sessionList.Items)
	migratedCount := 0
	skippedCount := 0
	errorCount := 0
	activeSkippedCount := 0

	log.Printf("Found %d AgenticSessions to process", totalSessions)

	for _, session := range sessionList.Items {
		name := session.GetName()
		namespace := session.GetNamespace()

		// Check if session is currently active
		status, found, _ := unstructured.NestedMap(session.Object, "status")
		if found && status != nil {
			phase, found, _ := unstructured.NestedString(status, "phase")
			if found && (phase == "Running" || phase == "Creating") {
				// Add annotation to track that migration was skipped due to active status
				// This allows us to identify sessions that need migration once they complete
				annotations := session.GetAnnotations()
				if annotations == nil {
					annotations = make(map[string]string)
				}
				// Only add if not already present
				if _, exists := annotations["ambient-code.io/migration-skipped-active"]; !exists {
					annotations["ambient-code.io/migration-skipped-active"] = phase
					session.SetAnnotations(annotations)

					gvr := types.GetAgenticSessionResource()
					if _, err := config.DynamicClient.Resource(gvr).Namespace(namespace).Update(context.TODO(), &session, metav1.UpdateOptions{}); err != nil {
						log.Printf("Warning: failed to add skip annotation to active session %s/%s: %v", namespace, name, err)
					} else {
						log.Printf("Marked active session %s/%s (phase: %s) as skipped for migration", namespace, name, phase)
					}
				}
				activeSkippedCount++
				continue
			}
		}

		// Check if already migrated
		annotations := session.GetAnnotations()
		if annotations != nil {
			if version, exists := annotations[MigrationAnnotation]; exists {
				if version == MigrationVersion {
					skippedCount++
					continue
				}
			}
		}

		// Check if session has repos that need migration
		if needsMigration, err := sessionNeedsMigration(&session); err != nil {
			log.Printf("ERROR: Failed to check migration status for session %s/%s: %v", namespace, name, err)
			recordMigrationEvent(&session, corev1.EventTypeWarning, eventReasonMigrationFailed,
				fmt.Sprintf("Failed to check migration status: %v", err))
			errorCount++
			continue
		} else if !needsMigration {
			// Session already has new format (no migration annotation but already using v2 format)
			if err := addMigrationAnnotation(&session, namespace, name); err != nil {
				log.Printf("ERROR: Failed to add migration annotation to session %s/%s: %v", namespace, name, err)
				errorCount++
			} else {
				skippedCount++
			}
			continue
		}

		// Migrate the session
		if err := migrateSession(&session, namespace, name); err != nil {
			log.Printf("ERROR: Failed to migrate session %s/%s: %v", namespace, name, err)
			recordMigrationEvent(&session, corev1.EventTypeWarning, eventReasonMigrationFailed,
				fmt.Sprintf("Failed to migrate to v2 repo format: %v", err))
			errorCount++
			continue
		}

		migratedCount++
		log.Printf("Successfully migrated session %s/%s to v2 repo format", namespace, name)
		recordMigrationEvent(&session, corev1.EventTypeNormal, eventReasonMigrationCompleted,
			"Successfully migrated to v2 repo format")
	}

	log.Println("========================================")
	log.Println("Migration Summary:")
	log.Printf("  Total sessions processed: %d", totalSessions)
	log.Printf("  Successfully migrated: %d", migratedCount)
	log.Printf("  Already migrated (skipped): %d", skippedCount)
	log.Printf("  Active sessions (skipped): %d", activeSkippedCount)
	log.Printf("  Errors: %d", errorCount)

	if errorCount > 0 {
		log.Printf("⚠️  Migration completed with %d errors - failed sessions will retry on next restart", errorCount)
	} else {
		log.Println("✅ Migration completed successfully")
	}
	log.Println("========================================")

	// Note: We return nil even with errors to allow operator startup to continue.
	// Individual session errors are logged above and don't prevent other sessions
	// from being migrated. Failed sessions will be retried on next operator restart.
	return nil
}

// sessionNeedsMigration checks if a session has repos in legacy format.
// Returns true if migration is needed, false otherwise.
// NOTE: Active sessions are filtered out before this function is called.
func sessionNeedsMigration(session *unstructured.Unstructured) (bool, error) {
	spec, found, err := unstructured.NestedMap(session.Object, "spec")
	if err != nil || !found {
		return false, nil // No spec or error reading it - nothing to migrate
	}

	repos, found, err := unstructured.NestedSlice(spec, "repos")
	if err != nil {
		return false, fmt.Errorf("failed to read repos: %w", err)
	}
	if !found || len(repos) == 0 {
		return false, nil // No repos - nothing to migrate
	}

	// Defensive type checking: validate first element type before processing
	// Catches type errors early with clearer error messages
	if len(repos) > 0 {
		if _, ok := repos[0].(map[string]interface{}); !ok {
			return false, fmt.Errorf("repos[0]: invalid type %T (expected map)", repos[0])
		}
	}

	// Check all repos to detect if any are in legacy format
	// We check all repos (not just the first) to handle edge cases where
	// someone manually edited a CR and created mixed formats
	for i, repoInterface := range repos {
		repo, ok := repoInterface.(map[string]interface{})
		if !ok {
			return false, fmt.Errorf("repo[%d]: invalid format (not a map)", i)
		}

		// Legacy format has "url" field directly
		// New format has "input" object with "url" field
		_, hasURL := repo["url"]
		_, hasInput := repo["input"]

		if hasURL && !hasInput {
			return true, nil // Found at least one legacy repo
		} else if hasInput {
			// New format - continue checking other repos
			continue
		} else {
			// Neither url nor input - invalid repo
			return false, fmt.Errorf("repo[%d]: missing both 'url' and 'input' fields", i)
		}
	}

	return false, nil // All repos are in new format
}

// migrateSession converts a session's repos from legacy to v2 format and updates the CR.
func migrateSession(session *unstructured.Unstructured, namespace, name string) error {
	ctx := context.Background()
	gvr := types.GetAgenticSessionResource()

	spec, found, err := unstructured.NestedMap(session.Object, "spec")
	if err != nil || !found {
		return fmt.Errorf("failed to read spec: %w", err)
	}

	repos, found, err := unstructured.NestedSlice(spec, "repos")
	if err != nil {
		return fmt.Errorf("failed to read repos: %w", err)
	}
	if !found || len(repos) == 0 {
		return nil // No repos to migrate
	}

	// Read session-level autoPushOnComplete to use as default for per-repo autoPush
	// This preserves existing behavior when migrating from v1 to v2 format.
	// Defaults to false if the field is missing or has an invalid type (safe default).
	defaultAutoPush := false
	if autoPushOnComplete, found, err := unstructured.NestedBool(spec, "autoPushOnComplete"); err == nil && found {
		defaultAutoPush = autoPushOnComplete
	}

	// Convert each repo from legacy to new format
	// Handle mixed v1/v2 format (edge case from manual CR editing)
	migratedRepos := make([]interface{}, 0, len(repos))
	for i, repoInterface := range repos {
		repo, ok := repoInterface.(map[string]interface{})
		if !ok {
			return fmt.Errorf("repo[%d] is not a map", i)
		}

		// Check if repo is already in v2 format
		_, hasInput := repo["input"]
		if hasInput {
			// Already v2 format, preserve as-is
			migratedRepos = append(migratedRepos, repo)
			continue
		}

		// Migrate from v1 format
		url, hasURL := repo["url"].(string)
		if !hasURL {
			return fmt.Errorf("repo[%d] missing url field", i)
		}

		branch, _ := repo["branch"].(string) // Optional field

		// Create new format
		newRepo := map[string]interface{}{
			"input": map[string]interface{}{
				"url": url,
			},
			"autoPush": defaultAutoPush, // Use session-level autoPushOnComplete as default
		}

		if branch != "" {
			inputMap, ok := newRepo["input"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("migration failed: input field at repo[%d] is not a map", i)
			}
			inputMap["branch"] = branch
		}

		// Preserve name if present
		if repoName, hasName := repo["name"].(string); hasName {
			newRepo["name"] = repoName
		}

		migratedRepos = append(migratedRepos, newRepo)
	}

	// Validate migrated data before updating
	if len(migratedRepos) == 0 {
		return fmt.Errorf("migration produced empty repos list")
	}

	for i, repoInterface := range migratedRepos {
		repo, ok := repoInterface.(map[string]interface{})
		if !ok {
			return fmt.Errorf("validation failed: migrated repo[%d] is not a map", i)
		}

		input, ok := repo["input"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("validation failed: migrated repo[%d] missing input object", i)
		}

		if _, ok := input["url"].(string); !ok {
			return fmt.Errorf("validation failed: migrated repo[%d] missing input.url", i)
		}

		if _, ok := repo["autoPush"].(bool); !ok {
			return fmt.Errorf("validation failed: migrated repo[%d] missing autoPush field", i)
		}
	}

	// Update spec with migrated repos
	if err := unstructured.SetNestedSlice(spec, migratedRepos, "repos"); err != nil {
		return fmt.Errorf("failed to set migrated repos: %w", err)
	}

	if err := unstructured.SetNestedMap(session.Object, spec, "spec"); err != nil {
		return fmt.Errorf("failed to set spec: %w", err)
	}

	// Add migration annotation
	annotations := session.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[MigrationAnnotation] = MigrationVersion
	session.SetAnnotations(annotations)

	// Update the CR
	_, err = config.DynamicClient.Resource(gvr).Namespace(namespace).Update(ctx, session, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update CR: %w", err)
	}

	return nil
}

// addMigrationAnnotation adds the migration annotation to a session that's already in v2 format.
func addMigrationAnnotation(session *unstructured.Unstructured, namespace, name string) error {
	ctx := context.Background()
	gvr := types.GetAgenticSessionResource()

	annotations := session.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[MigrationAnnotation] = MigrationVersion
	session.SetAnnotations(annotations)

	_, err := config.DynamicClient.Resource(gvr).Namespace(namespace).Update(ctx, session, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to add annotation: %w", err)
	}

	return nil
}

// recordMigrationEvent records a Kubernetes Event for migration tracking.
// Events appear in `kubectl describe agenticsession` output.
func recordMigrationEvent(session *unstructured.Unstructured, eventType, reason, message string) {
	ctx := context.Background()

	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s.%s", session.GetName(), metav1.Now().Format("20060102150405")),
			Namespace: session.GetNamespace(),
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:       session.GetKind(),
			Namespace:  session.GetNamespace(),
			Name:       session.GetName(),
			UID:        session.GetUID(),
			APIVersion: session.GetAPIVersion(),
		},
		Reason:  reason,
		Message: message,
		Type:    eventType,
		Source: corev1.EventSource{
			Component: "agentic-operator",
		},
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
		Count:          1,
	}

	_, err := config.K8sClient.CoreV1().Events(session.GetNamespace()).Create(ctx, event, metav1.CreateOptions{})
	if err != nil {
		// Don't fail migration if event recording fails
		log.Printf("Warning: failed to record migration event for %s/%s: %v", session.GetNamespace(), session.GetName(), err)
	}
}
