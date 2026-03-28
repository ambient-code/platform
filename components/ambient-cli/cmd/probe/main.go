package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ambient-code/platform/components/ambient-cli/internal/probe"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
)

func main() {
	log.SetFlags(0)

	client, err := connection.NewClientFromConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	s := &probe.State{
		Client:  client,
		BaseURL: cfg.GetAPIUrl(),
		Token:   cfg.GetToken(),
		Project: cfg.GetProject(),
		Log:     log.Printf,
	}

	if err := probe.Run(context.Background(), s); err != nil {
		fmt.Fprintf(os.Stderr, "\nFAILED: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nall steps passed ✓")
}
