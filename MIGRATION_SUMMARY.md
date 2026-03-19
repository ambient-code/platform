# AG-UI Claude SDK Migration Summary

## Overview

Replaced custom `ag_ui_claude_sdk` implementation with upstream package from https://github.com/ag-ui-protocol/ag-ui/tree/main/integrations/claude-agent-sdk/python

## Changes Made

### 1. Dependencies (`pyproject.toml`)
- **Added**: `ag-ui-claude-sdk` from GitHub (not yet on PyPI)
  ```toml
  "ag-ui-claude-sdk @ git+https://github.com/ag-ui-protocol/ag-ui.git#subdirectory=integrations/claude-agent-sdk/python"
  ```
- **Removed**: `ag_ui_claude_sdk` from packages list (no longer vendored)

### 2. Removed Custom Implementation
- **Deleted**: `components/runners/ambient-runner/ag_ui_claude_sdk/` directory (~2000 lines)
- **Backed up to**: `/tmp/ag_ui_claude_sdk_backup` (for reference)

### 3. New Simplified Bridge (`bridge_v2.py`)
Created `ClaudeBridgeV2` that uses upstream architecture:

**Simplified responsibilities:**
- Platform setup (auth, MCP, observability) - kept
- Adapter creation with options - kept
- Worker management - **delegated to upstream adapter**
- Message streaming - **delegated to upstream adapter**
- Tracing middleware - kept

**Removed complexity:**
- Custom `SessionManager` - upstream handles this
- Custom `SessionWorker` - upstream handles this
- Manual session ID persistence - upstream uses `resume` via forwarded_props
- Direct message_stream parameter - upstream manages internally

### 4. Mock Client Support (`mock_patch.py`)
Added monkey-patch module for testing:
- Detects `ANTHROPIC_API_KEY=mock-replay-key`
- Patches `claude_agent_sdk.ClaudeSDKClient` → `MockClaudeSDKClient`
- Auto-applies when mock API key detected
- Preserves existing test infrastructure

### 5. Updated Exports
`ambient_runner/bridges/claude/__init__.py` now exports:
- `ClaudeBridge` (original, still works)
- `ClaudeBridgeV2` (new, uses upstream)

## Architecture Comparison

### Before (Custom)
```
ClaudeBridge
  ├─> SessionManager (manages worker pool)
  │     └─> SessionWorker (owns ClaudeSDKClient)
  │           ├─> query() → message_stream
  │           └─> MockClaudeSDKClient (test mode)
  └─> ClaudeAgentAdapter (custom)
        └─> run(input_data, message_stream)  # Passive translator
```

### After (Upstream)
```
ClaudeBridgeV2
  └─> ClaudeAgentAdapter (upstream)
        ├─> SessionWorker pool (built-in)
        │     └─> ClaudeSDKClient (patched for mock)
        └─> run(input_data)  # Active - manages workers
```

## Key Differences

| Feature | Custom | Upstream |
|---------|--------|----------|
| Worker management | External (SessionManager) | Internal (adapter) |
| Session persistence | Manual disk writes | `resume` forwarded prop |
| Mock testing | Native SessionWorker hook | Monkey-patch approach |
| API surface | `run(input_data, message_stream)` | `run(input_data)` |
| Maintenance | ~2000 LOC to maintain | Upstream dependency |
| Updates | Manual sync | `pip install --upgrade` |

## Testing Status

- ✅ Syntax valid (Python compiler)
- ⏳ Unit tests - pending
- ⏳ Integration tests - pending
- ⏳ E2E tests - pending

## Next Steps

1. **Test imports** - Verify upstream package installs correctly from GitHub
2. **Run tests** - `pytest tests/test_bridge_claude.py` (expect failures)
3. **Fix failures** - Adapt tests to new architecture
4. **Switch default** - Replace `ClaudeBridge` with `ClaudeBridgeV2` when stable
5. **Remove old code** - Delete `bridge.py`, `session.py` once migration complete

## Rollback Plan

If migration fails:
1. Restore `/tmp/ag_ui_claude_sdk_backup` → `ag_ui_claude_sdk/`
2. Revert `pyproject.toml` changes
3. Remove `bridge_v2.py` and `mock_patch.py`
4. Keep using `ClaudeBridge` (original)

## Open Questions

1. **MCP Status endpoint** - Not yet ported to v2 (returns stub)
2. **Graceful shutdown** - Does upstream preserve session state correctly?
3. **Pod restart resume** - Does upstream `resume` work the same as our manual approach?
4. **Performance** - Worker TTL and eviction vs. our persistent workers?

## Compatibility Notes

- Upstream requires `ag-ui-protocol>=0.1.0` (we use >=0.1.13, compatible)
- Upstream requires `claude-agent-sdk>=0.1.12` (we use >=0.1.23, compatible)
- Reasoning events now in `ag_ui.core` (our custom `reasoning_events.py` no longer needed)
