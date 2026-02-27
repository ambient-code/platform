// Package describe implements the describe subcommand for detailed resource output.
package describe

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/output"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "describe <resource> <name>",
	Short: "Show detailed information about a resource",
	Long: `Show detailed information about a specific resource.

Valid resource types:
  session    (aliases: sess)
  project    (aliases: proj)`,
	Args: cobra.ExactArgs(2),
	RunE: run,
}

func run(cmd *cobra.Command, cmdArgs []string) error {
	resource := strings.ToLower(cmdArgs[0])
	name := cmdArgs[1]

	client, err := connection.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	printer := output.NewPrinter(output.FormatJSON)

	switch resource {
	case "session", "sessions", "sess":
		session, err := client.Sessions().Get(ctx, name)
		if err != nil {
			return fmt.Errorf("describe session %q: %w", name, err)
		}
		return printer.PrintJSON(session)

	case "project", "projects", "proj":
		project, err := client.Projects().Get(ctx, name)
		if err != nil {
			return fmt.Errorf("describe project %q: %w", name, err)
		}
		return printer.PrintJSON(project)

	case "project-settings", "projectsettings", "ps":
		settings, err := client.ProjectSettings().Get(ctx, name)
		if err != nil {
			return fmt.Errorf("describe project-settings %q: %w", name, err)
		}
		return printer.PrintJSON(settings)

	default:
		return fmt.Errorf("unknown resource type: %s\nValid types: session, project, project-settings", cmdArgs[0])
	}
}
