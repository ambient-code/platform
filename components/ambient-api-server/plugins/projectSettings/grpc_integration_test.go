package projectSettings_test

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

func TestProjectSettingsGRPCCrud(t *testing.T) {
	h, _ := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	token := h.CreateJWTString(account)

	conn, err := grpc.NewClient(
		h.GRPCAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	Expect(err).NotTo(HaveOccurred())
	defer func() { _ = conn.Close() }()

	client := pb.NewProjectSettingsServiceClient(conn)
	projectClient := pb.NewProjectServiceClient(conn)
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)

	project, err := projectClient.CreateProject(ctx, &pb.CreateProjectRequest{
		Name: "grpc-settings-test-project",
	})
	Expect(err).NotTo(HaveOccurred())
	projectID := project.GetMetadata().GetId()

	groupAccess := "admin"
	created, err := client.CreateProjectSettings(ctx, &pb.CreateProjectSettingsRequest{
		ProjectId:   projectID,
		GroupAccess: &groupAccess,
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(created.GetProjectId()).To(Equal(projectID))
	Expect(created.GetMetadata().GetId()).NotTo(BeEmpty())
	Expect(created.GetMetadata().GetKind()).To(Equal("ProjectSettings"))

	got, err := client.GetProjectSettings(ctx, &pb.GetProjectSettingsRequest{Id: created.GetMetadata().GetId()})
	Expect(err).NotTo(HaveOccurred())
	Expect(got.GetProjectId()).To(Equal(projectID))

	newGroupAccess := "editor"
	updated, err := client.UpdateProjectSettings(ctx, &pb.UpdateProjectSettingsRequest{
		Id:          created.GetMetadata().GetId(),
		GroupAccess: &newGroupAccess,
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(updated.GetGroupAccess()).To(Equal("editor"))

	listResp, err := client.ListProjectSettings(ctx, &pb.ListProjectSettingsRequest{Page: 1, Size: 10})
	Expect(err).NotTo(HaveOccurred())
	Expect(listResp.GetMetadata().GetTotal()).To(BeNumerically(">=", 1))

	_, err = client.DeleteProjectSettings(ctx, &pb.DeleteProjectSettingsRequest{Id: created.GetMetadata().GetId()})
	Expect(err).NotTo(HaveOccurred())

	_, err = client.GetProjectSettings(ctx, &pb.GetProjectSettingsRequest{Id: created.GetMetadata().GetId()})
	Expect(err).To(HaveOccurred())
	st, ok := status.FromError(err)
	Expect(ok).To(BeTrue())
	Expect(st.Code()).To(Equal(codes.NotFound))
}

func TestProjectSettingsGRPCWatch(t *testing.T) {
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
	defer func() { _ = conn.Close() }()

	projectClient := pb.NewProjectServiceClient(conn)
	client := pb.NewProjectSettingsServiceClient(conn)
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)

	project, err := projectClient.CreateProject(ctx, &pb.CreateProjectRequest{
		Name: "watch-settings-project",
	})
	Expect(err).NotTo(HaveOccurred())

	watchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	stream, err := client.WatchProjectSettings(watchCtx, &pb.WatchProjectSettingsRequest{})
	Expect(err).NotTo(HaveOccurred())

	received := make(chan *pb.ProjectSettingsWatchEvent, 10)
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

	groupAccess := "admin"
	created, err := client.CreateProjectSettings(ctx, &pb.CreateProjectSettingsRequest{
		ProjectId:   project.GetMetadata().GetId(),
		GroupAccess: &groupAccess,
	})
	Expect(err).NotTo(HaveOccurred())
	resourceID := created.GetMetadata().GetId()

	select {
	case event := <-received:
		Expect(event.GetType()).To(Equal(pb.EventType_EVENT_TYPE_CREATED))
		Expect(event.GetResourceId()).To(Equal(resourceID))
		Expect(event.GetProjectSettings()).NotTo(BeNil())
		Expect(event.GetProjectSettings().GetGroupAccess()).To(Equal("admin"))
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for CREATED watch event")
	}

	updatedAccess := "editor"
	_, err = client.UpdateProjectSettings(ctx, &pb.UpdateProjectSettingsRequest{
		Id:          resourceID,
		GroupAccess: &updatedAccess,
	})
	Expect(err).NotTo(HaveOccurred())

	select {
	case event := <-received:
		Expect(event.GetType()).To(Equal(pb.EventType_EVENT_TYPE_UPDATED))
		Expect(event.GetResourceId()).To(Equal(resourceID))
		Expect(event.GetProjectSettings()).NotTo(BeNil())
		Expect(event.GetProjectSettings().GetGroupAccess()).To(Equal("editor"))
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for UPDATED watch event")
	}

	_, err = client.DeleteProjectSettings(ctx, &pb.DeleteProjectSettingsRequest{Id: resourceID})
	Expect(err).NotTo(HaveOccurred())

	select {
	case event := <-received:
		Expect(event.GetType()).To(Equal(pb.EventType_EVENT_TYPE_DELETED))
		Expect(event.GetResourceId()).To(Equal(resourceID))
		Expect(event.GetProjectSettings()).To(BeNil())
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for DELETED watch event")
	}
}
