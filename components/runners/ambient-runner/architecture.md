  Ambient Runner: Complete Architecture

  Overview

  The runner is a FastAPI server running in a Kubernetes Job pod per session. It implements the https://github.com/ag-ui-protocol/ag-ui — a Server-Sent Events (SSE) streaming protocol for AI agents. The runner bridges between the platform backend and the underlying AI model
  (Claude, Gemini, or LangGraph).

  Backend → POST /agui/run → Runner (FastAPI) → ClaudeBridge → SessionWorker → ClaudeSDKClient (subprocess)
                                                                      ↕
                                                              asyncio.Queue pair

  ---
  Startup & Lifecycle (app.py, main.py)

  1. main.py: Reads RUNNER_TYPE env var (claude-agent-sdk, gemini-cli, or langgraph), dynamically imports and instantiates the corresponding bridge class.
  2. create_ambient_app(bridge): Creates the FastAPI app with a lifespan async context manager:
    - Creates a RunnerContext from env vars (SESSION_ID, WORKSPACE_PATH)
    - Calls bridge.set_context(context) — gives the bridge its runtime identity
    - Checks IS_RESUME — if this is a resumed session, skips auto-prompt
    - If INITIAL_PROMPT env var is set (and not a resume), fires _auto_execute_initial_prompt() as a background asyncio.Task
    - On shutdown: calls bridge.shutdown() for graceful cleanup
  3. Auto-prompt (_auto_execute_initial_prompt): POSTs back to the backend's /agui/run endpoint with the INITIAL_PROMPT as a hidden user message. Uses exponential backoff (up to 8 retries, 2s→30s) because K8s DNS may not propagate before the pod is ready.

  ---
  The Bridge Pattern (bridge.py)

  PlatformBridge is an abstract base class with three required methods:
  - capabilities() — declares framework features to the frontend
  - run(input_data) — async generator yielding AG-UI BaseEvent objects
  - interrupt(thread_id) — stops the current run

  Key lifecycle hooks (override as needed):
  - set_context() — stores the RunnerContext
  - _ensure_ready() / _setup_platform() — lazy one-time init on first run()
  - _refresh_credentials_if_stale() — refreshes tokens every 60s or when GitHub token expiring
  - shutdown() — called on pod termination
  - mark_dirty() — called by repos/workflow endpoints when workspace changes; rebuilds the adapter on next run()

  ---
  ClaudeBridge: The Full Claude Lifecycle (bridges/claude/bridge.py)

  This is the most complete bridge implementation. Its run() method:

  1. _ensure_ready() — On first call, runs _setup_platform():
    - Auth setup (Anthropic API key or Vertex AI credentials)
    - populate_runtime_credentials() / populate_mcp_server_credentials() — fetches GitHub tokens, Google OAuth, Jira tokens from the backend
    - resolve_workspace_paths() — determines cwd and additional dirs
    - build_mcp_servers() — assembles full MCP server config (external + platform tools)
    - build_sdk_system_prompt() — builds the system prompt
    - Initializes ObservabilityManager (Langfuse)
    - Creates SessionManager
  2. _ensure_adapter() — Builds ClaudeAgentAdapter with all options (cwd, permission mode, allowed tools, MCP servers, system prompt). The adapter is cached and reused across runs. A ring buffer of 50 stderr lines is maintained for error reporting.
  3. Worker selection — Gets or creates a SessionWorker for the thread, optionally resuming from a previously saved CLI session ID (for pod restarts).
  4. Event streaming — Acquires a per-thread asyncio.Lock (prevents concurrent requests to the same thread from mixing), calls worker.query(user_msg), wraps the stream through tracing_middleware, and yields events.
  5. Halt detection — After the stream ends, checks adapter.halted. If the adapter halted (because Claude called a frontend HITL tool like AskUserQuestion), calls worker.interrupt() to prevent the SDK from auto-approving the tool call.
  6. Session persistence — After each turn, saves the CLI session ID to disk (claude_session_ids.json) so --resume works after pod restart.

  ---
  SessionWorker & Queue Architecture (bridges/claude/session.py)

  This is the core mechanism that allows the long-lived Claude CLI process to work inside FastAPI:

  Request Handler (async context A)          Background Task (async context B)
          |                                           |
     worker.query(prompt)                      worker._run() loop
          |                                           |
    puts (prompt, session_id,           ←── input_queue.get()
          output_queue) on input_queue               |
          |                               client.query(prompt)
     output_queue.get() in loop          async for msg in client.receive_response()
          |                                    output_queue.put(msg)
          ↓                                    ...
    yields messages                           output_queue.put(None)  ← sentinel

  Why this exists: The Claude Agent SDK uses anyio task groups internally. When you try to use a persistent ClaudeSDKClient inside a FastAPI SSE handler (a different async context), you hit anyio's task group context mismatch bug. The worker pattern sidesteps this by running
  the SDK client entirely inside one stable background asyncio.Task.

  Queue protocol:
  - Input queue items: (prompt: str, session_id: str, output_queue: asyncio.Queue) or _SHUTDOWN sentinel
  - Output queue items: SDK Message objects, WorkerError(exception) wrapper, or None sentinel (signals end of turn)
  - WorkerError is a typed wrapper to avoid ambiguous isinstance(item, Exception) checks

  Worker lifecycle:
  - start() — spawns asyncio.create_task(self._run())
  - _run() loop — connects SDK client, then loops: get from input queue → query client → stream responses to output queue → put None sentinel
  - On any error during a query, puts WorkerError then None, then breaks (worker dies; SessionManager will recreate it)
  - stop() — puts _SHUTDOWN, waits up to 15s, then cancels task

  Graceful disconnect: Closes stdin of the Claude CLI subprocess so the CLI saves its session state to .claude/ before terminating. Enables --resume on pod restart.

  SessionManager: One worker per thread_id. Also maintains a per-thread asyncio.Lock to serialize concurrent requests. Session IDs are persisted to claude_session_ids.json and restored on startup.

  ---
  AG-UI Protocol Translation (ag_ui_claude_sdk/adapter.py)

  ClaudeAgentAdapter._stream_claude_sdk() is the protocol translator. It consumes Claude SDK messages and emits AG-UI events:

  | Claude SDK message                                                 | AG-UI event(s) emitted                                       |
  |--------------------------------------------------------------------|--------------------------------------------------------------|
  | StreamEvent(type=message_start)                                    | (starts tracking current_message_id)                         |
  | StreamEvent(type=content_block_start, block_type=thinking)         | ReasoningStartEvent, ReasoningMessageStartEvent              |
  | StreamEvent(type=content_block_delta, delta_type=thinking_delta)   | ReasoningMessageContentEvent                                 |
  | StreamEvent(type=content_block_start, block_type=tool_use)         | ToolCallStartEvent                                           |
  | StreamEvent(type=content_block_delta, delta_type=input_json_delta) | ToolCallArgsEvent                                            |
  | StreamEvent(type=content_block_stop) for tool                      | ToolCallEndEvent (or halt if frontend tool)                  |
  | StreamEvent(type=content_block_delta, delta_type=text_delta)       | TextMessageStartEvent (first chunk), TextMessageContentEvent |
  | StreamEvent(type=message_stop)                                     | TextMessageEndEvent                                          |
  | AssistantMessage (non-streamed fallback)                           | accumulated into run_messages                                |
  | ToolResultBlock                                                    | ToolCallEndEvent + ToolCallResultEvent                       |
  | SystemMessage                                                      | TextMessageStart/Content/End (system text)                   |
  | ResultMessage                                                      | captured as _last_result_data for RunFinishedEvent           |
  | End of stream                                                      | MessagesSnapshotEvent (full conversation snapshot)           |

  The entire run is wrapped with RunStartedEvent → ... → RunFinishedEvent (or RunErrorEvent).

  ---
  Interrupts (endpoints/interrupt.py, bridges/claude/bridge.py, session.py)

  HTTP trigger: POST /interrupt with optional { "thread_id": "..." } body.

  Flow:
  1. interrupt_run() endpoint → calls bridge.interrupt(thread_id)
  2. ClaudeBridge.interrupt() → looks up the SessionWorker for the thread → calls worker.interrupt()
  3. SessionWorker.interrupt() → calls self._client.interrupt() on the underlying ClaudeSDKClient

  The SDK client's interrupt propagates to the Claude CLI subprocess (typically via a signal or stdin close), which stops generation mid-stream. The output queue will drain (the _run() loop breaks), and None is eventually put on the output queue, causing worker.query() to
  return.

  Frontend tool halt: A different kind of "interrupt" — not triggered by HTTP, but automatically by the adapter when Claude calls a frontend tool (e.g. AskUserQuestion). The adapter sets self._halted = True, and after the stream ends, ClaudeBridge.run() calls
  worker.interrupt() automatically. This prevents the SDK from auto-approving the pending tool call.

  Observability: bridge.interrupt() calls self._obs.record_interrupt() if tracing is enabled.

  ---
  Queue Draining

  There is no explicit "drain" operation. The queue naturally drains through normal flow:

  1. Normal turn completion: _run() puts all response messages on the output queue, then puts None. worker.query() yields messages until None is received, then returns.
  2. Interrupt: The SDK stops generation. The _run() loop's async for msg in client.receive_response() ends. None is put on the output queue (in the finally block of the try/except around the query). worker.query() receives the None and returns.
  3. Worker error: WorkerError is put first, then None. worker.query() raises the exception, which propagates up through adapter.run() → tracing_middleware → bridge.run() → event_stream() in the run endpoint, which emits a RunErrorEvent.
  4. Worker death: If the background task dies, SessionManager.get_or_create() detects worker.is_alive == False on the next request, destroys the dead worker, and creates a fresh one (using --resume to restore session state).

  Per-thread lock: The asyncio.Lock per thread in SessionManager prevents a second request from being processed while the first is still draining. The lock is held for the entire duration of worker.query().

  ---
  How New Messages Are Added

  From the user (normal turn):
  1. Frontend sends POST / (or /agui/run via backend proxy) with RunnerInput JSON containing messages array
  2. run_agent() endpoint creates RunAgentInput, calls bridge.run(input_data)
  3. ClaudeBridge.run() calls process_messages(input_data) to extract the last user message
  4. worker.query(user_msg) puts (user_msg, session_id, output_queue) on the input queue
  5. The background worker picks it up, sends to Claude CLI, streams responses back

  Auto-prompt (initial prompt on session start):
  - _auto_execute_initial_prompt() POSTs directly to the backend's run endpoint as if it were a user message, with metadata.hidden = True and metadata.autoSent = True

  Tool results (frontend HITL tools):
  - After Claude halts for a frontend tool, the user responds
  - Frontend sends the next message containing the tool result in messages
  - On the next run() call, adapter.run() detects previous_halted_tool_call_id and emits a ToolCallResultEvent before starting the new turn, then sends the user's message to Claude so it can continue

  Tool results (backend MCP tools):
  - Handled internally by Claude CLI — the SDK calls the MCP server in-process, gets a result, and continues without any HTTP round-trip

  ---
  MCP Tools (bridges/claude/mcp.py, tools.py, corrections.py)

  Three categories of platform-injected MCP servers:

  | Server name | Tool(s)             | Purpose                                                   |
  |-------------|---------------------|-----------------------------------------------------------|
  | session     | refresh_credentials | Lets Claude refresh GitHub/Google/Jira tokens mid-run     |
  | rubric      | evaluate_rubric     | Scores Claude's output against a rubric; logs to Langfuse |
  | corrections | log_correction      | Logs human corrections to Langfuse for the feedback loop  |

  Plus external MCP servers loaded from .mcp.json in the workspace. All are passed to ClaudeAgentOptions.mcp_servers. Wildcard permissions (mcp__session__*, etc.) are added to allowed_tools.

  ---
  Tracing Middleware (middleware/tracing.py)

  A transparent async generator wrapper around the event stream. If obs (Langfuse ObservabilityManager) is present:
  - obs.track_agui_event(event) is called for each event (tracks turns, tool calls, usage)
  - Once a trace ID is available (after first assistant message), emits a CustomEvent("ambient:langfuse_trace", {"traceId": ...}) — frontend uses this to link feedback to the trace
  - On exception: calls obs.cleanup_on_error(exc) to mark the Langfuse trace as errored
  - On normal completion: calls obs.finalize_event_tracking()

  ---
  Feedback (endpoints/feedback.py)

  POST /feedback accepts META events with metaType: thumbs_up | thumbs_down. Resolves the Langfuse trace ID (from payload or from bridge.obs.last_trace_id), creates a BOOLEAN score in Langfuse. Returns a RAW event for the backend to persist.

  ---
  mark_dirty() and Adapter Rebuilds

  When repos or workflows are added at runtime (via POST /repos or POST /workflow), the endpoint calls bridge.mark_dirty(). This:
  1. Sets self._ready = False (triggers _setup_platform() on next run)
  2. Sets self._adapter = None (triggers _ensure_adapter() on next run)
  3. Captures all current session IDs → self._saved_session_ids
  4. Async-shuts down the current SessionManager (fire-and-forget)
  5. On next run(): full re-init with new workspace/MCP config, but existing conversations are resumed via --resume <session_id>
