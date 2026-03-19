package client

import (
	"context"
	"fmt"
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	ambient_v1 "github.com/ambient-code/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

type InboxWatcher struct {
	conn          *grpc.ClientConn
	msgs          chan *types.InboxMessage
	errors        chan error
	ctx           context.Context
	cancel        context.CancelFunc
	timeoutCancel context.CancelFunc
	done          chan struct{}
}

func (w *InboxWatcher) Messages() <-chan *types.InboxMessage {
	return w.msgs
}

func (w *InboxWatcher) Errors() <-chan error {
	return w.errors
}

func (w *InboxWatcher) Done() <-chan struct{} {
	return w.done
}

func (w *InboxWatcher) Stop() {
	if w.timeoutCancel != nil {
		w.timeoutCancel()
	}
	w.cancel()
	if w.conn != nil {
		_ = w.conn.Close()
	}
}

func (w *InboxWatcher) receive(stream ambient_v1.InboxService_WatchInboxMessagesClient) {
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
				case w.errors <- fmt.Errorf("inbox watch stream error: %w", err):
				case <-w.ctx.Done():
				}
				return
			}
			msg := protoInboxMsgToSDK(pbMsg)
			select {
			case w.msgs <- msg:
			case <-w.ctx.Done():
				return
			}
		}
	}
}

func (a *InboxMessageAPI) WatchInboxMessages(ctx context.Context, agentID string, opts *WatchOptions) (*InboxWatcher, error) {
	if opts == nil {
		opts = &WatchOptions{Timeout: 30 * time.Minute}
	}

	conn, err := a.createGRPCConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	grpcClient := ambient_v1.NewInboxServiceClient(conn)

	md := metadata.New(map[string]string{
		"authorization":     "Bearer " + a.client.token,
		"x-ambient-project": a.client.project,
	})

	watchCtx, watchCancel := context.WithCancel(ctx)
	watcher := &InboxWatcher{
		conn:   conn,
		msgs:   make(chan *types.InboxMessage, 64),
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

	stream, err := grpcClient.WatchInboxMessages(streamCtx, &ambient_v1.WatchInboxMessagesRequest{
		AgentId: agentID,
	})
	if err != nil {
		watchCancel()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to start WatchInboxMessages stream: %w", err)
	}

	go watcher.receive(stream)

	return watcher, nil
}

func (a *InboxMessageAPI) createGRPCConnection() (*grpc.ClientConn, error) {
	sessionAPI := &SessionAPI{client: a.client}
	return sessionAPI.createGRPCConnection()
}

func protoInboxMsgToSDK(pb *ambient_v1.InboxMessage) *types.InboxMessage {
	msg := &types.InboxMessage{
		AgentID:     pb.GetAgentId(),
		Body:        pb.GetBody(),
		FromAgentID: pb.GetFromAgentId(),
		FromName:    pb.GetFromName(),
		Read:        pb.GetRead(),
	}
	msg.ID = pb.GetId()
	if pb.GetCreatedAt() != nil {
		t := pb.GetCreatedAt().AsTime()
		msg.CreatedAt = &t
	}
	return msg
}
