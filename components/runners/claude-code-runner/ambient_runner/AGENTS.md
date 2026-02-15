# Ambient Runner SDK — Agent Context

> This file is optimised for AI coding agents. It contains the rules,
> patterns, and constraints needed to work with the ambient_runner package.

## What This Package Is

`ambient_runner` is a reusable SDK for building AG-UI agent runners on the
Ambient Code Platform. It provides the FastAPI server, endpoints, lifespan,
and platform services. Framework-specific logic lives in bridge subpackages.

## Package Structure (authoritative)

```
ambient_runner/
├── app.py                    # create_ambient_app(), run_ambient_app(), add_ambient_endpoints()
├── bridge.py                 # PlatformBridge ABC, FrameworkCapabilities, RunnerContext import
├── observability.py          # ObservabilityManager (Langfuse, ~900 lines)
├── bridges/
│   ├── claude/               # Claude Agent SDK bridge
│   │   ├── bridge.py         # ClaudeBridge class (full lifecycle)
│   │   ├── auth.py           # API key + Vertex AI setup
│   │   ├── mcp.py            # MCP server building + auth checks
│   │   ├── prompts.py        # Claude Code preset prompt wrapping
│   │   ├── session.py        # SessionManager, SessionWorker
│   │   └── tools.py          # restart_session, evaluate_rubric MCP tools
│   └── langgraph/
│       └── bridge.py         # LangGraphBridge (minimal reference)
├── endpoints/                # FastAPI routers (access bridge via request.app.state.bridge)
│   ├── run.py                # POST /
│   ├── interrupt.py          # POST /interrupt
│   ├── health.py             # GET /health
│   ├── capabilities.py       # GET /capabilities
│   ├── feedback.py           # POST /feedback
│   ├── repos.py              # POST /repos/add, POST /repos/remove, GET /repos/status
│   ├── workflow.py           # POST /workflow
│   └── mcp_status.py         # GET /mcp/status
├── middleware/
│   ├── tracing.py            # Langfuse tracing wrapper for event streams
│   └── developer_events.py   # Emits developer-role AG-UI TextMessages
└── platform/                 # Framework-agnostic services
    ├── context.py            # RunnerContext dataclass
    ├── config.py             # ambient.json, MCP config, repos config loading
    ├── auth.py               # Credential fetching from backend API
    ├── workspace.py          # Path resolution, multi-repo, workflow setup
    ├── prompts.py            # Workspace context prompt builder + constants
    ├── security_utils.py     # Sanitization, timeout helpers
    └── utils.py              # parse_owner_repo, shared utilities
```

## Critical Rules

### Layer Dependencies (NEVER violate)

- `endpoints/` → accesses bridge ONLY via `request.app.state.bridge`
- `endpoints/` → NEVER imports from `bridges/claude/` or `bridges/langgraph/`
- `middleware/` → depends ONLY on `ag_ui.core` types
- `bridges/claude/` → may import from `platform/`, `middleware/`, `ag_ui_claude_sdk`
- `bridges/langgraph/` → may import from `platform/`, `ag_ui_langgraph`
- `platform/` → NEVER imports from `bridges/`, `endpoints/`, or `middleware/`

### PlatformBridge Contract

Three abstract methods MUST be implemented:

```python
class PlatformBridge(ABC):
    @abstractmethod
    def capabilities(self) -> FrameworkCapabilities: ...

    @abstractmethod
    async def run(self, input_data: RunAgentInput) -> AsyncIterator[BaseEvent]: ...

    @abstractmethod
    async def interrupt(self, thread_id: Optional[str] = None) -> None: ...
```

Optional lifecycle methods (with safe defaults):

```python
    def set_context(self, context: RunnerContext) -> None: pass
    async def shutdown(self) -> None: pass
    def mark_dirty(self) -> None: pass
    async def get_mcp_status(self) -> dict: return {"servers": [], "totalCount": 0}
    def get_error_context(self) -> str: return ""
```

Properties endpoints use:

```python
    @property
    def context(self) -> Optional[RunnerContext]: return None

    @property
    def configured_model(self) -> str: return ""

    @property
    def obs(self) -> Any: return None
```

### AG-UI Event Stream Rules

The `run()` method MUST:
1. Yield `RUN_STARTED` as the FIRST event
2. Yield `RUN_FINISHED` as the LAST event
3. Wrap text in `TEXT_MESSAGE_START` → `TEXT_MESSAGE_CONTENT` (1+) → `TEXT_MESSAGE_END`
4. Wrap tool calls in `TOOL_CALL_START` → `TOOL_CALL_ARGS` (1+) → `TOOL_CALL_END`

The `run()` method MUST NOT:
- Yield `RUN_ERROR` (the endpoint does this automatically on exception)
- Catch and swallow exceptions silently (let them propagate)

### Endpoint Pattern

All endpoints access the bridge through `request.app.state.bridge`:

```python
@router.get("/my-endpoint")
async def my_endpoint(request: Request):
    bridge = request.app.state.bridge
    context = bridge.context
    # ... use bridge methods ...
```

NEVER import bridge classes directly in endpoint files.
NEVER store mutable global state in endpoint modules.

### Adding a New Bridge

1. Create `ambient_runner/bridges/my_framework/__init__.py`:
   ```python
   from ambient_runner.bridges.my_framework.bridge import MyBridge
   __all__ = ["MyBridge"]
   ```

2. Create `ambient_runner/bridges/my_framework/bridge.py`:
   - Subclass `PlatformBridge`
   - Implement `capabilities()`, `run()`, `interrupt()`
   - Override lifecycle methods as needed

3. Add the package to `pyproject.toml`:
   ```toml
   packages = [..., "ambient_runner.bridges.my_framework"]
   ```

4. Add optional deps if needed:
   ```toml
   [project.optional-dependencies]
   my_framework = ["my-framework-sdk>=1.0"]
   ```

### Adding a New Endpoint

1. Create `ambient_runner/endpoints/my_endpoint.py`
2. Define a `router = APIRouter()`
3. Access bridge via `request.app.state.bridge`
4. Register in `ambient_runner/app.py` → `add_ambient_endpoints()`

### Environment Variables (read by lifespan)

| Variable | Default | Used By |
|----------|---------|---------|
| `SESSION_ID` | `"unknown"` | RunnerContext creation |
| `WORKSPACE_PATH` | `"/workspace"` | RunnerContext creation |
| `INTERACTIVE` | `"true"` | Auto-prompt decision |
| `IS_RESUME` | `""` | Auto-prompt + session continuation |
| `INITIAL_PROMPT` | `""` | Auto-execute on startup |
| `AGUI_HOST` | `"0.0.0.0"` | `run_ambient_app()` |
| `AGUI_PORT` | `"8000"` | `run_ambient_app()` |

## Common Operations

### Run tests

```bash
# All tests
pytest tests/ -v

# E2E only (structural — no API key)
pytest tests/test_e2e_api.py -v

# E2E with live agent
ANTHROPIC_API_KEY=... pytest tests/test_e2e_api.py -v -s
```

### Build container

```bash
make build-runner CONTAINER_ENGINE=podman REGISTRY=quay.io/my-registry
```

### Key files to read first

1. `bridge.py` — the contract (read this first)
2. `app.py` — how the app is assembled
3. `bridges/claude/bridge.py` — full reference implementation
4. `bridges/langgraph/bridge.py` — minimal reference implementation
5. `endpoints/run.py` — how the run endpoint delegates to bridge
