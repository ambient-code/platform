# Claude Agent SDK Integration Guide

**Author:** Platform Team
**Date:** 2026-01-27
**Status:** Active

## Overview

This document describes how the Ambient platform integrates the Claude Agent SDK.
The SDK provides the core agent execution capabilities through a subprocess-based architecture.

## Current Configuration

### Versions

- **claude-agent-sdk:** 0.1.12
- **anthropic:** 0.68.0
- **Python:** 3.10+

### Architecture

The platform uses a two-tier design:

- **Go Backend:** Kubernetes orchestration, API server, authentication
- **Python Runner:** Claude Agent SDK wrapper, AG-UI protocol translation

## SDK Client Lifecycle

### Initialization

The SDK client is created fresh for each run to ensure clean state and avoid subprocess reuse issues.

```python
from claude_agent_sdk import ClaudeSDKClient, ClaudeAgentOptions

options = ClaudeAgentOptions(
    cwd=workspace_path,
    permission_mode="acceptEdits",
    allowed_tools=["Read", "Write", "Bash", "Glob", "Grep", "Edit"],
    mcp_servers=mcp_config,
    system_prompt={"type": "text", "text": prompt_text},
    include_partial_messages=True
)

client = ClaudeSDKClient(options=options)
await client.connect()
```

### Execution Flow

1. Client connects and initializes subprocess
2. Query is sent via `client.query(prompt)`
3. Response stream is consumed via `client.receive_response()`
4. Messages are converted to AG-UI events
5. Client disconnects at end of run

### Message Processing

The SDK emits several message types:

- **AssistantMessage:** Claude's response with content blocks
- **ToolUseBlock:** Tool invocation request
- **ToolResultBlock:** Tool execution result
- **ResultMessage:** Final usage metrics and cost
- **StreamEvent:** Real-time streaming chunks

## Configuration Options

### Working Directory

The `cwd` parameter determines the execution context:

- **Workflow Mode:** `/workspace/workflows/{name}`
- **Multi-Repo Mode:** `/workspace/repos/{main-repo}`
- **Default:** `/workspace/artifacts`

### Permission Mode

Currently set to `acceptEdits` which auto-approves file modifications.
The SDK supports more granular permission controls that we do not currently use.

### Model Configuration

Model selection is dynamic based on environment variables:

```python
model = os.getenv('LLM_MODEL', 'claude-sonnet-4-5@20250929')
options.model = model
```

For Vertex AI, model names are mapped:

```python
'claude-opus-4-5' -> 'claude-opus-4-5@20251101'
'claude-sonnet-4-5' -> 'claude-sonnet-4-5@20250929'
'claude-haiku-4-5' -> 'claude-haiku-4-5@20251001'
```

### Authentication

Two authentication methods are supported:

**Anthropic API:**

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

**Vertex AI:**

```bash
export CLAUDE_CODE_USE_VERTEX=1
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/credentials.json
export ANTHROPIC_VERTEX_PROJECT_ID=your-project
export CLOUD_ML_REGION=us-east5
```

### Conversation Continuation

The SDK persists conversation state to disk in `.claude/` directory.
Sessions resume automatically after the first run:

```python
if not self._first_run:
    options.continue_conversation = True
```

## Tool System

### Built-in Tools

The platform enables these SDK tools:

- **Read:** Read files
- **Write:** Create or overwrite files
- **Bash:** Execute shell commands
- **Glob:** Find files by pattern
- **Grep:** Search file contents
- **Edit:** Modify existing files
- **MultiEdit:** Batch edits across files
- **WebSearch:** Search the web

### MCP Tools

MCP servers extend the SDK with external integrations.
Tools are registered dynamically:

```python
from claude_agent_sdk import tool as sdk_tool, create_sdk_mcp_server

@sdk_tool("restart_session", "Restart the session", {})
async def restart_session_tool(args: dict) -> dict:
    return {"content": [{"type": "text", "text": "Restarting..."}]}

server = create_sdk_mcp_server(
    name="session",
    version="1.0.0",
    tools=[restart_session_tool]
)
mcp_servers["session"] = server
```

MCP servers are loaded from `/app/claude-runner/.mcp.json`:

```json
{
  "mcpServers": {
    "webfetch": {
      "command": "mcp-server-webfetch",
      "args": []
    }
  }
}
```

### Tool Permissions

Tool permissions are managed via `allowed_tools` list:

```python
allowed_tools = ["Read", "Write", "Bash"]
for server_name in mcp_servers:
    allowed_tools.append(f"mcp__{server_name}")
```

## System Prompt

The system prompt provides workspace context to the agent.
It is generated dynamically based on session configuration:

```python
prompt = "# Workspace Structure\n\n"
prompt += f"**Working Directory**: {cwd}\n"
prompt += f"**Artifacts**: artifacts/\n"
prompt += f"**Repositories**: {', '.join(repo_names)}\n"

if workflow_config.get("systemPrompt"):
    prompt += f"\n## Workflow Instructions\n{workflow_config['systemPrompt']}\n"

options.system_prompt = {"type": "text", "text": prompt}
```

## AG-UI Protocol Translation

The adapter converts SDK messages to AG-UI events for the frontend.

### Event Mapping

| SDK Message | AG-UI Event | Purpose |
|-------------|-------------|---------|
| AssistantMessage | TEXT_MESSAGE_START | Begin assistant response |
| TextBlock | TEXT_MESSAGE_CONTENT | Stream text content |
| ToolUseBlock | TOOL_CALL_START, TOOL_CALL_ARGS | Tool invocation |
| ToolResultBlock | TOOL_CALL_END | Tool result or error |
| ResultMessage | STATE_DELTA | Usage metrics, cost |
| StreamEvent | TEXT_MESSAGE_CONTENT | Real-time chunks |

### Streaming Implementation

Real-time streaming is enabled via `include_partial_messages=True`:

```python
async for message in client.receive_response():
    if isinstance(message, StreamEvent):
        event_data = message.event
        if event_data.get('type') == 'content_block_delta':
            delta = event_data.get('delta', {})
            if delta.get('type') == 'text_delta':
                text_chunk = delta.get('text', '')
                # Emit TEXT_MESSAGE_CONTENT event
```

## Observability

### Langfuse Integration

Each conversation turn creates a Langfuse trace:

```python
trace = langfuse.trace(
    name="claude_interaction",
    metadata={"turn": turn_count}
)

generation = trace.generation(
    name="claude_turn",
    model=model,
    input=user_message,
    output=assistant_message
)
```

Tool calls are tracked as spans:

```python
span = trace.span(
    name=f"tool_{tool_name}",
    input=tool_input,
    output=tool_result
)
```

### Privacy Masking

Message content can be masked for privacy:

```bash
export LANGFUSE_MASK_MESSAGES=true
```

When enabled, only metadata is logged:

```python
if mask_messages:
    generation_input = {"masked": True}
    generation_output = {"masked": True}
```

## Error Handling

### Resume Failures

If conversation continuation fails, the adapter starts fresh:

```python
try:
    options.continue_conversation = True
    client = ClaudeSDKClient(options)
    await client.connect()
except Exception as e:
    if "no conversation found" in str(e).lower():
        options.continue_conversation = False
        client = ClaudeSDKClient(options)
        await client.connect()
```

### Interrupt Support

Sessions can be interrupted gracefully:

```python
async def interrupt(self):
    if self._active_client:
        await self._active_client.interrupt()
```

## File Locations

```
components/runners/claude-code-runner/
├── adapter.py              # SDK integration, message handling
├── observability.py        # Langfuse tracking
├── main.py                 # FastAPI server, AG-UI protocol
├── context.py              # Environment configuration
├── security_utils.py       # Input validation, timeouts
└── pyproject.toml          # Dependencies
```

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| ANTHROPIC_API_KEY | Anthropic API authentication | Required |
| CLAUDE_CODE_USE_VERTEX | Enable Vertex AI mode | false |
| LLM_MODEL | Model selection | claude-sonnet-4-5@20250929 |
| LLM_MAX_TOKENS | Maximum output tokens | SDK default |
| LLM_TEMPERATURE | Sampling temperature | SDK default |
| LANGFUSE_ENABLED | Enable observability | false |
| LANGFUSE_MASK_MESSAGES | Privacy masking | true |
| IS_RESUME | Continue conversation | false |

## Known Limitations

### Client Reuse

The Python SDK does not handle client reuse reliably.
We create a fresh client for each run to avoid subprocess issues.

### Vertex AI Authentication

Vertex AI requires environment variables to be set before importing the SDK.
The adapter sets these conditionally based on `CLAUDE_CODE_USE_VERTEX`.

### Streaming Completeness

StreamEvent provides partial chunks, but not all message types support streaming.
TextBlock and ToolResultBlock arrive complete.

## References

- [Claude Agent SDK Documentation](https://github.com/anthropics/claude-agent-sdk-python)
- [AG-UI Protocol Specification](https://github.com/anthropics/ag-ui)
- [Langfuse Python SDK](https://langfuse.com/docs/sdk/python)
