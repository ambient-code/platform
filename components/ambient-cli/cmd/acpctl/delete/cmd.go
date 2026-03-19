// Package delete implements the delete subcommand with interactive confirmation.
package delete

import (
	"context"
	"fmt"
	"strings"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
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
  project           (aliases: proj)
  project-settings  (aliases: ps)
  session           (aliases: sess)
  agent
  role
  role-binding      (aliases: rolebinding, rb)`,
	Args: cobra.ExactArgs(2),
	RunE: run,
}

func init() {
	Cmd.Flags().BoolVarP(&deleteArgs.yes, "yes", "y", false, "Skip confirmation prompt")
}

func run(cmd *cobra.Command, cmdArgs []string) error {
	resource := strings.ToLower(cmdArgs[0])
	name := cmdArgs[1]

	if !deleteArgs.yes {
		fmt.Fprintf(cmd.OutOrStdout(), "Delete %s/%s? [y/N]: ", resource, name)
		var confirm string
		_, err := fmt.Fscanln(cmd.InOrStdin(), &confirm)
		if err != nil {
			return fmt.Errorf("interactive confirmation required; use --yes/-y to skip")
		}
		if strings.ToLower(confirm) != "y" {
			fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
			return nil
		}
	}

	client, err := connection.NewClientFromConfig()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.GetRequestTimeout())
	defer cancel()

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

	case "session", "sessions", "sess":
		if err := client.Sessions().Delete(ctx, name); err != nil {
			return fmt.Errorf("delete session %q: %w", name, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "session/%s deleted\n", name)
		return nil

	case "agent", "agents":
		if err := client.Agents().Delete(ctx, name); err != nil {
			return fmt.Errorf("delete agent %q: %w", name, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "agent/%s deleted\n", name)
		return nil

	case "role", "roles":
		if err := client.Roles().Delete(ctx, name); err != nil {
			return fmt.Errorf("delete role %q: %w", name, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "role/%s deleted\n", name)
		return nil

	case "role-binding", "role-bindings", "rolebinding", "rolebindings", "rb":
		if err := client.RoleBindings().Delete(ctx, name); err != nil {
			return fmt.Errorf("delete role-binding %q: %w", name, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "role-binding/%s deleted\n", name)
		return nil

	default:
		return fmt.Errorf("unknown or non-deletable resource type: %s\nDeletable types: project, project-settings, session, agent, role, role-binding", cmdArgs[0])
	}
}
