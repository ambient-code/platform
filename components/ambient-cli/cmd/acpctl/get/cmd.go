// Package get implements the get subcommand for listing and retrieving resources.
package get

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/output"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/spf13/cobra"
)

var args struct {
	outputFormat string
	limit        int
}

var Cmd = &cobra.Command{
	Use:   "get <resource> [name]",
	Short: "Display one or many resources",
	Long: `Display one or many resources.

Valid resource types:
  sessions    (aliases: session, sess)
  projects    (aliases: project, proj)
  project-settings (aliases: projectsettings, ps)`,
	Args:    cobra.RangeArgs(1, 2),
	RunE:    run,
	Example: "  acpctl get sessions\n  acpctl get session my-session-id\n  acpctl get projects -o json",
}

func init() {
	Cmd.Flags().StringVarP(&args.outputFormat, "output", "o", "", "Output format: json|wide")
	Cmd.Flags().IntVar(&args.limit, "limit", 100, "Maximum number of items to return")
}

func run(cmd *cobra.Command, cmdArgs []string) error {
	resource := normalizeResource(cmdArgs[0])

	var name string
	if len(cmdArgs) > 1 {
		name = cmdArgs[1]
	}

	client, err := connection.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	format, err := output.ParseFormat(args.outputFormat)
	if err != nil {
		return err
	}
	printer := output.NewPrinter(format)

	switch resource {
	case "sessions":
		return getSessions(ctx, client, printer, name)
	case "projects":
		return getProjects(ctx, client, printer, name)
	case "project-settings":
		return getProjectSettings(ctx, client, printer, name)
	default:
		return fmt.Errorf("unknown resource type: %s\nValid types: sessions, projects, project-settings", cmdArgs[0])
	}
}

func normalizeResource(r string) string {
	switch strings.ToLower(r) {
	case "session", "sessions", "sess":
		return "sessions"
	case "project", "projects", "proj":
		return "projects"
	case "project-settings", "projectsettings", "project-setting", "ps":
		return "project-settings"
	default:
		return r
	}
}

func getSessions(ctx context.Context, client *sdkclient.Client, printer *output.Printer, name string) error {
	if name != "" {
		session, err := client.Sessions().Get(ctx, name)
		if err != nil {
			return fmt.Errorf("get session %q: %w", name, err)
		}
		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(session)
		}
		return printSessionTable(printer, []sdktypes.Session{*session})
	}

	opts := sdktypes.NewListOptions().Size(args.limit).Build()
	list, err := client.Sessions().List(ctx, opts)
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}

	if printer.Format() == output.FormatJSON {
		return printer.PrintJSON(list)
	}

	return printSessionTable(printer, list.Items)
}

func printSessionTable(printer *output.Printer, sessions []sdktypes.Session) error {
	columns := []output.Column{
		{Name: "ID", Width: 27},
		{Name: "NAME", Width: 30},
		{Name: "PHASE", Width: 12},
		{Name: "MODEL", Width: 16},
		{Name: "AGE", Width: 10},
	}

	table := output.NewTable(printer.Writer(), columns)
	table.WriteHeaders()

	for _, s := range sessions {
		age := ""
		if s.CreatedAt != nil {
			age = formatAge(time.Since(*s.CreatedAt))
		}
		table.WriteRow(s.ID, s.Name, s.Phase, s.LlmModel, age)
	}
	return nil
}

func getProjects(ctx context.Context, client *sdkclient.Client, printer *output.Printer, name string) error {
	if name != "" {
		project, err := client.Projects().Get(ctx, name)
		if err != nil {
			return fmt.Errorf("get project %q: %w", name, err)
		}
		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(project)
		}
		return printProjectTable(printer, []sdktypes.Project{*project})
	}

	opts := sdktypes.NewListOptions().Size(args.limit).Build()
	list, err := client.Projects().List(ctx, opts)
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	}

	if printer.Format() == output.FormatJSON {
		return printer.PrintJSON(list)
	}

	return printProjectTable(printer, list.Items)
}

func printProjectTable(printer *output.Printer, projects []sdktypes.Project) error {
	columns := []output.Column{
		{Name: "ID", Width: 27},
		{Name: "NAME", Width: 30},
		{Name: "DISPLAY NAME", Width: 30},
		{Name: "STATUS", Width: 10},
	}

	table := output.NewTable(printer.Writer(), columns)
	table.WriteHeaders()

	for _, p := range projects {
		table.WriteRow(p.ID, p.Name, p.DisplayName, p.Status)
	}
	return nil
}

func getProjectSettings(ctx context.Context, client *sdkclient.Client, printer *output.Printer, name string) error {
	if name != "" {
		settings, err := client.ProjectSettings().Get(ctx, name)
		if err != nil {
			return fmt.Errorf("get project-settings %q: %w", name, err)
		}
		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(settings)
		}
		return printProjectSettingsTable(printer, []sdktypes.ProjectSettings{*settings})
	}

	opts := sdktypes.NewListOptions().Size(args.limit).Build()
	list, err := client.ProjectSettings().List(ctx, opts)
	if err != nil {
		return fmt.Errorf("list project-settings: %w", err)
	}

	if printer.Format() == output.FormatJSON {
		return printer.PrintJSON(list)
	}

	return printProjectSettingsTable(printer, list.Items)
}

func printProjectSettingsTable(printer *output.Printer, settings []sdktypes.ProjectSettings) error {
	columns := []output.Column{
		{Name: "ID", Width: 27},
		{Name: "PROJECT ID", Width: 27},
		{Name: "AGE", Width: 10},
	}

	table := output.NewTable(printer.Writer(), columns)
	table.WriteHeaders()

	for _, s := range settings {
		age := ""
		if s.CreatedAt != nil {
			age = formatAge(time.Since(*s.CreatedAt))
		}
		table.WriteRow(s.ID, s.ProjectID, age)
	}
	return nil
}

func formatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
