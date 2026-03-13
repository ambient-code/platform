// Package create implements the create subcommand for sessions and projects.
package create

import (
	"context"
	"fmt"
	"strings"

	"github.com/ambient-code/platform/components/ambient-cli/pkg/config"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/connection"
	"github.com/ambient-code/platform/components/ambient-cli/pkg/output"
	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "create <resource>",
	Short: "Create a resource",
	Long: `Create a resource.

Valid resource types:
  session       Create an agentic session
  project       Create a project
  agent         Create an agent
  role          Create a role
  role-binding  Create a role binding
  agent-message Send a message to an agent`,
	Args: cobra.MinimumNArgs(1),
	RunE: run,
}

var createArgs struct {
	name             string
	prompt           string
	repoURL          string
	model            string
	maxTokens        int
	temperature      float64
	timeout          int
	displayName      string
	description      string
	outputFormat     string
	projectID        string
	ownerUserID      string
	permissions      string
	userID           string
	roleID           string
	scope            string
	scopeID          string
	recipientAgentID string
	body             string
}

func init() {
	Cmd.Flags().StringVar(&createArgs.name, "name", "", "Resource name")
	Cmd.Flags().StringVar(&createArgs.prompt, "prompt", "", "Session/agent prompt")
	Cmd.Flags().StringVar(&createArgs.repoURL, "repo-url", "", "Repository URL")
	Cmd.Flags().StringVar(&createArgs.model, "model", "", "LLM model")
	Cmd.Flags().IntVar(&createArgs.maxTokens, "max-tokens", 0, "LLM max tokens")
	Cmd.Flags().Float64Var(&createArgs.temperature, "temperature", 0, "LLM temperature")
	Cmd.Flags().IntVar(&createArgs.timeout, "timeout", 0, "Session timeout in seconds")
	Cmd.Flags().StringVar(&createArgs.displayName, "display-name", "", "Display name")
	Cmd.Flags().StringVar(&createArgs.description, "description", "", "Description")
	Cmd.Flags().StringVarP(&createArgs.outputFormat, "output", "o", "", "Output format: json")
	Cmd.Flags().StringVar(&createArgs.projectID, "project-id", "", "Project ID")
	Cmd.Flags().StringVar(&createArgs.ownerUserID, "owner-user-id", "", "Owner user ID (agent)")
	Cmd.Flags().StringVar(&createArgs.permissions, "permissions", "", "Role permissions (JSON)")
	Cmd.Flags().StringVar(&createArgs.userID, "user-id", "", "User ID (role-binding)")
	Cmd.Flags().StringVar(&createArgs.roleID, "role-id", "", "Role ID (role-binding)")
	Cmd.Flags().StringVar(&createArgs.scope, "scope", "", "Scope (role-binding)")
	Cmd.Flags().StringVar(&createArgs.scopeID, "scope-id", "", "Scope ID (role-binding)")
	Cmd.Flags().StringVar(&createArgs.recipientAgentID, "recipient-agent-id", "", "Recipient agent ID (agent-message)")
	Cmd.Flags().StringVar(&createArgs.body, "body", "", "Message body (agent-message)")
}

func run(cmd *cobra.Command, cmdArgs []string) error {
	resource := strings.ToLower(cmdArgs[0])

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

	switch resource {
	case "session", "sess":
		return createSession(cmd, ctx, client)
	case "project", "proj":
		return createProject(cmd, ctx, client)
	case "agent":
		return createAgent(cmd, ctx, client)
	case "role":
		return createRole(cmd, ctx, client)
	case "role-binding", "rolebinding", "rb":
		return createRoleBinding(cmd, ctx, client)
	case "agent-message", "agentmessage", "am":
		return createAgentMessage(cmd, ctx, client)
	default:
		return fmt.Errorf("unknown resource type: %s\nValid types: session, project, agent, role, role-binding, agent-message", cmdArgs[0])
	}
}

func warnUnusedFlags(cmd *cobra.Command, names ...string) {
	for _, name := range names {
		if cmd.Flags().Changed(name) {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: --%s is not applicable to this resource type and will be ignored\n", name)
		}
	}
}

func printCreated(cmd *cobra.Command, kind, id string, obj interface{}) error {
	if createArgs.outputFormat == "json" {
		printer := output.NewPrinter(output.FormatJSON)
		return printer.PrintJSON(obj)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s/%s created\n", kind, id)
	return nil
}

func createSession(cmd *cobra.Command, ctx context.Context, client *sdkclient.Client) error {
	warnUnusedFlags(cmd, "display-name", "project-id", "owner-user-id", "permissions", "user-id", "role-id", "scope", "scope-id", "recipient-agent-id", "body")

	if createArgs.name == "" {
		return fmt.Errorf("--name is required")
	}

	// Get current project from config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	currentProject := cfg.GetProject()
	if currentProject == "" {
		return fmt.Errorf("no project set; run 'acpctl project <name>' first")
	}

	builder := sdktypes.NewSessionBuilder().Name(createArgs.name).ProjectID(currentProject)

	if createArgs.prompt != "" {
		builder = builder.Prompt(createArgs.prompt)
	}
	if createArgs.repoURL != "" {
		builder = builder.RepoURL(createArgs.repoURL)
	}
	if createArgs.model != "" {
		builder = builder.LlmModel(createArgs.model)
	}
	if cmd.Flags().Changed("max-tokens") {
		builder = builder.LlmMaxTokens(createArgs.maxTokens)
	}
	if cmd.Flags().Changed("temperature") {
		builder = builder.LlmTemperature(createArgs.temperature)
	}
	if cmd.Flags().Changed("timeout") {
		builder = builder.Timeout(createArgs.timeout)
	}

	session, err := builder.Build()
	if err != nil {
		return fmt.Errorf("build session: %w", err)
	}

	created, err := client.Sessions().Create(ctx, session)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return printCreated(cmd, "session", created.ID, created)
}

func createProject(cmd *cobra.Command, ctx context.Context, client *sdkclient.Client) error {
	warnUnusedFlags(cmd, "prompt", "repo-url", "model", "max-tokens", "temperature", "timeout", "project-id", "owner-user-id", "permissions", "user-id", "role-id", "scope", "scope-id", "recipient-agent-id", "body")

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

	created, err := client.Projects().Create(ctx, project)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}

	return printCreated(cmd, "project", created.ID, created)
}

func createAgent(cmd *cobra.Command, ctx context.Context, client *sdkclient.Client) error {
	warnUnusedFlags(cmd, "timeout", "user-id", "role-id", "scope", "scope-id", "recipient-agent-id", "body")

	if createArgs.name == "" {
		return fmt.Errorf("--name is required")
	}
	if createArgs.projectID == "" {
		return fmt.Errorf("--project-id is required")
	}
	if createArgs.ownerUserID == "" {
		return fmt.Errorf("--owner-user-id is required")
	}

	builder := sdktypes.NewAgentBuilder().
		Name(createArgs.name).
		ProjectID(createArgs.projectID).
		OwnerUserID(createArgs.ownerUserID)

	if createArgs.prompt != "" {
		builder = builder.Prompt(createArgs.prompt)
	}
	if createArgs.repoURL != "" {
		builder = builder.RepoURL(createArgs.repoURL)
	}
	if createArgs.model != "" {
		builder = builder.LlmModel(createArgs.model)
	}
	if createArgs.displayName != "" {
		builder = builder.DisplayName(createArgs.displayName)
	}
	if createArgs.description != "" {
		builder = builder.Description(createArgs.description)
	}

	agent, err := builder.Build()
	if err != nil {
		return fmt.Errorf("build agent: %w", err)
	}

	created, err := client.Agents().Create(ctx, agent)
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}

	return printCreated(cmd, "agent", created.ID, created)
}

func createRole(cmd *cobra.Command, ctx context.Context, client *sdkclient.Client) error {
	warnUnusedFlags(cmd, "prompt", "repo-url", "model", "max-tokens", "temperature", "timeout", "project-id", "owner-user-id", "user-id", "role-id", "scope", "scope-id", "recipient-agent-id", "body")

	if createArgs.name == "" {
		return fmt.Errorf("--name is required")
	}

	builder := sdktypes.NewRoleBuilder().Name(createArgs.name)

	if createArgs.displayName != "" {
		builder = builder.DisplayName(createArgs.displayName)
	}
	if createArgs.description != "" {
		builder = builder.Description(createArgs.description)
	}
	if createArgs.permissions != "" {
		builder = builder.Permissions(createArgs.permissions)
	}

	role, err := builder.Build()
	if err != nil {
		return fmt.Errorf("build role: %w", err)
	}

	created, err := client.Roles().Create(ctx, role)
	if err != nil {
		return fmt.Errorf("create role: %w", err)
	}

	return printCreated(cmd, "role", created.ID, created)
}

func createRoleBinding(cmd *cobra.Command, ctx context.Context, client *sdkclient.Client) error {
	warnUnusedFlags(cmd, "name", "prompt", "repo-url", "model", "max-tokens", "temperature", "timeout", "display-name", "description", "project-id", "owner-user-id", "permissions", "recipient-agent-id", "body")

	if createArgs.userID == "" {
		return fmt.Errorf("--user-id is required")
	}
	if createArgs.roleID == "" {
		return fmt.Errorf("--role-id is required")
	}
	if createArgs.scope == "" {
		return fmt.Errorf("--scope is required")
	}

	builder := sdktypes.NewRoleBindingBuilder().
		UserID(createArgs.userID).
		RoleID(createArgs.roleID).
		Scope(createArgs.scope)

	if createArgs.scopeID != "" {
		builder = builder.ScopeID(createArgs.scopeID)
	}

	rb, err := builder.Build()
	if err != nil {
		return fmt.Errorf("build role-binding: %w", err)
	}

	created, err := client.RoleBindings().Create(ctx, rb)
	if err != nil {
		return fmt.Errorf("create role-binding: %w", err)
	}

	return printCreated(cmd, "role-binding", created.ID, created)
}

func createAgentMessage(cmd *cobra.Command, ctx context.Context, client *sdkclient.Client) error {
	warnUnusedFlags(cmd, "name", "prompt", "repo-url", "model", "max-tokens", "temperature", "timeout", "display-name", "description", "project-id", "owner-user-id", "permissions", "user-id", "role-id", "scope", "scope-id")

	if createArgs.recipientAgentID == "" {
		return fmt.Errorf("--recipient-agent-id is required")
	}

	builder := sdktypes.NewAgentMessageBuilder().
		RecipientAgentID(createArgs.recipientAgentID)

	if createArgs.body != "" {
		builder = builder.Body(createArgs.body)
	}

	msg, err := builder.Build()
	if err != nil {
		return fmt.Errorf("build agent-message: %w", err)
	}

	created, err := client.AgentMessages().Create(ctx, msg)
	if err != nil {
		return fmt.Errorf("create agent-message: %w", err)
	}

	return printCreated(cmd, "agent-message", created.ID, created)
}
