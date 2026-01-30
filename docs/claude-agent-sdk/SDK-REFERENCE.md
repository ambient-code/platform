# Claude Agent SDK Reference

**Version:** 0.1.12 (latest: 0.1.23)
**Last Updated:** 2026-01-27

Complete reference for Claude Agent SDK integration in the Ambient platform.

## Quick Reference

**Current Setup:**
- SDK wraps Claude via subprocess
- Fresh client per run (no reuse)
- State persists to `.claude/` directory
- AG-UI protocol for frontend streaming

**Key Files:**
- `adapter.py` - SDK integration, message handling
- `observability.py` - Langfuse tracking
- `main.py` - AG-UI server

**Common Tasks:**
- [Client lifecycle](#client-lifecycle)
- [Configuration](#configuration-options)
- [Tools and MCP](#tool-system)
- [Debugging](#debugging)
- [Performance](#performance-optimization)

## Architecture

```
Frontend (SSE) ← AG-UI Events ← adapter.py ← ClaudeSDKClient ← Anthropic API
```

**Flow:**
1. Create `ClaudeSDKClient` with options
2. Connect (spawns subprocess)
3. Query with user prompt
4. Stream messages (Assistant, Tool, Result)
5. Convert to AG-UI events
6. Disconnect

## Client Lifecycle

### Standard Pattern

```python
from claude_agent_sdk import ClaudeSDKClient, ClaudeAgentOptions

options = ClaudeAgentOptions(
    cwd=workspace_path,
    permission_mode="acceptEdits",
    allowed_tools=["Read", "Write", "Bash"],
)

client = ClaudeSDKClient(options=options)
try:
    await client.connect()
    await client.query(prompt)

    async for message in client.receive_response():
        # Process messages
        pass
finally:
    await client.disconnect()
```

### Conversation Continuation

```python
# First run
options.continue_conversation = False  # Default
client = ClaudeSDKClient(options)
# ... SDK writes state to .claude/

# Subsequent runs
options.continue_conversation = True
client = ClaudeSDKClient(options)
# ... SDK resumes from .claude/
```

### Resume Failure Recovery

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

## Configuration Options

### ClaudeAgentOptions

```python
options = ClaudeAgentOptions(
    # Working directory
    cwd=str(workspace_path),

    # Permission mode (acceptEdits = auto-approve)
    permission_mode="acceptEdits",

    # Available tools
    allowed_tools=["Read", "Write", "Bash", "Glob", "Grep", "Edit"],

    # MCP server integrations
    mcp_servers={"webfetch": webfetch_server, ...},

    # System prompt (workspace context)
    system_prompt={"type": "text", "text": prompt_text},

    # Streaming support
    include_partial_messages=True,

    # Optional: Continue from disk
    continue_conversation=True,  # Default: False

    # Optional: Model selection
    model="claude-sonnet-4-5@20250929",

    # Optional: Generation limits
    max_tokens=4096,
    temperature=1.0,
)
```

### Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| ANTHROPIC_API_KEY | Required | API authentication |
| CLAUDE_CODE_USE_VERTEX | false | Enable Vertex AI |
| LLM_MODEL | claude-sonnet-4-5 | Model selection |
| LLM_MAX_TOKENS | - | Output limit |
| LLM_TEMPERATURE | - | Sampling temp |
| IS_RESUME | false | Continue conversation |

### Authentication

**Anthropic API:**
```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

**Vertex AI:**
```bash
export CLAUDE_CODE_USE_VERTEX=1
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/creds.json
export ANTHROPIC_VERTEX_PROJECT_ID=project-id
export CLOUD_ML_REGION=us-east5
```

## Message Handling

### Message Types

**AssistantMessage:**
- Contains content blocks (Text, ToolUse, ToolResult)
- Triggers observability turn tracking
- Maps to TEXT_MESSAGE events

**ToolUseBlock:**
- Tool invocation with name + input
- Maps to TOOL_CALL_START + TOOL_CALL_ARGS events

**ToolResultBlock:**
- Tool execution result or error
- Maps to TOOL_CALL_END event

**ResultMessage:**
- Final metrics (usage, cost, turns)
- Maps to STATE_DELTA event

**StreamEvent:**
- Real-time text chunks
- Requires `include_partial_messages=True`
- Maps to TEXT_MESSAGE_CONTENT deltas

### Example Processing

```python
async for message in client.receive_response():
    if isinstance(message, AssistantMessage):
        for block in message.content:
            if isinstance(block, TextBlock):
                # Emit text content
                yield TEXT_MESSAGE_CONTENT(delta=block.text)

            elif isinstance(block, ToolUseBlock):
                # Emit tool call
                yield TOOL_CALL_START(
                    tool_call_id=block.id,
                    tool_call_name=block.name
                )

    elif isinstance(message, ResultMessage):
        # Emit usage stats
        yield STATE_DELTA(usage=message.usage)
```

## Tool System

### Built-in Tools

Enabled via `allowed_tools`:
- **Read** - Read files
- **Write** - Create/overwrite files
- **Bash** - Execute shell commands
- **Glob** - Find files by pattern
- **Grep** - Search file contents
- **Edit** - Modify existing files
- **MultiEdit** - Batch edits
- **WebSearch** - Search web

### MCP Tools

**Loading Configuration:**

```python
# From /app/claude-runner/.mcp.json
mcp_servers = self._load_mcp_config(cwd_path)

# Grant permissions
for server_name in mcp_servers:
    allowed_tools.append(f"mcp__{server_name}")
```

**Example .mcp.json:**

```json
{
  "mcpServers": {
    "webfetch": {
      "command": "mcp-server-webfetch",
      "args": []
    },
    "google-workspace": {
      "command": "mcp-server-google-workspace",
      "env": {
        "CREDENTIALS_PATH": "/workspace/.google/credentials.json"
      }
    }
  }
}
```

### Custom MCP Tools

```python
from claude_agent_sdk import tool as sdk_tool, create_sdk_mcp_server

@sdk_tool("tool_name", "Tool description", {})
async def custom_tool(args: dict) -> dict:
    result = do_work(args)
    return {
        "content": [{
            "type": "text",
            "text": str(result)
        }]
    }

server = create_sdk_mcp_server(
    name="custom_server",
    version="1.0.0",
    tools=[custom_tool]
)

mcp_servers["custom_server"] = server
```

## System Prompt

Dynamic workspace context:

```python
def build_workspace_prompt(repos, workflows, files):
    prompt = "# Workspace Structure\n\n"

    if workflows:
        prompt += f"**Working Directory:** workflows/{workflow_name}/\n\n"

    prompt += f"**Artifacts:** artifacts/ (output files)\n\n"

    if files:
        prompt += f"**Uploaded Files:** {', '.join(files[:10])}\n\n"

    if repos:
        prompt += f"**Repositories:** {', '.join(repo_names)}\n"
        prompt += f"**Working Branch:** ambient/{session_id}\n\n"

    return prompt

options.system_prompt = {"type": "text", "text": build_workspace_prompt(...)}
```

## Debugging

### Common Issues

**Streaming not working:**
- Check `include_partial_messages=True`
- Verify StreamEvent handling
- Ensure `content_block_delta` parsing

**Tools not executing:**
- Verify tool in `allowed_tools`
- Check MCP server config
- Validate permissions (`mcp__{server}`)

**Conversation not resuming:**
- Check `.claude/` directory exists
- Verify `continue_conversation=True`
- Ensure workspace path unchanged

**Client connection timeout:**
- Check API key validity
- Verify network connectivity
- Review rate limiting

### Troubleshooting Checklist

1. **Authentication:**
   - API key set?
   - Vertex credentials valid?

2. **Configuration:**
   - `cwd` exists?
   - `allowed_tools` includes needed tools?
   - MCP servers configured?

3. **State:**
   - `.claude/` writable?
   - State files not corrupted?

4. **Logs:**
   - Check subprocess errors
   - Review permission denials
   - Look for rate limits

## Performance Optimization

### Current Bottlenecks

**Measured:**
- JSON encoding: 18% CPU per session
- SSE framing: 8% overhead
- Network I/O: Saturates at ~1000 concurrent sessions

**Limits:**
- Events per run: 50-200
- Event size: 200-500 bytes
- Stream duration: 5-60 seconds

### Optimization Strategies

#### 1. Event Batching (P0 - High Impact, Low Complexity)

**Problem:** Per-event encoding overhead

**Solution:**

```python
class EventBatcher:
    def __init__(self, batch_size=10, timeout_ms=100):
        self.buffer = []
        self.batch_size = batch_size
        self.timeout = timeout_ms / 1000.0
        self.last_flush = time.time()

    async def add(self, event):
        self.buffer.append(event)
        if len(self.buffer) >= self.batch_size or \
           (time.time() - self.last_flush) >= self.timeout:
            yield self.flush()

    def flush(self):
        batch = {"type": "event_batch", "events": self.buffer}
        self.buffer.clear()
        self.last_flush = time.time()
        return batch
```

**Impact:** +30-40% throughput, -15% CPU

#### 2. Compression (P1 - High Impact, Medium Complexity)

**Problem:** Text-heavy events consume bandwidth

**Solution:**

```python
import gzip

def compress_event(event, threshold=1024):
    encoded = json.dumps(event)

    if len(encoded) < threshold:
        return encoded

    compressed = gzip.compress(encoded.encode())

    if len(compressed) < len(encoded) * 0.8:
        return {
            "compressed": True,
            "data": base64.b64encode(compressed).decode()
        }

    return encoded
```

**Impact:** -30-50% bandwidth

#### 3. Metadata Optimization (P1 - Medium Impact, Low Complexity)

**Problem:** Repeated fields (thread_id, run_id) in every event

**Solution:**

```python
# Send once at session start
yield RawEvent(
    type=EventType.RAW,
    event={
        "type": "session_metadata",
        "thread_id": thread_id,
        "run_id": run_id,
        "trace_id": trace_id,
    }
)

# Subsequent events omit these fields
# Frontend reconstructs from session metadata
```

**Impact:** -5-8% bandwidth

### Combined Impact

Implementing P0 + P1:
- CPU: -25%
- Bandwidth: -45%
- **Capacity: 3-5x more concurrent sessions**

### Monitoring

```python
from prometheus_client import Counter, Histogram

ag_ui_events_total = Counter('ag_ui_events_total', 'Events emitted')
ag_ui_event_bytes = Histogram('ag_ui_event_bytes', 'Event size')
ag_ui_stream_duration = Histogram('ag_ui_stream_duration', 'Stream duration')
```

## Observability

### Langfuse Integration

```python
from observability import ObservabilityManager

obs = ObservabilityManager(
    session_id=session_id,
    user_id=user_id,
)

await obs.initialize(prompt=prompt, model=model)

# Start turn
obs.start_turn(model, user_input=prompt)

# Track tools
obs.track_tool_use(tool_name, tool_id, tool_input)
obs.track_tool_result(tool_id, result, is_error)

# End turn
obs.end_turn(turn_count, message, usage)

# Cleanup
await obs.finalize()
```

### Privacy Masking

```bash
export LANGFUSE_MASK_MESSAGES=true
```

When enabled:
- Input/output replaced with `{"masked": True}`
- Metadata still tracked (model, tokens, cost)
- Tool names logged, not inputs/results

## Security

### Credential Handling

**Never log secrets:**

```python
def _redact_secrets(text):
    text = re.sub(r'sk-ant-[a-zA-Z0-9\-_]{30,200}', 'sk-ant-***', text)
    text = re.sub(r'gh[pousr]_[a-zA-Z0-9]{36,255}', 'gh*_***', text)
    return text
```

**Set environment before import:**

```python
os.environ['ANTHROPIC_API_KEY'] = api_key
from claude_agent_sdk import ClaudeSDKClient  # After env set
```

### Input Validation

**Model names:**

```python
def sanitize_model(model: str) -> str:
    if not re.match(r'^[a-zA-Z0-9\-@.]+$', model):
        raise ValueError("Invalid model name")
    return model
```

**Timeouts:**

```python
from security_utils import with_timeout

result = await with_timeout(
    client.query(prompt),
    timeout_seconds=300,
    operation_name="SDK query"
)
```

## Advanced Features

### Available but Unused

**File Checkpointing (SDK 0.1.15+):**

```python
options.enable_file_checkpointing = True

# Later, rollback changes
client.rewind_files(user_message_id)
```

**MCP Status Monitoring (SDK 0.1.23+):**

```python
status = client.get_mcp_status()
# Returns: {server: {connected: bool, error: str}}
```

**Hooks System:**

```python
from claude_agent_sdk import PreToolUseHookInput, PostToolUseHookInput

async def pre_tool_hook(input: PreToolUseHookInput):
    # Validate, rate limit, track
    return PermissionResultAllow()

async def post_tool_hook(input: PostToolUseHookInput):
    # Log, audit, cleanup
    pass
```

## Testing

See `tests/smoketest/` for validation suite.

**Quick validation:**

```bash
pytest tests/smoketest/ -v -m "not slow"
```

**Full validation:**

```bash
pytest tests/smoketest/ -v
```

## References

- [SDK GitHub](https://github.com/anthropics/claude-agent-sdk-python)
- [AG-UI Protocol](https://github.com/anthropics/ag-ui)
- [Upgrade Guide](UPGRADE-GUIDE.md)
