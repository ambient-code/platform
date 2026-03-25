# Control Plane Development Context

**When to load:** Working on the Control Plane (CP) component — gRPC bridge, session orchestration, or runner coordination

## Quick Reference

- **Role:** Sits between the Ambient API Server and the runner pods; orchestrates session lifecycle via gRPC
- **Language:** Python (or Go — confirm with `components/` structure)
- **Protocol:** gRPC (proto definitions shared with api-server at `proto/ambient/v1/`)
- **Key concern:** Must NOT break the existing runner — CP and runner have a compatibility contract

## Compatibility Warning

The CP was reverted from `upstream/main` because it interfered with the existing Claude Code runner's SSE/polling flow. When working on CP:

1. **Do not change the runner's existing event consumption path** — the runner reads AG-UI SSE events directly
2. **gRPC watches are additive** — CP adds gRPC streaming on top of existing REST/SSE, not replacing it
3. **Test with the existing runner** before any CP change lands in a PR

## CP ↔ Runner Contract

| Concern | Existing runner expects | CP must preserve |
|---|---|---|
| Session start | Job pod scheduled by operator | CP does not reschedule |
| Event emission | Runner pushes AG-UI events to gRPC | CP forwards, never drops |
| `RUN_FINISHED` | Emitted once at end | CP forwards exactly once |
| `MESSAGES_SNAPSHOT` | Emitted periodically | CP forwards in order |
| Token | Runner receives token from K8s secret | CP does not touch runner token |

## gRPC Session Watch Flow

```
Client (SDK/CLI/UI)
  └── WatchSessionMessages RPC (streaming)
        └── CP subscribes to runner gRPC push
              └── Runner pod pushes AG-UI events → CP → Client
```

The CP is a **fan-out multiplexer** — multiple clients can watch the same session; the runner pushes once.

## Key Files (confirm paths in your worktree)

- gRPC handler: `grpc_handler.go` or equivalent — `WatchSessionMessages` implementation
- Auth: skip ownership check when JWT username not in context (non-JWT tokens like test-user-token)
- Fan-out: in-memory subscriber map per session ID

## Runner Compatibility Test

Before any CP change:
```bash
# Start a session, verify runner emits events correctly
acpctl create session --project my-project --name test-cp "echo hello"
acpctl session messages -f --project my-project test-cp
# Should see: RUN_STARTED → TEXT_MESSAGE_CONTENT (tokens) → RUN_FINISHED
# Must NOT see: connection errors, dropped events, duplicate RUN_FINISHED
```

## Pre-Commit Checklist

- [ ] Existing runner SSE path untouched
- [ ] gRPC WatchSessionMessages tested with `acpctl session messages -f`
- [ ] `RUN_FINISHED` forwarded exactly once
- [ ] Non-JWT tokens (test-user-token) work — no ownership check failure
- [ ] Multiple concurrent watchers tested (fan-out)
- [ ] CP revert scenario documented — can disable CP without breaking runner
