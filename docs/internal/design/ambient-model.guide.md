# Ambient Model: Implementation Guide

**Date:** 2026-03-20
**Status:** Living Document — updated continuously as the workflow is executed and improved
**Spec:** `ambient-model.spec.md` — all Kinds, fields, relationships, API surface, CLI commands
**Run Logs:** `ambient-model.runlog.N.md` — one file per execution; gap tables, fixes applied, stopped-at state

---

## This Document Is Iterative

This document is updated as the workflow runs. Each time the workflow is invoked, start from the top, follow the steps, and update this document with what you learned — what worked, what broke, what the step actually requires in practice.

**The goal is convergence, not perfection on the first run.** Expect failures. Expect missing steps. Expect that the workflow itself needs fixing. When something breaks, fix the workflow doc before moving on.

> We start from the top each time. We update as we go. We run it until it Just Works™.

---

## Overview

This document describes a reusable autonomous workflow for implementing changes to the Ambient platform. The workflow is spec-driven: the data model doc is the source of truth, and agents reconcile code status against it, plan implementation work, and execute in parallel across components.

Each invocation starts from Step 1 and works through the steps in order. Steps are updated to reflect reality as it is discovered. The workflow does not require a clean-slate implementation — it is designed to run repeatedly until code and spec converge.

---

## The Pipeline

Changes flow downstream in a fixed dependency order:

```
Spec (ambient-model.spec.md)
  └─► API (openapi.yaml)
        └─► SDK Generator
              └─► Go SDK (types, builders, clients)
                    ├─► BE  (REST handlers, DAOs, migrations)
                    ├─► CLI (commands, output formatters)
                    ├─► CP  (gRPC middleware, interceptors)
                    ├─► Operator (CRD reconcilers, Job spawning)
                    ├─► Runners (Python SDK calls, gRPC push)
                    └─► FE  (TypeScript API layer, UI components)
```

Each stage depends on the stage above it being settled. Agents must not implement downstream work against an unstable upstream.

---

## Agents and Ownership

| Agent      | Component                                | Primary Responsibility                                                        |
| ---------- | ---------------------------------------- | ----------------------------------------------------------------------------- |
| `Lead`     | *(orchestrator)*                         | Reads spec, produces gap table, assigns waves to component agents via inbox, gates downstream waves on upstream completion, monitors progress, coordinates Reviewer |
| `API`      | `components/ambient-api-server/`         | OpenAPI spec, REST route registration, handler stubs, plugins, DAOs, service logic, migrations, gRPC presenters |
| `Public`   | `components/public-api/`                 | Stateless HTTP gateway — proxies to ambient-api-server; no direct K8s access  |
| `Reviewer` | *(cross-cutting)*                        | Standards-driven code review via `/amber.review`; evaluates spec alignment, security, workflow compliance, architecture across all components |
| `SDK`      | `components/ambient-sdk/`                | SDK generator, generated Go types, client methods                             |
| `BE`       | `components/backend/`                    | Original Gin/K8s backend — owns K8s CR management, runner proxying, websocket/AG-UI bridge |
| `CP`       | *(standalone service)*                   | gRPC fan-out bridge between api-server and runner pods; session orchestration; runner compatibility contract |
| `CLI`      | `components/ambient-cli/`                | `acpctl` commands, table formatters, TUI                                      |
| `Operator` | `components/operator/`                   | CRD reconcilers, Kubernetes Job lifecycle                                     |
| `Runners`  | `components/runners/ambient-runner/`     | Python runner, gRPC push, AG-UI event emission                                |
| `FE`       | `components/frontend/`                   | React components, API service layer, queries                                  |

> **Scope note:** `API` owns all of `components/ambient-api-server/` — including `plugins/`, `pkg/`, `openapi/`, proto definitions, and gRPC middleware. `BE` owns `components/backend/` (the original Gin/K8s backend with direct cluster access). These are separate services with separate databases and separate deployment units. Do not conflate them.

---

## Workflow Steps

> **Each invocation: start from Step 1. Update this document before moving to the next step if anything is wrong or missing.**

### Step 1 — Acknowledge Iteration

Before doing anything else, internalize that this run may not succeed. The workflow is the product. If a step fails, edit this document to capture the failure and what the step actually requires, then retry.

Checklist:

- [ ]  Read this document top to bottom
- [ ]  Read the most recent run log (`ambient-model.runlog.N.md`)
- [ ]  Confirm you are working on the correct branch and project
- [ ]  Verify the kind cluster name: `podman ps | grep kind` (do not assume — cluster name drifts)

### Step 2 — Read the Spec

Read `docs/internal/design/ambient-model.spec.md` in full.

Extract and hold in working memory:

- All entities and their fields
- All relationships
- All API routes
- CLI table (✅ implemented / 🔲 planned)
- Design decisions
- Ignition context assembly order

This is the **desired state**. Everything else is measured against it.

### Step 3 — Assess What Has Changed

Compare the spec against the current state of the code. For each component, ask:

| Component    | What to check                                                                                         |
| ------------ | ----------------------------------------------------------------------------------------------------- |
| **API**      | Does `openapi/openapi.yaml` have all spec entities, routes, and fields? **Read the actual fragments.**|
| **SDK**      | Do generated types/builders/clients exist for all spec entities?                                      |
| **BE**       | Read `plugins/<kind>/model.go` for every Kind. Compare field-by-field against the Spec. Drift here is the most common source of gaps. |
| **CP**       | Does middleware handle new RBAC scopes and auth requirements?                                         |
| **CLI**      | Does `acpctl` implement every route marked ✅ in the spec CLI table?                                  |
| **Operator** | Do CRD reconcilers handle Agent-scoped session ignition?                                              |
| **Runners**  | Does the runner drain inbox at ignition and push correct event types?                                 |
| **FE**       | Do API service layer, queries, and components exist for all new entities?                             |

**The gap table must compare Spec against every component simultaneously.** A field removal touches API, SDK, BE (model + migration), and CLI — all four must be in the gap table from the start. Do not discover mid-wave that the CLI still has a flag the API no longer accepts.

Produce a gap table:

```
ENTITY          COMPONENT   STATUS      GAP
Agent           API         missing     no routes in openapi.yaml
Agent           SDK         missing     no generated type
Agent           BE          missing     no DAO, no handler, no migration
Inbox           BE          missing     no DAO, no handler
Inbox           CLI         missing     no acpctl commands
Session.prompt  BE          present     —
```

The gap table is the implementation backlog. The current run log (`ambient-model.runlog.N.md`) captures the last known state. Start from there.

### Step 4 — Break It Into Work by Agent

Decompose the gap table into per-agent work items, sequenced by pipeline order:

**Wave 1 — Spec consensus** (no code; human approval)

- Confirm gap table is complete and agreed upon
- Freeze spec for this run

**Wave 2 — API** (gates everything downstream)

- Update `openapi/openapi.yaml` for all new entities and routes
- Register routes in `routes.go`
- Add handler stubs (`501 Not Implemented`) to complete the surface
- **Security gate:** new routes use `environments.JWTMiddleware`; no user token logged; RBAC scopes documented in openapi
- **Acceptance:** `make test` passes, `make binary` succeeds, `make lint` clean

**Wave 3 — SDK** (gates BE, CLI, FE)

- Run SDK generator against updated `openapi.yaml`
- Commit generated types, builders, client methods
- **Verify TS and Python client paths** for any nested resource — the generator uses the first path segment as base path; nested resources require hand-written extension files (see [SDK Generator Pitfalls](#sdk-generator-pitfalls))
- **Acceptance:** `go build ./...` in go-sdk clean; Python SDK `python -m pytest tests/` passes

**Wave 4 — BE + CP** (parallel after Wave 3)

- **BE**: migrations, DAOs, service logic, gRPC presenters
- **CP**: runner fan-out compatibility verified (see runner compat contract below)
- **Security gate:** all handler paths check user token via service layer; no token values in logs; input validated before DB write
- **Acceptance:** `make test` passes, `go vet ./... && golangci-lint run` clean

**Wave 5 — CLI + Operator + Runners** (parallel after Wave 3 + BE)

- **CLI**: implement all 🔲 commands that are now unblocked
- **Operator**: CRD reconciler updates for Agent ignition; `go vet ./... && golangci-lint run` clean; tested in kind cluster
- **Runners**: inbox drain at ignition, correct event types; `python -m pytest tests/` passes
- **Security gate (Operator):** all Job pods set `SecurityContext` with `AllowPrivilegeEscalation: false`, capabilities dropped; OwnerReferences set on all child resources

**Wave 6 — FE** (after Wave 4 BE)

- API service layer and React Query hooks for new entities
- UI components: Agent list, Inbox, Project Home
- **FE hard rules:** zero `any` types; Shadcn UI components only; all data via React Query (no manual `fetch`); `type` over `interface`; single-use components colocated with their page
- **Security gate:** no tokens or credentials in frontend state or logs; all API errors surface structured messages, not raw server responses
- **Acceptance:** `npm run build` — 0 errors, 0 warnings

**Wave 7 — Integration**

- End-to-end smoke: ignite Agent → watch session stream → send inbox message
- `make test` and `make lint` across all components
- **Final step:** push new image to kind cluster (see [Mandatory Image Push Playbook](#mandatory-image-push-playbook))

Each wave is a gate. Record what actually happened in a new run log file.

### Step 5 — Send Messages to Each Agent

Post work assignments to each agent via inbox. Use `acpctl send` or the API directly. Keep messages self-contained — each message should include what to do, where to find the spec, and what done looks like.

```sh
acpctl send api --body "Wave 2: Update openapi.yaml per ambient-model.spec.md. Add Agent, Inbox routes. Register in routes.go. Add 501 stubs. Done = openapi.yaml compiles and all spec routes exist."
acpctl start api

acpctl send sdk --body "Wave 3 (after API merges): Regen SDK from updated openapi.yaml. Commit generated types and clients. Done = all spec entities have Go types and client methods."
acpctl start sdk
```

Continue for each wave. Do not ignite downstream agents until upstream wave is merged.

Monitor progress via `acpctl get sessions -w`.

### Step 6 — Ascertain the Work

After each agent reports done:

- Read the PR or the agent's check-in
- Re-run the gap table (Step 3) for that component only
- If gaps remain, send a follow-up message and re-ignite
- If clean, mark that wave item as complete and proceed to the next wave

When all waves are complete and the gap table is empty, the workflow run is done. **Write a new `ambient-model.runlog.N.md`** with the gap table, fixes applied, and stopped-at state.

---

## Invocation

### Manual (current)

```sh
# 1. Human reads spec and writes gap list manually
# 2. Human posts inbox messages to each agent
# 3. Human ignites agents one at a time or in parallel
acpctl send api --body "Wave 2: update openapi.yaml per ambient-model.spec.md spec..."
acpctl start api
```

### Assisted (near-term)

An Overlord agent is ignited with a prompt scoped to orchestration. It reads the spec, runs the assessment, posts inbox messages to each component agent, and monitors completion via session messages.

```sh
acpctl send overlord --body "Run spec-to-code reconciliation for ambient-model.spec.md. Assign waves 2-6 to component agents. Report back when wave 2 (API) is complete."
acpctl start overlord
```

### Autonomous (target)

The Project Home dashboard shows all Agents and their inbox counts. A standing Overlord agent monitors for spec changes (new commits to `docs/internal/design/`) and automatically invokes the workflow, assigning work and gating downstream waves on upstream completions.

---

## CP ↔ Runner Compatibility Contract

The Control Plane (CP) is a **fan-out multiplexer** — it sits between the api-server and runner pods. Multiple clients can watch the same session; the runner pushes once. CP must preserve these invariants on every change:

| Concern | Runner expects | CP must preserve |
|---|---|---|
| Session start | Job pod scheduled by operator | CP does not reschedule |
| Event emission | Runner pushes AG-UI events via gRPC | CP forwards in order, never drops |
| `RUN_FINISHED` | Emitted once at end | CP forwards exactly once — not duplicated |
| `MESSAGES_SNAPSHOT` | Emitted periodically | CP forwards in order |
| Token | Runner receives token from K8s secret | CP does not touch runner token |
| Non-JWT tokens | test-user-token has no username claim | CP skips ownership check when JWT username absent |

**Runner compat test (run before any CP PR):**
```bash
acpctl create session --project my-project --name test-cp "echo hello"
acpctl session messages -f --project my-project test-cp
```
Expected: `RUN_STARTED` → `TEXT_MESSAGE_CONTENT` (tokens) → `RUN_FINISHED`. No connection errors, no dropped events, no duplicate `RUN_FINISHED`.

---

## Runner Pod Addressing

The api-server does **not** have a built-in proxy to runner pods. Runner pods are addressed by Kubernetes cluster-internal DNS, constructed at request time from the session's stored fields:

```
http://session-{KubeCrName}.{KubeNamespace}.svc.cluster.local:8001
```

The `Session` model stores `KubeCrName` and `KubeNamespace` — both are available from the DB. The runner listens on port `8001` (set via `AGUI_PORT` env var by the operator; default in runner code is `8000` but the operator overrides it).

This pattern is used by `components/backend/websocket/agui_proxy.go` (the V1 backend). The ambient-api-server does not currently proxy to runner pods — any new proxy endpoint must add this logic.

### Implementing `GET /sessions/{id}/events` (Runner SSE Proxy)

This endpoint proxies the runner pod's `GET /events/{thread_id}` SSE stream through to the client. The runner's `/events/{thread_id}` endpoint:

- Registers an asyncio queue into `bridge._active_streams[thread_id]`
- Streams every AG-UI event as SSE until `RUN_FINISHED` / `RUN_ERROR` or client disconnect
- Sends 30s keepalive pings (`: ping`) to hold the connection
- Cleans up the queue on exit regardless of how it ends

`thread_id` in the runner corresponds to the session ID (the value stored in `Session.KubeCrName`).

**Implementation steps for BE agent (Wave 4):**

1. Look up session from DB by `id` path param — get `KubeCrName` and `KubeNamespace`
2. Construct runner URL: `http://session-{KubeCrName}.{KubeNamespace}.svc.cluster.local:8001/events/{KubeCrName}`
3. Open an HTTP GET to that URL with `Accept: text/event-stream`
4. Set response headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`, `X-Accel-Buffering: no`
5. Stream the runner's SSE body directly to the client response writer, flushing after each `\n\n` boundary
6. On runner disconnect or `RUN_FINISHED` / `RUN_ERROR` event, close the client stream

**Key differences from `/sessions/{id}/messages`:**

| | `/messages` | `/events` |
|---|---|---|
| Source | api-server DB + gRPC fan-out | Runner pod SSE (live only) |
| Persistence | Persisted; supports replay from `seq=0` | Ephemeral; runner-local in-memory queue |
| Reconnect | Resume via `Last-Event-ID` / `after_seq` | No replay; live only |
| Keepalive | 30s ticker `: ping` | 30s ticker from runner; proxy must pass through |

**SSE proxy pattern** (follow `plugins/sessions/message_handler.go:streamMessages` for SSE writer setup):

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

    io.Copy(w, resp.Body) // SSE frames pass through verbatim
    if f, ok := w.(http.Flusher); ok {
        f.Flush()
    }
}
```

Register in `plugin.go`:
```go
sessionsRouter.HandleFunc("/{id}/events", eventsHandler.StreamRunnerEvents).Methods(http.MethodGet)
```

---

## Constraints

- **Pipeline order is strict**: no downstream agent starts a wave until the upstream wave is merged and SDK is regenerated
- **One active session per agent**: ignition is idempotent; do not force-restart a running agent
- **Spec is frozen during execution**: no spec changes while a wave is in flight; queue changes for next cycle
- **PRs are atomic per wave per component**: one PR per agent per wave; avoids merge conflicts across components
- **Agents stay in their lane**: cross-component edits require a spec change and a new wave assignment

---

## Code Generation

All new Kinds must use the generator and templates in `components/ambient-api-server/templates/`. Do not hand-write plugin boilerplate.

```bash
go run ./scripts/generator.go \
  --kind Agent \
  --fields "project_id:string:required,name:string:required,prompt:string,current_session_id:string" \
  --project ambient \
  --repo github.com/ambient-code/platform/components \
  --library github.com/openshift-online/rh-trex-ai
```

Templates of interest:
| Template | Produces |
|---|---|
| `generate-openapi-kind.txt` | OpenAPI paths + schemas for a Kind |
| `generate-plugin.txt` | `plugin.go` init registration |
| `generate-dao.txt` | DAO interface + implementation |
| `generate-services.txt` | Service layer |
| `generate-handlers.txt` | HTTP handlers |
| `generate-presenters.txt` | OpenAPI ↔ model converters |
| `generate-migration.txt` | Gormigrate migration |
| `generate-mock.txt` | Mock DAO for tests |

The SDK generator (`components/ambient-sdk/`) consumes the updated `openapi.yaml` — run it after any openapi change.

---

## SDK Generator Pitfalls

These issues have caused failures in past runs. Check for them after every generator invocation.

**Nested resource base path (TS + Python):** The generator uses the first path segment of a resource's routes as the base path for generated client methods. For resources nested under `/projects/{id}/agents/...`, the generator emits `/projects` as the base path for all methods — wrong. Fix: write hand-crafted extension files that override the generated class.

- Go SDK: `go-sdk/client/agent_extensions.go` — non-CRUD methods (ignite, sessions, inbox)
- TS SDK: `ts-sdk/src/project_agent_api.ts`, `ts-sdk/src/inbox_message_api.ts` — complete rewrites
- After fixing the generator, delete the extension files and re-verify

**Auxiliary DTO schemas in sub-spec files:** Each `openapi.*.yaml` sub-spec file must have exactly one primary resource schema. That schema must use `allOf`. Auxiliary DTOs (request/response bodies, view models) that don't end in `List`, `PatchRequest`, or `StatusPatchRequest` must live in `openapi.yaml` main components — not in sub-spec files. The generator picks schemas alphabetically; if the first candidate lacks `allOf`, the entire parse fails.

**Generator directory naming:** Generator creates directory named `{kindLowerPlural}`. For `InboxMessage` this is `inboxMessages`, not `inbox`. Copy and rename manually when the desired package name differs.

**Generated handler variable names:** `mux.Vars` key must match the route variable name. Nested routes use `{msg_id}`, not `{id}` — generated handlers always use `{id}`. Fix after generation.

**Generated middleware import:** `RegisterRoutes` callback must use `environments.JWTMiddleware`, not `auth.JWTMiddleware`. Generated code always gets this wrong. Fix after generation.

**Generated integration tests for nested routes:** Integration tests generated by the code generator reference flat openapi client methods that don't exist when routes are nested. Stub with `t.Skip` and mark for future update.

---

## Known Code Invariants

These rules were established by fixing bugs found in production. Violating them causes panics, security holes, or incorrect behavior. Apply them when reading or writing any handler/presenter/SDK code.

**gRPC presenter completeness:** `grpc_presenter.go` `sessionToProto()` must map every field that exists in both the DB model and proto message. Missing fields cause downstream consumers (CP, operator) to receive zero values silently.

**Inbox handler scoping:** `inbox/handler.go` List must always inject `agent_id = 'X'` from URL `pa_id` into the TSL search filter. Never return cross-agent data. `listArgs.Search` is `string`, not `*string` — use empty-string checks, not nil checks.

**Inbox handler agent_id enforcement:** `inbox/handler.go` Create must set `inboxMessage.AgentId = mux.Vars(r)["pa_id"]` from the URL, ignoring any `agent_id` in the request body. Prevents body spoofing.

**Inbox presenter nil safety:** Nil-guard `UpdatedAt` independently from `CreatedAt`. They can be nil independently; treating them as a pair causes panics.

**InboxMessagePatchRequest scope:** Only `Read *bool` is permitted. No other fields. Prevents privilege escalation via PATCH.

**IgniteResponse field name:** `ignition_prompt` is the canonical field name across openapi, Go SDK, TS SDK. Not `ignition_context`.

**Ignite HTTP status:** `Ignite` returns HTTP 201 on new session creation, HTTP 200 on re-ignite (session already active). SDK `doMultiStatus()` accepts both.

**Nested resource URL encoding:** All nested resource URLs must use `encodeURIComponent` (TS) / `url.PathEscape` (Go) on every path segment.

**proto field addition:** Edit `.proto` → `make proto` → verify `*.pb.go` regenerated → wire through presenter. Do not edit `*.pb.go` directly.

---

## Mandatory Image Push Playbook

This sequence is required after every wave or bug-fix batch. Do not mark a wave complete until the rollout succeeds.

```bash
# 0. Find the running cluster name
CLUSTER=$(podman ps --format '{{.Names}}' | grep 'kind' | grep 'control-plane' | sed 's/-control-plane//')
echo "Cluster: $CLUSTER"

# 1. Build without cache (cache misses source changes when go.mod/go.sum unchanged)
podman build --no-cache -t localhost/vteam_api_server:latest components/ambient-api-server

# 2. Load into kind via ctr import (kind load docker-image fails with podman localhost/ prefix)
podman save localhost/vteam_api_server:latest | \
  podman exec -i ${CLUSTER}-control-plane ctr --namespace=k8s.io images import -

# 3. Restart and verify
kubectl rollout restart deployment/ambient-api-server -n ambient-code
kubectl rollout status deployment/ambient-api-server -n ambient-code --timeout=60s
```

**Why `kind load docker-image` fails with podman:** It calls `docker inspect` internally and cannot resolve `localhost/` prefix images. The `podman save | ctr import` approach bypasses kind's image loader and writes directly to containerd's k8s.io namespace inside the control-plane container.

**Why `--no-cache` is required:** The Dockerfile copies source in layers. If `go.mod`/`go.sum` are unchanged, the `go build` step hits cache and emits the old binary.

---

## gRPC Local Port-Forward

The Go SDK derives the gRPC address from the REST base URL hostname + port `9000`. When pointing at `http://127.0.0.1:8000`, it derives `127.0.0.1:9000`. If local port 9000 is occupied (e.g. minio), gRPC streaming fails.

Fix for local development:
```bash
kubectl port-forward svc/ambient-api-server 19000:9000 -n ambient-code &
export AMBIENT_GRPC_URL=127.0.0.1:19000
```

The TUI's `PortForwardEntry` for gRPC maps to local port `19000` — use this consistently.

Long-term: add `grpc_url` to `pkg/config/config.go` so it can be set once via `acpctl config set grpc_url 127.0.0.1:19000`.

---

## Build and Test by Component

Each component has its own build, generate, test, and lint commands. Run these before opening a PR for that component.

### ambient-api-server (`components/ambient-api-server/`)

```bash
make generate          # Regenerate OpenAPI Go client from openapi/*.yaml (runs containerized generator)
make binary            # Compile the ambient-api-server binary
make test              # Integration tests — spins up testcontainer PostgreSQL (AMBIENT_ENV=integration_testing)
make test-integration  # Run only ./test/integration/... package
make proto             # Regenerate gRPC stubs from proto/
make proto-lint        # Lint proto definitions
go fmt ./...           # Format Go source
golangci-lint run      # Lint
```

> `make generate` must be run after any change to `openapi/*.yaml`. It emits to `pkg/api/openapi/` — never edit that directory manually.

### ambient-sdk (`components/ambient-sdk/`)

```bash
make build-generator   # Compile the SDK generator binary
make generate-sdk      # Run generator against openapi.yaml → Go + Python + TypeScript SDKs
make verify-sdk        # generate-sdk + compile-check all three outputs
```

> `make generate-sdk` must be run after `make generate` on the API server. SDK consumers (BE, CLI, FE) depend on this output.

### ambient-cli (`components/ambient-cli/`)

```bash
make build             # Compile the acpctl binary
make test              # go test -race ./...
make lint              # go vet + golangci-lint run
make fmt               # gofmt -l -w .
```

### operator (`components/operator/`)

No per-component Makefile. Build and test via:

```bash
cd components/operator
go build ./...
go test ./...
go fmt ./...
golangci-lint run
```

Top-level: `make build-operator` builds the container image.

### runners/ambient-runner (`components/runners/ambient-runner/`)

No Makefile — managed by `uv` and `pyproject.toml`:

```bash
cd components/runners/ambient-runner
uv venv && uv pip install -e .   # Set up virtualenv
python -m pytest tests/          # Run tests
ruff format .                    # Format
ruff check .                     # Lint
```

### frontend (`components/frontend/`)

No Makefile — npm scripts only:

```bash
cd components/frontend
npm run build          # Production build — must pass 0 errors, 0 warnings
npm run lint           # ESLint
npm run test:unit      # Vitest unit tests
npm run test:unit:coverage  # Vitest with coverage
```

### Top-level orchestration

```bash
make lint              # All linters via pre-commit (all files)
make test-all          # CLI tests + local smoke tests
make build-all         # All container images
make kind-up           # Start local kind cluster and deploy
make test-e2e-local    # Full e2e: kind-up + Cypress + cleanup
```

---

## Artifacts

| Artifact              | Location                                             | Owner             |
| --------------------- | ---------------------------------------------------- | ----------------- |
| Spec                  | `docs/internal/design/ambient-model.spec.md`         | Human / consensus |
| Run logs              | `docs/internal/design/ambient-model.runlog.N.md`     | Agent / human     |
| OpenAPI spec          | `components/ambient-api-server/openapi/openapi.yaml` | API agent         |
| Generated SDK         | `components/ambient-sdk/go-sdk/`                     | SDK agent         |
| Wave PRs              | GitHub, tagged by wave and component                 | Component agents  |

---

## Lessons Learned (Run Log — 2026-03-22)

### The Spec Can Lag the Code

The spec CLI table had every Agent and Inbox command marked 🔲 planned. The code had 456-line `agent/cmd.go` and 301-line `inbox/cmd.go` — fully implemented. **Always verify with `wc -l` and `go build` before assuming a gap is real.** The spec table is maintained manually; the code moves faster.

**Fix applied:** Added the Implementation Coverage Matrix to the end of the spec as the authoritative cross-component index. Update it whenever code ships, not after the next review cycle.

### SDK Extension Methods Must Be Symmetric

The `apply` command needed `GetInProject`, `ListInboxInProject`, and `SendInboxInProject` on `ProjectAgentAPI` — methods the generator would never emit for a nested resource. These had to be hand-written in `agent_extensions.go`. The pattern: any method that uses a nested URL (`/projects/{p}/agents/{a}/...`) must live in an extensions file, not in generated code.

**Rule:** When adding a new nested operation to the CLI or a new command that calls an API endpoint, check `agent_extensions.go` (or the relevant `*_extensions.go`) first. If the method isn't there, add it before writing the CLI command.

### CLI `events.go` Should Use SDK, Not Raw HTTP

The first implementation of `acpctl session events` used `net/http` directly, bypassing the SDK client (no auth header construction from config, no `X-Ambient-Project` header). This was a shortcut taken because `SessionAPI.StreamEvents` didn't exist yet.

**Fix applied:** Added `StreamEvents(ctx, sessionID) (io.ReadCloser, error)` to `session_messages.go`. Refactored `events.go` to use `connection.NewClientFromConfig()` + `client.Sessions().StreamEvents()`. Now the auth and project headers are handled consistently with all other SDK calls.

**Rule:** Never bypass the SDK client in CLI commands. If a method is missing, add it to the SDK first, then write the CLI command against it.

### `StreamEvents` Cannot Use `do()` — It Must Return the Body

The SDK's `do()` and `doMultiStatus()` methods unmarshal the response body into a typed result and close the connection. For SSE streams, you need the body open and streaming. `StreamEvents` uses `a.client.httpClient.Do(req)` directly and returns `resp.Body` as `io.ReadCloser`. The caller closes it.

This means `StreamEvents` needs access to `a.client.baseURL`, `a.client.token`, and `a.client.httpClient` — all unexported fields. Since `session_messages.go` is in the same `client` package, this works without accessors.

**Rule:** SSE / streaming endpoints require a separate implementation pattern. Do not try to fit them into `do()`. Return `io.ReadCloser` from the SDK; let the CLI layer scan it with `bufio.Scanner`.

### `gopkg.in/yaml.v3` Was Missing From CLI `go.mod`

The `apply` command imported `yaml.v3` but the CLI `go.mod` didn't declare it. The build failure message (`missing go.sum entry`) was clear but required running `go get gopkg.in/yaml.v3` to resolve. The dependency was already transitively available (via the SDK), but Go modules require explicit declaration for direct imports.

**Rule:** When adding a new file to the CLI that imports a new package, run `go build ./...` immediately. Fix `go.mod` before committing.

### Spec Coverage Matrix Is the Right Indexing Artifact

The gap between what the spec said (🔲 everywhere for agents/inbox) and what the code had (full implementations) was only discoverable by reading actual source files. An implementation coverage matrix embedded in the spec — with direct references to SDK method names and CLI commands — turns the spec into a live index that can be scanned in seconds.

**Rule:** The coverage matrix in `ambient-model.spec.md` is the primary index. Update it immediately when a component ships a feature. Do not rely on the CLI table alone — it maps REST→CLI but doesn't tell you what the SDK exposes.
