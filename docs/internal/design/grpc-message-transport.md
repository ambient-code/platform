# gRPC Message Transport — Implementation Plan

## Goal

Make the DB Message table the canonical source of truth for all session
messages. Introduce a gRPC-backed transport so that sessions created
programmatically via API are first-class citizens in the frontend UI, and
so that all turns — regardless of origin — are durably persisted.

The existing HTTP/SSE path is preserved and extended, not replaced.

---

## Architecture Summary

```
┌─────────────────────────────────────────────────────────────────┐
│                      DB Message Table                           │
│              (canonical bus, append-only, jsonb payload)        │
└──────────────────────────┬──────────────────────────────────────┘
                           │ gRPC fan-out (WatchSessionMessages)
           ┌───────────────┼───────────────┐
           ▼               ▼               ▼
     Backend SSE      Runner listener   API consumer
   (history replay    (inbound trigger  (programmatic
    + new messages)    for all turns)    subscriber)
```

### Inbound (user message → runner)

All user messages — from the frontend UI or a programmatic API caller —
are created as DB Messages via `PushSessionMessage` gRPC. The runner's gRPC
listener picks them up and triggers `bridge.run()` directly.

### Outbound (runner response → DB)

The runner consumes the full event stream internally. At turn end it calls
`PushSessionMessage` with an assembled payload (MESSAGES_SNAPSHOT content +
`run_id` for correlation). One DB write per turn.

### Real-time streaming (frontend UX)

The backend opens `GET /events/{thread_id}` on the runner **before** calling
`PushSessionMessage`. It proxies this SSE tap to the frontend for the duration
of the active turn. The runner closes the tap when the turn ends
(`RUN_FINISHED`). The backend's SSE response to the frontend closes with
it.

The durable DB message arrives via gRPC fan-out shortly after. The
frontend uses the `run_id` embedded in the payload to correlate it with
the already-rendered streamed content and reconcile (replace, not append).

---

## Component Changes

### 1. Runner (Python) — `components/runners/ambient-runner/`

#### 1.1 New file: `ambient_runner/bridges/claude/grpc_transport.py`

Contains two classes.

**`GRPCSessionListener`**

Subscribes to the Messages API gRPC stream for this session. Translates
inbound user messages into `RunAgentInput` and calls `bridge.run()`.

```
GRPCSessionListener
  __init__             → self.ready = asyncio.Event()
  start()              → asyncio.create_task(_listen_loop())
  stop()               → cancel task, close channel
  _listen_loop()       → stream = _grpc_client.session_messages.watch(session_id, after_seq=last_seq)
                          self.ready.set()   # stream open; server delivers from here
                          for msg in stream:
                            filter event_type == "user"
                            build RunAgentInput from message.payload
                            thread_id = message.payload["thread_id"]  # backend populates this
                            writer = GRPCMessageWriter(session_id)
                            # Listener owns the fan-out loop; bridge.run() is a pure generator:
                            async for event in bridge.run(input_data):
                              # (a) feed SSE tap queue if a subscriber is connected
                              if thread_id in bridge._active_streams:
                                  await bridge._active_streams[thread_id].put(event)
                              # (b) feed writer for assembled DB write
                              await writer.consume(event)
```

Reconnects with exponential backoff on transient gRPC errors. Uses K8s
service account token from `BOT_TOKEN` (mounted by
the operator) as bearer auth on the channel.

**`GRPCMessageWriter`**

Consumes the event stream from `bridge.run()`. Emits one `PushSessionMessage`
call per turn when `RUN_FINISHED` is seen. Does not emit `MESSAGES_SNAPSHOT`
to any SSE stream — that event is the source of the DB payload only.

```
GRPCMessageWriter
  consume(event)        # called once per event by the listener's fan-out loop
    accumulate messages from MESSAGES_SNAPSHOT (store, don't forward)
    on RUN_FINISHED:
      call _write_message(assembled_messages, run_id, status="completed")
    on RUN_ERROR:
      call _write_message(assembled_messages, run_id, status="error")
    on RUN_FINISHED after HITL halt:
      call _write_message(assembled_messages, run_id, status="waiting_input")
  _write_message(messages, run_id, status)
    stub.PushSessionMessage(
      session_id=session_id,
      event_type="assistant",
      payload={
        "run_id": run_id,
        "status": status,           # "completed" | "error" | "waiting_input"
        "messages": messages,       # assembled list from MESSAGES_SNAPSHOT
      }
    )
```

DB payload is intentionally lean — not the full AG-UI event envelope. The
`run_id` field is the correlation key for deduplication on the frontend.

#### 1.2 New endpoint: `ambient_runner/endpoints/events.py`

`GET /events/{thread_id}` — real-time SSE tap for an in-progress turn.

```
GET /events/{thread_id}
  bridge = request.app.state.bridge
  subscribe to in-progress event stream for thread_id
  yield all AG-UI events EXCEPT MESSAGES_SNAPSHOT
  close when RUN_FINISHED or RUN_ERROR is yielded
  client closes connection when done listening
```

The runner closes the SSE response when the turn's generator exhausts
(same lifecycle as today's `POST /`). `MESSAGES_SNAPSHOT` is filtered out
— it is only used internally by `GRPCMessageWriter` to build the DB
payload.

**SSE tap ordering — primary guarantee**: the backend opens
`GET /events/{thread_id}` before calling `PushSessionMessage` (see §2.1).
The `/events` endpoint creates a bounded `asyncio.Queue(maxsize=100)` and
registers it in `bridge._active_streams[thread_id]` on connection. Because
the queue exists before the message is pushed, the gRPC fan-out cannot
fire before a subscriber is registered. No polling required in the normal
path.

**Defensive fallback**: in edge cases (very slow lifespan startup, queue
not yet registered when the first event arrives), the endpoint polls
`_active_streams[thread_id]` with 100ms sleep intervals up to 2s before
returning 404. The backend retries the tap once with a 500ms delay on 404
as a last resort before surfacing an error to the frontend.

To support this, `ClaudeBridge` needs an in-progress stream registry: a
dict keyed by `thread_id` so that `GET /events/{thread_id}` can attach to
a run started by the gRPC listener.

**In-progress stream registry** (added to `ClaudeBridge`):

```python
self._active_streams: dict[str, asyncio.Queue] = {}
```

The `/events` endpoint creates the queue and registers it before returning
the SSE response. The listener's fan-out loop finds the queue and feeds it
event-by-event. The listener removes the queue entry when the turn ends.
`bridge.run()` is a pure generator — it has no knowledge of `_active_streams`
and is unchanged from the HTTP path.

#### 1.3 Modified: `ambient_runner/bridges/claude/bridge.py`

**`_setup_platform()`** — start the gRPC listener if `AMBIENT_GRPC_URL`
is set:

```python
grpc_url = os.getenv("AMBIENT_GRPC_URL", "").strip()
if grpc_url:
    from ambient_runner.bridges.claude.grpc_transport import GRPCSessionListener
    self._grpc_listener = GRPCSessionListener(
        bridge=self,
        session_id=self._context.session_id,
        grpc_url=grpc_url,
    )
    await self._grpc_listener.start()
    logger.info("gRPC listener started for session %s", self._context.session_id)
else:
    self._grpc_listener = None
```

**`shutdown()`** — stop the listener:

```python
if self._grpc_listener:
    await self._grpc_listener.stop()
```

**`__init__()`** — add:

```python
self._grpc_listener = None
self._active_streams: dict[str, asyncio.Queue] = {}
```

#### 1.4 Modified: `ambient_runner/app.py`

Register the new events endpoint in `add_ambient_endpoints()`:

```python
from ambient_runner.endpoints.events import router as events_router
app.include_router(events_router)
```

**Remove `_background_inbound_watcher`**: the existing background watcher
in `app.py` that subscribes to gRPC and POSTs back to the runner's own
`POST /` endpoint is replaced entirely by `GRPCSessionListener`. The
difference is architectural: the old watcher did an HTTP round-trip back
to itself (`POST /`) to trigger a run; `GRPCSessionListener` calls
`bridge.run()` directly with no round-trip. Remove `_background_inbound_watcher`
and its `asyncio.create_task` call from the lifespan.

**Eager platform setup**: when `AMBIENT_GRPC_URL` is set, `_setup_platform()`
must be called eagerly in the lifespan startup block — not lazily on first
`bridge.run()` call. This ensures `GRPCSessionListener` is started and its
`ready` event is set before `_auto_execute_initial_prompt()` pushes the
initial message. Without eager setup, the initial prompt arrives before the
listener is subscribed and is silently dropped.

```python
# In lifespan startup (app.py):
grpc_url = os.getenv("AMBIENT_GRPC_URL", "").strip()
if grpc_url:
    await bridge._setup_platform()   # starts GRPCSessionListener eagerly
    await bridge._grpc_listener.ready.wait()  # wait for WatchSessionMessages stream
asyncio.create_task(_auto_execute_initial_prompt(...))
```

**`_auto_execute_initial_prompt()`** — when `AMBIENT_GRPC_URL` is set,
write the initial prompt as a DB Message via `PushSessionMessage` instead
of POSTing to `BACKEND_API_URL/agui/run`. The runner's own
`GRPCSessionListener` will pick it up and trigger the run. This makes the
initial prompt observable to API consumers and visible in the frontend
session history.

```python
grpc_url = os.getenv("AMBIENT_GRPC_URL", "").strip()
if grpc_url:
    await _push_initial_prompt_message(prompt, session_id, grpc_url)
else:
    # existing HTTP POST path unchanged
    await _http_post_initial_prompt(prompt, session_id, ...)
```

#### 1.5 Env vars (runner)

No new variables required. The runner's `_grpc_client.py` already reads
`AMBIENT_GRPC_URL` and `BOT_TOKEN`; the operator already injects both.

| Variable | Purpose | Required |
|---|---|---|
| `AMBIENT_GRPC_URL` | gRPC endpoint for Messages API (already read by `_grpc_client.py`) | No — absence keeps existing HTTP/SSE behavior |
| `BOT_TOKEN` | K8s SA token for gRPC bearer auth (already injected by operator) | If `AMBIENT_GRPC_URL` set |

---

### 2. Backend (Go) — `components/backend/`

#### 2.1 Modified: session run handler

Today: `POST /agui/run` proxies the request body to the runner pod and
streams the runner's SSE response back to the frontend inline.

New behavior when `AMBIENT_GRPC_URL` is configured (feature-flagged):

```
POST /agui/run
  1. Open GET /events/{thread_id} on runner pod (SSE tap — BEFORE push)
     → runner registers stream_queue in _active_streams[thread_id]
  2. Call stub.PushSessionMessage(session_id, event_type="user", payload={...RunAgentInput...})
     → DB Message created → gRPC fan-out → runner listener fires
  3. Proxy runner SSE tap to frontend as SSE response
  4. Runner closes tap on RUN_FINISHED → backend closes frontend SSE
  5. (Async) runner writes assembled message to DB → gRPC fan-out → frontend reconciles
```

Opening the SSE tap first ensures the queue is registered before the
gRPC fan-out fires. The race condition is eliminated by ordering, not by
polling.

Old behavior (no `AMBIENT_GRPC_URL`): unchanged. The proxy path remains
as-is for backward compatibility.

#### 2.2 Modified: backend SSE session stream endpoint

Today: not backed by gRPC (proxies runner SSE or reads CR status).

New: when a frontend client is watching a session (not actively sending a
message), the backend subscribes to `WatchSessionMessages(session_id)` via
the Go gRPC SDK and streams new DB Messages to the frontend as SSE events.

This powers:
- Session list updates (new session appears immediately)
- History display when clicking into a gRPC-originated session
- Receiving the durable assembled message after a turn completes (for
  reconciliation using `run_id`)

#### 2.3 Go SDK usage

The backend imports the existing Go SDK for the Messages API (already
available in the repo). Two methods used:

| SDK method | Signature | Actual gRPC method | Where used |
|---|---|---|---|
| `PushSessionMessage` | `(session_id, event_type, payload)` | `/ambient.v1.SessionService/PushSessionMessage` | Run handler (inbound trigger) |
| `WatchSessionMessages` | `(session_id)` | `/ambient.v1.SessionService/WatchSessionMessages` | Session stream SSE endpoint |

The `event_type` field (not `role`) is the proto field used to classify
messages. `components/backend` is a separate service from `ambient-api-server`
(the Messages API) — `WatchSessionMessages` is a normal cross-service gRPC
call, not an in-process subscription.

Auth: K8s service account token, same mechanism as the runner.

#### 2.4 Env vars (backend)

| Variable | Purpose |
|---|---|
| `AMBIENT_GRPC_URL` | Messages API gRPC endpoint |
| `BOT_TOKEN` | K8s SA bearer token for gRPC auth |

---

### 3. Frontend — `components/frontend/`

No changes required for the core flow. The frontend already handles
`MESSAGES_SNAPSHOT` as a replacement signal (not additive). The `run_id`
in the DB payload allows reconciliation of the streamed content with the
durable record.

Deduplication rule (existing behavior, already implemented):
> When `MESSAGES_SNAPSHOT` arrives, replace the optimistically-rendered
> streamed content for the matching `run_id`. Do not append.

One addition to verify: the frontend's session list subscription should
be backed by the gRPC-sourced SSE endpoint (§2.2) so that
programmatically-created sessions appear without a page refresh.

---

## Turn Lifecycle — End to End

### Frontend-initiated turn

```
User types message → POST /agui/run
  Backend: GET /events/{thread_id} on runner (opens SSE tap — BEFORE push)
  Backend: PushSessionMessage(event_type="user", payload={content, thread_id, run_id})
  Backend: proxies SSE tap to frontend

  Runner GRPCSessionListener: receives user message → bridge.run(input_data) directly
  Runner: streams events → SSE tap → backend → frontend (real-time)
  Runner: RUN_FINISHED → closes SSE tap → backend closes frontend SSE

  Runner GRPCMessageWriter: PushSessionMessage(event_type="assistant", payload={run_id, status, messages})
  gRPC fan-out → frontend: receives durable message, reconciles by run_id
```

### API-initiated turn (no frontend connected)

```
API caller: PushSessionMessage(event_type="user", payload={content, thread_id, run_id})
  gRPC fan-out → runner GRPCSessionListener: bridge.run(input_data) directly
  Runner: streams events internally (no SSE tap subscriber)
  Runner: RUN_FINISHED
  Runner GRPCMessageWriter: PushSessionMessage(event_type="assistant", payload={run_id, status, messages})
  gRPC fan-out → API subscriber: receives assembled message
```

### API-initiated turn (human clicks in mid-session)

```
Session already running or idle
  Frontend: subscribes to WatchSessionMessages(session_id) via backend SSE
  Frontend: receives history replay (seq=0 to seq=current)
  Frontend: renders existing conversation

User sends message → same as frontend-initiated turn above
```

### HITL halt (AskUserQuestion)

```
Runner: Claude calls AskUserQuestion → adapter halts → RUN_FINISHED emitted
Runner GRPCMessageWriter: PushSessionMessage(event_type="assistant", payload={run_id, status="waiting_input", messages})
gRPC fan-out → frontend: sees status="waiting_input" → renders waiting state

User answers → new POST /agui/run → new turn starts (full cycle above)
```

---

## Deduplication

The frontend receives the assistant's text twice per turn:
1. Streamed as deltas via the SSE tap (real-time)
2. As a complete assembled message via gRPC fan-out (durable)

Correlation key: `run_id` is present in both streams.

Rule: when a gRPC message arrives with a `run_id` matching a
`RUN_FINISHED` already received on the SSE tap, treat it as the canonical
replacement for that turn's content — not new content. The frontend already
implements this logic for `MESSAGES_SNAPSHOT` on the existing SSE stream.
No new frontend logic required beyond verifying the `run_id` field is read
from the DB payload.

---

## What Does Not Change

- `SessionWorker` / `SessionManager` — unchanged
- `ClaudeAgentAdapter` — unchanged
- `tracing_middleware` — unchanged (gRPC path passes through it identically)
- `GeminiCLIBridge`, `LangGraphBridge` — unchanged (gRPC listener is
  `ClaudeBridge`-specific initially; the `PlatformBridge` base class can
  expose a hook later)
- `/interrupt` endpoint — unchanged (control signal, not a message)
- `/feedback`, `/repos`, `/workflow`, `/capabilities`, `/health`,
  `/mcp/status` endpoints — unchanged
- MCP tools, observability, Langfuse tracing — unchanged
- Sessions without `AMBIENT_GRPC_URL` set — fully unchanged, existing
  HTTP/SSE path active

---

### 4. Operator — `components/operator/`

No operator code changes required. The operator already injects both env
vars into runner Job pods and the runner service account already has RBAC
permission to call the Messages API gRPC service.

| Env var | Already injected | Notes |
|---|---|---|
| `AMBIENT_GRPC_URL` | Yes | Controls feature flag — runner falls back to HTTP/SSE if absent |
| `BOT_TOKEN` | Yes | K8s SA bearer token; runner reads it directly |

---

## Open Questions (deferred)

- **Frontend history on reconnect**: what exact format does the frontend
  expect when replaying session history from the DB? The `messages` array
  in the DB payload matches the `MESSAGES_SNAPSHOT` content today, but
  the frontend's reconnect logic should be verified against the DB payload
  shape before shipping.
- **`GeminiCLIBridge` / `LangGraphBridge` gRPC support**: the listener
  is implemented in `ClaudeBridge` first. A shared hook in `PlatformBridge`
  (e.g. `start_transport()`) can generalize this later.
- **`/interrupt` and the gRPC listener task**: interrupt calls
  `bridge.interrupt()` → `worker.interrupt()` → `client.interrupt()`, which
  signals the SDK process. This path is unchanged regardless of whether
  `bridge.run()` was called from the HTTP handler or `GRPCSessionListener`.
  Confirm that `client.interrupt()` is accessible from the listener context
  (i.e. the `SessionWorker` reference is stable) before closing.
