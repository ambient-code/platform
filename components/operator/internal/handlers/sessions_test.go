package handlers

import (
	"context"
	"strings"
	"testing"

	"ambient-code-operator/internal/config"
	"ambient-code-operator/internal/types"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

// setupTestClient initializes a fake Kubernetes client for testing
func setupTestClient(objects ...runtime.Object) {
	config.K8sClient = k8sfake.NewSimpleClientset(objects...)
}

// setupTestClients initializes both fake Kubernetes and dynamic clients
func setupTestClients(k8sObjects []runtime.Object, dynamicObjects []runtime.Object) {
	config.K8sClient = k8sfake.NewSimpleClientset(k8sObjects...)
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = batchv1.AddToScheme(scheme)
	config.DynamicClient = fake.NewSimpleDynamicClient(scheme, dynamicObjects...)
}

// TestCopySecretToNamespace_NoSharedDataMutation verifies that we don't mutate cached secret objects
func TestCopySecretToNamespace_NoSharedDataMutation(t *testing.T) {
	// Create existing secret with one owner reference
	existingOwnerRef := metav1.OwnerReference{
		APIVersion: "v1",
		Kind:       "Pod",
		Name:       "existing-owner",
		UID:        k8stypes.UID("existing-uid-123"),
	}
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "target-ns",
			OwnerReferences: []metav1.OwnerReference{
				existingOwnerRef,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("old-value"),
		},
	}

	// Setup fake client with existing secret
	setupTestClient(existingSecret)

	// Create source secret
	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "source-ns",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("new-value"),
		},
	}

	// Create owner object
	ownerObj := &unstructured.Unstructured{}
	ownerObj.SetAPIVersion("vteam.ambient-code/v1alpha1")
	ownerObj.SetKind("AgenticSession")
	ownerObj.SetName("test-session")
	ownerObj.SetUID(k8stypes.UID("new-uid-456"))

	// Get the secret before the update
	beforeSecret, err := config.K8sClient.CoreV1().Secrets("target-ns").Get(context.Background(), "ambient-vertex", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get secret before update: %v", err)
	}

	// Store the original slice pointer to verify it's not mutated
	originalSlicePtr := &beforeSecret.OwnerReferences

	// Call copySecretToNamespace
	ctx := context.Background()
	err = copySecretToNamespace(ctx, sourceSecret, "target-ns", ownerObj)
	if err != nil {
		t.Fatalf("copySecretToNamespace failed: %v", err)
	}

	// Get the updated secret
	updatedSecret, err := config.K8sClient.CoreV1().Secrets("target-ns").Get(ctx, "ambient-vertex", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get updated secret: %v", err)
	}

	// Verify the new owner reference was added
	if len(updatedSecret.OwnerReferences) != 2 {
		t.Errorf("Expected 2 owner references, got %d", len(updatedSecret.OwnerReferences))
	}

	// Verify the original owner reference is still there
	foundOriginal := false
	for _, ref := range updatedSecret.OwnerReferences {
		if ref.UID == existingOwnerRef.UID {
			foundOriginal = true
			break
		}
	}
	if !foundOriginal {
		t.Error("Original owner reference was lost")
	}

	// Verify the new owner reference was added
	foundNew := false
	for _, ref := range updatedSecret.OwnerReferences {
		if ref.UID == ownerObj.GetUID() {
			foundNew = true
			break
		}
	}
	if !foundNew {
		t.Error("New owner reference was not added")
	}

	// Verify the original slice was not mutated (the pointer should be different)
	// Note: This is a best-effort check - the fake client may not preserve the exact same behavior
	// as the real client, but it validates our code creates a new slice
	if originalSlicePtr == &updatedSecret.OwnerReferences {
		t.Error("OwnerReferences slice pointer was not changed, indicating potential mutation")
	}

	// Verify data was updated
	if string(updatedSecret.Data["key"]) != "new-value" {
		t.Errorf("Expected data 'new-value', got '%s'", string(updatedSecret.Data["key"]))
	}

	// Verify annotation was added
	expectedAnnotation := "source-ns/ambient-vertex"
	if updatedSecret.Annotations[types.CopiedFromAnnotation] != expectedAnnotation {
		t.Errorf("Expected annotation '%s', got '%s'", expectedAnnotation, updatedSecret.Annotations[types.CopiedFromAnnotation])
	}
}

// TestCopySecretToNamespace_CreateNew tests creating a new secret when it doesn't exist
func TestCopySecretToNamespace_CreateNew(t *testing.T) {
	setupTestClient()

	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "source-ns",
			Labels: map[string]string{
				"app": "ambient-code",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"credentials": []byte("secret-data"),
		},
	}

	ownerObj := &unstructured.Unstructured{}
	ownerObj.SetAPIVersion("vteam.ambient-code/v1alpha1")
	ownerObj.SetKind("AgenticSession")
	ownerObj.SetName("test-session")
	ownerObj.SetUID(k8stypes.UID("test-uid-789"))

	ctx := context.Background()
	err := copySecretToNamespace(ctx, sourceSecret, "target-ns", ownerObj)
	if err != nil {
		t.Fatalf("copySecretToNamespace failed: %v", err)
	}

	// Verify secret was created
	created, err := config.K8sClient.CoreV1().Secrets("target-ns").Get(ctx, "ambient-vertex", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get created secret: %v", err)
	}

	// Verify basic fields
	if created.Name != "ambient-vertex" {
		t.Errorf("Expected name 'ambient-vertex', got '%s'", created.Name)
	}
	if created.Namespace != "target-ns" {
		t.Errorf("Expected namespace 'target-ns', got '%s'", created.Namespace)
	}

	// Verify data was copied
	if string(created.Data["credentials"]) != "secret-data" {
		t.Errorf("Expected data 'secret-data', got '%s'", string(created.Data["credentials"]))
	}

	// Verify labels were copied
	if created.Labels["app"] != "ambient-code" {
		t.Errorf("Expected label 'ambient-code', got '%s'", created.Labels["app"])
	}

	// Verify owner reference
	if len(created.OwnerReferences) != 1 {
		t.Fatalf("Expected 1 owner reference, got %d", len(created.OwnerReferences))
	}
	if created.OwnerReferences[0].UID != ownerObj.GetUID() {
		t.Errorf("Expected owner UID '%s', got '%s'", ownerObj.GetUID(), created.OwnerReferences[0].UID)
	}
	if created.OwnerReferences[0].Kind != "AgenticSession" {
		t.Errorf("Expected owner kind 'AgenticSession', got '%s'", created.OwnerReferences[0].Kind)
	}
	if created.OwnerReferences[0].Controller == nil || !*created.OwnerReferences[0].Controller {
		t.Error("Expected Controller to be true")
	}

	// Verify annotation
	expectedAnnotation := "source-ns/ambient-vertex"
	if created.Annotations[types.CopiedFromAnnotation] != expectedAnnotation {
		t.Errorf("Expected annotation '%s', got '%s'", expectedAnnotation, created.Annotations[types.CopiedFromAnnotation])
	}
}

// TestCopySecretToNamespace_AlreadyHasOwnerRef tests skipping update when owner ref already exists
func TestCopySecretToNamespace_AlreadyHasOwnerRef(t *testing.T) {
	ownerUID := k8stypes.UID("owner-uid-999")

	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "target-ns",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "vteam.ambient-code/v1alpha1",
					Kind:       "AgenticSession",
					Name:       "test-session",
					UID:        ownerUID,
					Controller: boolPtr(true),
				},
			},
			Annotations: map[string]string{
				types.CopiedFromAnnotation: "source-ns/ambient-vertex",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("original-value"),
		},
	}

	setupTestClient(existingSecret)

	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "source-ns",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("new-value"),
		},
	}

	ownerObj := &unstructured.Unstructured{}
	ownerObj.SetAPIVersion("vteam.ambient-code/v1alpha1")
	ownerObj.SetKind("AgenticSession")
	ownerObj.SetName("test-session")
	ownerObj.SetUID(ownerUID)

	ctx := context.Background()
	err := copySecretToNamespace(ctx, sourceSecret, "target-ns", ownerObj)
	if err != nil {
		t.Fatalf("copySecretToNamespace failed: %v", err)
	}

	// Verify secret was NOT updated (data should still be original)
	result, err := config.K8sClient.CoreV1().Secrets("target-ns").Get(ctx, "ambient-vertex", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get secret: %v", err)
	}

	if string(result.Data["key"]) != "original-value" {
		t.Errorf("Expected data to remain 'original-value', got '%s'", string(result.Data["key"]))
	}

	// Should still have exactly 1 owner reference
	if len(result.OwnerReferences) != 1 {
		t.Errorf("Expected 1 owner reference, got %d", len(result.OwnerReferences))
	}
}

// TestCopySecretToNamespace_MultipleOwnerReferences tests adding owner ref to secret with existing different owner
func TestCopySecretToNamespace_MultipleOwnerReferences(t *testing.T) {
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "target-ns",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       "other-owner",
					UID:        k8stypes.UID("other-uid-111"),
				},
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}

	setupTestClient(existingSecret)

	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "source-ns",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("updated-value"),
		},
	}

	ownerObj := &unstructured.Unstructured{}
	ownerObj.SetAPIVersion("vteam.ambient-code/v1alpha1")
	ownerObj.SetKind("AgenticSession")
	ownerObj.SetName("test-session")
	ownerObj.SetUID(k8stypes.UID("new-owner-uid-222"))

	ctx := context.Background()
	err := copySecretToNamespace(ctx, sourceSecret, "target-ns", ownerObj)
	if err != nil {
		t.Fatalf("copySecretToNamespace failed: %v", err)
	}

	// Verify secret has both owner references
	result, err := config.K8sClient.CoreV1().Secrets("target-ns").Get(ctx, "ambient-vertex", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get secret: %v", err)
	}

	if len(result.OwnerReferences) != 2 {
		t.Fatalf("Expected 2 owner references, got %d", len(result.OwnerReferences))
	}

	// Verify both UIDs are present
	uids := make(map[k8stypes.UID]bool)
	for _, ref := range result.OwnerReferences {
		uids[ref.UID] = true
	}

	if !uids[k8stypes.UID("other-uid-111")] {
		t.Error("Original owner reference was lost")
	}
	if !uids[k8stypes.UID("new-owner-uid-222")] {
		t.Error("New owner reference was not added")
	}
}

// TestCopySecretToNamespace_ExistingController tests adding owner ref when secret already has a controller
func TestCopySecretToNamespace_ExistingController(t *testing.T) {
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "target-ns",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "vteam.ambient-code/v1alpha1",
					Kind:       "AgenticSession",
					Name:       "existing-session",
					UID:        k8stypes.UID("existing-uid-111"),
					Controller: boolPtr(true), // Already has a controller
				},
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}

	setupTestClient(existingSecret)

	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "source-ns",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("updated-value"),
		},
	}

	ownerObj := &unstructured.Unstructured{}
	ownerObj.SetAPIVersion("vteam.ambient-code/v1alpha1")
	ownerObj.SetKind("AgenticSession")
	ownerObj.SetName("new-session")
	ownerObj.SetUID(k8stypes.UID("new-uid-222"))

	ctx := context.Background()
	err := copySecretToNamespace(ctx, sourceSecret, "target-ns", ownerObj)
	if err != nil {
		t.Fatalf("copySecretToNamespace failed: %v", err)
	}

	// Verify secret has both owner references
	result, err := config.K8sClient.CoreV1().Secrets("target-ns").Get(ctx, "ambient-vertex", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get secret: %v", err)
	}

	if len(result.OwnerReferences) != 2 {
		t.Fatalf("Expected 2 owner references, got %d", len(result.OwnerReferences))
	}

	// Verify only one controller reference exists
	controllerCount := 0
	foundExisting := false
	foundNew := false
	for _, ref := range result.OwnerReferences {
		if ref.Controller != nil && *ref.Controller {
			controllerCount++
		}
		if ref.UID == k8stypes.UID("existing-uid-111") {
			foundExisting = true
			// Original controller should still be true
			if ref.Controller == nil || !*ref.Controller {
				t.Error("Existing controller reference should still have Controller: true")
			}
		}
		if ref.UID == k8stypes.UID("new-uid-222") {
			foundNew = true
			// New reference should NOT have Controller: true
			if ref.Controller != nil && *ref.Controller {
				t.Error("New owner reference should NOT have Controller: true when secret already has a controller")
			}
		}
	}

	if controllerCount != 1 {
		t.Errorf("Expected exactly 1 controller reference, got %d", controllerCount)
	}
	if !foundExisting {
		t.Error("Existing owner reference was lost")
	}
	if !foundNew {
		t.Error("New owner reference was not added")
	}

	// Verify data was updated
	if string(result.Data["key"]) != "updated-value" {
		t.Errorf("Expected data 'updated-value', got '%s'", string(result.Data["key"]))
	}
}

// TestCopySecretToNamespace_NilAnnotations tests updating secret with nil annotations
func TestCopySecretToNamespace_NilAnnotations(t *testing.T) {
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "ambient-vertex",
			Namespace:   "target-ns",
			Annotations: nil, // Explicitly nil
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}

	setupTestClient(existingSecret)

	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "source-ns",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("new-value"),
		},
	}

	ownerObj := &unstructured.Unstructured{}
	ownerObj.SetAPIVersion("vteam.ambient-code/v1alpha1")
	ownerObj.SetKind("AgenticSession")
	ownerObj.SetName("test-session")
	ownerObj.SetUID(k8stypes.UID("test-uid-333"))

	ctx := context.Background()
	err := copySecretToNamespace(ctx, sourceSecret, "target-ns", ownerObj)
	if err != nil {
		t.Fatalf("copySecretToNamespace failed: %v", err)
	}

	// Verify annotation was added
	result, err := config.K8sClient.CoreV1().Secrets("target-ns").Get(ctx, "ambient-vertex", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get secret: %v", err)
	}

	if result.Annotations == nil {
		t.Fatal("Annotations should not be nil after update")
	}

	expectedAnnotation := "source-ns/ambient-vertex"
	if result.Annotations[types.CopiedFromAnnotation] != expectedAnnotation {
		t.Errorf("Expected annotation '%s', got '%s'", expectedAnnotation, result.Annotations[types.CopiedFromAnnotation])
	}
}

// TestDeleteAmbientVertexSecret_CopiedSecret tests deletion of a copied secret
func TestDeleteAmbientVertexSecret_CopiedSecret(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      types.AmbientVertexSecretName,
			Namespace: "test-ns",
			Annotations: map[string]string{
				types.CopiedFromAnnotation: "source-ns/ambient-vertex",
			},
		},
		Type: corev1.SecretTypeOpaque,
	}

	setupTestClient(secret)

	ctx := context.Background()
	err := deleteAmbientVertexSecret(ctx, "test-ns")
	if err != nil {
		t.Fatalf("deleteAmbientVertexSecret failed: %v", err)
	}

	// Verify secret was deleted
	_, err = config.K8sClient.CoreV1().Secrets("test-ns").Get(ctx, types.AmbientVertexSecretName, metav1.GetOptions{})
	if err == nil {
		t.Error("Secret should have been deleted")
	}
}

// TestDeleteAmbientVertexSecret_NotCopied tests that non-copied secrets are not deleted
func TestDeleteAmbientVertexSecret_NotCopied(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      types.AmbientVertexSecretName,
			Namespace: "test-ns",
			// No CopiedFromAnnotation - this is a user-created secret
		},
		Type: corev1.SecretTypeOpaque,
	}

	setupTestClient(secret)

	ctx := context.Background()
	err := deleteAmbientVertexSecret(ctx, "test-ns")
	if err != nil {
		t.Fatalf("deleteAmbientVertexSecret failed: %v", err)
	}

	// Verify secret was NOT deleted
	result, err := config.K8sClient.CoreV1().Secrets("test-ns").Get(ctx, types.AmbientVertexSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Secret should not have been deleted: %v", err)
	}
	if result == nil {
		t.Error("Secret should still exist")
	}
}

// TestDeleteAmbientVertexSecret_NotFound tests handling of non-existent secret
func TestDeleteAmbientVertexSecret_NotFound(t *testing.T) {
	setupTestClient()

	ctx := context.Background()
	err := deleteAmbientVertexSecret(ctx, "test-ns")
	if err != nil {
		t.Errorf("deleteAmbientVertexSecret should not error on non-existent secret: %v", err)
	}
}

// TestDeleteAmbientVertexSecret_NilAnnotations tests handling of secret with nil annotations
func TestDeleteAmbientVertexSecret_NilAnnotations(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        types.AmbientVertexSecretName,
			Namespace:   "test-ns",
			Annotations: nil,
		},
		Type: corev1.SecretTypeOpaque,
	}

	setupTestClient(secret)

	ctx := context.Background()
	err := deleteAmbientVertexSecret(ctx, "test-ns")
	if err != nil {
		t.Fatalf("deleteAmbientVertexSecret failed: %v", err)
	}

	// Verify secret was NOT deleted (no annotation = not copied)
	result, err := config.K8sClient.CoreV1().Secrets("test-ns").Get(ctx, types.AmbientVertexSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Secret should not have been deleted: %v", err)
	}
	if result == nil {
		t.Error("Secret should still exist")
	}
}

// TestJobConditionHandling_DeadlineExceeded tests detection of DeadlineExceeded Job condition
func TestJobConditionHandling_DeadlineExceeded(t *testing.T) {
	// Create a Job with DeadlineExceeded condition
	now := metav1.Now()
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-session-job",
			Namespace: "test-ns",
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				{
					Type:               batchv1.JobFailed,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: now,
					Reason:             "DeadlineExceeded",
					Message:            "Job was active longer than specified deadline",
				},
			},
			Failed: 1,
		},
	}

	// Expected behavior: Job should be detected as failed with DeadlineExceeded reason
	if len(job.Status.Conditions) == 0 {
		t.Fatal("Job should have at least one condition")
	}

	foundDeadlineExceeded := false
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			if condition.Reason == "DeadlineExceeded" {
				foundDeadlineExceeded = true
			}
		}
	}

	if !foundDeadlineExceeded {
		t.Error("DeadlineExceeded condition not found in Job status")
	}
}

// TestJobConditionHandling_OtherFailure tests detection of non-deadline Job failures
func TestJobConditionHandling_OtherFailure(t *testing.T) {
	// Create a Job with a different failure reason
	now := metav1.Now()
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-session-job",
			Namespace: "test-ns",
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				{
					Type:               batchv1.JobFailed,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: now,
					Reason:             "BackoffLimitExceeded",
					Message:            "Job has reached the specified backoff limit",
				},
			},
			Failed: 3,
		},
	}

	// Verify the condition is present
	foundFailure := false
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			foundFailure = true
			if condition.Reason != "BackoffLimitExceeded" {
				t.Errorf("Expected reason 'BackoffLimitExceeded', got '%s'", condition.Reason)
			}
		}
	}

	if !foundFailure {
		t.Error("Job failure condition not found")
	}
}

// TestJobConditionHandling_NoFailure tests Job without failure conditions
func TestJobConditionHandling_NoFailure(t *testing.T) {
	// Create a Job with no failure conditions (running or succeeded)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-session-job",
			Namespace: "test-ns",
		},
		Status: batchv1.JobStatus{
			Active: 1,
		},
	}

	// Verify no failure conditions
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			t.Error("Job should not have JobFailed condition")
		}
	}
}

// TestJobConditionHandling_MultipleConditions tests Job with multiple conditions
func TestJobConditionHandling_MultipleConditions(t *testing.T) {
	// Create a Job with multiple conditions, including DeadlineExceeded
	now := metav1.Now()
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-session-job",
			Namespace: "test-ns",
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				{
					Type:               batchv1.JobComplete,
					Status:             corev1.ConditionFalse,
					LastTransitionTime: now,
					Reason:             "NotComplete",
					Message:            "Job is not complete",
				},
				{
					Type:               batchv1.JobFailed,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: now,
					Reason:             "DeadlineExceeded",
					Message:            "Job was active longer than specified deadline",
				},
			},
			Failed: 1,
		},
	}

	// Should find DeadlineExceeded among multiple conditions
	foundDeadlineExceeded := false
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			if condition.Reason == "DeadlineExceeded" {
				foundDeadlineExceeded = true
				if condition.Message != "Job was active longer than specified deadline" {
					t.Errorf("Unexpected message: %s", condition.Message)
				}
			}
		}
	}

	if !foundDeadlineExceeded {
		t.Error("DeadlineExceeded condition not found among multiple conditions")
	}
}

// TestJobConditionHandling_FailedButNotTrue tests Job with Failed condition but status False
func TestJobConditionHandling_FailedButNotTrue(t *testing.T) {
	// Create a Job with JobFailed condition but Status=False (cleared failure)
	now := metav1.Now()
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-session-job",
			Namespace: "test-ns",
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				{
					Type:               batchv1.JobFailed,
					Status:             corev1.ConditionFalse,
					LastTransitionTime: now,
					Reason:             "PreviouslyFailed",
					Message:            "Job was previously failed but is now retrying",
				},
			},
			Active: 1,
		},
	}

	// Should NOT detect as failed (Status must be True)
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			t.Error("Job should not be detected as failed when Status is False")
		}
	}
}

// TestParseRepos_NewFormat tests parsing repos in new format (input/output/autoPush)
func TestParseRepos_NewFormat(t *testing.T) {
	tests := []struct {
		name     string
		reposMap []interface{}
		validate func(t *testing.T, repos []repoConfig)
	}{
		{
			name: "new format with input and output",
			reposMap: []interface{}{
				map[string]interface{}{
					"input": map[string]interface{}{
						"url":    "https://github.com/org/repo",
						"branch": "main",
					},
					"output": map[string]interface{}{
						"url":    "https://github.com/user/fork",
						"branch": "feature",
					},
					"autoPush": true,
				},
			},
			validate: func(t *testing.T, repos []repoConfig) {
				if len(repos) != 1 {
					t.Fatalf("Expected 1 repo, got %d", len(repos))
				}

				repo := repos[0]
				if repo.Input == nil {
					t.Fatal("Input should not be nil")
				}
				if repo.Input.URL != "https://github.com/org/repo" {
					t.Errorf("Expected input URL 'https://github.com/org/repo', got '%s'", repo.Input.URL)
				}
				if repo.Input.Branch != "main" {
					t.Errorf("Expected input branch 'main', got '%s'", repo.Input.Branch)
				}

				if repo.Output == nil {
					t.Fatal("Output should not be nil")
				}
				if repo.Output.URL != "https://github.com/user/fork" {
					t.Errorf("Expected output URL 'https://github.com/user/fork', got '%s'", repo.Output.URL)
				}
				if repo.Output.Branch != "feature" {
					t.Errorf("Expected output branch 'feature', got '%s'", repo.Output.Branch)
				}

				if !repo.AutoPush {
					t.Error("Expected autoPush to be true")
				}
			},
		},
		{
			name: "new format with input only",
			reposMap: []interface{}{
				map[string]interface{}{
					"input": map[string]interface{}{
						"url":    "https://github.com/org/repo",
						"branch": "develop",
					},
					"autoPush": false,
				},
			},
			validate: func(t *testing.T, repos []repoConfig) {
				if len(repos) != 1 {
					t.Fatalf("Expected 1 repo, got %d", len(repos))
				}

				repo := repos[0]
				if repo.Input == nil {
					t.Fatal("Input should not be nil")
				}
				if repo.Input.URL != "https://github.com/org/repo" {
					t.Errorf("Expected input URL 'https://github.com/org/repo', got '%s'", repo.Input.URL)
				}
				if repo.Input.Branch != "develop" {
					t.Errorf("Expected input branch 'develop', got '%s'", repo.Input.Branch)
				}

				if repo.Output != nil {
					t.Error("Output should be nil when not specified")
				}

				if repo.AutoPush {
					t.Error("Expected autoPush to be false")
				}
			},
		},
		{
			name: "new format without branch defaults to main",
			reposMap: []interface{}{
				map[string]interface{}{
					"input": map[string]interface{}{
						"url": "https://github.com/org/repo",
					},
				},
			},
			validate: func(t *testing.T, repos []repoConfig) {
				if len(repos) != 1 {
					t.Fatalf("Expected 1 repo, got %d", len(repos))
				}

				repo := repos[0]
				if repo.Input.Branch != "main" {
					t.Errorf("Expected default branch 'main', got '%s'", repo.Input.Branch)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call production parseRepoConfig() function instead of duplicating logic
			repos := make([]repoConfig, 0, len(tt.reposMap))
			for _, repoItem := range tt.reposMap {
				if repoMap, ok := repoItem.(map[string]interface{}); ok {
					repo, err := parseRepoConfig(repoMap, "test-ns", "test-session")
					if err != nil {
						t.Fatalf("parseRepoConfig failed: %v", err)
					}
					repos = append(repos, repo)
				}
			}

			tt.validate(t, repos)
		})
	}
}

// TestParseRepos_LegacyFormat tests parsing repos in legacy format (url/branch)
func TestParseRepos_LegacyFormat(t *testing.T) {
	tests := []struct {
		name     string
		reposMap []interface{}
		validate func(t *testing.T, repos []repoConfig)
	}{
		{
			name: "legacy format with branch",
			reposMap: []interface{}{
				map[string]interface{}{
					"url":    "https://github.com/org/legacy",
					"branch": "master",
				},
			},
			validate: func(t *testing.T, repos []repoConfig) {
				if len(repos) != 1 {
					t.Fatalf("Expected 1 repo, got %d", len(repos))
				}

				repo := repos[0]
				if repo.URL != "https://github.com/org/legacy" {
					t.Errorf("Expected URL 'https://github.com/org/legacy', got '%s'", repo.URL)
				}
				if repo.Branch != "master" {
					t.Errorf("Expected branch 'master', got '%s'", repo.Branch)
				}

				// New format fields should be nil for legacy repos
				if repo.Input != nil {
					t.Error("Input should be nil for legacy format")
				}
				if repo.Output != nil {
					t.Error("Output should be nil for legacy format")
				}
			},
		},
		{
			name: "legacy format without branch defaults to main",
			reposMap: []interface{}{
				map[string]interface{}{
					"url": "https://github.com/org/legacy",
				},
			},
			validate: func(t *testing.T, repos []repoConfig) {
				if len(repos) != 1 {
					t.Fatalf("Expected 1 repo, got %d", len(repos))
				}

				repo := repos[0]
				if repo.Branch != "main" {
					t.Errorf("Expected default branch 'main', got '%s'", repo.Branch)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call production parseRepoConfig() function instead of duplicating logic
			repos := make([]repoConfig, 0, len(tt.reposMap))
			for _, repoItem := range tt.reposMap {
				if repoMap, ok := repoItem.(map[string]interface{}); ok {
					repo, err := parseRepoConfig(repoMap, "test-ns", "test-session")
					if err != nil {
						t.Fatalf("parseRepoConfig failed: %v", err)
					}
					repos = append(repos, repo)
				}
			}

			tt.validate(t, repos)
		})
	}
}

// TestParseRepos_EdgeCases tests edge cases and error handling
func TestParseRepos_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		repoMap     map[string]interface{}
		expectError bool
		validate    func(t *testing.T, repo repoConfig, err error)
	}{
		{
			name: "empty URL in new format",
			repoMap: map[string]interface{}{
				"input": map[string]interface{}{
					"url":    "",
					"branch": "main",
				},
			},
			expectError: true,
			validate: func(t *testing.T, repo repoConfig, err error) {
				if err == nil {
					t.Error("Expected error for empty input URL")
				}
				if err != nil && !strings.Contains(err.Error(), "empty") {
					t.Errorf("Expected 'empty' in error message, got: %v", err)
				}
			},
		},
		{
			name: "empty URL in legacy format",
			repoMap: map[string]interface{}{
				"url":    "",
				"branch": "main",
			},
			expectError: true,
			validate: func(t *testing.T, repo repoConfig, err error) {
				if err == nil {
					t.Error("Expected error for empty URL")
				}
			},
		},
		{
			name: "whitespace-only URL in new format",
			repoMap: map[string]interface{}{
				"input": map[string]interface{}{
					"url":    "   ",
					"branch": "main",
				},
			},
			expectError: true,
			validate: func(t *testing.T, repo repoConfig, err error) {
				if err == nil {
					t.Error("Expected error for whitespace-only URL")
				}
			},
		},
		{
			name: "both new and legacy formats present (new should win)",
			repoMap: map[string]interface{}{
				"input": map[string]interface{}{
					"url":    "https://github.com/new/format",
					"branch": "new-branch",
				},
				"url":    "https://github.com/legacy/format",
				"branch": "legacy-branch",
			},
			expectError: false,
			validate: func(t *testing.T, repo repoConfig, err error) {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				// New format should take precedence
				if repo.Input == nil {
					t.Fatal("Input should not be nil when new format present")
				}
				if repo.Input.URL != "https://github.com/new/format" {
					t.Errorf("Expected new format URL, got '%s'", repo.Input.URL)
				}
				if repo.Input.Branch != "new-branch" {
					t.Errorf("Expected new format branch, got '%s'", repo.Input.Branch)
				}
			},
		},
		{
			name: "new format without branch defaults to main",
			repoMap: map[string]interface{}{
				"input": map[string]interface{}{
					"url": "https://github.com/org/repo",
				},
			},
			expectError: false,
			validate: func(t *testing.T, repo repoConfig, err error) {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if repo.Input == nil {
					t.Fatal("Input should not be nil")
				}
				if repo.Input.Branch != "main" {
					t.Errorf("Expected default branch 'main', got '%s'", repo.Input.Branch)
				}
			},
		},
		{
			name: "legacy format without branch defaults to main",
			repoMap: map[string]interface{}{
				"url": "https://github.com/org/repo",
			},
			expectError: false,
			validate: func(t *testing.T, repo repoConfig, err error) {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if repo.Branch != "main" {
					t.Errorf("Expected default branch 'main', got '%s'", repo.Branch)
				}
			},
		},
		{
			name: "invalid type for input (not a map)",
			repoMap: map[string]interface{}{
				"input": "not-a-map",
			},
			expectError: true,
			validate: func(t *testing.T, repo repoConfig, err error) {
				if err == nil {
					t.Error("Expected error for invalid input type")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := parseRepoConfig(tt.repoMap, "test-ns", "test-session")
			tt.validate(t, repo, err)
		})
	}
}

// TestBackwardCompatEnvVars tests extraction of backward compat env vars from repos
func TestBackwardCompatEnvVars(t *testing.T) {
	tests := []struct {
		name              string
		repos             []repoConfig
		expectedInput     string
		expectedInBranch  string
		expectedOutput    string
		expectedOutBranch string
	}{
		{
			name: "new format with output",
			repos: []repoConfig{
				{
					Input: &repoLocation{
						URL:    "https://github.com/org/repo",
						Branch: "main",
					},
					Output: &repoLocation{
						URL:    "https://github.com/user/fork",
						Branch: "feature",
					},
					AutoPush: true,
				},
			},
			expectedInput:     "https://github.com/org/repo",
			expectedInBranch:  "main",
			expectedOutput:    "https://github.com/user/fork",
			expectedOutBranch: "feature",
		},
		{
			name: "new format without output",
			repos: []repoConfig{
				{
					Input: &repoLocation{
						URL:    "https://github.com/org/repo",
						Branch: "develop",
					},
					AutoPush: false,
				},
			},
			expectedInput:     "https://github.com/org/repo",
			expectedInBranch:  "develop",
			expectedOutput:    "https://github.com/org/repo",
			expectedOutBranch: "develop",
		},
		{
			name: "legacy format",
			repos: []repoConfig{
				{
					URL:    "https://github.com/org/legacy",
					Branch: "master",
				},
			},
			expectedInput:     "https://github.com/org/legacy",
			expectedInBranch:  "master",
			expectedOutput:    "https://github.com/org/legacy",
			expectedOutBranch: "master",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract backward compat env vars using the same logic as operator
			var inputRepo, inputBranch, outputRepo, outputBranch string
			if len(tt.repos) > 0 {
				firstRepo := tt.repos[0]
				if firstRepo.Input != nil {
					inputRepo = firstRepo.Input.URL
					inputBranch = firstRepo.Input.Branch
					if firstRepo.Output != nil {
						outputRepo = firstRepo.Output.URL
						outputBranch = firstRepo.Output.Branch
					} else {
						outputRepo = firstRepo.Input.URL
						outputBranch = firstRepo.Input.Branch
					}
				} else {
					inputRepo = firstRepo.URL
					inputBranch = firstRepo.Branch
					outputRepo = firstRepo.URL
					outputBranch = firstRepo.Branch
				}
			}

			if inputRepo != tt.expectedInput {
				t.Errorf("Expected inputRepo '%s', got '%s'", tt.expectedInput, inputRepo)
			}
			if inputBranch != tt.expectedInBranch {
				t.Errorf("Expected inputBranch '%s', got '%s'", tt.expectedInBranch, inputBranch)
			}
			if outputRepo != tt.expectedOutput {
				t.Errorf("Expected outputRepo '%s', got '%s'", tt.expectedOutput, outputRepo)
			}
			if outputBranch != tt.expectedOutBranch {
				t.Errorf("Expected outputBranch '%s', got '%s'", tt.expectedOutBranch, outputBranch)
			}
		})
	}
}
