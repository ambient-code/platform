package users

import (
	pb "github.com/ambient/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func userToProto(u *User) *pb.User {
	if u == nil {
		return nil
	}

	proto := &pb.User{
		Metadata: &pb.ObjectReference{
			Id:        u.ID,
			CreatedAt: timestamppb.New(u.CreatedAt),
			UpdatedAt: timestamppb.New(u.UpdatedAt),
			Kind:      "User",
			Href:      "/api/ambient-api-server/v1/users/" + u.ID,
		},
		Username: u.Username,
		Name:     u.Name,
	}

	if u.Email != nil {
		proto.Email = u.Email
	}

	return proto
}
