package main

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/ambient-code/platform/components/ambient-mcp/client"
	"github.com/ambient-code/platform/components/ambient-mcp/tools"
)

func newServer(c *client.Client, transport string) *server.MCPServer {
	s := server.NewMCPServer(
		"ambient-platform",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	registerSessionTools(s, c, transport)
	registerAgentTools(s, c)
	registerProjectTools(s, c)

	return s
}

func registerSessionTools(s *server.MCPServer, c *client.Client, transport string) {
	s.AddTool(
		mcp.NewTool("list_sessions",
			mcp.WithDescription("List sessions visible to the caller, with optional filters."),
			mcp.WithString("project_id", mcp.Description("Filter to sessions belonging to this project ID.")),
			mcp.WithString("phase",
				mcp.Description("Filter by session phase."),
				mcp.Enum("Pending", "Running", "Completed", "Failed"),
			),
			mcp.WithNumber("page", mcp.Description("Page number (1-indexed). Default: 1.")),
			mcp.WithNumber("size", mcp.Description("Page size. Default: 20. Max: 100.")),
		),
		tools.ListSessions(c),
	)

	s.AddTool(
		mcp.NewTool("get_session",
			mcp.WithDescription("Returns full detail for a single session."),
			mcp.WithString("session_id",
				mcp.Description("Session ID."),
				mcp.Required(),
			),
		),
		tools.GetSession(c),
	)

	s.AddTool(
		mcp.NewTool("create_session",
			mcp.WithDescription("Creates and starts a new agentic session. Returns the session in Pending phase."),
			mcp.WithString("project_id",
				mcp.Description("Project ID in which to create the session."),
				mcp.Required(),
			),
			mcp.WithString("prompt",
				mcp.Description("Task prompt for the session."),
				mcp.Required(),
			),
			mcp.WithString("agent_id", mcp.Description("Agent ID to execute the session.")),
			mcp.WithString("model", mcp.Description("LLM model override (e.g. 'claude-sonnet-4-6').")),
			mcp.WithString("parent_session_id", mcp.Description("Calling session ID for agent-to-agent delegation.")),
			mcp.WithString("name", mcp.Description("Human-readable session name.")),
		),
		tools.CreateSession(c),
	)

	s.AddTool(
		mcp.NewTool("push_message",
			mcp.WithDescription("Appends a user message to a session's message log. Supports @mention syntax for agent delegation."),
			mcp.WithString("session_id",
				mcp.Description("ID of the target session."),
				mcp.Required(),
			),
			mcp.WithString("text",
				mcp.Description("Message text. May contain @agent_id or @agent_name mentions to trigger delegation."),
				mcp.Required(),
			),
		),
		tools.PushMessage(c),
	)

	s.AddTool(
		mcp.NewTool("patch_session_labels",
			mcp.WithDescription("Merges key-value label pairs into a session's labels field."),
			mcp.WithString("session_id",
				mcp.Description("ID of the session to update."),
				mcp.Required(),
			),
			mcp.WithObject("labels",
				mcp.Description("Key-value label pairs to merge."),
				mcp.Required(),
			),
		),
		tools.PatchSessionLabels(c),
	)

	s.AddTool(
		mcp.NewTool("patch_session_annotations",
			mcp.WithDescription("Merges key-value annotation pairs into a session's annotations field. Annotations are arbitrary string metadata — a programmable state store scoped to the session lifetime."),
			mcp.WithString("session_id",
				mcp.Description("ID of the session to update."),
				mcp.Required(),
			),
			mcp.WithObject("annotations",
				mcp.Description("Key-value annotation pairs to merge. Keys use reverse-DNS prefix convention (e.g. 'myapp.io/status'). Empty-string values delete a key."),
				mcp.Required(),
			),
		),
		tools.PatchSessionAnnotations(c),
	)

	s.AddTool(
		mcp.NewTool("watch_session_messages",
			mcp.WithDescription("Subscribes to a session's message stream. Returns a subscription_id immediately; messages are pushed as notifications/progress events."),
			mcp.WithString("session_id",
				mcp.Description("ID of the session to watch."),
				mcp.Required(),
			),
			mcp.WithNumber("after_seq", mcp.Description("Deliver only messages with seq > after_seq. Default: 0 (replay all).")),
		),
		tools.WatchSessionMessages(c, transport),
	)

	s.AddTool(
		mcp.NewTool("unwatch_session_messages",
			mcp.WithDescription("Cancels an active watch_session_messages subscription."),
			mcp.WithString("subscription_id",
				mcp.Description("Subscription ID returned by watch_session_messages."),
				mcp.Required(),
			),
		),
		tools.UnwatchSessionMessages(),
	)
}

func registerAgentTools(s *server.MCPServer, c *client.Client) {
	s.AddTool(
		mcp.NewTool("list_agents",
			mcp.WithDescription("Lists agents visible to the caller."),
			mcp.WithString("project_id",
				mcp.Description("Project ID to list agents for."),
				mcp.Required(),
			),
			mcp.WithString("search", mcp.Description("Search filter (e.g. \"name like 'code-%'\").")),
			mcp.WithNumber("page", mcp.Description("Page number (1-indexed). Default: 1.")),
			mcp.WithNumber("size", mcp.Description("Page size. Default: 20. Max: 100.")),
		),
		tools.ListAgents(c),
	)

	s.AddTool(
		mcp.NewTool("get_agent",
			mcp.WithDescription("Returns detail for a single agent by ID or name."),
			mcp.WithString("project_id",
				mcp.Description("Project ID the agent belongs to."),
				mcp.Required(),
			),
			mcp.WithString("agent_id",
				mcp.Description("Agent ID (UUID) or agent name."),
				mcp.Required(),
			),
		),
		tools.GetAgent(c),
	)

	s.AddTool(
		mcp.NewTool("create_agent",
			mcp.WithDescription("Creates a new agent."),
			mcp.WithString("project_id",
				mcp.Description("Project ID to create the agent in."),
				mcp.Required(),
			),
			mcp.WithString("name",
				mcp.Description("Agent name. Must be unique. Alphanumeric, hyphens, underscores only."),
				mcp.Required(),
			),
			mcp.WithString("prompt",
				mcp.Description("System prompt defining the agent's persona and behavior."),
				mcp.Required(),
			),
		),
		tools.CreateAgent(c),
	)

	s.AddTool(
		mcp.NewTool("update_agent",
			mcp.WithDescription("Updates an agent's prompt, labels, or annotations. Creates a new immutable version."),
			mcp.WithString("project_id",
				mcp.Description("Project ID the agent belongs to."),
				mcp.Required(),
			),
			mcp.WithString("agent_id",
				mcp.Description("Agent ID (UUID)."),
				mcp.Required(),
			),
			mcp.WithString("prompt", mcp.Description("New system prompt.")),
			mcp.WithObject("labels", mcp.Description("Labels to merge.")),
			mcp.WithObject("annotations", mcp.Description("Annotations to merge. Empty-string values delete a key.")),
		),
		tools.UpdateAgent(c),
	)

	s.AddTool(
		mcp.NewTool("patch_agent_annotations",
			mcp.WithDescription("Merges key-value annotation pairs into an Agent's annotations. Agent annotations are persistent across sessions — use them for durable agent state."),
			mcp.WithString("project_id",
				mcp.Description("Project ID the agent belongs to."),
				mcp.Required(),
			),
			mcp.WithString("agent_id",
				mcp.Description("Agent ID (UUID) or agent name."),
				mcp.Required(),
			),
			mcp.WithObject("annotations",
				mcp.Description("Key-value annotation pairs to merge. Empty-string values delete a key."),
				mcp.Required(),
			),
		),
		tools.PatchAgentAnnotations(c),
	)
}

func registerProjectTools(s *server.MCPServer, c *client.Client) {
	s.AddTool(
		mcp.NewTool("list_projects",
			mcp.WithDescription("Lists projects visible to the caller."),
			mcp.WithNumber("page", mcp.Description("Page number (1-indexed). Default: 1.")),
			mcp.WithNumber("size", mcp.Description("Page size. Default: 20. Max: 100.")),
		),
		tools.ListProjects(c),
	)

	s.AddTool(
		mcp.NewTool("get_project",
			mcp.WithDescription("Returns detail for a single project by ID or name."),
			mcp.WithString("project_id",
				mcp.Description("Project ID (UUID) or project name."),
				mcp.Required(),
			),
		),
		tools.GetProject(c),
	)

	s.AddTool(
		mcp.NewTool("patch_project_annotations",
			mcp.WithDescription("Merges key-value annotation pairs into a Project's annotations. Project annotations are the widest-scope state store — visible to every agent and session in the project."),
			mcp.WithString("project_id",
				mcp.Description("Project ID (UUID) or project name."),
				mcp.Required(),
			),
			mcp.WithObject("annotations",
				mcp.Description("Key-value annotation pairs to merge. Empty-string values delete a key."),
				mcp.Required(),
			),
		),
		tools.PatchProjectAnnotations(c),
	)
}
