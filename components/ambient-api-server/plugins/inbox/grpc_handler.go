package inbox

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/ambient-code/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
	"github.com/ambient-code/platform/components/ambient-api-server/pkg/middleware"
	"google.golang.org/grpc"
)

type inboxGRPCHandler struct {
	pb.UnimplementedInboxServiceServer
	watchSvc InboxWatchService
}

func NewInboxGRPCHandler(watchSvc InboxWatchService) pb.InboxServiceServer {
	return &inboxGRPCHandler{watchSvc: watchSvc}
}

func (h *inboxGRPCHandler) WatchInboxMessages(req *pb.WatchInboxMessagesRequest, stream grpc.ServerStreamingServer[pb.InboxMessage]) error {
	if req.GetAgentId() == "" {
		return status.Error(codes.InvalidArgument, "agent_id is required")
	}

	ctx := stream.Context()

	if !middleware.IsServiceCaller(ctx) {
		return status.Error(codes.PermissionDenied, "only service callers may watch inbox streams")
	}

	ch, cancel := h.watchSvc.Subscribe(ctx, req.GetAgentId())
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			proto := inboxMessageToProto(msg)
			if err := stream.Send(proto); err != nil {
				return err
			}
		}
	}
}

func inboxMessageToProto(msg *InboxMessage) *pb.InboxMessage {
	p := &pb.InboxMessage{
		Id:        msg.ID,
		AgentId:   msg.AgentId,
		Body:      msg.Body,
		CreatedAt: timestamppb.New(msg.CreatedAt),
		UpdatedAt: timestamppb.New(msg.UpdatedAt),
	}
	if msg.FromAgentId != nil {
		p.FromAgentId = msg.FromAgentId
	}
	if msg.FromName != nil {
		p.FromName = msg.FromName
	}
	if msg.Read != nil {
		p.Read = msg.Read
	}
	return p
}
