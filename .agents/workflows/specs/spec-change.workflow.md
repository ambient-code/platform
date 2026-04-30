# Ambient Spec Change Workflow

## Phase 1 — Frame before writing

Before drafting, establish:

- **Desired state only.** The spec describes where things should be, not where they are. Code divergence from the spec is expected and intentional.
- **Scope boundary.** Which components does this change touch? (schema, gRPC, runner, operator, CLI, frontend, SDK, RBAC) — this drives which critics to spawn.
- **Reserved terms check.** Ambient has a specific domain model (Inbox, Session, Agent, Project, Credential, SessionMessage, etc.). Don't repurpose these terms.

## Phase 2 — Draft

Write the spec. Include: data model, write paths, read paths, RBAC, migration plan for all existing consumers.

## Phase 3 — Critic pass

Spawn subagents as critics in parallel. Critics are always evidence-based (read actual code, cite file:line) and assigned narrow mandates. Two categories of critics:

### Standard critics (every spec change)

- **Schema / migration** — DDL correctness, index semantics, rollback, migration registration
- **RBAC / auth** — correct mechanism (not aspirational), all endpoints covered
- **Ambient terminology** — no reserved term collision

### Scope-driven critics (based on Phase 1 scope boundary)

- One critic per major consumer: runner, operator, CLI, frontend, SDK, gRPC proto
- One critic per major concern in the spec: write paths, read paths, compaction/lifecycle

Each critic reports **BLOCKER** / **MAJOR** / **MINOR** with citations.

## Phase 4 — Synthesize and separate

Collapse duplicates. Split findings:

- **Factual errors** — one right answer (wrong SQL semantics, wrong path, wrong auth mechanism, missing enum value) → fix directly
- **Design decisions** — valid tradeoffs exist → ask the author

## Phase 5 — Design questions to author

Present only design decisions. For each: 2–3 concrete options with tradeoffs, one question at a time. Do not ask the author to validate factual correctness.

## Phase 6 — Apply fixes

One pass: all factual corrections + design decisions resolved. Commit with a category-per-line message so the diff is auditable.

## Phase 7 — Second critic pass

Run the same critics again against the updated spec. First-round fixes introduce new surface; the second pass catches what the first missed or created. Stop when the second pass produces only MINORs.

## Heuristics

- **Critics should outnumber reviewers.** Ten parallel critics for 45 minutes beats one sequential review over a day.
- **The author's time is for design decisions only.** Everything with a right answer should never reach them.
- **"Desired state" framing eliminates the largest class of false positives** (current code ≠ spec). Establish it before the first critic pass, not after.
- **The Ambient domain model is a minefield of reserved terms.** A dedicated terminology critic is cheaper than discovering the collision during implementation.
- **Migration path completeness is the most common gap:** for every existing consumer of what you're changing, the spec must say what happens to it.
