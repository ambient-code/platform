package session

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
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

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	token := cfg.GetToken()
	if token == "" {
		return fmt.Errorf("not logged in; run 'acpctl login' first")
	}

	apiURL := strings.TrimRight(cfg.GetAPIUrl(), "/")
	url := fmt.Sprintf("%s/api/ambient/v1/sessions/%s/events", apiURL, sessionID)

	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("connect to event stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s", resp.Status)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Streaming events for session %s (Ctrl+C to stop)...\n\n", sessionID)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "data: "):
			fmt.Fprintln(cmd.OutOrStdout(), line[6:])
		case strings.HasPrefix(line, ": "):
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return fmt.Errorf("stream error: %w", scanErr)
	}
	return nil
}
