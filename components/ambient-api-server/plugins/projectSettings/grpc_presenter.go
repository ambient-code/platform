package projectSettings

import (
	pb "github.com/ambient/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func projectSettingsToProto(ps *ProjectSettings) *pb.ProjectSettings {
	if ps == nil {
		return nil
	}

	return &pb.ProjectSettings{
		Metadata: &pb.ObjectReference{
			Id:        ps.ID,
			CreatedAt: timestamppb.New(ps.CreatedAt),
			UpdatedAt: timestamppb.New(ps.UpdatedAt),
			Kind:      "ProjectSettings",
			Href:      "/api/ambient-api-server/v1/project_settings/" + ps.ID,
		},
		ProjectId:    ps.ProjectId,
		GroupAccess:  ps.GroupAccess,
		Repositories: ps.Repositories,
	}
}
