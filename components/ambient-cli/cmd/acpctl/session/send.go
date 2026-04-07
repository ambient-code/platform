package session

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"

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

	streamCtx, streamCancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer streamCancel()

	stream, err := client.Sessions().StreamEvents(streamCtx, sessionID)
	if err != nil {
		return fmt.Errorf("stream events: %w", err)
	}
	defer stream.Close()

	out := cmd.OutOrStdout()
	scanner := bufio.NewScanner(stream)
	var reasoningBuf strings.Builder
	var inText bool
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := line[6:]

		if sendFollowJSON {
			fmt.Fprintln(out, data)
			continue
		}

		var evt struct {
			Type         string `json:"type"`
			Delta        string `json:"delta"`
			ToolCallName string `json:"toolCallName"`
			Content      string `json:"content"`
		}
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			continue
		}
		switch evt.Type {
		case "REASONING_MESSAGE_CONTENT":
			reasoningBuf.WriteString(evt.Delta)
		case "REASONING_END":
			if reasoningBuf.Len() > 0 {
				fmt.Fprintf(out, "[thinking] %s\n", strings.TrimSpace(reasoningBuf.String()))
				reasoningBuf.Reset()
			}
		case "TEXT_MESSAGE_CONTENT":
			if evt.Delta != "" {
				inText = true
				fmt.Fprint(out, evt.Delta)
			}
		case "TEXT_MESSAGE_END":
			if inText {
				fmt.Fprintln(out)
				inText = false
			}
		case "TOOL_CALL_START":
			if evt.ToolCallName != "" {
				fmt.Fprintf(out, "[%s] ", evt.ToolCallName)
			}
		case "TOOL_CALL_RESULT":
			if evt.Content != "" {
				var content string
				if err := json.Unmarshal([]byte(evt.Content), &content); err != nil {
					content = evt.Content
				}
				lines := strings.SplitN(strings.TrimSpace(content), "\n", 4)
				preview := strings.Join(lines, " | ")
				if len(lines) >= 4 {
					preview += " ..."
				}
				fmt.Fprintf(out, "→ %s\n", preview)
			}
		}
	}

	if inText {
		fmt.Fprintln(out)
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return fmt.Errorf("stream error: %w", scanErr)
	}
	return nil
}
