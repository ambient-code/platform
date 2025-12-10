// Package handlers provides migration functions for upgrading AgenticSession resources.
package handlers

import (
	"context"
	"fmt"
	"log"

	"ambient-code-operator/internal/config"
	"ambient-code-operator/internal/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	// MigrationAnnotation marks sessions that have been migrated to v2 repo format
	MigrationAnnotation = "ambient-code.io/repos-migrated"
	// MigrationVersion is the current migration version
	MigrationVersion = "v2"
)

// MigrateAllSessions migrates all AgenticSessions from legacy repo format to v2 format.
// This function is idempotent and safe to run multiple times.
//
// Legacy format: repos[].{url, branch}
// New format: repos[].{input: {url, branch}, autoPush: false}
//
// Returns the number of sessions successfully migrated.
func MigrateAllSessions() error {
	ctx := context.Background()
	gvr := types.GetAgenticSessionResource()

	// List all AgenticSessions across all namespaces
	log.Println("Scanning for AgenticSessions requiring migration...")
	sessionList, err := config.DynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list AgenticSessions: %w", err)
	}

	totalSessions := len(sessionList.Items)
	migratedCount := 0
	skippedCount := 0
	errorCount := 0

	log.Printf("Found %d AgenticSessions total", totalSessions)

	for _, session := range sessionList.Items {
		name := session.GetName()
		namespace := session.GetNamespace()

		// Check if session is currently active (skip without annotation)
		status, found, _ := unstructured.NestedMap(session.Object, "status")
		if found && status != nil {
			phase, found, _ := unstructured.NestedString(status, "phase")
			if found && (phase == "Running" || phase == "Creating") {
				skippedCount++
				continue // Skip active sessions entirely (don't add annotation)
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
			errorCount++
			continue
		}

		migratedCount++
		log.Printf("Successfully migrated session %s/%s to v2 repo format", namespace, name)
	}

	log.Printf("Migration complete: %d migrated, %d skipped, %d errors (out of %d total)",
		migratedCount, skippedCount, errorCount, totalSessions)

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

	// Convert each repo from legacy to new format
	migratedRepos := make([]interface{}, 0, len(repos))
	for i, repoInterface := range repos {
		repo, ok := repoInterface.(map[string]interface{})
		if !ok {
			return fmt.Errorf("repo[%d] is not a map", i)
		}

		// Extract legacy fields
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
			"autoPush": false, // Default to false for safety
		}

		if branch != "" {
			newRepo["input"].(map[string]interface{})["branch"] = branch
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
