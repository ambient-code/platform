# Control Plane Development Context

**When to load:** Working on the Control Plane (CP) component, the runner (`components/runners/ambient-runner/`), or the runner ↔ CP protocol

## Quick Reference

- **CP role:** gRPC fan-out multiplexer between api-server and runner pods — multiple clients can watch one session; runner pushes once
- **Runner role:** Python process inside the session pod — executes Claude Code CLI, pushes AG-UI events via gRPC
- **Protocol:** gRPC (proto definitions in `components/ambient-api-server/proto/ambient/v1/`)
- **Runner entry:** `components/runners/ambient-runner/main.py`
- **gRPC bridge:** `components/runners/ambient-runner/ambient_runner/bridges/claude/grpc_transport.py`
- **Runner env:** controlled by CP via pod env vars — see `kube_reconciler.go:buildEnv()`

---

## CP ↔ Runner Compatibility Contract

The CP was once reverted from upstream because it interfered with the runner's SSE/polling flow. Every CP change must be validated against the existing runner before merging.

| Concern | Runner expects | CP must preserve |
|---|---|---|
| Session start | Pod provisioned by CP | CP does not reschedule |
| Event emission | Runner pushes AG-UI events via gRPC | CP forwards in order, never drops |
| `RUN_FINISHED` | Emitted once, last | CP forwards exactly once — never duplicated |
| `MESSAGES_SNAPSHOT` | Emitted periodically | CP forwards in order |
| Token | Runner receives token from K8s secret | CP does not touch runner token |
| Non-JWT tokens | `test-user-token` has no username claim | CP skips ownership check when JWT username absent |

**gRPC watches are additive** — CP adds gRPC streaming on top of existing REST/SSE, not replacing it. Do not change the runner's existing event consumption path.

**Runner compat test — run before any CP PR:**
```bash
acpctl create session --project my-project --name test-cp "echo hello"
acpctl session messages -f --project my-project test-cp
```
Expected: `RUN_STARTED` → `TEXT_MESSAGE_CONTENT` (tokens) → `RUN_FINISHED`
Must NOT see: connection errors, dropped events, duplicate `RUN_FINISHED`

---

## CP Fan-Out Architecture

```
Client (SDK/CLI/UI)
  └── WatchSessionMessages RPC (streaming)
        └── CP in-memory subscriber map (session_id → []chan)
              └── GRPCSessionListener (runner side)
                    └── Runner pod pushes AG-UI events → GRPCMessageWriter → PushSessionMessage RPC
```

- CP maintains one in-memory subscriber map per session ID
- Multiple clients can watch the same session simultaneously
- Runner pushes once per event; CP fans out to all active watchers
- CP does not persist events — api-server DB is the durable store

**Auth in gRPC:** Skip ownership check when JWT username is not in context — non-JWT tokens (`test-user-token`) will fail ownership checks. The check is `if username == "" { skip }`.

---

## Runner Architecture

The runner (`ambient_runner/`) runs inside the session pod. Its main components:

### Bridge (`bridge.py` / `claude/bridge.py`)
- Drives the Claude Code CLI subprocess
- Emits AG-UI events: `RUN_STARTED` → `TEXT_MESSAGE_CONTENT` (N) → `TEXT_MESSAGE_END` → `MESSAGES_SNAPSHOT` → `RUN_FINISHED`
- `RUN_FINISHED` is emitted exactly once, last — CP relies on this to close all watch streams

### gRPC Transport (`bridges/claude/grpc_transport.py`)

Two classes:

**`GRPCSessionListener`** — pod-lifetime subscriber. Watches `WatchSessionMessages` for this session. For each `event_type=="user"` message, parses payload as `RunnerInput` and calls `bridge.run()`. Sets `self.ready` event once stream is open.

**`GRPCMessageWriter`** — per-turn event consumer. Accumulates `MESSAGES_SNAPSHOT` content. On `RUN_FINISHED` or `RUN_ERROR`, calls `PushSessionMessage(event_type="assistant", payload=assistant_text)` — writes the durable DB record.

Only active when `AMBIENT_GRPC_ENABLED=true` (set by CP when `AMBIENT_GRPC_URL` is non-empty).

### Inbox drain at session start

The runner drains the agent's inbox before starting the Claude Code session. All unread messages are assembled into `INITIAL_PROMPT` via `assembleInitialPrompt()` in the CP (`reconciler/kube_reconciler.go`). The runner receives this as the `INITIAL_PROMPT` env var.

### Credential fetch (Wave 5)

The CP resolves credentials for the session before pod creation. It calls `sdk.Credentials().ListAll()` — the API server applies RBAC-scoped filtering server-side, returning only credentials visible to the session's service account. The CP takes the first credential per provider (first-match wins; ordering is server-determined). It then:

1. Builds `CREDENTIAL_IDS` — a JSON map of `provider → credential_id` — and injects it into the runner pod env
2. Grants `credential:token-reader` on each credential ID to the runner pod's service account

The runner reads `CREDENTIAL_IDS` at startup and calls `GET /api/ambient/v1/credentials/{id}/token` per provider. Response always uses `token` field (uniform across all providers). See `platform/auth.py:_fetch_credential()`.

| Provider | Env var(s) set | File written |
|----------|---------------|--------------|
| `github` | `GITHUB_TOKEN` | `/tmp/.ambient_github_token` |
| `gitlab` | `GITLAB_TOKEN` | `/tmp/.ambient_gitlab_token` |
| `jira` | `JIRA_URL`, `JIRA_API_TOKEN`, `JIRA_EMAIL` | — |
| `google` | `USER_GOOGLE_EMAIL` | `credentials.json` (token value is full SA JSON) |

### AG-UI event order (invariant)

```
RUN_STARTED
  → TEXT_MESSAGE_CONTENT (emitted N times, streaming tokens)
  → TEXT_MESSAGE_END
  → MESSAGES_SNAPSHOT     (complete conversation snapshot)
  → RUN_FINISHED          (terminal — emitted exactly once)
```

Deviation from this order breaks CP's stream closing logic and `GRPCMessageWriter`'s assembly.

---

## Runner Pod Addressing

The api-server does not have a built-in proxy to runner pods. Runner pods are addressed by Kubernetes cluster-internal DNS:

```
http://session-{KubeCrName}.{KubeNamespace}.svc.cluster.local:8001
```

The `Session` model stores `KubeCrName` and `KubeNamespace` — both available from the DB. The runner listens on port `8001` (set via `AGUI_PORT` env var by the CP; runner default is `8000` but the CP overrides it).

This pattern is used by `components/backend/websocket/agui_proxy.go` (V1 backend). Any new proxy endpoint in the api-server must implement this same addressing.

### Implementing `GET /sessions/{id}/events` (Runner SSE Proxy)

This endpoint proxies the runner pod's `GET /events/{thread_id}` SSE stream through to the client:

```go
func (h *eventsHandler) StreamRunnerEvents(w http.ResponseWriter, r *http.Request) {
    id := mux.Vars(r)["id"]
    session, err := h.sessionSvc.Get(r.Context(), id)
    if err != nil {
        // 404
        return
    }
    runnerURL := fmt.Sprintf("http://session-%s.%s.svc.cluster.local:8001/events/%s",
        *session.KubeCrName, *session.KubeNamespace, *session.KubeCrName)

    req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, runnerURL, nil)
    req.Header.Set("Accept", "text/event-stream")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        // 502
        return
    }
    defer resp.Body.Close()

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no")
    w.WriteHeader(http.StatusOK)
    if f, ok := w.(http.Flusher); ok {
        f.Flush()
    }

    io.Copy(w, resp.Body)
    if f, ok := w.(http.Flusher); ok {
        f.Flush()
    }
}
```

Register in `plugin.go`:
```go
sessionsRouter.HandleFunc("/{id}/events", eventsHandler.StreamRunnerEvents).Methods(http.MethodGet)
```

`thread_id` in the runner = `session.KubeCrName` (the session ID as stored in `KubeCrName`).

---

## gRPC Local Port-Forward

The Go SDK derives the gRPC address from the REST base URL hostname + port `9000`. When pointing at `http://127.0.0.1:8000`, it derives `127.0.0.1:9000`. Port 9000 may be occupied by minio locally.

Fix for local development:
```bash
kubectl port-forward svc/ambient-api-server 19000:9000 -n ambient-code &
export AMBIENT_GRPC_URL=127.0.0.1:19000
```

The TUI's `PortForwardEntry` for gRPC maps to local port `19000` — use this consistently.

Long-term: add `grpc_url` to `pkg/config/config.go` so it can be set once via `acpctl config set grpc_url 127.0.0.1:19000`.

---

## Runner Build Commands

```bash
cd components/runners/ambient-runner

uv venv && uv pip install -e .   # Set up virtualenv
python -m pytest tests/          # Run tests
ruff format .                    # Format
ruff check .                     # Lint
```

**Build and push runner image (for kind):**
```bash
podman build --no-cache -t localhost/vteam_runner:latest components/runners/ambient-runner
podman save localhost/vteam_runner:latest | \
  podman exec -i ${CLUSTER}-control-plane ctr --namespace=k8s.io images import -
kubectl rollout restart deployment/ambient-runner -n ambient-code
```

---

## Pre-Commit Checklist (CP)

- [ ] Existing runner SSE path untouched
- [ ] gRPC `WatchSessionMessages` tested with `acpctl session messages -f`
- [ ] `RUN_FINISHED` forwarded exactly once — no duplication
- [ ] Non-JWT tokens (`test-user-token`) work — no ownership check failure
- [ ] Multiple concurrent watchers tested (fan-out correctness)
- [ ] CP revert scenario documented — can disable CP without breaking runner

## Pre-Commit Checklist (Runner)

- [ ] `python -m pytest tests/` passes
- [ ] `ruff check .` clean
- [ ] AG-UI event order preserved: `RUN_STARTED` → `TEXT_MESSAGE_CONTENT` → `TEXT_MESSAGE_END` → `MESSAGES_SNAPSHOT` → `RUN_FINISHED`
- [ ] `RUN_FINISHED` emitted exactly once, last
- [ ] `GRPCMessageWriter` accumulates `MESSAGES_SNAPSHOT` correctly
- [ ] `AMBIENT_GRPC_ENABLED` guard respected — no gRPC code runs when flag is false
- [ ] Runner compat test passes end-to-end against kind
