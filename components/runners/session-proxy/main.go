// session-proxy is a sidecar that provides a streaming exec API for workspace containers.
// It listens on localhost and executes commands in the workspace pod via kubectl exec.
// ADR-0006: The proxy holds the SA token; the runner container has no K8s API access.
package main

import (
	"log"
	"os"

	"github.com/ambient-code/platform/components/runners/session-proxy/pkg/proxy"
)

func main() {
	config := proxy.Config{
		SessionName: os.Getenv("SESSION_NAME"),
		Namespace:   os.Getenv("NAMESPACE"),
		ListenAddr:  getEnvOrDefault("LISTEN_ADDR", ":8080"),
	}

	if config.SessionName == "" {
		log.Fatal("SESSION_NAME environment variable is required")
	}
	if config.Namespace == "" {
		log.Fatal("NAMESPACE environment variable is required")
	}

	log.Printf("Starting session-proxy for session %s in namespace %s", config.SessionName, config.Namespace)
	log.Printf("Listening on %s", config.ListenAddr)

	server, err := proxy.New(config)
	if err != nil {
		log.Fatalf("Failed to create proxy: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start proxy: %v", err)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
