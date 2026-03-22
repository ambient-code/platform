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
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	ambient_v1 "github.com/ambient-code/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

const grpcDefaultPort = "9000"

var defaultOpenShiftPatterns = []string{"apps.rosa", "apps.ocp", "apps.openshift"}

// MessageWatcher streams session messages from a single session via gRPC.
type MessageWatcher struct {
	conn          *grpc.ClientConn
	msgs          chan *types.SessionMessage
	errors        chan error
	ctx           context.Context
	cancel        context.CancelFunc
	timeoutCancel context.CancelFunc
	done          chan struct{}
}

// Messages returns a channel of session messages as they arrive.
func (w *MessageWatcher) Messages() <-chan *types.SessionMessage {
	return w.msgs
}

// Errors returns a channel of stream errors.
func (w *MessageWatcher) Errors() <-chan error {
	return w.errors
}

// Done returns a channel closed when the watcher stops.
func (w *MessageWatcher) Done() <-chan struct{} {
	return w.done
}

// Stop closes the watcher and releases resources.
func (w *MessageWatcher) Stop() {
	if w.timeoutCancel != nil {
		w.timeoutCancel()
	}
	w.cancel()
	if w.conn != nil {
		_ = w.conn.Close()
	}
}

func (w *MessageWatcher) receive(stream grpc.ServerStreamingClient[ambient_v1.SessionMessage]) {
	defer close(w.done)
	defer close(w.msgs)
	defer close(w.errors)

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			pbMsg, err := stream.Recv()
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
			msg := protoMsgToSessionMessage(pbMsg)
			select {
			case w.msgs <- msg:
			case <-w.ctx.Done():
				return
			}
		}
	}
}

// WatchSessionMessages opens a gRPC stream for real-time session message delivery.
// afterSeq sets the replay cursor — pass 0 to replay all messages from the beginning.
// Call Stop() on the returned watcher to cancel and clean up.
func (a *SessionAPI) WatchSessionMessages(ctx context.Context, sessionID string, afterSeq int64, opts *WatchOptions) (*MessageWatcher, error) {
	if opts == nil {
		opts = &WatchOptions{Timeout: 30 * time.Minute}
	}

	conn, err := a.createGRPCConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	grpcClient := ambient_v1.NewSessionServiceClient(conn)

	md := metadata.New(map[string]string{
		"authorization":     "Bearer " + a.client.token,
		"x-ambient-project": a.client.project,
	})

	watchCtx, watchCancel := context.WithCancel(ctx)
	watcher := &MessageWatcher{
		conn:   conn,
		msgs:   make(chan *types.SessionMessage, 64),
		errors: make(chan error, 5),
		ctx:    watchCtx,
		cancel: watchCancel,
		done:   make(chan struct{}),
	}

	streamCtx := metadata.NewOutgoingContext(watchCtx, md)
	if opts.Timeout > 0 {
		var timeoutCancel context.CancelFunc
		streamCtx, timeoutCancel = context.WithTimeout(streamCtx, opts.Timeout)
		watcher.timeoutCancel = timeoutCancel
	}

	stream, err := grpcClient.WatchSessionMessages(streamCtx, &ambient_v1.WatchSessionMessagesRequest{
		SessionId: sessionID,
		AfterSeq:  afterSeq,
	})
	// stream type: grpc.ServerStreamingClient[ambient_v1.SessionMessage]
	if err != nil {
		watchCancel()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to start WatchSessionMessages stream: %w", err)
	}

	go watcher.receive(stream)

	return watcher, nil
}

func protoMsgToSessionMessage(pb *ambient_v1.SessionMessage) *types.SessionMessage {
	msg := &types.SessionMessage{
		SessionID: pb.GetSessionId(),
		Seq:       int(pb.GetSeq()),
		EventType: pb.GetEventType(),
		Payload:   pb.GetPayload(),
	}
	msg.ID = pb.GetId()
	if pb.GetCreatedAt() != nil {
		t := pb.GetCreatedAt().AsTime()
		msg.CreatedAt = &t
	}
	return msg
}

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
		_ = conn.Close()
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
		_ = w.conn.Close()
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
	if session.ProjectId != nil {
		result.ProjectID = *session.ProjectId
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
		tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
		if a.client.insecureSkipVerify {
			tlsCfg.InsecureSkipVerify = true //nolint:gosec
		}
		creds = credentials.NewTLS(tlsCfg)
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
	// Allow explicit override via environment variable
	if grpcURL := os.Getenv("AMBIENT_GRPC_URL"); grpcURL != "" {
		return grpcURL
	}

	u, err := url.Parse(a.client.baseURL)
	if err != nil || u.Host == "" {
		return net.JoinHostPort(a.client.baseURL, grpcDefaultPort)
	}

	// Configurable OpenShift Route pattern detection
	if isOpenShiftRoute(u.Host) {
		return deriveOpenShiftGRPCAddress(u)
	}

	// Use the hostname only (strip any HTTP port) and apply gRPC default port
	return net.JoinHostPort(u.Hostname(), grpcDefaultPort)
}

// isOpenShiftRoute detects if the hostname follows OpenShift Route patterns
func isOpenShiftRoute(host string) bool {
	patterns := defaultOpenShiftPatterns
	if customPattern := os.Getenv("AMBIENT_OPENSHIFT_PATTERN"); customPattern != "" {
		patterns = []string{customPattern}
	}

	for _, pattern := range patterns {
		if strings.Contains(host, pattern) && strings.Contains(host, "ambient-api-server") {
			return true
		}
	}
	return false
}

// deriveOpenShiftGRPCAddress converts OpenShift HTTP route to gRPC route
func deriveOpenShiftGRPCAddress(u *url.URL) string {
	// Convert: ambient-api-server-namespace.apps.rosa.xxx
	// To:      ambient-api-server-grpc-namespace.apps.rosa.xxx
	grpcHost := strings.Replace(u.Host, "ambient-api-server", "ambient-api-server-grpc", 1)

	// Use port 443 for OpenShift Route (maps to pod port 9000 via targetPort)
	// OpenShift Routes only expose ports 80/443 externally
	return grpcHost + ":443"
}
