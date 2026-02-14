package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

const (
	// Default API endpoint for local development
	defaultAPIURL = "http://localhost:8080"
	
	// Example session configuration
	exampleTask  = "Analyze the repository structure and provide a brief summary of the codebase organization."
	exampleModel = "claude-3.5-sonnet"
)

func main() {
	fmt.Println("🌐 Ambient Platform SDK - HTTP Client Example")
	fmt.Println("============================================")

	// Get configuration from environment or use defaults
	apiURL := getEnvOrDefault("AMBIENT_API_URL", defaultAPIURL)
	token := getEnvOrDefault("AMBIENT_TOKEN", "")
	project := getEnvOrDefault("AMBIENT_PROJECT", "mturansk")
	
	if token == "" {
		log.Fatal("❌ AMBIENT_TOKEN environment variable is required")
	}

	// Create HTTP client
	client, err := client.NewClientWithTimeout(apiURL, token, project, 60*time.Second)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	fmt.Printf("✓ Created client for API: %s\n", apiURL)
	fmt.Printf("✓ Using project: %s\n", project)

	ctx := context.Background()

	// Example 1: Create a new session
	fmt.Println("\n📝 Creating new session...")
	createReq := &types.CreateSessionRequest{
		Task:  exampleTask,
		Model: exampleModel,
		Repos: []types.RepoHTTP{
			{
				URL:    "https://github.com/ambient-code/platform",
				Branch: "main",
			},
		},
	}

	createResp, err := client.CreateSession(ctx, createReq)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	sessionID := createResp.ID
	fmt.Printf("✓ Created session: %s\n", sessionID)

	// Example 2: Get session details
	fmt.Println("\n🔍 Getting session details...")
	session, err := client.GetSession(ctx, sessionID)
	if err != nil {
		log.Fatalf("Failed to get session: %v", err)
	}

	printSessionDetails(session)

	// Example 3: List all sessions
	fmt.Println("\n📋 Listing all sessions...")
	listResp, err := client.ListSessions(ctx)
	if err != nil {
		log.Fatalf("Failed to list sessions: %v", err)
	}

	fmt.Printf("✓ Found %d sessions (total: %d)\n", len(listResp.Items), listResp.Total)
	for i, s := range listResp.Items {
		if i < 3 { // Show first 3 sessions
			fmt.Printf("  %d. %s (%s) - %s\n", i+1, s.ID, s.Status, truncateString(s.Task, 60))
		}
	}
	if len(listResp.Items) > 3 {
		fmt.Printf("  ... and %d more\n", len(listResp.Items)-3)
	}

	// Example 4: Monitor session (optional)
	if shouldMonitorSession() {
		fmt.Println("\n⏳ Monitoring session completion...")
		fmt.Println("   Note: This may take time depending on the task complexity")
		
		completedSession, err := client.WaitForCompletion(ctx, sessionID, 5*time.Second)
		if err != nil {
			log.Printf("❌ Monitoring failed: %v", err)
		} else {
			fmt.Println("\n🎉 Session completed!")
			printSessionDetails(completedSession)
		}
	}

	fmt.Println("\n✅ HTTP Client demonstration complete!")
	fmt.Println("\n💡 Next steps:")
	fmt.Println("   • Check session status periodically")
	fmt.Println("   • Use the session ID to retrieve results")
	fmt.Println("   • Create additional sessions as needed")
}

// printSessionDetails displays detailed information about a session
func printSessionDetails(session *types.SessionResponse) {
	fmt.Printf("   ID: %s\n", session.ID)
	fmt.Printf("   Status: %s\n", session.Status)
	fmt.Printf("   Task: %s\n", truncateString(session.Task, 80))
	fmt.Printf("   Model: %s\n", session.Model)
	fmt.Printf("   Created: %s\n", session.CreatedAt)
	
	if session.CompletedAt != "" {
		fmt.Printf("   Completed: %s\n", session.CompletedAt)
	}
	
	if session.Result != "" {
		fmt.Printf("   Result: %s\n", truncateString(session.Result, 100))
	}
	
	if session.Error != "" {
		fmt.Printf("   Error: %s\n", session.Error)
	}
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// shouldMonitorSession checks if user wants to monitor session completion
func shouldMonitorSession() bool {
	monitor := getEnvOrDefault("MONITOR_SESSION", "false")
	return monitor == "true" || monitor == "1"
}

// truncateString truncates a string to specified length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}