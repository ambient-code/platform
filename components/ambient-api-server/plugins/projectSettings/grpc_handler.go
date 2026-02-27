package projectSettings

import (
	"context"

	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ambient/platform/components/ambient-api-server/pkg/api"
	localgrpc "github.com/ambient/platform/components/ambient-api-server/pkg/api/grpc"
	pb "github.com/ambient/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
	"github.com/openshift-online/rh-trex-ai/pkg/server"
	"github.com/openshift-online/rh-trex-ai/pkg/server/grpcutil"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

type projectSettingsGRPCHandler struct {
	pb.UnimplementedProjectSettingsServiceServer
	service    ProjectSettingsService
	generic    services.GenericService
	brokerFunc func() *server.EventBroker
}

func NewProjectSettingsGRPCHandler(service ProjectSettingsService, generic services.GenericService, brokerFunc func() *server.EventBroker) pb.ProjectSettingsServiceServer {
	return &projectSettingsGRPCHandler{
		service:    service,
		generic:    generic,
		brokerFunc: brokerFunc,
	}
}

func (h *projectSettingsGRPCHandler) GetProjectSettings(ctx context.Context, req *pb.GetProjectSettingsRequest) (*pb.ProjectSettings, error) {
	if err := grpcutil.ValidateRequiredID(req.GetId()); err != nil {
		return nil, err
	}

	ps, svcErr := h.service.Get(ctx, req.GetId())
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	return projectSettingsToProto(ps), nil
}

func (h *projectSettingsGRPCHandler) CreateProjectSettings(ctx context.Context, req *pb.CreateProjectSettingsRequest) (*pb.ProjectSettings, error) {
	if err := grpcutil.ValidateStringField("project_id", req.GetProjectId(), true); err != nil {
		return nil, err
	}

	ps := &ProjectSettings{
		ProjectId:    req.GetProjectId(),
		GroupAccess:  req.GroupAccess,
		Repositories: req.Repositories,
	}

	created, svcErr := h.service.Create(ctx, ps)
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	return projectSettingsToProto(created), nil
}

func (h *projectSettingsGRPCHandler) UpdateProjectSettings(ctx context.Context, req *pb.UpdateProjectSettingsRequest) (*pb.ProjectSettings, error) {
	if err := grpcutil.ValidateRequiredID(req.GetId()); err != nil {
		return nil, err
	}

	found, svcErr := h.service.Get(ctx, req.GetId())
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	if req.ProjectId != nil {
		found.ProjectId = *req.ProjectId
	}
	if req.GroupAccess != nil {
		found.GroupAccess = req.GroupAccess
	}
	if req.Repositories != nil {
		found.Repositories = req.Repositories
	}

	updated, svcErr := h.service.Replace(ctx, found)
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	return projectSettingsToProto(updated), nil
}

func (h *projectSettingsGRPCHandler) DeleteProjectSettings(ctx context.Context, req *pb.DeleteProjectSettingsRequest) (*pb.DeleteProjectSettingsResponse, error) {
	if err := grpcutil.ValidateRequiredID(req.GetId()); err != nil {
		return nil, err
	}

	svcErr := h.service.Delete(ctx, req.GetId())
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	return &pb.DeleteProjectSettingsResponse{}, nil
}

func (h *projectSettingsGRPCHandler) ListProjectSettings(ctx context.Context, req *pb.ListProjectSettingsRequest) (*pb.ListProjectSettingsResponse, error) {
	page, size := grpcutil.NormalizePagination(req.GetPage(), req.GetSize())

	listArgs := services.ListArguments{
		Page: int(page),
		Size: int64(size),
	}

	var psList []ProjectSettings
	paging, svcErr := h.generic.List(ctx, "id", &listArgs, &psList)
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	items := make([]*pb.ProjectSettings, 0, len(psList))
	for i := range psList {
		items = append(items, projectSettingsToProto(&psList[i]))
	}

	return &pb.ListProjectSettingsResponse{
		Items: items,
		Metadata: &pb.ListMeta{
			Page:  int32(paging.Page),
			Size:  int32(paging.Size),
			Total: int32(paging.Total),
		},
	}, nil
}

func (h *projectSettingsGRPCHandler) WatchProjectSettings(req *pb.WatchProjectSettingsRequest, stream grpc.ServerStreamingServer[pb.ProjectSettingsWatchEvent]) error {
	broker := h.brokerFunc()
	if broker == nil {
		return status.Error(codes.Unavailable, "event broker not available")
	}

	ctx := stream.Context()
	sub, err := broker.Subscribe(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe to event broker: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-sub.Events:
			if !ok {
				return nil
			}

			if event.Source != EventSource {
				continue
			}

			watchEvent := &pb.ProjectSettingsWatchEvent{
				Type:       localgrpc.APIEventTypeToProto(event.EventType),
				ResourceId: event.SourceID,
			}

			if event.EventType != api.DeleteEventType {
				ps, svcErr := h.service.Get(ctx, event.SourceID)
				if svcErr != nil {
					glog.Errorf("WatchProjectSettings: failed to get project settings %s: %v", event.SourceID, svcErr)
					continue
				}
				watchEvent.ProjectSettings = projectSettingsToProto(ps)
			}

			if err := stream.Send(watchEvent); err != nil {
				return err
			}
		}
	}
}
