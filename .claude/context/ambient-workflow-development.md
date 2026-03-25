# Ambient Workflow Development Context

**When to load:** Writing or executing the agentic development workflow — the instruction set for building Ambient components from a Spec diff.

---

## What the Workflow Is

The Workflow is a **policy function**: given a diff between Spec and Status, it produces a deterministic, ordered task graph.

```
Workflow(Δ: Spec - Status) → TaskGraph
```

The Workflow is not a one-time script. It is a **living instruction set** that grows more complete with every run. Each time it hits an ambiguity — a case it doesn't know how to handle — that ambiguity gets resolved and written back into the Workflow as a new rule. The next agent to encounter the same case proceeds without stopping.

**The Workflow is our saving grace.** Ambient's `components/` are not abstractions. They are concrete, deterministic software. The code patterns are highly repetitive by design. The Workflow captures those patterns and applies them mechanically from the Spec.

---

## The Reconciliation Loop

```
while Spec != Status:
    Δ = compute_diff(Spec, Status)
    tasks = Workflow(Δ)

    for task in topological_order(tasks):
        execute(task)

        if ambiguity_encountered:
            surface_question()
            update_Workflow_with_answer()
            stop  ← human resolves, then re-enters loop

        update_Status()

done  ← Spec == Status
```

Every stop is an opportunity to improve the Workflow. A mature Workflow stops rarely.

---

## Workflow File

**Location:** `docs/internal/design/agent-workflow.md`

This is the canonical Workflow document. It is a step-by-step guide for how to propagate any Spec change through all Ambient layers. It reads like a runbook but reasons like a policy.

**Structure of the Workflow document:**

1. **Diff classification rules** — given a Spec change of type X, which layers are affected
2. **Layer-ordered execution** — the invariant propagation order (see below)
3. **Per-layer instructions** — concrete steps for each component
4. **Pattern library** — reusable code patterns for common operations
5. **Ambiguity log** — resolved ambiguities, so future agents don't re-encounter them

---

## The Propagation Order (invariant)

Spec changes always propagate in this order. No exceptions.

```
1. Spec          ambient-data-model.md + openapi.*.yaml
2. API Server    plugins/<kind>/ — model, dao, service, handler, presenter, migration
3. SDK           go-sdk, python-sdk, ts-sdk — regenerated from openapi.yaml
4. CLI           acpctl commands — consume Go SDK
5. Operator      reconciler logic — if new K8s resource behavior needed
6. Runner        Python runner — if new session lifecycle events needed
7. Frontend      NextJS pages + React Query hooks — consume REST API
```

Dependencies flow downward. You cannot implement layer N+1 before layer N is complete and tested.

**Why this order is correct:**

- The API Server defines the data contract. Everything downstream depends on it.
- The SDK is generated. It cannot exist before the API Server's openapi.yaml is correct.
- The CLI is built on the SDK. It cannot exist before the SDK is generated.
- The Frontend calls the REST API directly (not via SDK) but depends on the API contract.
- The Operator and Runner have their own contracts with the K8s layer and runner protocol.

---

## Context References (Building Blocks)

These context files contain the authoritative how-to for each layer. The Workflow references them by name — it does not repeat their content.

| Layer | Context File | What It Covers |
|---|---|---|
| API Server | `.claude/context/api-server-development.md` | Plugin scaffold, gRPC, openapi generation, test setup |
| SDK | `.claude/context/sdk-development.md` | Generator pipeline, all 3 languages, regen commands |
| CLI | `.claude/context/cli-development.md` | Command structure, streaming flags, SDK dependency |
| Control Plane | `.claude/context/control-plane-development.md` | Fan-out, runner compat, ownership bypass |
| Backend (K8s) | `.claude/context/backend-development.md` | Gin handlers, client-go, RBAC, K8s CRDs |
| Frontend | `.claude/context/frontend-development.md` | NextJS, React Query, Shadcn, no-any rule |
| Security | `.claude/context/security-standards.md` | Auth, token handling, RBAC patterns |
| Spec | `.claude/context/ambient-spec-development.md` | What the Spec is, what belongs in it, completeness rules |

**When a per-layer rule below says "follow the context"** — read the referenced file. The Workflow does not duplicate those instructions.

---

## Per-Layer Workflow Rules

### Layer 1: Spec

**Trigger:** Human changes `ambient-data-model.md` or `openapi.*.yaml`

**Rules:**
- Run `make generate` in `ambient-api-server` after any openapi change — regenerates `pkg/api/openapi/`
- Validate openapi.yaml is valid before proceeding
- Update `ambient-cli/README.md` if new CLI commands are implied
- Update SDK examples if new usage patterns are implied

**Acceptance:** `openapi.yaml` is valid, all affected fragments are consistent with `ambient-data-model.md`

---

### Layer 2: API Server

**Trigger:** New Kind, new field, new endpoint, new gRPC RPC

**How:** Follow `api-server-development.md` — plugin scaffold, gRPC, test setup.

**Business logic specific to Ambient:**
- Nested routes (e.g. `/{project}/agents/{id}/ignite`) require `environments.JWTMiddleware`, not `auth.JWTMiddleware`
- Generator artifact directories may be named wrong (e.g. `inboxMessages` vs `inbox`) — rename manually
- Auxiliary DTO schemas (request/response bodies that aren't `List`/`PatchRequest`/`StatusPatchRequest`) must live in `openapi.yaml` main components, not in sub-spec `openapi.*.yaml` files — the SDK generator's `inferResourceName` selects alphabetically and requires `allOf` on the first candidate
- Integration tests for nested routes reference flat client methods that don't exist — stub with `t.Skip` until SDK catches up

**Acceptance:** `make test` passes, `make binary` succeeds, `make lint` clean

---

### Layer 3: SDK

**Trigger:** Any Layer 2 change that modifies openapi.yaml

**How:** Follow `sdk-development.md` — generator pipeline, all 3 languages.

**Business logic specific to Ambient:**
- Run `make generate` in api-server first, then `make generate` in ambient-sdk
- Verify with `go build ./...` in go-sdk before declaring done
- Examples in `go-sdk/examples/` must run against the kind cluster — they are e2e tests

**Acceptance:** All SDK tests pass, go-sdk examples run against kind

---

### Layer 4: CLI

**Trigger:** New SDK capability, new user-facing command in Spec CLI table

**How:** Follow `cli-development.md` — command structure, streaming, SDK dependency.

**Business logic specific to Ambient:**
- Every ✅ entry in the Spec CLI table must have an `acpctl` command
- `session messages -f` is the canonical watch command — all new streaming commands follow this pattern
- `acpctl session follow` (if added for agent-deck integration) streams AG-UI events as terminal output

**Acceptance:** `make test` passes, command works against kind, `README.md` updated

---

### Layer 5: Operator

**Trigger:** New CRD field, new reconciler phase, new K8s resource lifecycle behavior

**How:** Follow `backend-development.md` for K8s client patterns and OwnerReferences.

**Business logic specific to Ambient:**
- CRD changes: `components/manifests/base/crds/`
- Reconciler phases: `internal/controller/reconcile_phases.go`
- `ensureImagePullAccess()` creates RoleBindings that fail in kind — guard with environment check
- Session namespace is derived from `project_id` field — if absent, runner lands in `default` namespace

**Acceptance:** `go vet ./... && golangci-lint run` clean, tested in kind cluster

---

### Layer 6: Runner

**Trigger:** New AG-UI event type, new session lifecycle event, runner ↔ CP protocol change

**How:** Follow `control-plane-development.md` for the runner ↔ CP compatibility contract.

**Business logic specific to Ambient:**
- Runner drains inbox at ignition before starting the Claude Code session
- AG-UI event order: `RUN_STARTED` → `TEXT_MESSAGE_CONTENT` (N) → `TEXT_MESSAGE_END` → `MESSAGES_SNAPSHOT` → `RUN_FINISHED`
- `RUN_FINISHED` emitted exactly once, last — CP relies on this to close all watch streams
- `python -m pytest tests/` must pass

**Acceptance:** Runner compat test passes (see `control-plane-development.md`)

---

### Layer 7: Frontend

**Trigger:** New resource visible to users, new action, new page

**How:** Follow `frontend-development.md` — React Query, Shadcn, no-any, colocation.

**Business logic specific to Ambient:**
- New resource pages follow the pattern: `app/projects/[projectName]/<resource>/page.tsx`
- Session stream uses `use-agui-stream.ts` hook — do not implement a new SSE consumer
- `workflow-picker.tsx` is the canonical component for selecting agent workflows

**Acceptance:** `npm run build` — 0 errors, 0 warnings

---

## Ambiguity Log

When the Workflow hits a case it can't handle, the agent stops and records the ambiguity here. Once resolved, the resolution becomes a new rule above.

| Date | Layer | Ambiguity | Resolution |
|---|---|---|---|
| *(add entries as encountered)* | | | |

---

## Adding a New Rule to the Workflow

When an ambiguity is resolved:

1. Identify which layer the rule belongs to
2. Add it to the appropriate per-layer section above
3. Add the resolution to the Ambiguity Log
4. Commit with: `docs(workflow): add rule for <case>`

The Workflow document is a first-class deliverable. Improving it is as valuable as improving code.

---

## Relationship to `ambient.plan`

`ambient.plan` is the **automated execution** of this Workflow against a computed diff:

1. Compute `Δ = Spec - Status`
2. Classify each delta item using the diff classification rules
3. Apply per-layer rules to generate concrete tasks
4. Order tasks by the propagation order
5. Flag any delta items not covered by a Workflow rule (workflow incomplete → stop)
6. Output the task graph

The task graph is the input to `ambient.reconcile`, which executes it layer by layer.
