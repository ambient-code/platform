package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	openapi "github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
)

func main() {
	serverURL := flag.String("server", "http://localhost:8000", "API server base URL")
	debug := flag.Bool("debug", false, "enable HTTP debug logging")
	flag.Parse()

	cfg := openapi.NewConfiguration()
	cfg.Servers = openapi.ServerConfigurations{
		{URL: *serverURL},
	}
	cfg.Debug = *debug

	client := openapi.NewAPIClient(cfg)
	ctx := context.Background()

	fmt.Println("=== Create User ===")
	user := openapi.NewUser("jdoe", "Jane Doe")
	createdUser, _, err := client.DefaultAPI.ApiAmbientV1UsersPost(ctx).User(*user).Execute()
	if err != nil {
		log.Fatalf("create user: %v", err)
	}
	prettyPrint(createdUser)

	fmt.Println("\n=== List Users ===")
	userList, _, err := client.DefaultAPI.ApiAmbientV1UsersGet(ctx).Execute()
	if err != nil {
		log.Fatalf("list users: %v", err)
	}
	prettyPrint(userList)

	fmt.Println("\n=== Create Session ===")
	session := openapi.NewSession("test-session")
	session.SetPrompt("Implement a REST endpoint for health checks.")
	createdSession, _, err := client.DefaultAPI.ApiAmbientV1SessionsPost(ctx).Session(*session).Execute()
	if err != nil {
		log.Fatalf("create session: %v", err)
	}
	prettyPrint(createdSession)

	fmt.Println("\n=== List Sessions ===")
	sessionList, _, err := client.DefaultAPI.ApiAmbientV1SessionsGet(ctx).Execute()
	if err != nil {
		log.Fatalf("list sessions: %v", err)
	}
	prettyPrint(sessionList)

	fmt.Println("\nAll resources created and listed successfully.")
}

func prettyPrint(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal error: %v\n", err)
		return
	}
	fmt.Println(string(data))
}
