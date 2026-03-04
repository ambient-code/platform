# CLI Runners Implementation Plan

## Goal

Add Gemini CLI and Codex SDK as runners alongside Claude Code. Both use the same pattern: wrap a CLI subprocess that emits JSONL, translate events to AG-UI protocol.

---

## Team Structure

```
Lead ──────────── Infrastructure (main.py, Dockerfile, ConfigMap, pyproject.toml)
  ├── gemini-agent ── Gemini CLI adapter + bridge (full package)
  └── codex-agent ─── Codex SDK adapter + bridge (full package)
```

**All three work in parallel.** No file conflicts — each agent writes to its own directory.

---

## Phase 0: Lead — Infrastructure (done first, ~5 min)

### 0.1 — `main.py`: Add new RUNNER_TYPE branches

```python
elif RUNNER_TYPE == "gemini-cli":
    from ambient_runner.bridges.gemini_cli import GeminiCLIBridge
    return GeminiCLIBridge()
elif RUNNER_TYPE == "codex-sdk":
    from ambient_runner.bridges.codex import CodexBridge
    return CodexBridge()
```

### 0.2 — `pyproject.toml`: Add dependencies

```toml
[project.optional-dependencies]
codex = ["openai-codex-sdk>=0.1.11"]
# gemini-cli is npm, installed in Dockerfile

all = [
  "ambient-runner[claude,adk,codex,observability,mcp-atlassian]",
]
```

### 0.3 — `Dockerfile`: Install CLI binaries

```dockerfile
# Gemini CLI (npm package, Node.js already available)
RUN npm install -g @google/gemini-cli

# Codex CLI binary (Rust, via Python wrapper)
RUN pip install --no-cache-dir openai-codex-sdk && \
    python -c "from openai_codex_sdk import Codex; Codex.install()"
```

### 0.4 — Agent Registry ConfigMap

Add two entries to `components/manifests/base/agent-registry-configmap.yaml`:

```json
{
  "id": "gemini-cli",
  "displayName": "Gemini CLI",
  "description": "Google Gemini coding agent with built-in file, bash, and search tools",
  "defaultModel": "gemini-2.5-flash",
  "models": [
    {"value": "gemini-2.5-flash", "label": "Gemini 2.5 Flash"},
    {"value": "gemini-2.5-pro", "label": "Gemini 2.5 Pro"},
    {"value": "gemini-2.0-flash", "label": "Gemini 2.0 Flash"}
  ],
  "requiredSecrets": ["GOOGLE_API_KEY"],
  "internalEnvVars": {
    "RUNNER_TYPE": "gemini-cli",
    "RUNNER_STATE_DIR": ".gemini"
  }
},
{
  "id": "codex-sdk",
  "displayName": "OpenAI Codex",
  "description": "OpenAI Codex coding agent with sandbox execution",
  "defaultModel": "gpt-5.1-codex",
  "models": [
    {"value": "gpt-5.1-codex", "label": "GPT 5.1 Codex"},
    {"value": "gpt-5.1-codex-max", "label": "GPT 5.1 Codex Max"},
    {"value": "gpt-5.1-codex-mini", "label": "GPT 5.1 Codex Mini"}
  ],
  "requiredSecrets": ["OPENAI_API_KEY"],
  "internalEnvVars": {
    "RUNNER_TYPE": "codex-sdk",
    "RUNNER_STATE_DIR": ".codex"
  }
}
```

### 0.5 — Empty package placeholders

Create `__init__.py` files so the imports in `main.py` will resolve:

```
ag_ui_gemini_cli/__init__.py
ag_ui_codex_sdk/__init__.py
ambient_runner/bridges/gemini_cli/__init__.py
ambient_runner/bridges/codex/__init__.py
```

### 0.6 — Register packages in setuptools

Add to `pyproject.toml` packages list:
```
"ag_ui_gemini_cli", "ag_ui_codex_sdk",
"ambient_runner.bridges.gemini_cli", "ambient_runner.bridges.codex"
```

---

## Phase 1: gemini-agent — Gemini CLI Adapter + Bridge

**Scope**: Everything under `ag_ui_gemini_cli/` and `ambient_runner/bridges/gemini_cli/`

### Reference files to read first

1. `ag_ui_claude_sdk/adapter.py` — the pattern to follow
2. `ag_ui_claude_sdk/utils.py` — message processing helpers
3. `ag_ui_claude_sdk/handlers.py` — block-level event handlers
4. `ag_ui_claude_sdk/config.py` — constants
5. `ambient_runner/bridges/claude/bridge.py` — bridge lifecycle
6. `ambient_runner/bridges/claude/session.py` — subprocess session management
7. `ambient_runner/bridge.py` — PlatformBridge ABC

### 1.1 — `ag_ui_gemini_cli/types.py` (~60 lines)

Python dataclasses mirroring Gemini CLI's 6 JSONL event types:

```python
@dataclass
class InitEvent:
    type: Literal["init"]
    timestamp: str
    session_id: str
    model: str

@dataclass
class MessageEvent:
    type: Literal["message"]
    timestamp: str
    role: Literal["user", "assistant"]
    content: str
    delta: bool = False

@dataclass
class ToolUseEvent:
    type: Literal["tool_use"]
    timestamp: str
    tool_name: str
    tool_id: str
    parameters: dict

@dataclass
class ToolResultEvent:
    type: Literal["tool_result"]
    timestamp: str
    tool_id: str
    status: Literal["success", "error"]
    output: str | None = None
    error: dict | None = None  # {"type": str, "message": str}

@dataclass
class ErrorEvent:
    type: Literal["error"]
    timestamp: str
    severity: Literal["warning", "error"]
    message: str

@dataclass
class ResultEvent:
    type: Literal["result"]
    timestamp: str
    status: Literal["success", "error"]
    error: dict | None = None
    stats: dict | None = None  # {total_tokens, input_tokens, output_tokens, ...}
```

### 1.2 — `ag_ui_gemini_cli/config.py` (~20 lines)

```python
STATE_MANAGEMENT_TOOL_NAME = "ag_ui_update_state"
AG_UI_MCP_SERVER_NAME = "ag_ui"  # Same as Claude adapter
DEFAULT_MODEL = "gemini-2.5-flash"
```

### 1.3 — `ag_ui_gemini_cli/adapter.py` (~250 lines)

`GeminiCLIAdapter` class:

```python
class GeminiCLIAdapter:
    """Translates Gemini CLI stream-json NDJSON to AG-UI events."""

    async def run(self, input_data: RunAgentInput, *, line_stream) -> AsyncIterator[BaseEvent]:
        """
        Args:
            input_data: AG-UI run input
            line_stream: async iterator of NDJSON lines from gemini subprocess

        Yields:
            AG-UI BaseEvent instances
        """
```

Event translation logic:

```python
async def _process_stream(self, line_stream, thread_id, run_id, ...):
    text_started = False
    current_message_id = None

    async for line in line_stream:
        event = parse_jsonl_event(line)

        if event.type == "init":
            # Capture session_id, model
            pass

        elif event.type == "message":
            if event.role == "assistant" and event.delta:
                if not text_started:
                    current_message_id = str(uuid4())
                    yield TextMessageStartEvent(messageId=current_message_id, role="assistant")
                    text_started = True
                yield TextMessageContentEvent(messageId=current_message_id, delta=event.content)

        elif event.type == "tool_use":
            # Close any open text message first
            if text_started:
                yield TextMessageEndEvent(messageId=current_message_id)
                text_started = False
            yield ToolCallStartEvent(toolCallId=event.tool_id, toolCallName=event.tool_name)
            yield ToolCallArgsEvent(toolCallId=event.tool_id, delta=json.dumps(event.parameters))

        elif event.type == "tool_result":
            result = event.output if event.status == "success" else json.dumps(event.error)
            yield ToolCallEndEvent(toolCallId=event.tool_id)

        elif event.type == "error":
            if event.severity == "error":
                yield RunErrorEvent(message=event.message)

        elif event.type == "result":
            if text_started:
                yield TextMessageEndEvent(messageId=current_message_id)
            # Stats available in event.stats
```

### 1.4 — `ag_ui_gemini_cli/utils.py` (~50 lines)

- `parse_jsonl_event(line: str) -> GeminiEvent` — parse one NDJSON line
- `extract_user_message(input_data: RunAgentInput) -> str` — get prompt text from AG-UI input
- Reuse `process_messages()` from `ag_ui_claude_sdk.utils` if possible (it's mostly SDK-agnostic)

### 1.5 — `ambient_runner/bridges/gemini_cli/bridge.py` (~200 lines)

```python
class GeminiCLIBridge(PlatformBridge):
    def capabilities(self) -> FrameworkCapabilities:
        return FrameworkCapabilities(
            framework="gemini-cli",
            agent_features=["agentic_chat", "backend_tool_rendering"],
            file_system=True,
            mcp=True,  # Gemini CLI supports MCP servers
            tracing="langfuse" if has_tracing else None,
        )

    async def run(self, input_data):
        await self._ensure_ready()
        # Refresh credentials if stale
        # Get or create session worker for thread
        # Get line stream from worker.query(prompt)
        # Pass to adapter.run(input_data, line_stream=stream)
        # Wrap with tracing_middleware
```

Follows the exact same lazy-setup pattern as ClaudeBridge:
- `_setup_platform()` — auth, workspace, observability
- `_ensure_adapter()` — create GeminiCLIAdapter
- Session worker manages the `gemini` subprocess

### 1.6 — `ambient_runner/bridges/gemini_cli/session.py` (~180 lines)

`GeminiSessionWorker` — manages the `gemini` CLI subprocess:

```python
class GeminiSessionWorker:
    """Manages a gemini CLI subprocess for one conversation thread."""

    async def start(self):
        """Spawn: gemini -p <prompt> --output-format stream-json --yolo --model <model>"""

    async def query(self, prompt: str, session_id: str) -> AsyncIterator[str]:
        """Send prompt, yield NDJSON lines from stdout."""
        # For multi-turn: re-invoke gemini with --resume <session_id>
        # For first turn: new invocation
        self._process = await asyncio.create_subprocess_exec(
            "gemini", "-p", prompt,
            "--output-format", "stream-json",
            "--yolo",
            "--model", self._model,
            "--cwd", self._cwd,
            *(["--resume", session_id] if session_id else []),
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        async for line in self._process.stdout:
            yield line.decode().strip()

    async def interrupt(self):
        """Send SIGINT to subprocess."""
        if self._process:
            self._process.send_signal(signal.SIGINT)

    async def stop(self):
        """Graceful shutdown."""
```

Key difference from Claude's SessionWorker: Gemini CLI is **one invocation per turn** (not a long-lived process). Each `query()` spawns a new subprocess. For multi-turn, use `--resume <session_id>`.

### 1.7 — `ambient_runner/bridges/gemini_cli/auth.py` (~40 lines)

```python
async def setup_gemini_cli_auth(context):
    """Configure Gemini CLI authentication.

    Supports:
    - GOOGLE_API_KEY env var (direct API key)
    - gcloud auth (Application Default Credentials)
    """
    api_key = os.getenv("GOOGLE_API_KEY", "").strip()
    model = os.getenv("LLM_MODEL", "gemini-2.5-flash").strip()
    return model
```

### 1.8 — `ambient_runner/bridges/gemini_cli/__init__.py`

```python
from ambient_runner.bridges.gemini_cli.bridge import GeminiCLIBridge
__all__ = ["GeminiCLIBridge"]
```

---

## Phase 2: codex-agent — Codex SDK Adapter + Bridge

**Scope**: Everything under `ag_ui_codex_sdk/` and `ambient_runner/bridges/codex/`

### Reference files to read first

Same as gemini-agent, plus:
- `openai-codex-sdk` Python package source (pip install and inspect)
- Codex SDK docs at https://developers.openai.com/codex/sdk/

### 2.1 — `ag_ui_codex_sdk/types.py` (~80 lines)

Python dataclasses mirroring Codex event types:

```python
@dataclass
class ThreadStartedEvent:
    type: Literal["thread.started"]
    thread_id: str

@dataclass
class TurnStartedEvent:
    type: Literal["turn.started"]

@dataclass
class TurnCompletedEvent:
    type: Literal["turn.completed"]
    usage: dict  # {input_tokens, cached_input_tokens, output_tokens}

@dataclass
class TurnFailedEvent:
    type: Literal["turn.failed"]
    error: dict  # {message}

@dataclass
class ItemStartedEvent:
    type: Literal["item.started"]
    item: ThreadItem

@dataclass
class ItemUpdatedEvent:
    type: Literal["item.updated"]
    item: ThreadItem

@dataclass
class ItemCompletedEvent:
    type: Literal["item.completed"]
    item: ThreadItem

@dataclass
class ThreadErrorEvent:
    type: Literal["error"]
    message: str

# Item types
@dataclass
class AgentMessageItem:
    id: str
    type: Literal["agent_message"]
    text: str

@dataclass
class ReasoningItem:
    id: str
    type: Literal["reasoning"]
    text: str

@dataclass
class CommandExecutionItem:
    id: str
    type: Literal["command_execution"]
    command: str
    aggregated_output: str = ""
    exit_code: int | None = None
    status: str = "in_progress"

@dataclass
class FileChangeItem:
    id: str
    type: Literal["file_change"]
    changes: list  # [{path, kind: "add"|"delete"|"update"}]
    status: str = "completed"

@dataclass
class McpToolCallItem:
    id: str
    type: Literal["mcp_tool_call"]
    server: str
    tool: str
    arguments: dict
    result: dict | None = None
    error: dict | None = None
    status: str = "in_progress"
```

### 2.2 — `ag_ui_codex_sdk/config.py` (~20 lines)

```python
STATE_MANAGEMENT_TOOL_NAME = "ag_ui_update_state"
AG_UI_MCP_SERVER_NAME = "ag_ui"
DEFAULT_MODEL = "gpt-5.1-codex"
DEFAULT_APPROVAL_MODE = "never"
DEFAULT_SANDBOX_MODE = "workspace-write"
```

### 2.3 — `ag_ui_codex_sdk/adapter.py` (~280 lines)

`CodexAdapter` class:

```python
class CodexAdapter:
    """Translates Codex SDK events to AG-UI events."""

    async def run(self, input_data: RunAgentInput, *, event_stream) -> AsyncIterator[BaseEvent]:
        """
        Args:
            input_data: AG-UI run input
            event_stream: async iterator from codex thread.run_streamed()
        """
```

Event translation:

```python
async def _process_stream(self, event_stream, thread_id, run_id, ...):
    async for event in event_stream:
        if event.type == "thread.started":
            # Capture thread_id
            pass

        elif event.type == "item.started":
            item = event.item
            if item.type == "agent_message":
                msg_id = item.id
                yield TextMessageStartEvent(messageId=msg_id, role="assistant")
                if item.text:
                    yield TextMessageContentEvent(messageId=msg_id, delta=item.text)

            elif item.type == "reasoning":
                yield CustomEvent(name="thinking_start", value={})

            elif item.type == "command_execution":
                yield ToolCallStartEvent(toolCallId=item.id, toolCallName="bash")
                yield ToolCallArgsEvent(toolCallId=item.id, delta=json.dumps({"command": item.command}))

            elif item.type == "file_change":
                yield ToolCallStartEvent(toolCallId=item.id, toolCallName="file_edit")
                yield ToolCallArgsEvent(toolCallId=item.id, delta=json.dumps({"changes": item.changes}))

            elif item.type == "mcp_tool_call":
                yield ToolCallStartEvent(toolCallId=item.id, toolCallName=item.tool)
                yield ToolCallArgsEvent(toolCallId=item.id, delta=json.dumps(item.arguments))

        elif event.type == "item.updated":
            item = event.item
            if item.type == "agent_message":
                yield TextMessageContentEvent(messageId=item.id, delta=item.text)

        elif event.type == "item.completed":
            item = event.item
            if item.type == "agent_message":
                yield TextMessageEndEvent(messageId=item.id)

            elif item.type == "reasoning":
                yield CustomEvent(name="thinking_end", value={"text": item.text})

            elif item.type == "command_execution":
                yield ToolCallEndEvent(toolCallId=item.id)

            elif item.type == "file_change":
                yield ToolCallEndEvent(toolCallId=item.id)

            elif item.type == "mcp_tool_call":
                yield ToolCallEndEvent(toolCallId=item.id)

        elif event.type == "turn.completed":
            # Stats in event.usage
            pass

        elif event.type == "turn.failed":
            yield RunErrorEvent(message=event.error.get("message", "Unknown error"))

        elif event.type == "error":
            yield RunErrorEvent(message=event.message)
```

### 2.4 — `ag_ui_codex_sdk/utils.py` (~40 lines)

- `parse_codex_event(raw) -> CodexEvent` — parse event dict to typed dataclass
- `extract_user_message(input_data) -> str`

### 2.5 — `ambient_runner/bridges/codex/bridge.py` (~200 lines)

```python
class CodexBridge(PlatformBridge):
    def capabilities(self) -> FrameworkCapabilities:
        return FrameworkCapabilities(
            framework="codex-sdk",
            agent_features=["agentic_chat", "backend_tool_rendering", "thinking"],
            file_system=True,
            mcp=True,  # Codex supports MCP
            tracing="langfuse" if has_tracing else None,
        )

    async def run(self, input_data):
        await self._ensure_ready()
        # Create/resume Codex thread
        # Call thread.run_streamed(prompt)
        # Pass event stream to adapter.run()
        # Wrap with tracing_middleware
```

### 2.6 — `ambient_runner/bridges/codex/session.py` (~120 lines)

Codex SDK has **native thread persistence** — simpler than Gemini/Claude:

```python
class CodexSessionManager:
    """Manages Codex SDK threads."""

    def __init__(self):
        self._codex = None
        self._threads: dict[str, Thread] = {}

    def _ensure_client(self):
        if not self._codex:
            from openai_codex_sdk import Codex
            self._codex = Codex()

    async def get_or_create_thread(self, thread_id, config):
        if thread_id in self._threads:
            return self._threads[thread_id]
        self._ensure_client()
        thread = self._codex.start_thread(config={
            "working_directory": config.get("cwd", "/workspace"),
            "skip_git_repo_check": True,
        })
        self._threads[thread_id] = thread
        return thread

    async def resume_thread(self, thread_id):
        self._ensure_client()
        thread = self._codex.resume_thread(thread_id)
        self._threads[thread_id] = thread
        return thread
```

### 2.7 — `ambient_runner/bridges/codex/auth.py` (~30 lines)

```python
async def setup_codex_auth(context):
    """Configure Codex authentication via OPENAI_API_KEY."""
    api_key = os.getenv("OPENAI_API_KEY", "").strip()
    if not api_key:
        raise RuntimeError("OPENAI_API_KEY not set")
    model = os.getenv("LLM_MODEL", "gpt-5.1-codex").strip()
    return model
```

### 2.8 — `ambient_runner/bridges/codex/__init__.py`

```python
from ambient_runner.bridges.codex.bridge import CodexBridge
__all__ = ["CodexBridge"]
```

---

## Agent Team Execution Plan

### Setup (Lead, before spawning agents)

1. Create empty `__init__.py` placeholder packages
2. Update `main.py` with new RUNNER_TYPE branches
3. Update `pyproject.toml` with new deps and package list
4. Update ConfigMap with new entries
5. Commit as base

### Parallel Execution (2 agents, spawned simultaneously)

| Agent | Scope | Files | Est. Lines |
|-------|-------|-------|-----------|
| **gemini-agent** | `ag_ui_gemini_cli/` (4 files) + `bridges/gemini_cli/` (4 files) | 8 new files | ~750 |
| **codex-agent** | `ag_ui_codex_sdk/` (4 files) + `bridges/codex/` (4 files) | 8 new files | ~700 |

**Zero file overlap** — each agent writes to its own directories.

### Post-Merge (Lead)

1. Update Dockerfile with CLI installations
2. Verify all imports work
3. Run `/simplify` review
4. Commit + build + push + rollout

---

## File Summary

| Phase | File | Change |
|-------|------|--------|
| 0 | `runners/ambient-runner/main.py` | Add gemini-cli + codex-sdk branches |
| 0 | `runners/ambient-runner/pyproject.toml` | Add deps + packages |
| 0 | `runners/ambient-runner/Dockerfile` | Install CLI binaries |
| 0 | `manifests/base/agent-registry-configmap.yaml` | Add 2 entries |
| 1 | `runners/ambient-runner/ag_ui_gemini_cli/__init__.py` | New |
| 1 | `runners/ambient-runner/ag_ui_gemini_cli/types.py` | New (~60 lines) |
| 1 | `runners/ambient-runner/ag_ui_gemini_cli/config.py` | New (~20 lines) |
| 1 | `runners/ambient-runner/ag_ui_gemini_cli/adapter.py` | New (~250 lines) |
| 1 | `runners/ambient-runner/ag_ui_gemini_cli/utils.py` | New (~50 lines) |
| 1 | `runners/ambient-runner/.../bridges/gemini_cli/__init__.py` | New |
| 1 | `runners/ambient-runner/.../bridges/gemini_cli/bridge.py` | New (~200 lines) |
| 1 | `runners/ambient-runner/.../bridges/gemini_cli/session.py` | New (~180 lines) |
| 1 | `runners/ambient-runner/.../bridges/gemini_cli/auth.py` | New (~40 lines) |
| 2 | `runners/ambient-runner/ag_ui_codex_sdk/__init__.py` | New |
| 2 | `runners/ambient-runner/ag_ui_codex_sdk/types.py` | New (~80 lines) |
| 2 | `runners/ambient-runner/ag_ui_codex_sdk/config.py` | New (~20 lines) |
| 2 | `runners/ambient-runner/ag_ui_codex_sdk/adapter.py` | New (~280 lines) |
| 2 | `runners/ambient-runner/ag_ui_codex_sdk/utils.py` | New (~40 lines) |
| 2 | `runners/ambient-runner/.../bridges/codex/__init__.py` | New |
| 2 | `runners/ambient-runner/.../bridges/codex/bridge.py` | New (~200 lines) |
| 2 | `runners/ambient-runner/.../bridges/codex/session.py` | New (~120 lines) |
| 2 | `runners/ambient-runner/.../bridges/codex/auth.py` | New (~30 lines) |

---

## Verification

1. **Gemini CLI**: Create session → pick "Gemini CLI" → send message → text streams back → tool calls visible → multi-turn works via --resume
2. **Codex SDK**: Create session → pick "OpenAI Codex" → send message → text appears → file edits and bash visible as tool calls → reasoning blocks render
3. **Backward compat**: Claude Code sessions unchanged
4. **ADK sessions**: Still work (keep as alternative Gemini path)
