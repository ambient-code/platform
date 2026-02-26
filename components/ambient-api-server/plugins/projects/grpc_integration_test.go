package projects_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/ambient/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
	"github.com/ambient/platform/components/ambient-api-server/test"
)

func TestProjectGRPCCrud(t *testing.T) {
	h, _ := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	token := h.CreateJWTString(account)

	conn, err := grpc.NewClient(
		h.GRPCAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	Expect(err).NotTo(HaveOccurred())
	defer conn.Close()

	client := pb.NewProjectServiceClient(conn)
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)

	displayName := "Test Project Display"
	created, err := client.CreateProject(ctx, &pb.CreateProjectRequest{
		Name:        "grpc-test-project",
		DisplayName: &displayName,
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(created.GetName()).To(Equal("grpc-test-project"))
	Expect(created.GetMetadata().GetId()).NotTo(BeEmpty())
	Expect(created.GetMetadata().GetKind()).To(Equal("Project"))
	Expect(created.GetDisplayName()).To(Equal("Test Project Display"))

	got, err := client.GetProject(ctx, &pb.GetProjectRequest{Id: created.GetMetadata().GetId()})
	Expect(err).NotTo(HaveOccurred())
	Expect(got.GetName()).To(Equal("grpc-test-project"))

	newName := "updated-project"
	updated, err := client.UpdateProject(ctx, &pb.UpdateProjectRequest{
		Id:   created.GetMetadata().GetId(),
		Name: &newName,
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(updated.GetName()).To(Equal("updated-project"))

	listResp, err := client.ListProjects(ctx, &pb.ListProjectsRequest{Page: 1, Size: 10})
	Expect(err).NotTo(HaveOccurred())
	Expect(listResp.GetMetadata().GetTotal()).To(BeNumerically(">=", 1))

	_, err = client.DeleteProject(ctx, &pb.DeleteProjectRequest{Id: created.GetMetadata().GetId()})
	Expect(err).NotTo(HaveOccurred())

	_, err = client.GetProject(ctx, &pb.GetProjectRequest{Id: created.GetMetadata().GetId()})
	Expect(err).To(HaveOccurred())
	st, ok := status.FromError(err)
	Expect(ok).To(BeTrue())
	Expect(st.Code()).To(Equal(codes.NotFound))
}
