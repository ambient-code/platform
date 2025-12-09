# Claude Agent SDK Patches

This directory contains patches applied to the `claude-agent-sdk` package to ensure compliance with ADR-0006 (Agent Injection Architecture).

## Bash Tool Removal (ADR-0006)

**File**: `patch_sdk_remove_bash.py`

### Problem

Per ADR-0006, all command execution must go through the AgenticTask CRD, not the native Bash tool. While the SDK supports `allowed_tools` and `disallowed_tools` parameters, these may only control which tools can be *used*, not which tools appear in the initialization message sent to Claude.

When the SDK initializes, it sends a SystemMessage with `subtype='init'` that lists all available tools. If Bash appears in this list, Claude may attempt to use it regardless of the `allowed_tools` configuration.

### Solution

This patch implements a **belt-and-suspenders approach** with three layers of defense:

1. **Allowed Tools List** (`wrapper.py:410`): Bash is NOT included in the `allowed_tools` list
2. **Disallowed Tools List** (`wrapper.py:415`): Bash is explicitly added to `disallowed_tools`
3. **Runtime Monkey Patch** (`patch_sdk_remove_bash.py`): Intercepts SDK initialization to filter Bash from the tools list before it reaches Claude

### How It Works

#### Phase 1: Build-time Patching

The `patch_sdk_remove_bash.py` script runs during Docker image build (see `Dockerfile:44`):

```dockerfile
RUN pip install --no-cache-dir /app/claude-runner[observability] \
    && pip install --no-cache-dir aiofiles mcp \
    && python3 /app/claude-runner/patches/patch_sdk_remove_bash.py
```

This script:
1. Locates the installed `claude-agent-sdk` package
2. Creates a runtime wrapper module (`_bash_filter_wrapper.py`) in the SDK directory
3. This wrapper monkey-patches the SDK to filter Bash from tool lists

#### Phase 2: Runtime Import

Before importing the SDK, `wrapper.py:258-262` imports the filter:

```python
try:
    import _bash_filter_wrapper
    logging.info("Bash filter wrapper loaded - Bash tool will be excluded from init message")
except ImportError as e:
    logging.warning(f"Bash filter wrapper not available: {e} - relying on allowed_tools filtering")
```

The wrapper patches:
- `SystemMessage.__init__()`: Filters Bash from `data['tools']` if present
- Any SDK methods that construct tool lists

#### Phase 3: SDK Configuration

The SDK is configured with both positive (allowed) and negative (disallowed) tool lists:

```python
allowed_tools = ["Read", "Write", "Glob", "Grep", "Edit", "MultiEdit", "WebSearch", "WebFetch"]
disallowed_tools = ["Bash"]

options = ClaudeAgentOptions(
    allowed_tools=allowed_tools,
    disallowed_tools=disallowed_tools,
    ...
)
```

### Verification

To verify Bash is not in the init message, check the logs when a session starts:

```bash
# Look for the SDK init message in pod logs
kubectl logs -n <namespace> <claude-runner-pod> | grep -A 20 "ClaudeSDKClient.*init"
```

Expected output should show tools list WITHOUT "Bash":

```json
{
  "type": "system",
  "subtype": "init",
  "data": {
    "tools": ["Read", "Write", "Glob", "Grep", "Edit", "MultiEdit", "WebSearch", "WebFetch", "mcp__agentictask"],
    ...
  }
}
```

### Troubleshooting

#### Patch fails during build

If the patch script fails to locate the SDK files:

```
[PATCH] ERROR: Cannot locate client.py - patch failed
```

This is non-fatal. The runtime wrapper (`_bash_filter_wrapper.py`) will still be created and imported.

#### Runtime wrapper not imported

If you see this warning:

```
Bash filter wrapper not available: <error> - relying on allowed_tools filtering
```

The monkey patch failed to load. The fallback is the `allowed_tools` / `disallowed_tools` configuration, which should still prevent Bash usage (but may not remove it from the init message).

#### Bash still appears in init message

If Bash appears in the init message despite all filtering:

1. Check that `disallowed_tools = ["Bash"]` is set (wrapper.py:415)
2. Check that "Bash" is NOT in `allowed_tools` (wrapper.py:410)
3. Verify the runtime wrapper was imported successfully (check logs)
4. The SDK version may have changed its tool filtering logic - update the patch script

### Maintenance

When upgrading `claude-agent-sdk`, review and update:

1. `patch_sdk_remove_bash.py`: Update file paths if SDK structure changes
2. `_bash_filter_wrapper.py`: Update monkey patches if SDK API changes
3. Test that Bash does NOT appear in init message logs

### Alternative Approaches Considered

**Option B from ADR-0006**: Use PreToolUse hook to intercept Bash calls

```python
async def intercept_bash(input_data, tool_use_id, context):
    if input_data["tool_name"] == "Bash":
        # Redirect to AgenticTask
        return {"permissionDecision": "deny", ...}
```

**Why rejected**: This still allows Bash to appear in the init message, which means Claude will attempt to use it and receive permission errors. Our approach prevents Claude from seeing the tool at all, which is cleaner.

## Related

- **ADR-0006**: Agent Injection Architecture (`docs/adr/0006-agent-injection-architecture.md`)
- **AgenticTask MCP**: Command execution tool (`mcp-servers/agentictask.py`)
- **Wrapper Configuration**: Tool filtering (`wrapper.py:407-449`)
