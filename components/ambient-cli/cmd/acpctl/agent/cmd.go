package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/output"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage project-scoped agents",
	Long: `Manage project-scoped agents.

Subcommands:
  list        List agents in a project
  get         Get a specific agent
  create      Create an agent in a project
  update      Update an agent's name, prompt, labels, or annotations
  delete      Delete an agent
  ignite      Start a new session for an agent
  ignition    Preview ignition context (dry run)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func resolveProject(projectID string) (string, error) {
	if projectID != "" {
		return projectID, nil
	}
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	p := cfg.GetProject()
	if p == "" {
		return "", fmt.Errorf("no project set; use --project-id or run 'acpctl config set project <name>'")
	}
	return p, nil
}

func resolveAgent(ctx context.Context, client *sdkclient.Client, projectID, agentArg string) (string, error) {
	if agentArg == "" {
		return "", fmt.Errorf("agent name or ID is required")
	}
	pa, err := client.ProjectAgents().GetInProject(ctx, projectID, agentArg)
	if err != nil {
		pa2, err2 := client.ProjectAgents().GetByProject(ctx, projectID, agentArg)
		if err2 != nil {
			return "", fmt.Errorf("agent %q not found in project %q", agentArg, projectID)
		}
		return pa2.ID, nil
	}
	return pa.ID, nil
}

var listArgs struct {
	projectID    string
	outputFormat string
	limit        int
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List agents in a project",
	Example: `  acpctl agent list
  acpctl agent list --project-id <id> -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProject(listArgs.projectID)
		if err != nil {
			return err
		}

		client, err := connection.NewClientFromConfig()
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.GetRequestTimeout())
		defer cancel()

		format, err := output.ParseFormat(listArgs.outputFormat)
		if err != nil {
			return err
		}
		printer := output.NewPrinter(format, cmd.OutOrStdout())

		opts := sdktypes.NewListOptions().Size(listArgs.limit).Build()
		list, err := client.ProjectAgents().ListByProject(ctx, projectID, opts)
		if err != nil {
			return fmt.Errorf("list agents: %w", err)
		}

		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(list)
		}

		return printAgentTable(printer, list.Items)
	},
}

var getArgs struct {
	projectID    string
	outputFormat string
}

var getCmd = &cobra.Command{
	Use:   "get <name-or-id>",
	Short: "Get a specific agent",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl agent get api
  acpctl agent get api -o json
  acpctl agent get <id> --project-id <id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProject(getArgs.projectID)
		if err != nil {
			return err
		}

		client, err := connection.NewClientFromConfig()
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.GetRequestTimeout())
		defer cancel()

		pa, err := client.ProjectAgents().GetInProject(ctx, projectID, args[0])
		if err != nil {
			pa, err = client.ProjectAgents().GetByProject(ctx, projectID, args[0])
			if err != nil {
				return fmt.Errorf("get agent %q: %w", args[0], err)
			}
		}

		format, err := output.ParseFormat(getArgs.outputFormat)
		if err != nil {
			return err
		}
		printer := output.NewPrinter(format, cmd.OutOrStdout())

		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(pa)
		}
		return printAgentTable(printer, []sdktypes.ProjectAgent{*pa})
	},
}

var createArgs struct {
	projectID    string
	name         string
	prompt       string
	labels       string
	annotations  string
	outputFormat string
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an agent in a project",
	Example: `  acpctl agent create --name my-agent
  acpctl agent create --name my-agent --prompt "You are a code reviewer"
  acpctl agent create --project-id <id> --name my-agent`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProject(createArgs.projectID)
		if err != nil {
			return err
		}
		if createArgs.name == "" {
			return fmt.Errorf("--name is required")
		}

		client, err := connection.NewClientFromConfig()
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.GetRequestTimeout())
		defer cancel()

		builder := sdktypes.NewProjectAgentBuilder().
			ProjectID(projectID).
			Name(createArgs.name)

		if createArgs.prompt != "" {
			builder = builder.Prompt(createArgs.prompt)
		}
		if createArgs.labels != "" {
			builder = builder.Labels(createArgs.labels)
		}
		if createArgs.annotations != "" {
			builder = builder.Annotations(createArgs.annotations)
		}

		agent, err := builder.Build()
		if err != nil {
			return fmt.Errorf("build agent: %w", err)
		}

		created, err := client.ProjectAgents().CreateInProject(ctx, projectID, agent)
		if err != nil {
			return fmt.Errorf("create agent: %w", err)
		}

		format, err := output.ParseFormat(createArgs.outputFormat)
		if err != nil {
			return err
		}
		printer := output.NewPrinter(format, cmd.OutOrStdout())

		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(created)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "agent/%s created\n", created.Name)
		return nil
	},
}

var updateArgs struct {
	projectID   string
	name        string
	prompt      string
	labels      string
	annotations string
}

var updateCmd = &cobra.Command{
	Use:   "update <name-or-id>",
	Short: "Update an agent",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl agent update api --prompt "New instructions"
  acpctl agent update api --name new-name
  acpctl agent update <id> --project-id <id> --prompt "..."`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProject(updateArgs.projectID)
		if err != nil {
			return err
		}

		client, err := connection.NewClientFromConfig()
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.GetRequestTimeout())
		defer cancel()

		agentID, err := resolveAgent(ctx, client, projectID, args[0])
		if err != nil {
			return err
		}

		patch := sdktypes.NewProjectAgentPatchBuilder()
		if cmd.Flags().Changed("name") {
			patch = patch.Name(updateArgs.name)
		}
		if cmd.Flags().Changed("prompt") {
			patch = patch.Prompt(updateArgs.prompt)
		}
		if cmd.Flags().Changed("labels") {
			patch = patch.Labels(updateArgs.labels)
		}
		if cmd.Flags().Changed("annotations") {
			patch = patch.Annotations(updateArgs.annotations)
		}

		updated, err := client.ProjectAgents().UpdateInProject(ctx, projectID, agentID, patch.Build())
		if err != nil {
			return fmt.Errorf("update agent: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "agent/%s updated\n", updated.Name)
		return nil
	},
}

var deleteArgs struct {
	projectID string
	confirm   bool
}

var deleteCmd = &cobra.Command{
	Use:   "delete <name-or-id>",
	Short: "Delete an agent",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl agent delete api --confirm
  acpctl agent delete <id> --project-id <id> --confirm`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProject(deleteArgs.projectID)
		if err != nil {
			return err
		}
		if !deleteArgs.confirm {
			return fmt.Errorf("add --confirm to delete agent/%s", args[0])
		}

		client, err := connection.NewClientFromConfig()
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.GetRequestTimeout())
		defer cancel()

		agentID, err := resolveAgent(ctx, client, projectID, args[0])
		if err != nil {
			return err
		}

		if err := client.ProjectAgents().DeleteInProject(ctx, projectID, agentID); err != nil {
			return fmt.Errorf("delete agent: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "agent/%s deleted\n", args[0])
		return nil
	},
}

var igniteArgs struct {
	projectID string
	prompt    string
}

var igniteCmd = &cobra.Command{
	Use:   "ignite <name-or-id>",
	Short: "Start a new session for an agent",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl agent ignite api
  acpctl agent ignite api --prompt "fix the bug"
  acpctl agent ignite <id> --project-id <id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProject(igniteArgs.projectID)
		if err != nil {
			return err
		}

		client, err := connection.NewClientFromConfig()
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.GetRequestTimeout())
		defer cancel()

		agentID, err := resolveAgent(ctx, client, projectID, args[0])
		if err != nil {
			return err
		}

		resp, err := client.ProjectAgents().Ignite(ctx, projectID, agentID, igniteArgs.prompt)
		if err != nil {
			return fmt.Errorf("ignite agent: %w", err)
		}

		if resp.Session != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "session/%s started (phase: %s)\n", resp.Session.ID, resp.Session.Phase)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "agent/%s ignited\n", args[0])
		}
		return nil
	},
}

var ignitionArgs struct {
	projectID string
}

var ignitionCmd = &cobra.Command{
	Use:   "ignition <name-or-id>",
	Short: "Preview ignition context for an agent (dry run)",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl agent ignition api
  acpctl agent ignition <id> --project-id <id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProject(ignitionArgs.projectID)
		if err != nil {
			return err
		}

		client, err := connection.NewClientFromConfig()
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.GetRequestTimeout())
		defer cancel()

		agentID, err := resolveAgent(ctx, client, projectID, args[0])
		if err != nil {
			return err
		}

		resp, err := client.ProjectAgents().GetIgnition(ctx, projectID, agentID)
		if err != nil {
			return fmt.Errorf("get ignition for agent %q: %w", args[0], err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), resp.IgnitionContext)
		return nil
	},
}

var sessionsArgs struct {
	projectID    string
	outputFormat string
	limit        int
}

var sessionsCmd = &cobra.Command{
	Use:   "sessions <name-or-id>",
	Short: "List session history for an agent",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl agent sessions api
  acpctl agent sessions <id> --project-id <id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProject(sessionsArgs.projectID)
		if err != nil {
			return err
		}

		client, err := connection.NewClientFromConfig()
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.GetRequestTimeout())
		defer cancel()

		agentID, err := resolveAgent(ctx, client, projectID, args[0])
		if err != nil {
			return err
		}

		opts := sdktypes.NewListOptions().Size(sessionsArgs.limit).Build()
		list, err := client.ProjectAgents().Sessions(ctx, projectID, agentID, opts)
		if err != nil {
			return fmt.Errorf("list sessions for agent %q: %w", args[0], err)
		}

		format, err := output.ParseFormat(sessionsArgs.outputFormat)
		if err != nil {
			return err
		}
		printer := output.NewPrinter(format, cmd.OutOrStdout())

		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(list)
		}

		return printSessionTable(printer, list.Items)
	},
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(igniteCmd)
	Cmd.AddCommand(ignitionCmd)
	Cmd.AddCommand(sessionsCmd)

	listCmd.Flags().StringVar(&listArgs.projectID, "project-id", "", "Project ID (defaults to configured project)")
	listCmd.Flags().StringVarP(&listArgs.outputFormat, "output", "o", "", "Output format: json|wide")
	listCmd.Flags().IntVar(&listArgs.limit, "limit", 100, "Maximum number of items to return")

	getCmd.Flags().StringVar(&getArgs.projectID, "project-id", "", "Project ID (defaults to configured project)")
	getCmd.Flags().StringVarP(&getArgs.outputFormat, "output", "o", "", "Output format: json")

	createCmd.Flags().StringVar(&createArgs.projectID, "project-id", "", "Project ID (defaults to configured project)")
	createCmd.Flags().StringVar(&createArgs.name, "name", "", "Agent name (required)")
	createCmd.Flags().StringVar(&createArgs.prompt, "prompt", "", "Standing instructions prompt")
	createCmd.Flags().StringVar(&createArgs.labels, "labels", "", "Labels (JSON string)")
	createCmd.Flags().StringVar(&createArgs.annotations, "annotations", "", "Annotations (JSON string)")
	createCmd.Flags().StringVarP(&createArgs.outputFormat, "output", "o", "", "Output format: json")

	updateCmd.Flags().StringVar(&updateArgs.projectID, "project-id", "", "Project ID (defaults to configured project)")
	updateCmd.Flags().StringVar(&updateArgs.name, "name", "", "New agent name")
	updateCmd.Flags().StringVar(&updateArgs.prompt, "prompt", "", "New standing instructions prompt")
	updateCmd.Flags().StringVar(&updateArgs.labels, "labels", "", "New labels (JSON string)")
	updateCmd.Flags().StringVar(&updateArgs.annotations, "annotations", "", "New annotations (JSON string)")

	deleteCmd.Flags().StringVar(&deleteArgs.projectID, "project-id", "", "Project ID (defaults to configured project)")
	deleteCmd.Flags().BoolVar(&deleteArgs.confirm, "confirm", false, "Confirm deletion")

	igniteCmd.Flags().StringVar(&igniteArgs.projectID, "project-id", "", "Project ID (defaults to configured project)")
	igniteCmd.Flags().StringVar(&igniteArgs.prompt, "prompt", "", "Task prompt for this run")

	ignitionCmd.Flags().StringVar(&ignitionArgs.projectID, "project-id", "", "Project ID (defaults to configured project)")

	sessionsCmd.Flags().StringVar(&sessionsArgs.projectID, "project-id", "", "Project ID (defaults to configured project)")
	sessionsCmd.Flags().StringVarP(&sessionsArgs.outputFormat, "output", "o", "", "Output format: json")
	sessionsCmd.Flags().IntVar(&sessionsArgs.limit, "limit", 100, "Maximum number of items to return")
}

func printAgentTable(printer *output.Printer, agents []sdktypes.ProjectAgent) error {
	columns := []output.Column{
		{Name: "ID", Width: 27},
		{Name: "NAME", Width: 24},
		{Name: "PROJECT", Width: 27},
		{Name: "SESSION", Width: 27},
		{Name: "AGE", Width: 10},
	}

	table := output.NewTable(printer.Writer(), columns)
	table.WriteHeaders()

	for _, a := range agents {
		age := ""
		if a.CreatedAt != nil {
			age = output.FormatAge(time.Since(*a.CreatedAt))
		}
		table.WriteRow(a.ID, a.Name, a.ProjectID, a.CurrentSessionID, age)
	}
	return nil
}

func printSessionTable(printer *output.Printer, sessions []sdktypes.Session) error {
	columns := []output.Column{
		{Name: "ID", Width: 27},
		{Name: "NAME", Width: 32},
		{Name: "PHASE", Width: 12},
		{Name: "AGE", Width: 10},
	}

	table := output.NewTable(printer.Writer(), columns)
	table.WriteHeaders()

	for _, s := range sessions {
		age := ""
		if s.CreatedAt != nil {
			age = output.FormatAge(time.Since(*s.CreatedAt))
		}
		table.WriteRow(s.ID, s.Name, s.Phase, age)
	}
	return nil
}
