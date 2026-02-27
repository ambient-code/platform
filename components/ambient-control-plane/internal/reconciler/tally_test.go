package reconciler

import (
	"context"
	"sync"
	"testing"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/ambient/platform/components/ambient-control-plane/internal/informer"
	"github.com/rs/zerolog"
)

func newTestTallyReconciler(resource string) *TallyReconciler {
	return NewTallyReconciler(resource, nil, zerolog.Nop())
}

func testSession(id string) types.Session {
	return types.Session{ObjectReference: types.ObjectReference{ID: id}}
}

func testProject(id, name string) types.Project {
	return types.Project{ObjectReference: types.ObjectReference{ID: id}, Name: name}
}

func testProjectSettings(id, projectID string) types.ProjectSettings {
	return types.ProjectSettings{ObjectReference: types.ObjectReference{ID: id}, ProjectID: projectID}
}

func TestTallyReconcilerResource(t *testing.T) {
	tests := []struct {
		resource string
	}{
		{"sessions"},
		{"projects"},
		{"project_settings"},
	}
	for _, tt := range tests {
		r := newTestTallyReconciler(tt.resource)
		if got := r.Resource(); got != tt.resource {
			t.Errorf("Resource() = %q, want %q", got, tt.resource)
		}
	}
}

func TestTallyReconcilerCountsEventTypes(t *testing.T) {
	r := newTestTallyReconciler("sessions")
	ctx := context.Background()

	events := []informer.ResourceEvent{
		{Type: informer.EventAdded, Resource: "sessions", Object: testSession("s1")},
		{Type: informer.EventAdded, Resource: "sessions", Object: testSession("s2")},
		{Type: informer.EventModified, Resource: "sessions", Object: testSession("s1")},
		{Type: informer.EventDeleted, Resource: "sessions", Object: testSession("s2")},
	}

	for _, e := range events {
		if err := r.Reconcile(ctx, e); err != nil {
			t.Fatalf("Reconcile() returned error: %v", err)
		}
	}

	snap := r.Snapshot()
	if snap.Tally.Added != 2 {
		t.Errorf("Added = %d, want 2", snap.Tally.Added)
	}
	if snap.Tally.Modified != 1 {
		t.Errorf("Modified = %d, want 1", snap.Tally.Modified)
	}
	if snap.Tally.Deleted != 1 {
		t.Errorf("Deleted = %d, want 1", snap.Tally.Deleted)
	}
	if r.Total() != 4 {
		t.Errorf("Total() = %d, want 4", r.Total())
	}
}

func TestTallyReconcilerTracksSeenIDs(t *testing.T) {
	r := newTestTallyReconciler("sessions")
	ctx := context.Background()

	_ = r.Reconcile(ctx, informer.ResourceEvent{
		Type: informer.EventAdded, Resource: "sessions", Object: testSession("aaa"),
	})
	_ = r.Reconcile(ctx, informer.ResourceEvent{
		Type: informer.EventAdded, Resource: "sessions", Object: testSession("bbb"),
	})
	_ = r.Reconcile(ctx, informer.ResourceEvent{
		Type: informer.EventModified, Resource: "sessions", Object: testSession("aaa"),
	})

	snap := r.Snapshot()
	seen := make(map[string]bool)
	for _, id := range snap.SeenIDs {
		seen[id] = true
	}
	if !seen["aaa"] || !seen["bbb"] {
		t.Errorf("SeenIDs = %v, want [aaa, bbb]", snap.SeenIDs)
	}
	if len(snap.SeenIDs) != 2 {
		t.Errorf("len(SeenIDs) = %d, want 2 (duplicates should be deduped)", len(snap.SeenIDs))
	}
}

func TestTallyReconcilerProjectAndSettingsTypes(t *testing.T) {
	ctx := context.Background()

	projR := newTestTallyReconciler("projects")
	_ = projR.Reconcile(ctx, informer.ResourceEvent{
		Type: informer.EventAdded, Resource: "projects", Object: testProject("p1", "default"),
	})
	if projR.Total() != 1 {
		t.Errorf("project Total() = %d, want 1", projR.Total())
	}
	if snap := projR.Snapshot(); len(snap.SeenIDs) != 1 || snap.SeenIDs[0] != "p1" {
		t.Errorf("project SeenIDs = %v, want [p1]", snap.SeenIDs)
	}

	psR := newTestTallyReconciler("project_settings")
	_ = psR.Reconcile(ctx, informer.ResourceEvent{
		Type: informer.EventAdded, Resource: "project_settings", Object: testProjectSettings("ps1", "p1"),
	})
	if psR.Total() != 1 {
		t.Errorf("project_settings Total() = %d, want 1", psR.Total())
	}
	if snap := psR.Snapshot(); len(snap.SeenIDs) != 1 || snap.SeenIDs[0] != "ps1" {
		t.Errorf("project_settings SeenIDs = %v, want [ps1]", snap.SeenIDs)
	}
}

func TestTallyReconcilerNilAndUnknownObjects(t *testing.T) {
	r := newTestTallyReconciler("sessions")
	ctx := context.Background()

	_ = r.Reconcile(ctx, informer.ResourceEvent{
		Type: informer.EventAdded, Resource: "sessions", Object: nil,
	})
	_ = r.Reconcile(ctx, informer.ResourceEvent{
		Type: informer.EventAdded, Resource: "sessions", Object: "not-a-session",
	})

	if r.Total() != 2 {
		t.Errorf("Total() = %d, want 2 (events still counted)", r.Total())
	}
	snap := r.Snapshot()
	if len(snap.SeenIDs) != 0 {
		t.Errorf("SeenIDs = %v, want empty (no ID extractable)", snap.SeenIDs)
	}
}

func TestTallyReconcilerConcurrentAccess(t *testing.T) {
	r := newTestTallyReconciler("sessions")
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Reconcile(ctx, informer.ResourceEvent{
				Type:     informer.EventAdded,
				Resource: "sessions",
				Object:   testSession("concurrent"),
			})
		}()
	}
	wg.Wait()

	if r.Total() != 100 {
		t.Errorf("Total() = %d after 100 concurrent reconciles, want 100", r.Total())
	}
}

func TestTallyReconcilerSnapshotIsIsolated(t *testing.T) {
	r := newTestTallyReconciler("sessions")
	ctx := context.Background()

	_ = r.Reconcile(ctx, informer.ResourceEvent{
		Type: informer.EventAdded, Resource: "sessions", Object: testSession("s1"),
	})
	snap1 := r.Snapshot()

	_ = r.Reconcile(ctx, informer.ResourceEvent{
		Type: informer.EventAdded, Resource: "sessions", Object: testSession("s2"),
	})
	snap2 := r.Snapshot()

	if snap1.Tally.Added != 1 {
		t.Errorf("snap1 should be frozen at 1, got %d", snap1.Tally.Added)
	}
	if snap2.Tally.Added != 2 {
		t.Errorf("snap2 should be 2, got %d", snap2.Tally.Added)
	}
}

func TestTallyReconcilerLastEventAt(t *testing.T) {
	r := newTestTallyReconciler("sessions")
	ctx := context.Background()

	snap := r.Snapshot()
	if !snap.LastEventAt.IsZero() {
		t.Error("LastEventAt should be zero before any events")
	}

	_ = r.Reconcile(ctx, informer.ResourceEvent{
		Type: informer.EventAdded, Resource: "sessions", Object: testSession("s1"),
	})

	snap = r.Snapshot()
	if snap.LastEventAt.IsZero() {
		t.Error("LastEventAt should be non-zero after an event")
	}
}

func TestTallyReconcilerSatisfiesInterface(t *testing.T) {
	var _ Reconciler = newTestTallyReconciler("sessions")
}

func TestExtractResourceID(t *testing.T) {
	tests := []struct {
		name   string
		object any
		want   string
	}{
		{"session", testSession("s1"), "s1"},
		{"project", testProject("p1", "default"), "p1"},
		{"project_settings", testProjectSettings("ps1", "p1"), "ps1"},
		{"nil", nil, ""},
		{"unknown type", 42, ""},
		{"empty session", types.Session{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractResourceID(informer.ResourceEvent{Object: tt.object})
			if got != tt.want {
				t.Errorf("extractResourceID() = %q, want %q", got, tt.want)
			}
		})
	}
}
