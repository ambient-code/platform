// Watch functionality for Session API
// REST-based streaming for compatibility

package client

import (
	"context"
	"time"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

// SessionWatcher provides real-time session events via REST polling
type SessionWatcher struct {
	client   *SessionAPI
	ctx      context.Context
	cancel   context.CancelFunc
	events   chan *types.SessionWatchEvent
	errors   chan error
	done     chan struct{}
	interval time.Duration
	lastList []types.Session
}

// WatchOptions configures session watching
type WatchOptions struct {
	// ResourceVersion to start watching from (empty = latest)
	ResourceVersion string
	// Timeout for the watch connection
	Timeout time.Duration
}

// Watch creates a new session watcher with polling-based events
func (a *SessionAPI) Watch(ctx context.Context, opts *WatchOptions) (*SessionWatcher, error) {
	if opts == nil {
		opts = &WatchOptions{Timeout: 30 * time.Minute}
	}

	watchCtx, cancel := context.WithCancel(ctx)
	if opts.Timeout > 0 {
		watchCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
	}

	watcher := &SessionWatcher{
		client:   a,
		ctx:      watchCtx,
		cancel:   cancel,
		events:   make(chan *types.SessionWatchEvent, 10),
		errors:   make(chan error, 5),
		done:     make(chan struct{}),
		interval: 2 * time.Second,
	}

	// Start polling goroutine
	go watcher.poll()

	return watcher, nil
}

// Events returns a channel of session watch events
func (w *SessionWatcher) Events() <-chan *types.SessionWatchEvent {
	return w.events
}

// Errors returns a channel of watch errors
func (w *SessionWatcher) Errors() <-chan error {
	return w.errors
}

// Done returns a channel that's closed when the watcher stops
func (w *SessionWatcher) Done() <-chan struct{} {
	return w.done
}

// Stop closes the watcher and cleans up resources
func (w *SessionWatcher) Stop() {
	w.cancel()
}

// poll runs in a goroutine to poll and detect changes
func (w *SessionWatcher) poll() {
	defer close(w.done)
	defer close(w.events)
	defer close(w.errors)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	listOpts := types.NewListOptions().Size(100).Build()
	firstPoll := true

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			list, err := w.client.List(w.ctx, listOpts)
			if err != nil {
				select {
				case w.errors <- err:
				case <-w.ctx.Done():
				}
				continue
			}

			sessions := list.Items

			// On first poll, just show all sessions as CREATED
			if firstPoll {
				for _, session := range sessions {
					event := &types.SessionWatchEvent{
						Type:       "CREATED",
						Session:    &session,
						ResourceID: session.ID,
					}
					select {
					case w.events <- event:
					case <-w.ctx.Done():
						return
					}
				}
				w.lastList = sessions
				firstPoll = false
				continue
			}

			// Detect changes between this poll and last poll
			changes := w.detectChanges(w.lastList, sessions)
			for _, event := range changes {
				select {
				case w.events <- event:
				case <-w.ctx.Done():
					return
				}
			}

			w.lastList = sessions
		}
	}
}

// detectChanges compares old and new session lists to generate events
func (w *SessionWatcher) detectChanges(old, new []types.Session) []*types.SessionWatchEvent {
	oldMap := make(map[string]types.Session)
	for _, s := range old {
		oldMap[s.ID] = s
	}

	newMap := make(map[string]types.Session)
	for _, s := range new {
		newMap[s.ID] = s
	}

	var events []*types.SessionWatchEvent

	// Check for new or updated sessions
	for _, session := range new {
		if oldSession, exists := oldMap[session.ID]; !exists {
			// New session
			events = append(events, &types.SessionWatchEvent{
				Type:       "CREATED",
				Session:    &session,
				ResourceID: session.ID,
			})
		} else if w.sessionChanged(oldSession, session) {
			// Updated session
			events = append(events, &types.SessionWatchEvent{
				Type:       "UPDATED", 
				Session:    &session,
				ResourceID: session.ID,
			})
		}
	}

	// Check for deleted sessions
	for _, oldSession := range old {
		if _, exists := newMap[oldSession.ID]; !exists {
			events = append(events, &types.SessionWatchEvent{
				Type:       "DELETED",
				Session:    nil,
				ResourceID: oldSession.ID,
			})
		}
	}

	return events
}

// sessionChanged compares two sessions to detect changes
func (w *SessionWatcher) sessionChanged(old, new types.Session) bool {
	return old.Phase != new.Phase ||
		old.Name != new.Name ||
		old.LlmModel != new.LlmModel ||
		(old.UpdatedAt != nil && new.UpdatedAt != nil && !old.UpdatedAt.Equal(*new.UpdatedAt))
}