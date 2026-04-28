// Package scheduledsession implements CLI commands for managing scheduled sessions.
package scheduledsession

import (
	"context"
	"fmt"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/output"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "scheduled-session",
	Short: "Manage scheduled sessions",
	Long: `Manage project-scoped scheduled sessions.

Subcommands:
  list      List scheduled sessions in a project
  get       Get a specific scheduled session
  create    Create a scheduled session
  update    Update a scheduled session
  delete    Delete a scheduled session
  suspend   Suspend a scheduled session (disable)
  resume    Resume a suspended scheduled session
  trigger   Manually trigger a scheduled session
  runs      List session runs for a scheduled session`,
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

func resolveScheduledSession(ctx context.Context, projectID, arg string) (string, error) {
	client, err := connection.NewClientFromConfig()
	if err != nil {
		return "", err
	}
	ss, err := client.ScheduledSessions().Get(ctx, projectID, arg)
	if err != nil {
		ss, err = client.ScheduledSessions().GetByName(ctx, projectID, arg)
		if err != nil {
			return "", fmt.Errorf("scheduled session %q not found in project %q", arg, projectID)
		}
	}
	return ss.ID, nil
}

// ---------------------------------------------------------------------------
// list
// ---------------------------------------------------------------------------

var listArgs struct {
	projectID    string
	outputFormat string
	limit        int
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List scheduled sessions in a project",
	Example: `  acpctl scheduled-session list
  acpctl scheduled-session list --project-id <id> -o json`,
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

		opts := sdktypes.NewListOptions().Size(listArgs.limit).Build()
		list, err := client.ScheduledSessions().List(ctx, projectID, opts)
		if err != nil {
			return fmt.Errorf("list scheduled sessions: %w", err)
		}

		format, err := output.ParseFormat(listArgs.outputFormat)
		if err != nil {
			return err
		}
		printer := output.NewPrinter(format, cmd.OutOrStdout())

		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(list)
		}
		return printTable(printer, list.Items)
	},
}

// ---------------------------------------------------------------------------
// get
// ---------------------------------------------------------------------------

var getArgs struct {
	projectID    string
	outputFormat string
}

var getCmd = &cobra.Command{
	Use:   "get <name-or-id>",
	Short: "Get a specific scheduled session",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl scheduled-session get my-schedule
  acpctl scheduled-session get <id> --project-id <id> -o json`,
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

		ss, err := client.ScheduledSessions().Get(ctx, projectID, args[0])
		if err != nil {
			ss, err = client.ScheduledSessions().GetByName(ctx, projectID, args[0])
			if err != nil {
				return fmt.Errorf("get scheduled session %q: %w", args[0], err)
			}
		}

		format, err := output.ParseFormat(getArgs.outputFormat)
		if err != nil {
			return err
		}
		printer := output.NewPrinter(format, cmd.OutOrStdout())

		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(ss)
		}
		return printTable(printer, []sdktypes.ScheduledSession{*ss})
	},
}

// ---------------------------------------------------------------------------
// create
// ---------------------------------------------------------------------------

var createArgs struct {
	projectID     string
	name          string
	agentID       string
	schedule      string
	timezone      string
	sessionPrompt string
	description   string
	outputFormat  string
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a scheduled session",
	Example: `  acpctl scheduled-session create --name daily --agent-id <id> --schedule "0 9 * * *"
  acpctl scheduled-session create --name daily --agent-id <id> --schedule "0 9 * * 1-5" --timezone America/New_York`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProject(createArgs.projectID)
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

		builder := sdktypes.NewScheduledSessionBuilder().
			ProjectID(projectID).
			Name(createArgs.name).
			AgentID(createArgs.agentID).
			Schedule(createArgs.schedule)

		if createArgs.timezone != "" {
			builder = builder.Timezone(createArgs.timezone)
		}
		if createArgs.sessionPrompt != "" {
			builder = builder.SessionPrompt(createArgs.sessionPrompt)
		}
		if createArgs.description != "" {
			builder = builder.Description(createArgs.description)
		}

		ss, err := builder.Build()
		if err != nil {
			return fmt.Errorf("build scheduled session: %w", err)
		}

		created, err := client.ScheduledSessions().Create(ctx, projectID, ss)
		if err != nil {
			return fmt.Errorf("create scheduled session: %w", err)
		}

		format, err := output.ParseFormat(createArgs.outputFormat)
		if err != nil {
			return err
		}
		printer := output.NewPrinter(format, cmd.OutOrStdout())

		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(created)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "scheduled-session/%s created\n", created.Name)
		return nil
	},
}

// ---------------------------------------------------------------------------
// update
// ---------------------------------------------------------------------------

var updateArgs struct {
	projectID     string
	name          string
	schedule      string
	timezone      string
	sessionPrompt string
	description   string
}

var updateCmd = &cobra.Command{
	Use:   "update <name-or-id>",
	Short: "Update a scheduled session",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl scheduled-session update my-schedule --schedule "0 10 * * *"
  acpctl scheduled-session update my-schedule --prompt "new instructions"`,
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

		id, err := resolveScheduledSession(ctx, projectID, args[0])
		if err != nil {
			return err
		}

		patch := sdktypes.NewScheduledSessionPatchBuilder()
		if cmd.Flags().Changed("name") {
			patch = patch.Name(updateArgs.name)
		}
		if cmd.Flags().Changed("schedule") {
			patch = patch.Schedule(updateArgs.schedule)
		}
		if cmd.Flags().Changed("timezone") {
			patch = patch.Timezone(updateArgs.timezone)
		}
		if cmd.Flags().Changed("prompt") {
			patch = patch.SessionPrompt(updateArgs.sessionPrompt)
		}
		if cmd.Flags().Changed("description") {
			patch = patch.Description(updateArgs.description)
		}

		updated, err := client.ScheduledSessions().Update(ctx, projectID, id, patch.Build())
		if err != nil {
			return fmt.Errorf("update scheduled session: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "scheduled-session/%s updated\n", updated.Name)
		return nil
	},
}

// ---------------------------------------------------------------------------
// delete
// ---------------------------------------------------------------------------

var deleteArgs struct {
	projectID string
	confirm   bool
}

var deleteCmd = &cobra.Command{
	Use:   "delete <name-or-id>",
	Short: "Delete a scheduled session",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl scheduled-session delete my-schedule --confirm
  acpctl scheduled-session delete <id> --project-id <id> --confirm`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProject(deleteArgs.projectID)
		if err != nil {
			return err
		}
		if !deleteArgs.confirm {
			return fmt.Errorf("add --confirm to delete scheduled-session/%s", args[0])
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

		id, err := resolveScheduledSession(ctx, projectID, args[0])
		if err != nil {
			return err
		}

		if err := client.ScheduledSessions().Delete(ctx, projectID, id); err != nil {
			return fmt.Errorf("delete scheduled session: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "scheduled-session/%s deleted\n", args[0])
		return nil
	},
}

// ---------------------------------------------------------------------------
// suspend
// ---------------------------------------------------------------------------

var suspendArgs struct {
	projectID string
}

var suspendCmd = &cobra.Command{
	Use:   "suspend <name-or-id>",
	Short: "Suspend a scheduled session (disable firing)",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl scheduled-session suspend my-schedule
  acpctl scheduled-session suspend <id> --project-id <id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProject(suspendArgs.projectID)
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

		id, err := resolveScheduledSession(ctx, projectID, args[0])
		if err != nil {
			return err
		}

		ss, err := client.ScheduledSessions().Suspend(ctx, projectID, id)
		if err != nil {
			return fmt.Errorf("suspend scheduled session: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "scheduled-session/%s suspended (enabled=%v)\n", ss.Name, ss.Enabled)
		return nil
	},
}

// ---------------------------------------------------------------------------
// resume
// ---------------------------------------------------------------------------

var resumeArgs struct {
	projectID string
}

var resumeCmd = &cobra.Command{
	Use:   "resume <name-or-id>",
	Short: "Resume a suspended scheduled session",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl scheduled-session resume my-schedule
  acpctl scheduled-session resume <id> --project-id <id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProject(resumeArgs.projectID)
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

		id, err := resolveScheduledSession(ctx, projectID, args[0])
		if err != nil {
			return err
		}

		ss, err := client.ScheduledSessions().Resume(ctx, projectID, id)
		if err != nil {
			return fmt.Errorf("resume scheduled session: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "scheduled-session/%s resumed (enabled=%v)\n", ss.Name, ss.Enabled)
		return nil
	},
}

// ---------------------------------------------------------------------------
// trigger
// ---------------------------------------------------------------------------

var triggerArgs struct {
	projectID string
}

var triggerCmd = &cobra.Command{
	Use:   "trigger <name-or-id>",
	Short: "Manually trigger a scheduled session now",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl scheduled-session trigger my-schedule
  acpctl scheduled-session trigger <id> --project-id <id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProject(triggerArgs.projectID)
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

		id, err := resolveScheduledSession(ctx, projectID, args[0])
		if err != nil {
			return err
		}

		if err := client.ScheduledSessions().Trigger(ctx, projectID, id); err != nil {
			return fmt.Errorf("trigger scheduled session: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "scheduled-session/%s triggered\n", args[0])
		return nil
	},
}

// ---------------------------------------------------------------------------
// runs
// ---------------------------------------------------------------------------

var runsArgs struct {
	projectID    string
	outputFormat string
	limit        int
}

var runsCmd = &cobra.Command{
	Use:   "runs <name-or-id>",
	Short: "List session runs for a scheduled session",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl scheduled-session runs my-schedule
  acpctl scheduled-session runs <id> --project-id <id> -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID, err := resolveProject(runsArgs.projectID)
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

		id, err := resolveScheduledSession(ctx, projectID, args[0])
		if err != nil {
			return err
		}

		opts := sdktypes.NewListOptions().Size(runsArgs.limit).Build()
		list, err := client.ScheduledSessions().Runs(ctx, projectID, id, opts)
		if err != nil {
			return fmt.Errorf("list runs: %w", err)
		}

		format, err := output.ParseFormat(runsArgs.outputFormat)
		if err != nil {
			return err
		}
		printer := output.NewPrinter(format, cmd.OutOrStdout())

		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(list)
		}

		return printRunsTable(printer, list.Items)
	},
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(suspendCmd)
	Cmd.AddCommand(resumeCmd)
	Cmd.AddCommand(triggerCmd)
	Cmd.AddCommand(runsCmd)

	listCmd.Flags().StringVar(&listArgs.projectID, "project-id", "", "Project ID (defaults to configured project)")
	listCmd.Flags().StringVarP(&listArgs.outputFormat, "output", "o", "", "Output format: json")
	listCmd.Flags().IntVar(&listArgs.limit, "limit", 100, "Maximum number of items to return")

	getCmd.Flags().StringVar(&getArgs.projectID, "project-id", "", "Project ID (defaults to configured project)")
	getCmd.Flags().StringVarP(&getArgs.outputFormat, "output", "o", "", "Output format: json")

	createCmd.Flags().StringVar(&createArgs.projectID, "project-id", "", "Project ID (defaults to configured project)")
	createCmd.Flags().StringVar(&createArgs.name, "name", "", "Scheduled session name (required)")
	createCmd.Flags().StringVar(&createArgs.agentID, "agent-id", "", "Agent ID to run (required)")
	createCmd.Flags().StringVar(&createArgs.schedule, "schedule", "", "Cron expression, e.g. \"0 9 * * 1-5\" (required)")
	createCmd.Flags().StringVar(&createArgs.timezone, "timezone", "", "IANA timezone, e.g. America/New_York")
	createCmd.Flags().StringVar(&createArgs.sessionPrompt, "prompt", "", "Session prompt for each run")
	createCmd.Flags().StringVar(&createArgs.description, "description", "", "Description")
	createCmd.Flags().StringVarP(&createArgs.outputFormat, "output", "o", "", "Output format: json")

	updateCmd.Flags().StringVar(&updateArgs.projectID, "project-id", "", "Project ID (defaults to configured project)")
	updateCmd.Flags().StringVar(&updateArgs.name, "name", "", "New name")
	updateCmd.Flags().StringVar(&updateArgs.schedule, "schedule", "", "New cron expression")
	updateCmd.Flags().StringVar(&updateArgs.timezone, "timezone", "", "New timezone")
	updateCmd.Flags().StringVar(&updateArgs.sessionPrompt, "prompt", "", "New session prompt")
	updateCmd.Flags().StringVar(&updateArgs.description, "description", "", "New description")

	deleteCmd.Flags().StringVar(&deleteArgs.projectID, "project-id", "", "Project ID (defaults to configured project)")
	deleteCmd.Flags().BoolVar(&deleteArgs.confirm, "confirm", false, "Confirm deletion")

	suspendCmd.Flags().StringVar(&suspendArgs.projectID, "project-id", "", "Project ID (defaults to configured project)")
	resumeCmd.Flags().StringVar(&resumeArgs.projectID, "project-id", "", "Project ID (defaults to configured project)")
	triggerCmd.Flags().StringVar(&triggerArgs.projectID, "project-id", "", "Project ID (defaults to configured project)")

	runsCmd.Flags().StringVar(&runsArgs.projectID, "project-id", "", "Project ID (defaults to configured project)")
	runsCmd.Flags().StringVarP(&runsArgs.outputFormat, "output", "o", "", "Output format: json")
	runsCmd.Flags().IntVar(&runsArgs.limit, "limit", 100, "Maximum number of items to return")
}

func printTable(printer *output.Printer, items []sdktypes.ScheduledSession) error {
	columns := []output.Column{
		{Name: "ID", Width: 27},
		{Name: "NAME", Width: 24},
		{Name: "SCHEDULE", Width: 20},
		{Name: "TIMEZONE", Width: 20},
		{Name: "ENABLED", Width: 8},
		{Name: "NEXT RUN", Width: 20},
	}

	table := output.NewTable(printer.Writer(), columns)
	table.WriteHeaders()

	for _, ss := range items {
		enabled := "false"
		if ss.Enabled {
			enabled = "true"
		}
		nextRun := ""
		if ss.NextRunAt != nil {
			nextRun = ss.NextRunAt.Format(time.RFC3339)
		}
		table.WriteRow(ss.ID, ss.Name, ss.Schedule, ss.Timezone, enabled, nextRun)
	}
	return nil
}

func printRunsTable(printer *output.Printer, sessions []sdktypes.Session) error {
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
