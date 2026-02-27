package delete

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	"github.com/spf13/cobra"
)

var deleteArgs struct {
	yes bool
}

var Cmd = &cobra.Command{
	Use:   "delete <resource> <name>",
	Short: "Delete a resource",
	Long: `Delete a resource by ID.

Valid resource types:
  project    (aliases: proj)
  project-settings (aliases: ps)`,
	Args: cobra.ExactArgs(2),
	RunE: run,
}

func init() {
	Cmd.Flags().BoolVarP(&deleteArgs.yes, "yes", "y", false, "Skip confirmation prompt")
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

	if !deleteArgs.yes {
		fmt.Fprintf(cmd.OutOrStdout(), "Delete %s/%s? [y/N]: ", resource, name)
		var confirm string
		if _, err := fmt.Fscanln(cmd.InOrStdin(), &confirm); err != nil || strings.ToLower(confirm) != "y" {
			fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
			return nil
		}
	}

	switch resource {
	case "project", "projects", "proj":
		if err := client.Projects().Delete(ctx, name); err != nil {
			return fmt.Errorf("delete project %q: %w", name, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "project/%s deleted\n", name)
		return nil

	case "project-settings", "projectsettings", "ps":
		if err := client.ProjectSettings().Delete(ctx, name); err != nil {
			return fmt.Errorf("delete project-settings %q: %w", name, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "project-settings/%s deleted\n", name)
		return nil

	// TODO: Add "session" deletion once the SDK exposes Sessions().Delete().
	default:
		return fmt.Errorf("unknown or non-deletable resource type: %s\nDeletable types: project, project-settings", cmdArgs[0])
	}
}
