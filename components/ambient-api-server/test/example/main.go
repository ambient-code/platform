package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	pb "github.com/ambient/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
	openapi "github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	restURL := flag.String("rest", "http://localhost:8000", "REST API server URL")
	grpcAddr := flag.String("grpc", "localhost:9000", "gRPC server address")
	debug := flag.Bool("debug", false, "enable HTTP debug logging")
	flag.Parse()

	cfg := openapi.NewConfiguration()
	cfg.Servers = openapi.ServerConfigurations{{URL: *restURL}}
	cfg.Debug = *debug
	restClient := openapi.NewAPIClient(cfg)
	ctx := context.Background()

	conn, err := grpc.NewClient(*grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("gRPC dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	sessionGRPC := pb.NewSessionServiceClient(conn)
	userGRPC := pb.NewUserServiceClient(conn)

	header("REST API: Health Check")
	resp, _, err := restClient.DefaultAPI.ApiAmbientV1UsersGet(ctx).Size(1).Execute()
	if err != nil {
		log.Fatalf("REST health check failed (is the server running at %s?): %v", *restURL, err)
	}
	fmt.Printf("  REST API is reachable. Users total: %d\n", resp.GetTotal())

	header("gRPC: Health Check")
	grpcUsers, err := userGRPC.ListUsers(ctx, &pb.ListUsersRequest{Page: 1, Size: 1})
	if err != nil {
		log.Fatalf("gRPC health check failed (is the server running at %s?): %v", *grpcAddr, err)
	}
	fmt.Printf("  gRPC API is reachable. Users total: %d\n", grpcUsers.Metadata.Total)

	header("gRPC: Start Watch Stream (background)")
	watchCtx, watchCancel := context.WithCancel(ctx)
	var watchWg sync.WaitGroup
	watchWg.Add(1)
	watchEvents := make([]string, 0)
	var watchMu sync.Mutex
	go func() {
		defer watchWg.Done()
		stream, err := sessionGRPC.WatchSessions(watchCtx, &pb.WatchSessionsRequest{})
		if err != nil {
			fmt.Printf("  Watch stream error: %v\n", err)
			return
		}
		fmt.Println("  Watch stream connected. Listening for session events...")
		for {
			event, err := stream.Recv()
			if err != nil {
				if err == io.EOF || watchCtx.Err() != nil {
					return
				}
				return
			}
			msg := fmt.Sprintf("  >> WATCH EVENT: type=%s resource_id=%s name=%q",
				event.Type, event.ResourceId, event.Session.GetName())
			watchMu.Lock()
			watchEvents = append(watchEvents, msg)
			watchMu.Unlock()
			fmt.Println(msg)
		}
	}()
	time.Sleep(500 * time.Millisecond)

	header("REST: Create User")
	user := openapi.NewUser("demo-user", "Demo User")
	email := "demo@example.com"
	user.SetEmail(email)
	createdUser, _, err := restClient.DefaultAPI.ApiAmbientV1UsersPost(ctx).User(*user).Execute()
	if err != nil {
		log.Fatalf("create user: %v", err)
	}
	prettyPrint(createdUser)
	userID := createdUser.GetId()

	header("REST: Create Project")
	project := openapi.NewProject("demo-project")
	project.SetDisplayName("Demo Project")
	project.SetDescription("Created by the ambient-api-server example")
	createdProject, _, err := restClient.DefaultAPI.ApiAmbientV1ProjectsPost(ctx).Project(*project).Execute()
	if err != nil {
		log.Fatalf("create project: %v", err)
	}
	prettyPrint(createdProject)
	projectID := createdProject.GetId()

	header("gRPC: Create Session (triggers Watch event)")
	prompt := "Implement a REST endpoint for health checks."
	createdSession, err := sessionGRPC.CreateSession(ctx, &pb.CreateSessionRequest{
		Name:      "demo-session",
		Prompt:    &prompt,
		ProjectId: &projectID,
	})
	if err != nil {
		log.Fatalf("gRPC create session: %v", err)
	}
	prettyPrint(createdSession)
	sessionID := createdSession.Metadata.Id
	time.Sleep(500 * time.Millisecond)

	header("gRPC: Get Session")
	fetched, err := sessionGRPC.GetSession(ctx, &pb.GetSessionRequest{Id: sessionID})
	if err != nil {
		log.Fatalf("gRPC get session: %v", err)
	}
	prettyPrint(fetched)

	header("REST: Patch Session (triggers Watch event)")
	patchReq := openapi.NewSessionPatchRequest()
	newName := "demo-session-updated"
	patchReq.SetName(newName)
	patchedSession, _, err := restClient.DefaultAPI.ApiAmbientV1SessionsIdPatch(ctx, sessionID).
		SessionPatchRequest(*patchReq).Execute()
	if err != nil {
		log.Fatalf("patch session: %v", err)
	}
	prettyPrint(patchedSession)
	time.Sleep(500 * time.Millisecond)

	header("gRPC: List Sessions")
	sessionList, err := sessionGRPC.ListSessions(ctx, &pb.ListSessionsRequest{Page: 1, Size: 10})
	if err != nil {
		log.Fatalf("gRPC list sessions: %v", err)
	}
	fmt.Printf("  Total sessions: %d\n", sessionList.Metadata.Total)
	for _, s := range sessionList.Items {
		fmt.Printf("  - %s (id=%s, project=%s)\n", s.Name, s.Metadata.Id, s.GetProjectId())
	}

	header("REST: List Projects")
	projectList, _, err := restClient.DefaultAPI.ApiAmbientV1ProjectsGet(ctx).Execute()
	if err != nil {
		log.Fatalf("list projects: %v", err)
	}
	fmt.Printf("  Total projects: %d\n", projectList.GetTotal())
	for _, p := range projectList.GetItems() {
		fmt.Printf("  - %s (id=%s, display=%s)\n", p.GetName(), p.GetId(), p.GetDisplayName())
	}

	header("gRPC: Delete Session (triggers Watch event)")
	_, err = sessionGRPC.DeleteSession(ctx, &pb.DeleteSessionRequest{Id: sessionID})
	if err != nil {
		log.Fatalf("gRPC delete session: %v", err)
	}
	fmt.Printf("  Deleted session %s\n", sessionID)
	time.Sleep(500 * time.Millisecond)

	header("Cleanup: Delete Project & User")
	_, err = restClient.DefaultAPI.ApiAmbientV1ProjectsIdDelete(ctx, projectID).Execute()
	if err != nil {
		log.Fatalf("delete project: %v", err)
	}
	fmt.Printf("  Deleted project %s\n", projectID)

	_, err = userGRPC.DeleteUser(ctx, &pb.DeleteUserRequest{Id: userID})
	if err != nil {
		log.Fatalf("gRPC delete user: %v", err)
	}
	fmt.Printf("  Deleted user %s\n", userID)

	header("Stop Watch Stream")
	watchCancel()
	watchWg.Wait()

	header("Watch Event Summary")
	watchMu.Lock()
	if len(watchEvents) == 0 {
		fmt.Println("  No watch events received (event propagation may be slow)")
	}
	for _, e := range watchEvents {
		fmt.Println(e)
	}
	watchMu.Unlock()

	fmt.Println("\nDemo complete. All REST + gRPC operations succeeded.")
}

func header(title string) {
	fmt.Printf("\n=== %s ===\n", title)
}

func prettyPrint(v interface{}) {
	data, err := json.MarshalIndent(v, "  ", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "  marshal error: %v\n", err)
		return
	}
	fmt.Printf("  %s\n", string(data))
}
