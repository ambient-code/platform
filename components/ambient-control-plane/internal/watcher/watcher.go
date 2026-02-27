package watcher

import (
	"context"
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	pb "github.com/ambient/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

type EventType string

const (
	EventCreated EventType = "CREATED"
	EventUpdated EventType = "UPDATED"
	EventDeleted EventType = "DELETED"
)

type WatchEvent struct {
	Type       EventType
	Resource   string
	ResourceID string
	Object     any
}

type EventHandler func(ctx context.Context, event WatchEvent) error

type WatchManager struct {
	conn     *grpc.ClientConn
	handlers map[string][]EventHandler
	mu       sync.RWMutex
	logger   zerolog.Logger
}

func NewWatchManager(conn *grpc.ClientConn, logger zerolog.Logger) *WatchManager {
	return &WatchManager{
		conn:     conn,
		handlers: make(map[string][]EventHandler),
		logger:   logger.With().Str("component", "watcher").Logger(),
	}
}

func (wm *WatchManager) RegisterHandler(resource string, handler EventHandler) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.handlers[resource] = append(wm.handlers[resource], handler)
}

func (wm *WatchManager) Run(ctx context.Context) {
	wm.mu.RLock()
	resources := make([]string, 0, len(wm.handlers))
	for r := range wm.handlers {
		resources = append(resources, r)
	}
	wm.mu.RUnlock()

	var wg sync.WaitGroup
	for _, resource := range resources {
		wg.Add(1)
		go func(res string) {
			defer wg.Done()
			wm.watchLoop(ctx, res)
		}(resource)
	}
	wg.Wait()
}

func (wm *WatchManager) watchLoop(ctx context.Context, resource string) {
	var attempt int
	for {
		if ctx.Err() != nil {
			return
		}

		wm.logger.Info().Str("resource", resource).Int("attempt", attempt).Msg("opening watch stream")

		err := wm.watchOnce(ctx, resource)
		if ctx.Err() != nil {
			return
		}

		if err != nil {
			wm.logger.Warn().Err(err).Str("resource", resource).Msg("watch stream ended")
		}

		backoff := backoffDuration(attempt)
		wm.logger.Info().Str("resource", resource).Dur("backoff", backoff).Msg("reconnecting")

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		attempt++
	}
}

func (wm *WatchManager) watchOnce(ctx context.Context, resource string) error {
	switch resource {
	case "sessions":
		return wm.watchSessions(ctx)
	case "projects":
		return wm.watchProjects(ctx)
	case "project_settings":
		return wm.watchProjectSettings(ctx)
	default:
		wm.logger.Warn().Str("resource", resource).Msg("no gRPC watch available for resource")
		<-ctx.Done()
		return ctx.Err()
	}
}

func (wm *WatchManager) watchSessions(ctx context.Context) error {
	client := pb.NewSessionServiceClient(wm.conn)
	stream, err := client.WatchSessions(ctx, &pb.WatchSessionsRequest{})
	if err != nil {
		return err
	}

	wm.logger.Info().Msg("session watch stream established")

	for {
		event, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		wm.dispatch(ctx, WatchEvent{
			Type:       protoEventType(event.Type),
			Resource:   "sessions",
			ResourceID: event.ResourceId,
			Object:     event.Session,
		})
	}
}

func (wm *WatchManager) watchProjects(ctx context.Context) error {
	client := pb.NewProjectServiceClient(wm.conn)
	stream, err := client.WatchProjects(ctx, &pb.WatchProjectsRequest{})
	if err != nil {
		return err
	}

	wm.logger.Info().Msg("project watch stream established")

	for {
		event, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		wm.dispatch(ctx, WatchEvent{
			Type:       protoEventType(event.Type),
			Resource:   "projects",
			ResourceID: event.ResourceId,
			Object:     event.Project,
		})
	}
}

func (wm *WatchManager) watchProjectSettings(ctx context.Context) error {
	client := pb.NewProjectSettingsServiceClient(wm.conn)
	stream, err := client.WatchProjectSettings(ctx, &pb.WatchProjectSettingsRequest{})
	if err != nil {
		return err
	}

	wm.logger.Info().Msg("project_settings watch stream established")

	for {
		event, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		wm.dispatch(ctx, WatchEvent{
			Type:       protoEventType(event.Type),
			Resource:   "project_settings",
			ResourceID: event.ResourceId,
			Object:     event.ProjectSettings,
		})
	}
}

func (wm *WatchManager) dispatch(ctx context.Context, event WatchEvent) {
	wm.mu.RLock()
	handlers := wm.handlers[event.Resource]
	wm.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			wm.logger.Error().
				Err(err).
				Str("resource", event.Resource).
				Str("event_type", string(event.Type)).
				Str("resource_id", event.ResourceID).
				Msg("handler failed")
		}
	}
}

func protoEventType(t pb.EventType) EventType {
	switch t {
	case pb.EventType_EVENT_TYPE_CREATED:
		return EventCreated
	case pb.EventType_EVENT_TYPE_UPDATED:
		return EventUpdated
	case pb.EventType_EVENT_TYPE_DELETED:
		return EventDeleted
	default:
		return EventType(fmt.Sprintf("UNKNOWN(%d)", int32(t)))
	}
}

func backoffDuration(attempt int) time.Duration {
	base := float64(time.Second)
	d := base * math.Pow(2, float64(attempt))
	max := float64(30 * time.Second)
	if d > max {
		d = max
	}
	return time.Duration(d)
}
