// Package application implements the application subcommand for managing GitOps applications.
package application

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
	Use:     "application",
	Aliases: []string{"app", "apps"},
	Short:   "Manage GitOps applications",
	Long: `Manage GitOps applications that continuously sync agent fleet definitions from git.

Subcommands:
  list        List applications
  get         Get a specific application
  create      Create an application
  update      Update an application's fields
  delete      Delete an application
  sync        Trigger a sync for an application
  refresh     Refresh an application's sync status`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var listArgs struct {
	outputFormat string
	limit        int
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List applications",
	Example: `  acpctl application list
  acpctl app list -o json
  acpctl app list --limit 10`,
	RunE: func(cmd *cobra.Command, args []string) error {
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
		list, err := client.Applications().List(ctx, opts)
		if err != nil {
			return fmt.Errorf("list applications: %w", err)
		}

		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(list)
		}

		return printApplicationTable(printer, list.Items)
	},
}

var getArgs struct {
	outputFormat string
}

var getCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a specific application",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl application get <id>
  acpctl app get <id> -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		app, err := client.Applications().Get(ctx, args[0])
		if err != nil {
			return fmt.Errorf("get application %q: %w", args[0], err)
		}

		format, err := output.ParseFormat(getArgs.outputFormat)
		if err != nil {
			return err
		}
		printer := output.NewPrinter(format, cmd.OutOrStdout())

		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(app)
		}
		return printApplicationTable(printer, []sdktypes.Application{*app})
	},
}

var createArgs struct {
	name                 string
	sourceRepoURL        string
	sourcePath           string
	sourceTargetRevision string
	destinationProject   string
	destinationURL       string
	credentialID         string
	autoSync             bool
	autoPrune            bool
	selfHeal             bool
	retryLimit           int32
	labels               string
	annotations          string
	outputFormat         string
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an application",
	Example: `  acpctl app create --name my-fleet --source-repo-url https://github.com/org/repo --source-path manifests/ --destination-project my-project
  acpctl app create --name prod-sync --source-repo-url https://github.com/org/repo --source-path . --destination-project prod --auto-sync --auto-prune`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if createArgs.name == "" {
			return fmt.Errorf("--name is required")
		}
		if createArgs.sourceRepoURL == "" {
			return fmt.Errorf("--source-repo-url is required")
		}
		if createArgs.sourcePath == "" {
			return fmt.Errorf("--source-path is required")
		}
		if createArgs.destinationProject == "" {
			return fmt.Errorf("--destination-project is required")
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

		builder := sdktypes.NewApplicationBuilder().
			Name(createArgs.name).
			SourceRepoURL(createArgs.sourceRepoURL).
			SourcePath(createArgs.sourcePath).
			DestinationProject(createArgs.destinationProject)

		if createArgs.sourceTargetRevision != "" {
			builder = builder.SourceTargetRevision(createArgs.sourceTargetRevision)
		}
		if createArgs.destinationURL != "" {
			builder = builder.DestinationAmbientURL(createArgs.destinationURL)
		}
		if createArgs.credentialID != "" {
			builder = builder.CredentialID(createArgs.credentialID)
		}
		if createArgs.autoSync {
			builder = builder.AutoSync(true)
		}
		if createArgs.autoPrune {
			builder = builder.AutoPrune(true)
		}
		if createArgs.selfHeal {
			builder = builder.SelfHeal(true)
		}
		if cmd.Flags().Changed("retry-limit") {
			builder = builder.RetryLimit(createArgs.retryLimit)
		}
		if createArgs.labels != "" {
			builder = builder.Labels(createArgs.labels)
		}
		if createArgs.annotations != "" {
			builder = builder.Annotations(createArgs.annotations)
		}

		app, err := builder.Build()
		if err != nil {
			return fmt.Errorf("build application: %w", err)
		}

		created, err := client.Applications().Create(ctx, app)
		if err != nil {
			return fmt.Errorf("create application: %w", err)
		}

		format, err := output.ParseFormat(createArgs.outputFormat)
		if err != nil {
			return err
		}
		printer := output.NewPrinter(format, cmd.OutOrStdout())

		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(created)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "application/%s created\n", created.Name)
		return nil
	},
}

var updateArgs struct {
	name                 string
	sourceRepoURL        string
	sourcePath           string
	sourceTargetRevision string
	destinationProject   string
	destinationURL       string
	credentialID         string
	autoSync             bool
	autoPrune            bool
	selfHeal             bool
	retryLimit           int32
	labels               string
	annotations          string
}

var updateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an application",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl app update <id> --auto-sync --auto-prune --self-heal
  acpctl app update <id> --source-target-revision main`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		patch := sdktypes.NewApplicationPatchBuilder()
		if cmd.Flags().Changed("name") {
			patch = patch.Name(updateArgs.name)
		}
		if cmd.Flags().Changed("source-repo-url") {
			patch = patch.SourceRepoURL(updateArgs.sourceRepoURL)
		}
		if cmd.Flags().Changed("source-path") {
			patch = patch.SourcePath(updateArgs.sourcePath)
		}
		if cmd.Flags().Changed("source-target-revision") {
			patch = patch.SourceTargetRevision(updateArgs.sourceTargetRevision)
		}
		if cmd.Flags().Changed("destination-project") {
			patch = patch.DestinationProject(updateArgs.destinationProject)
		}
		if cmd.Flags().Changed("destination-url") {
			patch = patch.DestinationAmbientURL(updateArgs.destinationURL)
		}
		if cmd.Flags().Changed("credential-id") {
			patch = patch.CredentialID(updateArgs.credentialID)
		}
		if cmd.Flags().Changed("auto-sync") {
			patch = patch.AutoSync(updateArgs.autoSync)
		}
		if cmd.Flags().Changed("auto-prune") {
			patch = patch.AutoPrune(updateArgs.autoPrune)
		}
		if cmd.Flags().Changed("self-heal") {
			patch = patch.SelfHeal(updateArgs.selfHeal)
		}
		if cmd.Flags().Changed("retry-limit") {
			patch = patch.RetryLimit(updateArgs.retryLimit)
		}
		if cmd.Flags().Changed("labels") {
			patch = patch.Labels(updateArgs.labels)
		}
		if cmd.Flags().Changed("annotations") {
			patch = patch.Annotations(updateArgs.annotations)
		}

		updated, err := client.Applications().Update(ctx, args[0], patch.Build())
		if err != nil {
			return fmt.Errorf("update application: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "application/%s updated\n", updated.Name)
		return nil
	},
}

var deleteArgs struct {
	confirm bool
}

var deleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Short:   "Delete an application",
	Args:    cobra.ExactArgs(1),
	Example: `  acpctl app delete <id> --confirm`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !deleteArgs.confirm {
			return fmt.Errorf("add --confirm to delete application/%s", args[0])
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

		if err := client.Applications().Delete(ctx, args[0]); err != nil {
			return fmt.Errorf("delete application: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "application/%s deleted\n", args[0])
		return nil
	},
}

var syncArgs struct {
	outputFormat string
}

var syncCmd = &cobra.Command{
	Use:   "sync <id>",
	Short: "Trigger a sync for an application",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl app sync <id>
  acpctl app sync <id> -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		patch := sdktypes.NewApplicationPatchBuilder().OperationPhase("Running").Build()
		updated, err := client.Applications().Update(ctx, args[0], patch)
		if err != nil {
			return fmt.Errorf("sync application: %w", err)
		}

		format, err := output.ParseFormat(syncArgs.outputFormat)
		if err != nil {
			return err
		}
		printer := output.NewPrinter(format, cmd.OutOrStdout())

		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(updated)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "application/%s sync triggered\n", updated.Name)
		return nil
	},
}

var refreshArgs struct {
	outputFormat string
}

var refreshCmd = &cobra.Command{
	Use:   "refresh <id>",
	Short: "Refresh an application's sync status",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl app refresh <id>
  acpctl app refresh <id> -o json`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		patch := sdktypes.NewApplicationPatchBuilder().SyncStatus("Unknown").Build()
		updated, err := client.Applications().Update(ctx, args[0], patch)
		if err != nil {
			return fmt.Errorf("refresh application: %w", err)
		}

		format, err := output.ParseFormat(refreshArgs.outputFormat)
		if err != nil {
			return err
		}
		printer := output.NewPrinter(format, cmd.OutOrStdout())

		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(updated)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "application/%s refresh triggered\n", updated.Name)
		return nil
	},
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(syncCmd)
	Cmd.AddCommand(refreshCmd)

	listCmd.Flags().StringVarP(&listArgs.outputFormat, "output", "o", "", "Output format: json")
	listCmd.Flags().IntVar(&listArgs.limit, "limit", 100, "Maximum number of items to return")

	getCmd.Flags().StringVarP(&getArgs.outputFormat, "output", "o", "", "Output format: json")

	createCmd.Flags().StringVar(&createArgs.name, "name", "", "Application name (required)")
	createCmd.Flags().StringVar(&createArgs.sourceRepoURL, "source-repo-url", "", "Git repository URL (required)")
	createCmd.Flags().StringVar(&createArgs.sourcePath, "source-path", "", "Path within the repository (required)")
	createCmd.Flags().StringVar(&createArgs.sourceTargetRevision, "source-target-revision", "", "Git branch, tag, or commit")
	createCmd.Flags().StringVar(&createArgs.destinationProject, "destination-project", "", "Target project (required)")
	createCmd.Flags().StringVar(&createArgs.destinationURL, "destination-url", "", "Target Ambient instance URL")
	createCmd.Flags().StringVar(&createArgs.credentialID, "credential-id", "", "Credential ID for private repos")
	createCmd.Flags().BoolVar(&createArgs.autoSync, "auto-sync", false, "Enable automatic sync")
	createCmd.Flags().BoolVar(&createArgs.autoPrune, "auto-prune", false, "Enable automatic pruning")
	createCmd.Flags().BoolVar(&createArgs.selfHeal, "self-heal", false, "Enable self-healing")
	createCmd.Flags().Int32Var(&createArgs.retryLimit, "retry-limit", 0, "Maximum retry attempts")
	createCmd.Flags().StringVar(&createArgs.labels, "labels", "", "Labels (JSON string)")
	createCmd.Flags().StringVar(&createArgs.annotations, "annotations", "", "Annotations (JSON string)")
	createCmd.Flags().StringVarP(&createArgs.outputFormat, "output", "o", "", "Output format: json")

	updateCmd.Flags().StringVar(&updateArgs.name, "name", "", "New application name")
	updateCmd.Flags().StringVar(&updateArgs.sourceRepoURL, "source-repo-url", "", "New git repository URL")
	updateCmd.Flags().StringVar(&updateArgs.sourcePath, "source-path", "", "New path within the repository")
	updateCmd.Flags().StringVar(&updateArgs.sourceTargetRevision, "source-target-revision", "", "New git branch, tag, or commit")
	updateCmd.Flags().StringVar(&updateArgs.destinationProject, "destination-project", "", "New target project")
	updateCmd.Flags().StringVar(&updateArgs.destinationURL, "destination-url", "", "New target Ambient instance URL")
	updateCmd.Flags().StringVar(&updateArgs.credentialID, "credential-id", "", "New credential ID")
	updateCmd.Flags().BoolVar(&updateArgs.autoSync, "auto-sync", false, "Enable automatic sync")
	updateCmd.Flags().BoolVar(&updateArgs.autoPrune, "auto-prune", false, "Enable automatic pruning")
	updateCmd.Flags().BoolVar(&updateArgs.selfHeal, "self-heal", false, "Enable self-healing")
	updateCmd.Flags().Int32Var(&updateArgs.retryLimit, "retry-limit", 0, "Maximum retry attempts")
	updateCmd.Flags().StringVar(&updateArgs.labels, "labels", "", "New labels (JSON string)")
	updateCmd.Flags().StringVar(&updateArgs.annotations, "annotations", "", "New annotations (JSON string)")

	deleteCmd.Flags().BoolVar(&deleteArgs.confirm, "confirm", false, "Confirm deletion")

	syncCmd.Flags().StringVarP(&syncArgs.outputFormat, "output", "o", "", "Output format: json")

	refreshCmd.Flags().StringVarP(&refreshArgs.outputFormat, "output", "o", "", "Output format: json")
}

func printApplicationTable(printer *output.Printer, applications []sdktypes.Application) error {
	columns := []output.Column{
		{Name: "ID", Width: 27},
		{Name: "NAME", Width: 24},
		{Name: "REPO", Width: 36},
		{Name: "PROJECT", Width: 18},
		{Name: "SYNC", Width: 10},
		{Name: "HEALTH", Width: 10},
		{Name: "AGE", Width: 10},
	}

	table := output.NewTable(printer.Writer(), columns)
	table.WriteHeaders()

	for _, a := range applications {
		age := ""
		if a.CreatedAt != nil {
			age = output.FormatAge(time.Since(*a.CreatedAt))
		}
		table.WriteRow(a.ID, a.Name, a.SourceRepoURL, a.DestinationProject, a.SyncStatus, a.HealthStatus, age)
	}
	return nil
}
