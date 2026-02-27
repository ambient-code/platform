package projects_test

import (
	"context"
	"testing"
	"time"

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

func TestProjectGRPCWatch(t *testing.T) {
	h, _ := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	token := h.CreateJWTString(account)

	h.StartControllersServer()
	time.Sleep(500 * time.Millisecond)

	conn, err := grpc.NewClient(
		h.GRPCAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	Expect(err).NotTo(HaveOccurred())
	defer conn.Close()

	client := pb.NewProjectServiceClient(conn)
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)

	watchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	stream, err := client.WatchProjects(watchCtx, &pb.WatchProjectsRequest{})
	Expect(err).NotTo(HaveOccurred())

	received := make(chan *pb.ProjectWatchEvent, 10)
	go func() {
		for {
			event, err := stream.Recv()
			if err != nil {
				return
			}
			select {
			case received <- event:
			case <-watchCtx.Done():
				return
			}
		}
	}()

	time.Sleep(200 * time.Millisecond)

	created, err := client.CreateProject(ctx, &pb.CreateProjectRequest{
		Name: "watch-test-project",
	})
	Expect(err).NotTo(HaveOccurred())
	resourceID := created.GetMetadata().GetId()

	select {
	case event := <-received:
		Expect(event.GetType()).To(Equal(pb.EventType_EVENT_TYPE_CREATED))
		Expect(event.GetResourceId()).To(Equal(resourceID))
		Expect(event.GetProject()).NotTo(BeNil())
		Expect(event.GetProject().GetName()).To(Equal("watch-test-project"))
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for CREATED watch event")
	}

	updatedName := "updated-watch-project"
	_, err = client.UpdateProject(ctx, &pb.UpdateProjectRequest{
		Id:   resourceID,
		Name: &updatedName,
	})
	Expect(err).NotTo(HaveOccurred())

	select {
	case event := <-received:
		Expect(event.GetType()).To(Equal(pb.EventType_EVENT_TYPE_UPDATED))
		Expect(event.GetResourceId()).To(Equal(resourceID))
		Expect(event.GetProject()).NotTo(BeNil())
		Expect(event.GetProject().GetName()).To(Equal("updated-watch-project"))
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for UPDATED watch event")
	}

	_, err = client.DeleteProject(ctx, &pb.DeleteProjectRequest{Id: resourceID})
	Expect(err).NotTo(HaveOccurred())

	select {
	case event := <-received:
		Expect(event.GetType()).To(Equal(pb.EventType_EVENT_TYPE_DELETED))
		Expect(event.GetResourceId()).To(Equal(resourceID))
		Expect(event.GetProject()).To(BeNil())
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for DELETED watch event")
	}
}
