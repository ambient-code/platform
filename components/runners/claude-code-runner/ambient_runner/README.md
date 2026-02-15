# Ambient Runner SDK

Reusable platform package for building AG-UI agent runners. Provides the FastAPI server, endpoint routing, lifespan management, and platform services — you bring the AI framework.

## Quick Start

### 3-Line Runner

```python
# main.py
from ambient_runner import create_ambient_app, run_ambient_app
from ambient_runner.bridges.claude import ClaudeBridge

app = create_ambient_app(ClaudeBridge(), title="My Runner")

if __name__ == "__main__":
    run_ambient_app(app)
```

Run it:

```bash
ANTHROPIC_API_KEY=sk-ant-... python main.py
# or
ANTHROPIC_API_KEY=sk-ant-... uvicorn main:app
```

### What You Get

That single `create_ambient_app()` call gives you:

- `POST /` — AG-UI run endpoint (SSE event stream)
- `POST /interrupt` — interrupt a running agent
- `GET /health` — liveness check
- `GET /capabilities` — framework + platform feature manifest
- `POST /feedback` — Langfuse thumbs-up/down scoring
- `POST /repos/add`, `POST /repos/remove`, `GET /repos/status` — repository management
- `POST /workflow` — runtime workflow switching
- `GET /mcp/status` — MCP server diagnostics

Plus: automatic lifespan management, context creation from environment variables, non-interactive session auto-prompting, and graceful shutdown.

---

## Architecture

```
ambient_runner/
├── __init__.py              # Public API: create_ambient_app, run_ambient_app, PlatformBridge
├── app.py                   # App factory, lifespan, auto-prompt
├── bridge.py                # PlatformBridge ABC + FrameworkCapabilities
├── observability.py         # Langfuse integration (optional)
│
├── bridges/                 # One subpackage per framework
│   ├── claude/              #   Claude Agent SDK (full reference)
│   │   ├── bridge.py        #     ClaudeBridge — full lifecycle
│   │   ├── auth.py          #     API key + Vertex AI setup
│   │   ├── mcp.py           #     MCP server building
│   │   ├── prompts.py       #     System prompt construction
│   │   ├── session.py       #     SessionManager / SessionWorker
│   │   └── tools.py         #     MCP tool definitions
│   └── langgraph/           #   LangGraph (minimal reference)
│       └── bridge.py        #     LangGraphBridge
│
├── endpoints/               # FastAPI routers (all use bridge pattern)
│   ├── run.py               #   POST /
│   ├── interrupt.py         #   POST /interrupt
│   ├── health.py            #   GET /health
│   ├── capabilities.py      #   GET /capabilities
│   ├── feedback.py          #   POST /feedback
│   ├── repos.py             #   /repos/*
│   ├── workflow.py          #   POST /workflow
│   └── mcp_status.py        #   GET /mcp/status
│
├── middleware/               # Event stream wrappers
│   ├── tracing.py           #   Langfuse tracing
│   └── developer_events.py  #   Developer-role AG-UI messages
│
└── platform/                # Framework-agnostic services
    ├── context.py           #   RunnerContext dataclass
    ├── config.py            #   ambient.json, MCP config, repos config
    ├── auth.py              #   Credential fetching (GitHub, Google, Jira, GitLab)
    ├── workspace.py         #   Path resolution, multi-repo setup
    ├── prompts.py           #   Workspace context prompt builder
    ├── security_utils.py    #   Sanitization, timeout utilities
    └── utils.py             #   Shared helpers
```

### Layer Rules

| Layer | Depends On | Never Depends On |
|-------|-----------|-----------------|
| `endpoints/` | `bridge` (via `request.app.state.bridge`) | framework bridges, platform internals |
| `middleware/` | `ag_ui.core` types only | bridges, endpoints, platform |
| `bridges/claude/` | `platform/`, `ag_ui_claude_sdk`, `middleware/` | endpoints |
| `bridges/langgraph/` | `platform/`, `ag_ui_langgraph` | endpoints, claude bridge |
| `platform/` | stdlib, `RunnerContext` | bridges, endpoints, middleware |

---

## Building Your Own Bridge

Subclass `PlatformBridge` and implement three methods:

```python
from typing import AsyncIterator, Optional
from ag_ui.core import (
    BaseEvent, EventType, RunAgentInput,
    RunStartedEvent, RunFinishedEvent,
    TextMessageStartEvent, TextMessageContentEvent, TextMessageEndEvent,
    MessagesSnapshotEvent, Message,
)
from ambient_runner import PlatformBridge, FrameworkCapabilities
from ambient_runner.platform.context import RunnerContext


class MyBridge(PlatformBridge):
    """Minimal bridge that echoes user messages."""

    def __init__(self):
        self._context: RunnerContext | None = None

    # --- Required ---

    def capabilities(self) -> FrameworkCapabilities:
        return FrameworkCapabilities(
            framework="echo",
            agent_features=["agentic_chat"],
        )

    async def run(self, input_data: RunAgentInput) -> AsyncIterator[BaseEvent]:
        thread_id = input_data.thread_id or "default"
        run_id = input_data.run_id or "unknown"

        # 1. RUN_STARTED (always first)
        yield RunStartedEvent(
            type=EventType.RUN_STARTED,
            thread_id=thread_id,
            run_id=run_id,
        )

        # 2. Text message
        user_text = ""
        for msg in input_data.messages:
            if msg.get("role") == "user":
                user_text = msg.get("content", "")

        msg_id = "echo-1"
        yield TextMessageStartEvent(
            type=EventType.TEXT_MESSAGE_START,
            message_id=msg_id,
            role="assistant",
        )
        yield TextMessageContentEvent(
            type=EventType.TEXT_MESSAGE_CONTENT,
            message_id=msg_id,
            delta=f"You said: {user_text}",
        )
        yield TextMessageEndEvent(
            type=EventType.TEXT_MESSAGE_END,
            message_id=msg_id,
        )

        # 3. RUN_FINISHED (always last)
        yield RunFinishedEvent(
            type=EventType.RUN_FINISHED,
            thread_id=thread_id,
            run_id=run_id,
        )

    async def interrupt(self, thread_id: Optional[str] = None) -> None:
        pass  # Nothing to interrupt for echo

    # --- Lifecycle (optional) ---

    def set_context(self, context: RunnerContext) -> None:
        self._context = context

    @property
    def context(self) -> RunnerContext | None:
        return self._context
```

Use it:

```python
from ambient_runner import create_ambient_app
app = create_ambient_app(MyBridge(), title="Echo Runner")
```

### Bridge Contract

| Method | Required | When Called | What To Do |
|--------|----------|------------|-----------|
| `capabilities()` | Yes | On every `/capabilities` request | Return `FrameworkCapabilities` declaring your features |
| `run(input_data)` | Yes | On every `POST /` request | Async generator yielding AG-UI `BaseEvent`s |
| `interrupt(thread_id)` | Yes | On `POST /interrupt` | Cancel the running agent |
| `set_context(context)` | No | Once at startup (lifespan) | Store `RunnerContext` for later use |
| `shutdown()` | No | Once at server shutdown | Clean up resources, persist state |
| `mark_dirty()` | No | When repos/workflows change | Signal adapter rebuild on next `run()` |
| `get_mcp_status()` | No | On `GET /mcp/status` | Return MCP server diagnostics dict |
| `get_error_context()` | No | When `run()` raises an exception | Return extra error info (e.g. stderr) |
| `context` (property) | No | By endpoints needing session_id | Return stored `RunnerContext` |
| `configured_model` (property) | No | By `/capabilities` endpoint | Return model name string |

### AG-UI Event Stream Contract

Your `run()` method **must** yield events in this order:

```
RUN_STARTED                          # Always first
├── TEXT_MESSAGE_START (role=assistant)
│   ├── TEXT_MESSAGE_CONTENT (delta=...)  # One or more
│   └── TEXT_MESSAGE_END
├── TOOL_CALL_START (optional)
│   ├── TOOL_CALL_ARGS (one or more)
│   └── TOOL_CALL_END
├── TOOL_CALL_RESULT (optional)
├── CUSTOM (optional, e.g. trace IDs)
├── MESSAGES_SNAPSHOT (optional)
RUN_FINISHED                         # Always last
```

If an error occurs, the SDK's run endpoint catches it and emits a `RUN_ERROR` event automatically — you don't need to handle that.

---

## Platform Services

Available to any bridge via `ambient_runner.platform`:

### RunnerContext

```python
from ambient_runner.platform.context import RunnerContext

# Created automatically by the lifespan from env vars:
#   SESSION_ID, WORKSPACE_PATH
# Passed to bridge via set_context()

ctx.session_id          # "my-session-123"
ctx.workspace_path      # "/workspace"
ctx.get_env("MY_VAR")   # reads from merged environment
ctx.set_metadata("k", v)  # arbitrary state store
```

### Config

```python
from ambient_runner.platform.config import (
    get_repos_config,       # Parse REPOS_JSON env var
    load_ambient_config,    # Load .ambient/ambient.json
    load_mcp_config,        # Load .mcp.json for MCP servers
)
```

### Auth

```python
from ambient_runner.platform.auth import (
    populate_runtime_credentials,  # Fetch all creds from backend API
    fetch_github_token,            # GitHub PAT
    fetch_google_credentials,      # Google OAuth
    fetch_jira_credentials,        # Jira API token
    fetch_gitlab_token,            # GitLab PAT
    sanitize_user_context,         # Clean user ID/name for logging
)
```

### Workspace

```python
from ambient_runner.platform.workspace import (
    validate_prerequisites,    # Check workspace structure
    resolve_workspace_paths,   # Get CWD + additional directories
    setup_multi_repo_paths,    # Configure multi-repo workspace
    setup_workflow_paths,      # Configure workflow directory
)
```

### Prompts

```python
from ambient_runner.platform.prompts import (
    build_workspace_context_prompt,  # Framework-agnostic workspace prompt
    RESTART_TOOL_DESCRIPTION,        # Prompt constant for restart tool
)
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SESSION_ID` | `"unknown"` | Unique session identifier |
| `WORKSPACE_PATH` | `"/workspace"` | Root workspace directory |
| `INTERACTIVE` | `"true"` | Enable interactive mode |
| `IS_RESUME` | `""` | Set to `"true"` for resumed sessions |
| `INITIAL_PROMPT` | `""` | Auto-execute this prompt on startup (non-interactive only) |
| `INITIAL_PROMPT_DELAY_SECONDS` | `"1"` | Delay before auto-prompt execution |
| `AGUI_HOST` | `"0.0.0.0"` | Server bind address |
| `AGUI_PORT` | `"8000"` | Server bind port |
| `ANTHROPIC_API_KEY` | — | Anthropic API key (Claude bridge) |
| `LLM_MODEL` | `"claude-sonnet-4-5"` | Model name override |
| `BACKEND_API_URL` | — | Platform backend URL (for credential fetching) |
| `PROJECT_NAME` | — | Project namespace |
| `BOT_TOKEN` | — | Service account token for backend API calls |
| `REPOS_JSON` | `"[]"` | JSON array of repository configs |
| `ACTIVE_WORKFLOW_GIT_URL` | — | Active workflow repository URL |
| `LANGFUSE_ENABLED` | `""` | Enable Langfuse observability (`"true"`) |
| `LANGFUSE_PUBLIC_KEY` | — | Langfuse public key |
| `LANGFUSE_SECRET_KEY` | — | Langfuse secret key |
| `LANGFUSE_HOST` | — | Langfuse server URL |

---

## HTTP API Reference

### `POST /` — Run Agent

AG-UI run endpoint. Accepts a prompt, returns an SSE event stream.

**Request:**
```json
{
  "threadId": "session-123",
  "runId": "run-abc",
  "messages": [
    {"id": "msg-1", "role": "user", "content": "Hello"}
  ],
  "tools": [],
  "state": {},
  "forwardedProps": {},
  "context": []
}
```

**Response:** `text/event-stream` with AG-UI events.

### `POST /interrupt` — Interrupt Run

```json
{"thread_id": "session-123"}
```

### `GET /health` — Health Check

```json
{"status": "healthy", "session_id": "session-123"}
```

### `GET /capabilities` — Feature Manifest

```json
{
  "framework": "claude-agent-sdk",
  "agent_features": ["agentic_chat", "thinking", "human_in_the_loop"],
  "platform_features": ["repos", "workflows", "feedback", "mcp_diagnostics"],
  "file_system": true,
  "mcp": true,
  "tracing": "langfuse",
  "session_persistence": true,
  "model": "claude-sonnet-4-5",
  "session_id": "session-123"
}
```

### `POST /feedback` — User Feedback

```json
{
  "type": "META",
  "metaType": "thumbs_up",
  "payload": {"userId": "user-1", "comment": "Great response!"}
}
```

### `POST /repos/add` — Add Repository

```json
{"url": "https://github.com/org/repo.git", "branch": "main", "name": "repo"}
```

### `POST /repos/remove` — Remove Repository

```json
{"name": "repo"}
```

### `GET /repos/status` — Repository Status

```json
{
  "repos": [
    {"name": "repo", "url": "https://...", "currentActiveBranch": "main", "branches": ["main"]}
  ]
}
```

### `POST /workflow` — Switch Workflow

```json
{"gitUrl": "https://github.com/org/workflow.git", "branch": "main", "path": ""}
```

### `GET /mcp/status` — MCP Diagnostics

```json
{
  "servers": [
    {"name": "jira", "status": "connected", "tools": [{"name": "search_issues"}]}
  ],
  "totalCount": 1
}
```

---

## Testing

```bash
# Unit tests (no API key needed)
pytest tests/ -v

# Full E2E with live agent
ANTHROPIC_API_KEY=sk-ant-... pytest tests/test_e2e_api.py -v -s
```

## Existing Bridges

| Bridge | Package | Framework | File System | MCP | Tracing |
|--------|---------|-----------|------------|-----|---------|
| `ClaudeBridge` | `ambient_runner.bridges.claude` | Claude Agent SDK | Yes | Yes | Langfuse |
| `LangGraphBridge` | `ambient_runner.bridges.langgraph` | LangGraph | No | No | LangSmith |
