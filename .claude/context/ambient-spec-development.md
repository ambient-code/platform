# Ambient Spec Development Context

**When to load:** Writing or reviewing any spec, guide, openapi.yaml, or any document that defines *what* the platform should be or *how* to implement it.

---

## The Spec/Guide Pair

Every feature area has exactly two documents that travel together:

| File | Role | Answers |
|---|---|---|
| `*.spec.md` | **Desired state** — what it is | What are the fields? What are the endpoints? What is the behavior? |
| `*.guide.md` | **Workplan** — how to build it | What waves? What commands? What does done look like? |

**The spec is written first. The guide is written from the spec.** You do not write a guide without a spec. You do not implement without a guide.

The guide contains exactly the same instructions you would give a junior engineer assigned to a bug or feature: which files to read, what to change, how to verify it works, in what order. The same instructions work for a human and for an agent. There is no special "agent format" — clear, sequential, verifiable steps are universal.

### Current Pairs

| Spec | Guide | What it covers |
|---|---|---|
| `docs/internal/design/ambient-model.spec.md` | `docs/internal/design/ambient-model.guide.md` | Platform data model — all Kinds, fields, relationships, API surface |
| `docs/internal/design/mcp-server.spec.md` | `docs/internal/design/mcp-server.guide.md` | MCP server — tool definitions, annotation state, transport, sidecar |

### Pairing Rules

1. Every spec must link to its guide in the header: `**Guide:** [filename.guide.md]`
2. Every guide must link to its spec in the header: `**Spec:** [filename.spec.md]`
3. When a spec changes, the guide's gap table and wave definitions must be updated in the same commit
4. A guide without a spec is a plan without a source of truth — delete it or find the spec
5. A spec without a guide means no one can implement it — write the guide before assigning work

---

## What the Spec Is

The Spec is **desired state** — the authoritative definition of what Ambient should be. It is not documentation that follows code. It is documentation that *precedes and governs* code.

**The Spec is correct. The code either matches it or is wrong.**

The reconciliation direction is always:

```
Spec → Status (code)
```

Never the reverse. If code was written that doesn't match the Spec, the code is wrong — unless the Spec is updated consciously to reflect a decision.

---

## What Belongs in the Spec

### Resource definitions (Kinds)

Every Kind (Session, Project, Agent, Role, RoleBinding, User, ProjectSettings, ...) must be fully documented:

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

CLI commands are the **user-facing surface of the Spec**. If a command doesn't exist in the CLI, the feature isn't shipped — regardless of what the API supports.

### SDK examples

Every resource API must have a working example in the Go SDK, Python SDK, and TypeScript SDK. These examples are:

1. **Documentation** — how to use the SDK
2. **End-to-end tests** — if the example doesn't run against a real cluster, something is broken

SDK examples belong in the Spec because they define the *intended usage contract*, not just the implementation.

---

## What Belongs in the Guide

The Guide is the **workplan** — instructions for implementing what the Spec defines.

Write the guide as if onboarding a new engineer to a specific task. Include:

- **Which files to read first** (spec, related docs, relevant code)
- **A gap table** — what exists vs. what the spec requires, row by row
- **Ordered implementation waves** — what to do first, what gates what
- **Acceptance criteria per wave** — specific commands to run, specific outputs to verify
- **Build and test commands** — exact shell commands, no ambiguity
- **Inbox message templates** — what to send each agent when assigning wave work
- **A run log** — updated after each execution with lessons learned

The guide is an **executable document**. Given the guide and the spec, a person or agent with no prior context should be able to implement the change. If the guide requires knowledge not in the guide, add that knowledge to the guide.

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

## Spec Quality Rules

A Spec entry is **complete** when:

1. The Kind/endpoint/command is fully described — no fields marked TBD
2. All relationships to other Kinds are documented
3. At least one CLI command exposes it
4. At least one SDK example exercises it
5. The openapi.yaml entry exists and is valid
6. A guide exists that covers implementation of this entry

A Spec entry is **incomplete** (blocks implementation) when:
- A field exists with no description or type
- A relationship is implied but not stated
- An endpoint exists in openapi.yaml but not in the spec (or vice versa)
- No CLI command or SDK example exists for the feature
- No guide covers how to implement it

**Incomplete Spec = implementation ambiguity = agent stops and asks.**

---

## Spec ↔ Status Relationship

```
Spec (*.spec.md + openapi.yaml + CLI README + SDK examples)
    │
    Δ = Spec - Status
    │
    ├── Δ == ∅  →  no-op
    └── Δ ≠ ∅   →  Guide(Δ) → TaskGraph → execute
```

Status is the sum of all running code: api-server plugins, SDK clients, CLI commands, frontend pages, operator logic, runner behavior.

When you change the Spec, you are creating a diff that must propagate through every layer of Status. The Guide tells you how.

---

## Conventions

- **Edit the spec first**, then openapi.yaml, never the reverse
- **Update the guide's gap table** whenever the spec changes — before assigning any work
- **openapi.yaml is generated from fragments** (`openapi.*.yaml`) — edit the fragment, not the merged file directly
- **Never add a field to code that isn't in the Spec** — if you discover the code needs something not in the Spec, update the Spec first
- **SDK examples must be runnable** — test them against kind before committing
- **CLI README is part of the Spec** — if you add a command, update the README in the same PR

---

## Canonical Commit Structure

Every Spec-driven change produces a commit history in a fixed relational structure. This structure is the **binding** between the Spec, the Guide that processed it, and the code that resulted.

### The Pipeline

```
┌─────────────────────────────────────────────────────────────────┐
│                        INPUTS                                   │
│                                                                 │
│   Spec Δ                          Guide                         │
│   (what changed)                  (how to propagate it)         │
│                                                                 │
│   *.spec.md        ──┐            *.guide.md       ──┐          │
│   openapi.*.yaml     │            .claude/context/   │          │
│                      │            .claude/skills/    │          │
└──────────────────────┼────────────────────────────── ┼──────────┘
                       │                               │
                       ▼                               ▼
              ┌────────────────────────────────────────┐
              │                                        │
              │   Guide( Spec Δ )  →  TaskGraph        │
              │                                        │
              │   for each Δ item:                     │
              │     classify by layer                  │
              │     apply per-layer rules              │
              │     order by propagation               │
              │                                        │
              └──────────────┬─────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                        OUTPUTS (git commits)                    │
│                                                                 │
│   commit 1  spec(…)    Spec Δ only — *.spec.md + openapi frags  │
│   commit 2  docs(…)    Guide snapshot — the one that worked     │
│   commit 3  feat(api)  Wave 2 — API Server + BE plugins         │
│   commit 4  feat(sdk)  Wave 3 — regenerated SDK (all 3 langs)   │
│   commit 5  feat(cli)  Wave 5 — acpctl commands                 │
│   commit 6  …          Wave N — operator / runner / FE          │
│                                                                 │
│   commit 2 is always the Guide that produced commits 3+         │
└─────────────────────────────────────────────────────────────────┘
```

**Reading the diagram:**
- The Spec Δ is the *trigger* — what changed in the desired state.
- The Guide is the *policy* — it knows the codebase and maps each Δ item to the correct layer and order.
- The output commits are the *evidence* — a durable record of what the Guide did.
- Commit 2 seals the relationship: given any code commit, walk back one commit to find the Guide that produced it.

### The Model

Think of a workflow run as a record in a relational table:

```
WorkflowRun {
    spec_commit:  sha   -- the Spec diff that triggered the run
    guide_commit: sha   -- the Guide snapshot that processed that diff
    code_commits: [sha] -- the code changes the Guide produced
}
```

Every time you change the Spec and run the Guide, you are creating a new WorkflowRun. The commit history is the durable record of that run.

### The Commit Order

```
commit 1: spec changes  — *.spec.md, openapi.*.yaml fragments
commit 2: guide changes — *.guide.md, .claude/context/*.md, any new docs
commit 3+: code changes — one commit per component/wave (API, SDK, BE, CLI, ...)
```

**Commit 2 is always the Guide that produced commit 3+.** This invariant means: given any code commit, walk back one commit and read exactly the instruction set that was in effect when that code was written. The Guide is not documentation about what happened; it is the executable that ran.

### Why This Structure

- **Traceability:** Any future agent can `git log --oneline` and immediately identify the spec boundary, the guide snapshot, and the code wave commits.
- **Iterability:** During development, the guide is run many times and improves on every pass. The final rebase collapses all intermediate attempts into this canonical structure — the messy iteration history disappears, and what remains is the Spec that motivated the work, the Guide that finally succeeded, and the code it produced.
- **Reproducibility:** A new agent can checkout commit 2, read the Guide, and re-execute it against a clean environment — and should produce the same code commits (commit 3+).

### What Goes in Each Commit

**Commit 1 — Spec:**
- `docs/internal/design/*.spec.md` changes
- `components/ambient-api-server/openapi/openapi.*.yaml` fragment changes
- No code changes. No guide changes.

**Commit 2 — Guide:**
- `docs/internal/design/*.guide.md` — updated run log, new lessons, corrected steps, updated gap table
- `.claude/context/*.md` — any context file updates that reflect lessons from this run
- `.claude/commands/` — any new slash commands added during the run
- No application code changes.

**Commits 3+ — Code (one per wave/component):**
- Application code only: plugins, SDK output, CLI commands, operator logic, runner changes
- Each commit message follows `feat(<component>): <what changed>`
- Wave commits are ordered by the propagation order (API → SDK → BE → CLI → ...)

### Rebase Is the Seal

After any iterative development session, `git rebase -i` is used to squash the run into this structure before the PR is opened. See `ambient-workflow-development.md` for the rebase discipline.
