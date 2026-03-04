package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	ambient_v1 "github.com/ambient-code/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

type SessionWatcher struct {
	stream   ambient_v1.SessionService_WatchSessionsClient
	conn     *grpc.ClientConn
	events   chan *types.SessionWatchEvent
	errors   chan error
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	timeoutC context.CancelFunc
}

type WatchOptions struct {
	ResourceVersion string
	Timeout         time.Duration
	GRPCPort        string
}

func (a *SessionAPI) Watch(ctx context.Context, opts *WatchOptions) (*SessionWatcher, error) {
	if opts == nil {
		opts = &WatchOptions{Timeout: 30 * time.Minute}
	}

	conn, err := a.createGRPCConnection(ctx, opts.GRPCPort)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	client := ambient_v1.NewSessionServiceClient(conn)

	md := metadata.New(map[string]string{
		"authorization":     "Bearer " + a.client.token,
		"x-ambient-project": a.client.project,
	})
	streamCtx := metadata.NewOutgoingContext(ctx, md)

	var timeoutCancel context.CancelFunc
	if opts.Timeout > 0 {
		streamCtx, timeoutCancel = context.WithTimeout(streamCtx, opts.Timeout)
	}

	stream, err := client.WatchSessions(streamCtx, &ambient_v1.WatchSessionsRequest{})
	if err != nil {
		if timeoutCancel != nil {
			timeoutCancel()
		}
		conn.Close()
		return nil, fmt.Errorf("failed to start watch stream: %w", err)
	}

	watchCtx, cancel := context.WithCancel(ctx)
	watcher := &SessionWatcher{
		stream:   stream,
		conn:     conn,
		events:   make(chan *types.SessionWatchEvent, 10),
		errors:   make(chan error, 5),
		ctx:      watchCtx,
		cancel:   cancel,
		done:     make(chan struct{}),
		timeoutC: timeoutCancel,
	}

	go watcher.receiveEvents()

	return watcher, nil
}

func (w *SessionWatcher) Events() <-chan *types.SessionWatchEvent {
	return w.events
}

func (w *SessionWatcher) Errors() <-chan error {
	return w.errors
}

func (w *SessionWatcher) Done() <-chan struct{} {
	return w.done
}

func (w *SessionWatcher) Stop() {
	w.cancel()
	if w.timeoutC != nil {
		w.timeoutC()
	}
	if w.conn != nil {
		w.conn.Close()
	}
}

func (w *SessionWatcher) receiveEvents() {
	defer close(w.done)
	defer close(w.events)
	defer close(w.errors)

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			event, err := w.stream.Recv()
			if err != nil {
				if err == io.EOF {
					return
				}
				select {
				case w.errors <- fmt.Errorf("watch stream error: %w", err):
				case <-w.ctx.Done():
				}
				return
			}

			sdkEvent := w.convertEvent(event)
			if sdkEvent != nil {
				select {
				case w.events <- sdkEvent:
				case <-w.ctx.Done():
					return
				}
			}
		}
	}
}

func (w *SessionWatcher) convertEvent(event *ambient_v1.SessionWatchEvent) *types.SessionWatchEvent {
	if event == nil {
		return nil
	}

	eventType := ""
	switch event.GetType() {
	case ambient_v1.EventType_EVENT_TYPE_CREATED:
		eventType = "CREATED"
	case ambient_v1.EventType_EVENT_TYPE_UPDATED:
		eventType = "UPDATED"
	case ambient_v1.EventType_EVENT_TYPE_DELETED:
		eventType = "DELETED"
	default:
		eventType = "UNKNOWN"
	}

	result := &types.SessionWatchEvent{
		Type:       eventType,
		ResourceID: event.GetResourceId(),
	}

	if event.GetSession() != nil {
		result.Session = w.convertSession(event.GetSession())
	}

	return result
}

func (w *SessionWatcher) convertSession(session *ambient_v1.Session) *types.Session {
	if session == nil {
		return nil
	}

	result := &types.Session{
		Name: session.GetName(),
	}

	if meta := session.GetMetadata(); meta != nil {
		result.ID = meta.GetId()
		result.Kind = meta.GetKind()
		result.Href = meta.GetHref()
		if meta.GetCreatedAt() != nil {
			createdAt := meta.GetCreatedAt().AsTime()
			result.CreatedAt = &createdAt
		}
		if meta.GetUpdatedAt() != nil {
			updatedAt := meta.GetUpdatedAt().AsTime()
			result.UpdatedAt = &updatedAt
		}
	}

	if session.RepoUrl != nil {
		result.RepoURL = *session.RepoUrl
	}
	if session.Prompt != nil {
		result.Prompt = *session.Prompt
	}
	if session.CreatedByUserId != nil {
		result.CreatedByUserID = *session.CreatedByUserId
	}
	if session.AssignedUserId != nil {
		result.AssignedUserID = *session.AssignedUserId
	}
	if session.WorkflowId != nil {
		result.WorkflowID = *session.WorkflowId
	}
	if session.Repos != nil {
		result.Repos = *session.Repos
	}
	if session.Timeout != nil {
		result.Timeout = int(*session.Timeout)
	}
	if session.LlmModel != nil {
		result.LlmModel = *session.LlmModel
	}
	if session.LlmTemperature != nil {
		result.LlmTemperature = *session.LlmTemperature
	}
	if session.LlmMaxTokens != nil {
		result.LlmMaxTokens = int(*session.LlmMaxTokens)
	}
	if session.Phase != nil {
		result.Phase = *session.Phase
	}
	if session.GetStartTime() != nil {
		startTime := session.GetStartTime().AsTime()
		result.StartTime = &startTime
	}
	if session.GetCompletionTime() != nil {
		completionTime := session.GetCompletionTime().AsTime()
		result.CompletionTime = &completionTime
	}

	return result
}

func (a *SessionAPI) createGRPCConnection(ctx context.Context, grpcPort string) (*grpc.ClientConn, error) {
	grpcAddr := a.deriveGRPCAddress(grpcPort)

	var creds credentials.TransportCredentials
	if strings.HasPrefix(a.client.baseURL, "https://") {
		creds = credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})
	} else {
		creds = insecure.NewCredentials()
	}

	conn, err := grpc.NewClient(grpcAddr,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client for %s: %w", grpcAddr, err)
	}

	return conn, nil
}

func (a *SessionAPI) deriveGRPCAddress(grpcPort string) string {
	addr := strings.TrimPrefix(a.client.baseURL, "https://")
	addr = strings.TrimPrefix(addr, "http://")
	addr = strings.TrimSuffix(addr, "/api/ambient/v1")
	addr = strings.TrimSuffix(addr, "/")

	if grpcPort == "" {
		grpcPort = "9000"
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	return net.JoinHostPort(host, grpcPort)
}
