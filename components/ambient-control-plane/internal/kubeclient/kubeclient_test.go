package kubeclient

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
)

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	for _, gvk := range []schema.GroupVersionKind{
		{Group: "", Version: "v1", Kind: "Namespace"},
		{Group: "", Version: "v1", Kind: "Pod"},
		{Group: "", Version: "v1", Kind: "Service"},
		{Group: "", Version: "v1", Kind: "Secret"},
		{Group: "", Version: "v1", Kind: "ServiceAccount"},
		{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"},
		{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"},
	} {
		scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
		listGVK := gvk
		listGVK.Kind += "List"
		scheme.AddKnownTypeWithName(listGVK, &unstructured.UnstructuredList{})
	}
	return scheme
}

func newFakeKubeClientFull(objects ...runtime.Object) *KubeClient {
	fakeClient := dynamicfake.NewSimpleDynamicClient(newScheme(), objects...)
	return &KubeClient{
		dynamic: fakeClient,
		logger:  zerolog.Nop(),
	}
}

func newFakeClientWithTracker(objects ...runtime.Object) (*dynamicfake.FakeDynamicClient, *KubeClient) {
	fakeClient := dynamicfake.NewSimpleDynamicClient(newScheme(), objects...)
	kc := &KubeClient{dynamic: fakeClient, logger: zerolog.Nop()}
	return fakeClient, kc
}

func buildPod(namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"containers": []interface{}{
					map[string]interface{}{"name": "runner", "image": "ambient-runner:latest"},
				},
			},
		},
	}
}

func buildService(namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{"app": name},
				"ports":    []interface{}{map[string]interface{}{"port": int64(8080)}},
			},
		},
	}
}

func buildSecret(namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"data": map[string]interface{}{
				"token": "dGVzdA==",
			},
		},
	}
}

func buildServiceAccount(namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ServiceAccount",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
}

func buildRole(namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "Role",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"rules": []interface{}{
				map[string]interface{}{
					"apiGroups": []interface{}{""},
					"resources": []interface{}{"pods"},
					"verbs":     []interface{}{"get", "list"},
				},
			},
		},
	}
}

func buildNamespace(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": name,
			},
		},
	}
}

func buildRoleBinding(namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "RoleBinding",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"roleRef": map[string]interface{}{
				"apiGroup": "rbac.authorization.k8s.io",
				"kind":     "ClusterRole",
				"name":     "edit",
			},
			"subjects": []interface{}{
				map[string]interface{}{
					"kind":     "Group",
					"name":     "developers",
					"apiGroup": "rbac.authorization.k8s.io",
				},
			},
		},
	}
}

func TestGetNamespace_Found(t *testing.T) {
	ns := buildNamespace("my-project")
	kc := newFakeKubeClientFull(ns)

	got, err := kc.GetNamespace(context.Background(), "my-project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.GetName() != "my-project" {
		t.Errorf("expected name 'my-project', got %q", got.GetName())
	}
}

func TestGetNamespace_NotFound(t *testing.T) {
	kc := newFakeKubeClientFull()

	_, err := kc.GetNamespace(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent namespace")
	}
}

func TestCreateNamespace(t *testing.T) {
	kc := newFakeKubeClientFull()
	ns := buildNamespace("new-project")

	created, err := kc.CreateNamespace(context.Background(), ns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.GetName() != "new-project" {
		t.Errorf("expected name 'new-project', got %q", created.GetName())
	}

	got, err := kc.GetNamespace(context.Background(), "new-project")
	if err != nil {
		t.Fatalf("get after create failed: %v", err)
	}
	if got.GetName() != "new-project" {
		t.Errorf("round-trip name mismatch: %q", got.GetName())
	}
}

func TestUpdateNamespace_Labels(t *testing.T) {
	ns := buildNamespace("label-test")
	kc := newFakeKubeClientFull(ns)

	existing, _ := kc.GetNamespace(context.Background(), "label-test")
	existing.SetLabels(map[string]string{
		"ambient-code.io/managed":    "true",
		"ambient-code.io/project-id": "proj-123",
	})

	updated, err := kc.UpdateNamespace(context.Background(), existing)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	labels := updated.GetLabels()
	if labels["ambient-code.io/managed"] != "true" {
		t.Errorf("expected managed label, got %q", labels["ambient-code.io/managed"])
	}
	if labels["ambient-code.io/project-id"] != "proj-123" {
		t.Errorf("expected project-id label, got %q", labels["ambient-code.io/project-id"])
	}
}

func TestNamespaceGVR(t *testing.T) {
	if NamespaceGVR.Group != "" {
		t.Errorf("expected empty group, got %q", NamespaceGVR.Group)
	}
	if NamespaceGVR.Version != "v1" {
		t.Errorf("expected version 'v1', got %q", NamespaceGVR.Version)
	}
	if NamespaceGVR.Resource != "namespaces" {
		t.Errorf("expected resource 'namespaces', got %q", NamespaceGVR.Resource)
	}
}

func TestRoleBindingGVR(t *testing.T) {
	if RoleBindingGVR.Group != "rbac.authorization.k8s.io" {
		t.Errorf("expected group 'rbac.authorization.k8s.io', got %q", RoleBindingGVR.Group)
	}
	if RoleBindingGVR.Version != "v1" {
		t.Errorf("expected version 'v1', got %q", RoleBindingGVR.Version)
	}
	if RoleBindingGVR.Resource != "rolebindings" {
		t.Errorf("expected resource 'rolebindings', got %q", RoleBindingGVR.Resource)
	}
}

func TestCreateRoleBinding(t *testing.T) {
	kc := newFakeKubeClientFull()
	rb := buildRoleBinding("my-project", "ambient-devs-edit")

	created, err := kc.CreateRoleBinding(context.Background(), "my-project", rb)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.GetName() != "ambient-devs-edit" {
		t.Errorf("expected name 'ambient-devs-edit', got %q", created.GetName())
	}
}

func TestGetRoleBinding(t *testing.T) {
	rb := buildRoleBinding("my-project", "ambient-devs-edit")
	kc := newFakeKubeClientFull(rb)

	got, err := kc.GetRoleBinding(context.Background(), "my-project", "ambient-devs-edit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.GetName() != "ambient-devs-edit" {
		t.Errorf("expected name 'ambient-devs-edit', got %q", got.GetName())
	}
}

func TestGetRoleBinding_NotFound(t *testing.T) {
	kc := newFakeKubeClientFull()

	_, err := kc.GetRoleBinding(context.Background(), "my-project", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent rolebinding")
	}
}

func TestUpdateRoleBinding(t *testing.T) {
	rb := buildRoleBinding("my-project", "ambient-devs-edit")
	kc := newFakeKubeClientFull(rb)

	existing, _ := kc.GetRoleBinding(context.Background(), "my-project", "ambient-devs-edit")
	_ = unstructured.SetNestedField(existing.Object, "admin", "roleRef", "name")

	updated, err := kc.UpdateRoleBinding(context.Background(), "my-project", existing)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	role, _, _ := unstructured.NestedString(updated.Object, "roleRef", "name")
	if role != "admin" {
		t.Errorf("expected role 'admin', got %q", role)
	}
}

func TestDeleteRoleBinding(t *testing.T) {
	rb := buildRoleBinding("my-project", "ambient-devs-edit")
	kc := newFakeKubeClientFull(rb)

	err := kc.DeleteRoleBinding(context.Background(), "my-project", "ambient-devs-edit")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, err = kc.GetRoleBinding(context.Background(), "my-project", "ambient-devs-edit")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestDeleteRoleBinding_NotFound(t *testing.T) {
	kc := newFakeKubeClientFull()

	err := kc.DeleteRoleBinding(context.Background(), "my-project", "nonexistent")
	if err == nil {
		t.Fatal("expected error deleting nonexistent rolebinding")
	}
}

func TestCreateAndGetPod(t *testing.T) {
	kc := newFakeKubeClientFull()
	pod := buildPod("my-project", "runner-pod")

	created, err := kc.CreatePod(context.Background(), pod)
	if err != nil {
		t.Fatalf("create pod failed: %v", err)
	}
	if created.GetName() != "runner-pod" {
		t.Errorf("expected name 'runner-pod', got %q", created.GetName())
	}

	got, err := kc.GetPod(context.Background(), "my-project", "runner-pod")
	if err != nil {
		t.Fatalf("get pod failed: %v", err)
	}
	if got.GetNamespace() != "my-project" {
		t.Errorf("expected namespace 'my-project', got %q", got.GetNamespace())
	}
}

func TestDeletePod(t *testing.T) {
	pod := buildPod("my-project", "runner-pod")
	kc := newFakeKubeClientFull(pod)

	err := kc.DeletePod(context.Background(), "my-project", "runner-pod", nil)
	if err != nil {
		t.Fatalf("delete pod failed: %v", err)
	}

	_, err = kc.GetPod(context.Background(), "my-project", "runner-pod")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestCreateAndGetService(t *testing.T) {
	kc := newFakeKubeClientFull()
	svc := buildService("my-project", "runner-svc")

	created, err := kc.CreateService(context.Background(), svc)
	if err != nil {
		t.Fatalf("create service failed: %v", err)
	}
	if created.GetName() != "runner-svc" {
		t.Errorf("expected name 'runner-svc', got %q", created.GetName())
	}

	got, err := kc.GetService(context.Background(), "my-project", "runner-svc")
	if err != nil {
		t.Fatalf("get service failed: %v", err)
	}
	if got.GetName() != "runner-svc" {
		t.Errorf("round-trip name mismatch: %q", got.GetName())
	}
}

func TestCreateAndGetSecret(t *testing.T) {
	kc := newFakeKubeClientFull()
	secret := buildSecret("my-project", "runner-token")

	created, err := kc.CreateSecret(context.Background(), secret)
	if err != nil {
		t.Fatalf("create secret failed: %v", err)
	}
	if created.GetName() != "runner-token" {
		t.Errorf("expected name 'runner-token', got %q", created.GetName())
	}

	got, err := kc.GetSecret(context.Background(), "my-project", "runner-token")
	if err != nil {
		t.Fatalf("get secret failed: %v", err)
	}
	if got.GetName() != "runner-token" {
		t.Errorf("round-trip name mismatch: %q", got.GetName())
	}
}

func TestCreateAndGetServiceAccount(t *testing.T) {
	kc := newFakeKubeClientFull()
	sa := buildServiceAccount("my-project", "runner-sa")

	created, err := kc.CreateServiceAccount(context.Background(), sa)
	if err != nil {
		t.Fatalf("create serviceaccount failed: %v", err)
	}
	if created.GetName() != "runner-sa" {
		t.Errorf("expected name 'runner-sa', got %q", created.GetName())
	}

	got, err := kc.GetServiceAccount(context.Background(), "my-project", "runner-sa")
	if err != nil {
		t.Fatalf("get serviceaccount failed: %v", err)
	}
	if got.GetName() != "runner-sa" {
		t.Errorf("round-trip name mismatch: %q", got.GetName())
	}
}

func TestCreateAndGetRole(t *testing.T) {
	kc := newFakeKubeClientFull()
	role := buildRole("my-project", "runner-role")

	created, err := kc.CreateRole(context.Background(), role)
	if err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	if created.GetName() != "runner-role" {
		t.Errorf("expected name 'runner-role', got %q", created.GetName())
	}

	got, err := kc.GetRole(context.Background(), "my-project", "runner-role")
	if err != nil {
		t.Fatalf("get role failed: %v", err)
	}
	if got.GetName() != "runner-role" {
		t.Errorf("round-trip name mismatch: %q", got.GetName())
	}
}

func TestGVRConstants(t *testing.T) {
	cases := []struct {
		name     string
		gvr      schema.GroupVersionResource
		group    string
		version  string
		resource string
	}{
		{"Namespace", NamespaceGVR, "", "v1", "namespaces"},
		{"Pod", PodGVR, "", "v1", "pods"},
		{"Service", ServiceGVR, "", "v1", "services"},
		{"Secret", SecretGVR, "", "v1", "secrets"},
		{"ServiceAccount", ServiceAccountGVR, "", "v1", "serviceaccounts"},
		{"RoleBinding", RoleBindingGVR, "rbac.authorization.k8s.io", "v1", "rolebindings"},
		{"Role", RoleGVR, "rbac.authorization.k8s.io", "v1", "roles"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.gvr.Group != tc.group {
				t.Errorf("group: expected %q got %q", tc.group, tc.gvr.Group)
			}
			if tc.gvr.Version != tc.version {
				t.Errorf("version: expected %q got %q", tc.version, tc.gvr.Version)
			}
			if tc.gvr.Resource != tc.resource {
				t.Errorf("resource: expected %q got %q", tc.resource, tc.gvr.Resource)
			}
		})
	}
}

func assertDeleteCollectionAction(t *testing.T, fake *dynamicfake.FakeDynamicClient, wantResource, wantNamespace, wantSelector string) {
	// wantSelector must be a simple "key=value" expression; set-based selectors
	// may not round-trip correctly through Labels.String().
	t.Helper()
	actions := fake.Actions()
	if len(actions) == 0 {
		t.Fatal("expected a delete-collection action, got none")
	}
	last := actions[len(actions)-1]
	dc, ok := last.(k8stesting.DeleteCollectionAction)
	if !ok {
		t.Fatalf("expected DeleteCollectionAction, got %T", last)
	}
	if dc.GetResource().Resource != wantResource {
		t.Errorf("resource: expected %q got %q", wantResource, dc.GetResource().Resource)
	}
	if dc.GetNamespace() != wantNamespace {
		t.Errorf("namespace: expected %q got %q", wantNamespace, dc.GetNamespace())
	}
	if got := dc.GetListRestrictions().Labels.String(); got != wantSelector {
		t.Errorf("label selector: expected %q got %q", wantSelector, got)
	}
}

func TestDeletePodsByLabel(t *testing.T) {
	fake, kc := newFakeClientWithTracker(buildPod("ns", "pod-1"))
	if err := kc.DeletePodsByLabel(context.Background(), "ns", "app=runner"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertDeleteCollectionAction(t, fake, "pods", "ns", "app=runner")
}

func TestDeleteServicesByLabel(t *testing.T) {
	fake, kc := newFakeClientWithTracker(buildService("ns", "svc-1"))
	if err := kc.DeleteServicesByLabel(context.Background(), "ns", "app=runner"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertDeleteCollectionAction(t, fake, "services", "ns", "app=runner")
}

func TestDeleteSecretsByLabel(t *testing.T) {
	fake, kc := newFakeClientWithTracker(buildSecret("ns", "secret-1"))
	if err := kc.DeleteSecretsByLabel(context.Background(), "ns", "app=runner"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertDeleteCollectionAction(t, fake, "secrets", "ns", "app=runner")
}

func TestDeleteServiceAccountsByLabel(t *testing.T) {
	fake, kc := newFakeClientWithTracker(buildServiceAccount("ns", "sa-1"))
	if err := kc.DeleteServiceAccountsByLabel(context.Background(), "ns", "app=runner"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertDeleteCollectionAction(t, fake, "serviceaccounts", "ns", "app=runner")
}

func TestDeleteRolesByLabel(t *testing.T) {
	fake, kc := newFakeClientWithTracker(buildRole("ns", "role-1"))
	if err := kc.DeleteRolesByLabel(context.Background(), "ns", "app=runner"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertDeleteCollectionAction(t, fake, "roles", "ns", "app=runner")
}

func TestDeleteRoleBindingsByLabel(t *testing.T) {
	fake, kc := newFakeClientWithTracker(buildRoleBinding("ns", "rb-1"))
	if err := kc.DeleteRoleBindingsByLabel(context.Background(), "ns", "app=runner"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertDeleteCollectionAction(t, fake, "rolebindings", "ns", "app=runner")
}
