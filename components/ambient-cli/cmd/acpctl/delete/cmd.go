// Package delete implements the delete subcommand with interactive confirmation.
package delete

import (
	"context"
	"fmt"
	"strings"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/spf13/cobra"
)

var activePhases = map[string]bool{"Pending": true, "Creating": true, "Running": true}

var deleteArgs struct {
	yes bool
	all bool
}

var Cmd = &cobra.Command{
	Use:   "delete <resource> [name]",
	Short: "Delete a resource",
	Long: `Delete a resource by ID, or delete all resources of a type with --all/-A.

Valid resource types:
  project           (aliases: proj)
  project-settings  (aliases: ps)
  session           (aliases: sess)
  agent
  role
  role-binding      (aliases: rolebinding, rb)
  credential        (aliases: cred)

Use --all / -A to delete all resources of the given type.
For sessions, active sessions are stopped before deletion.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: run,
	Example: `  acpctl delete session s1 --yes
  acpctl delete sessions --all --yes
  acpctl delete sessions -Ay`,
}

func init() {
	Cmd.Flags().BoolVarP(&deleteArgs.yes, "yes", "y", false, "Skip confirmation prompt")
	Cmd.Flags().BoolVarP(&deleteArgs.all, "all", "A", false, "Delete all resources of this type")
}

func run(cmd *cobra.Command, cmdArgs []string) error {
	resource := strings.ToLower(cmdArgs[0])

	if deleteArgs.all {
		if len(cmdArgs) > 1 {
			return fmt.Errorf("cannot specify resource name with --all")
		}
		return runDeleteAll(cmd, resource)
	}

	if len(cmdArgs) < 2 {
		return fmt.Errorf("resource name is required (or use --all)")
	}
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

	case "credential", "credentials", "cred", "creds":
		if err := client.Credentials().Delete(ctx, name); err != nil {
			return fmt.Errorf("delete credential %q: %w", name, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "credential/%s deleted\n", name)
		return nil

	default:
		return fmt.Errorf("unknown or non-deletable resource type: %s\nDeletable types: project, project-settings, session, agent, role, role-binding, credential", cmdArgs[0])
	}
}

func runDeleteAll(cmd *cobra.Command, resource string) error {
	switch resource {
	case "session", "sessions", "sess":
		return deleteAllSessions(cmd)
	default:
		return fmt.Errorf("--all is only supported for sessions; got %q", resource)
	}
}

func deleteAllSessions(cmd *cobra.Command) error {
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

	sessions, err := listAllSessions(ctx, client)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "no sessions to delete")
		return nil
	}

	if !deleteArgs.yes {
		fmt.Fprintf(cmd.OutOrStdout(), "Delete all %d sessions? [y/N]: ", len(sessions))
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

	var stopped, deleted, failed int
	for _, s := range sessions {
		if activePhases[s.Phase] {
			if _, stopErr := client.Sessions().Stop(ctx, s.ID); stopErr != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "session/%s: stop failed: %v\n", s.ID, stopErr)
				failed++
				continue
			}
			stopped++
		}

		if delErr := client.Sessions().Delete(ctx, s.ID); delErr != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "session/%s: delete failed: %v\n", s.ID, delErr)
			failed++
			continue
		}
		deleted++
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%d sessions deleted", deleted)
	if stopped > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), " (%d stopped first)", stopped)
	}
	fmt.Fprintln(cmd.OutOrStdout())

	if failed > 0 {
		return fmt.Errorf("%d of %d sessions failed", failed, len(sessions))
	}
	return nil
}

func listAllSessions(ctx context.Context, client *sdkclient.Client) ([]sdktypes.Session, error) {
	var all []sdktypes.Session
	page := 1
	pageSize := 500
	for {
		opts := sdktypes.NewListOptions().Page(page).Size(pageSize).Build()
		list, err := client.Sessions().List(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("list sessions: %w", err)
		}
		all = append(all, list.Items...)
		if len(list.Items) < pageSize {
			break
		}
		page++
	}
	return all, nil
}
