# ADR-0006: Ambient Runner SDK Architecture

**Status:** Accepted  
**Date:** 2026-02-10  
**Authors:** Gavin Krumbacher  
**Deciders:** Platform Team

## Context

The Ambient Code Platform currently has a single runner implementation tightly coupled to the Claude Agent SDK. As we look to support additional frameworks (LangGraph, Cursor SDK, etc.) and adopt the [AG-UI protocol](https://docs.ag-ui.com/) properly, we need a clean architecture that separates:

1. **Framework-specific logic** (how each SDK works)
2. **Protocol translation** (framework → AG-UI events)
3. **Platform concerns** (auth, workspace, observability, repos, workflows)
4. **Event delivery** (FastAPI, SSE, middleware)

This ADR defines the layered architecture for the Ambient Runner SDK — a reusable package that lets any AG-UI-compatible framework adapter plug into the Ambient platform.

## Decision

### Layer Architecture

```
┌──────────────────────────────────────────────────────────┐
│  Frontend (CopilotKit / Ambient UI)                      │
│  Consumes AG-UI events, shows features per capabilities  │
├──────────────────────────────────────────────────────────┤
│  Backend API (Go)                                        │
│  Proxies SSE stream, persists MESSAGES_SNAPSHOT to DB,   │
│  manages session lifecycle (K8s pods)                    │
├──────────────────────────────────────────────────────────┤
│  Ambient Runner (Python — this ADR)                      │
│  ┌────────────────────────────────────────────────────┐  │
│  │  FastAPI App                                       │  │
│  │  add_ambient_endpoints(app, bridge)                │  │
│  │                                                    │  │
│  │  ┌──────────────────────────────────────────────┐  │  │
│  │  │  Ambient Middleware (AG-UI middleware)        │  │  │
│  │  │  - Tracing (Langfuse)                        │  │  │
│  │  │  - Capability declaration                    │  │  │
│  │  └──────────────────────────────────────────────┘  │  │
│  │                                                    │  │
│  │  ┌──────────────────────────────────────────────┐  │  │
│  │  │  Platform Bridge (per framework)             │  │  │
│  │  │  Translates platform concepts → framework    │  │  │
│  │  │  config. One bridge per supported framework. │  │  │
│  │  └──────────────────────────────────────────────┘  │  │
│  │                                                    │  │
│  │  ┌──────────────────────────────────────────────┐  │  │
│  │  │  AG-UI Adapter (per framework)               │  │  │
│  │  │  ag_ui_claude_sdk, ag_ui_langgraph, etc.     │  │  │
│  │  │  Framework SDK → AG-UI events                │  │  │
│  │  └──────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────┘  │
├──────────────────────────────────────────────────────────┤
│  Framework SDK (Claude Agent SDK, LangGraph, etc.)       │
└──────────────────────────────────────────────────────────┘
```

### Layer Responsibilities

#### Layer 1: Framework SDK
The raw framework. Each has its own option shapes, tool systems, session management, and model config. **Not our code** — third-party.

#### Layer 2: AG-UI Adapter
Normalises the framework into [AG-UI events](https://docs.ag-ui.com/concepts/events). Open source, community-maintained. Each adapter implements:

```python
async def run(input_data: RunAgentInput) -> AsyncIterator[BaseEvent]
async def interrupt() -> None  # optional
```

Examples: `ag_ui_claude_sdk.ClaudeAgentAdapter`, `ag_ui_langgraph.LangGraphAgent`.

#### Layer 3: Platform Bridge
Translates platform concepts into framework-specific configuration. **One bridge per framework.** This is where framework differences are handled.

#### Layer 4: Ambient Middleware
[AG-UI middleware](https://docs.ag-ui.com/concepts/middleware) that adds platform concerns to the event stream as a side-channel. Does NOT emit protocol events — only observes and annotates.

#### Layer 5: FastAPI App + Platform Endpoints
`add_ambient_endpoints(app, bridge)` registers all platform routes. Conditional on framework capabilities.

---

## Platform Bridge

### Contract

```python
@dataclass
class PlatformContext:
    """Platform concepts provided to every framework bridge."""
    workspace_path: str
    cwd_path: str
    add_dirs: list[str]
    model: str
    max_tokens: int | None
    temperature: float | None
    system_prompt_append: str
    mcp_servers: dict
    custom_tools: list
    session_id: str
    is_continuation: bool

@dataclass
class FrameworkCapabilities:
    """What this framework can do — declared by the bridge."""
    file_system: bool = False       # Read/Write/Bash tools
    mcp: bool = False               # Native MCP support
    session_persistence: bool = False
    streaming: bool = True
    thinking: bool = False
    tool_use: bool = True

class PlatformBridge(ABC):
    @abstractmethod
    def create_adapter(self, ctx: PlatformContext) -> Any:
        """Build the AG-UI adapter with platform config applied."""
        ...

    @abstractmethod
    def capabilities(self) -> FrameworkCapabilities:
        """Declare what this framework supports."""
        ...
```

### Bridge Implementations

**Claude Agent SDK Bridge:**
```python
class ClaudeBridge(PlatformBridge):
    def capabilities(self):
        return FrameworkCapabilities(
            file_system=True, mcp=True,
            session_persistence=True, thinking=True,
        )

    def create_adapter(self, ctx):
        return ClaudeAgentAdapter(name="session", options={
            "cwd": ctx.cwd_path,
            "model": ctx.model,
            "mcp_servers": ctx.mcp_servers,    # native MCP
            "system_prompt": {"type": "preset", "preset": "claude_code", "append": ctx.system_prompt_append},
            "permission_mode": "acceptEdits",
            "continue_conversation": ctx.is_continuation,
        })
```

**LangGraph Bridge (example):**
```python
class LangGraphBridge(PlatformBridge):
    def __init__(self, graph):
        self.graph = graph

    def capabilities(self):
        return FrameworkCapabilities(
            file_system=False, mcp=False,
            session_persistence=True,
        )

    def create_adapter(self, ctx):
        return LangGraphAgent(name="session", graph=self.graph)
```

The bridge is the extension point for new frameworks. Users implement `PlatformBridge` (2 methods), and the platform handles everything else.

---

## Adapter Lifecycle

The AG-UI adapter is created **once** per session (not per run) and reused across runs. This preserves session state (e.g. `thread_id → session_id` mapping for conversation resumption).

The adapter is rebuilt only when configuration changes (workflow switch, repo add/remove).

For pod restarts (K8s), the `.claude/` directory is restored from S3 by the init container. The bridge sets `is_continuation=True` so the adapter resumes from disk state.

---

## AG-UI Middleware

Following the [AG-UI middleware pattern](https://docs.ag-ui.com/concepts/middleware), platform concerns are implemented as middleware that wraps the adapter's event stream.

### Tracing Middleware (Langfuse)

Observes AG-UI events and drives Langfuse spans/traces. Pure side-channel — no event emission.

- `TEXT_MESSAGE_START` (role=assistant) → starts a Langfuse turn trace
- `TEXT_MESSAGE_CONTENT` → accumulates text for the trace output
- `TOOL_CALL_START` → creates a Langfuse tool span
- `TOOL_CALL_END` → closes the tool span with result
- `RUN_FINISHED` → closes the turn with usage data from `result`
- `RUN_STARTED` → annotates with Langfuse trace ID in event metadata (not a separate RawEvent)

The trace ID is added to the `RUN_STARTED` event's metadata by the middleware, not injected as a separate event. The frontend reads it from there for feedback association.

### Capability Middleware

Optionally emits a `CustomEvent("ambient:capabilities", {...})` at the start of the first run so the frontend knows what features are available without a separate REST call.

---

## Platform Endpoints

Registered via `add_ambient_endpoints(app, bridge)`. Conditional on `bridge.capabilities()`.

### Always registered

| Endpoint | Method | Purpose |
|---|---|---|
| `/` | POST | AG-UI run endpoint — `adapter.run(input_data)` → SSE stream |
| `/interrupt` | POST | Interrupt active execution — `adapter.interrupt()` |
| `/health` | GET | Health check |
| `/capabilities` | GET | Returns framework + platform capabilities |
| `/feedback` | POST | Thumbs up/down → Langfuse scoring |

### Conditional on `file_system=True`

| Endpoint | Method | Purpose |
|---|---|---|
| `/repos/add` | POST | Clone repo into workspace |
| `/repos/remove` | POST | Remove repo from workspace |
| `/repos/status` | GET | Branch/status info for all repos |
| `/workflow` | POST | Change active workflow |

### Conditional on `mcp=True`

| Endpoint | Method | Purpose |
|---|---|---|
| `/mcp/status` | GET | MCP server connection diagnostics |

---

## Event Stream Patterns

### AG-UI Protocol Events (emitted by adapter)

The adapter handles all standard AG-UI events. The runner does NOT emit these:
- `RUN_STARTED`, `RUN_FINISHED`, `RUN_ERROR`
- `TEXT_MESSAGE_START/CONTENT/END`
- `TOOL_CALL_START/ARGS/END`
- `STATE_SNAPSHOT`, `STATE_DELTA`
- `MESSAGES_SNAPSHOT`
- `THINKING_START/END`, `THINKING_TEXT_MESSAGE_*`

### Developer Events (role="developer")

Platform setup lifecycle uses the standard `TextMessage` events with [`role="developer"`](https://docs.ag-ui.com/concepts/events#textmessagestart) — a first-class AG-UI role:
- "Auth connected"
- "Workspace validated"
- "MCP servers initialised"

Frontends can show/hide developer messages based on user preference (debug mode).

### Custom Events (platform-specific)

[`CustomEvent`](https://docs.ag-ui.com/concepts/events#custom) is used for platform-specific extensions that standard AG-UI clients would ignore, but the Ambient frontend understands:

- `ambient:repo_added` — repo cloned into workspace
- `ambient:repo_removed` — repo removed
- `ambient:workflow_changed` — active workflow switched
- `ambient:capabilities` — framework + platform capabilities declaration
- `ambient:setup_error` — platform setup failure details

### Meta Events (user annotations)

[`MetaEvent`](https://docs.ag-ui.com/concepts/events#metaevent) (draft spec) for user feedback that needs to be in the event stream for UI state tracking:

- `thumbs_up` — positive feedback on a message
- `thumbs_down` — negative feedback on a message

The REST `POST /feedback` endpoint triggers the Langfuse scoring AND emits a MetaEvent so the frontend can update the thumbs icon state.

---

## Message Persistence

### Principle: No double compaction

The AG-UI adapter produces `MESSAGES_SNAPSHOT` at the end of each run with the complete conversation history. This snapshot IS the compacted truth.

### Flow

```
Runner: adapter.run() → emits MESSAGES_SNAPSHOT
   ↓
Backend API: intercepts MESSAGES_SNAPSHOT in SSE proxy → persists to DB
   ↓
Session resume: Backend loads messages from DB → sends as RunAgentInput.messages
   ↓
Runner: feeds messages to adapter → SDK continues conversation
```

The runner never touches the DB. The backend does not reconstruct or compact messages — it stores what `MESSAGES_SNAPSHOT` provides.

### Framework-specific persistence

The Claude Agent SDK persists its own session state in `.claude/` on disk. This is synced to/from S3 for pod restarts. This is complementary to (not competing with) the backend's DB persistence:

| What | Where | Purpose |
|---|---|---|
| `.claude/` directory | S3 ↔ PVC | SDK session resume (framework concern) |
| `MESSAGES_SNAPSHOT` | Backend DB | Platform history, audit trail, multi-client sync |

---

## Public API

### `add_ambient_endpoints(app, bridge)`

The primary integration point. Follows the `add_langgraph_fastapi_endpoint` pattern — adds routes to YOUR app, doesn't create one.

```python
from ambient_runner import add_ambient_endpoints
from ambient_runner.bridges.claude import ClaudeBridge

app = FastAPI()
add_ambient_endpoints(app, bridge=ClaudeBridge())
```

### Adding a new framework

1. Implement `PlatformBridge` (2 methods: `create_adapter` + `capabilities`)
2. Optionally implement framework-specific middleware
3. Call `add_ambient_endpoints(app, bridge=YourBridge())`

```python
from ambient_runner import add_ambient_endpoints, PlatformBridge, PlatformContext, FrameworkCapabilities

class MyBridge(PlatformBridge):
    def capabilities(self):
        return FrameworkCapabilities(file_system=False, mcp=False)

    def create_adapter(self, ctx):
        return MyAdapter(model=ctx.model, prompt=ctx.system_prompt_append)

app = FastAPI()
add_ambient_endpoints(app, bridge=MyBridge())
```

---

## Implementation Phases

### Phase 1: Foundation (current PR)
- [x] Vendored `ag_ui_claude_sdk` package
- [x] Clean adapter as functions module (not class)
- [x] Split modules: `auth.py`, `workspace.py`, `prompts.py`, `mcp.py`
- [x] Split endpoints: `endpoints/repos.py`, `endpoints/workflow.py`, `endpoints/feedback.py`, `endpoints/mcp_status.py`
- [x] Observability tracks AG-UI events (not SDK messages)
- [x] Langfuse trace ID emitted once per run

### Phase 2: Tracing middleware
- [x] Move `obs.track_agui_event()` into a proper AG-UI middleware class
- [x] Trace ID injected via CustomEvent (not separate RawEvent)
- [x] Developer events for setup lifecycle (role="developer")

### Phase 3: Capabilities endpoint
- [x] `GET /capabilities` returns framework + platform features
- [x] Frontend reads capabilities and shows/hides UI features
- [x] Conditional endpoint registration based on capabilities

### Phase 4: `ambient_runner` package extraction
- [x] Extract into standalone package: `ambient_runner`
- [x] `add_ambient_endpoints(app, bridge)` as public API
- [x] `PlatformBridge` ABC and `PlatformContext`/`FrameworkCapabilities` types
- [x] `ClaudeBridge` as first implementation
- [x] Current `main.py` becomes a thin consumer

### Phase 5: Adapter persistence
- [x] Adapter created once, reused across runs
- [x] Rebuild only on config change (workflow, repo)
- [x] Session resume via bridge + `.claude/` S3 sync

### Phase 6: Backend consumes MESSAGES_SNAPSHOT
- [x] Backend proxy intercepts `MESSAGES_SNAPSHOT` events
- [x] Persists to DB directly (no separate compaction)
- [x] On session resume, loads from DB into `RunAgentInput.messages`
- [x] Remove duplicated compaction logic from backend

### Phase 7: CopilotKit frontend adoption
- [x] Replace custom `useAGUIStream` hook with CopilotKit's `useCopilotChat`
- [x] Use `useCopilotAction` for human-in-the-loop tools
- [x] MetaEvents for feedback UI state

### Phase 8: Second framework (LangGraph)
- [x] `LangGraphBridge` implementation
- [x] Validates the abstraction works across frameworks
- [x] Different capabilities (no file_system, no MCP)
- [x] Frontend adapts UI based on capabilities

---

## Consequences

### Benefits
- **Framework-agnostic**: Any AG-UI adapter plugs in via the bridge pattern
- **Clean separation**: Platform concerns (auth, repos, workflows) are isolated from protocol translation
- **Standards-based**: Uses AG-UI events, middleware, and roles as designed
- **No double work**: MESSAGES_SNAPSHOT eliminates duplicate compaction
- **Extensible**: New frameworks = implement 2 methods, get all platform features

### Trade-offs
- **Bridge per framework**: Each new framework needs a bridge implementation (but it's just 2 methods)
- **MetaEvent is draft spec**: Feedback via MetaEvent depends on AG-UI finalising the draft
- **Adapter reuse**: Need to handle config changes carefully (rebuild vs reuse)

### Risks
- AG-UI protocol is still evolving (MetaEvents, Interrupts are drafts)
- CopilotKit adoption is a frontend-wide change
- LangGraph bridge may surface abstraction gaps not visible with Claude SDK alone

---

## References

- [AG-UI Protocol — Events](https://docs.ag-ui.com/concepts/events)
- [AG-UI Protocol — Middleware](https://docs.ag-ui.com/concepts/middleware)
- [AG-UI Dojo — Feature Registry (menu.ts)](https://github.com/ag-ui-protocol/ag-ui/blob/main/apps/dojo/src/menu.ts)
- [AG-UI LangGraph Integration](https://github.com/ag-ui-protocol/ag-ui/tree/main/integrations/langgraph)
- [AG-UI Claude Agent SDK Integration](https://github.com/ag-ui-protocol/ag-ui/tree/main/integrations/claude-agent-sdk)
- [ADR-0004: Go Backend, Python Runner](../adr/0004-go-backend-python-runner.md)
- [ADR-0001: Kubernetes-Native Architecture](../adr/0001-kubernetes-native-architecture.md)
