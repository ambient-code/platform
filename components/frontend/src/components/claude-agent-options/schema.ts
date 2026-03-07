import * as z from "zod";

// ---------------------------------------------------------------------------
// Enums / Literals matching claude_agent_sdk 0.1.48 types.py
// ---------------------------------------------------------------------------

export const permissionModeSchema = z.enum([
  "default",
  "acceptEdits",
  "plan",
  "bypassPermissions",
]);

export const effortSchema = z.enum(["low", "medium", "high", "max"]);

export const settingSourceSchema = z.enum(["user", "project", "local"]);

export const sdkBetaSchema = z.enum(["context-1m-2025-08-07"]);

// ---------------------------------------------------------------------------
// SystemPromptPreset — { type: "preset", preset: "claude_code", append?: str }
// ---------------------------------------------------------------------------
export const systemPromptPresetSchema = z.object({
  type: z.literal("preset"),
  preset: z.literal("claude_code"),
  append: z.string().optional(),
});

export const systemPromptSchema = z.union([
  z.string(),
  systemPromptPresetSchema,
]);

// ---------------------------------------------------------------------------
// ToolsPreset — { type: "preset", preset: "claude_code" }
// ---------------------------------------------------------------------------
export const toolsPresetSchema = z.object({
  type: z.literal("preset"),
  preset: z.literal("claude_code"),
});

export const toolsSchema = z.union([
  z.array(z.string()),
  toolsPresetSchema,
]);

// ---------------------------------------------------------------------------
// AgentDefinition
// ---------------------------------------------------------------------------
export const agentDefinitionSchema = z.object({
  description: z.string().min(1, "Description is required"),
  prompt: z.string().min(1, "Prompt is required"),
  tools: z.array(z.string()).nullable().optional(),
  model: z.enum(["sonnet", "opus", "haiku", "inherit"]).nullable().optional(),
});

// ---------------------------------------------------------------------------
// MCP Server configs (stdio / sse / http — sdk is programmatic only)
// ---------------------------------------------------------------------------
export const mcpServerFormConfigSchema = z.discriminatedUnion("type", [
  z.object({
    type: z.literal("stdio"),
    command: z.string().min(1, "Command is required"),
    args: z.array(z.string()).default([]),
    env: z.record(z.string()).default({}),
  }),
  z.object({
    type: z.literal("sse"),
    url: z.string().url("Must be a valid URL"),
    headers: z.record(z.string()).default({}),
  }),
  z.object({
    type: z.literal("http"),
    url: z.string().url("Must be a valid URL"),
    headers: z.record(z.string()).default({}),
  }),
]);

// ---------------------------------------------------------------------------
// ThinkingConfig (adaptive / enabled / disabled)
// ---------------------------------------------------------------------------
export const thinkingConfigSchema = z.discriminatedUnion("type", [
  z.object({ type: z.literal("adaptive") }),
  z.object({
    type: z.literal("enabled"),
    budget_tokens: z.number().int().min(1024).max(128000),
  }),
  z.object({ type: z.literal("disabled") }),
]);

// ---------------------------------------------------------------------------
// SandboxSettings
// ---------------------------------------------------------------------------
export const sandboxNetworkConfigSchema = z.object({
  allowUnixSockets: z.array(z.string()).optional(),
  allowAllUnixSockets: z.boolean().optional(),
  allowLocalBinding: z.boolean().optional(),
  httpProxyPort: z.number().int().optional(),
  socksProxyPort: z.number().int().optional(),
});

export const sandboxIgnoreViolationsSchema = z.object({
  file: z.array(z.string()).optional(),
  network: z.array(z.string()).optional(),
});

export const sandboxSettingsSchema = z.object({
  enabled: z.boolean().default(false),
  autoAllowBashIfSandboxed: z.boolean().default(true),
  excludedCommands: z.array(z.string()).default([]),
  allowUnsandboxedCommands: z.boolean().default(true),
  network: sandboxNetworkConfigSchema.optional(),
  ignoreViolations: sandboxIgnoreViolationsSchema.optional(),
  enableWeakerNestedSandbox: z.boolean().default(false),
});

// ---------------------------------------------------------------------------
// HookEvent
// ---------------------------------------------------------------------------
export const hookEventSchema = z.enum([
  "PreToolUse",
  "PostToolUse",
  "PostToolUseFailure",
  "UserPromptSubmit",
  "Stop",
  "SubagentStop",
  "PreCompact",
  "Notification",
  "SubagentStart",
  "PermissionRequest",
]);

export const hookMatcherFormSchema = z.object({
  matcher: z.string().nullable().optional(),
  timeout: z.number().min(1).optional(),
});

// ---------------------------------------------------------------------------
// SdkPluginConfig
// ---------------------------------------------------------------------------
export const sdkPluginConfigSchema = z.object({
  type: z.literal("local"),
  path: z.string().min(1, "Plugin path is required"),
});

// ---------------------------------------------------------------------------
// OutputFormat
// ---------------------------------------------------------------------------
export const outputFormatSchema = z.object({
  type: z.literal("json_schema").optional(),
  schema: z.record(z.unknown()).optional(),
});

// ---------------------------------------------------------------------------
// Main ClaudeAgentOptions schema (form-safe subset)
// ---------------------------------------------------------------------------
export const claudeAgentOptionsSchema = z.object({
  // Core (model/fallback_model are set via the main session form's model selector)
  system_prompt: systemPromptSchema.optional(),
  permission_mode: permissionModeSchema.optional(),
  max_turns: z.number().int().min(1).optional(),
  max_budget_usd: z.number().min(0).optional(),
  effort: effortSchema.optional(),

  // Tools
  tools: toolsSchema.optional(),
  allowed_tools: z.array(z.string()).default([]),
  disallowed_tools: z.array(z.string()).default([]),

  // MCP Servers
  mcp_servers: z.record(mcpServerFormConfigSchema).default({}),

  // Thinking
  thinking: thinkingConfigSchema.optional(),

  // Session
  continue_conversation: z.boolean().default(false),
  resume: z.string().optional(),
  fork_session: z.boolean().default(false),
  cwd: z.string().optional(),
  add_dirs: z.array(z.string()).default([]),

  // Environment
  env: z.record(z.string()).default({}),
  extra_args: z.record(z.string().nullable()).default({}),
  cli_path: z.string().optional(),
  settings: z.string().optional(),
  setting_sources: z.array(settingSourceSchema).optional(),

  // Advanced
  max_buffer_size: z.number().int().min(1024).optional(),
  include_partial_messages: z.boolean().default(false),
  enable_file_checkpointing: z.boolean().default(false),
  user: z.string().optional(),
  permission_prompt_tool_name: z.string().optional(),

  // Sandbox
  sandbox: sandboxSettingsSchema.optional(),

  // Multi-Agent
  agents: z.record(agentDefinitionSchema).optional(),

  // Hooks
  hooks: z.record(hookEventSchema, z.array(hookMatcherFormSchema)).optional(),

  // Output
  output_format: outputFormatSchema.optional(),

  // Betas & Plugins
  betas: z.array(sdkBetaSchema).default([]),
  plugins: z.array(sdkPluginConfigSchema).default([]),
});

export type ClaudeAgentOptionsForm = z.infer<typeof claudeAgentOptionsSchema>;

export const claudeAgentOptionsDefaults: Partial<ClaudeAgentOptionsForm> = {
  permission_mode: "default",
  effort: "high",
  continue_conversation: false,
  fork_session: false,
  include_partial_messages: false,
  enable_file_checkpointing: false,
  allowed_tools: [],
  disallowed_tools: [],
  mcp_servers: {},
  env: {},
  extra_args: {},
  add_dirs: [],
  betas: [],
  plugins: [],
};
