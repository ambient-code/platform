package session

import (
	"context"
	"fmt"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	"github.com/spf13/cobra"
)

var sendCmd = &cobra.Command{
	Use:   "send <session-id> <message>",
	Short: "Send a message to a session",
	Long: `Send a message to a session.

Examples:
  acpctl session send <id> "Hello! What's today's date?"
  acpctl session send <id> "Run the tests"`,
	Args: cobra.ExactArgs(2),
	RunE: runSend,
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
	return nil
}
