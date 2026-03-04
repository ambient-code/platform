# CLI-Based Runners: Gemini CLI + Codex SDK

## Context

The current ADK bridge builds a mini coding agent from scratch with custom file/bash tools (~800 lines). A better approach: wrap existing coding agent CLIs that have all tools built in — the same pattern we already use with Claude Code via `ag_ui_claude_sdk`.

Both Gemini CLI and Codex CLI support **streaming JSONL output** over stdin/stdout, making them direct analogs to Claude Code's `--output-format stream-json`.

---

## Architecture Comparison

```
Current (Claude Code):
  Claude Agent SDK (subprocess) → ag_ui_claude_sdk adapter → AG-UI events → platform

New (same pattern):
  Gemini CLI (subprocess) → ag_ui_gemini_cli adapter → AG-UI events → platform
  Codex CLI  (subprocess) → ag_ui_codex_sdk adapter  → AG-UI events → platform
```

Each adapter is a **pure protocol translator** — no tool implementation needed. The CLIs provide all file/bash/git/search tools natively.

---

## Gemini CLI Runner

### Event Format

Gemini CLI with `--output-format stream-json` emits NDJSON with 6 event types:

| Event | Fields | Description |
|-------|--------|-------------|
| `init` | `session_id`, `model` | First event, session metadata |
| `message` | `role`, `content`, `delta?` | Text content (user or assistant) |
| `tool_use` | `tool_name`, `tool_id`, `parameters` | Tool call request |
| `tool_result` | `tool_id`, `status`, `output?`, `error?` | Tool execution result |
| `error` | `severity`, `message` | Warning or error |
| `result` | `status`, `error?`, `stats?` | Final event with token usage |

### Event Mapping: Gemini CLI → AG-UI

| Gemini Event | AG-UI Event(s) |
|---|---|
| `init` | `RUN_STARTED` |
| `message(role=user)` | (captured for context, not emitted) |
| `message(role=assistant, delta=true)` | `TEXT_MESSAGE_START` (first) + `TEXT_MESSAGE_CONTENT` |
| `tool_use` | `TOOL_CALL_START` + `TOOL_CALL_ARGS` |
| `tool_result(success)` | `TOOL_CALL_END` + `TOOL_CALL_RESULT` |
| `tool_result(error)` | `TOOL_CALL_END` + `TOOL_CALL_RESULT` (with error) |
| `error(severity=error)` | `RUN_ERROR` |
| `error(severity=warning)` | (log, don't emit) |
| `result(success)` | `TEXT_MESSAGE_END` + `RUN_FINISHED` |
| `result(error)` | `RUN_ERROR` + `RUN_FINISHED` |

### Built-in Tools (all available, no custom implementation needed)

`read_file`, `read_many_files`, `write_file`, `replace` (edit), `glob`, `grep_search`, `list_directory`, `run_shell_command`, `google_web_search`, `web_fetch`, `save_memory`, `write_todos`

### CLI Invocation

```bash
gemini -p "user prompt here" \
  --output-format stream-json \
  --yolo \                          # auto-approve all tools
  --model gemini-2.5-flash \
  --sandbox-permissions "read,write,exec" \
  --cwd /workspace
```

### Limitations

- **Single-turn per invocation** — each `gemini -p` is one session. Multi-turn requires the SDK (nightly-only) or re-invoking with `--resume`.
- **No thinking/reasoning events** in stream-json format (internal `Thought` type not exposed).
- **Tool approval** — must use `--yolo` or `--approval-mode yolo` for headless operation.
- **SDK not stable** — `@google/gemini-cli-sdk` is nightly-only on npm. CLI subprocess is the reliable path.

### Package Structure

```
ag_ui_gemini_cli/
├── __init__.py          # Exports GeminiCLIAdapter
├── config.py            # Constants, CLI flags, allowed props
├── types.py             # Python dataclasses mirroring Gemini JSONL events
├── adapter.py           # GeminiCLIAdapter — NDJSON parser → AG-UI events (~250 lines)
└── utils.py             # Message processing, tool name mapping
```

### Bridge

```
ambient_runner/bridges/gemini_cli/
├── __init__.py
├── bridge.py            # GeminiCLIBridge(PlatformBridge) (~200 lines)
├── session.py           # SessionWorker managing gemini subprocess (~150 lines)
└── auth.py              # GOOGLE_API_KEY / gcloud auth setup (~40 lines)
```

---

## Codex SDK Runner

### Event Format

Codex Python SDK (`openai-codex-sdk`) wraps `codex exec --json`, yielding JSONL with item-lifecycle events:

| Event | Key Fields | Description |
|-------|-----------|-------------|
| `thread.started` | `thread_id` | Session created |
| `turn.started` | — | New turn begins |
| `item.started` | `item: {id, type, ...}` | Tool/message begins |
| `item.updated` | `item: {id, type, ...}` | Tool/message progress |
| `item.completed` | `item: {id, type, ...}` | Tool/message finished |
| `turn.completed` | `usage` | Turn done with token counts |
| `turn.failed` | `error` | Turn errored |
| `error` | `message` | General error |

### Item Types

| Item Type | What It Represents |
|-----------|-------------------|
| `agent_message` | Text response (`.text` field) |
| `reasoning` | Chain-of-thought (`.text` field) |
| `command_execution` | Shell command (`.command`, `.aggregated_output`, `.exit_code`) |
| `file_change` | File edit (`.changes[]` with path + kind) |
| `mcp_tool_call` | MCP tool (`.server`, `.tool`, `.arguments`, `.result`) |
| `web_search` | Web search (`.query`) |
| `todo_list` | Task plan (`.items[]`) |
| `error` | Error (`.message`) |

### Event Mapping: Codex → AG-UI

| Codex Event | AG-UI Event(s) |
|---|---|
| `thread.started` | `RUN_STARTED` |
| `item.started(agent_message)` | `TEXT_MESSAGE_START` |
| `item.updated(agent_message)` | `TEXT_MESSAGE_CONTENT` (delta) |
| `item.completed(agent_message)` | `TEXT_MESSAGE_END` |
| `item.started(reasoning)` | `THINKING_START` + `THINKING_TEXT_MESSAGE_START` |
| `item.completed(reasoning)` | `THINKING_TEXT_MESSAGE_CONTENT` + `THINKING_TEXT_MESSAGE_END` + `THINKING_END` |
| `item.started(command_execution)` | `TOOL_CALL_START` |
| `item.completed(command_execution)` | `TOOL_CALL_END` + `TOOL_CALL_RESULT` |
| `item.started(file_change)` | `TOOL_CALL_START` |
| `item.completed(file_change)` | `TOOL_CALL_END` + `TOOL_CALL_RESULT` |
| `item.started(mcp_tool_call)` | `TOOL_CALL_START` |
| `item.completed(mcp_tool_call)` | `TOOL_CALL_END` + `TOOL_CALL_RESULT` |
| `turn.completed` | `RUN_FINISHED` |
| `turn.failed` | `RUN_ERROR` |

### Built-in Capabilities

Shell execution, file editing (with diffs), MCP tool calls, web search, reasoning/thinking, task planning — all built into the Codex CLI binary.

### Python SDK Usage

```python
from openai_codex_sdk import Codex

codex = Codex()
thread = codex.start_thread(config={
    "working_directory": "/workspace",
    "skip_git_repo_check": True,
})
streamed = await thread.run_streamed("Fix the failing test")
async for event in streamed.events:
    # event.type: "item.started", "item.completed", etc.
    # event.item.type: "agent_message", "command_execution", etc.
    yield translate_to_agui(event)
```

### Limitations

- **No streaming text deltas** in `exec --json` mode — `agent_message` items arrive complete, not character-by-character. True deltas require the `app-server` JSON-RPC protocol.
- **Binary not bundled** — must install the `codex` Rust binary separately via `Codex.install()` or in the Dockerfile.
- **Git repo required** by default — use `skip_git_repo_check: True` for containers.
- **Models are Codex-specific** — uses `gpt-5.1-codex`, `gpt-5.1-codex-mini`, `gpt-5.1-codex-max` (not standard GPT models).
- **Network disabled by default** — must enable explicitly.
- **Proprietary CLI binary** — the Python wrapper is Apache-2.0 but the binary has its own license.

### Package Structure

```
ag_ui_codex_sdk/
├── __init__.py          # Exports CodexAdapter
├── config.py            # Constants, sandbox/approval modes
├── types.py             # Python dataclasses mirroring Codex events
├── adapter.py           # CodexAdapter — event stream → AG-UI events (~250 lines)
└── utils.py             # Item-to-event translation helpers
```

### Bridge

```
ambient_runner/bridges/codex/
├── __init__.py
├── bridge.py            # CodexBridge(PlatformBridge) (~200 lines)
├── session.py           # Thread management (Codex has native thread persistence)
└── auth.py              # OPENAI_API_KEY setup (~30 lines)
```

---

## Shared Infrastructure Changes

### Agent Registry

Add entries to `agent-registry-configmap.yaml`:

```yaml
{
  "id": "gemini-cli",
  "displayName": "Gemini CLI",
  "description": "Google Gemini coding agent with full tool access",
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

### main.py

Add two new branches:

```python
elif RUNNER_TYPE == "gemini-cli":
    from ambient_runner.bridges.gemini_cli import GeminiCLIBridge
    return GeminiCLIBridge()
elif RUNNER_TYPE == "codex-sdk":
    from ambient_runner.bridges.codex import CodexBridge
    return CodexBridge()
```

### pyproject.toml

```toml
gemini-cli = ["@google/gemini-cli"]  # npm package, installed in Dockerfile
codex = ["openai-codex-sdk>=0.1.11"]
```

Note: Gemini CLI is an npm package (Node.js), not a pip package. It's already in the Dockerfile's Node.js environment. The Codex binary must be downloaded separately.

### Dockerfile

```dockerfile
# Gemini CLI (npm, already have Node.js)
RUN npm install -g @google/gemini-cli

# Codex CLI binary (Rust, download for platform)
RUN pip install --no-cache-dir openai-codex-sdk && \
    python -c "from openai_codex_sdk import Codex; Codex.install()"
```

### State Sync

Both CLIs store session state in their respective directories:
- Gemini: `/workspace/.gemini/` (sessions, memory)
- Codex: `/workspace/.codex/` (sessions, config)

The existing `RUNNER_STATE_DIR` mechanism handles this automatically — no state-sync changes needed.

---

## Effort Estimate

| Component | Gemini CLI | Codex SDK |
|-----------|-----------|-----------|
| Adapter package (JSONL → AG-UI) | ~300 lines | ~300 lines |
| Bridge + session + auth | ~400 lines | ~350 lines |
| ConfigMap + main.py + Dockerfile | ~30 lines | ~30 lines |
| **Total new code** | **~730 lines** | **~680 lines** |

Compare to ADK bridge: ~800 lines + ~300 lines of custom file tools = **1100 lines** for worse functionality.

### What We Get For Free (vs. ADK)

| Capability | ADK (hand-built) | Gemini CLI / Codex CLI |
|-----------|-----------------|----------------------|
| File read/write/edit | Custom FunctionTools (280 lines) | Built-in |
| Bash execution | Custom FunctionTool (40 lines) | Built-in |
| Glob/grep search | Custom FunctionTools (80 lines) | Built-in |
| Git operations | Not implemented | Built-in |
| Web search | Not implemented | Built-in |
| Memory/context | Not implemented | Built-in |
| Error recovery | Manual | Built-in (auto-retry) |
| Tool approval | N/A | Built-in (configurable) |
| Session persistence | Custom SQLite setup | Built-in (file-based) |

---

## Implementation Order

1. **Gemini CLI adapter** — highest value, replaces the ADK bridge with a better approach
2. **Codex SDK adapter** — adds OpenAI as a third provider
3. **Deprecate ADK bridge** — keep as fallback but recommend Gemini CLI for Gemini models

---

## Open Questions

1. **Gemini CLI multi-turn**: The CLI is single-turn per invocation. For multi-turn chat, we either:
   - Re-invoke with `--resume <session_id>` (if supported in headless)
   - Accumulate history and pass full context each time
   - Wait for the SDK to stabilize (nightly-only currently)

2. **Codex text streaming**: The `exec --json` mode doesn't stream text deltas — messages arrive complete. For character-by-character streaming, we'd need the `app-server` JSON-RPC protocol (more complex but possible).

3. **Codex binary licensing**: The Python wrapper is Apache-2.0 but the Rust binary has its own terms. Verify for production deployment.

4. **Image size**: Adding both CLIs to the runner image adds ~200MB (Node.js packages + Rust binary). Consider whether to keep a single fat image or create per-runner images.
