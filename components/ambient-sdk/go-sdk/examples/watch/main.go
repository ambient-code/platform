package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

func main() {
	fmt.Println("Ambient Platform SDK — Real-Time Session Watch Example")
	fmt.Println("=====================================================")
	fmt.Println()

	// Create client
	c, err := client.NewClientFromEnv(client.WithTimeout(120 * time.Second))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		fmt.Println("\n\nReceived interrupt, stopping watch...")
		cancel()
	}()

	// Start watching sessions
	fmt.Printf("Starting real-time watch for sessions...\n")
	fmt.Printf("Press Ctrl+C to stop.\n\n")

	watcher, err := c.Sessions().Watch(ctx, &client.WatchOptions{
		Timeout: 30 * time.Minute,
	})
	if err != nil {
		log.Fatalf("Failed to start watch: %v", err)
	}
	defer watcher.Stop()

	// Process events
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Watch context cancelled")
			return
		case <-watcher.Done():
			fmt.Println("Watch stream ended")
			return
		case err := <-watcher.Errors():
			log.Printf("Watch error: %v", err)
			return
		case event := <-watcher.Events():
			handleWatchEvent(event)
		}
	}
}

func handleWatchEvent(event *types.SessionWatchEvent) {
	timestamp := time.Now().Format("15:04:05")
	
	switch {
	case event.IsCreated():
		fmt.Printf("[%s] 🆕 CREATED session: %s (id=%s)\n", 
			timestamp, event.Session.Name, event.ResourceID)
		if event.Session.Phase != "" {
			fmt.Printf("        Phase: %s\n", event.Session.Phase)
		}
	case event.IsUpdated():
		fmt.Printf("[%s] 📝 UPDATED session: %s (id=%s)\n", 
			timestamp, event.Session.Name, event.ResourceID)
		if event.Session.Phase != "" {
			fmt.Printf("        Phase: %s\n", event.Session.Phase)
		}
		if event.Session.StartTime != nil {
			fmt.Printf("        Started: %s\n", event.Session.StartTime.Format("15:04:05"))
		}
	case event.IsDeleted():
		fmt.Printf("[%s] 🗑️  DELETED session: id=%s\n", 
			timestamp, event.ResourceID)
	default:
		fmt.Printf("[%s] ❓ UNKNOWN event type: %s (id=%s)\n", 
			timestamp, event.Type, event.ResourceID)
	}
	fmt.Println()
}