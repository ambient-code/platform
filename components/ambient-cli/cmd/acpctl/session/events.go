package session

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	"github.com/spf13/cobra"
)

var eventsCmd = &cobra.Command{
	Use:   "events <session-id>",
	Short: "Stream live AG-UI events from a running session",
	Long: `Stream live AG-UI events from a running session.

Events are proxied from the runner pod in real time via SSE.
Only available while the session is actively running.

Examples:
  acpctl session events <id>   # stream events (Ctrl+C to stop)`,
	Args: cobra.ExactArgs(1),
	RunE: runEvents,
}

func runEvents(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	client, err := connection.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer cancel()

	stream, err := client.Sessions().StreamEvents(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("stream events: %w", err)
	}
	defer stream.Close()

	fmt.Fprintf(cmd.OutOrStdout(), "Streaming events for session %s (Ctrl+C to stop)...\n\n", sessionID)

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			fmt.Fprintln(cmd.OutOrStdout(), line[6:])
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return fmt.Errorf("stream error: %w", scanErr)
	}
	return nil
}
