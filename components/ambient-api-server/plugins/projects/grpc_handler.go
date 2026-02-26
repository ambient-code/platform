package projects

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

type projectGRPCHandler struct {
	pb.UnimplementedProjectServiceServer
	service    ProjectService
	generic    services.GenericService
	brokerFunc func() *server.EventBroker
}

func NewProjectGRPCHandler(service ProjectService, generic services.GenericService, brokerFunc func() *server.EventBroker) pb.ProjectServiceServer {
	return &projectGRPCHandler{
		service:    service,
		generic:    generic,
		brokerFunc: brokerFunc,
	}
}

func (h *projectGRPCHandler) GetProject(ctx context.Context, req *pb.GetProjectRequest) (*pb.Project, error) {
	if err := grpcutil.ValidateRequiredID(req.GetId()); err != nil {
		return nil, err
	}

	project, svcErr := h.service.Get(ctx, req.GetId())
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	return projectToProto(project), nil
}

func (h *projectGRPCHandler) CreateProject(ctx context.Context, req *pb.CreateProjectRequest) (*pb.Project, error) {
	if err := grpcutil.ValidateStringField("name", req.GetName(), true); err != nil {
		return nil, err
	}

	project := &Project{
		Name:        req.GetName(),
		DisplayName: req.DisplayName,
		Description: req.Description,
		Labels:      req.Labels,
		Annotations: req.Annotations,
	}

	created, svcErr := h.service.Create(ctx, project)
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	return projectToProto(created), nil
}

func (h *projectGRPCHandler) UpdateProject(ctx context.Context, req *pb.UpdateProjectRequest) (*pb.Project, error) {
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
	if req.DisplayName != nil {
		found.DisplayName = req.DisplayName
	}
	if req.Description != nil {
		found.Description = req.Description
	}
	if req.Labels != nil {
		found.Labels = req.Labels
	}
	if req.Annotations != nil {
		found.Annotations = req.Annotations
	}
	if req.Status != nil {
		found.Status = req.Status
	}

	updated, svcErr := h.service.Replace(ctx, found)
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	return projectToProto(updated), nil
}

func (h *projectGRPCHandler) DeleteProject(ctx context.Context, req *pb.DeleteProjectRequest) (*pb.DeleteProjectResponse, error) {
	if err := grpcutil.ValidateRequiredID(req.GetId()); err != nil {
		return nil, err
	}

	svcErr := h.service.Delete(ctx, req.GetId())
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	return &pb.DeleteProjectResponse{}, nil
}

func (h *projectGRPCHandler) ListProjects(ctx context.Context, req *pb.ListProjectsRequest) (*pb.ListProjectsResponse, error) {
	page, size := grpcutil.NormalizePagination(req.GetPage(), req.GetSize())

	listArgs := services.ListArguments{
		Page: int(page),
		Size: int64(size),
	}

	var projects []Project
	paging, svcErr := h.generic.List(ctx, "id", &listArgs, &projects)
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	items := make([]*pb.Project, 0, len(projects))
	for i := range projects {
		items = append(items, projectToProto(&projects[i]))
	}

	return &pb.ListProjectsResponse{
		Items: items,
		Metadata: &pb.ListMeta{
			Page:  int32(paging.Page),
			Size:  int32(paging.Size),
			Total: int32(paging.Total),
		},
	}, nil
}

func (h *projectGRPCHandler) WatchProjects(req *pb.WatchProjectsRequest, stream grpc.ServerStreamingServer[pb.ProjectWatchEvent]) error {
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

			if event.Source != "Projects" {
				continue
			}

			watchEvent := &pb.ProjectWatchEvent{
				Type:       localgrpc.APIEventTypeToProto(event.EventType),
				ResourceId: event.SourceID,
			}

			if event.EventType != api.DeleteEventType {
				project, svcErr := h.service.Get(ctx, event.SourceID)
				if svcErr != nil {
					glog.Errorf("WatchProjects: failed to get project %s: %v", event.SourceID, svcErr)
					continue
				}
				watchEvent.Project = projectToProto(project)
			}

			if err := stream.Send(watchEvent); err != nil {
				return err
			}
		}
	}
}
