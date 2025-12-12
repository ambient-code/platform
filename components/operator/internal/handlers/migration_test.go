package handlers_test

import (
	"context"
	"testing"

	"ambient-code-operator/internal/config"
	"ambient-code-operator/internal/handlers"
	"ambient-code-operator/internal/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

// setupTestDynamicClient initializes a fake dynamic client for testing with custom resource support
func setupTestDynamicClient(objects ...runtime.Object) {
	scheme := runtime.NewScheme()

	// Register the AgenticSession resource type
	// We need to use NewSimpleDynamicClientWithCustomListKinds to support List operations
	gvrToListKind := map[schema.GroupVersionResource]string{
		types.GetAgenticSessionResource(): "AgenticSessionList",
	}

	config.DynamicClient = fake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind, objects...)

	// Initialize fake K8sClient for Event recording
	config.K8sClient = k8sfake.NewSimpleClientset()
}

// createLegacySession creates an AgenticSession with legacy repo format
func createLegacySession(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"repos": []interface{}{
					map[string]interface{}{
						"url":    "https://github.com/org/repo1.git",
						"branch": "main",
					},
					map[string]interface{}{
						"url":    "https://github.com/org/repo2.git",
						"branch": "develop",
						"name":   "secondary-repo",
					},
				},
			},
		},
	}
}

// createV2Session creates an AgenticSession with v2 repo format
func createV2Session(name, namespace string, withAnnotation bool) *unstructured.Unstructured {
	session := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"repos": []interface{}{
					map[string]interface{}{
						"input": map[string]interface{}{
							"url":    "https://github.com/org/repo.git",
							"branch": "main",
						},
						"autoPush": true,
					},
				},
			},
		},
	}

	if withAnnotation {
		session.SetAnnotations(map[string]string{
			handlers.MigrationAnnotation: handlers.MigrationVersion,
		})
	}

	return session
}

// createSessionWithoutRepos creates an AgenticSession with no repos
func createSessionWithoutRepos(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{},
		},
	}
}

func TestMigrateAllSessions_NoSessions(t *testing.T) {
	setupTestDynamicClient()

	err := handlers.MigrateAllSessions()
	if err != nil {
		t.Errorf("MigrateAllSessions() with no sessions should not error, got: %v", err)
	}
}

func TestMigrateAllSessions_SingleLegacySession(t *testing.T) {
	session := createLegacySession("test-session", "default")
	setupTestDynamicClient(session)

	err := handlers.MigrateAllSessions()
	if err != nil {
		t.Fatalf("MigrateAllSessions() failed: %v", err)
	}

	// Verify the session was migrated
	gvr := types.GetAgenticSessionResource()
	updated, err := config.DynamicClient.Resource(gvr).Namespace("default").Get(context.Background(), "test-session", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get updated session: %v", err)
	}

	// Check migration annotation
	annotations := updated.GetAnnotations()
	if annotations == nil {
		t.Fatal("Expected annotations to be set")
	}
	if annotations[handlers.MigrationAnnotation] != handlers.MigrationVersion {
		t.Errorf("Expected migration annotation %s, got: %s", handlers.MigrationVersion, annotations[handlers.MigrationAnnotation])
	}

	// Check repo format
	spec, found, err := unstructured.NestedMap(updated.Object, "spec")
	if err != nil || !found {
		t.Fatal("Failed to get spec")
	}

	repos, found, err := unstructured.NestedSlice(spec, "repos")
	if err != nil || !found {
		t.Fatal("Failed to get repos")
	}

	if len(repos) != 2 {
		t.Fatalf("Expected 2 repos, got %d", len(repos))
	}

	// Verify first repo
	repo1, ok := repos[0].(map[string]interface{})
	if !ok {
		t.Fatal("Repo is not a map")
	}

	input1, found, err := unstructured.NestedMap(repo1, "input")
	if err != nil || !found {
		t.Fatal("Expected repo to have 'input' field")
	}

	url1, found, err := unstructured.NestedString(input1, "url")
	if err != nil || !found {
		t.Fatal("Expected input to have 'url' field")
	}
	if url1 != "https://github.com/org/repo1.git" {
		t.Errorf("Expected url 'https://github.com/org/repo1.git', got: %s", url1)
	}

	branch1, found, err := unstructured.NestedString(input1, "branch")
	if err != nil || !found {
		t.Fatal("Expected input to have 'branch' field")
	}
	if branch1 != "main" {
		t.Errorf("Expected branch 'main', got: %s", branch1)
	}

	autoPush1, found, err := unstructured.NestedBool(repo1, "autoPush")
	if err != nil || !found {
		t.Fatal("Expected repo to have 'autoPush' field")
	}
	if autoPush1 {
		t.Error("Expected autoPush to be false by default")
	}

	// Verify second repo (has name field)
	repo2, ok := repos[1].(map[string]interface{})
	if !ok {
		t.Fatal("Repo 2 is not a map")
	}

	name2, found, err := unstructured.NestedString(repo2, "name")
	if err != nil || !found {
		t.Fatal("Expected repo 2 to have 'name' field")
	}
	if name2 != "secondary-repo" {
		t.Errorf("Expected name 'secondary-repo', got: %s", name2)
	}
}

func TestMigrateAllSessions_AlreadyMigrated(t *testing.T) {
	session := createV2Session("test-session", "default", true)
	setupTestDynamicClient(session)

	err := handlers.MigrateAllSessions()
	if err != nil {
		t.Fatalf("MigrateAllSessions() failed: %v", err)
	}

	// Verify session was not modified (still has annotation)
	gvr := types.GetAgenticSessionResource()
	updated, err := config.DynamicClient.Resource(gvr).Namespace("default").Get(context.Background(), "test-session", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	annotations := updated.GetAnnotations()
	if annotations[handlers.MigrationAnnotation] != handlers.MigrationVersion {
		t.Error("Migration annotation should be preserved")
	}

	// Verify repo format is still v2
	spec, found, err := unstructured.NestedMap(updated.Object, "spec")
	if err != nil || !found {
		t.Fatal("Failed to get spec")
	}

	repos, found, err := unstructured.NestedSlice(spec, "repos")
	if err != nil || !found {
		t.Fatal("Failed to get repos")
	}

	repo, ok := repos[0].(map[string]interface{})
	if !ok {
		t.Fatal("Repo is not a map")
	}

	// Should still have input field
	_, found, err = unstructured.NestedMap(repo, "input")
	if err != nil || !found {
		t.Error("Expected repo to still have 'input' field")
	}
}

func TestMigrateAllSessions_V2WithoutAnnotation(t *testing.T) {
	// Session already has v2 format but no annotation
	session := createV2Session("test-session", "default", false)
	setupTestDynamicClient(session)

	err := handlers.MigrateAllSessions()
	if err != nil {
		t.Fatalf("MigrateAllSessions() failed: %v", err)
	}

	// Verify annotation was added without modifying repos
	gvr := types.GetAgenticSessionResource()
	updated, err := config.DynamicClient.Resource(gvr).Namespace("default").Get(context.Background(), "test-session", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	annotations := updated.GetAnnotations()
	if annotations == nil {
		t.Fatal("Expected annotations to be set")
	}
	if annotations[handlers.MigrationAnnotation] != handlers.MigrationVersion {
		t.Errorf("Expected migration annotation to be added")
	}

	// Verify autoPush value was preserved (not reset to false)
	spec, found, err := unstructured.NestedMap(updated.Object, "spec")
	if err != nil || !found {
		t.Fatal("Failed to get spec")
	}

	repos, found, err := unstructured.NestedSlice(spec, "repos")
	if err != nil || !found {
		t.Fatal("Failed to get repos")
	}

	repo, ok := repos[0].(map[string]interface{})
	if !ok {
		t.Fatal("Repo is not a map")
	}

	autoPush, found, err := unstructured.NestedBool(repo, "autoPush")
	if err != nil || !found {
		t.Fatal("Expected repo to have 'autoPush' field")
	}
	if autoPush != true {
		t.Error("Expected autoPush to be preserved as true")
	}
}

func TestMigrateAllSessions_SessionWithoutRepos(t *testing.T) {
	session := createSessionWithoutRepos("test-session", "default")
	setupTestDynamicClient(session)

	err := handlers.MigrateAllSessions()
	if err != nil {
		t.Fatalf("MigrateAllSessions() with session without repos should not error, got: %v", err)
	}

	// Session should be marked as checked (annotation added) even with no repos
	gvr := types.GetAgenticSessionResource()
	updated, err := config.DynamicClient.Resource(gvr).Namespace("default").Get(context.Background(), "test-session", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	// Annotation should be added to mark it as checked (even with no repos)
	annotations := updated.GetAnnotations()
	if annotations == nil || annotations[handlers.MigrationAnnotation] != handlers.MigrationVersion {
		t.Error("Should add migration annotation to sessions without repos to mark as checked")
	}
}

func TestMigrateAllSessions_MultipleNamespaces(t *testing.T) {
	session1 := createLegacySession("session-1", "namespace-1")
	session2 := createLegacySession("session-2", "namespace-2")
	session3 := createV2Session("session-3", "namespace-3", true)

	setupTestDynamicClient(session1, session2, session3)

	err := handlers.MigrateAllSessions()
	if err != nil {
		t.Fatalf("MigrateAllSessions() failed: %v", err)
	}

	gvr := types.GetAgenticSessionResource()

	// Verify session-1 was migrated
	updated1, err := config.DynamicClient.Resource(gvr).Namespace("namespace-1").Get(context.Background(), "session-1", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get session-1: %v", err)
	}
	annotations1 := updated1.GetAnnotations()
	if annotations1 == nil || annotations1[handlers.MigrationAnnotation] != handlers.MigrationVersion {
		t.Error("Session-1 should have migration annotation")
	}

	// Verify session-2 was migrated
	updated2, err := config.DynamicClient.Resource(gvr).Namespace("namespace-2").Get(context.Background(), "session-2", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get session-2: %v", err)
	}
	annotations2 := updated2.GetAnnotations()
	if annotations2 == nil || annotations2[handlers.MigrationAnnotation] != handlers.MigrationVersion {
		t.Error("Session-2 should have migration annotation")
	}

	// Verify session-3 was skipped (already migrated)
	updated3, err := config.DynamicClient.Resource(gvr).Namespace("namespace-3").Get(context.Background(), "session-3", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get session-3: %v", err)
	}
	annotations3 := updated3.GetAnnotations()
	if annotations3[handlers.MigrationAnnotation] != handlers.MigrationVersion {
		t.Error("Session-3 should still have migration annotation")
	}
}

func TestMigrateAllSessions_Idempotency(t *testing.T) {
	session := createLegacySession("test-session", "default")
	setupTestDynamicClient(session)

	// Run migration first time
	err := handlers.MigrateAllSessions()
	if err != nil {
		t.Fatalf("First migration failed: %v", err)
	}

	// Get the migrated session
	gvr := types.GetAgenticSessionResource()
	firstMigration, err := config.DynamicClient.Resource(gvr).Namespace("default").Get(context.Background(), "test-session", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get session after first migration: %v", err)
	}

	// Run migration second time (should be idempotent)
	err = handlers.MigrateAllSessions()
	if err != nil {
		t.Fatalf("Second migration failed: %v", err)
	}

	// Get the session again
	secondMigration, err := config.DynamicClient.Resource(gvr).Namespace("default").Get(context.Background(), "test-session", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get session after second migration: %v", err)
	}

	// Verify the session was not modified on second run
	firstSpec, _, _ := unstructured.NestedMap(firstMigration.Object, "spec")
	secondSpec, _, _ := unstructured.NestedMap(secondMigration.Object, "spec")

	firstRepos, _, _ := unstructured.NestedSlice(firstSpec, "repos")
	secondRepos, _, _ := unstructured.NestedSlice(secondSpec, "repos")

	if len(firstRepos) != len(secondRepos) {
		t.Error("Idempotency check failed: repo count changed on second migration")
	}

	// Verify annotations are the same
	firstAnnotations := firstMigration.GetAnnotations()
	secondAnnotations := secondMigration.GetAnnotations()

	if firstAnnotations[handlers.MigrationAnnotation] != secondAnnotations[handlers.MigrationAnnotation] {
		t.Error("Idempotency check failed: annotation changed on second migration")
	}
}

func TestMigrateAllSessions_InvalidRepoFormat(t *testing.T) {
	// Session with invalid repo format (string instead of map)
	session := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "invalid-session",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"repos": []interface{}{
					"invalid-string-instead-of-map",
				},
			},
		},
	}

	setupTestDynamicClient(session)

	// Should not panic, but should log error and continue
	err := handlers.MigrateAllSessions()
	if err != nil {
		t.Errorf("MigrateAllSessions() should return nil even with invalid sessions, got: %v", err)
	}

	// Session should not have migration annotation (failed validation)
	gvr := types.GetAgenticSessionResource()
	updated, err := config.DynamicClient.Resource(gvr).Namespace("default").Get(context.Background(), "invalid-session", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	annotations := updated.GetAnnotations()
	if annotations != nil && annotations[handlers.MigrationAnnotation] == handlers.MigrationVersion {
		t.Error("Invalid session should not have migration annotation")
	}
}

func TestMigrateAllSessions_MissingURLField(t *testing.T) {
	// Session with repo missing required url field
	session := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "missing-url-session",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"repos": []interface{}{
					map[string]interface{}{
						"branch": "main",
						// Missing "url" field
					},
				},
			},
		},
	}

	setupTestDynamicClient(session)

	// Should handle gracefully
	err := handlers.MigrateAllSessions()
	if err != nil {
		t.Errorf("MigrateAllSessions() should return nil even with errors, got: %v", err)
	}
}

func TestMigrateAllSessions_PartialFailure(t *testing.T) {
	// Create mix of valid and invalid sessions
	validSession := createLegacySession("valid-session", "default")
	invalidSession := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "invalid-session",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"repos": []interface{}{
					"not-a-map",
				},
			},
		},
	}

	setupTestDynamicClient(validSession, invalidSession)

	err := handlers.MigrateAllSessions()
	if err != nil {
		t.Errorf("MigrateAllSessions() should return nil even with partial failures, got: %v", err)
	}

	gvr := types.GetAgenticSessionResource()

	// Verify valid session was migrated successfully
	validUpdated, err := config.DynamicClient.Resource(gvr).Namespace("default").Get(context.Background(), "valid-session", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get valid session: %v", err)
	}

	validAnnotations := validUpdated.GetAnnotations()
	if validAnnotations == nil || validAnnotations[handlers.MigrationAnnotation] != handlers.MigrationVersion {
		t.Error("Valid session should have been migrated successfully")
	}

	// Verify invalid session was not migrated
	invalidUpdated, err := config.DynamicClient.Resource(gvr).Namespace("default").Get(context.Background(), "invalid-session", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get invalid session: %v", err)
	}

	invalidAnnotations := invalidUpdated.GetAnnotations()
	if invalidAnnotations != nil && invalidAnnotations[handlers.MigrationAnnotation] == handlers.MigrationVersion {
		t.Error("Invalid session should not have migration annotation")
	}
}

func TestMigrateAllSessions_ActiveSession(t *testing.T) {
	// Session with Running status should be skipped
	session := createLegacySession("running-session", "default")
	// Add Running status
	status := map[string]interface{}{
		"phase": "Running",
	}
	session.Object["status"] = status

	setupTestDynamicClient(session)

	err := handlers.MigrateAllSessions()
	if err != nil {
		t.Fatalf("MigrateAllSessions() failed: %v", err)
	}

	// Verify session was skipped (no annotation added)
	gvr := types.GetAgenticSessionResource()
	updated, err := config.DynamicClient.Resource(gvr).Namespace("default").Get(context.Background(), "running-session", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	annotations := updated.GetAnnotations()
	if annotations != nil && annotations[handlers.MigrationAnnotation] == handlers.MigrationVersion {
		t.Error("Running session should not have been migrated")
	}

	// Verify repos are still in legacy format (not migrated)
	spec, found, _ := unstructured.NestedMap(updated.Object, "spec")
	if !found {
		t.Fatal("Failed to get spec")
	}

	repos, found, _ := unstructured.NestedSlice(spec, "repos")
	if !found {
		t.Fatal("Failed to get repos")
	}

	repo, ok := repos[0].(map[string]interface{})
	if !ok {
		t.Fatal("Repo is not a map")
	}

	// Should still have legacy "url" field (not migrated)
	if _, hasURL := repo["url"]; !hasURL {
		t.Error("Running session should retain legacy format")
	}
}

func TestMigrateAllSessions_MixedFormats(t *testing.T) {
	legacySession := createLegacySession("legacy-session", "default")
	v2Session := createV2Session("v2-session", "default", false)
	v2MigratedSession := createV2Session("v2-migrated-session", "default", true)
	noReposSession := createSessionWithoutRepos("no-repos-session", "default")

	setupTestDynamicClient(legacySession, v2Session, v2MigratedSession, noReposSession)

	err := handlers.MigrateAllSessions()
	if err != nil {
		t.Fatalf("MigrateAllSessions() failed: %v", err)
	}

	gvr := types.GetAgenticSessionResource()

	// Verify legacy session was migrated
	legacyUpdated, _ := config.DynamicClient.Resource(gvr).Namespace("default").Get(context.Background(), "legacy-session", metav1.GetOptions{})
	legacyAnnotations := legacyUpdated.GetAnnotations()
	if legacyAnnotations == nil || legacyAnnotations[handlers.MigrationAnnotation] != handlers.MigrationVersion {
		t.Error("Legacy session should have migration annotation")
	}

	// Verify v2 session got annotation added
	v2Updated, _ := config.DynamicClient.Resource(gvr).Namespace("default").Get(context.Background(), "v2-session", metav1.GetOptions{})
	v2Annotations := v2Updated.GetAnnotations()
	if v2Annotations == nil || v2Annotations[handlers.MigrationAnnotation] != handlers.MigrationVersion {
		t.Error("V2 session should have migration annotation added")
	}

	// Verify already-migrated session was skipped
	v2MigratedUpdated, _ := config.DynamicClient.Resource(gvr).Namespace("default").Get(context.Background(), "v2-migrated-session", metav1.GetOptions{})
	v2MigratedAnnotations := v2MigratedUpdated.GetAnnotations()
	if v2MigratedAnnotations[handlers.MigrationAnnotation] != handlers.MigrationVersion {
		t.Error("Already-migrated session annotation should be preserved")
	}

	// Verify no-repos session was marked as checked
	noReposUpdated, _ := config.DynamicClient.Resource(gvr).Namespace("default").Get(context.Background(), "no-repos-session", metav1.GetOptions{})
	noReposAnnotations := noReposUpdated.GetAnnotations()
	if noReposAnnotations == nil || noReposAnnotations[handlers.MigrationAnnotation] != handlers.MigrationVersion {
		t.Error("No-repos session should have migration annotation to mark as checked")
	}
}

// TestMigrateAllSessions_SingleSessionMixedV1V2Repos tests migration of a session
// with some repos in v1 format and some in v2 format (edge case from manual editing)
func TestMigrateAllSessions_SingleSessionMixedV1V2Repos(t *testing.T) {
	// Create a session with mixed repo formats (could happen from manual CR editing)
	mixedSession := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "mixed-repos-session",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"repos": []interface{}{
					// v1 format repo
					map[string]interface{}{
						"url":    "https://github.com/org/legacy-repo.git",
						"branch": "main",
					},
					// v2 format repo
					map[string]interface{}{
						"input": map[string]interface{}{
							"url":    "https://github.com/org/new-repo.git",
							"branch": "develop",
						},
						"autoPush": false,
					},
					// v1 format repo without branch
					map[string]interface{}{
						"url": "https://github.com/org/another-legacy.git",
					},
				},
			},
		},
	}

	setupTestDynamicClient(mixedSession)

	err := handlers.MigrateAllSessions()
	if err != nil {
		t.Fatalf("MigrateAllSessions() failed: %v", err)
	}

	gvr := types.GetAgenticSessionResource()

	// Verify session was migrated
	updated, err := config.DynamicClient.Resource(gvr).Namespace("default").Get(context.Background(), "mixed-repos-session", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get updated session: %v", err)
	}

	// Check migration annotation
	annotations := updated.GetAnnotations()
	if annotations == nil || annotations[handlers.MigrationAnnotation] != handlers.MigrationVersion {
		t.Error("Session with mixed repo formats should have migration annotation")
	}

	// Verify all repos are now in v2 format
	spec, found, err := unstructured.NestedMap(updated.Object, "spec")
	if !found || err != nil {
		t.Fatalf("Failed to get spec: %v", err)
	}

	repos, found, err := unstructured.NestedSlice(spec, "repos")
	if !found || err != nil {
		t.Fatalf("Failed to get repos: %v", err)
	}

	if len(repos) != 3 {
		t.Fatalf("Expected 3 repos, got %d", len(repos))
	}

	// Verify each repo is in v2 format
	for i, repoInterface := range repos {
		repo, ok := repoInterface.(map[string]interface{})
		if !ok {
			t.Errorf("Repo %d is not a map", i)
			continue
		}

		// All repos should have "input" field
		input, hasInput := repo["input"]
		if !hasInput {
			t.Errorf("Repo %d missing input field after migration", i)
			continue
		}

		inputMap, ok := input.(map[string]interface{})
		if !ok {
			t.Errorf("Repo %d input is not a map", i)
			continue
		}

		// Check URL exists
		if _, hasURL := inputMap["url"]; !hasURL {
			t.Errorf("Repo %d input missing url field", i)
		}

		// All repos should have autoPush field
		if _, hasAutoPush := repo["autoPush"]; !hasAutoPush {
			t.Errorf("Repo %d missing autoPush field after migration", i)
		}

		// Legacy fields should not be present
		if _, hasLegacyURL := repo["url"]; hasLegacyURL {
			t.Errorf("Repo %d should not have legacy url field after migration", i)
		}
	}

	// Verify that migration events were recorded
	events, err := config.K8sClient.CoreV1().Events("default").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list events: %v", err)
	}

	// Should have one event for the successful migration
	foundMigrationEvent := false
	for _, event := range events.Items {
		if event.InvolvedObject.Name == "mixed-repos-session" && event.Reason == "MigrationCompleted" {
			foundMigrationEvent = true
			if event.Type != "Normal" {
				t.Errorf("Expected event type 'Normal', got '%s'", event.Type)
			}
			if event.Message != "Successfully migrated to v2 repo format" {
				t.Errorf("Unexpected event message: %s", event.Message)
			}
		}
	}

	if !foundMigrationEvent {
		t.Error("Expected MigrationCompleted event to be recorded")
	}
}
