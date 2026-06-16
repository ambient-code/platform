package reconciler

import (
	"context"
	"testing"

	"github.com/ambient-code/platform/components/ambient-control-plane/internal/kubeclient"
	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

type stubProvisioner struct{}

func (s *stubProvisioner) NamespaceName(projectID string) string { return projectID }
func (s *stubProvisioner) ProvisionNamespace(_ context.Context, _ string, _ map[string]string) error {
	return nil
}
func (s *stubProvisioner) DeprovisionNamespace(_ context.Context, _ string) error { return nil }

func newFakeKubeClient(objects ...runtime.Object) *kubeclient.KubeClient {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"},
		&unstructured.Unstructured{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"},
		&unstructured.Unstructured{},
	)
	dynClient := fake.NewSimpleDynamicClient(scheme, objects...)
	return kubeclient.NewFromDynamic(dynClient, zerolog.Nop())
}

func newTestProjectReconciler(kube *kubeclient.KubeClient) *ProjectReconciler {
	return &ProjectReconciler{
		kube:               kube,
		provisioner:        &stubProvisioner{},
		cpRuntimeNamespace: "ambient-system",
		logger:             zerolog.Nop(),
	}
}

func TestControlPlaneRBACRules_ContainsAllAPIGroups(t *testing.T) {
	r := newTestProjectReconciler(nil)
	rules := r.controlPlaneRBACRules()

	expectedGroups := map[string]bool{
		"":                      false,
		"rbac.authorization.k8s.io": false,
		"build.openshift.io":    false,
		"image.openshift.io":    false,
		"route.openshift.io":    false,
	}

	for _, rule := range rules {
		ruleMap := rule.(map[string]interface{})
		apiGroups := ruleMap["apiGroups"].([]interface{})
		for _, g := range apiGroups {
			expectedGroups[g.(string)] = true
		}
	}

	for group, found := range expectedGroups {
		if !found {
			t.Errorf("expected API group %q in RBAC rules, but not found", group)
		}
	}

	if len(rules) != 6 {
		t.Errorf("expected 6 rule entries, got %d", len(rules))
	}
}

func TestControlPlaneRBACRules_OpenShiftBuildResources(t *testing.T) {
	r := newTestProjectReconciler(nil)
	rules := r.controlPlaneRBACRules()

	var buildRule map[string]interface{}
	for _, rule := range rules {
		ruleMap := rule.(map[string]interface{})
		groups := ruleMap["apiGroups"].([]interface{})
		if len(groups) > 0 && groups[0] == "build.openshift.io" {
			buildRule = ruleMap
			break
		}
	}

	if buildRule == nil {
		t.Fatal("build.openshift.io rule not found")
	}

	resources := buildRule["resources"].([]interface{})
	expected := map[string]bool{
		"buildconfigs":             false,
		"buildconfigs/instantiate": false,
		"builds":                   false,
		"builds/log":              false,
	}
	for _, r := range resources {
		expected[r.(string)] = true
	}
	for res, found := range expected {
		if !found {
			t.Errorf("expected resource %q in build.openshift.io rule", res)
		}
	}
}

func TestEnsureControlPlaneRBAC_CreatesRoleAndBinding(t *testing.T) {
	kube := newFakeKubeClient()
	r := newTestProjectReconciler(kube)

	project := types.Project{ObjectReference: types.ObjectReference{ID: "test-project"}, Name: "Test"}

	if err := r.ensureControlPlaneRBAC(context.Background(), project); err != nil {
		t.Fatalf("ensureControlPlaneRBAC (create) failed: %v", err)
	}

	role, err := kube.GetRole(context.Background(), "test-project", "ambient-control-plane-project-manager")
	if err != nil {
		t.Fatalf("expected Role to exist after create: %v", err)
	}

	rules, found, err := unstructured.NestedSlice(role.Object, "rules")
	if err != nil || !found {
		t.Fatalf("expected rules in created Role: found=%v err=%v", found, err)
	}
	if len(rules) != 6 {
		t.Errorf("expected 6 rules in created Role, got %d", len(rules))
	}

	_, err = kube.GetRoleBinding(context.Background(), "test-project", "ambient-control-plane-project-manager")
	if err != nil {
		t.Fatalf("expected RoleBinding to exist after create: %v", err)
	}
}

func TestEnsureControlPlaneRBAC_UpdatesExistingRole(t *testing.T) {
	existingRole := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "Role",
			"metadata": map[string]interface{}{
				"name":      "ambient-control-plane-project-manager",
				"namespace": "test-project",
			},
			"rules": []interface{}{
				map[string]interface{}{
					"apiGroups": []interface{}{""},
					"resources": []interface{}{"secrets"},
					"verbs":     []interface{}{"get"},
				},
			},
		},
	}

	kube := newFakeKubeClient(existingRole)
	r := newTestProjectReconciler(kube)

	project := types.Project{ObjectReference: types.ObjectReference{ID: "test-project"}, Name: "Test"}

	if err := r.ensureControlPlaneRBAC(context.Background(), project); err != nil {
		t.Fatalf("ensureControlPlaneRBAC (update) failed: %v", err)
	}

	role, err := kube.GetRole(context.Background(), "test-project", "ambient-control-plane-project-manager")
	if err != nil {
		t.Fatalf("expected Role to exist after update: %v", err)
	}

	rules, found, err := unstructured.NestedSlice(role.Object, "rules")
	if err != nil || !found {
		t.Fatalf("expected rules in updated Role: found=%v err=%v", found, err)
	}
	if len(rules) != 6 {
		t.Errorf("expected 6 rules after update (was 1), got %d", len(rules))
	}
}
