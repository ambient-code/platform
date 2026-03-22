package watcher

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	pb "github.com/ambient-code/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type EventType string

const (
	EventCreated EventType = "CREATED"
	EventUpdated EventType = "UPDATED"
	EventDeleted EventType = "DELETED"
)

type SessionWatchEvent struct {
	Type       EventType
	ResourceID string
	Session    *pb.Session
}

type ProjectWatchEvent struct {
	Type       EventType
	ResourceID string
	Project    *pb.Project
}

type ProjectSettingsWatchEvent struct {
	Type            EventType
	ResourceID      string
	ProjectSettings *pb.ProjectSettings
}

type SessionEventHandler func(ctx context.Context, event SessionWatchEvent) error
type ProjectEventHandler func(ctx context.Context, event ProjectWatchEvent) error
type ProjectSettingsEventHandler func(ctx context.Context, event ProjectSettingsWatchEvent) error

type WatchManager struct {
	conn                    *grpc.ClientConn
	token                   string
	sessionHandlers         []SessionEventHandler
	projectHandlers         []ProjectEventHandler
	projectSettingsHandlers []ProjectSettingsEventHandler
	mu                      sync.RWMutex
	logger                  zerolog.Logger
}

func NewWatchManager(conn *grpc.ClientConn, token string, logger zerolog.Logger) *WatchManager {
	return &WatchManager{
		conn:   conn,
		token:  token,
		logger: logger.With().Str("component", "watcher").Logger(),
	}
}

func (wm *WatchManager) authContext(ctx context.Context) context.Context {
	if wm.token == "" {
		return ctx
	}
	return metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+wm.token))
}

func (wm *WatchManager) RegisterSessionHandler(handler SessionEventHandler) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.sessionHandlers = append(wm.sessionHandlers, handler)
}

func (wm *WatchManager) RegisterProjectHandler(handler ProjectEventHandler) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.projectHandlers = append(wm.projectHandlers, handler)
}

func (wm *WatchManager) RegisterProjectSettingsHandler(handler ProjectSettingsEventHandler) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.projectSettingsHandlers = append(wm.projectSettingsHandlers, handler)
}

func (wm *WatchManager) Run(ctx context.Context) {
	wm.mu.RLock()
	hasSessions := len(wm.sessionHandlers) > 0
	hasProjects := len(wm.projectHandlers) > 0
	hasProjectSettings := len(wm.projectSettingsHandlers) > 0
	wm.mu.RUnlock()

	var wg sync.WaitGroup
	if hasSessions {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wm.watchLoop(ctx, "sessions")
		}()
	}
	if hasProjects {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wm.watchLoop(ctx, "projects")
		}()
	}
	if hasProjectSettings {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wm.watchLoop(ctx, "project_settings")
		}()
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
	stream, err := client.WatchSessions(wm.authContext(ctx), &pb.WatchSessionsRequest{})
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

		wm.dispatchSession(ctx, SessionWatchEvent{
			Type:       protoEventType(event.Type),
			ResourceID: event.ResourceId,
			Session:    event.Session,
		})
	}
}

func (wm *WatchManager) watchProjects(ctx context.Context) error {
	client := pb.NewProjectServiceClient(wm.conn)
	stream, err := client.WatchProjects(wm.authContext(ctx), &pb.WatchProjectsRequest{})
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

		wm.dispatchProject(ctx, ProjectWatchEvent{
			Type:       protoEventType(event.Type),
			ResourceID: event.ResourceId,
			Project:    event.Project,
		})
	}
}

func (wm *WatchManager) watchProjectSettings(ctx context.Context) error {
	client := pb.NewProjectSettingsServiceClient(wm.conn)
	stream, err := client.WatchProjectSettings(wm.authContext(ctx), &pb.WatchProjectSettingsRequest{})
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

		wm.dispatchProjectSettings(ctx, ProjectSettingsWatchEvent{
			Type:            protoEventType(event.Type),
			ResourceID:      event.ResourceId,
			ProjectSettings: event.ProjectSettings,
		})
	}
}

func (wm *WatchManager) dispatchSession(ctx context.Context, event SessionWatchEvent) {
	wm.mu.RLock()
	handlers := wm.sessionHandlers
	wm.mu.RUnlock()
	for _, h := range handlers {
		if err := h(ctx, event); err != nil {
			wm.logger.Error().Err(err).Str("resource", "sessions").Str("event_type", string(event.Type)).Str("resource_id", event.ResourceID).Msg("handler failed")
		}
	}
}

func (wm *WatchManager) dispatchProject(ctx context.Context, event ProjectWatchEvent) {
	wm.mu.RLock()
	handlers := wm.projectHandlers
	wm.mu.RUnlock()
	for _, h := range handlers {
		if err := h(ctx, event); err != nil {
			wm.logger.Error().Err(err).Str("resource", "projects").Str("event_type", string(event.Type)).Str("resource_id", event.ResourceID).Msg("handler failed")
		}
	}
}

func (wm *WatchManager) dispatchProjectSettings(ctx context.Context, event ProjectSettingsWatchEvent) {
	wm.mu.RLock()
	handlers := wm.projectSettingsHandlers
	wm.mu.RUnlock()
	for _, h := range handlers {
		if err := h(ctx, event); err != nil {
			wm.logger.Error().Err(err).Str("resource", "project_settings").Str("event_type", string(event.Type)).Str("resource_id", event.ResourceID).Msg("handler failed")
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
	maxBackoff := float64(30 * time.Second)
	if d > maxBackoff {
		d = maxBackoff
	}
	jitter := d * 0.25 * (rand.Float64()*2 - 1)
	return time.Duration(d + jitter)
}
