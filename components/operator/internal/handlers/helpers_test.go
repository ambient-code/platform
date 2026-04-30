package handlers

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestNeedsWorkflowReconciliation(t *testing.T) {
	tests := []struct {
		name    string
		session *unstructured.Unstructured
		want    bool
	}{
		{
			name:    "no activeWorkflow in spec",
			session: makeSession(nil, nil),
			want:    false,
		},
		{
			name: "activeWorkflow with empty gitUrl",
			session: makeSession(
				map[string]interface{}{"gitUrl": "", "branch": "main", "path": ".ambient/workflows/test"},
				nil,
			),
			want: false,
		},
		{
			name: "activeWorkflow present, no conditions at all",
			session: makeSession(
				map[string]interface{}{"gitUrl": "https://github.com/org/repo.git", "branch": "main", "path": ".ambient/workflows/test"},
				nil,
			),
			want: true,
		},
		{
			name: "WorkflowReconciled condition is True",
			session: makeSession(
				map[string]interface{}{"gitUrl": "https://github.com/org/repo.git", "branch": "main", "path": ".ambient/workflows/test"},
				[]interface{}{
					map[string]interface{}{
						"type":   "WorkflowReconciled",
						"status": "True",
						"reason": "Reconciled",
					},
				},
			),
			want: false,
		},
		{
			name: "WorkflowReconciled condition is False",
			session: makeSession(
				map[string]interface{}{"gitUrl": "https://github.com/org/repo.git", "branch": "main", "path": ".ambient/workflows/test"},
				[]interface{}{
					map[string]interface{}{
						"type":   "WorkflowReconciled",
						"status": "False",
						"reason": "UpdateFailed",
					},
				},
			),
			want: true,
		},
		{
			name: "other conditions present but no WorkflowReconciled",
			session: makeSession(
				map[string]interface{}{"gitUrl": "https://github.com/org/repo.git", "branch": "main", "path": ".ambient/workflows/test"},
				[]interface{}{
					map[string]interface{}{
						"type":   "Ready",
						"status": "True",
					},
				},
			),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NeedsWorkflowReconciliation(tt.session)
			if got != tt.want {
				t.Errorf("NeedsWorkflowReconciliation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func makeSession(activeWorkflow map[string]interface{}, conditions []interface{}) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]interface{}{
				"name":      "test-session",
				"namespace": "test-ns",
			},
			"spec": map[string]interface{}{},
		},
	}

	if activeWorkflow != nil {
		spec := obj.Object["spec"].(map[string]interface{})
		spec["activeWorkflow"] = activeWorkflow
	}

	if conditions != nil {
		obj.Object["status"] = map[string]interface{}{
			"conditions": conditions,
		}
	}

	return obj
}
