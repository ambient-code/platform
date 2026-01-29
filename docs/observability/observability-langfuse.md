# Observability with Langfuse

## Overview

The Ambient Code Platform uses **Langfuse** for LLM-specific observability and cost tracking. Langfuse provides detailed insights into Claude interactions, token usage, and tool executions across all agentic sessions.

**Key capabilities:**

- Turn-level generations with accurate token and cost tracking
- Tool execution visibility without cost inflation
- Session grouping and multi-user cost allocation
- Real-time trace streaming with async flush

**OpenTelemetry Compatibility**: Langfuse v3 is built on OpenTelemetry standards. While we use the native Langfuse SDK for simplicity, the platform can integrate with any OTEL-compatible observability backend if desired.

## Architecture

### Trace Structure

Our instrumentation creates a **flat trace hierarchy** where each Claude turn is a top-level trace:

```
claude_turn_1 (trace)
  ├─ input: user prompt
  ├─ output: assistant response
  ├─ usage: tokens + cost
  └─ tool_Read, tool_Write (child spans for visibility)

claude_turn_2 (trace)
  ├─ input: follow-up prompt
  ├─ output: assistant response
  ├─ usage: tokens + cost
  └─ tool_Bash (child span)

Grouped by: session_id via propagate_attributes
```

**Design rationale:**

- **Turns as traces**: Each `claude_turn_X` is a top-level trace (not nested under a session trace)
- **Session grouping**: `propagate_attributes()` groups related traces by `session_id`, `user_id`, and tags
- **Tool visibility**: Tool spans provide execution details without duplicate token counting
- **Real-time streaming**: Explicit flush after each turn for immediate UI visibility

### Async Streaming Architecture

The Claude SDK processes responses as an **async generator** that yields messages across multiple loop iterations:

```python
async for message in client.receive_response():
    # Messages arrive sequentially:
    # 1. AssistantMessage → start turn
    # 2. ToolUseBlock(s) → track tool spans
    # 3. ToolResultBlock(s) → update tool results
    # 4. ResultMessage → close turn with usage data
```

**Manual context management required**: Python's `with` statement cannot maintain state across async iterations, so we manually call `__enter__()` and `__exit__()` on Langfuse contexts.

## Session Context Propagation

All traces inherit consistent metadata via `propagate_attributes()`:

- **user_id**: For cost allocation and multi-user tracking
- **session_id**: Kubernetes AgenticSession name for grouping
- **tags**: `["claude-code", "namespace:X", "model:Y"]`
- **metadata**:
  - `namespace`: Project namespace
  - `user_name`: User display name
  - `model`: Specific model used (e.g., `claude-sonnet-4-5@20250929`)
  - `initial_prompt`: First 200 chars of session prompt

## Turn-Level Generations

**Name**: `claude_turn_X` (sequential: 1, 2, 3...)

- **Type**: Generation with usage tracking
- **Input**: Turn prompt (user message or continuation)
- **Output**: Complete Claude response
- **Model**: Propagated from session context
- **Usage**: Canonical format for accurate cost calculation
  - `input`: Regular input tokens
  - `output`: Output tokens
  - `cache_read_input_tokens`: Cache hits (90% discount)
  - `cache_creation_input_tokens`: Cache writes (25% premium)
  - See [Model Pricing](model-pricing.md) for complete pricing details

**Turn counting**: Uses SDK's authoritative `num_turns` field from `ResultMessage` to ensure accuracy.

**Deferred creation**: Turn 1 is not created until the first `AssistantMessage` arrives, ensuring traces have real user input (not synthetic prompts).

## Tool Spans

**Name**: `tool_{ToolName}` (e.g., `tool_Read`, `tool_Write`, `tool_Bash`)

- **Type**: Span (no usage tracking)
- **Purpose**: Execution visibility only
- **Input**: Full tool parameters
- **Output**: Tool results (truncated to 500 chars for large outputs)
- **No tokens**: Local operations already counted in parent turn

## Implementation Highlights

### Key Technical Decisions

1. **Flat trace hierarchy**: Turns as top-level traces (not observations under session trace)
2. **Manual context management**: Required for async streaming architecture
3. **Real-time flush**: Explicit `langfuse_client.flush()` after each turn completes
4. **Deferred turn creation**: Store initial prompt, create turn when interaction actually begins
5. **Authoritative turn counting**: Use SDK's `num_turns` field (not manual increment)
6. **Clean error handling**: Trust Langfuse SDK for incomplete traces (no synthetic error messages)

### Why Not Python ContextManagers?

Standard `with` statements cannot maintain state across async loop iterations:

```python
# ❌ This doesn't work - context closes at end of iteration
async for message in stream:
    with langfuse.start_as_current_observation() as turn:
        process(message)
    # Context already closed, but we need it for next message!

# ✅ Manual context management - stays open across iterations
def start_turn():
    self._ctx = langfuse.start_as_current_observation(...)
    self._generation = self._ctx.__enter__()  # Manually enter

def end_turn():
    self._ctx.__exit__(None, None, None)  # Manually exit when ready
```

### Deployment

Deploy Langfuse to your cluster using the provided script:

```bash
# Auto-detect platform (OpenShift or Kubernetes)
./e2e/scripts/deploy-langfuse.sh

# Or specify explicitly
./e2e/scripts/deploy-langfuse.sh --openshift
./e2e/scripts/deploy-langfuse.sh --kubernetes
```

The script handles:
- Platform detection (OpenShift vs Kubernetes)
- Helm chart installation
- PostgreSQL database setup
- Ingress/Route configuration
- Namespace creation

### Configuration

Langfuse is configured platform-wide via the `ambient-admin-langfuse-secret` secret in the operator namespace. See deployment documentation for setup details.

## Benefits

**Why Langfuse for LLM observability:**

- **LLM-optimized**: Purpose-built for prompt/response tracking and cost analysis
- **Simple setup**: Only API keys required, no additional infrastructure
- **Cost tracking**: Automatic token and cost calculation for Claude API usage
- **Rich insights**: Full tool I/O, generation content, and performance metrics
- **Multi-user support**: Track usage by user_id for cost allocation
- **OTEL compatible**: Can migrate to any OTEL backend if requirements change

## References

- **Implementation**: `components/runners/claude-code-runner/observability.py`
- **Integration**: `components/runners/claude-code-runner/wrapper.py`
- **Langfuse Docs**: https://langfuse.com/docs
- **Python SDK v3**: https://langfuse.com/docs/sdk/python
