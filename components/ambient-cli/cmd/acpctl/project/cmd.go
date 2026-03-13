// Package project implements project management commands
package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/output"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/spf13/cobra"
)

var dnsLabelPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

var Cmd = &cobra.Command{
	Use:   "project [name|command]",
	Short: "Manage projects",
	Long:  `Manage projects in the Ambient Code Platform.`,
	Example: `  # Set current project context (shorthand)
  acpctl project my-project

  # Set current project context (explicit)
  acpctl project set my-project

  # Get current project context
  acpctl project current

  # List all projects
  acpctl project list`,
	Args: cobra.MaximumNArgs(1),
	RunE: projectMain,
}

var setCmd = &cobra.Command{
	Use:   "set <project-name>",
	Short: "Set the current project context",
	Args:  cobra.ExactArgs(1),
	RunE:  setProject,
}

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Display the current project context",
	Args:  cobra.NoArgs,
	RunE:  getCurrentProject,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	Args:  cobra.NoArgs,
	RunE:  listProjects,
}

func init() {
	Cmd.AddCommand(setCmd)
	Cmd.AddCommand(currentCmd)
	Cmd.AddCommand(listCmd)
}

func projectMain(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return getCurrentProject(cmd, args)
	}
	return setProject(cmd, []string{args[0]})
}

func setProject(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	if err := validateProjectName(projectName); err != nil {
		return fmt.Errorf("invalid project name: %w", err)
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

	if _, err := client.Projects().Get(ctx, projectName); err != nil {
		if !isNotFoundError(err) {
			return fmt.Errorf("failed to get project %q: %w", projectName, err)
		}
		return fmt.Errorf("project %q does not exist; create it first with: acpctl create project --name %s", projectName, projectName)
	}

	cfg.Project = projectName

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Switched to project %q\n", projectName)
	return nil
}

func getCurrentProject(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	currentProject := cfg.GetProject()
	if currentProject == "" {
		fmt.Fprintln(cmd.OutOrStdout(), "No project context set")
		fmt.Fprintln(cmd.OutOrStdout(), "Use 'acpctl project set <project-name>' to set a project context")
		return nil
	}

	if env := os.Getenv("AMBIENT_PROJECT"); env != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Current project: %s (from AMBIENT_PROJECT environment variable)\n", currentProject)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Current project: %s\n", currentProject)
	}
	return nil
}

func listProjects(cmd *cobra.Command, args []string) error {
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

	listOpts := sdktypes.NewListOptions().Size(100).Build()
	list, err := client.Projects().List(ctx, listOpts)
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	}

	if len(list.Items) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No projects found")
		return nil
	}

	printer := output.NewPrinter(output.FormatTable)
	return printProjectTable(printer, list.Items)
}

func printProjectTable(printer *output.Printer, projects []sdktypes.Project) error {
	columns := []output.Column{
		{Name: "NAME", Width: 30},
		{Name: "DESCRIPTION", Width: 50},
		{Name: "AGE", Width: 10},
	}

	table := output.NewTable(printer.Writer(), columns)
	table.WriteHeaders()

	for _, p := range projects {
		age := ""
		if p.CreatedAt != nil {
			age = output.FormatAge(time.Since(*p.CreatedAt))
		}
		table.WriteRow(p.Name, p.Description, age)
	}
	return nil
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *sdktypes.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 404
	}
	return false
}

func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if len(name) > 63 {
		return fmt.Errorf("project name must be 63 characters or less")
	}
	if !dnsLabelPattern.MatchString(name) {
		return fmt.Errorf("project name must contain only lowercase letters, numbers, and hyphens, and must start and end with an alphanumeric character")
	}
	return nil
}
