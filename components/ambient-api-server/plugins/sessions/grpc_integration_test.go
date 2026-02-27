package sessions_test

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

func TestSessionGRPCCrud(t *testing.T) {
	h, _ := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	token := h.CreateJWTString(account)

	conn, err := grpc.NewClient(
		h.GRPCAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	Expect(err).NotTo(HaveOccurred())
	defer conn.Close()

	client := pb.NewSessionServiceClient(conn)
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)

	created, err := client.CreateSession(ctx, &pb.CreateSessionRequest{
		Name:    "grpc-test-session",
		Prompt:  stringPtr("test prompt"),
		RepoUrl: stringPtr("https://github.com/test/repo"),
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(created.GetName()).To(Equal("grpc-test-session"))
	Expect(created.GetMetadata().GetId()).NotTo(BeEmpty())
	Expect(created.GetMetadata().GetKind()).To(Equal("Session"))
	Expect(created.GetPrompt()).To(Equal("test prompt"))

	got, err := client.GetSession(ctx, &pb.GetSessionRequest{Id: created.GetMetadata().GetId()})
	Expect(err).NotTo(HaveOccurred())
	Expect(got.GetName()).To(Equal("grpc-test-session"))
	Expect(got.GetMetadata().GetId()).To(Equal(created.GetMetadata().GetId()))

	updated, err := client.UpdateSession(ctx, &pb.UpdateSessionRequest{
		Id:   created.GetMetadata().GetId(),
		Name: stringPtr("updated-session"),
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(updated.GetName()).To(Equal("updated-session"))

	listResp, err := client.ListSessions(ctx, &pb.ListSessionsRequest{Page: 1, Size: 10})
	Expect(err).NotTo(HaveOccurred())
	Expect(listResp.GetMetadata().GetTotal()).To(BeNumerically(">=", 1))

	found := false
	for _, s := range listResp.GetItems() {
		if s.GetMetadata().GetId() == created.GetMetadata().GetId() {
			found = true
			break
		}
	}
	Expect(found).To(BeTrue())

	_, err = client.DeleteSession(ctx, &pb.DeleteSessionRequest{Id: created.GetMetadata().GetId()})
	Expect(err).NotTo(HaveOccurred())

	_, err = client.GetSession(ctx, &pb.GetSessionRequest{Id: created.GetMetadata().GetId()})
	Expect(err).To(HaveOccurred())
	st, ok := status.FromError(err)
	Expect(ok).To(BeTrue())
	Expect(st.Code()).To(Equal(codes.NotFound))
}

func TestSessionGRPCUpdateStatus(t *testing.T) {
	h, _ := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	token := h.CreateJWTString(account)

	conn, err := grpc.NewClient(
		h.GRPCAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	Expect(err).NotTo(HaveOccurred())
	defer conn.Close()

	client := pb.NewSessionServiceClient(conn)
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)

	created, err := client.CreateSession(ctx, &pb.CreateSessionRequest{
		Name: "status-test-session",
	})
	Expect(err).NotTo(HaveOccurred())

	phase := "Running"
	updated, err := client.UpdateSessionStatus(ctx, &pb.UpdateSessionStatusRequest{
		Id:    created.GetMetadata().GetId(),
		Phase: &phase,
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(updated.GetPhase()).To(Equal("Running"))
}

func TestSessionGRPCErrors(t *testing.T) {
	h, _ := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	token := h.CreateJWTString(account)

	conn, err := grpc.NewClient(
		h.GRPCAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	Expect(err).NotTo(HaveOccurred())
	defer conn.Close()

	client := pb.NewSessionServiceClient(conn)
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)

	_, err = client.GetSession(ctx, &pb.GetSessionRequest{Id: "nonexistent-id"})
	Expect(err).To(HaveOccurred())
	st, ok := status.FromError(err)
	Expect(ok).To(BeTrue())
	Expect(st.Code()).To(Equal(codes.NotFound))

	_, err = client.DeleteSession(ctx, &pb.DeleteSessionRequest{Id: "nonexistent-id"})
	Expect(err).NotTo(HaveOccurred())

	_, err = client.CreateSession(ctx, &pb.CreateSessionRequest{Name: ""})
	Expect(err).To(HaveOccurred())
	st, ok = status.FromError(err)
	Expect(ok).To(BeTrue())
	Expect(st.Code()).To(Equal(codes.InvalidArgument))
}

func TestSessionGRPCWatch(t *testing.T) {
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

	client := pb.NewSessionServiceClient(conn)
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)

	watchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	stream, err := client.WatchSessions(watchCtx, &pb.WatchSessionsRequest{})
	Expect(err).NotTo(HaveOccurred())

	received := make(chan *pb.SessionWatchEvent, 10)
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

	created, err := client.CreateSession(ctx, &pb.CreateSessionRequest{
		Name:   "watch-test-session",
		Prompt: stringPtr("watch test"),
	})
	Expect(err).NotTo(HaveOccurred())
	resourceID := created.GetMetadata().GetId()

	select {
	case event := <-received:
		Expect(event.GetType()).To(Equal(pb.EventType_EVENT_TYPE_CREATED))
		Expect(event.GetResourceId()).To(Equal(resourceID))
		Expect(event.GetSession()).NotTo(BeNil())
		Expect(event.GetSession().GetName()).To(Equal("watch-test-session"))
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for CREATED watch event")
	}

	_, err = client.UpdateSession(ctx, &pb.UpdateSessionRequest{
		Id:   resourceID,
		Name: stringPtr("updated-watch-session"),
	})
	Expect(err).NotTo(HaveOccurred())

	select {
	case event := <-received:
		Expect(event.GetType()).To(Equal(pb.EventType_EVENT_TYPE_UPDATED))
		Expect(event.GetResourceId()).To(Equal(resourceID))
		Expect(event.GetSession()).NotTo(BeNil())
		Expect(event.GetSession().GetName()).To(Equal("updated-watch-session"))
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for UPDATED watch event")
	}

	_, err = client.DeleteSession(ctx, &pb.DeleteSessionRequest{Id: resourceID})
	Expect(err).NotTo(HaveOccurred())

	select {
	case event := <-received:
		Expect(event.GetType()).To(Equal(pb.EventType_EVENT_TYPE_DELETED))
		Expect(event.GetResourceId()).To(Equal(resourceID))
		Expect(event.GetSession()).To(BeNil())
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for DELETED watch event")
	}
}
