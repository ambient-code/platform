# Ambient Spec & Guide Development Context

**When to load:** Writing or reviewing any spec, guide, openapi.yaml, or any document
that defines *what* the platform should be or *how* to implement it. Also load when
executing or improving the agentic development workflow.

---

## The Relationship: Spec, Status, Guide

```
Spec      desired state   — what Ambient should be
Status    current state   — what the codebase actually does
Guide     reconciler      — how to make Spec == Status
```

**Spec is correct. Status either matches it or is wrong.**

The reconciliation direction is always:

```
Spec → Guide(Δ) → Status
```

Never the reverse. If code was written that doesn't match the Spec, the code is wrong
— unless the Spec is consciously updated to reflect a new decision.

**Changes to code always require changes to the Spec.** A field added to a handler
without a Spec update is undocumented behavior. Update the Spec first — or, if
discovered after the fact, update the Spec in the same PR.

**A Guide is always required.** Even if the Guide is just `make all` — that is still
a Guide. The Guide can be trivially simple when the codebase is mature and the patterns
are stable. We are not there yet. Along the way, the Guide captures the steps, pitfalls,
and ordering constraints that make it possible for a person or agent with no prior
context to implement a Spec change correctly.

---

## What Makes a Good Spec

A good Spec makes the Guide easy to write and the implementation unambiguous.
A bad Spec makes the Guide do guesswork — and guesswork produces wrong code.

### A Good Spec Is Complete

Every entity, endpoint, command, and field is fully defined. No TBDs. No implied
behaviors. No "see code for details."

A Spec entry is **complete** when:
1. The Kind/endpoint/command is fully described — no fields marked TBD
2. All relationships to other Kinds are documented
3. At least one CLI command exposes it
4. At least one SDK example exercises it
5. The `openapi.yaml` entry exists and is valid
6. A Guide exists that covers implementation of this entry

A Spec entry is **incomplete** (blocks implementation) when:
- A field exists with no description or type
- A relationship is implied but not stated
- An endpoint exists in `openapi.yaml` but not in the Spec (or vice versa)
- No CLI command or SDK example exists for the feature
- No Guide covers how to implement it

**Incomplete Spec = implementation ambiguity = agent stops and asks.**

### A Good Spec Is Unidirectional

The Spec defines the desired end state. It does not describe the current state of the
code. It does not say "currently, the field is X — in the future it will be Y." It says
what it should be, full stop. The gap between Spec and Status is the Guide's problem.

### A Good Spec Is Minimal Per Entity

One paragraph per Kind describing what it is. Fields as a flat table with types,
required/optional, constraints, and a one-line description. Relationships stated once.
No narrative history. No rationale unless it affects the implementation.

### A Good Spec Surfaces the User-Facing Surface

CLI commands and SDK examples are first-class Spec artifacts. A feature is not shipped
until a user can invoke it. If no CLI command or SDK example exists for a feature, it
is not in the Spec — regardless of what the API supports.

---

## The Spec/Guide Pair

Every feature area has exactly two documents that travel together:

| File | Role | Answers |
|---|---|---|
| `*.spec.md` | **Desired state** — what it is | Fields? Endpoints? Behavior? |
| `*.guide.md` | **Reconciler** — how to build it | What waves? What commands? What does done look like? |

**The Spec is written first. The Guide is written from the Spec.** You do not write a
Guide without a Spec. You do not implement without a Guide.

The Guide contains exactly the instructions you would give a new engineer assigned to a
bug or feature: which files to read, what to change, how to verify it works, in what
order. The same instructions work for a human and for an agent — clear, sequential,
verifiable steps are universal.

### Current Pairs

| Spec | Guide | What it covers |
|---|---|---|
| `docs/internal/design/ambient-model.spec.md` | `docs/internal/design/ambient-model.guide.md` | Platform data model — all Kinds, fields, relationships, API surface |
| `docs/internal/design/mcp-server.spec.md` | `docs/internal/design/mcp-server.guide.md` | MCP server — tool definitions, annotation state, transport, sidecar |

### Pairing Rules

1. Every Spec must link to its Guide in the header: `**Guide:** [filename.guide.md]`
2. Every Guide must link to its Spec in the header: `**Spec:** [filename.spec.md]`
3. When a Spec changes, the Guide's gap table and wave definitions must be updated in the same commit
4. A Guide without a Spec is a plan without a source of truth — delete it or find the Spec
5. A Spec without a Guide means no one can implement it — write the Guide before assigning work

---

## What Belongs in the Spec

### Resource definitions (Kinds)

Every Kind (Session, Project, Agent, Role, RoleBinding, User, ProjectSettings, ...)
must be fully documented:

- **What it is** — one-paragraph description of the concept
- **Fields** — name, type, required/optional, description, constraints, example values
- **Relationships** — how it relates to other Kinds (owns, references, derives from)
- **Lifecycle** — valid states and transitions (e.g. Session: `pending → running → completed | failed`)
- **Ownership** — which Kinds own which (OwnerReferences in K8s terms)

### API endpoints

Every REST endpoint must be documented in the Spec before it exists in code:

- **Method + path** — e.g. `POST /api/ambient/v1/{project}/sessions`
- **Path parameters** — name, type, description
- **Request body** — schema reference + example
- **Response** — schema reference + example + error codes
- **Authorization** — what RBAC permission is required
- **Behavior** — what happens (idempotency, side effects, ordering guarantees)

### gRPC streams

Every streaming RPC:

- **RPC name + proto service** — e.g. `Sessions.WatchSessionMessages`
- **Request fields**
- **Stream events** — each event type, when it's emitted, what it contains
- **Terminal conditions** — what ends the stream

### CLI commands

Every `acpctl` command is a first-class Spec artifact:

- **Command + subcommand** — e.g. `acpctl session messages -f`
- **Flags** — name, type, default, description
- **Behavior** — what it does, what it prints, exit codes
- **Example** — real invocation with expected output

CLI commands are the **user-facing surface of the Spec**. If a command doesn't exist
in the CLI, the feature isn't shipped — regardless of what the API supports.

### SDK examples

Every resource API must have a working example in the Go SDK, Python SDK, and
TypeScript SDK. These examples are:

1. **Documentation** — how to use the SDK
2. **End-to-end tests** — if the example doesn't run against a real cluster, something is broken

SDK examples belong in the Spec because they define the *intended usage contract*,
not just the implementation.

---

## What Belongs in the Guide

The Guide is the **workplan** — the reconciler that knows what steps to take to make
Spec == Status.

Write it as if onboarding a new engineer to a specific task. Include:

- **Which files to read first** (spec, related docs, relevant code)
- **A gap table** — what exists vs. what the spec requires, row by row
- **Ordered implementation waves** — what to do first, what gates what
- **Acceptance criteria per wave** — specific commands to run, specific outputs to verify
- **Build and test commands** — exact shell commands, no ambiguity
- **Inbox message templates** — what to send each agent when assigning wave work
- **A run log** — updated after each execution with lessons learned

The Guide is an **executable document**. Given the Guide and the Spec, a person or
agent with no prior context should be able to implement the change. If the Guide
requires knowledge not in it, add that knowledge to it.

The Guide's ideal end state is: `make all`. When the Guide is that simple, the
codebase is mature. Until then, the Guide is where we capture everything the code
requires that isn't yet obvious from reading it.

---

## The Reconciliation Loop

```
while Spec != Status:
    Δ = compute_diff(Spec, Status)
    tasks = Guide(Δ)

    for task in topological_order(tasks):
        execute(task)

        if ambiguity_encountered:
            surface_question()
            update_Guide_with_answer()
            stop  ← human resolves, then re-enters loop

        update_Status()

done  ← Spec == Status
```

Every stop is an opportunity to improve the Guide. A mature Guide stops rarely.

---

## The Propagation Order (invariant)

Spec changes always propagate in this order. No exceptions.

```
1. Spec          *.spec.md + openapi.*.yaml
2. API Server    plugins/<kind>/ — model, dao, service, handler, presenter, migration
3. SDK           go-sdk, python-sdk, ts-sdk — regenerated from openapi.yaml
4. CLI           acpctl commands — consume Go SDK
5. Operator      reconciler logic — if new K8s resource behavior needed
6. Runner        Python runner — if new session lifecycle events needed
7. Frontend      NextJS pages + React Query hooks — consume REST API
```

Dependencies flow downward. You cannot implement layer N+1 before layer N is complete
and tested.

---

## Per-Layer Rules

Each layer has a dedicated development guide with full implementation detail: file locations, code patterns, pitfalls, build commands, and acceptance criteria. The entries below state the trigger and point to the guide.

| Layer | Trigger | Development Guide |
|---|---|---|
| 1 — Spec | Human changes `*.spec.md` or `openapi.*.yaml` | *(this document)* |
| 2 — API Server | New Kind, field, endpoint, or gRPC RPC | `api-server-development.md` |
| 3 — SDK | Any Layer 2 change that modifies `openapi.yaml` | `sdk-development.md` |
| 4 — CLI | New SDK capability or user-facing command in Spec CLI table | `cli-development.md` |
| 5 — Operator | New CRD field, reconciler phase, or K8s resource lifecycle | `operator-development.md` |
| 6 — Runner | New AG-UI event type, session lifecycle event, or CP protocol change | `control-plane-development.md` |
| 7 — Frontend | New resource visible to users, new action or page | `frontend-development.md` |

### Layer 1: Spec — rules

- Run `make generate` in `ambient-api-server` after any openapi change — regenerates `pkg/api/openapi/`
- Validate `openapi.yaml` is valid before proceeding
- Update `ambient-cli/README.md` if new CLI commands are implied
- Update SDK examples if new usage patterns are implied
- **Acceptance:** `openapi.yaml` is valid, all affected fragments are consistent with the Spec

---

## Spec Files in This Repository

| File | Paired with | What it specifies |
|---|---|---|
| `docs/internal/design/ambient-model.spec.md` | `ambient-model.guide.md` | Canonical data model — all Kinds, fields, relationships |
| `docs/internal/design/mcp-server.spec.md` | `mcp-server.guide.md` | MCP server — tools, annotation state, transport |
| `components/ambient-api-server/openapi/openapi.yaml` | — | REST API surface — all endpoints, schemas, error codes |
| `components/ambient-api-server/openapi/openapi.*.yaml` | — | Per-resource OpenAPI fragments (merged into openapi.yaml) |
| `components/ambient-sdk/go-sdk/examples/` | — | Go SDK usage examples (spec-as-code) |
| `components/ambient-sdk/python-sdk/examples/` | — | Python SDK examples |
| `components/ambient-cli/README.md` | — | CLI command reference |

---

## Conventions

- **Edit the Spec first**, then `openapi.yaml`, never the reverse
- **Update the Guide's gap table** whenever the Spec changes — before assigning any work
- **`openapi.yaml` is generated from fragments** (`openapi.*.yaml`) — edit the fragment, not the merged file directly
- **Never add a field to code that isn't in the Spec** — if the code needs something not in the Spec, update the Spec first
- **SDK examples must be runnable** — test them against kind before committing
- **CLI README is part of the Spec** — if you add a command, update the README in the same PR

---

## Canonical Commit Structure

Every Spec-driven change produces a commit history in a fixed structure. This structure
is the binding between the Spec, the Guide that processed it, and the code that resulted.

```
commit A  spec(<scope>):  Spec Δ only — *.spec.md + openapi fragments
commit B  docs(guide):    Guide snapshot — the one that worked, with run log
commit C  feat(api):      Wave 2 — API Server + BE plugins
commit D  feat(sdk):      Wave 3 — regenerated SDK (all 3 langs)
commit E  feat(cli):      Wave 5 — acpctl commands
commit F  …               Wave N — operator / runner / FE
```

**Commit B is always the Guide that produced commits C+.** Given any code commit, walk
back one commit to read the exact instruction set in effect when that code was written.

### What Goes in Each Commit

**Commit A — Spec:**
- `docs/internal/design/*.spec.md` changes
- `components/ambient-api-server/openapi/openapi.*.yaml` fragment changes
- No code changes. No Guide changes.

**Commit B — Guide:**
- `docs/internal/design/*.guide.md` — updated run log, new lessons, corrected steps
- `.claude/context/*.md` — context file updates from lessons learned this run
- `.claude/commands/` — any new slash commands added during the run
- No application code changes.

**Commits C+ — Code (one per wave/component):**
- Application code only: plugins, SDK output, CLI commands, operator logic, runner changes
- Each commit message: `feat(<component>): <what changed>`
- Wave commits ordered by the propagation order (API → SDK → BE → CLI → ...)

---

## Rebase Discipline — Sealing a Guide Run

During active development a run is messy. The agent may attempt Wave 4 three times.
The Guide itself may be wrong and get corrected mid-run. Intermediate commits accumulate.

**Before opening a PR, always rebase into the canonical structure.**

### Rebase Procedure

```bash
git log --oneline <base>..HEAD
```

Categorize every commit:

| Category | Target commit | Action |
|---|---|---|
| Spec changes (data model, openapi fragments) | Commit A | `pick` or `squash` into A |
| Guide/doc changes (*.guide.md, context files) | Commit B | `pick` or `squash` into B |
| Code — Wave 2 (API) | Commit C | `pick` or `squash` into C |
| Code — Wave 3 (SDK) | Commit D | `pick` or `squash` into D |
| Code — Wave 5 (CLI) | Commit E | `pick` or `squash` into E |
| Fix-ups, reverts, partial attempts | — | `squash` or `drop` |
| Mid-run Guide corrections | — | `squash` into Commit B |

```bash
git rebase -i <base>
```

### Invariants That Must Hold After Rebase

1. **Commit A contains only Spec files** — no `.go`, no `.ts`, no `.py` application code
2. **Commit B contains only doc/Guide files** — `*.guide.md`, `.claude/context/`, `.claude/commands/`
3. **Commit B's Guide includes the run log entry for this run** — lessons learned, gap table updated
4. **Commits C+ contain only application code** — no Spec files, no Guide files
5. **The commit sequence matches the propagation order** — API before SDK before CLI
6. **All commits individually build** — verify at each commit (optional but ideal)

---

## Ambiguity Log

When the Guide hits a case it can't handle, the agent stops and records it here. Once
resolved, the resolution becomes a new rule in the per-layer section above.

| Date | Layer | Ambiguity | Resolution |
|---|---|---|---|
| *(add entries as encountered)* | | | |

---

## Relationship to `ambient.plan`

`ambient.plan` is the **automated execution** of this Guide against a computed diff:

1. Compute `Δ = Spec - Status`
2. Classify each delta item by layer using the propagation order
3. Apply per-layer rules to generate concrete tasks
4. Order tasks topologically
5. Flag any delta items not covered by a Guide rule (Guide incomplete → stop)
6. Output the task graph for `ambient.reconcile` to execute
