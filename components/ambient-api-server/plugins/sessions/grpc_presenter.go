package sessions

import (
	pb "github.com/ambient/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func sessionToProto(s *Session) *pb.Session {
	if s == nil {
		return nil
	}

	proto := &pb.Session{
		Metadata: &pb.ObjectReference{
			Id:        s.ID,
			CreatedAt: timestamppb.New(s.CreatedAt),
			UpdatedAt: timestamppb.New(s.UpdatedAt),
			Kind:      "Session",
			Href:      "/api/ambient-api-server/v1/sessions/" + s.ID,
		},
		Name:                 s.Name,
		RepoUrl:              s.RepoUrl,
		Prompt:               s.Prompt,
		CreatedByUserId:      s.CreatedByUserId,
		AssignedUserId:       s.AssignedUserId,
		WorkflowId:           s.WorkflowId,
		Repos:                s.Repos,
		Interactive:          s.Interactive,
		Timeout:              s.Timeout,
		LlmModel:             s.LlmModel,
		LlmMaxTokens:         s.LlmMaxTokens,
		ParentSessionId:      s.ParentSessionId,
		BotAccountName:       s.BotAccountName,
		ResourceOverrides:    s.ResourceOverrides,
		EnvironmentVariables: s.EnvironmentVariables,
		Labels:               s.SessionLabels,
		Annotations:          s.SessionAnnotations,
		ProjectId:            s.ProjectId,
		Phase:                s.Phase,
		SdkSessionId:         s.SdkSessionId,
		SdkRestartCount:      s.SdkRestartCount,
		Conditions:           s.Conditions,
		ReconciledRepos:      s.ReconciledRepos,
		ReconciledWorkflow:   s.ReconciledWorkflow,
		KubeCrName:           s.KubeCrName,
		KubeCrUid:            s.KubeCrUid,
		KubeNamespace:        s.KubeNamespace,
	}

	if s.LlmTemperature != nil {
		proto.LlmTemperature = s.LlmTemperature
	}

	if s.StartTime != nil {
		proto.StartTime = timestamppb.New(*s.StartTime)
	}
	if s.CompletionTime != nil {
		proto.CompletionTime = timestamppb.New(*s.CompletionTime)
	}

	return proto
}
