package session

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/spf13/cobra"
)

var agUICmd = &cobra.Command{
	Use:   "ag_ui",
	Short: "AG-UI event stream commands",
	Long: `Commands for the AG-UI event stream sub-resource.

The ag_ui endpoint is the canonical real-time event feed for frontends
and the correct write path for human-to-agent communication.

Examples:
  acpctl session ag_ui stream <id>             # stream all events live
  acpctl session ag_ui send <id> "Hello!"      # send a user turn`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var agUIStreamArgs struct {
	afterSeq int64
}

var agUIStreamCmd = &cobra.Command{
	Use:   "stream <session-id>",
	Short: "Stream AG-UI events for a session (SSE)",
	Long: `Stream all AG-UI events for a session in real time via SSE.

The stream replays existing events from after_seq, then delivers
new events as they arrive. Press Ctrl+C to stop.

Examples:
  acpctl session ag_ui stream <id>
  acpctl session ag_ui stream <id> --after 10`,
	Args: cobra.ExactArgs(1),
	RunE: runAgUIStream,
}

var agUISendCmd = &cobra.Command{
	Use:   "send <session-id> <message>",
	Short: "Send a user turn to a session via ag_ui",
	Long: `Push a user-turn message to a session. The event_type is always 'user'.
The runner's WatchSessionMessages gRPC stream picks this up as the next
human input. This is the canonical write path for human-to-agent input.

Examples:
  acpctl session ag_ui send <id> "What is today's date?"
  acpctl session ag_ui send <id> "Run the test suite"`,
	Args: cobra.ExactArgs(2),
	RunE: runAgUISend,
}

func init() {
	agUIStreamCmd.Flags().Int64Var(&agUIStreamArgs.afterSeq, "after", 0, "Only show events after this sequence number")
	agUICmd.AddCommand(agUIStreamCmd)
	agUICmd.AddCommand(agUISendCmd)
}

func runAgUIStream(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	client, err := connection.NewClientFromConfig()
	if err != nil {
		return err
	}

	return streamAgUI(cmd, client, sessionID)
}

func runAgUISend(cmd *cobra.Command, args []string) error {
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

	msg, err := client.Sessions().SendAgUI(ctx, sessionID, payload)
	if err != nil {
		return fmt.Errorf("send ag_ui turn: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "sent (seq=%d, event_type=%s)\n", msg.Seq, msg.EventType)
	return nil
}

func streamAgUI(cmd *cobra.Command, client *sdkclient.Client, sessionID string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	fmt.Fprintf(cmd.OutOrStdout(), "Streaming AG-UI events for session %s (Ctrl+C to stop)...\n\n", sessionID)

	msgs, stop, err := client.Sessions().WatchMessages(ctx, sessionID, agUIStreamArgs.afterSeq)
	if err != nil {
		return fmt.Errorf("watch messages: %w", err)
	}
	defer stop()

	for msg := range msgs {
		printAgUILine(cmd, msg)
	}
	return nil
}

func printAgUILine(cmd *cobra.Command, msg *sdktypes.SessionMessage) {
	ts := msg.CreatedAt.Format("15:04:05")
	display := displayPayload(msg.EventType, msg.Payload)
	fmt.Fprintf(cmd.OutOrStdout(), "[%s] #%d (%s) %s\n", ts, msg.Seq, msg.EventType, display)
}
