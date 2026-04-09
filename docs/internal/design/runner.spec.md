# Ambient Runner Spec

**Date:** 2026-04-05
**Status:** Living Document — current state documented
**Related:** `control-plane.spec.md` — CP provisioning, token endpoint, start context assembly

---

## Overview

The Ambient Runner is a Python FastAPI application that runs inside each session pod. It is the execution engine for one session: it owns the Claude Code subprocess lifecycle, bridges between the AG-UI HTTP protocol and the gRPC message store, streams results in real time, and exposes a local SSE tap for live event observation.

One runner pod runs per session. The pod is ephemeral — created by the CP when a session starts, deleted when the session ends.

```
CP creates runner pod
    │  env vars (SESSION_ID, INITIAL_PROMPT, AMBIENT_GRPC_URL, ...)
    ▼
Runner Pod (FastAPI + uvicorn)
    │
    ├── gRPC listener ←── WatchSessionMessages (api-server)
    │        │
    │        └──► bridge.run() ──► Claude Code subprocess
    │                    │
    │                    ├──► PushSessionMessage (api-server)       ← durable record
    │                    └──► _active_streams[thread_id] queue      ← SSE tap
    │
    └── HTTP endpoints
          ├── GET /events/{thread_id}      ← live SSE tap (drained by backend proxy)
          ├── POST /                       ← AG-UI run (HTTP path, backup)
          ├── POST /interrupt
          └── GET /health
```

---

## What the Runner Is

The runner is a **bridge**. It translates between three different message-passing systems:

| System | Protocol | Direction | Purpose |
|--------|----------|-----------|---------|
| api-server gRPC | `WatchSessionMessages` | inbound | User messages that trigger Claude turns |
| Claude Agent SDK | subprocess stdin/stdout | bidirectional | Drives Claude Code execution |
| api-server gRPC | `PushSessionMessage` | outbound | Durable conversation record (assistant turns) |
| SSE tap | `GET /events/{thread_id}` | outbound | Live event stream for the frontend and CLI |

The runner has no database. All persistent state (session messages, session phase) lives in the api-server.

---

## Source Layout

```
ambient_runner/
  app.py                          ← FastAPI application factory + lifespan
  bridge.py                       ← PlatformBridge ABC (integration contract)
  _grpc_client.py                 ← AmbientGRPCClient (RSA-OAEP auth, channel build)
  _session_messages_api.py        ← SessionMessagesAPI (hand-rolled proto codec)
  _inbox_messages_api.py          ← InboxMessagesAPI
  observability.py                ← ObservabilityManager (Langfuse)
  observability_models.py         ← Langfuse event model types

  platform/
    context.py                    ← RunnerContext dataclass (shared runtime state)
    config.py                     ← Config loaders (.ambient/ambient.json, .mcp.json, REPOS_JSON)
    auth.py                       ← Credential fetching + git identity + env population
    workspace.py                  ← Working directory resolution (workflow / multi-repo / default)
    prompts.py                    ← System prompt constants + workspace context builder
    utils.py                      ← Pure helpers (redact_secrets, get_bot_token, url_with_token)
    security_utils.py             ← Input validation helpers
    feedback.py                   ← User feedback storage
    workspace.py                  ← Workspace setup and validation

  bridges/claude/
    bridge.py                     ← ClaudeBridge (PlatformBridge impl)
    session.py                    ← SessionManager + SessionWorker (Claude subprocess isolation)
    grpc_transport.py             ← GRPCSessionListener + GRPCMessageWriter
    auth.py                       ← Vertex AI setup + model resolution
    mcp.py                        ← MCP server assembly
    tools.py                      ← In-process MCP tools (refresh_credentials, evaluate_rubric)
    backend_tools.py              ← acp_* MCP tools (backend API access for Claude)
    prompts.py                    ← SDK system prompt builder
    corrections.py                ← Correction detection and logging
    mock_client.py                ← Local dev mock (no Claude subprocess)

  bridges/gemini_cli/             ← Gemini CLI bridge (separate impl, same ABC)
  bridges/langgraph/              ← LangGraph bridge (stub)

  endpoints/
    run.py                        ← POST / (AG-UI run endpoint)
    events.py                     ← GET /events/{thread_id} (SSE tap)
    interrupt.py                  ← POST /interrupt
    health.py                     ← GET /health
    capabilities.py               ← GET /capabilities
    repos.py                      ← GET /repos
    workflow.py                   ← GET /workflow
    mcp_status.py                 ← GET /mcp-status
    content.py                    ← GET /content
    tasks.py                      ← GET /tasks
    feedback.py                   ← POST /feedback

  middleware/
    grpc_push.py                  ← grpc_push_middleware (HTTP-path event fan-out)
    developer_events.py           ← Dev-mode event logging
    secret_redaction.py           ← Token scrubbing from event payloads
    tracing.py                    ← Langfuse span injection

  tools/
    backend_api.py                ← BackendAPIClient (sync HTTP client for api-server REST)
```

---

## Startup Sequence

```
1. main.py calls run_ambient_app(bridge)
2. uvicorn starts; FastAPI lifespan() runs:

3. RunnerContext created from env vars:
     SESSION_ID, WORKSPACE_PATH, BACKEND_API_URL, ...

4. bridge.set_context(context)

5. If AMBIENT_GRPC_ENABLED=true:
     a. AmbientGRPCClient.from_env() called:
          - AMBIENT_CP_TOKEN_URL set → fetch token from CP /token
            (RSA-OAEP: encrypt SESSION_ID with public key, send as Bearer)
          - set_bot_token(token) — wires into get_bot_token() for all HTTP calls
          - Build gRPC channel with token
     b. GRPCSessionListener.start() → WatchSessionMessages RPC opens
     c. await listener.ready.wait()  ← blocks until stream confirmed open
     d. Pre-register SSE queue for SESSION_ID (prevents race with backend)

6. If INITIAL_PROMPT set and not IS_RESUME:
     _auto_execute_initial_prompt(prompt, session_id, grpc_url)
       In gRPC mode: push via PushSessionMessage("user", prompt)
         → listener receives its own push → triggers bridge.run()
       In HTTP mode: POST to backend /agui/run with exponential backoff

7. yield (app ready, uvicorn serving on AGUI_HOST:AGUI_PORT)

8. On shutdown: bridge.shutdown() → GRPCSessionListener.stop()
```

### First-Run Platform Setup (deferred, on first `bridge.run()` call)

```
bridge._setup_platform():
  1. validate_prerequisites(context)         ← phase-based slash command gating
  2. setup_sdk_authentication(context)       ← Vertex AI or Anthropic API key
  3. populate_runtime_credentials(context)   ← GitHub, GitLab, Google, Jira from backend
  4. resolve_workspace_paths(context)        ← CWD: workflow / multi-repo / artifacts
  5. setup_workspace(context)                ← log workspace state
  6. ObservabilityManager init               ← Langfuse (best-effort, no-op on failure)
  7. build_mcp_servers(context, cwd_path)    ← external + platform MCP servers
  8. build_sdk_system_prompt(...)            ← preset + workspace context string
```

---

## Token Authentication

The runner has two token identities:

| Token | Source | Used for |
|-------|--------|----------|
| **CP OIDC token** | `GET AMBIENT_CP_TOKEN_URL/token` (RSA-OAEP auth) | gRPC channel to api-server; all `PushSessionMessage` calls |
| **Caller token** | `x-caller-token` header on each run request | Backend HTTP credential fetches (`GET /credentials/{id}/token`) — scoped to the requesting user |

### CP Token Flow

```python
# _grpc_client.py
bearer = _encrypt_session_id(public_key_pem, session_id)   # RSA-OAEP
token  = _fetch_token_from_cp(cp_token_url, bearer)         # HTTP GET
set_bot_token(token)                                         # cache in utils.py
```

`get_bot_token()` priority (platform/utils.py):
1. CP-fetched token cache (`_cp_fetched_token`)
2. File mount `/var/run/secrets/ambient/bot-token` (kubelet-rotated)
3. `BOT_TOKEN` env var (local dev fallback)

On gRPC `UNAUTHENTICATED`, the listener calls `grpc_client.reconnect()` which re-fetches from the CP endpoint and rebuilds the channel.

---

## Bridge Layer

`PlatformBridge` (bridge.py) defines the integration contract:

| Method | Required | Purpose |
|--------|----------|---------|
| `capabilities()` | yes | Declare feature support to `/capabilities` endpoint |
| `run(input_data)` | yes | Async generator — execute one turn, yield AG-UI events |
| `interrupt(thread_id)` | yes | Halt the active run for a thread |
| `set_context(ctx)` | no | Receive `RunnerContext` before first run |
| `_setup_platform()` | no | Deferred first-run initialization |
| `shutdown()` | no | Graceful teardown |
| `mark_dirty()` | no | Force full re-setup on next run |
| `inject_message(msg)` | no | gRPC path — listener injects parsed `RunnerInput` |

`ClaudeBridge` is the production implementation. `GeminiCLIBridge` and `LangGraphBridge` exist as alternate bridge implementations using the same ABC.

---

## Claude Bridge Internals

### Session Isolation

Each `thread_id` (= session ID) gets one `SessionWorker`. The worker owns a single `ClaudeSDKClient` in a background `asyncio.Task` with a long-running stdin/stdout connection to the Claude Code subprocess.

```
SessionManager
  └── SessionWorker(thread_id)
        ├── _client: ClaudeSDKClient  ← Claude subprocess connection
        ├── _active_output_queue      ← yields events during a turn
        └── _between_run_queue        ← background messages between turns
```

`SessionWorker.query(prompt, session_id)` enqueues the request and yields SDK messages until the `None` sentinel. Worker death is detected on the next `query()` call — dead workers are replaced automatically.

`SessionManager` persists `thread_id → sdk_session_id` to `{state_dir}/claude_session_ids.json` on every new session. This enables `--resume` on pod restart.

### Per-Turn Lifecycle

```
bridge.run(input_data):
  1. _initialize_run(): set user context, refresh credentials if stale
  2. session_manager.get_or_create_worker(thread_id)
  3. worker.acquire_lock()                            ← prevent concurrent turns
  4. worker.query(prompt, session_id)
  5. wrap stream: tracing_middleware → secret_redaction_middleware
  6. yield events
  7. Detect HITL halt: _halted_by_thread[thread_id] = True → interrupt worker
  finally: clear_runtime_credentials(context)
```

Credentials are populated before step 1 and cleared in the `finally` block. This is intentional: each turn runs with a fresh credential set, and credentials are never retained between turns.

### Adapter Rebuild (`mark_dirty()`)

`mark_dirty()` is called when the MCP configuration changes (e.g. different user context). It:
1. Snapshots all `thread_id → sdk_session_id` mappings
2. Tears down the existing `SessionManager` (async, non-blocking)
3. Clears `_adapter` and `_ready` → next `run()` triggers full `_setup_platform()`
4. Restores saved session IDs after rebuild so `--resume` still works

---

## gRPC Transport Layer

### `GRPCSessionListener` (pod-lifetime)

```
WatchSessionMessages(session_id, last_seq)
    │
    │  [thread pool — blocking gRPC iterator]
    │
    ▼
  asyncio bridge (run_coroutine_threadsafe)
    │
    │  event_type == "user"
    ├──► parse RunnerInput → bridge.run()
    │         │
    │         ├──► _active_streams[thread_id].put_nowait(event)   ← SSE tap
    │         └──► GRPCMessageWriter.consume(event)               ← durable record
    │
    │  other event_type
    └──► log and skip
```

- Sets `self.ready` asyncio.Event once the stream is confirmed open
- Reconnects with exponential backoff (1s → 30s) on stream failure
- On `UNAUTHENTICATED`: calls `grpc_client.reconnect()` before retry
- Tracks `last_seq` to resume without replay

### `GRPCMessageWriter` (per-turn)

Accumulates `MESSAGES_SNAPSHOT` events (keeping only the latest — each snapshot is a full replacement). On `RUN_FINISHED` or `RUN_ERROR`, calls:

```python
PushSessionMessage(
    session_id=session_id,
    event_type="assistant",
    payload=assistant_text,   # extracted from last MESSAGES_SNAPSHOT
)
```

Push is synchronous gRPC; runs in a `ThreadPoolExecutor` to avoid blocking the event loop.

**Payload contract:**
- `event_type=user`: plain string (the user's message text)
- `event_type=assistant`: plain string (Claude's reply text only — no reasoning, no user echo)

---

## SSE Tap: `GET /events/{thread_id}`

The SSE tap endpoint in `endpoints/events.py` is a pure observer. It never calls `bridge.run()`.

```
Sequence:
  1. Backend registers GET /events/{thread_id} (before POST /sessions/{id}/messages)
  2. endpoints/events.py registers asyncio.Queue in bridge._active_streams[thread_id]
  3. User POST /sessions/{id}/messages → PushSessionMessage("user", text)
  4. GRPCSessionListener receives its own push → bridge.run()
  5. bridge.run() yields events → put_nowait into _active_streams[thread_id]
  6. GET /events stream reads from queue → SSE to client
  7. On RUN_FINISHED or RUN_ERROR: close stream
```

- Queue size: 100 (events dropped silently if consumer is slow)
- Heartbeat: `: keepalive` comment every 30s
- `MESSAGES_SNAPSHOT` events are filtered out (internal accumulator state, not for clients)
- Queue is removed from `_active_streams` on client disconnect or run end

---

## Credential Management

Credentials are **ephemeral per-turn**. They are populated before each Claude turn and cleared after.

```
populate_runtime_credentials(context):
  concurrent asyncio.gather:
    _fetch_credential("github") → GITHUB_TOKEN, /tmp/.ambient_github_token
    _fetch_credential("gitlab") → GITLAB_TOKEN, /tmp/.ambient_gitlab_token
    _fetch_credential("google") → GOOGLE_APPLICATION_CREDENTIALS, credentials.json
    _fetch_credential("jira")   → JIRA_URL, JIRA_API_TOKEN, JIRA_EMAIL

clear_runtime_credentials():
  unset all env vars + delete all temp files
```

The credential fetch uses `context.caller_token` (the user's bearer from `x-caller-token` header) so each user can only access their own credentials. The `BACKEND_API_URL` is validated to be a cluster-local hostname before any request is made (prevents token exfiltration to external hosts).

The `refresh_credentials` MCP tool (registered under the `session` MCP server) lets Claude proactively refresh credentials mid-turn. Rate-limited to once per 30 seconds.

---

## MCP Servers

The runner assembles the full MCP server configuration at setup time. Claude sees these servers as tools:

| Server | Transport | Tools | Source |
|--------|-----------|-------|--------|
| External (`.mcp.json`) | stdio / SSE | whatever the server exposes | user config |
| `ambient-mcp` | SSE (`AMBIENT_MCP_URL`) | platform-provided tools | operator-injected |
| `session` | in-process | `refresh_credentials` | always registered |
| `rubric` | in-process | `evaluate_rubric` | registered if `.ambient/rubric.md` found |
| `corrections` | in-process | `log_correction` | always registered |
| `acp` | in-process | `acp_*` (9 tools) | always registered |

### `acp` MCP Server Tools

Claude can call these tools to interact with the Ambient platform:

| Tool | Description |
|------|-------------|
| `acp_list_sessions` | List sessions with phase/search/pagination filtering |
| `acp_get_session` | Read full session object |
| `acp_create_session` | Create a child session (inherits parent credentials via `parentSessionId`) |
| `acp_stop_session` | Stop a running session |
| `acp_send_message` | Send a message to a session's AG-UI run endpoint |
| `acp_get_session_status` | Session details + recent text messages |
| `acp_restart_session` | Stop then start |
| `acp_list_workflows` | List OOTB workflows |
| `acp_get_api_reference` | Full Ambient REST API docs with current context values |

---

## System Prompt Construction

The system prompt is assembled once during `_setup_platform()` and passed to the Claude SDK:

```python
{
  "type": "preset",
  "preset": "claude_code",
  "append": f"{DEFAULT_AGENT_PREAMBLE}\n\n{workspace_context}"
}
```

`DEFAULT_AGENT_PREAMBLE` establishes Ambient platform identity and behavioral guidelines.

`workspace_context` is built by `build_workspace_context_prompt()` and includes:
- Fixed workspace paths (`/workspace/artifacts`, `/workspace/file-uploads`)
- Active workflow CWD and name
- List of uploaded files
- Repository list with URLs and branches
- Git push instructions (for auto-push repos)
- HITL interrupt instructions
- MCP integration-specific instructions (Google, Jira, GitLab, GitHub)
- Token presence hints
- Workflow-specific system prompt (from `ambient.json` `systemPrompt` field)
- Rubric evaluation section (if `rubric.md` found)
- Corrections feedback instructions

---

## Environment Variables

All env vars are injected by the CP at pod creation time.

| Var | Purpose |
|-----|---------|
| `SESSION_ID` | Primary session identifier; also the `thread_id` for AG-UI |
| `PROJECT_NAME` | Project context |
| `WORKSPACE_PATH` | Claude Code working directory root (`/workspace`) |
| `AGUI_HOST` / `AGUI_PORT` | Runner HTTP listener (default `0.0.0.0:8001`) |
| `BACKEND_API_URL` | api-server base URL (cluster-local) |
| `AMBIENT_GRPC_URL` | api-server gRPC address |
| `AMBIENT_GRPC_USE_TLS` | TLS flag for gRPC channel |
| `AMBIENT_CP_TOKEN_URL` | CP token endpoint (e.g. `http://ambient-control-plane.{ns}.svc:8080/token`) |
| `AMBIENT_CP_TOKEN_PUBLIC_KEY` | RSA public key PEM for CP token auth |
| `AMBIENT_GRPC_ENABLED` | Enables gRPC listener path (default: `true` when `AMBIENT_GRPC_URL` set) |
| `INITIAL_PROMPT` | Auto-execute prompt on startup |
| `IS_RESUME` | Skip `INITIAL_PROMPT` auto-execute on pod restart |
| `USE_VERTEX` | Enable Vertex AI (vs Anthropic API) |
| `ANTHROPIC_VERTEX_PROJECT_ID` / `CLOUD_ML_REGION` | Vertex AI config |
| `GOOGLE_APPLICATION_CREDENTIALS` | Vertex service account path |
| `LLM_MODEL` / `LLM_TEMPERATURE` / `LLM_MAX_TOKENS` | Per-session model config |
| `LLM_MODEL_VERTEX_ID` | Explicit Vertex model ID (overrides static map) |
| `CREDENTIAL_IDS` | JSON map `{provider: id}` — resolved credential IDs for this session |
| `AMBIENT_MCP_URL` | Ambient MCP sidecar URL (SSE transport) |
| `REPOS_JSON` | JSON array of `{url, branch, autoPush}` repo configs |
| `ACTIVE_WORKFLOW_GIT_URL` | Active workflow repo URL (overrides REPOS_JSON workspace setup) |

---

## Two Message Paths

| Path | Trigger | Fan-out | Persistence |
|------|---------|---------|-------------|
| **gRPC listener** | `WatchSessionMessages` stream receives `event_type=user` | SSE tap queue + `GRPCMessageWriter` | Assistant turn pushed to api-server DB |
| **HTTP POST `/`** | Direct HTTP AG-UI run request | `grpc_push_middleware` fire-and-forget | Each event pushed individually |

The gRPC listener path is the primary path in standard deployment. The HTTP POST path is the backup path and is used in local dev environments without a CP.

---

## Workspace Resolution

`resolve_workspace_paths(context)` determines the Claude working directory:

```
Priority order:
1. ACTIVE_WORKFLOW_GIT_URL set  →  /workspace/workflows/<name>
                                    add_dirs: all repos, artifacts, file-uploads
2. REPOS_JSON set               →  /workspace/<primary_repo>
                                    add_dirs: remaining repos
3. Default                      →  /workspace/artifacts
```

The resolved `(cwd_path, add_dirs)` tuple is passed to the Claude SDK via `ClaudeAgentAdapter`. Claude Code sees `cwd_path` as its working directory and `add_dirs` as additional indexed directories.

---

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| Bridge ABC over direct Claude dependency | Enables Gemini CLI, LangGraph, and future bridges without changing app or platform layer |
| `SessionWorker` isolates Claude subprocess | Claude SDK uses anyio internally — running it in a background asyncio.Task with queue-based API prevents anyio/asyncio event loop conflicts |
| `_setup_platform()` deferred to first run | App startup must be fast; credential fetching, MCP server loading, and system prompt construction are I/O-heavy and done once per pod lifetime |
| Credentials cleared after every turn | Enforces per-user isolation; prevents a second user's run from inheriting credentials from the first user's turn |
| RSA-OAEP for CP token auth | CP SA cannot create `tokenreviews` at cluster scope (tenant RBAC restriction); asymmetric encryption with a self-generated keypair (persisted in S0 Secret) requires no cluster-scoped permissions |
| `set_bot_token()` module-level cache | CP-fetched OIDC token must be available to `get_bot_token()` for all HTTP API calls (credential fetches, backend tools); gRPC token and HTTP token are the same identity |
| `GRPCMessageWriter` stores only last `MESSAGES_SNAPSHOT` | Each snapshot is a complete replacement; accumulating all would waste memory for long turns |
| Assistant payload = plain string | Symmetric with user payload; reasoning content is observability data not durable conversation record; payload size reduction is dramatic (reasoning can be 10x longer than reply) |
| SSE queue pre-registered before `INITIAL_PROMPT` push | Backend opens `GET /events/{thread_id}` before `PushSessionMessage`; pre-registration in lifespan eliminates the race |
| `--resume` via persisted session IDs | Claude Code saves state to `.claude/` on graceful subprocess shutdown; session IDs survive `mark_dirty()` rebuilds via JSON file and `_saved_session_ids` snapshot |
| Credential URL validated to cluster-local hostname | Prevents exfiltration of user tokens to external hosts if `BACKEND_API_URL` is tampered with |
