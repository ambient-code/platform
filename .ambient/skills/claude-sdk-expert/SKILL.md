# Claude SDK Expert Skill

**Version:** 1.0.0
**Purpose:** Expert guidance for working with Claude Agent SDK integration

## When to Use

Invoke when working on:
- SDK integration code (`components/runners/claude-code-runner/`)
- SDK upgrades or configuration changes
- Debugging SDK issues (streaming, tools, MCP)
- Performance optimization

## Quick Patterns

### Client Lifecycle

```python
from claude_agent_sdk import ClaudeSDKClient, ClaudeAgentOptions

options = ClaudeAgentOptions(
    cwd=workspace_path,
    permission_mode="acceptEdits",
    allowed_tools=["Read", "Write", "Bash"],
    mcp_servers=mcp_config,
    include_partial_messages=True,
)

client = ClaudeSDKClient(options=options)
try:
    await client.connect()
    await client.query(prompt)
    async for message in client.receive_response():
        # Process messages
finally:
    await client.disconnect()
```

### Message Handling

```python
if isinstance(message, AssistantMessage):
    for block in message.content:
        if isinstance(block, TextBlock):
            yield TEXT_MESSAGE_CONTENT(delta=block.text)
        elif isinstance(block, ToolUseBlock):
            yield TOOL_CALL_START(
                tool_call_id=block.id,
                tool_call_name=block.name
            )
        elif isinstance(block, ToolResultBlock):
            yield TOOL_CALL_END(
                tool_call_id=block.tool_use_id,
                result=block.content
            )

elif isinstance(message, ResultMessage):
    yield STATE_DELTA(usage=message.usage)
```

### Adding MCP Tools

```python
from claude_agent_sdk import tool as sdk_tool, create_sdk_mcp_server

@sdk_tool("tool_name", "Description", {})
async def custom_tool(args: dict) -> dict:
    return {"content": [{"type": "text", "text": "result"}]}

server = create_sdk_mcp_server(
    name="server_name",
    version="1.0.0",
    tools=[custom_tool]
)

mcp_servers["server_name"] = server
allowed_tools.append("mcp__server_name")
```

## Configuration Reference

### Required Environment Variables

| Variable | Purpose | Example |
|----------|---------|---------|
| ANTHROPIC_API_KEY | API auth | sk-ant-... |
| CLAUDE_CODE_USE_VERTEX | Vertex mode | 1 |
| LLM_MODEL | Model selection | claude-sonnet-4-5 |

### ClaudeAgentOptions Key Fields

| Field | Default | Purpose |
|-------|---------|---------|
| cwd | Required | Working directory |
| permission_mode | - | "acceptEdits" for auto-approve |
| allowed_tools | [] | Available tools list |
| mcp_servers | {} | MCP integrations |
| include_partial_messages | False | Enable streaming |
| continue_conversation | False | Resume from disk |

## Common Tasks

### Upgrading SDK

**Process:**
1. Review `docs/claude-agent-sdk/UPGRADE-GUIDE.md`
2. Update `pyproject.toml` versions
3. Run `uv sync`
4. Execute `pytest tests/smoketest/ -v`
5. Deploy to staging

**Current:** 0.1.12 → **Latest:** 0.1.23 (safe, no breaking changes)

### Debugging Streaming

**Check:**
- `include_partial_messages=True` in options
- StreamEvent handling in message loop
- `content_block_delta` event parsing

**Pattern:**

```python
if isinstance(message, StreamEvent):
    event_data = message.event
    if event_data.get('type') == 'content_block_delta':
        delta = event_data.get('delta', {})
        if delta.get('type') == 'text_delta':
            text = delta.get('text', '')
            # Emit chunk
```

### Debugging Tool Execution

**Check:**
- Tool in `allowed_tools` list
- MCP server in `.mcp.json`
- Permission granted (`mcp__{server_name}`)

**Validation:**

```python
logger.info(f"Allowed tools: {allowed_tools}")
logger.info(f"MCP servers: {list(mcp_servers.keys())}")

for block in message.content:
    if isinstance(block, ToolUseBlock):
        logger.info(f"Tool invoked: {block.name}")
```

### Debugging Resume Failures

**Check:**
- `.claude/` directory exists
- `continue_conversation=True` set
- Workspace path unchanged

**Pattern:**

```python
try:
    options.continue_conversation = True
    client = ClaudeSDKClient(options)
    await client.connect()
except Exception as e:
    if "no conversation found" in str(e).lower():
        # Start fresh
        options.continue_conversation = False
        client = ClaudeSDKClient(options)
        await client.connect()
```

## Troubleshooting Checklist

When SDK fails:

1. **Authentication:**
   - API key set?
   - Vertex credentials valid?

2. **Configuration:**
   - `cwd` exists?
   - Tools in `allowed_tools`?
   - MCP servers configured?

3. **State:**
   - `.claude/` writable?
   - State not corrupted?

4. **Logs:**
   - Subprocess errors?
   - Permission denials?
   - Rate limits?

## File Locations

**Integration:**
- `adapter.py` - SDK integration (lines 280-803)
- `observability.py` - Langfuse tracking
- `main.py` - AG-UI server

**Configuration:**
- `/app/claude-runner/.mcp.json` - MCP servers
- `pyproject.toml` - Dependencies

**Documentation:**
- `docs/claude-agent-sdk/SDK-REFERENCE.md` - Complete reference
- `docs/claude-agent-sdk/UPGRADE-GUIDE.md` - Upgrade planning
- `tests/smoketest/` - Validation tests

## Performance Optimization

**Top 3 Recommendations:**

1. **Event Batching (P0):** Buffer 10 events or 100ms timeout → +30-40% throughput
2. **Compression (P1):** Gzip events >1KB → -30-50% bandwidth
3. **Metadata Optimization (P1):** Send session metadata once → -5-8% bandwidth

**Combined Impact:** 3-5x more concurrent sessions

See `docs/claude-agent-sdk/SDK-REFERENCE.md#performance-optimization` for implementation.

## Security Notes

**Credential Handling:**
- Set env vars before importing SDK
- Never log API keys (use `_redact_secrets()`)

**Input Validation:**
- Sanitize model names (alphanumeric, hyphens, @ only)
- Enforce timeouts on async operations

## Available Features

**Currently Used:**
- Client lifecycle
- Message streaming
- Tool execution
- MCP integration
- Conversation continuation

**Available but Unused:**
- File checkpointing (SDK 0.1.15+)
- MCP status monitoring (SDK 0.1.23+)
- Hooks system
- Advanced permissions

## Answering User Questions

### "Is it safe to upgrade?"

**Response:**
```
Yes, upgrading from 0.1.12 to 0.1.23 is safe.

Security: No CVEs found
Breaking changes: None
Risk: Very Low
Effort: Trivial (dependency update)

See docs/claude-agent-sdk/UPGRADE-GUIDE.md for steps.
```

### "How do I add an MCP server?"

**Response:**
```
1. Add to /app/claude-runner/.mcp.json:
   {"mcpServers": {"server_name": {"command": "...", "args": []}}}

2. Grant permissions in adapter.py:
   allowed_tools.append("mcp__server_name")

3. Test connectivity:
   status = client.get_mcp_status()  # SDK 0.1.23+

See docs/claude-agent-sdk/SDK-REFERENCE.md#mcp-tools for examples.
```

### "Why isn't streaming working?"

**Response:**
```
Check these items:

1. include_partial_messages=True in ClaudeAgentOptions
2. StreamEvent handling in message loop
3. content_block_delta event parsing

See docs/claude-agent-sdk/SDK-REFERENCE.md#debugging for patterns.
```

### "How can we improve performance?"

**Response:**
```
Three high-impact optimizations:

P0 - Event batching: +30-40% throughput
P1 - Compression: -30-50% bandwidth
P1 - Metadata optimization: -5-8% bandwidth

Combined: Enables 3-5x more concurrent sessions

See docs/claude-agent-sdk/SDK-REFERENCE.md#performance-optimization.
```

## Response Standards

**Always Provide:**
1. Direct answer to question
2. File reference for details
3. Code example when applicable
4. Next steps

**Never:**
1. Invent information
2. Claim features without verification
3. Suggest changes without impact assessment
4. Skip source documentation

## Documentation Links

For detailed information, always reference:

- **SDK-REFERENCE.md** - Integration, configuration, debugging, performance
- **UPGRADE-GUIDE.md** - Version analysis, security, migration
- **tests/smoketest/** - Validation and test patterns

These documents are comprehensive. Reference them instead of duplicating content.
