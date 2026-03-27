package handlers

import (
	"context"
	"testing"

	"ambient-code-operator/internal/config"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

// setupTestClient initializes a fake Kubernetes client for testing.
// This mirrors the pattern in sessions_test.go.
func setupProjectSettingsTestClient(objects ...runtime.Object) {
	config.K8sClient = fake.NewSimpleClientset(objects...)
}

func testProjectSettingsOwner(namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "ProjectSettings",
			"metadata": map[string]interface{}{
				"name":      "projectsettings",
				"namespace": namespace,
				"uid":       string(k8stypes.UID("test-uid-1234")),
			},
		},
	}
}

func TestEnsureLimitRange_CreatesLimitRange(t *testing.T) {
	setupProjectSettingsTestClient()

	namespace := "test-namespace"

	err := ensureLimitRange(namespace, testProjectSettingsOwner(namespace))
	if err != nil {
		t.Fatalf("ensureLimitRange() returned error: %v", err)
	}

	ctx := context.Background()

	lr, err := config.K8sClient.CoreV1().LimitRanges(namespace).Get(ctx, "ambient-default-limits", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get LimitRange: %v", err)
	}
	if lr.Name != "ambient-default-limits" {
		t.Errorf("LimitRange name = %q, want %q", lr.Name, "ambient-default-limits")
	}
	if lr.Labels["ambient-code.io/managed"] != "true" {
		t.Errorf("LimitRange missing managed label")
	}

	// Verify correct values
	if len(lr.Spec.Limits) != 1 {
		t.Fatalf("Expected 1 limit item, got %d", len(lr.Spec.Limits))
	}
	item := lr.Spec.Limits[0]
	if item.Type != corev1.LimitTypeContainer {
		t.Errorf("Limit type = %q, want %q", item.Type, corev1.LimitTypeContainer)
	}

	// Check defaultRequest
	cpuReq := item.DefaultRequest[corev1.ResourceCPU]
	if cpuReq.String() != "250m" {
		t.Errorf("DefaultRequest CPU = %q, want %q", cpuReq.String(), "250m")
	}
	memReq := item.DefaultRequest[corev1.ResourceMemory]
	if memReq.String() != "256Mi" {
		t.Errorf("DefaultRequest Memory = %q, want %q", memReq.String(), "256Mi")
	}

	// Check default (limits)
	cpuLim := item.Default[corev1.ResourceCPU]
	if cpuLim.String() != "2" {
		t.Errorf("Default CPU = %q, want %q", cpuLim.String(), "2")
	}
	memLim := item.Default[corev1.ResourceMemory]
	if memLim.String() != "4Gi" {
		t.Errorf("Default Memory = %q, want %q", memLim.String(), "4Gi")
	}

	// Verify owner reference
	if len(lr.OwnerReferences) != 1 {
		t.Fatalf("Expected 1 owner reference, got %d", len(lr.OwnerReferences))
	}
	if lr.OwnerReferences[0].Kind != "ProjectSettings" {
		t.Errorf("OwnerReference Kind = %q, want %q", lr.OwnerReferences[0].Kind, "ProjectSettings")
	}
}

func TestEnsureLimitRange_Idempotent(t *testing.T) {
	setupProjectSettingsTestClient()

	namespace := "test-namespace"
	owner := testProjectSettingsOwner(namespace)

	// First call
	err := ensureLimitRange(namespace, owner)
	if err != nil {
		t.Fatalf("First ensureLimitRange() returned error: %v", err)
	}

	// Second call — should not error
	err = ensureLimitRange(namespace, owner)
	if err != nil {
		t.Fatalf("Second ensureLimitRange() returned error: %v", err)
	}

	ctx := context.Background()

	// Verify LimitRange still exists
	_, err = config.K8sClient.CoreV1().LimitRanges(namespace).Get(ctx, "ambient-default-limits", metav1.GetOptions{})
	if err != nil {
		t.Errorf("LimitRange should still exist: %v", err)
	}
}

func TestEnsureLimitRange_MultipleNamespaces(t *testing.T) {
	setupProjectSettingsTestClient()

	namespaces := []string{"project-alpha", "project-beta"}

	for _, ns := range namespaces {
		err := ensureLimitRange(ns, testProjectSettingsOwner(ns))
		if err != nil {
			t.Fatalf("ensureLimitRange(%q) returned error: %v", ns, err)
		}
	}

	ctx := context.Background()

	for _, ns := range namespaces {
		lr, err := config.K8sClient.CoreV1().LimitRanges(ns).Get(ctx, "ambient-default-limits", metav1.GetOptions{})
		if err != nil {
			t.Errorf("LimitRange should exist in namespace %q: %v", ns, err)
			continue
		}
		if len(lr.Spec.Limits) != 1 {
			t.Errorf("Namespace %q: expected 1 limit item, got %d", ns, len(lr.Spec.Limits))
		}
	}
}

func TestEnsureSessionTriggerRBAC_CreatesAllResources(t *testing.T) {
	setupProjectSettingsTestClient()

	namespace := "test-namespace"

	err := ensureSessionTriggerRBAC(namespace, testProjectSettingsOwner(namespace))
	if err != nil {
		t.Fatalf("ensureSessionTriggerRBAC() returned error: %v", err)
	}

	ctx := context.Background()

	// Verify ServiceAccount was created
	sa, err := config.K8sClient.CoreV1().ServiceAccounts(namespace).Get(ctx, "ambient-session-trigger", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get ServiceAccount: %v", err)
	}
	if sa.Name != "ambient-session-trigger" {
		t.Errorf("ServiceAccount name = %q, want %q", sa.Name, "ambient-session-trigger")
	}
	if sa.Labels["ambient-code.io/managed"] != "true" {
		t.Errorf("ServiceAccount missing managed label")
	}

	// Verify Role was created
	role, err := config.K8sClient.RbacV1().Roles(namespace).Get(ctx, "ambient-session-trigger", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get Role: %v", err)
	}
	if role.Name != "ambient-session-trigger" {
		t.Errorf("Role name = %q, want %q", role.Name, "ambient-session-trigger")
	}
	if role.Labels["ambient-code.io/managed"] != "true" {
		t.Errorf("Role missing managed label")
	}

	// Verify Role has correct rules
	if len(role.Rules) != 1 {
		t.Fatalf("Expected 1 policy rule, got %d", len(role.Rules))
	}
	rule := role.Rules[0]
	if len(rule.APIGroups) != 1 || rule.APIGroups[0] != "vteam.ambient-code" {
		t.Errorf("Rule APIGroups = %v, want [vteam.ambient-code]", rule.APIGroups)
	}
	if len(rule.Resources) != 1 || rule.Resources[0] != "agenticsessions" {
		t.Errorf("Rule Resources = %v, want [agenticsessions]", rule.Resources)
	}
	expectedVerbs := map[string]bool{"create": true, "get": true, "list": true}
	if len(rule.Verbs) != 3 {
		t.Errorf("Rule Verbs count = %d, want 3", len(rule.Verbs))
	}
	for _, verb := range rule.Verbs {
		if !expectedVerbs[verb] {
			t.Errorf("Unexpected verb %q in role rules", verb)
		}
	}

	// Verify RoleBinding was created
	rb, err := config.K8sClient.RbacV1().RoleBindings(namespace).Get(ctx, "ambient-session-trigger", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get RoleBinding: %v", err)
	}
	if rb.Name != "ambient-session-trigger" {
		t.Errorf("RoleBinding name = %q, want %q", rb.Name, "ambient-session-trigger")
	}
	if rb.RoleRef.Kind != "Role" {
		t.Errorf("RoleBinding RoleRef Kind = %q, want %q", rb.RoleRef.Kind, "Role")
	}
	if rb.RoleRef.Name != "ambient-session-trigger" {
		t.Errorf("RoleBinding RoleRef Name = %q, want %q", rb.RoleRef.Name, "ambient-session-trigger")
	}
	if len(rb.Subjects) != 1 {
		t.Fatalf("Expected 1 subject, got %d", len(rb.Subjects))
	}
	if rb.Subjects[0].Kind != "ServiceAccount" {
		t.Errorf("Subject Kind = %q, want %q", rb.Subjects[0].Kind, "ServiceAccount")
	}
	if rb.Subjects[0].Name != "ambient-session-trigger" {
		t.Errorf("Subject Name = %q, want %q", rb.Subjects[0].Name, "ambient-session-trigger")
	}
	if rb.Subjects[0].Namespace != namespace {
		t.Errorf("Subject Namespace = %q, want %q", rb.Subjects[0].Namespace, namespace)
	}
}

func TestEnsureSessionTriggerRBAC_Idempotent(t *testing.T) {
	// Pre-populate the fake client with an existing ServiceAccount to simulate
	// resources already existing in the namespace.
	existingSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-session-trigger",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"ambient-code.io/managed": "true",
			},
		},
	}
	setupProjectSettingsTestClient(existingSA)

	namespace := "test-namespace"

	// First call — Role and RoleBinding don't exist yet, SA already exists
	owner := testProjectSettingsOwner(namespace)
	err := ensureSessionTriggerRBAC(namespace, owner)
	if err != nil {
		t.Fatalf("First ensureSessionTriggerRBAC() returned error: %v", err)
	}

	// Second call — all three resources already exist
	err = ensureSessionTriggerRBAC(namespace, owner)
	if err != nil {
		t.Fatalf("Second ensureSessionTriggerRBAC() returned error: %v", err)
	}

	ctx := context.Background()

	// Verify resources still exist and are correct after idempotent calls
	_, err = config.K8sClient.CoreV1().ServiceAccounts(namespace).Get(ctx, "ambient-session-trigger", metav1.GetOptions{})
	if err != nil {
		t.Errorf("ServiceAccount should still exist: %v", err)
	}

	_, err = config.K8sClient.RbacV1().Roles(namespace).Get(ctx, "ambient-session-trigger", metav1.GetOptions{})
	if err != nil {
		t.Errorf("Role should still exist: %v", err)
	}

	_, err = config.K8sClient.RbacV1().RoleBindings(namespace).Get(ctx, "ambient-session-trigger", metav1.GetOptions{})
	if err != nil {
		t.Errorf("RoleBinding should still exist: %v", err)
	}
}

func TestEnsureSessionTriggerRBAC_MultipleNamespaces(t *testing.T) {
	setupProjectSettingsTestClient()

	namespaces := []string{"project-alpha", "project-beta"}

	for _, ns := range namespaces {
		err := ensureSessionTriggerRBAC(ns, testProjectSettingsOwner(ns))
		if err != nil {
			t.Fatalf("ensureSessionTriggerRBAC(%q) returned error: %v", ns, err)
		}
	}

	ctx := context.Background()

	// Verify resources exist in both namespaces
	for _, ns := range namespaces {
		_, err := config.K8sClient.CoreV1().ServiceAccounts(ns).Get(ctx, "ambient-session-trigger", metav1.GetOptions{})
		if err != nil {
			t.Errorf("ServiceAccount should exist in namespace %q: %v", ns, err)
		}

		_, err = config.K8sClient.RbacV1().Roles(ns).Get(ctx, "ambient-session-trigger", metav1.GetOptions{})
		if err != nil {
			t.Errorf("Role should exist in namespace %q: %v", ns, err)
		}

		_, err = config.K8sClient.RbacV1().RoleBindings(ns).Get(ctx, "ambient-session-trigger", metav1.GetOptions{})
		if err != nil {
			t.Errorf("RoleBinding should exist in namespace %q: %v", ns, err)
		}
	}
}
