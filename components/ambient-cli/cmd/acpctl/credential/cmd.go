// Package credential implements the credential subcommand for managing credentials.
package credential

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
	Use:   "credential",
	Short: "Manage credentials",
	Long: `Manage credentials for external service integrations.

Subcommands:
  list        List credentials
  get         Get a specific credential
  create      Create a credential
  update      Update a credential's fields
  delete      Delete a credential
  token       Retrieve the token for a credential`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var listArgs struct {
	outputFormat string
	limit        int
	provider     string
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List credentials",
	Example: `  acpctl credential list
  acpctl credential list --provider github -o json`,
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
		if listArgs.provider != "" {
			opts.Search = fmt.Sprintf("provider='%s'", listArgs.provider)
		}
		list, err := client.Credentials().List(ctx, opts)
		if err != nil {
			return fmt.Errorf("list credentials: %w", err)
		}

		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(list)
		}

		return printCredentialTable(printer, list.Items)
	},
}

var getArgs struct {
	outputFormat string
}

var getCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a specific credential",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl credential get <id>
  acpctl credential get <id> -o json`,
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

		credential, err := client.Credentials().Get(ctx, args[0])
		if err != nil {
			return fmt.Errorf("get credential %q: %w", args[0], err)
		}

		format, err := output.ParseFormat(getArgs.outputFormat)
		if err != nil {
			return err
		}
		printer := output.NewPrinter(format, cmd.OutOrStdout())

		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(credential)
		}
		return printCredentialTable(printer, []sdktypes.Credential{*credential})
	},
}

var createArgs struct {
	name         string
	provider     string
	token        string
	description  string
	url          string
	email        string
	labels       string
	annotations  string
	outputFormat string
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a credential",
	Example: `  acpctl credential create --name github-main --provider github --token ghp_xxx
  acpctl credential create --name jira-corp --provider jira --url https://corp.atlassian.net --token xxx`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if createArgs.name == "" {
			return fmt.Errorf("--name is required")
		}
		if createArgs.provider == "" {
			return fmt.Errorf("--provider is required")
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

		builder := sdktypes.NewCredentialBuilder().
			Name(createArgs.name).
			Provider(createArgs.provider)

		if createArgs.token != "" {
			builder = builder.Token(createArgs.token)
		}
		if createArgs.description != "" {
			builder = builder.Description(createArgs.description)
		}
		if createArgs.url != "" {
			builder = builder.URL(createArgs.url)
		}
		if createArgs.email != "" {
			builder = builder.Email(createArgs.email)
		}
		if createArgs.labels != "" {
			builder = builder.Labels(createArgs.labels)
		}
		if createArgs.annotations != "" {
			builder = builder.Annotations(createArgs.annotations)
		}

		cred, err := builder.Build()
		if err != nil {
			return fmt.Errorf("build credential: %w", err)
		}

		created, err := client.Credentials().Create(ctx, cred)
		if err != nil {
			return fmt.Errorf("create credential: %w", err)
		}

		format, err := output.ParseFormat(createArgs.outputFormat)
		if err != nil {
			return err
		}
		printer := output.NewPrinter(format, cmd.OutOrStdout())

		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(created)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "credential/%s created\n", created.Name)
		return nil
	},
}

var updateArgs struct {
	name        string
	token       string
	description string
	url         string
	email       string
	labels      string
	annotations string
}

var updateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a credential",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl credential update <id> --token ghp_newtoken
  acpctl credential update <id> --description "updated description"`,
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

		patch := sdktypes.NewCredentialPatchBuilder()
		if cmd.Flags().Changed("name") {
			patch = patch.Name(updateArgs.name)
		}
		if cmd.Flags().Changed("token") {
			patch = patch.Token(updateArgs.token)
		}
		if cmd.Flags().Changed("description") {
			patch = patch.Description(updateArgs.description)
		}
		if cmd.Flags().Changed("url") {
			patch = patch.URL(updateArgs.url)
		}
		if cmd.Flags().Changed("email") {
			patch = patch.Email(updateArgs.email)
		}
		if cmd.Flags().Changed("labels") {
			patch = patch.Labels(updateArgs.labels)
		}
		if cmd.Flags().Changed("annotations") {
			patch = patch.Annotations(updateArgs.annotations)
		}

		updated, err := client.Credentials().Update(ctx, args[0], patch.Build())
		if err != nil {
			return fmt.Errorf("update credential: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "credential/%s updated\n", updated.Name)
		return nil
	},
}

var deleteArgs struct {
	confirm bool
}

var deleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Short:   "Delete a credential",
	Args:    cobra.ExactArgs(1),
	Example: `  acpctl credential delete <id> --confirm`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !deleteArgs.confirm {
			return fmt.Errorf("add --confirm to delete credential/%s", args[0])
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

		if err := client.Credentials().Delete(ctx, args[0]); err != nil {
			return fmt.Errorf("delete credential: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "credential/%s deleted\n", args[0])
		return nil
	},
}

var tokenArgs struct {
	outputFormat string
}

var tokenCmd = &cobra.Command{
	Use:   "token <id>",
	Short: "Retrieve the token for a credential",
	Args:  cobra.ExactArgs(1),
	Example: `  acpctl credential token <id>
  acpctl credential token <id> -o json`,
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

		resp, err := client.Credentials().GetToken(ctx, args[0])
		if err != nil {
			return fmt.Errorf("get token for credential %q: %w", args[0], err)
		}

		format, err := output.ParseFormat(tokenArgs.outputFormat)
		if err != nil {
			return err
		}
		printer := output.NewPrinter(format, cmd.OutOrStdout())

		if printer.Format() == output.FormatJSON {
			return printer.PrintJSON(resp)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", resp.Token)
		return nil
	},
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(tokenCmd)

	listCmd.Flags().StringVarP(&listArgs.outputFormat, "output", "o", "", "Output format: json")
	listCmd.Flags().IntVar(&listArgs.limit, "limit", 100, "Maximum number of items to return")
	listCmd.Flags().StringVar(&listArgs.provider, "provider", "", "Filter by provider (github|gitlab|jira|google|kubeconfig)")

	getCmd.Flags().StringVarP(&getArgs.outputFormat, "output", "o", "", "Output format: json")

	createCmd.Flags().StringVar(&createArgs.name, "name", "", "Credential name (required)")
	createCmd.Flags().StringVar(&createArgs.provider, "provider", "", "Provider (github|gitlab|jira|google|kubeconfig) (required)")
	createCmd.Flags().StringVar(&createArgs.token, "token", "", "Secret token or API key")
	createCmd.Flags().StringVar(&createArgs.description, "description", "", "Description")
	createCmd.Flags().StringVar(&createArgs.url, "url", "", "Service URL")
	createCmd.Flags().StringVar(&createArgs.email, "email", "", "Associated email")
	createCmd.Flags().StringVar(&createArgs.labels, "labels", "", "Labels (JSON string)")
	createCmd.Flags().StringVar(&createArgs.annotations, "annotations", "", "Annotations (JSON string)")
	createCmd.Flags().StringVarP(&createArgs.outputFormat, "output", "o", "", "Output format: json")

	updateCmd.Flags().StringVar(&updateArgs.name, "name", "", "New credential name")
	updateCmd.Flags().StringVar(&updateArgs.token, "token", "", "New secret token or API key")
	updateCmd.Flags().StringVar(&updateArgs.description, "description", "", "New description")
	updateCmd.Flags().StringVar(&updateArgs.url, "url", "", "New service URL")
	updateCmd.Flags().StringVar(&updateArgs.email, "email", "", "New associated email")
	updateCmd.Flags().StringVar(&updateArgs.labels, "labels", "", "New labels (JSON string)")
	updateCmd.Flags().StringVar(&updateArgs.annotations, "annotations", "", "New annotations (JSON string)")

	deleteCmd.Flags().BoolVar(&deleteArgs.confirm, "confirm", false, "Confirm deletion")

	tokenCmd.Flags().StringVarP(&tokenArgs.outputFormat, "output", "o", "", "Output format: json")
}

func printCredentialTable(printer *output.Printer, credentials []sdktypes.Credential) error {
	columns := []output.Column{
		{Name: "ID", Width: 27},
		{Name: "NAME", Width: 24},
		{Name: "PROVIDER", Width: 12},
		{Name: "DESCRIPTION", Width: 32},
		{Name: "AGE", Width: 10},
	}

	table := output.NewTable(printer.Writer(), columns)
	table.WriteHeaders()

	for _, c := range credentials {
		age := ""
		if c.CreatedAt != nil {
			age = output.FormatAge(time.Since(*c.CreatedAt))
		}
		table.WriteRow(c.ID, c.Name, c.Provider, c.Description, age)
	}
	return nil
}
