package sessions

import (
	"context"
	"fmt"

	localgrpc "github.com/ambient/platform/components/ambient-api-server/pkg/api/grpc"
	pb "github.com/ambient/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
	"github.com/openshift-online/rh-trex-ai/pkg/auth"
	"github.com/openshift-online/rh-trex-ai/pkg/server"
	"github.com/openshift-online/rh-trex-ai/pkg/server/grpcutil"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
	"google.golang.org/grpc"
)

type sessionGRPCHandler struct {
	pb.UnimplementedSessionServiceServer
	service    SessionService
	generic    services.GenericService
	brokerFunc func() *server.EventBroker
}

func NewSessionGRPCHandler(service SessionService, generic services.GenericService, brokerFunc func() *server.EventBroker) pb.SessionServiceServer {
	return &sessionGRPCHandler{
		service:    service,
		generic:    generic,
		brokerFunc: brokerFunc,
	}
}

func (h *sessionGRPCHandler) GetSession(ctx context.Context, req *pb.GetSessionRequest) (*pb.Session, error) {
	if err := grpcutil.ValidateRequiredID(req.GetId()); err != nil {
		return nil, err
	}

	session, svcErr := h.service.Get(ctx, req.GetId())
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	return sessionToProto(session), nil
}

func (h *sessionGRPCHandler) CreateSession(ctx context.Context, req *pb.CreateSessionRequest) (*pb.Session, error) {
	if err := grpcutil.ValidateStringField("name", req.GetName(), true); err != nil {
		return nil, err
	}

	session := &Session{
		Name:                 req.GetName(),
		RepoUrl:              req.RepoUrl,
		Prompt:               req.Prompt,
		AssignedUserId:       req.AssignedUserId,
		WorkflowId:           req.WorkflowId,
		Repos:                req.Repos,
		Interactive:          req.Interactive,
		Timeout:              req.Timeout,
		LlmModel:             req.LlmModel,
		LlmMaxTokens:         req.LlmMaxTokens,
		ParentSessionId:      req.ParentSessionId,
		BotAccountName:       req.BotAccountName,
		ResourceOverrides:    req.ResourceOverrides,
		EnvironmentVariables: req.EnvironmentVariables,
		SessionLabels:        req.Labels,
		SessionAnnotations:   req.Annotations,
		ProjectId:            req.ProjectId,
	}

	if req.LlmTemperature != nil {
		session.LlmTemperature = req.LlmTemperature
	}

	if username := auth.GetUsernameFromContext(ctx); username != "" {
		session.CreatedByUserId = &username
	}

	created, svcErr := h.service.Create(ctx, session)
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	return sessionToProto(created), nil
}

func (h *sessionGRPCHandler) UpdateSession(ctx context.Context, req *pb.UpdateSessionRequest) (*pb.Session, error) {
	if err := grpcutil.ValidateRequiredID(req.GetId()); err != nil {
		return nil, err
	}

	found, svcErr := h.service.Get(ctx, req.GetId())
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	if req.Name != nil {
		found.Name = *req.Name
	}
	if req.RepoUrl != nil {
		found.RepoUrl = req.RepoUrl
	}
	if req.Prompt != nil {
		found.Prompt = req.Prompt
	}
	if req.AssignedUserId != nil {
		found.AssignedUserId = req.AssignedUserId
	}
	if req.WorkflowId != nil {
		found.WorkflowId = req.WorkflowId
	}
	if req.Repos != nil {
		found.Repos = req.Repos
	}
	if req.Interactive != nil {
		found.Interactive = req.Interactive
	}
	if req.Timeout != nil {
		found.Timeout = req.Timeout
	}
	if req.LlmModel != nil {
		found.LlmModel = req.LlmModel
	}
	if req.LlmTemperature != nil {
		found.LlmTemperature = req.LlmTemperature
	}
	if req.LlmMaxTokens != nil {
		found.LlmMaxTokens = req.LlmMaxTokens
	}
	if req.ParentSessionId != nil {
		found.ParentSessionId = req.ParentSessionId
	}
	if req.BotAccountName != nil {
		found.BotAccountName = req.BotAccountName
	}
	if req.ResourceOverrides != nil {
		found.ResourceOverrides = req.ResourceOverrides
	}
	if req.EnvironmentVariables != nil {
		found.EnvironmentVariables = req.EnvironmentVariables
	}
	if req.Labels != nil {
		found.SessionLabels = req.Labels
	}
	if req.Annotations != nil {
		found.SessionAnnotations = req.Annotations
	}
	if req.ProjectId != nil {
		found.ProjectId = req.ProjectId
	}

	updated, svcErr := h.service.Replace(ctx, found)
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	return sessionToProto(updated), nil
}

func (h *sessionGRPCHandler) UpdateSessionStatus(ctx context.Context, req *pb.UpdateSessionStatusRequest) (*pb.Session, error) {
	if err := grpcutil.ValidateRequiredID(req.GetId()); err != nil {
		return nil, err
	}

	patch := &SessionStatusPatchRequest{}
	if req.Phase != nil {
		patch.Phase = req.Phase
	}
	if req.StartTime != nil {
		t := req.StartTime.AsTime()
		patch.StartTime = &t
	}
	if req.CompletionTime != nil {
		t := req.CompletionTime.AsTime()
		patch.CompletionTime = &t
	}
	if req.SdkSessionId != nil {
		patch.SdkSessionId = req.SdkSessionId
	}
	if req.SdkRestartCount != nil {
		patch.SdkRestartCount = req.SdkRestartCount
	}
	if req.Conditions != nil {
		patch.Conditions = req.Conditions
	}
	if req.ReconciledRepos != nil {
		patch.ReconciledRepos = req.ReconciledRepos
	}
	if req.ReconciledWorkflow != nil {
		patch.ReconciledWorkflow = req.ReconciledWorkflow
	}
	if req.KubeCrUid != nil {
		patch.KubeCrUid = req.KubeCrUid
	}
	if req.KubeNamespace != nil {
		patch.KubeNamespace = req.KubeNamespace
	}

	updated, svcErr := h.service.UpdateStatus(ctx, req.GetId(), patch)
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	return sessionToProto(updated), nil
}

func (h *sessionGRPCHandler) DeleteSession(ctx context.Context, req *pb.DeleteSessionRequest) (*pb.DeleteSessionResponse, error) {
	if err := grpcutil.ValidateRequiredID(req.GetId()); err != nil {
		return nil, err
	}

	svcErr := h.service.Delete(ctx, req.GetId())
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	return &pb.DeleteSessionResponse{}, nil
}

func (h *sessionGRPCHandler) ListSessions(ctx context.Context, req *pb.ListSessionsRequest) (*pb.ListSessionsResponse, error) {
	page, size := grpcutil.NormalizePagination(req.GetPage(), req.GetSize())

	listArgs := services.ListArguments{
		Page: int(page),
		Size: int64(size),
	}

	var sessions []Session
	paging, svcErr := h.generic.List(ctx, "id", &listArgs, &sessions)
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	items := make([]*pb.Session, 0, len(sessions))
	for i := range sessions {
		items = append(items, sessionToProto(&sessions[i]))
	}

	return &pb.ListSessionsResponse{
		Items: items,
		Metadata: &pb.ListMeta{
			Page:  int32(paging.Page),
			Size:  int32(paging.Size),
			Total: int32(paging.Total),
		},
	}, nil
}

func (h *sessionGRPCHandler) WatchSessions(req *pb.WatchSessionsRequest, stream grpc.ServerStreamingServer[pb.SessionWatchEvent]) error {
	broker := h.brokerFunc()
	if broker == nil {
		return fmt.Errorf("event broker not available")
	}

	ctx := stream.Context()
	sub, err := broker.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to event broker: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-sub.Events:
			if !ok {
				return nil
			}

			if event.Source != "Sessions" {
				continue
			}

			watchEvent := &pb.SessionWatchEvent{
				Type:       localgrpc.APIEventTypeToProto(event.EventType),
				ResourceId: event.SourceID,
			}

			if event.EventType != "delete" {
				session, svcErr := h.service.Get(ctx, event.SourceID)
				if svcErr != nil {
					continue
				}
				watchEvent.Session = sessionToProto(session)
			}

			if err := stream.Send(watchEvent); err != nil {
				return err
			}
		}
	}
}
