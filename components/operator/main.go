package main

import (
	"log"
	"os"

	"ambient-code-operator/internal/config"
	"ambient-code-operator/internal/handlers"
	"ambient-code-operator/internal/preflight"
)

// Build-time metadata (set via -ldflags -X during build)
// These are embedded directly in the binary, so they're always accurate
var (
	GitCommit  = "unknown"
	GitBranch  = "unknown"
	GitVersion = "unknown"
	BuildDate  = "unknown"
)

func logBuildInfo() {
	log.Println("==============================================")
	log.Println("Agentic Session Operator - Build Information")
	log.Println("==============================================")
	log.Printf("Version:     %s", GitVersion)
	log.Printf("Commit:      %s", GitCommit)
	log.Printf("Branch:      %s", GitBranch)
	log.Printf("Repository:  %s", getEnvOrDefault("GIT_REPO", "unknown"))
	log.Printf("Built:       %s", BuildDate)
	log.Printf("Built by:    %s", getEnvOrDefault("BUILD_USER", "unknown"))
	log.Println("==============================================")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Log build information
	logBuildInfo()

	// Initialize Kubernetes clients
	if err := config.InitK8sClients(); err != nil {
		log.Fatalf("Failed to initialize Kubernetes clients: %v", err)
	}

	// Load application configuration
	appConfig := config.LoadConfig()

	log.Printf("Agentic Session Operator starting in namespace: %s", appConfig.Namespace)
	log.Printf("Using ambient-code runner image: %s", appConfig.AmbientCodeRunnerImage)

	// Validate Vertex AI configuration at startup if enabled
	if os.Getenv("CLAUDE_CODE_USE_VERTEX") == "1" {
		if err := preflight.ValidateVertexConfig(appConfig.Namespace); err != nil {
			log.Fatalf("Vertex AI validation failed: %v", err)
		}
	}

	// Run v2 repo format migration before starting watchers
	log.Println("Running v2 repo format migration...")
	handlers.MigrateAllSessions()
	log.Println("Migration check complete - see logs above for details")

	// Start watching AgenticSession resources
	go handlers.WatchAgenticSessions()

	// Start watching for managed namespaces
	go handlers.WatchNamespaces()

	// Start watching ProjectSettings resources
	go handlers.WatchProjectSettings()

	// Start cleanup of expired temporary content pods
	go handlers.CleanupExpiredTempContentPods()

	// Keep the operator running
	select {}
}
