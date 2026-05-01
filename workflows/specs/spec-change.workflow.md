# Ambient Spec Change Workflow

## Phase 1 — Frame before writing

Before drafting, establish:

- **Desired state only.** The spec describes where things should be, not where they are. Code divergence from the spec is expected and intentional.
- **Scope boundary.** Which components does this change touch? (schema, gRPC, runner, operator, CLI, frontend, SDK, RBAC) — this drives which critics to spawn.
- **Reserved terms check.** Ambient has a specific domain model (Inbox, Session, Agent, Project, Credential, SessionMessage, etc.). Don't repurpose these terms.

## Phase 2 — Ground in the codebase

Before drafting, read the actual code and specs in the affected areas. The goal is to confirm your understanding of the user's intent without wasting their time on things you can answer yourself.

1. **Read existing specs** in the target domain — what's already specified? Is this an amendment to an existing spec or a new one?
2. **Read the current implementation** — grep for the components identified in Phase 1. Understand what exists today so the spec's migration plan is grounded in reality.
3. **Summarize back to the user** in 3–5 sentences: what you found, what you believe they want to change, and what you're unsure about. Be specific — cite files and current behavior.
4. **Ask only where the codebase is ambiguous.** If the code answers a question, don't ask the user. If two valid interpretations exist, surface it now — not after a full draft.

Do not proceed to drafting until the user confirms the framing is right.

## Phase 3 — Draft

Write the spec. Include: data model, write paths, read paths, RBAC, migration plan for all existing consumers.

## Phase 4 — Critic pass

Spawn subagents as critics in parallel. Critics are always evidence-based (read actual code, cite file:line) and assigned narrow mandates. Two categories of critics:

### Standard critics (every spec change)

- **Schema / migration** — DDL correctness, index semantics, rollback, migration registration
- **RBAC / auth** — correct mechanism (not aspirational), all endpoints covered
- **Ambient terminology** — no reserved term collision

### Scope-driven critics (based on Phase 1 scope boundary)

- One critic per major consumer: runner, operator, CLI, frontend, SDK, gRPC proto
- One critic per major concern in the spec: write paths, read paths, compaction/lifecycle

Each critic reports **BLOCKER** / **MAJOR** / **MINOR** with citations.

## Phase 5 — Synthesize and separate

Collapse duplicates. Split findings:

- **Factual errors** — one right answer (wrong SQL semantics, wrong path, wrong auth mechanism, missing enum value) → fix directly
- **Design decisions** — valid tradeoffs exist → ask the author

## Phase 6 — Design questions to author

Present only design decisions. For each: 2–3 concrete options with tradeoffs, one question at a time. Do not ask the author to validate factual correctness.

## Phase 7 — Apply fixes

One pass: all factual corrections + design decisions resolved. Commit with a category-per-line message so the diff is auditable.

## Phase 8 — Second critic pass

Run the same critics again against the updated spec. First-round fixes introduce new surface; the second pass catches what the first missed or created. Stop when the second pass produces only MINORs.

## Heuristics

- **Critics should outnumber reviewers.** Ten parallel critics for 45 minutes beats one sequential review over a day.
- **The author's time is for design decisions only.** Everything with a right answer should never reach them.
- **"Desired state" framing eliminates the largest class of false positives** (current code ≠ spec). Establish it before the first critic pass, not after.
- **The Ambient domain model is a minefield of reserved terms.** A dedicated terminology critic is cheaper than discovering the collision during implementation.
- **Migration path completeness is the most common gap:** for every existing consumer of what you're changing, the spec must say what happens to it.
