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

var sessionArgs struct {
	name        string
	prompt      string
	repoURL     string
	model       string
	maxTokens   int
	temperature float64
	timeout     int
	interactive bool
	outputJSON  bool
}

var projectArgs struct {
	name        string
	displayName string
	description string
	outputJSON  bool
}

func init() {
	Cmd.Flags().StringVar(&sessionArgs.name, "name", "", "Resource name (required)")
	Cmd.Flags().StringVar(&sessionArgs.prompt, "prompt", "", "Session prompt")
	Cmd.Flags().StringVar(&sessionArgs.repoURL, "repo-url", "", "Repository URL")
	Cmd.Flags().StringVar(&sessionArgs.model, "model", "", "LLM model")
	Cmd.Flags().IntVar(&sessionArgs.maxTokens, "max-tokens", 0, "LLM max tokens")
	Cmd.Flags().Float64Var(&sessionArgs.temperature, "temperature", 0, "LLM temperature")
	Cmd.Flags().IntVar(&sessionArgs.timeout, "timeout", 0, "Session timeout in seconds")
	Cmd.Flags().BoolVar(&sessionArgs.interactive, "interactive", false, "Interactive mode")
	Cmd.Flags().StringVar(&projectArgs.displayName, "display-name", "", "Project display name")
	Cmd.Flags().StringVar(&projectArgs.description, "description", "", "Project description")
	Cmd.Flags().BoolVarP(&sessionArgs.outputJSON, "json", "o", false, "Output as JSON")
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

func createSession(cmd *cobra.Command) error {
	if sessionArgs.name == "" {
		return fmt.Errorf("--name is required")
	}

	builder := sdktypes.NewSessionBuilder().Name(sessionArgs.name)

	if sessionArgs.prompt != "" {
		builder.Prompt(sessionArgs.prompt)
	}
	if sessionArgs.repoURL != "" {
		builder.RepoURL(sessionArgs.repoURL)
	}
	if sessionArgs.model != "" {
		builder.LlmModel(sessionArgs.model)
	}
	if sessionArgs.maxTokens > 0 {
		builder.LlmMaxTokens(sessionArgs.maxTokens)
	}
	if sessionArgs.temperature > 0 {
		builder.LlmTemperature(sessionArgs.temperature)
	}
	if sessionArgs.timeout > 0 {
		builder.Timeout(sessionArgs.timeout)
	}
	if sessionArgs.interactive {
		builder.Interactive(true)
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

	if sessionArgs.outputJSON {
		printer := output.NewPrinter(output.FormatJSON)
		return printer.PrintJSON(created)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "session/%s created\n", created.ID)
	return nil
}

func createProject(cmd *cobra.Command) error {
	if sessionArgs.name == "" {
		return fmt.Errorf("--name is required")
	}

	builder := sdktypes.NewProjectBuilder().Name(sessionArgs.name)

	if projectArgs.displayName != "" {
		builder.DisplayName(projectArgs.displayName)
	}
	if projectArgs.description != "" {
		builder.Description(projectArgs.description)
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

	if sessionArgs.outputJSON {
		printer := output.NewPrinter(output.FormatJSON)
		return printer.PrintJSON(created)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "project/%s created\n", created.ID)
	return nil
}
