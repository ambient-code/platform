package handlers

import (
	"context"
	"testing"
	"time"

	"ambient-code-operator/internal/config"
	"ambient-code-operator/internal/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

// newSessionObj builds an unstructured AgenticSession for testing.
// Fields that are nil/zero are omitted from the object.
func newSessionObj(name, namespace string, opts ...func(map[string]any)) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "AgenticSession",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
	for _, opt := range opts {
		opt(obj.Object)
	}
	return obj
}

func withCreationTimestamp(ts time.Time) func(map[string]any) {
	return func(obj map[string]any) {
		meta := obj["metadata"].(map[string]any)
		meta["creationTimestamp"] = ts.UTC().Format(time.RFC3339)
	}
}

func withSpec(fields map[string]any) func(map[string]any) {
	return func(obj map[string]any) {
		obj["spec"] = fields
	}
}

func withStatus(fields map[string]any) func(map[string]any) {
	return func(obj map[string]any) {
		obj["status"] = fields
	}
}

// gvrForKind returns the GVR matching the given Kind string, or an empty GVR.
func gvrForKind(kind string) schema.GroupVersionResource {
	switch kind {
	case "AgenticSession":
		return types.GetAgenticSessionResource()
	case "ProjectSettings":
		return types.GetProjectSettingsResource()
	default:
		return schema.GroupVersionResource{}
	}
}

// setupFakeDynamicClient creates a fake dynamic client and assigns it to config.DynamicClient.
// Unstructured objects are pre-populated via Create() because the scheme-based tracker
// cannot resolve GVR from unstructured objects passed through the constructor.
func setupFakeDynamicClient(objects ...*unstructured.Unstructured) {
	scheme := runtime.NewScheme()

	gvrToListKind := map[schema.GroupVersionResource]string{
		types.GetAgenticSessionResource():  "AgenticSessionList",
		types.GetProjectSettingsResource(): "ProjectSettingsList",
	}
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind)
	config.DynamicClient = client

	for _, obj := range objects {
		gvr := gvrForKind(obj.GetKind())
		if gvr.Resource == "" {
			continue
		}
		// Best-effort: errors here will surface as test failures later.
		_, _ = client.Resource(gvr).Namespace(obj.GetNamespace()).Create(
			context.Background(), obj, metav1.CreateOptions{},
		)
	}
}

// resetTimeoutCache clears the project timeout cache between tests.
func resetTimeoutCache() {
	psTimeoutCache.mu.Lock()
	psTimeoutCache.entries = make(map[string]projectTimeoutEntry)
	psTimeoutCache.mu.Unlock()
}

// ==============================
// shouldAutoStop tests
// ==============================

func TestShouldAutoStop(t *testing.T) {
	// Set up a fake dynamic client so resolveInactivityTimeout can query project settings.
	// For most subtests we set spec.inactivityTimeout on the session itself, so this
	// is only needed as a fallback for the "uses default timeout" subtest.
	setupFakeDynamicClient()

	t.Run("no status returns false", func(t *testing.T) {
		resetTimeoutCache()
		obj := newSessionObj("s1", "ns1",
			withSpec(map[string]any{"inactivityTimeout": int64(300)}),
		)
		if shouldAutoStop(obj) {
			t.Error("expected false when status is missing")
		}
	})

	t.Run("non-running phase returns false", func(t *testing.T) {
		resetTimeoutCache()
		for _, phase := range []string{"Pending", "Creating", "Stopping", "Stopped", "Completed"} {
			obj := newSessionObj("s1", "ns1",
				withSpec(map[string]any{"inactivityTimeout": int64(300)}),
				withStatus(map[string]any{
					"phase":            phase,
					"lastActivityTime": time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339),
				}),
			)
			if shouldAutoStop(obj) {
				t.Errorf("expected false for phase %q", phase)
			}
		}
	})

	t.Run("timeout zero disables auto-stop", func(t *testing.T) {
		resetTimeoutCache()
		obj := newSessionObj("s1", "ns1",
			withSpec(map[string]any{"inactivityTimeout": int64(0)}),
			withStatus(map[string]any{
				"phase":            "Running",
				"lastActivityTime": time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339),
			}),
		)
		if shouldAutoStop(obj) {
			t.Error("expected false when timeout is 0 (disabled)")
		}
	})

	t.Run("within timeout returns false", func(t *testing.T) {
		resetTimeoutCache()
		obj := newSessionObj("s1", "ns1",
			withSpec(map[string]any{"inactivityTimeout": int64(3600)}),
			withStatus(map[string]any{
				"phase":            "Running",
				"lastActivityTime": time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339),
			}),
		)
		if shouldAutoStop(obj) {
			t.Error("expected false when within timeout window")
		}
	})

	t.Run("beyond timeout returns true", func(t *testing.T) {
		resetTimeoutCache()
		obj := newSessionObj("s1", "ns1",
			withSpec(map[string]any{"inactivityTimeout": int64(300)}),
			withStatus(map[string]any{
				"phase":            "Running",
				"lastActivityTime": time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339),
			}),
		)
		if !shouldAutoStop(obj) {
			t.Error("expected true when beyond timeout (600s idle > 300s timeout)")
		}
	})

	t.Run("falls back to startTime when lastActivityTime missing", func(t *testing.T) {
		resetTimeoutCache()
		obj := newSessionObj("s1", "ns1",
			withSpec(map[string]any{"inactivityTimeout": int64(300)}),
			withStatus(map[string]any{
				"phase":     "Running",
				"startTime": time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339),
			}),
		)
		if !shouldAutoStop(obj) {
			t.Error("expected true when falling back to startTime (600s > 300s)")
		}
	})

	t.Run("falls back to creationTimestamp when startTime missing", func(t *testing.T) {
		resetTimeoutCache()
		obj := newSessionObj("s1", "ns1",
			withCreationTimestamp(time.Now().Add(-10*time.Minute)),
			withSpec(map[string]any{"inactivityTimeout": int64(300)}),
			withStatus(map[string]any{
				"phase": "Running",
			}),
		)
		if !shouldAutoStop(obj) {
			t.Error("expected true when falling back to creationTimestamp (600s > 300s)")
		}
	})

	t.Run("no timestamps at all returns false", func(t *testing.T) {
		resetTimeoutCache()
		obj := newSessionObj("s1", "ns1",
			withSpec(map[string]any{"inactivityTimeout": int64(300)}),
			withStatus(map[string]any{
				"phase": "Running",
			}),
		)
		if shouldAutoStop(obj) {
			t.Error("expected false when no timestamps are available")
		}
	})

	t.Run("uses default timeout when no session or project timeout set", func(t *testing.T) {
		resetTimeoutCache()
		// No spec.inactivityTimeout, no project settings → default 86400s (24h)
		obj := newSessionObj("s1", "ns1",
			withStatus(map[string]any{
				"phase":            "Running",
				"lastActivityTime": time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339),
			}),
		)
		// 1 hour < 24 hours → should not auto-stop
		if shouldAutoStop(obj) {
			t.Error("expected false when using default 24h timeout and only 1h idle")
		}
	})
}

// ==============================
// resolveInactivityTimeout tests
// ==============================

func TestResolveInactivityTimeout(t *testing.T) {
	t.Run("session-level timeout takes precedence", func(t *testing.T) {
		resetTimeoutCache()
		setupFakeDynamicClient()
		obj := newSessionObj("s1", "ns1",
			withSpec(map[string]any{"inactivityTimeout": int64(600)}),
		)
		got := resolveInactivityTimeout(obj)
		if got != 600 {
			t.Errorf("expected 600, got %d", got)
		}
	})

	t.Run("project-level timeout used when session has none", func(t *testing.T) {
		resetTimeoutCache()

		// Create a ProjectSettings object with inactivityTimeoutSeconds
		ps := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "vteam.ambient-code/v1alpha1",
				"kind":       "ProjectSettings",
				"metadata": map[string]any{
					"name":      projectSettingsName,
					"namespace": "ns-with-project",
				},
				"spec": map[string]any{
					"inactivityTimeoutSeconds": int64(1800),
				},
			},
		}
		setupFakeDynamicClient(ps)

		obj := newSessionObj("s1", "ns-with-project")
		got := resolveInactivityTimeout(obj)
		if got != 1800 {
			t.Errorf("expected 1800 (from project settings), got %d", got)
		}
	})

	t.Run("default timeout when neither session nor project set", func(t *testing.T) {
		resetTimeoutCache()
		setupFakeDynamicClient()
		obj := newSessionObj("s1", "ns-empty")
		got := resolveInactivityTimeout(obj)
		if got != defaultInactivityTimeoutSec {
			t.Errorf("expected %d (default), got %d", defaultInactivityTimeoutSec, got)
		}
	})

	t.Run("session timeout zero overrides project and default", func(t *testing.T) {
		resetTimeoutCache()

		ps := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "vteam.ambient-code/v1alpha1",
				"kind":       "ProjectSettings",
				"metadata": map[string]any{
					"name":      projectSettingsName,
					"namespace": "ns-ps",
				},
				"spec": map[string]any{
					"inactivityTimeoutSeconds": int64(3600),
				},
			},
		}
		setupFakeDynamicClient(ps)

		// Session explicitly sets timeout=0 (disabled)
		obj := newSessionObj("s1", "ns-ps",
			withSpec(map[string]any{"inactivityTimeout": int64(0)}),
		)
		got := resolveInactivityTimeout(obj)
		if got != 0 {
			t.Errorf("expected 0 (session overrides project), got %d", got)
		}
	})
}

// ==============================
// triggerInactivityStop tests
// ==============================

func TestTriggerInactivityStop(t *testing.T) {
	gvr := types.GetAgenticSessionResource()

	t.Run("sets desired-phase and stop-reason annotations", func(t *testing.T) {
		resetTimeoutCache()

		// Create a running session that IS idle (timeout=60, idle for 5 min)
		session := newSessionObj("idle-session", "ns1",
			withSpec(map[string]any{"inactivityTimeout": int64(60)}),
			withStatus(map[string]any{
				"phase":            "Running",
				"lastActivityTime": time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339),
			}),
		)
		setupFakeDynamicClient(session)

		err := triggerInactivityStop("ns1", "idle-session")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify annotations were set
		updated, err := config.DynamicClient.Resource(gvr).Namespace("ns1").Get(
			context.Background(), "idle-session", metav1.GetOptions{},
		)
		if err != nil {
			t.Fatalf("failed to get updated session: %v", err)
		}

		annotations := updated.GetAnnotations()
		if annotations["ambient-code.io/desired-phase"] != "Stopped" {
			t.Errorf("expected desired-phase=Stopped, got %q", annotations["ambient-code.io/desired-phase"])
		}
		if annotations[stopReasonAnnotation] != "inactivity" {
			t.Errorf("expected stop-reason=inactivity, got %q", annotations[stopReasonAnnotation])
		}
	})

	t.Run("skips when session is no longer idle on re-check", func(t *testing.T) {
		resetTimeoutCache()

		// Create a session that is recently active (within timeout)
		session := newSessionObj("active-session", "ns1",
			withSpec(map[string]any{"inactivityTimeout": int64(3600)}),
			withStatus(map[string]any{
				"phase":            "Running",
				"lastActivityTime": time.Now().UTC().Format(time.RFC3339), // just now
			}),
		)
		setupFakeDynamicClient(session)

		err := triggerInactivityStop("ns1", "active-session")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify annotations were NOT set
		obj, err := config.DynamicClient.Resource(gvr).Namespace("ns1").Get(
			context.Background(), "active-session", metav1.GetOptions{},
		)
		if err != nil {
			t.Fatalf("failed to get session: %v", err)
		}

		annotations := obj.GetAnnotations()
		if _, exists := annotations["ambient-code.io/desired-phase"]; exists {
			t.Error("expected no desired-phase annotation on active session")
		}
	})

	t.Run("returns nil when session not found", func(t *testing.T) {
		resetTimeoutCache()
		setupFakeDynamicClient()

		err := triggerInactivityStop("ns1", "nonexistent")
		if err != nil {
			t.Errorf("expected nil error for NotFound, got: %v", err)
		}
	})
}

// ==============================
// getProjectInactivityTimeout tests
// ==============================

func TestGetProjectInactivityTimeout(t *testing.T) {
	t.Run("returns -1 when project settings not found", func(t *testing.T) {
		resetTimeoutCache()
		setupFakeDynamicClient()

		got := getProjectInactivityTimeout("ns-no-ps")
		if got != -1 {
			t.Errorf("expected -1 when no ProjectSettings, got %d", got)
		}
	})

	t.Run("returns value from project settings", func(t *testing.T) {
		resetTimeoutCache()

		ps := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "vteam.ambient-code/v1alpha1",
				"kind":       "ProjectSettings",
				"metadata": map[string]any{
					"name":      projectSettingsName,
					"namespace": "ns-with-ps",
				},
				"spec": map[string]any{
					"inactivityTimeoutSeconds": int64(7200),
				},
			},
		}
		setupFakeDynamicClient(ps)

		got := getProjectInactivityTimeout("ns-with-ps")
		if got != 7200 {
			t.Errorf("expected 7200, got %d", got)
		}
	})

	t.Run("caches result for subsequent calls", func(t *testing.T) {
		resetTimeoutCache()

		ps := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "vteam.ambient-code/v1alpha1",
				"kind":       "ProjectSettings",
				"metadata": map[string]any{
					"name":      projectSettingsName,
					"namespace": "ns-cached",
				},
				"spec": map[string]any{
					"inactivityTimeoutSeconds": int64(900),
				},
			},
		}
		setupFakeDynamicClient(ps)

		// First call populates cache
		got1 := getProjectInactivityTimeout("ns-cached")
		if got1 != 900 {
			t.Fatalf("first call: expected 900, got %d", got1)
		}

		// Replace dynamic client with one that has no ProjectSettings.
		// If the cache is working, the second call should still return 900.
		setupFakeDynamicClient()

		got2 := getProjectInactivityTimeout("ns-cached")
		if got2 != 900 {
			t.Errorf("cached call: expected 900 (from cache), got %d", got2)
		}
	})

	t.Run("returns -1 when field not set in project settings", func(t *testing.T) {
		resetTimeoutCache()

		ps := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "vteam.ambient-code/v1alpha1",
				"kind":       "ProjectSettings",
				"metadata": map[string]any{
					"name":      projectSettingsName,
					"namespace": "ns-no-field",
				},
				"spec": map[string]any{
					// no inactivityTimeoutSeconds
				},
			},
		}
		setupFakeDynamicClient(ps)

		got := getProjectInactivityTimeout("ns-no-field")
		if got != -1 {
			t.Errorf("expected -1 when field missing, got %d", got)
		}
	})
}
