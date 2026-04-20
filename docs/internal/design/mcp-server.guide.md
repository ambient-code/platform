# MCP Server: Spec and Implementation Workflow

**Date:** 2026-03-22
**Status:** Living Document — updated continuously as the workflow is executed and improved
**Spec:** `mcp-server.spec.md` — tool definitions, input schemas, return shapes, error codes, transport, sidecar

---

## This Document Is Iterative

This document is updated as implementation runs. Each time the workflow is invoked, start from the top, follow the steps, and update this document with what was learned — what worked, what broke, what the step actually requires in practice.

> We start from the top each time. We update as we go. We run it until it Just Works™.

---

## Overview

The MCP server exposes the Ambient platform API as structured tools conforming to the [Model Context Protocol (MCP) 2024-11-05](https://spec.modelcontextprotocol.io/specification/2024-11-05/). It is the primary interaction surface for agents running inside the platform — every SEND, WAIT, and state read/write in an agent script is an MCP tool call.

Two deployment modes:

1. **Sidecar** — runs alongside Claude Code CLI in every runner Job pod. Claude Code connects via stdio. Auth token injected from the pod environment.
2. **Public endpoint** — exposed through `ambient-api-server` at `POST /api/ambient/v1/mcp`. Clients authenticate with the same bearer token as all other API calls.

The MCP server has **no direct Kubernetes access**. All operations proxy through `ambient-api-server`, inheriting the full RBAC model.

---

## The Pipeline

```
REST API (openapi.yaml)
  └─► MCP Tool Registry (server.go + tools/)
        ├─► Session tools    (tools/sessions.go)
        ├─► Agent tools      (tools/agents.go)
        ├─► Project tools    (tools/projects.go)
        └─► Annotation tools (tools/annotations.go)
              └─► Annotation State Protocol (agent-fleet-state-schema.md)
```

The MCP server depends on:
- A stable REST API — do not implement tools against unreleased endpoints
- The annotation schema defined in `docs/internal/proposals/agent-fleet-state-schema.md` — all `patch_*_annotations` tools must use the key conventions from that doc
- The sidecar annotation `ambient-code.io/mcp-sidecar: "true"` — required on Session for operator injection

---

## Component Location

```
components/ambient-mcp/
├── main.go              # Entrypoint; MCP_TRANSPORT env var selects stdio or SSE
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

**Image:** `localhost/vteam_ambient_mcp:latest`

**Naming rationale:** follows the `ambient-{role}` convention (`ambient-runner`, `ambient-cli`, `ambient-sdk`). Separate component with its own image — required because the operator injects it as a sidecar subprocess that Claude Code spawns via stdio. Cannot be embedded in `ambient-api-server`.

---

## Tool Surface

### Session Tools

| Tool | RBAC | Backed by | Description |
|---|---|---|---|
| `list_sessions` | `sessions:list` | `GET /sessions` | List sessions with optional phase/project filter |
| `get_session` | `sessions:get` | `GET /sessions/{id}` | Full session detail |
| `create_session` | `sessions:create` | `POST /sessions` + `/start` | Create and start a session; returns Pending |
| `push_message` | `sessions:patch` | `POST /sessions/{id}/messages` | Append user message; `@mention` spawns child session |
| `patch_session_labels` | `sessions:patch` | `PATCH /sessions/{id}` | Merge filterable label pairs |
| `patch_session_annotations` | `sessions:patch` | `PATCH /sessions/{id}` | Merge arbitrary state KV (scoped to session lifetime) |
| `watch_session_messages` | `sessions:get` | `GET /sessions/{id}/messages` SSE | Subscribe to message stream; pushes `notifications/progress` |
| `unwatch_session_messages` | — | internal | Cancel active subscription |

### Agent Tools

| Tool | RBAC | Backed by | Description |
|---|---|---|---|
| `list_agents` | `agents:list` | `GET /projects/{p}/agents` | List agents with search filter |
| `get_agent` | `agents:get` | `GET /projects/{p}/agents/{id}` | Agent detail by ID or name |
| `create_agent` | `agents:create` | `POST /projects/{p}/agents` | Create agent with name + prompt |
| `update_agent` | `agents:patch` | `PATCH /projects/{p}/agents/{id}` | Update prompt (creates new version) |
| `patch_agent_annotations` | `agents:patch` | `PATCH /projects/{p}/agents/{id}` | Merge durable state KV (persists across sessions) |

### Project Tools

| Tool | RBAC | Backed by | Description |
|---|---|---|---|
| `list_projects` | `projects:list` | `GET /projects` | List projects |
| `get_project` | `projects:get` | `GET /projects/{id}` | Project detail |
| `patch_project_annotations` | `projects:patch` | `PATCH /projects/{id}` | Merge fleet-wide shared state KV |

---

## Annotations as Programmable State

Annotations form a three-level scoped state store. All annotation tools follow the merge-not-replace semantics: existing keys not in the patch are preserved; empty-string values delete a key.

| Scope | Tool | Lifetime | Primary Use |
|---|---|---|---|
| Session | `patch_session_annotations` | Session lifetime | Retry count, current step, in-flight task status |
| Agent | `patch_agent_annotations` | Persistent | Last task, index SHA, external IDs, PR status |
| Project | `patch_project_annotations` | Project lifetime | Fleet protocol, contracts, agent roster, shared flags |

### Key Conventions (from `agent-fleet-state-schema.md`)

Annotation keys follow reverse-DNS prefix conventions. All agent self-reporting uses these namespaces:

| Namespace | Used for |
|---|---|
| `ambient.io/` | Platform coordination state (blocked, ready, blocker, roster, protocol, contracts, summary) |
| `work.ambient.io/` | Task tracking (epic, issue, current-task, next-tasks, completed-tasks) |
| `git.ambient.io/` | Git state (branch, worktree, pr-url, pr-status, last-commit-sha) |
| `myapp.io/` | User application state (any key; 4096 byte value limit) |

### Fleet Protocol Keys on Project

The project carries four top-level annotation keys that define the self-describing coordination layer:

- **`ambient.io/protocol`** — how agents communicate (check-in triggers, blocker escalation, handoff rules, roster entry field list)
- **`ambient.io/contracts`** — shared agreements (git conventions, API source of truth, SDK regeneration requirements, blocking thresholds)
- **`ambient.io/agent-roster`** — live fleet state array; each agent owns and writes only its own entry
- **`ambient.io/summary`** — human-readable current project state

Agents read these on every session start via `get_project`, reconcile their own state against the protocol and contracts, update their roster entry via `patch_agent_annotations` + `patch_project_annotations`, and then proceed with work.

---

## @mention Pattern

`push_message` supports `@{identifier}` syntax for agent-to-agent delegation.

**Resolution:** UUID → direct lookup. Name → search. Ambiguous name → `AMBIGUOUS_AGENT_NAME` error.

**Delegation:** each resolved mention strips the token from the prompt, calls `create_session` with the remaining text as `prompt` and `parent_session_id` set to the calling session. The child session is started immediately.

**Response shape:**
```json
{
  "message": { "seq": 5, "event_type": "user", "payload": "..." },
  "delegated_session": { "id": "...", "phase": "Pending" }
}
```

---

## Transport

| Mode | Transport | Binding |
|---|---|---|
| Sidecar (runner pod) | stdio | stdin/stdout of sidecar process |
| Public endpoint | SSE over HTTP | `MCP_BIND_ADDR` (proxied through `ambient-api-server`) |

### Sidecar Opt-in

Session must have annotation `ambient-code.io/mcp-sidecar: "true"` at creation time. Operator reads this and injects the `mcp-server` container into the runner Job pod.

**Pod layout:**
```
Job Pod (session-{id}-runner)
├── container: claude-code-runner
│     CLAUDE_CODE_MCP_CONFIG=/etc/mcp/config.json
│     connects to mcp-server via stdio
└── container: mcp-server
      MCP_TRANSPORT=stdio
      AMBIENT_API_URL=http://ambient-api-server.ambient-code.svc:8000
      AMBIENT_TOKEN={session bearer token from projected volume}
```

### Public Endpoint

The `ambient-api-server` exposes the MCP server's SSE transport at:

```
GET  /api/ambient/v1/mcp/sse
POST /api/ambient/v1/mcp/message
```

Auth: `Authorization: Bearer {token}` forwarded to MCP server as `AMBIENT_TOKEN`.

**Request flow:**

```
Client
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

---

## Error Codes

| Code | HTTP | Description |
|---|---|---|
| `UNAUTHORIZED` | 401 | Token missing, invalid, or expired |
| `FORBIDDEN` | 403 | Token valid but lacks required RBAC permission |
| `SESSION_NOT_FOUND` | 404 | No session with the given ID |
| `SESSION_NOT_RUNNING` | 409 | Operation requires session in Running phase |
| `AGENT_NOT_FOUND` | 404 | No agent matches ID or name |
| `AMBIGUOUS_AGENT_NAME` | 409 | Name search matched more than one agent |
| `PROJECT_NOT_FOUND` | 404 | No project matches ID or name |
| `MENTION_NOT_RESOLVED` | 422 | `@mention` token could not be matched to any agent |
| `INVALID_REQUEST` | 400 | Missing required field or malformed input |
| `INVALID_LABEL_KEY` | 400 | Label key contains `=` or whitespace |
| `ANNOTATION_VALUE_TOO_LARGE` | 400 | Annotation value exceeds 4096 bytes |
| `AGENT_NAME_CONFLICT` | 409 | Agent name already exists for this owner |
| `SUBSCRIPTION_NOT_FOUND` | 404 | No active subscription with the given ID |
| `TRANSPORT_NOT_SUPPORTED` | 400 | Streaming requires SSE transport; caller is on stdio |
| `INTERNAL` | 500 | Backend returned an unexpected error |

---

## Implementation Workflow

> **Each invocation: start from Step 1. Update this document before moving to the next step if anything is discovered.**

### Step 1 — Acknowledge Iteration

Before doing anything else, internalize that this run may not succeed. The workflow is the product. If a step fails, edit this document to capture the failure and what the step actually requires.

Checklist:
- [ ] Read this document top to bottom
- [ ] Note the last run's lessons (see [Run Log](#run-log) below)
- [ ] Confirm the REST API endpoints the tools depend on are present and stable
- [ ] Confirm `components/mcp-server/` directory exists or create the scaffold

---

### Step 2 — Read the Full Tool Spec

Read `docs/internal/design/mcp-server.spec.md` in full for the complete per-tool input schemas, return shapes, and error tables.

Read `docs/internal/proposals/agent-fleet-state-schema.md` for the annotation key conventions that all `patch_*_annotations` tools must honor.

Extract and hold in working memory:
- Every tool name, required inputs, optional inputs, return shape
- Every error code per tool
- The three annotation scopes and their lifetime semantics
- The `@mention` resolution algorithm
- The fleet protocol keys (`ambient.io/protocol`, `ambient.io/contracts`, `ambient.io/agent-roster`, `ambient.io/summary`)

---

### Step 3 — Assess What Has Been Implemented

For each tool, determine its current status:

| Tool | Status | Gap |
|---|---|---|
| `list_sessions` | ✅ implemented | — |
| `get_session` | ✅ implemented | — |
| `create_session` | ✅ implemented | — |
| `push_message` | ✅ implemented | — |
| `patch_session_labels` | ✅ implemented | — |
| `patch_session_annotations` | ✅ implemented | — |
| `watch_session_messages` | ✅ implemented | SSE transport guard in place; full streaming (Wave 5) pending |
| `unwatch_session_messages` | ✅ implemented | — |
| `list_agents` | 🔲 planned | — |
| `get_agent` | 🔲 planned | — |
| `create_agent` | ✅ implemented | — |
| `update_agent` | ✅ implemented | — |
| `patch_agent_annotations` | ✅ implemented | — |
| `list_projects` | ✅ implemented | — |
| `get_project` | ✅ implemented | — |
| `patch_project_annotations` | ✅ implemented | — |
| `@mention` resolution | ✅ implemented | — |
| stdio transport | ✅ implemented | — |
| SSE transport | ✅ implemented | — |
| sidecar injection (operator) | 🔲 planned | operator spec update required |

Update each row as implementation progresses. Mark ✅ when the tool has unit test coverage and the `acpctl mcp call` smoke test passes.

---

### Step 4 — Break Into Waves

**Wave 1 — Scaffold**

- Create `components/ambient-mcp/` with `go.mod`, `main.go`, `server.go`
- Wire `mark3labs/mcp-go` library
- Implement `MCP_TRANSPORT` env var dispatch (stdio vs SSE)
- Register all tools with real handlers — get `tools/list` to return all 16 tools
- **Acceptance:** `go build ./...` clean; `tools/list` via stdio shows complete tool list ✅ DONE

**Wave 2 — Read-only tools**

Implement tools that only `GET` from the REST API (no side effects):

- `list_sessions`, `get_session`
- `list_agents`, `get_agent`
- `list_projects`, `get_project`

No @mention. No SSE. No annotations. Get reads working first.

- **Acceptance:** `acpctl mcp call list_sessions --input '{}'` returns valid JSON; `get_session` returns 404 for unknown ID

**Wave 3 — Write tools (non-streaming)**

- `create_session` (POST + start)
- `push_message` (without @mention)
- `patch_session_labels`
- `patch_session_annotations`
- `patch_agent_annotations`
- `patch_project_annotations`
- `create_agent`, `update_agent`

Annotation merge semantics: read existing → merge patch → write back. Empty-string values delete the key.

- **Acceptance:** `acpctl mcp call push_message --input '{"session_id":"...","text":"hello"}'` returns message with seq; annotations round-trip correctly

**Wave 4 — @mention**

- Implement `mention/resolve.go`: UUID direct lookup, name search, ambiguity detection
- Wire into `push_message`: resolve mentions → spawn child sessions → return `delegated_session`
- **Acceptance:** `@agent-name` in message text spawns a child session with correct `parent_session_id`

**Wave 5 — Streaming**

- Implement `watch_session_messages`: open SSE to backend, forward as `notifications/progress`
- Implement `unwatch_session_messages`
- Phase polling loop (every 5s) for terminal notification
- Stdio guard: return `TRANSPORT_NOT_SUPPORTED` when called in stdio mode
- **Acceptance:** `watch_session_messages` delivers messages as they arrive; terminal notification fires on session completion

**Wave 6 — Sidecar**

- Update operator to read `ambient-code.io/mcp-sidecar: "true"` annotation
- Inject `mcp-server` container into Job pod spec
- Generate and mount `CLAUDE_CODE_MCP_CONFIG` volume
- **Acceptance:** session with `mcp-sidecar: true` annotation launches pod with two containers; Claude Code connects via stdio and `list_sessions` returns data

**Wave 7 — Integration**

- End-to-end smoke: ignite agent → agent calls `push_message` with @mention → child session starts → parent calls `watch_session_messages` → child completes → terminal notification received
- Annotation state round-trip: agent writes `patch_agent_annotations` → external call reads back via `get_agent`
- `make test` and `make lint` in `components/ambient-mcp/`

---

### Step 5 — Send Messages to the MCP Agent

```sh
acpctl send mcp --body "Wave 1: Scaffold components/mcp-server/. Wire mark3labs/mcp-go. Implement MCP_TRANSPORT dispatch. Register all 16 tools as stubs. Done = acpctl mcp tools lists all tools."
acpctl start mcp

acpctl send mcp --body "Wave 2: Implement read-only tools (list_sessions, get_session, list_agents, get_agent, list_projects, get_project). All tools proxy to REST API. Done = acpctl mcp call get_session returns correct data."
acpctl start mcp
```

Do not ignite Wave 3+ until Wave 2 is ✅. Do not ignite Wave 5 (streaming) until Wave 3 write tools are ✅.

Monitor via `acpctl get sessions -w` and the board at `http://localhost:8899`.

---

### Step 6 — Ascertain Completion

For each wave, the MCP agent reports done when:

1. All tools in the wave have passing unit tests
2. `go build ./...` and `go vet ./...` are clean
3. `golangci-lint run` is clean
4. The `acpctl mcp tools` output matches the full tool list above
5. `acpctl mcp call {tool}` smoke passes for each implemented tool

The workflow is complete when:
- All 16 tools are ✅ in the gap table (Step 3)
- Wave 6 (sidecar) is ✅
- An agent session successfully uses MCP tools to write and read its own annotations
- The fleet protocol keys (`ambient.io/protocol`, `ambient.io/agent-roster`) round-trip through `patch_project_annotations` + `get_project`

---

## Build and Test Commands

```sh
# Build binary
cd components/ambient-mcp && go build ./...

# Vet + lint
cd components/ambient-mcp && go vet ./... && golangci-lint run

# Unit tests
cd components/ambient-mcp && go test ./...

# Build image
podman build --platform linux/amd64 -t vteam_ambient_mcp:latest components/ambient-mcp/

# Load into kind cluster
podman save localhost/vteam_ambient_mcp:latest | \
  podman exec -i ambient-main-control-plane \
  ctr --namespace=k8s.io images import -

# Verify image in cluster
podman exec ambient-main-control-plane \
  ctr --namespace=k8s.io images ls | grep ambient_mcp

# Smoke test via stdio (no cluster needed)
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | \
  AMBIENT_TOKEN=your-token go run ./components/ambient-mcp/
```

---

## Run Log

Update this section after each implementation run.

### Run 1 — 2026-03-22

**Outcome:** Waves 1–4 complete. Image built and loaded into kind cluster `ambient-main`.

Gap table state at end of Run 1:
- All 16 tools: ✅ implemented
- stdio transport: ✅
- SSE transport: ✅ (server starts; full streaming/progress notifications pending Wave 5)
- @mention resolution: ✅
- Sidecar injection (operator): 🔲 planned

**Component renamed:** `components/mcp-server/` → `components/ambient-mcp/` (follows `ambient-{role}` naming convention).

**Image:** `localhost/vteam_ambient_mcp:latest` — built with `podman build`, loaded into `ambient-main-control-plane` via `ctr import`.

Lessons learned:
- `mark3labs/mcp-go v0.45.0` — `Required()` is a `PropertyOption` (not `WithRequired`); tool registration is `s.AddTool(mcp.NewTool(...), handler)`
- Annotation merge semantics must be implemented in tool layer: GET existing → unmarshal JSON string → merge map → marshal → PATCH back
- `watch_session_messages` must guard against stdio transport (`TRANSPORT_NOT_SUPPORTED`) before attempting SSE
- The binary is `./ambient-mcp` in the container (not `/usr/local/bin/mcp-server`); MCP config command must match

---

## References

- Full per-tool schemas, return shapes, and error tables: `docs/internal/design/mcp-server.spec.md`
- Annotation key conventions and fleet state protocol: `docs/internal/proposals/agent-fleet-state-schema.md`
- Agent visual language (how purple SEND/WAIT blocks map to MCP tools): `docs/internal/proposals/agent-script-visual-language.md`
- Platform data model: `docs/internal/design/ambient-data-model.md`
- Component pipeline and wave pattern: `docs/internal/design/ambient-model.guide.md`
