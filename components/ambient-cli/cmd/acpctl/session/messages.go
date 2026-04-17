package session

import (
	"bufio"
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
	"github.com/spf13/cobra"
)

var msgArgs struct {
	follow           bool
	followContinuous bool
	followJSON       bool
	outputFormat     string
	afterSeq         int
}

var messagesCmd = &cobra.Command{
	Use:   "messages <session-id>",
	Short: "List or stream messages for a session",
	Long: `List or stream messages for a session.

Without -f, fetches a snapshot of messages from the message store.
With -f, connects to the live SSE event stream and renders events
as readable text. The stream ends when the current turn finishes.
With -F, continuously follows the stream, reconnecting after each
turn ends. The stream stays open until Ctrl+C.
With -f --json or -F --json, emits raw AG-UI JSON events instead of text.

Examples:
  acpctl session messages <id>              # snapshot
  acpctl session messages <id> -f           # live stream (ends at turn end)
  acpctl session messages <id> -F           # continuous follow (Ctrl+C to stop)
  acpctl session messages <id> -F --json    # continuous raw AG-UI JSON events
  acpctl session messages <id> -o json      # JSON snapshot
  acpctl session messages <id> --after 5    # messages after seq 5`,
	Args: cobra.ExactArgs(1),
	RunE: runMessages,
}

func init() {
	messagesCmd.Flags().BoolVarP(&msgArgs.follow, "follow", "f", false, "Stream live SSE events until the current turn ends")
	messagesCmd.Flags().BoolVarP(&msgArgs.followContinuous, "follow-continuous", "F", false, "Continuously follow SSE events, reconnecting between turns (Ctrl+C to stop)")
	messagesCmd.Flags().BoolVar(&msgArgs.followJSON, "json", false, "with -f/-F: emit raw AG-UI JSON events instead of text")
	messagesCmd.Flags().StringVarP(&msgArgs.outputFormat, "output", "o", "", "Output format: json")
	messagesCmd.Flags().IntVar(&msgArgs.afterSeq, "after", 0, "Only show messages after this sequence number")
}

func runMessages(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	client, err := connection.NewClientFromConfig()
	if err != nil {
		return err
	}

	if msgArgs.followContinuous {
		return streamMessagesContinuous(cmd, client, sessionID)
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

func extractAGUIText(payload string) string {
	var envelope struct {
		Messages []struct {
			Role    string `json:"role"`
			Content any    `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal([]byte(payload), &envelope); err != nil || len(envelope.Messages) == 0 {
		return ""
	}
	var parts []string
	for _, msg := range envelope.Messages {
		switch v := msg.Content.(type) {
		case string:
			if t := strings.TrimSpace(v); t != "" {
				parts = append(parts, fmt.Sprintf("[%s] %s", msg.Role, t))
			}
		case []any:
			for _, item := range v {
				if block, ok := item.(map[string]any); ok {
					if text, ok := block["text"].(string); ok {
						if t := strings.TrimSpace(text); t != "" {
							parts = append(parts, fmt.Sprintf("[%s] %s", msg.Role, t))
						}
					}
				}
			}
		}
	}
	return strings.Join(parts, "\n")
}

func displayPayload(eventType, payload string) string {
	switch eventType {
	case "user", "assistant":
		if text := extractAGUIText(payload); text != "" {
			return text
		}
		return payload
	case "TEXT_MESSAGE_CONTENT", "REASONING_MESSAGE_CONTENT", "TOOL_CALL_ARGS":
		if d := extractField(payload, "delta"); d != "" {
			return d
		}
	case "TOOL_CALL_START":
		return displayToolCallStart(payload)
	case "TOOL_CALL_RESULT":
		return displayToolCallResult(payload)
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

func displayToolCallStart(payload string) string {
	var raw string
	if err := json.Unmarshal([]byte(payload), &raw); err == nil {
		payload = raw
	}
	var data struct {
		ToolCallName string          `json:"tool_call_name"`
		ToolCallID   string          `json:"tool_call_id"`
		Input        json.RawMessage `json:"input"`
	}
	if err := json.Unmarshal([]byte(payload), &data); err != nil || data.ToolCallName == "" {
		if name := extractField(payload, "tool_call_name"); name != "" {
			return name
		}
		return ""
	}
	if len(data.Input) == 0 || string(data.Input) == "null" || string(data.Input) == "{}" {
		return data.ToolCallName
	}
	var pretty map[string]any
	if err := json.Unmarshal(data.Input, &pretty); err != nil {
		return data.ToolCallName
	}
	var parts []string
	for k, v := range pretty {
		s := fmt.Sprintf("%v", v)
		if len(s) > 60 {
			s = s[:57] + "..."
		}
		parts = append(parts, k+"="+s)
	}
	return data.ToolCallName + "  " + strings.Join(parts, "  ")
}

func displayToolCallResult(payload string) string {
	var raw string
	if err := json.Unmarshal([]byte(payload), &raw); err == nil {
		payload = raw
	}
	var data struct {
		ToolCallID string          `json:"tool_call_id"`
		Content    json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal([]byte(payload), &data); err != nil || len(data.Content) == 0 {
		if c := extractField(payload, "content"); c != "" {
			return c
		}
		return ""
	}
	var contentStr string
	if err := json.Unmarshal(data.Content, &contentStr); err == nil {
		return strings.TrimSpace(contentStr)
	}
	var contentArr []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data.Content, &contentArr); err == nil {
		var parts []string
		for _, c := range contentArr {
			if c.Text != "" {
				parts = append(parts, strings.TrimSpace(c.Text))
			}
		}
		return strings.Join(parts, "\n")
	}
	b, _ := json.MarshalIndent(json.RawMessage(data.Content), "", "  ")
	return string(b)
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
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal([]byte(payload), &msgs); err != nil {
		return fmt.Sprintf("(%d bytes)", len(payload))
	}

	var lines []string
	for _, msg := range msgs {
		if msg.Role == "user" || len(msg.Content) == 0 {
			continue
		}
		var contentStr string
		if err := json.Unmarshal(msg.Content, &contentStr); err == nil {
			if t := strings.TrimSpace(contentStr); t != "" {
				lines = append(lines, fmt.Sprintf("[%s] %s", msg.Role, t))
			}
			continue
		}
		var blocks []struct {
			Type    string          `json:"type"`
			Text    string          `json:"text"`
			Name    string          `json:"name"`
			ID      string          `json:"id"`
			Input   json.RawMessage `json:"input"`
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(msg.Content, &blocks); err != nil {
			continue
		}
		for _, b := range blocks {
			switch b.Type {
			case "text":
				if t := strings.TrimSpace(b.Text); t != "" {
					lines = append(lines, fmt.Sprintf("[%s] %s", msg.Role, t))
				}
			case "tool_use":
				var inputMap map[string]any
				inputSummary := ""
				if len(b.Input) > 0 && json.Unmarshal(b.Input, &inputMap) == nil {
					var kv []string
					for k, v := range inputMap {
						s := fmt.Sprintf("%v", v)
						if len(s) > 60 {
							s = s[:57] + "..."
						}
						kv = append(kv, k+"="+s)
					}
					inputSummary = "  " + strings.Join(kv, "  ")
				}
				lines = append(lines, fmt.Sprintf("[tool_use] %s%s", b.Name, inputSummary))
			case "tool_result":
				var resultText string
				if len(b.Content) > 0 {
					var s string
					if json.Unmarshal(b.Content, &s) == nil {
						resultText = strings.TrimSpace(s)
					} else {
						var arr []struct {
							Type string `json:"type"`
							Text string `json:"text"`
						}
						if json.Unmarshal(b.Content, &arr) == nil {
							var parts []string
							for _, c := range arr {
								if t := strings.TrimSpace(c.Text); t != "" {
									parts = append(parts, t)
								}
							}
							resultText = strings.Join(parts, " | ")
						}
					}
				}
				if len(resultText) > 200 {
					resultText = resultText[:197] + "..."
				}
				lines = append(lines, fmt.Sprintf("[tool_result] %s", resultText))
			}
		}
	}
	if len(lines) == 0 {
		return fmt.Sprintf("(%d messages, no displayable content)", len(msgs))
	}
	return strings.Join(lines, "\n")
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

	stream, err := client.Sessions().StreamEvents(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("stream events: %w", err)
	}
	defer stream.Close()

	fmt.Fprintf(cmd.OutOrStdout(), "Streaming events for session %s (Ctrl+C to stop)...\n\n", sessionID)

	return renderSSEStream(stream, cmd.OutOrStdout(), msgArgs.followJSON, false)
}

func streamMessagesContinuous(cmd *cobra.Command, client *sdkclient.Client, sessionID string) error {
	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer cancel()

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Continuously following session %s (Ctrl+C to stop)...\n\n", sessionID)

	const reconnectDelay = 3 * time.Second

	for {
		stream, err := client.Sessions().StreamEvents(ctx, sessionID)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			fmt.Fprintf(out, "[reconnect] stream not available: %v — retrying in %s\n", err, reconnectDelay)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(reconnectDelay):
				continue
			}
		}

		streamErr := renderSSEStream(stream, out, msgArgs.followJSON, false)
		stream.Close()

		if ctx.Err() != nil {
			return nil
		}

		if streamErr != nil {
			fmt.Fprintf(out, "\n[reconnect] stream ended: %v — reconnecting in %s\n", streamErr, reconnectDelay)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(reconnectDelay):
			}
		} else {
			fmt.Fprintf(out, "\n[reconnect] stream ended cleanly — reconnecting immediately\n")
		}
	}
}

func renderSSEStream(stream io.Reader, out io.Writer, jsonMode, exitOnRunFinished bool) error {
	scanner := bufio.NewScanner(stream)
	var reasoningBuf strings.Builder
	var inText bool
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := line[6:]

		if jsonMode {
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
		case "RUN_FINISHED":
			if inText {
				fmt.Fprintln(out)
				inText = false
			}
			if exitOnRunFinished {
				return nil
			}
		case "RUN_ERROR":
			if inText {
				fmt.Fprintln(out)
				inText = false
			}
			if exitOnRunFinished {
				return fmt.Errorf("run failed")
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
