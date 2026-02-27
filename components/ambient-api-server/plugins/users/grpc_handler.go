package users

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

type userGRPCHandler struct {
	pb.UnimplementedUserServiceServer
	service    UserService
	generic    services.GenericService
	brokerFunc func() *server.EventBroker
}

func NewUserGRPCHandler(service UserService, generic services.GenericService, brokerFunc func() *server.EventBroker) pb.UserServiceServer {
	return &userGRPCHandler{
		service:    service,
		generic:    generic,
		brokerFunc: brokerFunc,
	}
}

func (h *userGRPCHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
	if err := grpcutil.ValidateRequiredID(req.GetId()); err != nil {
		return nil, err
	}

	user, svcErr := h.service.Get(ctx, req.GetId())
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	return userToProto(user), nil
}

func (h *userGRPCHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
	if err := grpcutil.ValidateStringField("username", req.GetUsername(), true); err != nil {
		return nil, err
	}
	if err := grpcutil.ValidateStringField("name", req.GetName(), true); err != nil {
		return nil, err
	}

	user := &User{
		Username: req.GetUsername(),
		Name:     req.GetName(),
		Email:    req.Email,
	}

	created, svcErr := h.service.Create(ctx, user)
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	return userToProto(created), nil
}

func (h *userGRPCHandler) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.User, error) {
	if err := grpcutil.ValidateRequiredID(req.GetId()); err != nil {
		return nil, err
	}

	found, svcErr := h.service.Get(ctx, req.GetId())
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	if req.Username != nil {
		found.Username = *req.Username
	}
	if req.Name != nil {
		found.Name = *req.Name
	}
	if req.Email != nil {
		found.Email = req.Email
	}

	updated, svcErr := h.service.Replace(ctx, found)
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	return userToProto(updated), nil
}

func (h *userGRPCHandler) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	if err := grpcutil.ValidateRequiredID(req.GetId()); err != nil {
		return nil, err
	}

	svcErr := h.service.Delete(ctx, req.GetId())
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	return &pb.DeleteUserResponse{}, nil
}

func (h *userGRPCHandler) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	page, size := grpcutil.NormalizePagination(req.GetPage(), req.GetSize())

	listArgs := services.ListArguments{
		Page: int(page),
		Size: int64(size),
	}

	var users []User
	paging, svcErr := h.generic.List(ctx, "id", &listArgs, &users)
	if svcErr != nil {
		return nil, grpcutil.ServiceErrorToGRPC(svcErr)
	}

	items := make([]*pb.User, 0, len(users))
	for i := range users {
		items = append(items, userToProto(&users[i]))
	}

	return &pb.ListUsersResponse{
		Items: items,
		Metadata: &pb.ListMeta{
			Page:  int32(paging.Page),
			Size:  int32(paging.Size),
			Total: int32(paging.Total),
		},
	}, nil
}

func (h *userGRPCHandler) WatchUsers(req *pb.WatchUsersRequest, stream grpc.ServerStreamingServer[pb.UserWatchEvent]) error {
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

			watchEvent := &pb.UserWatchEvent{
				Type:       localgrpc.APIEventTypeToProto(event.EventType),
				ResourceId: event.SourceID,
			}

			if event.EventType != api.DeleteEventType {
				user, svcErr := h.service.Get(ctx, event.SourceID)
				if svcErr != nil {
					glog.Errorf("WatchUsers: failed to get user %s: %v", event.SourceID, svcErr)
					continue
				}
				watchEvent.User = userToProto(user)
			}

			if err := stream.Send(watchEvent); err != nil {
				return err
			}
		}
	}
}
