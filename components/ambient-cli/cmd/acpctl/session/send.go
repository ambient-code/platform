package session

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	"github.com/spf13/cobra"
)

var sendFollow bool
var sendFollowJSON bool

var sendCmd = &cobra.Command{
	Use:   "send <session-id> <message>",
	Short: "Send a message to a session",
	Long: `Send a message to a session.

Without -f, prints the message sequence number and returns immediately.
With -f, streams the assistant response as readable text until RUN_FINISHED.
With -f --json, streams raw AG-UI JSON events instead of assembled text.

Examples:
  acpctl session send <id> "Hello! What's today's date?"
  acpctl session send <id> "Run the tests" -f
  acpctl session send <id> "Run the tests" -f --json`,
	Args: cobra.ExactArgs(2),
	RunE: runSend,
}

func init() {
	sendCmd.Flags().BoolVarP(&sendFollow, "follow", "f", false, "stream response after sending until RUN_FINISHED")
	sendCmd.Flags().BoolVar(&sendFollowJSON, "json", false, "with -f: emit raw AG-UI JSON events instead of text")
}

func runSend(cmd *cobra.Command, args []string) error {
	sessionID := args[0]
	payload := args[1]

	client, err := connection.NewClientFromConfig()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.GetRequestTimeout())
	defer cancel()

	msg, err := client.Sessions().PushMessage(ctx, sessionID, payload)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "sent (seq=%d)\n", msg.Seq)

	if !sendFollow {
		return nil
	}

	// NOTE: We intentionally open the SSE stream AFTER sending the message.
	// The api-server SSE proxy does not return HTTP 200 until the runner has
	// an active run, so subscribing first would deadlock. The event gap is
	// negligible in practice because the runner takes time to start producing
	// events after receiving the message.
	streamCtx, streamCancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer streamCancel()

	stream, err := client.Sessions().StreamEvents(streamCtx, sessionID)
	if err != nil {
		return fmt.Errorf("stream events: %w", err)
	}
	defer stream.Close()

	return renderSSEStream(stream, cmd.OutOrStdout(), sendFollowJSON, true)
}
