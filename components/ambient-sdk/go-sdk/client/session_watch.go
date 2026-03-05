// Watch functionality for Session API
// Implements real-time streaming of session changes via gRPC

package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	ambient_v1 "github.com/ambient-code/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

const grpcDefaultPort = "4434"

// SessionWatcher provides real-time session events
type SessionWatcher struct {
	stream        ambient_v1.SessionService_WatchSessionsClient
	conn          *grpc.ClientConn
	events        chan *types.SessionWatchEvent
	errors        chan error
	ctx           context.Context
	cancel        context.CancelFunc
	timeoutCancel context.CancelFunc
	done          chan struct{}
}

// WatchOptions configures session watching
type WatchOptions struct {
	// ResourceVersion to start watching from (empty = latest)
	ResourceVersion string
	// Timeout for the watch connection
	Timeout time.Duration
}

// Watch creates a new session watcher with real-time events
func (a *SessionAPI) Watch(ctx context.Context, opts *WatchOptions) (*SessionWatcher, error) {
	if opts == nil {
		opts = &WatchOptions{Timeout: 30 * time.Minute}
	}

	// Create gRPC connection to API server
	conn, err := a.createGRPCConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	// Create session service client
	grpcClient := ambient_v1.NewSessionServiceClient(conn)

	// Add authentication metadata
	md := metadata.New(map[string]string{
		"authorization":     "Bearer " + a.client.token,
		"x-ambient-project": a.client.project,
	})

	// Create watcher with its own cancellable context
	watchCtx, watchCancel := context.WithCancel(ctx)
	watcher := &SessionWatcher{
		conn:   conn,
		events: make(chan *types.SessionWatchEvent, 10),
		errors: make(chan error, 5),
		ctx:    watchCtx,
		cancel: watchCancel,
		done:   make(chan struct{}),
	}

	// Apply timeout to stream context but store cancel on watcher so Stop() controls lifetime
	streamCtx := metadata.NewOutgoingContext(watchCtx, md)
	if opts.Timeout > 0 {
		var timeoutCancel context.CancelFunc
		streamCtx, timeoutCancel = context.WithTimeout(streamCtx, opts.Timeout)
		watcher.timeoutCancel = timeoutCancel
	}

	// Start watch stream
	stream, err := grpcClient.WatchSessions(streamCtx, &ambient_v1.WatchSessionsRequest{})
	if err != nil {
		watchCancel()
		conn.Close()
		return nil, fmt.Errorf("failed to start watch stream: %w", err)
	}
	watcher.stream = stream

	// Start goroutine to receive events
	go watcher.receiveEvents()

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
	if w.timeoutCancel != nil {
		w.timeoutCancel()
	}
	w.cancel()
	if w.conn != nil {
		w.conn.Close()
	}
}

// receiveEvents runs in a goroutine to receive and convert events
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
					return // Stream ended normally
				}
				select {
				case w.errors <- fmt.Errorf("watch stream error: %w", err):
				case <-w.ctx.Done():
				}
				return
			}

			// Convert protobuf event to SDK event
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

// convertEvent converts protobuf SessionWatchEvent to SDK types
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

	// Convert session if present
	if event.GetSession() != nil {
		result.Session = w.convertSession(event.GetSession())
	}

	return result
}

// convertSession converts protobuf Session to SDK Session
func (w *SessionWatcher) convertSession(session *ambient_v1.Session) *types.Session {
	if session == nil {
		return nil
	}

	result := &types.Session{
		Name: session.GetName(),
	}

	// Set metadata
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

	// Set optional fields
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

// createGRPCConnection creates a gRPC connection to the ambient-api-server
func (a *SessionAPI) createGRPCConnection() (*grpc.ClientConn, error) {
	grpcAddr := a.deriveGRPCAddress()

	var creds credentials.TransportCredentials
	if strings.HasPrefix(a.client.baseURL, "https://") {
		creds = credentials.NewTLS(&tls.Config{})
	} else {
		creds = insecure.NewCredentials()
	}

	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client at %s: %w", grpcAddr, err)
	}

	return conn, nil
}

// deriveGRPCAddress converts HTTP base URL to gRPC address
func (a *SessionAPI) deriveGRPCAddress() string {
	u, err := url.Parse(a.client.baseURL)
	if err != nil || u.Host == "" {
		return a.client.baseURL + ":" + grpcDefaultPort
	}
	
	// Handle OpenShift Route pattern: convert HTTP route to gRPC route
	if strings.Contains(u.Host, "ambient-api-server") && strings.Contains(u.Host, "apps.rosa") {
		// Convert: ambient-api-server-ambient-code.apps.rosa.xxx
		// To:      ambient-api-server-grpc-ambient-code.apps.rosa.xxx
		grpcHost := strings.Replace(u.Host, "ambient-api-server", "ambient-api-server-grpc", 1)
		if u.Scheme == "https" {
			return grpcHost + ":443"
		} else {
			return grpcHost + ":80"
		}
	}
	
	// Fallback to default behavior for local development
	port := u.Port()
	if port == "" {
		port = grpcDefaultPort
	}
	return net.JoinHostPort(u.Hostname(), port)
}
