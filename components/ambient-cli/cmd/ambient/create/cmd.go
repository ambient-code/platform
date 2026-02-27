package create

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/output"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "create <resource>",
	Short: "Create a resource",
	Long: `Create a resource.

Valid resource types:
  session    Create an agentic session
  project    Create a project`,
	Args: cobra.MinimumNArgs(1),
	RunE: run,
}

var createArgs struct {
	name        string
	prompt      string
	repoURL     string
	model       string
	maxTokens   int
	temperature float64
	timeout     int
	displayName string
	description string
	outputJSON  bool
}

func init() {
	Cmd.Flags().StringVar(&createArgs.name, "name", "", "Resource name (required)")
	Cmd.Flags().StringVar(&createArgs.prompt, "prompt", "", "Session prompt")
	Cmd.Flags().StringVar(&createArgs.repoURL, "repo-url", "", "Repository URL")
	Cmd.Flags().StringVar(&createArgs.model, "model", "", "LLM model")
	Cmd.Flags().IntVar(&createArgs.maxTokens, "max-tokens", 0, "LLM max tokens")
	Cmd.Flags().Float64Var(&createArgs.temperature, "temperature", 0, "LLM temperature")
	Cmd.Flags().IntVar(&createArgs.timeout, "timeout", 0, "Session timeout in seconds")
	Cmd.Flags().StringVar(&createArgs.displayName, "display-name", "", "Project display name")
	Cmd.Flags().StringVar(&createArgs.description, "description", "", "Project description")
	Cmd.Flags().BoolVarP(&createArgs.outputJSON, "json", "o", false, "Output as JSON")
}

func run(cmd *cobra.Command, cmdArgs []string) error {
	resource := strings.ToLower(cmdArgs[0])

	switch resource {
	case "session", "sess":
		return createSession(cmd)
	case "project", "proj":
		return createProject(cmd)
	default:
		return fmt.Errorf("unknown resource type: %s\nValid types: session, project", cmdArgs[0])
	}
}

func warnUnusedFlags(cmd *cobra.Command, names ...string) {
	for _, name := range names {
		if cmd.Flags().Changed(name) {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: --%s is not applicable to this resource type and will be ignored\n", name)
		}
	}
}

func createSession(cmd *cobra.Command) error {
	warnUnusedFlags(cmd, "display-name", "description")

	if createArgs.name == "" {
		return fmt.Errorf("--name is required")
	}

	builder := sdktypes.NewSessionBuilder().Name(createArgs.name)

	if createArgs.prompt != "" {
		builder = builder.Prompt(createArgs.prompt)
	}
	if createArgs.repoURL != "" {
		builder = builder.RepoURL(createArgs.repoURL)
	}
	if createArgs.model != "" {
		builder = builder.LlmModel(createArgs.model)
	}
	if createArgs.maxTokens > 0 {
		builder = builder.LlmMaxTokens(createArgs.maxTokens)
	}
	if createArgs.temperature > 0 {
		builder = builder.LlmTemperature(createArgs.temperature)
	}
	if createArgs.timeout > 0 {
		builder = builder.Timeout(createArgs.timeout)
	}
	session, err := builder.Build()
	if err != nil {
		return fmt.Errorf("build session: %w", err)
	}

	client, err := connection.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	created, err := client.Sessions().Create(ctx, session)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	if createArgs.outputJSON {
		printer := output.NewPrinter(output.FormatJSON)
		return printer.PrintJSON(created)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "session/%s created\n", created.ID)
	return nil
}

func createProject(cmd *cobra.Command) error {
	warnUnusedFlags(cmd, "prompt", "repo-url", "model", "max-tokens", "temperature", "timeout")

	if createArgs.name == "" {
		return fmt.Errorf("--name is required")
	}

	builder := sdktypes.NewProjectBuilder().Name(createArgs.name)

	if createArgs.displayName != "" {
		builder = builder.DisplayName(createArgs.displayName)
	}
	if createArgs.description != "" {
		builder = builder.Description(createArgs.description)
	}

	project, err := builder.Build()
	if err != nil {
		return fmt.Errorf("build project: %w", err)
	}

	client, err := connection.NewClientFromConfig()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	created, err := client.Projects().Create(ctx, project)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}

	if createArgs.outputJSON {
		printer := output.NewPrinter(output.FormatJSON)
		return printer.PrintJSON(created)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "project/%s created\n", created.ID)
	return nil
}
