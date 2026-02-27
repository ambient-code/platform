package reconciler

import (
	"context"
	"sync"
	"time"

	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/ambient/platform/components/ambient-control-plane/internal/informer"
	"github.com/rs/zerolog"
)

type EventTally struct {
	Added    int
	Modified int
	Deleted  int
}

type TallySnapshot struct {
	Resource    string
	Tally       EventTally
	SeenIDs     []string
	LastEventAt time.Time
}

type TallyReconciler struct {
	resource string
	sdk      *sdkclient.Client
	logger   zerolog.Logger

	mu          sync.RWMutex
	tally       EventTally
	seenIDs     map[string]struct{}
	lastEventAt time.Time
}

func NewTallyReconciler(resource string, sdk *sdkclient.Client, logger zerolog.Logger) *TallyReconciler {
	return &TallyReconciler{
		resource: resource,
		sdk:      sdk,
		logger:   logger.With().Str("reconciler", "tally-"+resource).Logger(),
		seenIDs:  make(map[string]struct{}),
	}
}

func (r *TallyReconciler) Resource() string {
	return r.resource
}

func (r *TallyReconciler) Reconcile(ctx context.Context, event informer.ResourceEvent) error {
	resourceID := extractResourceID(event)

	r.mu.Lock()
	switch event.Type {
	case informer.EventAdded:
		r.tally.Added++
	case informer.EventModified:
		r.tally.Modified++
	case informer.EventDeleted:
		r.tally.Deleted++
	}
	if resourceID != "" {
		r.seenIDs[resourceID] = struct{}{}
	}
	r.lastEventAt = time.Now()
	r.mu.Unlock()

	r.logger.Info().
		Str("event", string(event.Type)).
		Str("resource_id", resourceID).
		Int("total_added", r.tally.Added).
		Int("total_modified", r.tally.Modified).
		Int("total_deleted", r.tally.Deleted).
		Msg("tally updated")

	return nil
}

func (r *TallyReconciler) Snapshot() TallySnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.seenIDs))
	for id := range r.seenIDs {
		ids = append(ids, id)
	}

	return TallySnapshot{
		Resource:    r.resource,
		Tally:       r.tally,
		SeenIDs:     ids,
		LastEventAt: r.lastEventAt,
	}
}

func (r *TallyReconciler) Total() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tally.Added + r.tally.Modified + r.tally.Deleted
}

func extractResourceID(event informer.ResourceEvent) string {
	if event.Object == nil {
		return ""
	}
	switch obj := event.Object.(type) {
	case types.Session:
		return obj.ID
	case types.Project:
		return obj.ID
	case types.ProjectSettings:
		return obj.ID
	default:
		return ""
	}
}
