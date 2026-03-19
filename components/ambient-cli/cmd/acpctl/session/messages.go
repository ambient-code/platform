package session

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/output"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/spf13/cobra"
)

var msgArgs struct {
	follow       bool
	outputFormat string
	afterSeq     int
}

var messagesCmd = &cobra.Command{
	Use:   "messages <session-id>",
	Short: "List or stream messages for a session",
	Long: `List or stream messages for a session.

Examples:
  acpctl session messages <id>           # snapshot
  acpctl session messages <id> -f        # live stream (Ctrl+C to stop)
  acpctl session messages <id> -o json   # JSON snapshot
  acpctl session messages <id> --after 5 # messages after seq 5`,
	Args: cobra.ExactArgs(1),
	RunE: runMessages,
}

func init() {
	messagesCmd.Flags().BoolVarP(&msgArgs.follow, "follow", "f", false, "Stream messages live")
	messagesCmd.Flags().StringVarP(&msgArgs.outputFormat, "output", "o", "", "Output format: json")
	messagesCmd.Flags().IntVar(&msgArgs.afterSeq, "after", 0, "Only show messages after this sequence number")
}

func runMessages(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	client, err := connection.NewClientFromConfig()
	if err != nil {
		return err
	}

	if msgArgs.follow {
		return streamMessages(cmd, client, sessionID)
	}

	format, err := output.ParseFormat(msgArgs.outputFormat)
	if err != nil {
		return err
	}
	printer := output.NewPrinter(format, cmd.OutOrStdout())

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.GetRequestTimeout())
	defer cancel()

	return listMessages(ctx, client, printer, sessionID)
}

var reKV = regexp.MustCompile(`(\w+)='((?:[^'\\]|\\.)*)'`)

func extractField(payload, field string) string {
	var raw string
	if err := json.Unmarshal([]byte(payload), &raw); err == nil {
		payload = raw
	}
	for _, m := range reKV.FindAllStringSubmatch(payload, -1) {
		if m[1] == field {
			return strings.ReplaceAll(m[2], `\'`, `'`)
		}
	}
	return ""
}

func displayPayload(eventType, payload string) string {
	switch eventType {
	case "user":
		return payload
	case "TEXT_MESSAGE_CONTENT", "REASONING_MESSAGE_CONTENT", "TOOL_CALL_ARGS":
		if d := extractField(payload, "delta"); d != "" {
			return d
		}
	case "TOOL_CALL_START":
		if name := extractField(payload, "tool_call_name"); name != "" {
			return name
		}
	case "TOOL_CALL_RESULT":
		if c := extractField(payload, "content"); c != "" {
			return c
		}
	case "RUN_FINISHED":
		return displayRunFinished(payload)
	case "MESSAGES_SNAPSHOT":
		return displayMessagesSnapshot(payload)
	case "RUN_ERROR":
		if msg := extractField(payload, "message"); msg != "" {
			return msg
		}
	}
	return ""
}

func displayRunFinished(payload string) string {
	var data struct {
		Result struct {
			DurationMs float64 `json:"duration_ms"`
			NumTurns   int     `json:"num_turns"`
			TotalCost  float64 `json:"total_cost_usd"`
			Usage      struct {
				InputTokens            int `json:"input_tokens"`
				CacheReadInputTokens   int `json:"cache_read_input_tokens"`
				CacheCreateInputTokens int `json:"cache_creation_input_tokens"`
				OutputTokens           int `json:"output_tokens"`
			} `json:"usage"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(payload), &data); err != nil || data.Result.DurationMs == 0 {
		return "[done]"
	}
	r := data.Result
	return fmt.Sprintf("[done] turns=%d out=%d cached=%d cost=$%.4f dur=%dms",
		r.NumTurns,
		r.Usage.OutputTokens,
		r.Usage.CacheReadInputTokens,
		r.TotalCost,
		int(r.DurationMs),
	)
}

func displayMessagesSnapshot(payload string) string {
	var raw string
	if err := json.Unmarshal([]byte(payload), &raw); err == nil {
		payload = raw
	}
	var msgs []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(payload), &msgs); err != nil {
		return fmt.Sprintf("(%d bytes)", len(payload))
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "assistant" && msgs[i].Content != "" {
			return msgs[i].Content
		}
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Content != "" {
			return fmt.Sprintf("[%s] %s", msgs[i].Role, msgs[i].Content)
		}
	}
	return fmt.Sprintf("(%d messages, no text content)", len(msgs))
}

func listMessages(ctx context.Context, client *sdkclient.Client, printer *output.Printer, sessionID string) error {
	msgs, err := client.Sessions().ListMessages(ctx, sessionID, msgArgs.afterSeq)
	if err != nil {
		return fmt.Errorf("list messages: %w", err)
	}

	if printer.Format() == output.FormatJSON {
		return printer.PrintJSON(msgs)
	}

	w := printer.Writer()
	width := output.TerminalWidthFor(w)
	if width < 40 {
		width = 80
	}

	for _, msg := range msgs {
		display := displayPayload(msg.EventType, msg.Payload)
		if display == "" {
			continue
		}
		var age string
		if msg.CreatedAt != nil {
			age = output.FormatAge(time.Since(*msg.CreatedAt))
		}
		header := fmt.Sprintf("#%-4d  %-28s  %s", msg.Seq, msg.EventType, age)
		fmt.Fprintln(w, header)
		printWrapped(w, display, width, "      ")
		fmt.Fprintln(w)
	}
	return nil
}

func printWrapped(w io.Writer, text string, width int, indent string) {
	text = strings.TrimSpace(text)
	lineWidth := width - len(indent)
	if lineWidth < 20 {
		lineWidth = 20
	}
	words := strings.Fields(text)
	line := indent
	for _, word := range words {
		if len(line)+len(word)+1 > lineWidth && line != indent {
			fmt.Fprintln(w, line)
			line = indent + word
		} else if line == indent {
			line += word
		} else {
			line += " " + word
		}
	}
	if line != indent {
		fmt.Fprintln(w, line)
	}
}

func streamMessages(cmd *cobra.Command, client *sdkclient.Client, sessionID string) error {
	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer cancel()

	fmt.Fprintf(cmd.OutOrStdout(), "Streaming messages for session %s (Ctrl+C to stop)...\n\n", sessionID)

	msgs, stop, err := client.Sessions().WatchMessages(ctx, sessionID, msgArgs.afterSeq)
	if err != nil {
		return fmt.Errorf("watch messages: %w", err)
	}
	defer stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return nil
			}
			printStreamLine(cmd, *msg)
		}
	}
}

func printStreamLine(cmd *cobra.Command, msg sdktypes.SessionMessage) {
	display := displayPayload(msg.EventType, msg.Payload)
	if display == "" {
		return
	}
	w := cmd.OutOrStdout()
	ts := msg.CreatedAt.Format("15:04:05")
	width := output.TerminalWidthFor(w)
	if width < 40 {
		width = 80
	}
	header := fmt.Sprintf("[%s] #%-4d  %s", ts, msg.Seq, msg.EventType)
	fmt.Fprintln(w, header)
	printWrapped(w, display, width, "             ")
	fmt.Fprintln(w)
}
