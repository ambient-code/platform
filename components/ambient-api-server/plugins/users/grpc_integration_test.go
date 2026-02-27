package users_test

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

func TestUserGRPCCrud(t *testing.T) {
	h, _ := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	token := h.CreateJWTString(account)

	conn, err := grpc.NewClient(
		h.GRPCAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	Expect(err).NotTo(HaveOccurred())
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)

	email := "test@example.com"
	created, err := client.CreateUser(ctx, &pb.CreateUserRequest{
		Username: "grpc-test-user",
		Name:     "gRPC Test User",
		Email:    &email,
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(created.GetUsername()).To(Equal("grpc-test-user"))
	Expect(created.GetName()).To(Equal("gRPC Test User"))
	Expect(created.GetMetadata().GetId()).NotTo(BeEmpty())
	Expect(created.GetMetadata().GetKind()).To(Equal("User"))

	got, err := client.GetUser(ctx, &pb.GetUserRequest{Id: created.GetMetadata().GetId()})
	Expect(err).NotTo(HaveOccurred())
	Expect(got.GetUsername()).To(Equal("grpc-test-user"))

	newName := "Updated User"
	updated, err := client.UpdateUser(ctx, &pb.UpdateUserRequest{
		Id:   created.GetMetadata().GetId(),
		Name: &newName,
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(updated.GetName()).To(Equal("Updated User"))

	listResp, err := client.ListUsers(ctx, &pb.ListUsersRequest{Page: 1, Size: 10})
	Expect(err).NotTo(HaveOccurred())
	Expect(listResp.GetMetadata().GetTotal()).To(BeNumerically(">=", 1))

	_, err = client.DeleteUser(ctx, &pb.DeleteUserRequest{Id: created.GetMetadata().GetId()})
	Expect(err).NotTo(HaveOccurred())

	_, err = client.GetUser(ctx, &pb.GetUserRequest{Id: created.GetMetadata().GetId()})
	Expect(err).To(HaveOccurred())
	st, ok := status.FromError(err)
	Expect(ok).To(BeTrue())
	Expect(st.Code()).To(Equal(codes.NotFound))
}

func TestUserGRPCWatch(t *testing.T) {
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

	client := pb.NewUserServiceClient(conn)
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)

	watchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	stream, err := client.WatchUsers(watchCtx, &pb.WatchUsersRequest{})
	Expect(err).NotTo(HaveOccurred())

	received := make(chan *pb.UserWatchEvent, 10)
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

	created, err := client.CreateUser(ctx, &pb.CreateUserRequest{
		Username: "watch-test-user",
		Name:     "Watch Test User",
	})
	Expect(err).NotTo(HaveOccurred())
	resourceID := created.GetMetadata().GetId()

	select {
	case event := <-received:
		Expect(event.GetType()).To(Equal(pb.EventType_EVENT_TYPE_CREATED))
		Expect(event.GetResourceId()).To(Equal(resourceID))
		Expect(event.GetUser()).NotTo(BeNil())
		Expect(event.GetUser().GetUsername()).To(Equal("watch-test-user"))
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for CREATED watch event")
	}

	updatedName := "Updated Watch User"
	_, err = client.UpdateUser(ctx, &pb.UpdateUserRequest{
		Id:   resourceID,
		Name: &updatedName,
	})
	Expect(err).NotTo(HaveOccurred())

	select {
	case event := <-received:
		Expect(event.GetType()).To(Equal(pb.EventType_EVENT_TYPE_UPDATED))
		Expect(event.GetResourceId()).To(Equal(resourceID))
		Expect(event.GetUser()).NotTo(BeNil())
		Expect(event.GetUser().GetName()).To(Equal("Updated Watch User"))
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for UPDATED watch event")
	}

	_, err = client.DeleteUser(ctx, &pb.DeleteUserRequest{Id: resourceID})
	Expect(err).NotTo(HaveOccurred())

	select {
	case event := <-received:
		Expect(event.GetType()).To(Equal(pb.EventType_EVENT_TYPE_DELETED))
		Expect(event.GetResourceId()).To(Equal(resourceID))
		Expect(event.GetUser()).To(BeNil())
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for DELETED watch event")
	}
}
