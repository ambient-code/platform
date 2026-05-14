# Skill: grpc-dev

**Activates when:** Working on gRPC streaming, AG-UI event flow, WatchSessionMessages, control plane ↔ runner protocol, or debugging gRPC connectivity.

---

## Architecture

```
Runner Pod (Claude Code)
  │  pushes AG-UI events via gRPC
  ▼
Control Plane (CP)
  │  fan-out multiplexer — one runner, N watchers
  ▼
WatchSessionMessages RPC (streaming)
  │
  ├── acpctl session messages -f
  ├── Go SDK session_watch.go
  ├── Python SDK _grpc_client.py
  └── TUI dashboard (acpctl ambient)
```

## Proto Definitions

Location: `components/ambient-api-server/proto/ambient/v1/sessions.proto`

Key RPC:
```protobuf
rpc WatchSessionMessages(WatchSessionMessagesRequest)
    returns (stream SessionMessageEvent);
```

Generated stubs: `pkg/api/grpc/ambient/v1/sessions_grpc.pb.go`

Regen: `cd components/ambient-api-server && make generate`

## AG-UI Event Types

| Event | Direction | Meaning |
|---|---|---|
| `RUN_STARTED` | runner → CP → client | Session began executing |
| `TEXT_MESSAGE_CONTENT` | runner → CP → client | Token chunk (streaming) |
| `TEXT_MESSAGE_END` | runner → CP → client | Message complete |
| `MESSAGES_SNAPSHOT` | runner → CP → client | Full message history |
| `RUN_FINISHED` | runner → CP → client | Session done (terminal event) |

**`RUN_FINISHED` must be forwarded exactly once.** CP must not duplicate or drop it.

## Authentication

gRPC auth: `pkg/middleware/bearer_token_grpc.go`

**Test token bypass:** When a non-JWT token (e.g. `test-user-token` K8s secret) is used, the JWT username claim is absent from the gRPC context. The `WatchSessionMessages` handler MUST skip the per-user ownership check in this case:

```go
username, ok := CallerUsernameFromContext(ctx)
if ok && username != session.Owner {
    return status.Error(codes.PermissionDenied, "not session owner")
}
// If !ok (no username in context), allow — non-JWT token
```

## Fan-Out Pattern

The CP maintains a subscriber map per session ID. When a new `WatchSessionMessages` client connects:

1. Add channel to subscriber map for `sessionID`
2. Stream events from channel until: client disconnects OR `RUN_FINISHED` received
3. On client disconnect: remove from map
4. On `RUN_FINISHED`: send to all subscribers, then close all channels for that session

```go
type fanOut struct {
    mu   sync.RWMutex
    subs map[string][]chan *SessionMessageEvent  // sessionID → subscribers
}
```

## Debugging gRPC

**Test connectivity:**
```bash
# With grpcurl (if installed)
grpcurl -plaintext -H "Authorization: Bearer $TOKEN" \
  localhost:13595 ambient.v1.Sessions/WatchSessionMessages

# With acpctl (always available)
AMBIENT_TOKEN=$TOKEN AMBIENT_API_URL=http://localhost:13595 \
  acpctl session messages -f --project <project> <session>
```

**Common errors:**

| Error | Cause | Fix |
|---|---|---|
| `PermissionDenied` | Ownership check failing for test token | Skip check when username not in context |
| `Unavailable` | gRPC server not listening | Check api-server pod logs, verify gRPC port |
| `connection reset` | CP crashed on fan-out | Check CP pod logs for panic |
| No events after `RUN_STARTED` | Runner not pushing to CP | Check runner logs for gRPC push errors |

**Check api-server gRPC logs:**
```bash
kubectl logs -n ambient-code -l app=ambient-api-server --tail=100 | grep -i grpc
```

## Runner ↔ CP Compatibility Contract

The runner was broken by a previous CP merge. To avoid repeating:

1. CP is additive — it DOES NOT change how the runner pushes events
2. Runner pushes to a gRPC endpoint on the CP; CP fans out to watchers
3. The runner's existing SSE emission path is UNTOUCHED
4. If CP is absent, the runner still works (degrades gracefully to REST polling)

**Compatibility test before any CP PR:**
```bash
# Create session, watch it, verify full event sequence
acpctl create session --project test --name compat-test "echo hello world"
acpctl session messages -f --project test compat-test
# Expected: RUN_STARTED → TEXT_MESSAGE_CONTENT (tokens) → RUN_FINISHED
# Must complete without errors
```
