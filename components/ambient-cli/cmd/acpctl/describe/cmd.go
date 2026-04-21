// Package describe implements the describe subcommand for detailed resource output.
package describe

import (
	"context"
	"fmt"
	"strings"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/output"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "describe <resource> <name>",
	Short: "Show detailed information about a resource",
	Long: `Show detailed information about a specific resource.

Valid resource types:
  session           (aliases: sess)
  project           (aliases: proj)
  project-settings  (aliases: ps)
  user              (aliases: usr)
  agent             (aliases: agents)
  role
  role-binding      (aliases: rb)
  credential        (aliases: cred)`,
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

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.GetRequestTimeout())
	defer cancel()

	printer := output.NewPrinter(output.FormatJSON, cmd.OutOrStdout())

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

	case "user", "users", "usr":
		user, err := client.Users().Get(ctx, name)
		if err != nil {
			return fmt.Errorf("describe user %q: %w", name, err)
		}
		return printer.PrintJSON(user)

	case "agent", "agents":
		agent, err := client.Agents().Get(ctx, name)
		if err != nil {
			return fmt.Errorf("describe agent %q: %w", name, err)
		}
		return printer.PrintJSON(agent)

	case "role", "roles":
		role, err := client.Roles().Get(ctx, name)
		if err != nil {
			return fmt.Errorf("describe role %q: %w", name, err)
		}
		return printer.PrintJSON(role)

	case "role-binding", "role-bindings", "rolebinding", "rb":
		rb, err := client.RoleBindings().Get(ctx, name)
		if err != nil {
			return fmt.Errorf("describe role-binding %q: %w", name, err)
		}
		return printer.PrintJSON(rb)

	case "credential", "credentials", "cred", "creds":
		cred, err := client.Credentials().Get(ctx, name)
		if err != nil {
			return fmt.Errorf("describe credential %q: %w", name, err)
		}
		return printer.PrintJSON(cred)

	default:
		return fmt.Errorf("unknown resource type: %s\nValid types: session, project, project-settings, user, agent, role, role-binding, credential", cmdArgs[0])
	}
}
