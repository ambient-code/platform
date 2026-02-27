// Package stop implements the stop subcommand for halting agentic sessions.
package stop

import (
	"context"
	"fmt"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "stop <session-id>",
	Short: "Stop an agentic session",
	Args:  cobra.ExactArgs(1),
	RunE:  run,
}

func run(cmd *cobra.Command, cmdArgs []string) error {
	sessionID := cmdArgs[0]

	client, err := connection.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session, err := client.Sessions().Stop(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("stop session %q: %w", sessionID, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "session/%s stopped (phase: %s)\n", session.ID, session.Phase)
	return nil
}
