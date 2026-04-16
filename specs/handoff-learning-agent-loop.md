# Session Handoff: Learning Agent Loop

**Date**: 2026-04-16
**Branch**: `main` (no feature branch created yet)
**Working Directory**: `/Users/jeder/repos/platform`

---

## Session Summary

Reviewed Erin Ahmed's (Cleric) talk on building self-learning agents from the Coding Agents Conference (March 2026). Proposed 6 targeted improvements to ACP, decomposed into 8 speckit specs (002-009), created interactive UI mockups, and began implementation. Hit a critical architectural constraint mid-implementation: **no new CRDs are allowed**. The original spec 002 used a CRD and must be rewritten. All design must be grounded in `jeremyeder/continuous-learning-example`.

## What Was Accomplished

### 1. Specs Written (8 files in `specs/`)

| Spec | File | Status |
|------|------|--------|
| 002 | `specs/002-project-memory-store/spec.md` | **NEEDS REWRITE** — designed around CRD, must use file-based CL pipeline |
| 003 | `specs/003-corrections-realtime-pipeline/spec.md` | Good — in-memory buffer, no CRDs |
| 004 | `specs/004-session-scoped-correction-injection/spec.md` | Good — runner-internal |
| 005 | `specs/005-correct-this-ux/spec.md` | Good — frontend popover |
| 006 | `specs/006-visible-attribution/spec.md` | Good — citation badges + dashboard |
| 007 | `specs/007-event-driven-feedback-loop/spec.md` | Good — threshold detection |
| 008 | `specs/008-cross-session-memory/spec.md` | **NEEDS REVISION** — referenced CRD-based memory store |
| 009 | `specs/009-post-session-insight-extraction/spec.md` | **NEEDS REVISION** — referenced CRD-based memory store |

### 2. UI Mockups (2 HTML files in `specs/`)

- `specs/learning-agent-mockup.html` — Conceptual interactive demo with animated learning loop
- `specs/learning-agent-realistic-mockup.html` — Pixel-accurate mockup matching real ACP UI (Geist font, oklch colors, exact layout). 4 screens: Project Memory, Learning Dashboard, Feedback Loop Config, Insight Extraction

### 3. Implementation

| Worktree | Spec | Status |
|----------|------|--------|
| `.claude/worktrees/agent-a715b813` | 002 (CRD-based) | **DEAD — discard entirely**. Built a ProjectMemory CRD, which is not allowed. |
| `.claude/worktrees/agent-a53bce21` | 003 (Corrections Pipeline) | **VALID — keep**. In-memory corrections buffer + runner dual-write. Go tests passing (470), Python tests passing (263). |

### 4. Feature Flag

`learning-agent-loop` with `scope:workspace` tag — added to `flags.json` in both worktrees. Single flag gates all learning features.

---

## Critical Decisions & Constraints

1. **No new CRDs** — ever. Memory must be stored as files in the repo. See memory file `feedback_no_new_crds.md`.

2. **Anchor repo**: `jeremyeder/continuous-learning-example` is the ground truth for the learning architecture:
   - `.ambient/config.json` — repo opts in with `{"learning": {"enabled": true}}`
   - `docs/learned/` — corrections/patterns captured as markdown files
   - Submitted as **draft PRs** with `continuous-learning` label
   - Wiki compiler (Karpathy/llm-wiki-compiler pattern) compiles `docs/` into topic-based wiki
   - Human curation happens through git PR review, not a custom UI

3. **Single feature flag** for everything: `learning-agent-loop`

4. **Admin dashboard direction**: `/admin` route group in existing frontend (option A — start simple, extract later). Cross-project triage, feedback loop monitoring, GitHub overlay.

5. **SQLite+vector deferred** — future enhancement for semantic search over memories, not in scope now.

6. **Screen 2 ("Chat with Correct This") dropped** from realistic mockup — user didn't find it clear.

---

## Worktree Cleanup Needed

**DELETE these worktrees** (dead):
```bash
git worktree remove .claude/worktrees/agent-a715b813 --force
```

**KEEP this worktree** (valid implementation of spec 003):
```bash
# Worktree: .claude/worktrees/agent-a53bce21
# Branch: worktree-agent-a53bce21
# Contains: corrections buffer (Go) + runner dual-write (Python)
# Files changed:
#   NEW: components/backend/handlers/corrections.go
#   NEW: components/backend/handlers/corrections_test.go
#   MOD: components/backend/routes.go
#   MOD: components/backend/tests/constants/labels.go
#   MOD: components/manifests/base/core/flags.json
#   MOD: components/runners/ambient-runner/ambient_runner/bridges/claude/corrections.py
#   MOD: components/runners/ambient-runner/ambient_runner/platform/feedback.py
#   MOD: components/runners/ambient-runner/tests/test_corrections_tool.py
```

---

## Next Steps (Priority Order)

### Step 1: Rewrite spec 002 grounded in the CL pipeline
Read `jeremyeder/continuous-learning-example` thoroughly. Rewrite spec 002 so that:
- Memories are markdown files in `docs/learned/` (not a CRD)
- New memories are submitted as draft PRs with `continuous-learning` label
- Runner reads `docs/learned/` at session init and injects into system prompt
- Backend exposes thin API for listing/reading learned files from workspace (uses existing workspace file API)
- `.ambient/config.json` controls opt-in

### Step 2: Revise specs 008 and 009
Both reference the CRD-based memory store. Update to use file-based approach from step 1.

### Step 3: Merge spec 003 implementation
The corrections pipeline worktree (`agent-a53bce21`) is valid and passing. Merge it into a `learning-agent-loop` feature branch.

### Step 4: Implement revised spec 002
Runner writes to `docs/learned/`, opens draft PRs. No CRD, no new backend CRUD handlers for memories.

### Step 5: Continue Phase 2 (007, 008, 004) and Phase 3 (006, 009)

### Step 6: Admin dashboard spec
New spec for `/admin` route group — correction triage, memory health, session analytics.

---

## Key Files to Read First

```
jeremyeder/continuous-learning-example/README.md     # THE anchor — read this first
jeremyeder/continuous-learning-example/CLAUDE.md      # CL pipeline conventions
jeremyeder/continuous-learning-example/.ambient/config.json  # Opt-in pattern

platform/specs/003-corrections-realtime-pipeline/spec.md    # Valid spec, implemented
platform/specs/002-project-memory-store/spec.md             # User stories are good, architecture is wrong

platform/.claude/projects/-Users-jeder-repos-platform/memory/project_learning_agent_loop.md   # Full initiative context
platform/.claude/projects/-Users-jeder-repos-platform/memory/feedback_no_new_crds.md           # Hard constraint
```

---

## Memory Files Created

| File | Type | Purpose |
|------|------|---------|
| `memory/feedback_no_new_crds.md` | feedback | Never create new CRDs |
| `memory/project_learning_agent_loop.md` | project | Full initiative context and status |

---

## Talk Transcript Reference

The original inspiration was Erin Ahmed's talk "How to Fix Your Agent's Amnesia" at the Coding Agents Conference (MLOps.community, March 3, 2026). Three lessons:
1. Good learning agents make it easy to correct via correction
2. Corrections should persist, compound, and be visible
3. Good learning agents absorb context continuously (ambient learning)

The full transcript was provided in the first message of this session.
