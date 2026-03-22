# Control Plane + Runner: Implementation Guide

**Date:** 2026-03-22
**Status:** Living Document
**Spec:** `control-plane.spec.md` — CP architecture, runner structure, message streams, proposed changes

---

## This Document Is Iterative

Each time this guide is invoked, start from Step 1, follow the steps in order, and update this document with what you learned. The goal is convergence. Expect gaps. Fix the guide before moving on.

---

## Overview

This guide covers implementation work on two components:

- **CP** (`components/ambient-control-plane/`) — Go service; K8s reconciler; session provisioner
- **Runner** (`components/runners/ambient-runner/`) — Python FastAPI service; Claude bridge; gRPC client

Changes to these components are independent of the api-server pipeline (no openapi.yaml, no SDK generator). They are deployed as container images to a kind cluster and tested via `acpctl`.

---

## Repository Layout

```
platform-control-plane/
├── components/
│   ├── ambient-control-plane/         ← CP (Go)
│   │   ├── cmd/ambient-control-plane/main.go
│   │   ├── internal/
│   │   │   ├── config/config.go       ← env-based config
│   │   │   ├── watcher/watcher.go     ← gRPC WatchManager
│   │   │   ├── informer/informer.go   ← list+watch Informer
│   │   │   ├── kubeclient/kubeclient.go
│   │   │   └── reconciler/
│   │   │       ├── kube_reconciler.go ← session pod provisioner
│   │   │       ├── shared.go          ← SDKClientFactory, constants
│   │   │       ├── project_reconciler.go
│   │   │       ├── project_settings_reconciler.go
│   │   │       ├── tally.go
│   │   │       └── tally_reconciler.go
│   │   ├── go.mod                     ← replace directives for local SDK
│   │   └── Dockerfile
│   │
│   └── runners/ambient-runner/        ← Runner (Python)
│       ├── ambient_runner/
│       │   ├── app.py                 ← FastAPI factory + lifespan
│       │   ├── _grpc_client.py        ← codegen — AmbientGRPCClient
│       │   ├── _session_messages_api.py ← codegen — SessionMessagesAPI
│       │   ├── bridge.py              ← PlatformBridge base class
│       │   ├── bridges/claude/
│       │   │   ├── bridge.py          ← ClaudeBridge
│       │   │   └── grpc_transport.py  ← GRPCSessionListener + GRPCMessageWriter
│       │   ├── endpoints/
│       │   │   ├── run.py             ← POST /
│       │   │   ├── events.py          ← GET /events/{thread_id}  ← NEW
│       │   │   └── ...
│       │   └── middleware/
│       │       └── grpc_push.py       ← HTTP path fire-and-forget push
│       └── pyproject.toml
│
├── Makefile                           ← build-runner, build-control-plane, local-reload-*
└── docs/internal/design/
    ├── control-plane.spec.md          ← YOU ARE HERE (spec)
    └── control-plane.guide.md         ← YOU ARE HERE (guide)
```

---

## Build Commands

### CP (Go)

```bash
# From repo root — use podman build directly (make build-control-plane uses wrong image name)
podman build --no-cache -f components/ambient-control-plane/Dockerfile -t localhost/ambient_control_plane:latest components

# Push to running kind cluster
CLUSTER_CTR=$(podman ps --format '{{.Names}}' | grep 'control-plane' | head -1)
podman exec ${CLUSTER_CTR} ctr --namespace=k8s.io images rm localhost/ambient_control_plane:latest 2>/dev/null || true
podman save localhost/ambient_control_plane:latest | \
  podman exec -i ${CLUSTER_CTR} ctr --namespace=k8s.io images import -
kubectl rollout restart deployment/ambient-control-plane -n ambient-code
kubectl rollout status deployment/ambient-control-plane -n ambient-code --timeout=90s
```

### Runner (Python)

```bash
# From components/runners/ — always --no-cache to pick up Python source changes
cd components/runners && podman build --no-cache -t localhost/vteam_claude_runner:latest -f ambient-runner/Dockerfile .

# Push to running kind cluster
CLUSTER_CTR=$(podman ps --format '{{.Names}}' | grep 'control-plane' | head -1)
podman save localhost/vteam_claude_runner:latest | \
  podman exec -i ${CLUSTER_CTR} ctr --namespace=k8s.io images import -
# No deployment restart needed — new image used on next session pod creation
```

### Lint + Test

```bash
# CP
cd components/ambient-control-plane
go build ./...
go vet ./...
gofmt -l .

# Runner
cd components/runners/ambient-runner
uv venv && uv pip install -e .
python -m pytest tests/
ruff format .
ruff check .
```

---

## Gap Table (Current)

```
ITEM                                    COMPONENT    STATUS      GAP
─────────────────────────────────────────────────────────────────────
assistant payload → plain string        Runner       closed      GRPCMessageWriter._write_message() fixed (Wave 1)
reasoning leaks into DB record          Runner       closed      reasoning stays in /events SSE only (Wave 1)
GET /events/{thread_id}                 Runner       closed      endpoints/events.py added
GET /sessions/{id}/events (proxy)       api-server   open        not in platform-api-server codebase yet
acpctl session events <id>              CLI          open        no command
Namespace delete RBAC                   CP manifests closed      delete added to namespaces ClusterRole (Wave 2)
```

---

## Workflow Steps

### Step 1 — Acknowledge Iteration

- [ ] Read `control-plane.spec.md` top to bottom
- [ ] Note the gap table above
- [ ] Confirm the running kind cluster name: `podman ps | grep kind | grep control-plane`
- [ ] Confirm CP is running: `kubectl get deploy ambient-control-plane -n ambient-code`

### Step 2 — Read the Spec

Read `control-plane.spec.md` in full. Hold in working memory:

- The two message streams and what belongs in each
- The proposed `GRPCMessageWriter` payload change
- The `GET /events/{thread_id}` runner endpoint (already done)
- The `GET /sessions/{id}/events` api-server proxy (not yet done)
- The namespace delete RBAC gap

### Step 3 — Current Gap Table

Use the table above. Update it as items close.

### Step 4 — Waves

#### Wave 1 — Runner: Fix assistant payload (no upstream dependency)

**File:** `components/runners/ambient-runner/ambient_runner/bridges/claude/grpc_transport.py`

**Target:** `GRPCMessageWriter._write_message()`

**What to do:**

Replace the full JSON blob with the assistant text only:

```python
async def _write_message(self, status: str) -> None:
    if self._grpc_client is None:
        logger.warning(
            "[GRPC WRITER] No gRPC client — cannot push: session=%s",
            self._session_id,
        )
        return

    assistant_text = next(
        (
            m.get("content", "")
            for m in self._accumulated_messages
            if m.get("role") == "assistant"
        ),
        "",
    )

    if not assistant_text:
        logger.warning(
            "[GRPC WRITER] No assistant message in snapshot: session=%s run=%s messages=%d",
            self._session_id,
            self._run_id,
            len(self._accumulated_messages),
        )

    logger.info(
        "[GRPC WRITER] PushSessionMessage: session=%s run=%s status=%s text_len=%d",
        self._session_id,
        self._run_id,
        status,
        len(assistant_text),
    )

    self._grpc_client.session_messages.push(
        self._session_id,
        event_type="assistant",
        payload=assistant_text,
    )
```

**Acceptance:**
- Create a session, send a message, check `acpctl session messages <id> -o json`
- `event_type=assistant` payload is plain text, not JSON
- `reasoning` content is absent from the DB record
- CLI `-f` can display it alongside `event_type=user` without JSON parsing

**Build + push runner image after this change.**

---

#### Wave 2 — CP Manifests: Namespace delete RBAC

**Files:** `components/manifests/base/` (or wherever CP RBAC is defined)

**What to do:**

Find the CP ClusterRole and add `delete` on `namespaces`:

```yaml
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list", "create", "delete"]
```

**Verify:**

After deploy, delete a session and confirm namespace is removed:

```bash
acpctl delete session <id>
kubectl get ns  # should not show the session namespace
```

---

#### Wave 3 — api-server: `GET /sessions/{id}/events` proxy

**Repo:** `platform-api-server` (separate repo — file this as a Wave 4 BE item in the ambient-model guide)

**What to do:**

In `components/ambient-api-server/plugins/sessions/`:

1. Add `StreamRunnerEvents` handler to `handler.go`:

```go
func (h *sessionHandler) StreamRunnerEvents(w http.ResponseWriter, r *http.Request) {
    id := mux.Vars(r)["id"]
    session, err := h.sessionSvc.Get(r.Context(), id)
    if err != nil || session.KubeCrName == nil || session.KubeNamespace == nil {
        w.WriteHeader(http.StatusNotFound)
        return
    }

    runnerURL := fmt.Sprintf(
        "http://session-%s.%s.svc.cluster.local:8001/events/%s",
        *session.KubeCrName, *session.KubeNamespace, *session.KubeCrName,
    )

    req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, runnerURL, nil)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    req.Header.Set("Accept", "text/event-stream")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        w.WriteHeader(http.StatusBadGateway)
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

2. Register in `plugin.go`:

```go
sessionsRouter.HandleFunc("/{id}/events", sessionHandler.StreamRunnerEvents).Methods(http.MethodGet)
```

3. Add to `openapi/openapi.sessions.yaml`:

```yaml
/sessions/{id}/events:
  get:
    summary: Stream live AG-UI events from runner pod
    description: |
      SSE stream of all AG-UI events for the active run. Proxies the runner pod's
      /events/{thread_id} endpoint. Ephemeral — no replay. Ends when RUN_FINISHED
      or RUN_ERROR is received, or the client disconnects.
    parameters:
      - name: id
        in: path
        required: true
        schema:
          type: string
    responses:
      '200':
        description: SSE event stream
        content:
          text/event-stream:
            schema:
              type: string
      '404':
        description: Session not found
      '502':
        description: Runner pod not reachable
```

**Acceptance:**

```bash
# With a running session and active run:
curl -N http://localhost:8000/api/ambient/v1/sessions/{id}/events
# Should stream AG-UI events until RUN_FINISHED
```

---

#### Wave 4 — CLI: `acpctl session events`

**Repo:** `platform-control-plane/components/ambient-cli/`

**What to do:**

Add `eventsCmd` to `cmd/acpctl/session/`:

```go
var eventsCmd = &cobra.Command{
    Use:   "events <session-id>",
    Short: "Stream live AG-UI events from an active session run",
    Args:  cobra.ExactArgs(1),
    RunE:  runEvents,
}

func runEvents(cmd *cobra.Command, args []string) error {
    sessionID := args[0]
    client := // get SDK client
    ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
    defer cancel()

    url := fmt.Sprintf("%s/api/ambient/v1/sessions/%s/events",
        client.BaseURL(), url.PathEscape(sessionID))

    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    req.Header.Set("Accept", "text/event-stream")
    req.Header.Set("Authorization", "Bearer "+client.Token())

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    scanner := bufio.NewScanner(resp.Body)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "data: ") {
            printEventLine(cmd, line[6:])
        }
    }
    return scanner.Err()
}
```

Register in `cmd.go`:
```go
Cmd.AddCommand(eventsCmd)
```

**Acceptance:**

```bash
acpctl session events <id>
# Shows tokens streaming as Claude responds
# Exits on RUN_FINISHED or Ctrl+C
```

---

## Known Invariants

These rules apply to all CP and Runner changes:

**CP:**
- Never call `panic()` — return `fmt.Errorf` with context
- All K8s child resources (pod, secret, service, service account) must have session labels (`sessionLabels(session.ID, session.ProjectID)`)
- `imagePullPolicy` is `IfNotPresent` for `localhost/` images, `Always` otherwise — kind's containerd has no registry at localhost so `Always` causes `ErrImagePull`
- Phase updates via `sdk.Sessions().UpdateStatus()` — never write K8s CR status directly
- Use `k8serrors.IsAlreadyExists(err)` and `k8serrors.IsNotFound(err)` for idempotent operations

**Runner:**
- `_setup_platform()` is called eagerly in lifespan when `AMBIENT_GRPC_URL` is set — do not call it again in `bridge.run()`
- `_active_streams` is the SSE fan-out dict — always `pop` on exit regardless of how the turn ends
- gRPC push errors are logged and swallowed — never propagate to `bridge.run()` caller
- `ruff format .` and `ruff check .` must pass before pushing runner image

---

## Verification Playbook

After any wave:

```bash
# 1. Create a test session
acpctl create session --project foo --name verify-N "what is 2+2?"

# 2. Watch CP logs for provisioning
kubectl logs -n ambient-code deployment/ambient-control-plane -f --tail=20

# 3. Watch runner logs
POD=$(kubectl get pods -n foo -l ambient-code.io/session-id=<id> -o name | head -1)
kubectl logs -n foo $POD -f

# 4. Check messages in DB
acpctl session messages <id> -o json

# 5. Verify assistant payload is plain text (Wave 1 acceptance)
acpctl session messages <id> -o json | python3 -c "
import json, sys
msgs = json.load(sys.stdin)
for m in msgs:
    print(m['event_type'], repr(m['payload'][:80]))
"
```

Expected after Wave 1:
```
user 'what is 2+2?'
assistant '2+2 equals 4.'
```

Not:
```
assistant '{"run_id": "...", "status": "completed", "messages": [...]}'
```

---

## Mandatory Image Push Playbook

After every code change, run this sequence before testing:

```bash
# Find cluster container name
CLUSTER_CTR=$(podman ps --format '{{.Names}}' | grep 'control-plane' | head -1)
echo "Cluster container: $CLUSTER_CTR"

# Build runner (always --no-cache to pick up Python source changes)
cd components/runners && podman build --no-cache -t localhost/vteam_claude_runner:latest -f ambient-runner/Dockerfile . && cd ../..
podman save localhost/vteam_claude_runner:latest | \
  podman exec -i ${CLUSTER_CTR} ctr --namespace=k8s.io images import -

# Build CP (always --no-cache to pick up Go source changes)
podman build --no-cache -f components/ambient-control-plane/Dockerfile -t localhost/ambient_control_plane:latest components
# Remove old image from containerd before importing (prevents stale digest)
podman exec ${CLUSTER_CTR} ctr --namespace=k8s.io images rm localhost/ambient_control_plane:latest 2>/dev/null || true
podman save localhost/ambient_control_plane:latest | \
  podman exec -i ${CLUSTER_CTR} ctr --namespace=k8s.io images import -
kubectl rollout restart deployment/ambient-control-plane -n ambient-code
kubectl rollout status deployment/ambient-control-plane -n ambient-code --timeout=90s

# Verify CP pod is running the new digest
kubectl get pod -n ambient-code -l app=ambient-control-plane \
  -o jsonpath='{.items[0].status.containerStatuses[0].imageID}'
```

Runner image changes take effect on the next new session pod — no restart needed.

**Image names (actual deployment):**
- CP deployment image: `localhost/ambient_control_plane:latest`
- Runner pod image: `localhost/vteam_claude_runner:latest`
- `make build-control-plane` builds `localhost/vteam_control_plane:latest` — **wrong name**, use the `podman build` command above instead

---

## Run Log

### Run 1 — 2026-03-22

**Status:** Spec and guide written. Wave 1 (assistant payload) queued. `GET /events/{thread_id}` already implemented.

**Gap table at start:**

```
ITEM                                    COMPONENT    STATUS
GET /events/{thread_id}                 Runner       closed (endpoints/events.py)
assistant payload → plain string        Runner       open
GET /sessions/{id}/events (proxy)       api-server   open
acpctl session events <id>              CLI          open
Namespace delete RBAC                   CP manifests open
```

**Lessons:**
- Runner image must be rebuilt and pushed to kind for every Python change — no hot reload in pods
- `make build-runner` must be run from the repo root (not the component dir)
- kind cluster name is `ambient-main` (not derived from branch name) — always verify with `podman ps`
- `acpctl session messages -f` now shows assistant payloads as raw JSON — Wave 1 will fix this

### Run 2 — 2026-03-22

**Status:** Wave 1 + Wave 2 complete.

**Changes:**
- `grpc_transport.py`: `_write_message()` now pushes plain assistant text only; `json` import removed; ruff clean
- `components/manifests/base/rbac/control-plane-clusterrole.yaml`: added `delete` to namespaces verbs

**Gap table after Run 2:**

```
ITEM                                    COMPONENT    STATUS
GET /events/{thread_id}                 Runner       closed
assistant payload → plain string        Runner       closed (Wave 1)
Namespace delete RBAC                   CP manifests closed (Wave 2)
GET /sessions/{id}/events (proxy)       api-server   open
acpctl session events <id>              CLI          open
```

**Next steps:**
- Build + push runner image: `make build-runner` then push to kind
- Apply manifests for RBAC fix: `kubectl apply -f components/manifests/base/rbac/control-plane-clusterrole.yaml`
- Verify: create session, check `acpctl session messages <id> -o json` — assistant payload should be plain text
