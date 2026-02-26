package projects

import (
	pb "github.com/ambient/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func projectToProto(p *Project) *pb.Project {
	if p == nil {
		return nil
	}

	return &pb.Project{
		Metadata: &pb.ObjectReference{
			Id:        p.ID,
			CreatedAt: timestamppb.New(p.CreatedAt),
			UpdatedAt: timestamppb.New(p.UpdatedAt),
			Kind:      "Project",
			Href:      "/api/ambient/v1/projects/" + p.ID,
		},
		Name:        p.Name,
		DisplayName: p.DisplayName,
		Description: p.Description,
		Labels:      p.Labels,
		Annotations: p.Annotations,
		Status:      p.Status,
	}
}
