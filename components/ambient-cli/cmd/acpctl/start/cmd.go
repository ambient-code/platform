// Package start implements the start subcommand for starting a project-agent session.
package start

import (
	"context"
	"fmt"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	"github.com/spf13/cobra"
)

var startArgs struct {
	projectID string
	prompt    string
}

var Cmd = &cobra.Command{
	Use:   "start <project-agent-id>",
	Short: "Start a session for a project-agent (idempotent)",
	Long: `Start a session for a project-agent.

If an active session already exists for this project-agent, it is returned.
If not, a new session is created. Unread inbox messages are drained and
injected into the start context.`,
	Args:    cobra.ExactArgs(1),
	RunE:    run,
	Example: "  acpctl start <pa-id> --project-id <project-id>\n  acpctl start <pa-id> --project-id <project-id> --prompt \"fix the RBAC middleware\"",
}

func init() {
	Cmd.Flags().StringVar(&startArgs.projectID, "project-id", "", "Project ID (required)")
	Cmd.Flags().StringVar(&startArgs.prompt, "prompt", "", "Task prompt for this session run")
}

func run(cmd *cobra.Command, cmdArgs []string) error {
	paID := cmdArgs[0]

	if startArgs.projectID == "" {
		return fmt.Errorf("--project-id is required")
	}

	client, err := connection.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Agents().Start(ctx, startArgs.projectID, paID, startArgs.prompt)
	if err != nil {
		return fmt.Errorf("start agent %q: %w", paID, err)
	}

	if resp.Session != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "session/%s started (phase: %s)\n", resp.Session.ID, resp.Session.Phase)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "project-agent/%s started\n", paID)
	}
	return nil
}
