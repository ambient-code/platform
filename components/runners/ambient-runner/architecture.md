# Ambient Runner: Architecture

## Overview

The runner is a FastAPI server running in a Kubernetes Job pod (one pod per session). It implements the [AG-UI protocol](https://github.com/ag-ui-protocol/ag-ui) — a Server-Sent Events (SSE) streaming protocol for AI agents. The runner bridges between the platform backend and the underlying AI model (Claude Agent SDK).

There are two delivery modes. The **HTTP path** is the original design: the backend POSTs to `/agui/run` and streams AG-UI events back over SSE. The **gRPC path** is an additive overlay that replaces the HTTP round-trip with a persistent bidirectional gRPC channel to the Ambient control plane. Both paths share the same `bridge.run()` execution primitive — only the delivery mechanism differs.

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                          Kubernetes Job Pod (one per session)                       │
│                                                                                     │
│  ENV: SESSION_ID, WORKSPACE_PATH, INITIAL_PROMPT                                    │
│  ENV: AMBIENT_GRPC_ENABLED=true, AMBIENT_GRPC_URL=...   ← only in gRPC mode        │
│                                                                                     │
│  ┌─────────────────────────────────────────────────────────────────────────────┐    │
│  │                         FastAPI app (create_ambient_app)                    │    │
│  │                                                                             │    │
│  │  lifespan startup:                                                          │    │
│  │    1. build RunnerContext                                                   │    │
│  │    2. bridge.set_context(ctx)                                               │    │
│  │    3. if GRPC_ENABLED → bridge.start_grpc_listener(url)  ← gRPC only       │    │
│  │       └── await listener.ready (10s timeout)                                │    │
│  │    4. asyncio.create_task(_auto_execute_initial_prompt)                     │    │
│  │       └── if grpc_url → _push_initial_prompt_via_grpc    ← gRPC only       │    │
│  │           else        → _push_initial_prompt_via_http    ← HTTP path        │    │
│  │                                                                             │    │
│  │  ┌───────────────────────────────────────────────────────────────────────┐ │    │
│  │  │                       ClaudeBridge                                    │ │    │
│  │  │                                                                       │ │    │
│  │  │  _active_streams: dict[thread_id → asyncio.Queue]   ← gRPC only      │ │    │
│  │  │  _grpc_listener:  GRPCSessionListener | None        ← gRPC only      │ │    │
│  │  │                                                                       │ │    │
│  │  │  run(input_data) → AsyncIterator[BaseEvent]         ← shared by both  │ │    │
│  │  └───────────────────────────────────────────────────────────────────────┘ │    │
│  │                                                                             │    │
│  │  HTTP endpoints (existing, always active):                                  │    │
│  │    POST /run        → bridge.run() → SSE to caller                         │    │
│  │    POST /interrupt  → bridge.interrupt()                                    │    │
│  │    GET  /capabilities, /mcp-status, /repos, /workflow, ...                 │    │
│  │                                                                             │    │
│  │  SSE tap endpoints (new, always mounted, only useful in gRPC mode):        │    │
│  │    GET /events/{thread_id}       → SSE tap (real-time)                     │    │
│  │    GET /events/{thread_id}/wait  → SSE tap (polling fallback)              │    │
│  └─────────────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

---

## Startup and Lifecycle (`app.py`, `main.py`)

1. **`main.py`** reads `RUNNER_TYPE` (e.g. `claude-agent-sdk`) and instantiates the bridge.
2. **`create_ambient_app(bridge)`** creates the FastAPI app with a lifespan context manager:
   - Builds `RunnerContext` from `SESSION_ID` / `WORKSPACE_PATH` env vars
   - Calls `bridge.set_context(context)`
   - If `AMBIENT_GRPC_ENABLED=true` and `AMBIENT_GRPC_URL` are set: calls `bridge.start_grpc_listener(url)` and awaits `listener.ready` (10s timeout) before proceeding — ensures the watch stream is open before the initial prompt fires
   - If `IS_RESUME` is not set and a prompt exists: fires `_auto_execute_initial_prompt()` as a background `asyncio.Task`
   - On shutdown: calls `bridge.shutdown()`
3. **Auto-prompt** (`_auto_execute_initial_prompt`):
   - **gRPC mode**: calls `_push_initial_prompt_via_grpc()` — pushes a `PushSessionMessage(event_type="user")` to the control plane; the listener picks it up and drives `bridge.run()` directly
   - **HTTP mode**: calls `_push_initial_prompt_via_http()` — POSTs to `BACKEND_API_URL/projects/{project}/agentic-sessions/{session}/agui/run` with exponential backoff (8 retries, 2s→30s) because K8s DNS may not propagate before the pod is ready

---

## The Bridge Pattern (`bridge.py`)

`PlatformBridge` is an abstract base class. All framework implementations must provide:

- `capabilities()` — declares features to the frontend
- `run(input_data)` — async generator yielding AG-UI `BaseEvent` objects
- `interrupt(thread_id)` — stops the current run

Key lifecycle hooks (override as needed):

- `set_context()` — stores `RunnerContext` at startup
- `_ensure_ready()` / `_setup_platform()` — lazy one-time init on first `run()`
- `_refresh_credentials_if_stale()` — refreshes tokens every 60s or when GitHub token is expiring
- `shutdown()` — called on pod termination
- `mark_dirty()` — called by repos/workflow endpoints when workspace changes; rebuilds adapter on next `run()`
- `inject_message()` — raises `NotImplementedError` on base class; must be overridden by any bridge that handles inbound messages

---

## The Two Delivery Paths

### HTTP Path (original)

The backend owns the entire request lifecycle. It POSTs to the runner and receives AG-UI events back over SSE. The runner never initiates contact.

```
  Frontend
      │  HTTP
      ▼
  Backend
      │  POST /projects/{proj}/agentic-sessions/{sess}/agui/run
      ▼
  POST /run endpoint (runner)
      │  bridge.run(input_data)
      ▼
  ClaudeBridge.run()
      │  yields AG-UI events
      ▼
  SSE stream ──────────────────────────────────────────► Backend
                                                          │
                                                          └── writes result to DB
```

### gRPC Path (additive overlay)

The control plane owns message delivery. The runner maintains a persistent outbound watch stream, and the control plane pushes messages into it. The runner calls `bridge.run()` internally, then fans events out through two parallel channels: an SSE tap for the backend to observe, and a `PushSessionMessage` to persist the assembled result.

```
  ┌─────────────────────┐        ┌────────────────────────────────────────────────┐
  │  Ambient Control    │        │  Pod                                           │
  │  Plane (gRPC)       │        │                                                │
  │                     │        │  ┌──────────────────────────────────────────┐  │
  │  WatchSessionMsgs   │◄───────┤  │ GRPCSessionListener (background thread)  │  │
  │  stream             │ watch  │  │   ThreadPoolExecutor                     │  │
  │                     │        │  │   blocks on gRPC stream                  │  │
  │                     │        │  │   sets listener.ready on stream open     │  │
  │                     │        │  └──────────────────────────────────────────┘  │
  │                     │        │                   │                             │
  │  PushSessionMessage │        │      event_type=="user" received               │
  │  (user message)     │───────►│      parse payload → RunnerInput               │
  │                     │        │      build RunAgentInput                        │
  │                     │        │                   │                             │
  │                     │        │      bridge.run(input_data)                     │
  │                     │        │           │                                     │
  │                     │        │           ├──► active_streams[thread_id]        │
  │                     │        │           │    asyncio.Queue.put_nowait()       │
  │                     │        │           │         │                           │
  │                     │        │           │         ▼                           │
  │                     │        │           │    GET /events/{thread_id}          │
  │                     │        │           │    SSE ──────────────► Backend      │
  │                     │        │           │                                     │
  │                     │        │           └──► GRPCMessageWriter.consume()      │
  │                     │        │                accumulates MESSAGES_SNAPSHOT    │
  │                     │        │                on RUN_FINISHED / RUN_ERROR:     │
  │                     │        │                                                 │
  │  PushSessionMessage │◄───────┤    PushSessionMessage(event_type="assistant")   │
  │  (assistant result) │        │    run_in_executor (non-blocking)               │
  │                     │        │    payload: {run_id, status, messages}          │
  └─────────────────────┘        └────────────────────────────────────────────────┘
```

---

## SSE Queue Lifecycle and the Ordering Contract

The key design decision in the gRPC path is the **ordering contract**: the backend must open `GET /events/{thread_id}` *before* it sends the user message via `PushSessionMessage`. Pre-registration eliminates the race — the queue exists in `active_streams` before the first event can arrive.

```
  Backend                     GET /events endpoint        GRPCSessionListener
     │                               │                           │
     │  GET /events/{thread_id}      │                           │
     │──────────────────────────────►│                           │
     │                               │  queue = existing or new  │
     │                               │  active_streams[id] = q   │
     │                               │                           │
     │                    (control plane delivers user message)   │
     │                               │                    bridge.run() starts
     │                               │                    events → q.put_nowait()
     │◄── SSE chunk ─────────────────│◄── q.get() ───────────────│
     │◄── SSE chunk ─────────────────│◄── q.get() ───────────────│
     │◄── RUN_FINISHED ──────────────│◄── q.get() ───────────────│
     │                               │  break (stream closes)    │
     │                               │  if q is active_streams[id]: pop
     │                               │                           │ finally:
     │                               │                           │ if registered_q
     │                               │                           │   is active_q:
     │                               │                           │     pop
```

**Identity-safe cleanup:** both the SSE endpoint and `GRPCSessionListener` capture the queue reference at the start of their respective lifetimes and only remove it from `active_streams` if the map still points to the same object. This prevents a reconnecting client or a new turn from having its queue silently removed by an older cleanup.

**Duplicate connect:** if a client connects to `/events/{thread_id}` when a queue is already registered (e.g. reconnect), the endpoint reuses the existing queue rather than replacing it. This prevents buffered events from being dropped.

---

## ClaudeBridge: The Full Claude Lifecycle (`bridges/claude/bridge.py`)

`ClaudeBridge` is the complete bridge implementation. Its `run()` method:

1. **`_ensure_ready()`** — on first call, runs `_setup_platform()`:
   - Auth setup (Anthropic API key or Vertex AI credentials)
   - `populate_runtime_credentials()` / `populate_mcp_server_credentials()` — fetches GitHub tokens, Google OAuth, Jira tokens from the backend
   - `resolve_workspace_paths()` — determines cwd and additional dirs
   - `build_mcp_servers()` — assembles full MCP server config (external + platform tools)
   - `build_sdk_system_prompt()` — builds the system prompt
   - Initializes `ObservabilityManager` (Langfuse)
   - Creates `SessionManager`
2. **`_ensure_adapter()`** — builds `ClaudeAgentAdapter` with all options (cwd, permission mode, allowed tools, MCP servers, system prompt). Adapter is cached and reused. A ring buffer of 50 stderr lines is maintained for error reporting.
3. **Worker selection** — gets or creates a `SessionWorker` for the thread, optionally resuming from a previously saved CLI session ID (for pod restarts).
4. **Event streaming** — acquires a per-thread `asyncio.Lock` (prevents concurrent requests to the same thread from mixing), calls `worker.query(user_msg)`, wraps the stream through `tracing_middleware`, and yields events.
5. **Halt detection** — after the stream ends, checks `adapter.halted`. If the adapter halted (because Claude called a frontend HITL tool like `AskUserQuestion`), calls `worker.interrupt()` to prevent the SDK from auto-approving the tool call.
6. **Session persistence** — after each turn, saves the CLI session ID to disk (`claude_session_ids.json`) so `--resume` works after pod restart.

**gRPC listener** (`start_grpc_listener`): a dedicated startup hook (separate from `_setup_platform`) that instantiates and starts `GRPCSessionListener`. Only called when both `AMBIENT_GRPC_ENABLED=true` and `AMBIENT_GRPC_URL` are set. The listener is started before the initial prompt fires so the watch stream is open before the first message arrives. Duplicate calls are idempotent.

---

## SessionWorker and Queue Architecture (`bridges/claude/session.py`)

This is the mechanism that lets the long-lived Claude CLI process work inside FastAPI's async event loop:

```
  Request Handler (async context A)          Background Task (async context B)
          │                                           │
     worker.query(prompt)                      worker._run() loop
          │                                           │
    puts (prompt, session_id,           ◄── input_queue.get()
          output_queue) on input_queue               │
          │                               client.query(prompt)
     output_queue.get() in loop          async for msg in client.receive_response()
          │                                    output_queue.put(msg)
          ▼                                    ...
    yields messages                           output_queue.put(None)  ← sentinel
```

**Why this exists:** the Claude Agent SDK uses `anyio` task groups internally. Using a persistent `ClaudeSDKClient` inside a FastAPI SSE handler (a different async context) hits anyio's task group context mismatch. The worker pattern sidesteps this by running the SDK client entirely inside one stable background `asyncio.Task`.

Queue protocol:
- Input queue items: `(prompt, session_id, output_queue)` or `_SHUTDOWN` sentinel
- Output queue items: SDK `Message` objects, `WorkerError(exception)` wrapper, or `None` sentinel (end of turn)
- `WorkerError` is a typed wrapper to avoid ambiguous `isinstance(item, Exception)` checks

Worker lifecycle:
- `start()` — spawns `asyncio.create_task(self._run())`
- `_run()` loop — connects SDK client, then: get from input queue → query client → stream responses to output queue → put `None` sentinel
- On any error during a query: puts `WorkerError` then `None`, then breaks (worker dies; `SessionManager` recreates it)
- `stop()` — puts `_SHUTDOWN`, waits up to 15s, then cancels task

**Graceful disconnect:** closes stdin of the Claude CLI subprocess so the CLI saves its session state to `.claude/` before terminating. Enables `--resume` on pod restart.

`SessionManager`: one worker per `thread_id`. Maintains a per-thread `asyncio.Lock` to serialize concurrent requests. Session IDs are persisted to `claude_session_ids.json` and restored on startup.

---

## AG-UI Protocol Translation (`ag_ui_claude_sdk/adapter.py`)

`ClaudeAgentAdapter._stream_claude_sdk()` consumes Claude SDK messages and emits AG-UI events:

| Claude SDK message | AG-UI event(s) emitted |
|---|---|
| `StreamEvent(type=message_start)` | (starts tracking `current_message_id`) |
| `StreamEvent(type=content_block_start, block_type=thinking)` | `ReasoningStartEvent`, `ReasoningMessageStartEvent` |
| `StreamEvent(type=content_block_delta, delta_type=thinking_delta)` | `ReasoningMessageContentEvent` |
| `StreamEvent(type=content_block_start, block_type=tool_use)` | `ToolCallStartEvent` |
| `StreamEvent(type=content_block_delta, delta_type=input_json_delta)` | `ToolCallArgsEvent` |
| `StreamEvent(type=content_block_stop)` for tool | `ToolCallEndEvent` (or halt if frontend tool) |
| `StreamEvent(type=content_block_delta, delta_type=text_delta)` | `TextMessageStartEvent` (first chunk), `TextMessageContentEvent` |
| `StreamEvent(type=message_stop)` | `TextMessageEndEvent` |
| `AssistantMessage` (non-streamed fallback) | accumulated into `run_messages` |
| `ToolResultBlock` | `ToolCallEndEvent` + `ToolCallResultEvent` |
| `SystemMessage` | `TextMessageStart/Content/End` |
| `ResultMessage` | captured as `_last_result_data` for `RunFinishedEvent` |
| End of stream | `MessagesSnapshotEvent` (full conversation snapshot) |

The entire run is wrapped: `RunStartedEvent` → ... → `RunFinishedEvent` (or `RunErrorEvent`).

---

## gRPC Transport Detail (`bridges/claude/grpc_transport.py`)

### `GRPCSessionListener`

Pod-lifetime background component. One instance per session, started in the lifespan before the initial prompt.

```
  start()
    │
    ├── AmbientGRPCClient.from_env()
    └── asyncio.create_task(_listen_loop())
              │
              └── _listen_loop()  [async, event loop]
                    │
                    ├── ThreadPoolExecutor(max_workers=1)
                    │     └── _watch_in_thread()  [blocking, thread]
                    │           ├── client.session_messages.watch(session_id, after_seq=N)
                    │           ├── loop.call_soon_threadsafe(ready.set)
                    │           └── for msg in stream:
                    │                 asyncio.run_coroutine_threadsafe(msg_queue.put(msg), loop)
                    │
                    └── while True:
                          msg = await msg_queue.get()
                          if msg.event_type == "user":
                              await _handle_user_message(msg)
                          # reconnects with backoff on stream end or error
```

**Reconnect logic:** when the gRPC stream ends (server-side close or network error), `_listen_loop` reconnects with exponential backoff (1s → 30s). `after_seq=last_seq` ensures no messages are replayed.

### `_handle_user_message`

Drives one complete bridge turn per inbound user message:

```
  _handle_user_message(msg)
    │
    ├── parse msg.payload as RunnerInput (fallback: raw string as content)
    ├── runner_input.to_run_agent_input() → RunAgentInput
    ├── capture run_queue = active_streams.get(thread_id)
    ├── GRPCMessageWriter(session_id, run_id, grpc_client)
    │
    ├── async for event in bridge.run(input_data):
    │     ├── active_streams.get(thread_id).put_nowait(event)   → SSE tap
    │     └── writer.consume(event)                              → DB writer
    │
    ├── on exception:
    │     _synthesize_run_error(thread_id, error, active_streams, writer)
    │       ├── put RunErrorEvent into SSE queue
    │       └── asyncio.ensure_future(writer._write_message(status="error"))
    │
    └── finally:
          if run_queue is not None and active_streams.get(thread_id) is run_queue:
              active_streams.pop(thread_id)   ← identity-safe cleanup
```

### `GRPCMessageWriter`

Per-turn consumer. Accumulates `MESSAGES_SNAPSHOT` content (each snapshot is a complete replacement of the conversation). On `RUN_FINISHED` or `RUN_ERROR`, pushes one `PushSessionMessage(event_type="assistant")` to the control plane via `run_in_executor` (non-blocking).

```
  consume(event)
    │
    ├── MESSAGES_SNAPSHOT → self._accumulated_messages = [...]
    ├── RUN_FINISHED      → _write_message(status="completed")
    └── RUN_ERROR         → _write_message(status="error")

  _write_message(status)
    │
    └── run_in_executor(None, _do_push)
          └── client.session_messages.push(
                session_id,
                event_type="assistant",
                payload={"run_id", "status", "messages"}
              )
```

---

## Interrupts (`endpoints/interrupt.py`, `bridges/claude/bridge.py`, `session.py`)

HTTP trigger: `POST /interrupt` with optional `{ "thread_id": "..." }` body.

Flow:
1. `interrupt_run()` endpoint → `bridge.interrupt(thread_id)`
2. `ClaudeBridge.interrupt()` → looks up `SessionWorker` → `worker.interrupt()`
3. `SessionWorker.interrupt()` → `self._client.interrupt()` on `ClaudeSDKClient`

The SDK client's interrupt propagates to the Claude CLI subprocess (signal or stdin close), which stops generation mid-stream. The output queue drains and `None` is eventually put on it, causing `worker.query()` to return.

**Frontend tool halt:** not triggered by HTTP — the adapter sets `self._halted = True` when Claude calls a frontend tool (e.g. `AskUserQuestion`). After the stream ends, `ClaudeBridge.run()` calls `worker.interrupt()` automatically to prevent the SDK from auto-approving the pending tool call.

**Observability:** `bridge.interrupt()` calls `self._obs.record_interrupt()` if tracing is enabled.

---

## Queue Draining

No explicit drain operation. The queue drains through normal flow:

1. **Normal completion:** `_run()` puts all response messages then `None`. `worker.query()` yields until `None`, then returns.
2. **Interrupt:** SDK stops generation. `async for` ends. `None` is put in the `finally` block. `worker.query()` returns.
3. **Worker error:** `WorkerError` then `None`. `worker.query()` raises, propagates through `bridge.run()` → `event_stream()` → `RunErrorEvent`.
4. **Worker death:** `SessionManager.get_or_create()` detects `worker.is_alive == False` on the next request, destroys the dead worker, creates a fresh one using `--resume`.

Per-thread lock: `asyncio.Lock` per thread prevents a second request from being processed while the first is still draining. Lock is held for the entire duration of `worker.query()`.

---

## How New Messages Are Added

**Normal turn (HTTP path):**
1. Frontend sends `POST /agui/run` via backend proxy with `RunnerInput` JSON
2. `run_agent()` endpoint creates `RunAgentInput`, calls `bridge.run(input_data)`
3. `ClaudeBridge.run()` calls `process_messages(input_data)` to extract the last user message
4. `worker.query(user_msg)` puts `(user_msg, session_id, output_queue)` on the input queue
5. Background worker picks it up, sends to Claude CLI, streams responses back

**Normal turn (gRPC path):**
1. Control plane pushes `PushSessionMessage(event_type="user")` to the watch stream
2. `GRPCSessionListener._handle_user_message()` parses payload, calls `bridge.run(input_data)` directly
3. Events are fanned out to the SSE tap queue and `GRPCMessageWriter`

**Auto-prompt:**
- HTTP mode: `_push_initial_prompt_via_http()` POSTs to the backend run endpoint with `metadata.hidden=True`, `metadata.autoSent=True`
- gRPC mode: `_push_initial_prompt_via_grpc()` pushes a `PushSessionMessage(event_type="user")` directly; listener handles it identically to any other user message

**Tool results (frontend HITL tools):**
- Claude halts; user responds; frontend sends next message containing tool result
- On next `run()`, adapter detects `previous_halted_tool_call_id` and emits `ToolCallResultEvent` before starting the new turn

**Tool results (backend MCP tools):**
- Handled internally by Claude CLI — SDK calls MCP server in-process, gets result, continues without HTTP round-trip

---

## MCP Tools (`bridges/claude/mcp.py`, `tools.py`, `corrections.py`)

Three categories of platform-injected MCP servers:

| Server | Tool | Purpose |
|---|---|---|
| `session` | `refresh_credentials` | Lets Claude refresh GitHub/Google/Jira tokens mid-run |
| `rubric` | `evaluate_rubric` | Scores Claude's output against a rubric; logs to Langfuse |
| `corrections` | `log_correction` | Logs human corrections to Langfuse for the feedback loop |

Plus external MCP servers loaded from `.mcp.json` in the workspace. All passed to `ClaudeAgentOptions.mcp_servers`. Wildcard permissions (`mcp__session__*`, etc.) added to `allowed_tools`.

---

## Tracing Middleware (`middleware/tracing.py`)

A transparent async generator wrapper around the event stream. If `obs` (Langfuse `ObservabilityManager`) is present:
- `obs.track_agui_event(event)` called for each event (tracks turns, tool calls, usage)
- Once a trace ID is available (after first assistant message), emits `CustomEvent("ambient:langfuse_trace", {"traceId": ...})` — frontend uses this to link feedback to the trace
- On exception: `obs.cleanup_on_error(exc)` marks the Langfuse trace as errored
- On normal completion: `obs.finalize_event_tracking()`

---

## Feedback (`endpoints/feedback.py`)

`POST /feedback` accepts META events with `metaType: thumbs_up | thumbs_down`. Resolves the Langfuse trace ID (from payload or from `bridge.obs.last_trace_id`), creates a BOOLEAN score in Langfuse. Returns a RAW event for the backend to persist.

---

## `mark_dirty()` and Adapter Rebuilds

When repos or workflows are added at runtime (`POST /repos` or `POST /workflow`), the endpoint calls `bridge.mark_dirty()`. This:

1. Sets `self._ready = False` (triggers `_setup_platform()` on next run)
2. Sets `self._adapter = None` (triggers `_ensure_adapter()` on next run)
3. Captures all current session IDs → `self._saved_session_ids`
4. Async-shuts down the current `SessionManager` (fire-and-forget)
5. On next `run()`: full re-init with new workspace/MCP config, existing conversations resumed via `--resume <session_id>`
