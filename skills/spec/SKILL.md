---
name: spec
description: >
  Create or modify a spec following the project's spec format and conventions.
  Use when the user wants to write a new spec, add requirements or scenarios
  to an existing spec, or restructure spec content. Triggers on: "write a spec",
  "create a spec", "add a requirement", "spec this out", "define the behavior",
  "what should the spec look like", "new spec for", "update the spec".
---

# Write or Modify a Spec

Help the user create or change a spec that describes desired system behavior.

## User Input

```text
$ARGUMENTS
```

## Before Anything Else

Read these two files in full before proceeding:

1. `specs/index.spec.md` — what a spec is, the required format, naming conventions, and what does and does not belong
2. `workflows/specs/spec-change.workflow.md` — the full spec change workflow (framing, grounding, drafting, critic passes, synthesis)

Then follow the workflow phases in order.

## Steps

### Phase 1 — Frame

Establish the framing before writing anything:

- **Desired state only.** Ask the user what the system should do, not what's currently broken. If they describe a bug, redirect: "What should the correct behavior be?"
- **Scope boundary.** Which components does this change touch? (schema, gRPC, runner, operator, CLI, frontend, SDK, RBAC)
- **Reserved terms check.** Verify no collision with Ambient domain model terms (Inbox, Session, Agent, Project, Credential, SessionMessage, etc.)

### Phase 2 — Ground in the codebase

Read actual code and existing specs in the affected areas. Confirm your understanding without wasting the user's time:

- Read existing specs in the target domain
- Grep the components identified in Phase 1
- Summarize back in 3–5 sentences: what you found, what you believe they want, what's ambiguous
- Ask only where the codebase doesn't give a clear answer

Do not proceed to drafting until the user confirms.

### Phase 3 — Draft the Spec

Follow the format from `specs/index.spec.md`:

- **Purpose section** — one paragraph describing the domain or feature
- **Requirements** — each states an observable behavior using RFC 2119 keywords (SHALL, MUST, SHOULD, MAY)
- **Scenarios** — concrete Given/When/Then examples for each requirement that could be turned into tests

Include: data model, write paths, read paths, RBAC, migration plan for all existing consumers.

### Phase 4 — Critic Pass

Spawn critics in parallel per the workflow. Standard critics (every spec change):
- Schema / migration
- RBAC / auth
- Ambient terminology

Plus scope-driven critics based on the components identified in Phase 1.

### Phase 5–6 — Synthesize and Present

Separate findings into factual errors (fix directly) and design decisions (present to user with 2–3 concrete options each, one at a time).

### Phase 7 — Apply and Verify

Apply all fixes. Run a second critic pass (Phase 8). Stop when only MINORs remain.

### Final — Name and Place the File

- Filename: `<descriptive-title>.spec.md`
- If the spec exceeds ~300 words or covers multiple distinct topics, split into a directory with multiple files
- Place in `specs/{domain}/`
