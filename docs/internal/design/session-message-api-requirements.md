# Session Message API — Requirements

## Context

This document defines the requirements for ambient-api-server, go-sdk, and
dependent components to support durable session message transport with correct
authentication and authorization. It supersedes the ad-hoc gRPC client
approach previously implemented in the Backend component.

Both the API agent (ambient-api-server, go-sdk, CLI, Frontend) and the CP
agent (Backend, Runner, ambient-control-plane) work from this document. Each
requirement is annotated with its owner.

## Component Roles (Canonical)

| Component | Role |
|---|---|
| **Frontend** | NextJS UI — talks to Backend only |
| **Backend** (`components/backend/`) | Frontend's API gateway — session CRUD, AG-UI SSE proxy |
| **ambient-api-server** | Authoritative service — persists sessions and messages, serves gRPC + REST |
| **ambient-control-plane** | Watches ambient-api-server, creates and manages runner Jobs |
| **Runner** | Executes Claude Code CLI inside Job pods, pushes `event_type="assistant"` messages |
| **go-sdk** | HTTP/gRPC client library — all first-party Go components talk to ambient-api-server through this |
| **CLI** (`acpctl`) | User-facing terminal interface, talks to ambient-api-server via go-sdk |

The Backend component must never import `ambient-api-server` directly. All
ambient-api-server interactions go through go-sdk.

---

## Requirements

### R1 — Session creation pushes `initialPrompt` as the first message
**Owner: API agent**

When a session is created with a non-empty `initialPrompt`, ambient-api-server
must push that prompt as the first session message atomically within the
session creation request handler, before returning the created session to the
caller.

```
POST /api/ambient/v1/sessions
Body: { ..., "initialPrompt": "fix the failing tests" }

Server-side, after persisting the session record:
  if req.InitialPrompt != "" {
      msgService.Push(ctx, session.ID, "user", req.InitialPrompt)
  }
```

**Payload format:** The API server stores the raw prompt string as-is. No
AG-UI formatting is applied at the server. The runner receives the raw string
and wraps it into `RunAgentInput` at execution time (see R-runner).

**Implementation gaps (API agent):**
- `initialPrompt` field must be added to the session OpenAPI request struct
- `MessageService` must be injected into the session handler or service
- `msgService.Push()` call added after successful session create

**Rationale:** The server creates the session and has full authority to seed
its own data as part of that transaction. No component needs to impersonate a
user to push the initial message. The prompt is in the DB before any runner
pod starts, so the runner's `GRPCSessionListener` receives it via `after_seq=0`
replay on subscription.

**Consequence for Backend — single flow, no duplication risk:**

The Backend currently creates AgenticSession CRs directly via the Kubernetes
API, not by calling `POST /api/ambient/v1/sessions`. This creates a fork in
the session creation path:

- **Option A (chosen):** Backend calls `POST /api/ambient/v1/sessions` on
  ambient-api-server (which applies R1 atomically), then ambient-api-server
  notifies ambient-control-plane, which creates the CR and pod. The Backend
  no longer writes CRs directly. This is the cleanest path and eliminates
  the duplication risk entirely.

- **Option B (rejected):** Backend creates the CR as before and then calls
  `sdk.Sessions.PushMessage()` as a follow-up. If ambient-api-server also
  applies R1 when it syncs the CR from the control-plane, the initial prompt
  is pushed twice. This option is not viable without a deduplication guard.

**Option A is the required implementation.** The Backend session creation
handler is updated to call `sdk.Sessions.Create()` (which hits
`POST /api/ambient/v1/sessions`) and must pass `initialPrompt` in the request
body. The Backend stops writing AgenticSession CRs directly. The
ambient-control-plane becomes the only CR writer, triggered by ambient-api-server.

---

### R2 — REST endpoint: push a session message
**Owner: API agent**

```
POST /api/ambient/v1/sessions/{id}/messages
Authorization: Bearer <user-jwt>
Body: {
  "event_type": "user",
  "payload": "<string>"
}
Response: SessionMessage { id, session_id, seq, event_type, payload, created_at }
```

**Auth:** JWT required. Session ownership enforced — the caller must own or
have write access to the session identified by `{id}`.

**event_type authority:** Only `event_type="user"` is accepted on this
endpoint. Requests with any other `event_type` are rejected with `400`. The
runner pushes `event_type="assistant"` via gRPC using its service token; that
is a separate code path.

**Current state:** Route is registered and JWT + RBAC middleware is applied.
Gap: no `event_type` enforcement in the handler. Add reject for non-`"user"`.

---

### R3 — REST endpoint: stream session messages (SSE)
**Owner: API agent**

```
GET /api/ambient/v1/sessions/{id}/messages
Authorization: Bearer <user-jwt>
Accept: text/event-stream
Query: ?after_seq=<int>   (optional, default 0)
Response: SSE stream of SessionMessage objects, replays existing then streams live
```

**Auth:** JWT required. Session read access enforced.

**Current state:** Route registered, JWT + RBAC applied, `after_seq` via
`cursorFromRequest` already exists. Verify middleware applies to this
sub-route specifically (not just the parent group). Likely complete.

---

### R4 — gRPC: runner push authority scoped to `event_type="assistant"`
**Owner: API agent**

`PushSessionMessage` via gRPC with `AMBIENT_API_TOKEN` (service token) must
only be accepted when `event_type="assistant"`. Attempts to push
`event_type="user"` with a service token must be rejected with
`codes.PermissionDenied`.

**Implementation:** Handler-level check in `sessionGRPCHandler.PushSessionMessage`:

```go
callerIsService := isServiceToken(ctx)
if callerIsService && req.GetEventType() == "user" {
    return nil, status.Error(codes.PermissionDenied,
        "service token may not push event_type=user")
}
```

**`isServiceToken` implementation requirement:** The JWT interceptor and
service token auth path must set distinguishable values in the request context.
The API team must define and document the context key and value used to mark
service token identity (e.g. a `callerType` key with values `"service"` vs
`"user"`). `isServiceToken(ctx)` reads this key. This detail must be specified
and communicated to the CP agent before R4 ships, as CP components may need
to consume it.

**Current state:** Context key not yet defined. No check in gRPC handler.

---

### R5 — gRPC: WatchSessionMessages session ownership
**Owner: API agent**

Add a handler-level check enforcing session ownership for `WatchSessionMessages`:

- JWT callers may only watch sessions they own or have explicit read access to.
- Service token callers (`AMBIENT_API_TOKEN`) may only watch the session whose
  ID matches the `session_id` field in the request. Watch-all is not granted.

**Current state:** No ownership check in `grpc_handler.go`.

---

### R6 — go-sdk: `types.SessionMessage`
**Owner: API agent**

`types.SessionMessage` already exists in `types/session_message.go` with the
correct shape. No work required. ✅

```go
type SessionMessage struct {
    ID        string
    SessionID string
    Seq       int64
    EventType string    // "user" | "assistant"
    Payload   string
    CreatedAt time.Time
}
```

---

### R7 — go-sdk: `SessionAPI.PushMessage()`
**Owner: API agent**

**Existing method conflict:** `SessionAPI` already has
`PushMessage(ctx, sessionID, *SessionMessagePush)`. R7 replaces it.

```go
// PushMessage sends a user message to a session.
// Uses HTTP POST /api/ambient/v1/sessions/{id}/messages.
// Always sets event_type="user". Payload is a plain string.
// Requires a user JWT — service tokens are rejected by the server (see R4).
func (a *SessionAPI) PushMessage(ctx context.Context, sessionID, payload string) (*types.SessionMessage, error)
```

The struct-based variant is removed. All callers are updated as part of this
work.

---

### R8 — go-sdk: `SessionAPI.WatchMessages()`
**Owner: API agent**

```go
// WatchMessages streams messages for a session from after_seq onward.
// Uses SSE GET /api/ambient/v1/sessions/{id}/messages?after_seq=<n>.
// Requires a user JWT.
// StreamMessages is deprecated; use WatchMessages instead.
func (a *SessionAPI) WatchMessages(ctx context.Context, sessionID string, afterSeq int64) (<-chan *types.SessionMessage, func(), error)
```

Stop-function pattern consistent with `SessionWatcher`. `StreamMessages` is
deprecated and removed once all callers migrate.

---

### R-runner — Runner payload handling for raw initial prompt
**Owner: CP agent**

After R1 lands, the initial prompt arrives via `GRPCSessionListener` as a raw
string payload (not a serialized `RunnerInput`). The current
`_handle_user_message` calls `RunnerInput.model_validate_json(msg.payload)`
unconditionally, which will fail on a plain string.

**Change:** `_handle_user_message` in `grpc_transport.py` must detect payload
shape and handle both cases:

```python
async def _handle_user_message(self, msg: Any) -> None:
    from ambient_runner.endpoints.run import RunnerInput
    import json as _json

    try:
        # Try parsing as full RunnerInput (interactive turn path)
        runner_input = RunnerInput.model_validate_json(msg.payload)
    except Exception:
        # Fall back: treat payload as raw prompt string (initial prompt path)
        runner_input = RunnerInput(
            messages=[{"id": str(uuid.uuid4()), "role": "user", "content": msg.payload}],
            thread_id=self._session_id,
        )

    input_data = runner_input.to_run_agent_input()
    # ... rest unchanged
```

**This is the only behavioral change to the runner in this release.** All
other runner work (items below) is removal.

---

### R-runner-cleanup — Remove initial prompt machinery from runner
**Owner: CP agent | Blocks on R1**

Remove the initial-prompt self-execution code from `app.py`:

- `_push_initial_prompt_via_grpc()`
- `_push_initial_prompt_via_http()`
- `_auto_execute_initial_prompt()`
- `INITIAL_PROMPT` env var read

`IS_RESUME` is retained. It guards existing behavior: when a pod restarts for
an already-started session, `IS_RESUME=true` prevents the runner from treating
the replayed message history as a new trigger. This is current functionality
that must remain at parity. The operator continues injecting `IS_RESUME`.

Startup simplifies to: start `GRPCSessionListener` and wait. If a prompt was
seeded at session creation (R1), it arrives via the listener replay on
subscription. No explicit `seq > 0` branch is needed — the listener loop is
the wait.

---

### R-backend-session — Backend session creation via SDK
**Owner: CP agent | Blocks on R1**

Replace direct AgenticSession CR writes in `components/backend/handlers/sessions.go`
with a call to `sdk.Sessions.Create()`. The `initialPrompt` field from the
frontend request body is passed through to the SDK call. ambient-api-server
applies R1 atomically. The Backend no longer writes CRs directly.

---

### R-backend-agui — Backend AG-UI proxy via SDK
**Owner: CP agent | Blocks on R7, R8**

- `HandleAGUIRunProxy`: replace gRPC push (`grpc_client.go`) with
  `sdk.Sessions.PushMessage(ctx, sessionID, payload)` using the user's JWT
  extracted from the incoming request.
- `HandleAGUIEvents`: replace gRPC watch with `sdk.Sessions.WatchMessages()`
  fanned into `publishLine()`.

---

### R-backend-cleanup — Remove direct ambient-api-server dependency
**Owner: CP agent | Blocks on R-backend-agui**

Once the AG-UI proxy is migrated to the SDK:

- Delete `components/backend/websocket/grpc_client.go`
- Remove `github.com/ambient-code/platform/components/ambient-api-server`
  from `go.mod` and the `replace` directive
- Revert `Dockerfile` to narrow `components/backend/` build context
- Revert `Makefile` `local-reload-backend` and `_build-and-load` targets

---

### R-operator — ambient-control-plane env var cleanup
**Owner: CP agent | Deferred — do not touch in this release**

Remove `INITIAL_PROMPT` env var injection from the Job spec in
`components/operator/internal/handlers/sessions.go`. Deferred until R1 and
R-runner-cleanup are confirmed stable in production.

`IS_RESUME` injection is retained permanently — it is existing functionality
that guards pod-restart behavior and is not part of this cleanup.

---

## Execution Plan

### Phase 1 — API team (parallel, no CP dependency)
| # | Work |
|---|---|
| 1a | R1: add `initialPrompt` to session OpenAPI struct; inject `MessageService`; call `Push()` in create handler |
| 1b | R2: add `event_type="user"` enforcement in `PushMessage` handler |
| 1c | R3: verify middleware on SSE sub-route; confirm `after_seq` complete |
| 1d | R4: define `callerType` context key; implement `isServiceToken`; add gRPC handler check |
| 1e | R5: add `WatchSessionMessages` ownership check |
| 1f | R6: verify `types.SessionMessage` shape — already done ✅ |
| 1g | R7: replace `PushMessage` signature; update callers |
| 1h | R8: add `WatchMessages` stop-function; deprecate `StreamMessages` |

### Phase 2 — CP agent, runner payload handling (unblocked, ship with Phase 1)
| # | Work |
|---|---|
| 2a | R-runner: update `_handle_user_message` to detect raw string vs `RunnerInput` JSON |

### Phase 3 — CP agent, gates on Phase 1 R1 landing
| # | Work |
|---|---|
| 3a | R-runner-cleanup: remove `_push_initial_prompt_via_grpc`, `_via_http`, `_auto_execute_initial_prompt`, `INITIAL_PROMPT` from `app.py`; retain `IS_RESUME` |
| 3b | R-backend-session: replace CR writes with `sdk.Sessions.Create()` |

### Phase 4 — CP agent, gates on Phase 1 R7 + R8 landing
| # | Work |
|---|---|
| 4a | R-backend-agui: `HandleAGUIRunProxy` → `sdk.Sessions.PushMessage()` with user JWT |
| 4b | R-backend-agui: `HandleAGUIEvents` → `sdk.Sessions.WatchMessages()` |

### Phase 5 — CP agent, gates on Phase 4 complete
| # | Work |
|---|---|
| 5a | R-backend-cleanup: delete `grpc_client.go`; revert `go.mod`, `Dockerfile`, `Makefile` |

### Deferred
| # | Work |
|---|---|
| — | R-operator: remove `INITIAL_PROMPT` from operator Job spec; `IS_RESUME` retained |

---

## Sequencing Diagram

```
Phase 1 (API team — parallel):
  R1, R2, R3, R4, R5, R6✅, R7, R8

Phase 2 (CP — unblocked):
  R-runner (payload shape detection)

          R1 lands
             ↓
Phase 3 (CP):
  R-runner-cleanup
  R-backend-session

          R7 + R8 land
               ↓
Phase 4 (CP):
  R-backend-agui

          Phase 4 complete
                  ↓
Phase 5 (CP):
  R-backend-cleanup

Deferred:
  R-operator (after Phase 3 stable in production)
```

---

## Component Change Summary

### ambient-api-server (API agent)

| Req | Change |
|---|---|
| R1 | Add `initialPrompt` to session create request; inject `MessageService`; push raw prompt on create |
| R2 | Reject `event_type != "user"` in `PushMessage` handler |
| R3 | Verify JWT + RBAC on SSE sub-route; confirm `after_seq` complete |
| R4 | Define `callerType` context key; `isServiceToken`; reject `event_type="user"` in gRPC handler |
| R5 | `WatchSessionMessages` ownership check — service token scoped to own session |

### go-sdk (API agent)

| Req | Change |
|---|---|
| R6 | Already complete ✅ |
| R7 | Replace `PushMessage(ctx, id, *struct)` → `PushMessage(ctx, id, payload string)` |
| R8 | Add `WatchMessages()` stop-function; deprecate `StreamMessages` |

### Runner (CP agent)

| Req | Change |
|---|---|
| R-runner | `_handle_user_message`: detect raw string payload; wrap into `RunnerInput` |
| R-runner-cleanup | Remove `_push_initial_prompt_via_grpc`, `_via_http`, `_auto_execute_initial_prompt`, `INITIAL_PROMPT`; retain `IS_RESUME` |

### Backend (CP agent)

| Req | Change |
|---|---|
| R-backend-session | Replace CR writes with `sdk.Sessions.Create()` passing `initialPrompt` |
| R-backend-agui | `HandleAGUIRunProxy` → `sdk.Sessions.PushMessage()`; `HandleAGUIEvents` → `sdk.Sessions.WatchMessages()` |
| R-backend-cleanup | Delete `grpc_client.go`; revert `go.mod`, `Dockerfile`, `Makefile` |

### ambient-control-plane (CP agent)

| Req | Change |
|---|---|
| R-operator | Remove `INITIAL_PROMPT` env var injection — **deferred**; `IS_RESUME` injection retained |

### Frontend (API agent)

No changes required.

### CLI (API agent)

No changes required until R7 + R8 land. `acpctl run` additions are additive.
