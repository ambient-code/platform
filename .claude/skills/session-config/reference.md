# Session Config Reference — Full Schemas

Complete schema reference for every configuration surface. Read this when generating files.

## 1. CLAUDE.md

Plain markdown, no special syntax. Loaded into every Claude Code session as project instructions.

**Locations**: project root `CLAUDE.md` or `.claude/CLAUDE.md`

## 2. .claude/settings.json

Team-shared project settings. Committed to the repo.

```jsonc
{
  // Model & AI
  "model": "claude-sonnet-4-6",           // Default model
  "availableModels": ["sonnet", "haiku"], // Models agents can choose
  "alwaysThinkingEnabled": true,          // Extended thinking

  // Permissions — tool pattern syntax: Tool, Tool(specifier), Tool(glob *)
  "permissions": {
    "allow": [
      "Bash(npm run *)",                  // Auto-allowed patterns
      "Bash(make *)",
      "Read"
    ],
    "ask": [
      "Bash(git push *)"                  // Require confirmation
    ],
    "deny": [
      "Read(./.env)",                     // Blocked patterns
      "Read(./.env.*)",
      "Read(./secrets/**)",
      "WebFetch"
    ],
    "defaultMode": "acceptEdits",         // acceptEdits | askForPermission | bypassPermissions
    "additionalDirectories": ["../docs/"] // Extra dirs to access
  },

  // Sandbox
  "sandbox": {
    "enabled": true,
    "autoAllowBashIfSandboxed": true,
    "filesystem": {
      "allowWrite": ["/tmp/build", "./output"],
      "denyWrite": ["//etc"],
      "denyRead": ["~/.aws/credentials"]
    },
    "network": {
      "allowedDomains": ["github.com", "*.npmjs.org"],
      "allowLocalBinding": true
    }
  },

  // Environment variables injected into sessions
  "env": {
    "NODE_ENV": "development",
    "CUSTOM_VAR": "value"
  },

  // Attribution
  "includeCoAuthoredBy": true,
  "attribution": {
    "commit": "Co-Authored-By: Claude <noreply@anthropic.com>",
    "pr": "Generated with Claude Code"
  },

  // MCP server controls
  "enableAllProjectMcpServers": true,
  "enabledMcpjsonServers": ["memory"],
  "disabledMcpjsonServers": ["filesystem"],

  // Hooks — see section 7 below
  "hooks": {}
}
```

### Permission Rule Syntax

```
Tool                           // All uses of tool
Tool(specifier)                // Specific uses
Bash(npm run *)                // Commands matching glob
Read(./.env)                   // Specific file
Read(./secrets/**)             // Recursive directory
WebFetch(domain:example.com)   // Requests to domain
Agent(Explore)                 // Block specific subagent type
MCP(server-name:tool-name)     // MCP tool access
```

## 3. .claude/rules/ — Path-Scoped Rules

Markdown files with optional YAML frontmatter. Rules without `paths` apply globally.

```yaml
---
paths:                           # Optional — omit for global rules
  - "src/components/**/*.tsx"    # Glob patterns
  - "src/screens/**/*.tsx"
description: "Optional description"
---

# Rule Title

Rule content in markdown. Claude follows these as instructions
when working on files matching the path patterns.
```

## 4. .claude/skills/ — Skills

Each skill: `<skill-name>/SKILL.md` with YAML frontmatter.

### Complete Frontmatter

```yaml
---
name: skill-name                 # Lowercase, hyphens, numbers (max 64 chars)
description: |                   # What + when to use (max 1024 chars, third person)
  Generates X. Use when creating Y or when user mentions Z.

# Invocation control
argument-hint: "[issue-number]"  # Autocomplete hint
disable-model-invocation: false  # true = only manual /skill invocation
user-invocable: true             # false = only Claude can invoke (hidden from / menu)

# Tool & model
allowed-tools: Read, Grep, Glob, Bash  # Tools allowed without permission prompts
model: sonnet                    # sonnet | opus | haiku | inherit

# Execution context
context: fork                    # Run in isolated subagent
agent: Explore                   # Subagent type when context: fork

# Lifecycle hooks
hooks:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "./scripts/validate.sh"
  PostToolUse:
    - matcher: "Edit|Write"
      hooks:
        - type: command
          command: "./scripts/lint.sh"
---

# Skill body (markdown)

Instructions for Claude when skill is invoked.

## String Substitutions

- $ARGUMENTS — all arguments passed
- $ARGUMENTS[N] or $N — Nth argument (0-indexed)
- ${CLAUDE_SESSION_ID} — current session ID
- !`command` — dynamic context injection (output injected before Claude sees prompt)
```

## 5. .claude/agents/ — Custom Subagents

Markdown files: `.claude/agents/<name>.md` with YAML frontmatter.

### Complete Frontmatter

```yaml
---
name: agent-name                 # Unique identifier
description: |                   # When to delegate (Claude uses for auto-dispatch)
  Reviews code for quality. Use proactively after code changes.

# System prompt alternative (body is preferred)
prompt: "You are a specialized agent for..."

# Tool configuration (use ONE of these)
tools: Read, Glob, Grep, Bash    # Allowlist
disallowedTools: Write, Edit     # Denylist

# Model & permissions
model: sonnet                    # sonnet | opus | haiku | inherit
permissionMode: default          # default | acceptEdits | dontAsk | bypassPermissions | plan
maxTurns: 10                     # Max conversation turns

# Advanced
skills:                          # Skills to preload at startup
  - api-conventions
  - error-handling
memory: project                  # Persistent memory scope: user | project | local
background: false                # Always run in background
isolation: worktree              # Git worktree isolation

# MCP servers
mcpServers:
  - slack                        # Reference existing .mcp.json server
  - database:                    # Inline server definition
      type: stdio
      command: npx
      args: ["-y", "@bytebase/dbhub"]
      env:
        DB_URL: "${DATABASE_URL}"

# Lifecycle hooks
hooks:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "./scripts/validate-readonly.sh"
  Stop:
    - hooks:
        - type: command
          command: "./scripts/cleanup.sh"
---

Agent system prompt (markdown body).
Claude reads this as the agent's instructions.
```

## 6. .mcp.json — MCP Servers

Project-root file declaring MCP servers. Environment variables expand via `${VAR}`.

```jsonc
{
  "mcpServers": {
    // Stdio server — runs a local process
    "memory": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-memory"],
      "env": {
        "MEMORY_DIR": "${HOME}/.memory"
      }
    },

    // HTTP server — remote endpoint
    "github": {
      "type": "http",
      "url": "https://api.githubcopilot.com/mcp/",
      "headers": {
        "Authorization": "Bearer ${GITHUB_TOKEN}"
      }
    },

    // HTTP with OAuth
    "oauth-service": {
      "type": "http",
      "url": "https://mcp.example.com/mcp",
      "oauth": {
        "clientId": "your-client-id",
        "callbackPort": 8080
      }
    }
  }
}
```

### Server Types

| Type | Fields | Use Case |
|------|--------|----------|
| `stdio` | `command`, `args`, `env` | Local process (npx, python, binary) |
| `http` | `url`, `headers`, `oauth` | Remote HTTP endpoint |
| `sse` | `url`, `headers` | SSE endpoint (deprecated, prefer http) |

## 7. Hooks — Lifecycle Automation

Configured in `.claude/settings.json` under the `hooks` key.

### Hook Events

| Event | Matcher Input | Fires When |
|-------|---------------|------------|
| `on-session-start` | — | Session begins |
| `on-session-end` | — | Session ends |
| `on-edit` | File path | After file edit |
| `on-bash` | Command | After bash execution |
| `PreToolUse` | Tool name | Before tool runs |
| `PostToolUse` | Tool name | After tool runs |
| `SubagentStart` | Agent type | Subagent begins |
| `SubagentStop` | Agent type | Subagent completes |

### Hook Types

```jsonc
// Command — runs a shell command
{ "type": "command", "command": "./scripts/hook.sh", "args": ["arg1"] }
// Exit codes: 0=allow, 1=error, 2=block (PreToolUse only)

// HTTP — calls a webhook
{ "type": "http", "url": "https://hooks.example.com/endpoint", "method": "POST",
  "headers": { "Authorization": "Bearer ${TOKEN}" } }

// Prompt — AI-evaluated gate
{ "type": "prompt", "model": "sonnet", "prompt": "Is this bash command safe?" }

// Agent — delegates to a subagent
{ "type": "agent", "agent": "security-validator", "prompt": "Verify this operation" }
```

### Example Configuration

```jsonc
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          { "type": "command", "command": "./scripts/validate-bash.sh" }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          { "type": "command", "command": "./scripts/run-linter.sh" }
        ]
      }
    ],
    "on-session-start": [
      { "type": "command", "command": "./scripts/setup-env.sh" }
    ]
  }
}
```

## 8. .ambient/workflows/ — ACP Workflows

JSON files under `.ambient/workflows/`. Each defines a reusable workflow.

### Session-Config Repo Format

```jsonc
{
  "name": "Workflow Name",            // Display name in ACP UI
  "description": "What it does",      // Shown in workflow selector
  "systemPrompt": "Agent persona",    // Short rules (use CLAUDE.md for long instructions)
  "startupPrompt": "Initial prompt",  // Hidden prompt sent at session start
  "results": [                        // Optional: artifact output paths
    { "path": "artifacts/output/" }
  ]
}
```

### Full Workflow Format (standalone workflow repos)

```jsonc
{
  "name": "Workflow Name",
  "description": "What it does",
  "systemPrompt": "Agent persona and hard rules",
  "startupPrompt": "Hidden prompt — agent responds to this",
  "rubric": {                         // Optional: quality evaluation
    "activationPrompt": "When to trigger evaluation",
    "schema": {
      "type": "object",
      "properties": {
        "completeness": { "type": "number", "description": "Score 1-5" },
        "quality": { "type": "number", "description": "Score 1-5" }
      },
      "required": ["completeness", "quality"]
    }
  }
}
```

Standalone workflow repos use `.ambient/ambient.json` at the repo root and can include `CLAUDE.md`, `.claude/`, and a `rubric.md` file for evaluation criteria.
