# Spec: Ambient Platform MCP Server

**Date:** 2026-03-22
**Status:** Design
**Guide:** `mcp-server.guide.md` — implementation waves, gap table, build commands, run log

---

## Overview

The Ambient Platform MCP Server exposes the platform's resource API as a set of structured tools conforming to the [Model Context Protocol (MCP) 2024-11-05](https://spec.modelcontextprotocol.io/specification/2024-11-05/). It has two deployment modes:

1. **Sidecar** — runs alongside the Claude Code CLI in every runner Job pod. Claude Code connects via stdio. The sidecar's auth token is injected from the pod environment.
2. **Public endpoint** — exposed through `ambient-api-server` at `POST /api/ambient/v1/mcp`. Clients authenticate with the same bearer token used for all other API calls. The frontend session panel connects here.

The MCP server has no direct Kubernetes access. All operations proxy through `ambient-api-server`, inheriting the full RBAC model.

---

## Component

**Location:** `components/ambient-mcp/`

**Language:** Go 1.24+

**Library:** `mark3labs/mcp-go v0.45.0`

**Constraint:** No direct Kubernetes API access. Reads and writes go through the platform REST API only.

**Image:** `localhost/vteam_ambient_mcp:latest`

### Directory Structure

```
components/ambient-mcp/
├── main.go              # Entrypoint; transport selected by MCP_TRANSPORT env var
├── server.go            # MCP server init, capability declaration, tool registration
├── Dockerfile           # ubi9/go-toolset builder → ubi9/ubi-minimal runtime, UID 1001
├── go.mod               # module: github.com/ambient-code/platform/components/ambient-mcp
├── client/
│   └── client.go        # Thin HTTP client wrapping ambient-api-server
├── mention/
│   └── resolve.go       # @mention extraction and agent resolution
└── tools/
    ├── helpers.go        # jsonResult / errResult utilities
    ├── sessions.go       # Session tool handlers + annotation merge
    ├── agents.go         # Agent tool handlers + annotation merge
    ├── projects.go       # Project tool handlers + annotation merge
    └── watch.go          # watch_session_messages, unwatch_session_messages
```

### Configuration

| Environment Variable | Required | Default | Description |
|---|---|---|---|
| `AMBIENT_API_URL` | Yes | — | Base URL of the ambient-api-server |
| `AMBIENT_TOKEN` | Yes | — | Bearer token. In sidecar mode, injected from the pod's service environment. In public-endpoint mode, forwarded from the HTTP request. |
| `MCP_TRANSPORT` | No | `stdio` | `stdio` for sidecar mode, `sse` for public endpoint |
| `MCP_BIND_ADDR` | No | `:8090` | Bind address for SSE mode |

---

## MCP Protocol

### Initialize

The server declares the following capabilities in its `initialize` response:

```json
{
  "protocolVersion": "2024-11-05",
  "capabilities": {
    "tools": {}
  },
  "serverInfo": {
    "name": "ambient-platform",
    "version": "1.0.0"
  }
}
```

Resources and prompts capabilities are not declared. All platform operations are exposed as tools.

### Transports

| Mode | Transport | Binding |
|---|---|---|
| Sidecar (runner pod) | stdio | stdin/stdout of the sidecar process |
| Public endpoint | SSE over HTTP | `MCP_BIND_ADDR` (proxied through `ambient-api-server`) |

In SSE mode, the server responds to:
- `GET /sse` — SSE event stream (client → server messages via query param or POST)
- `POST /message` — client sends JSON-RPC messages; server replies via the SSE stream

### Error Format

All tool errors follow MCP's structured error response. The `content` array contains a single text item with a JSON-encoded error body matching the platform's `Error` schema:

```json
{
  "isError": true,
  "content": [
    {
      "type": "text",
      "text": "{\"code\": \"SESSION_NOT_FOUND\", \"reason\": \"no session with id abc123\", \"operation_id\": \"get_session\"}"
    }
  ]
}
```

---

## Tool Definitions

### `list_sessions`

Lists sessions visible to the caller, with optional filters.

**RBAC required:** `sessions:list`

**Backed by:** `GET /api/ambient/v1/sessions`

**Input schema:**

```json
{
  "type": "object",
  "properties": {
    "project_id": {
      "type": "string",
      "description": "Filter to sessions belonging to this project ID. If omitted, returns all sessions visible to the caller's token."
    },
    "phase": {
      "type": "string",
      "enum": ["Pending", "Running", "Completed", "Failed"],
      "description": "Filter by session phase."
    },
    "page": {
      "type": "integer",
      "description": "Page number (1-indexed). Default: 1."
    },
    "size": {
      "type": "integer",
      "description": "Page size. Default: 20. Max: 100."
    }
  }
}
```

**Return value:** JSON-encoded `SessionList`. Content type `text`, format:

```json
{
  "kind": "SessionList",
  "page": 1,
  "size": 20,
  "total": 3,
  "items": [
    {
      "id": "3BEaN6kqawvTNUIXoSMcgOQvUDj",
      "name": "my-session",
      "project_id": "demo-6640",
      "phase": "Running",
      "created_at": "2026-03-21T10:00:00Z",
      "llm_model": "claude-sonnet-4-6"
    }
  ]
}
```

**Errors:**

| Code | Condition |
|---|---|
| `UNAUTHORIZED` | Token invalid or expired |
| `FORBIDDEN` | Token lacks `sessions:list` |
| `INTERNAL` | Backend returned 5xx |

---

### `get_session`

Returns full detail for a single session.

**RBAC required:** `sessions:get`

**Backed by:** `GET /api/ambient/v1/sessions/{id}`

**Input schema:**

```json
{
  "type": "object",
  "required": ["session_id"],
  "properties": {
    "session_id": {
      "type": "string",
      "description": "Session ID (UUID)."
    }
  }
}
```

**Return value:** JSON-encoded `Session`.

**Errors:**

| Code | Condition |
|---|---|
| `SESSION_NOT_FOUND` | No session with that ID |
| `UNAUTHORIZED` | Token invalid or expired |
| `FORBIDDEN` | Token lacks `sessions:get` |

---

### `create_session`

Creates and starts a new agentic session. The session enters `Pending` phase immediately and transitions to `Running` when the operator schedules the runner pod.

**RBAC required:** `sessions:create`

**Backed by:** `POST /api/ambient/v1/sessions`, then `POST /api/ambient/v1/sessions/{id}/start`

**Input schema:**

```json
{
  "type": "object",
  "required": ["project_id", "prompt"],
  "properties": {
    "project_id": {
      "type": "string",
      "description": "Project (Kubernetes namespace) in which to create the session. Must match the caller's token scope."
    },
    "prompt": {
      "type": "string",
      "description": "Task prompt for the session. Passed as Session.prompt to the runner."
    },
    "agent_id": {
      "type": "string",
      "description": "ID of the ProjectAgent to execute this session. If omitted, the project's default agent is used."
    },
    "model": {
      "type": "string",
      "description": "LLM model override (e.g. 'claude-sonnet-4-6'). If omitted, the agent's configured model is used."
    },
    "parent_session_id": {
      "type": "string",
      "description": "ID of the calling session. Used for agent-to-agent delegation. Sets Session.parent_session_id. The child session appears in the parent's lineage."
    },
    "name": {
      "type": "string",
      "description": "Human-readable name for the session. If omitted, a name is generated from the prompt (first 40 chars, slugified)."
    }
  }
}
```

**Return value:** JSON-encoded `Session` in `Pending` phase.

**Behavior:**
- Creates the Session CR via `POST /api/ambient/v1/sessions`
- Immediately calls `POST /api/ambient/v1/sessions/{id}/start`
- Returns the created Session object
- Does not wait for the session to reach `Running` — the caller must poll or `watch_session_messages` to observe progress

**Errors:**

| Code | Condition |
|---|---|
| `INVALID_REQUEST` | `project_id` or `prompt` missing |
| `AGENT_NOT_FOUND` | `agent_id` specified but does not exist |
| `FORBIDDEN` | Token lacks `sessions:create` or is not scoped to `project_id` |
| `INTERNAL` | Backend returned 5xx |

---

### `push_message`

Appends a user message to a session's message log. Supports `@mention` syntax for agent-to-agent delegation (see [@mention Pattern](#mention-pattern)).

**RBAC required:** `sessions:patch`

**Backed by:** `POST /api/ambient/v1/sessions/{id}/messages`

**Input schema:**

```json
{
  "type": "object",
  "required": ["session_id", "text"],
  "properties": {
    "session_id": {
      "type": "string",
      "description": "ID of the target session."
    },
    "text": {
      "type": "string",
      "description": "Message text. May contain @agent_id or @agent_name mentions to trigger agent delegation."
    }
  }
}
```

**Return value:** JSON object with the following fields:

```json
{
  "message": { /* SessionMessage */ },
  "delegated_session": null
}
```

If the message contained a resolvable `@mention`, `delegated_session` is the newly created child `Session`; otherwise it is `null`.

**Behavior:**
- Pushes the message to the session with `event_type: "user"`
- If `text` contains one or more `@mention` tokens, each is resolved and a child session is created (see [@mention Pattern](#mention-pattern))
- The original message is pushed as-is (including the `@mention` text) before delegation

**Errors:**

| Code | Condition |
|---|---|
| `SESSION_NOT_FOUND` | No session with that ID |
| `SESSION_NOT_RUNNING` | Session is in `Completed` or `Failed` phase. Messages cannot be pushed to terminal sessions. |
| `MENTION_NOT_RESOLVED` | `@mention` token could not be matched to any agent |
| `FORBIDDEN` | Token lacks `sessions:patch` |

---

### `patch_session_labels`

Merges key-value label pairs into a session's `labels` field. Existing labels not present in the patch are preserved.

**RBAC required:** `sessions:patch`

**Backed by:** `PATCH /api/ambient/v1/sessions/{id}`

**Input schema:**

```json
{
  "type": "object",
  "required": ["session_id", "labels"],
  "properties": {
    "session_id": {
      "type": "string",
      "description": "ID of the session to update."
    },
    "labels": {
      "type": "object",
      "additionalProperties": {
        "type": "string"
      },
      "description": "Key-value label pairs to merge. Keys and values must be non-empty strings. Keys may not contain '=' or whitespace.",
      "example": {"env": "prod", "team": "platform"}
    }
  }
}
```

**Return value:** JSON-encoded updated `Session`.

**Behavior:**
- Reads existing `Session.labels` (JSON-decoded from its stored string form)
- Merges the provided labels (provided keys overwrite existing values)
- Writes back via `PATCH /api/ambient/v1/sessions/{id}` with the merged label map serialized to JSON string

**Errors:**

| Code | Condition |
|---|---|
| `SESSION_NOT_FOUND` | No session with that ID |
| `INVALID_LABEL_KEY` | A key contains `=` or whitespace |
| `INVALID_LABEL_VALUE` | A value is empty |
| `FORBIDDEN` | Token lacks `sessions:patch` |

---

### `watch_session_messages`

Subscribes to a session's message stream. Returns a `subscription_id` immediately. The MCP server then pushes `notifications/progress` events to the client as messages arrive. The subscription terminates automatically when the session reaches a terminal phase (`Completed` or `Failed`).

**RBAC required:** `sessions:get`

**Backed by:** `GET /api/ambient/v1/sessions/{id}/messages` with `Accept: text/event-stream`

**Input schema:**

```json
{
  "type": "object",
  "required": ["session_id"],
  "properties": {
    "session_id": {
      "type": "string",
      "description": "ID of the session to watch."
    },
    "after_seq": {
      "type": "integer",
      "description": "Deliver only messages with seq > after_seq. Default: 0 (replay all messages, then stream new ones)."
    }
  }
}
```

**Return value:**

```json
{
  "subscription_id": "sub_abc123",
  "session_id": "3BEaN6kqawvTNUIXoSMcgOQvUDj"
}
```

**Progress notification shape** (pushed to client via `notifications/progress`):

```json
{
  "method": "notifications/progress",
  "params": {
    "progressToken": "{subscription_id}",
    "progress": {
      "session_id": "3BEaN6kqawvTNUIXoSMcgOQvUDj",
      "message": {
        "id": "msg_xyz",
        "session_id": "3BEaN6kqawvTNUIXoSMcgOQvUDj",
        "seq": 42,
        "event_type": "TEXT_MESSAGE_CONTENT",
        "payload": "delta='Hello from the agent'",
        "created_at": "2026-03-21T10:01:00Z"
      }
    }
  }
}
```

**Terminal notification** (sent when session reaches `Completed` or `Failed`):

```json
{
  "method": "notifications/progress",
  "params": {
    "progressToken": "{subscription_id}",
    "progress": {
      "session_id": "3BEaN6kqawvTNUIXoSMcgOQvUDj",
      "terminal": true,
      "phase": "Completed"
    }
  }
}
```

**Behavior:**
- The MCP server opens an SSE connection to the backend for the given session
- Messages received on the SSE stream are forwarded as `notifications/progress` events
- The server polls session phase every 5 seconds; when `Completed` or `Failed` is observed, sends the terminal notification and closes the subscription
- The client may call `unwatch_session_messages` at any time to cancel early

**Errors:**

| Code | Condition |
|---|---|
| `SESSION_NOT_FOUND` | No session with that ID |
| `FORBIDDEN` | Token lacks `sessions:get` |
| `TRANSPORT_NOT_SUPPORTED` | Client is connected via stdio transport; streaming notifications require SSE |

---

### `unwatch_session_messages`

Cancels an active `watch_session_messages` subscription.

**Input schema:**

```json
{
  "type": "object",
  "required": ["subscription_id"],
  "properties": {
    "subscription_id": {
      "type": "string",
      "description": "Subscription ID returned by watch_session_messages."
    }
  }
}
```

**Return value:**

```json
{ "cancelled": true }
```

**Errors:**

| Code | Condition |
|---|---|
| `SUBSCRIPTION_NOT_FOUND` | No active subscription with that ID |

---

### `list_agents`

Lists agents visible to the caller.

**RBAC required:** `agents:list`

**Backed by:** `GET /api/ambient/v1/agents`

**Input schema:**

```json
{
  "type": "object",
  "properties": {
    "search": {
      "type": "string",
      "description": "Search filter in SQL-like syntax (e.g. \"name like 'code-%'\"). Forwarded as the 'search' query parameter."
    },
    "page": {
      "type": "integer",
      "description": "Page number (1-indexed). Default: 1."
    },
    "size": {
      "type": "integer",
      "description": "Page size. Default: 20. Max: 100."
    }
  }
}
```

**Return value:** JSON-encoded `AgentList`.

```json
{
  "kind": "AgentList",
  "page": 1,
  "size": 20,
  "total": 2,
  "items": [
    {
      "id": "agent-uuid-1",
      "name": "code-review",
      "owner_user_id": "user-uuid",
      "version": 3
    }
  ]
}
```

Note: `Agent.prompt` is write-only in the API. `list_agents` and `get_agent` do not return prompt text.

**Errors:**

| Code | Condition |
|---|---|
| `UNAUTHORIZED` | Token invalid |
| `FORBIDDEN` | Token lacks `agents:list` |

---

### `get_agent`

Returns detail for a single agent by ID or name.

**RBAC required:** `agents:get`

**Backed by:** `GET /api/ambient/v1/agents/{id}` (by ID), or `GET /api/ambient/v1/agents?search=name='{name}'` (by name)

**Input schema:**

```json
{
  "type": "object",
  "required": ["agent_id"],
  "properties": {
    "agent_id": {
      "type": "string",
      "description": "Agent ID (UUID) or agent name. If the value does not parse as a UUID, it is treated as a name and resolved via search."
    }
  }
}
```

**Return value:** JSON-encoded `Agent`.

**Errors:**

| Code | Condition |
|---|---|
| `AGENT_NOT_FOUND` | No agent matches the ID or name |
| `AMBIGUOUS_AGENT_NAME` | Name search returns more than one agent |
| `FORBIDDEN` | Token lacks `agents:get` |

---

### `create_agent`

Creates a new agent.

**RBAC required:** `agents:create`

**Backed by:** `POST /api/ambient/v1/agents`

**Input schema:**

```json
{
  "type": "object",
  "required": ["name", "prompt"],
  "properties": {
    "name": {
      "type": "string",
      "description": "Agent name. Must be unique for the owning user. Alphanumeric, hyphens, underscores only."
    },
    "prompt": {
      "type": "string",
      "description": "System prompt defining the agent's persona and behavior."
    }
  }
}
```

**Return value:** JSON-encoded `Agent` at `version: 1`.

**Errors:**

| Code | Condition |
|---|---|
| `AGENT_NAME_CONFLICT` | An agent with this name already exists for the caller |
| `INVALID_REQUEST` | `name` contains disallowed characters or `prompt` is empty |
| `FORBIDDEN` | Token lacks `agents:create` |

---

### `update_agent`

Updates an agent's prompt. Creates a new immutable version (increments `Agent.version`). Prior versions are preserved.

**RBAC required:** `agents:patch`

**Backed by:** `PATCH /api/ambient/v1/agents/{id}`

**Input schema:**

```json
{
  "type": "object",
  "required": ["agent_id", "prompt"],
  "properties": {
    "agent_id": {
      "type": "string",
      "description": "Agent ID (UUID)."
    },
    "prompt": {
      "type": "string",
      "description": "New system prompt. Creates a new agent version."
    }
  }
}
```

**Return value:** JSON-encoded `Agent` at the new version number.

**Errors:**

| Code | Condition |
|---|---|
| `AGENT_NOT_FOUND` | No agent with that ID |
| `FORBIDDEN` | Token lacks `agents:patch` or caller does not own the agent |

---

### `patch_session_annotations`

Merges key-value annotation pairs into a session's `annotations` field. Annotations are unrestricted user-defined string metadata — unlike labels they are not used for filtering, but they are readable by any agent or external system with `sessions:get`. This makes them a scoped, programmable state store: any application can write and read agent/session state without a custom database.

**RBAC required:** `sessions:patch`

**Backed by:** `PATCH /api/ambient/v1/sessions/{id}`

**Input schema:**

```json
{
  "type": "object",
  "required": ["session_id", "annotations"],
  "properties": {
    "session_id": {
      "type": "string",
      "description": "ID of the session to update."
    },
    "annotations": {
      "type": "object",
      "additionalProperties": { "type": "string" },
      "description": "Key-value annotation pairs to merge. Keys use reverse-DNS prefix convention (e.g. 'myapp.io/status'). Values are arbitrary strings up to 4096 bytes. Existing annotations not present in the patch are preserved. To delete an annotation, set its value to the empty string.",
      "example": {"myapp.io/status": "blocked", "myapp.io/blocker-id": "PROJ-1234"}
    }
  }
}
```

**Return value:** JSON-encoded updated `Session`.

**Behavior:**
- Reads existing `Session.annotations`
- Merges the provided annotations (provided keys overwrite existing values; empty-string values remove the key)
- Writes back via `PATCH /api/ambient/v1/sessions/{id}`

**Errors:**

| Code | Condition |
|---|---|
| `SESSION_NOT_FOUND` | No session with that ID |
| `ANNOTATION_VALUE_TOO_LARGE` | A value exceeds 4096 bytes |
| `FORBIDDEN` | Token lacks `sessions:patch` |

---

### `patch_agent_annotations`

Merges key-value annotation pairs into a ProjectAgent's `annotations` field. Agent annotations are persistent across sessions — they survive session termination and are visible to all future sessions for that agent. Use them to store durable agent state: last-known task, accumulated context index, external system IDs, etc.

**RBAC required:** `agents:patch`

**Backed by:** `PATCH /api/ambient/v1/projects/{project_id}/agents/{agent_id}`

**Input schema:**

```json
{
  "type": "object",
  "required": ["project_id", "agent_id", "annotations"],
  "properties": {
    "project_id": {
      "type": "string",
      "description": "Project ID the agent belongs to."
    },
    "agent_id": {
      "type": "string",
      "description": "Agent ID (UUID) or agent name."
    },
    "annotations": {
      "type": "object",
      "additionalProperties": { "type": "string" },
      "description": "Key-value annotation pairs to merge. Empty-string values remove the key.",
      "example": {"myapp.io/last-task": "PROJ-1234", "myapp.io/index-sha": "abc123"}
    }
  }
}
```

**Return value:** JSON-encoded updated `ProjectAgent`.

**Errors:**

| Code | Condition |
|---|---|
| `AGENT_NOT_FOUND` | No agent with that ID or name |
| `ANNOTATION_VALUE_TOO_LARGE` | A value exceeds 4096 bytes |
| `FORBIDDEN` | Token lacks `agents:patch` |

---

### `patch_project_annotations`

Merges key-value annotation pairs into a Project's `annotations` field. Project annotations are the widest-scope state store — visible to every agent and session in the project. Use them for project-level configuration, feature flags, shared context, and cross-agent coordination state that outlives any single session.

**RBAC required:** `projects:patch`

**Backed by:** `PATCH /api/ambient/v1/projects/{id}`

**Input schema:**

```json
{
  "type": "object",
  "required": ["project_id", "annotations"],
  "properties": {
    "project_id": {
      "type": "string",
      "description": "Project ID (UUID) or project name."
    },
    "annotations": {
      "type": "object",
      "additionalProperties": { "type": "string" },
      "description": "Key-value annotation pairs to merge. Empty-string values remove the key.",
      "example": {"myapp.io/feature-flags": "{\"dark-mode\":true}", "myapp.io/release": "v2.3.0"}
    }
  }
}
```

**Return value:** JSON-encoded updated `Project`.

**Errors:**

| Code | Condition |
|---|---|
| `PROJECT_NOT_FOUND` | No project with that ID or name |
| `ANNOTATION_VALUE_TOO_LARGE` | A value exceeds 4096 bytes |
| `FORBIDDEN` | Token lacks `projects:patch` |

---

### `list_projects`

Lists projects visible to the caller.

**RBAC required:** `projects:list`

**Backed by:** `GET /api/ambient/v1/projects`

**Input schema:**

```json
{
  "type": "object",
  "properties": {
    "page": {
      "type": "integer",
      "description": "Page number (1-indexed). Default: 1."
    },
    "size": {
      "type": "integer",
      "description": "Page size. Default: 20. Max: 100."
    }
  }
}
```

**Return value:** JSON-encoded `ProjectList`.

---

### `get_project`

Returns detail for a single project by ID or name.

**RBAC required:** `projects:get`

**Backed by:** `GET /api/ambient/v1/projects/{id}` (by ID), or `GET /api/ambient/v1/projects?search=name='{name}'` (by name)

**Input schema:**

```json
{
  "type": "object",
  "required": ["project_id"],
  "properties": {
    "project_id": {
      "type": "string",
      "description": "Project ID (UUID) or project name."
    }
  }
}
```

**Return value:** JSON-encoded `Project`.

**Errors:**

| Code | Condition |
|---|---|
| `PROJECT_NOT_FOUND` | No project matches the ID or name |
| `FORBIDDEN` | Token lacks `projects:get` |

---

## @mention Pattern

### Syntax

A mention is any token in message text matching the pattern `@{identifier}`, where `{identifier}` matches `[a-zA-Z0-9_-]+`.

Multiple mentions in a single message are supported. Each resolves independently and spawns a separate child session.

### Resolution Algorithm

Given `@{identifier}` in a `push_message` call:

1. If `{identifier}` matches UUID format (`[0-9a-f-]{36}`): call `GET /api/ambient/v1/agents/{identifier}`. If found, resolution succeeds.
2. Otherwise: call `GET /api/ambient/v1/agents?search=name='{identifier}'`. If exactly one result, resolution succeeds. If zero results, return `MENTION_NOT_RESOLVED`. If more than one result, return `AMBIGUOUS_AGENT_NAME`.

### Delegation Behavior

For each successfully resolved mention:

1. The mention token is stripped from the prompt text. Example: `@code-review please check this` becomes `please check this`.
2. `create_session` is called with:
   - `project_id` = same project as the calling session
   - `prompt` = mention-stripped text
   - `agent_id` = resolved agent ID
   - `parent_session_id` = calling session ID
3. The child session is started immediately (same behavior as `create_session`).
4. The `push_message` response includes the child session in `delegated_session`.

### Example

```
Calling session ID: sess-parent
Message text: "@code-review check the auth module for security issues"

Resolution:
  @code-review → GET /api/ambient/v1/agents?search=name='code-review'
               → agent ID: agent-abc123

Delegation:
  POST /api/ambient/v1/sessions
    { name: "check-auth-...", project_id: "demo-6640",
      prompt: "check the auth module for security issues",
      project_agent_id: "agent-abc123",
      parent_session_id: "sess-parent" }
  POST /api/ambient/v1/sessions/{new-id}/start

Response:
  {
    "message": { "id": "msg-xyz", "seq": 5, "event_type": "user", "payload": "@code-review check the auth module for security issues" },
    "delegated_session": { "id": "sess-child", "name": "check-auth-...", "phase": "Pending", ... }
  }
```

---

## HTTP Endpoint (ambient-api-server Integration)

The `ambient-api-server` exposes the MCP server's SSE transport at:

```
GET  /api/ambient/v1/mcp/sse
POST /api/ambient/v1/mcp/message
```

**Authentication:** `Authorization: Bearer {token}` header. Required on all requests. The token is forwarded to the MCP server process as `AMBIENT_TOKEN`, which it uses for all backend API calls during the session.

**Request flow:**

```
Browser / MCP client
    │  GET /api/ambient/v1/mcp/sse
    │  Authorization: Bearer {token}
    ▼
ambient-api-server
    │  spawns or connects to mcp-server process
    │  injects AMBIENT_TOKEN={token}
    ▼
mcp-server (SSE mode)
    │  MCP JSON-RPC over SSE
    ▼
ambient-api-server REST API
    │  Authorization: Bearer {token}  ← same token, forwarded
    ▼
platform resources
```

**Error codes:**

| HTTP Status | Condition |
|---|---|
| `401` | Missing or invalid bearer token |
| `403` | Token valid but lacks minimum required permissions |
| `503` | MCP server process could not be started |

---

## Sidecar Deployment

### Annotation

Sessions opt into the MCP sidecar by setting the annotation:

```
ambient-code.io/mcp-sidecar: "true"
```

This annotation is set on the Session resource at creation time. The operator reads it and injects the `ambient-mcp` container into the runner Job pod.

### Pod Layout

```
Job Pod (session-{id}-runner)
├── container: claude-code-runner
│     CLAUDE_CODE_MCP_CONFIG=/etc/mcp/config.json
│     reads config → connects to ambient-mcp via stdio
│
└── container: ambient-mcp
      image: localhost/vteam_ambient_mcp:latest
      MCP_TRANSPORT=stdio
      AMBIENT_API_URL=http://ambient-api-server.ambient-code.svc:8000
      AMBIENT_TOKEN={session bearer token from projected volume}
```

### MCP Config (injected by operator)

```json
{
  "mcpServers": {
    "ambient": {
      "command": "./ambient-mcp",
      "args": [],
      "env": {
        "MCP_TRANSPORT": "stdio",
        "AMBIENT_API_URL": "http://ambient-api-server.ambient-code.svc:8000",
        "AMBIENT_TOKEN": "${AMBIENT_TOKEN}"
      }
    }
  }
}
```

---

## CLI Commands

The `acpctl` CLI gains a new subcommand group for interacting with the MCP server in development and testing contexts.

### `acpctl mcp tools`

Lists all tools registered on the MCP server.

**Flags:** none

**Behavior:** Connects to the MCP server in stdio mode, sends `tools/list`, prints results, exits.

**Example:**

```
$ acpctl mcp tools
TOOL                      DESCRIPTION
list_sessions             List sessions with optional filters
get_session               Get full detail for a session by ID
create_session            Create and start a new agentic session
push_message              Send a user message to a running session
patch_session_labels      Merge labels into a session
watch_session_messages    Subscribe to a session's message stream
unwatch_session_messages  Cancel a message stream subscription
list_agents               List agents visible to the caller
get_agent                 Get agent detail by ID or name
create_agent              Create a new agent
update_agent              Update an agent's prompt (creates new version)
patch_session_annotations Merge annotations into a session (programmable state)
patch_agent_annotations   Merge annotations into an agent (durable state)
patch_project_annotations Merge annotations into a project (shared state)
list_projects             List projects visible to the caller
get_project               Get project detail by ID or name
```

**Exit codes:** `0` success, `1` connection failed, `2` auth error.

---

### `acpctl mcp call <tool> [flags]`

Calls a single MCP tool and prints the result as JSON.

**Flags:**

| Flag | Type | Description |
|---|---|---|
| `--input` | string | JSON-encoded tool input. Required. |
| `--url` | string | MCP server URL (SSE mode). If omitted, uses stdio mode against a locally started mcp-server binary. |

**Example:**

```
$ acpctl mcp call list_sessions --input '{"phase":"Running"}'
{
  "kind": "SessionList",
  "total": 2,
  "items": [...]
}

$ acpctl mcp call push_message --input '{"session_id":"abc123","text":"@code-review check auth.go"}'
{
  "message": { "seq": 7, "event_type": "user", ... },
  "delegated_session": { "id": "sess-child", "phase": "Pending", ... }
}
```

**Exit codes:** `0` success, `1` tool returned error, `2` auth error, `3` tool not found.

---

## Go SDK Example

Location: `components/ambient-sdk/go-sdk/examples/mcp/main.go`

This example demonstrates connecting to the MCP server via SSE and calling `list_sessions` and `push_message`. It is runnable against a live cluster.

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"

    mcp "github.com/mark3labs/mcp-go/client"
)

func main() {
    serverURL := os.Getenv("AMBIENT_MCP_URL") // e.g. http://localhost:8090
    token := os.Getenv("AMBIENT_TOKEN")

    client, err := mcp.NewSSEMCPClient(serverURL+"/sse",
        mcp.WithHeader("Authorization", "Bearer "+token),
    )
    if err != nil {
        log.Fatal(err)
    }
    ctx := context.Background()

    if err := client.Start(ctx); err != nil {
        log.Fatal(err)
    }

    _, err = client.Initialize(ctx, mcp.InitializeRequest{})
    if err != nil {
        log.Fatal(err)
    }

    result, err := client.CallTool(ctx, mcp.CallToolRequest{
        Params: mcp.CallToolParams{
            Name:      "list_sessions",
            Arguments: map[string]any{"phase": "Running"},
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    var out any
    _ = json.Unmarshal([]byte(result.Content[0].Text), &out)
    b, _ := json.MarshalIndent(out, "", "  ")
    fmt.Println(string(b))
}
```

---

## Error Catalog

| Code | HTTP equivalent | Description |
|---|---|---|
| `UNAUTHORIZED` | 401 | Token missing, invalid, or expired |
| `FORBIDDEN` | 403 | Token valid but lacks required RBAC permission |
| `SESSION_NOT_FOUND` | 404 | No session with the given ID |
| `SESSION_NOT_RUNNING` | 409 | Operation requires session in Running phase |
| `AGENT_NOT_FOUND` | 404 | No agent matches the given ID or name |
| `AMBIGUOUS_AGENT_NAME` | 409 | Name search matched more than one agent |
| `PROJECT_NOT_FOUND` | 404 | No project matches the given ID or name |
| `MENTION_NOT_RESOLVED` | 422 | `@mention` token could not be matched to any agent |
| `INVALID_REQUEST` | 400 | Missing required field or malformed input |
| `INVALID_LABEL_KEY` | 400 | Label key contains `=` or whitespace |
| `INVALID_LABEL_VALUE` | 400 | Label value is empty |
| `AGENT_NAME_CONFLICT` | 409 | Agent name already exists for this owner |
| `SUBSCRIPTION_NOT_FOUND` | 404 | No active subscription with the given ID |
| `TRANSPORT_NOT_SUPPORTED` | 400 | Operation requires SSE transport; caller is on stdio |
| `ANNOTATION_VALUE_TOO_LARGE` | 400 | Annotation value exceeds 4096 bytes |
| `INTERNAL` | 500 | Backend returned an unexpected error |

---

## Spec Completeness Checklist

Per `ambient-spec-development.md`, this spec is complete when:

- [ ] Tool input schemas defined — **done above**
- [ ] Tool return shapes defined — **done above**
- [ ] Error codes per tool — **done above**
- [ ] MCP protocol behavior (initialize, capabilities, transport) — **done above**
- [ ] `@mention` parsing rules and resolution algorithm — **done above**
- [ ] `watch_session_messages` notification shape — **done above**
- [ ] HTTP endpoint spec for ambient-api-server — **done above**
- [ ] Auth/RBAC per tool — **done above**
- [ ] CLI commands (`acpctl mcp tools`, `acpctl mcp call`) — **done above**
- [ ] Go SDK example — **done above** (stub; must be runnable against kind before implementation merge)
- [ ] Sidecar opt-in annotation specified — **done above**
- [ ] Operator changes to inject sidecar — **not in this spec** (requires separate operator spec update)
- [ ] `openapi.mcp.yaml` fragment — **not yet written** (required before implementation)
- [ ] Frontend session panel integration — **not in this spec** (requires frontend spec)
- [x] Annotation tools (`patch_session_annotations`, `patch_agent_annotations`, `patch_project_annotations`) — **done above**
- [x] Annotations-as-state-store design rationale — **done above** (per-tool descriptions)
