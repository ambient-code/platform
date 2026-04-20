# Design Documents

## The Core Thesis

In 1996, Charles Fishman published *They Write The Right Stuff* in Fast Company. It profiled the software team at NASA's Johnson Space Center that writes the shuttle flight software — roughly 420,000 lines of code with an error rate approaching zero. The team produced something like one defect per 420,000 lines of delivered code.

They did not achieve this through exceptional individual talent. They achieved it through a system.

Every error was tracked. Every error was root-caused. Every root cause was fed back into the process that produced it. When a bug appeared in the code, it meant there was a bug in the process that wrote the code. The fix was not just to patch the code — it was to fix the process so the same class of error could not be produced again. Relentless, compounding self-improvement.

> "The people who write this software are not superhuman geniuses. They are disciplined professionals who follow a highly evolved process."
> — Charles Fishman, *They Write The Right Stuff*, Fast Company, 1996

---

## This Project's Equivalent

This codebase is the stake in the ground. Everything written so far exists. The question going forward is: how do we ensure that every change improves it, and that no change introduces a class of error that has already been seen?

The answer is the same as NASA's: a system.

**The system here has three layers:**

```
Spec          desired state    what Ambient should be
Guide         reconciler       how to change code to match the spec
Context       how-to           how to write correct code in each component
```

A bug in the code means one of three things:

1. **The spec was wrong** — the desired state was ambiguous or incorrect
2. **The guide was wrong** — the reconciliation process produced the wrong change
3. **The context was wrong** — the implementation instructions were incomplete or incorrect

The fix is always the same: find which document produced the bug, and update it so that document cannot produce that bug again. The code fix is secondary. The process fix is primary.

---

## The Documents in This Directory

### Spec files (`*.spec.md`)

Each spec defines the desired state for one area of the platform. Fields, endpoints, relationships, CLI commands, RBAC. No implementation detail. No current state description. Pure desired state.

A spec is **complete** when it is unambiguous enough that two engineers reading it independently would make the same implementation decision. If that's not true, the spec is incomplete.

**Current specs:**

| Spec | What it defines |
|---|---|
| `ambient-model.spec.md` | All platform Kinds: Session, Agent, Project, Inbox, Role, RoleBinding, User |
| `control-plane.spec.md` | Control plane gRPC protocol, session fan-out, runner contract |
| `mcp-server.spec.md` | MCP server tools, annotation state, sidecar transport |

### Guide files (`*.guide.md`)

Each guide is the reconciler for its paired spec. It answers: given a spec change, what steps produce correct code? In what order? What does done look like?

Guides are **living documents**. Every time the workflow runs and something is discovered — a missing step, a wrong assumption, a pitfall — the guide is updated before moving on. The guide that exists at the end of a run is more correct than the one at the beginning. This is the self-improvement mechanism.

**There are no separate run log files.** Lessons learned are corrections to the guide itself. Git history is the audit trail.

**Current guides:**

| Guide | Paired spec |
|---|---|
| `ambient-model.guide.md` | `ambient-model.spec.md` |
| `control-plane.guide.md` | `control-plane.spec.md` |
| `mcp-server.guide.md` | `mcp-server.spec.md` |

### Component context files (`.claude/context/*-development.md`)

Each context file is the deep how-to for one component. Where to write code. What patterns to follow. What pitfalls exist. Build commands. Acceptance criteria. These are the documents that prevent a class of implementation error from recurring.

When a bug is found in a component, the corresponding context file is updated so the same bug cannot be introduced again by following the instructions.

**Current context files:**

| Context | Component |
|---|---|
| `api-server-development.md` | `components/ambient-api-server/` |
| `sdk-development.md` | `components/ambient-sdk/` |
| `cli-development.md` | `components/ambient-cli/` |
| `operator-development.md` | `components/ambient-control-plane/` |
| `control-plane-development.md` | Runner + CP protocol |
| `frontend-development.md` | `components/frontend/` |
| `backend-development.md` | `components/backend/` (V1) |
| `ambient-spec-development.md` | Spec/guide authoring |

---

## The Change Process

**No code changes without a spec change.**

If the code needs to change, the spec changes first. If the spec is already correct and the code is wrong, fix the code — but also fix whichever guide or context file would have prevented the error.

The flow is always:

```
1. Spec changes (desired state updated)
2. Gap table produced (what is the delta between spec and code?)
3. Guide consulted (what steps close each gap?)
4. Context files read (how is each step implemented correctly?)
5. Code written
6. Spec, guide, and context updated with anything discovered
```

This is not bureaucracy. It is the mechanism that makes each run produce better code than the last one, and that makes the codebase improvable by anyone who reads the documents — human or agent.

---

## The Self-Improvement Loop

```
while code != spec:
    gap = spec - code
    guide → tasks(gap)
    for each task:
        context → correct implementation
        write code
        if something was wrong or missing:
            fix spec / guide / context   ← this is the key step
            continue
    verify
```

Every time the loop stops because something was wrong, the documents get better. A mature system stops rarely. We are not there yet. The documents are how we get there.

---

## Reading Order for a New Contributor

1. This README — the why
2. `ambient-model.spec.md` — what the platform is
3. `ambient-model.guide.md` — how changes are made
4. The context file for the component you are working on
