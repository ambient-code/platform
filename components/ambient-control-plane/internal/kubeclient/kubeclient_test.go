package kubeclient

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func newFakeKubeClient(namespace string, objects ...runtime.Object) *KubeClient {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "vteam.ambient-code", Version: "v1alpha1", Kind: "AgenticSession"},
		&unstructured.Unstructured{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "vteam.ambient-code", Version: "v1alpha1", Kind: "AgenticSessionList"},
		&unstructured.UnstructuredList{},
	)

	fakeClient := dynamicfake.NewSimpleDynamicClient(scheme, objects...)

	return &KubeClient{
		dynamic:   fakeClient,
		namespace: namespace,
		logger:    zerolog.Nop(),
	}
}

func buildAgenticSession(namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"displayName":   name,
				"initialPrompt": "test prompt",
			},
		},
	}
}

func TestGetAgenticSession_Found(t *testing.T) {
	ns := "test-ns"
	session := buildAgenticSession(ns, "my-session")
	kc := newFakeKubeClient(ns, session)

	cr, err := kc.GetAgenticSession(context.Background(), "my-session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cr.GetName() != "my-session" {
		t.Errorf("expected name 'my-session', got %q", cr.GetName())
	}
}

func TestGetAgenticSession_NotFound(t *testing.T) {
	kc := newFakeKubeClient("test-ns")

	_, err := kc.GetAgenticSession(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session, got nil")
	}
}

func TestListAgenticSessions(t *testing.T) {
	ns := "test-ns"
	s1 := buildAgenticSession(ns, "session-1")
	s2 := buildAgenticSession(ns, "session-2")
	s3 := buildAgenticSession(ns, "session-3")
	kc := newFakeKubeClient(ns, s1, s2, s3)

	list, err := kc.ListAgenticSessions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(list.Items))
	}

	names := map[string]bool{}
	for _, item := range list.Items {
		names[item.GetName()] = true
	}
	for _, expected := range []string{"session-1", "session-2", "session-3"} {
		if !names[expected] {
			t.Errorf("expected session %q in list", expected)
		}
	}
}

func TestNamespace_ReturnsConfigured(t *testing.T) {
	kc := newFakeKubeClient("ambient-code")
	if got := kc.Namespace(); got != "ambient-code" {
		t.Errorf("Namespace() = %q, want %q", got, "ambient-code")
	}
}

func TestGetAgenticSession_WrongNamespace(t *testing.T) {
	session := buildAgenticSession("other-ns", "my-session")
	kc := newFakeKubeClient("test-ns", session)

	_, err := kc.GetAgenticSession(context.Background(), "my-session")
	if err == nil {
		t.Fatal("expected error when session is in a different namespace")
	}
}

func TestListAgenticSessions_Empty(t *testing.T) {
	kc := newFakeKubeClient("test-ns")

	list, err := kc.ListAgenticSessions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(list.Items))
	}
}

func TestAgenticSessionGVR(t *testing.T) {
	if AgenticSessionGVR.Group != "vteam.ambient-code" {
		t.Errorf("expected group 'vteam.ambient-code', got %q", AgenticSessionGVR.Group)
	}
	if AgenticSessionGVR.Version != "v1alpha1" {
		t.Errorf("expected version 'v1alpha1', got %q", AgenticSessionGVR.Version)
	}
	if AgenticSessionGVR.Resource != "agenticsessions" {
		t.Errorf("expected resource 'agenticsessions', got %q", AgenticSessionGVR.Resource)
	}
}

func TestGetAgenticSession_VerifiesSpec(t *testing.T) {
	ns := "test-ns"
	session := buildAgenticSession(ns, "detailed-session")
	kc := newFakeKubeClient(ns, session)

	cr, err := kc.GetAgenticSession(context.Background(), "detailed-session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	displayName, found, err := unstructured.NestedString(cr.Object, "spec", "displayName")
	if err != nil || !found {
		t.Fatal("expected spec.displayName to exist")
	}
	if displayName != "detailed-session" {
		t.Errorf("expected displayName 'detailed-session', got %q", displayName)
	}

	prompt, found, err := unstructured.NestedString(cr.Object, "spec", "initialPrompt")
	if err != nil || !found {
		t.Fatal("expected spec.initialPrompt to exist")
	}
	if prompt != "test prompt" {
		t.Errorf("expected initialPrompt 'test prompt', got %q", prompt)
	}
}

func TestListAgenticSessions_NamespaceIsolation(t *testing.T) {
	s1 := buildAgenticSession("ns-a", "session-in-a")
	s2 := buildAgenticSession("ns-b", "session-in-b")

	fakeClient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), s1, s2)
	kc := &KubeClient{
		dynamic:   fakeClient,
		namespace: "ns-a",
		logger:    zerolog.Nop(),
	}

	list, err := kc.ListAgenticSessions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, item := range list.Items {
		if item.GetNamespace() != "ns-a" {
			t.Errorf("expected only ns-a items, got namespace %q for %q", item.GetNamespace(), item.GetName())
		}
	}
}

func TestGetAgenticSession_ReturnsFullObject(t *testing.T) {
	ns := "test-ns"
	session := buildAgenticSession(ns, "full-session")

	spec := session.Object["spec"].(map[string]interface{})
	spec["interactive"] = true
	spec["timeout"] = int64(3600)

	kc := newFakeKubeClient(ns, session)

	cr, err := kc.GetAgenticSession(context.Background(), "full-session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	interactive, found, _ := unstructured.NestedBool(cr.Object, "spec", "interactive")
	if !found || !interactive {
		t.Error("expected spec.interactive to be true")
	}

	labels := cr.GetLabels()
	_ = labels

	gvk := cr.GroupVersionKind()
	if gvk.Kind != "AgenticSession" {
		t.Errorf("expected kind 'AgenticSession', got %q", gvk.Kind)
	}
}

func TestListAgenticSessions_ChecksGVR(t *testing.T) {
	ns := "test-ns"
	kc := newFakeKubeClient(ns)

	_, err := kc.dynamic.Resource(AgenticSessionGVR).Namespace(ns).List(
		context.Background(),
		metav1.ListOptions{},
	)
	if err != nil {
		t.Fatalf("listing via GVR should not error on empty: %v", err)
	}
}

func TestCreateAgenticSession(t *testing.T) {
	ns := "test-ns"
	kc := newFakeKubeClient(ns)

	session := buildAgenticSession(ns, "new-session")
	created, err := kc.CreateAgenticSession(context.Background(), session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.GetName() != "new-session" {
		t.Errorf("expected name 'new-session', got %q", created.GetName())
	}

	got, err := kc.GetAgenticSession(context.Background(), "new-session")
	if err != nil {
		t.Fatalf("get after create failed: %v", err)
	}
	if got.GetName() != "new-session" {
		t.Errorf("round-trip name mismatch: %q", got.GetName())
	}
}

func TestUpdateAgenticSession(t *testing.T) {
	ns := "test-ns"
	session := buildAgenticSession(ns, "update-me")
	kc := newFakeKubeClient(ns, session)

	existing, err := kc.GetAgenticSession(context.Background(), "update-me")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	unstructured.SetNestedField(existing.Object, "updated prompt", "spec", "initialPrompt")
	updated, err := kc.UpdateAgenticSession(context.Background(), existing)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	prompt, _, _ := unstructured.NestedString(updated.Object, "spec", "initialPrompt")
	if prompt != "updated prompt" {
		t.Errorf("expected updated prompt, got %q", prompt)
	}

	reread, err := kc.GetAgenticSession(context.Background(), "update-me")
	if err != nil {
		t.Fatalf("re-read failed: %v", err)
	}
	prompt2, _, _ := unstructured.NestedString(reread.Object, "spec", "initialPrompt")
	if prompt2 != "updated prompt" {
		t.Errorf("re-read prompt mismatch: %q", prompt2)
	}
}

func TestDeleteAgenticSession(t *testing.T) {
	ns := "test-ns"
	session := buildAgenticSession(ns, "delete-me")
	kc := newFakeKubeClient(ns, session)

	_, err := kc.GetAgenticSession(context.Background(), "delete-me")
	if err != nil {
		t.Fatalf("session should exist before delete: %v", err)
	}

	err = kc.DeleteAgenticSession(context.Background(), "delete-me")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, err = kc.GetAgenticSession(context.Background(), "delete-me")
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestDeleteAgenticSession_NotFound(t *testing.T) {
	ns := "test-ns"
	kc := newFakeKubeClient(ns)

	err := kc.DeleteAgenticSession(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error deleting nonexistent session")
	}
}

func TestCreateAgenticSession_RoundTripsSpec(t *testing.T) {
	ns := "test-ns"
	kc := newFakeKubeClient(ns)

	session := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "full-spec-session",
				"namespace": ns,
			},
			"spec": map[string]interface{}{
				"displayName":   "Full Spec Test",
				"initialPrompt": "build something",
				"interactive":   true,
				"timeout":       int64(600),
				"project":       "my-project",
				"llmSettings": map[string]interface{}{
					"model":       "claude-3-7-sonnet",
					"temperature": float64(0.5),
					"maxTokens":   int64(8000),
				},
				"botAccount": map[string]interface{}{
					"name": "bot-1",
				},
			},
		},
	}

	_, err := kc.CreateAgenticSession(context.Background(), session)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	got, err := kc.GetAgenticSession(context.Background(), "full-spec-session")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	model, _, _ := unstructured.NestedString(got.Object, "spec", "llmSettings", "model")
	if model != "claude-3-7-sonnet" {
		t.Errorf("expected model 'claude-3-7-sonnet', got %q", model)
	}

	botName, _, _ := unstructured.NestedString(got.Object, "spec", "botAccount", "name")
	if botName != "bot-1" {
		t.Errorf("expected bot name 'bot-1', got %q", botName)
	}

	project, _, _ := unstructured.NestedString(got.Object, "spec", "project")
	if project != "my-project" {
		t.Errorf("expected project 'my-project', got %q", project)
	}
}

func newFakeKubeClientWithNamespaces(namespace string, objects ...runtime.Object) *KubeClient {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "vteam.ambient-code", Version: "v1alpha1", Kind: "AgenticSession"},
		&unstructured.Unstructured{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "vteam.ambient-code", Version: "v1alpha1", Kind: "AgenticSessionList"},
		&unstructured.UnstructuredList{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"},
		&unstructured.Unstructured{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "", Version: "v1", Kind: "NamespaceList"},
		&unstructured.UnstructuredList{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"},
		&unstructured.Unstructured{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBindingList"},
		&unstructured.UnstructuredList{},
	)
	fakeClient := dynamicfake.NewSimpleDynamicClient(scheme, objects...)
	return &KubeClient{
		dynamic:   fakeClient,
		namespace: namespace,
		logger:    zerolog.Nop(),
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
	kc := newFakeKubeClientWithNamespaces("default", ns)

	got, err := kc.GetNamespace(context.Background(), "my-project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.GetName() != "my-project" {
		t.Errorf("expected name 'my-project', got %q", got.GetName())
	}
}

func TestGetNamespace_NotFound(t *testing.T) {
	kc := newFakeKubeClientWithNamespaces("default")

	_, err := kc.GetNamespace(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent namespace")
	}
}

func TestCreateNamespace(t *testing.T) {
	kc := newFakeKubeClientWithNamespaces("default")
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
	kc := newFakeKubeClientWithNamespaces("default", ns)

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
	kc := newFakeKubeClientWithNamespaces("default")
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
	kc := newFakeKubeClientWithNamespaces("default", rb)

	got, err := kc.GetRoleBinding(context.Background(), "my-project", "ambient-devs-edit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.GetName() != "ambient-devs-edit" {
		t.Errorf("expected name 'ambient-devs-edit', got %q", got.GetName())
	}
}

func TestGetRoleBinding_NotFound(t *testing.T) {
	kc := newFakeKubeClientWithNamespaces("default")

	_, err := kc.GetRoleBinding(context.Background(), "my-project", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent rolebinding")
	}
}

func TestUpdateRoleBinding(t *testing.T) {
	rb := buildRoleBinding("my-project", "ambient-devs-edit")
	kc := newFakeKubeClientWithNamespaces("default", rb)

	existing, _ := kc.GetRoleBinding(context.Background(), "my-project", "ambient-devs-edit")
	unstructured.SetNestedField(existing.Object, "admin", "roleRef", "name")

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
	kc := newFakeKubeClientWithNamespaces("default", rb)

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
	kc := newFakeKubeClientWithNamespaces("default")

	err := kc.DeleteRoleBinding(context.Background(), "my-project", "nonexistent")
	if err == nil {
		t.Fatal("expected error deleting nonexistent rolebinding")
	}
}
