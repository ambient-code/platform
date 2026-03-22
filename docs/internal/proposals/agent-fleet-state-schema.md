# Spec: Agent Fleet State — Self-Describing Annotation Protocol

**Date:** 2026-03-22
**Status:** Draft

---

## Overview

The annotation system is **self-describing**. Agents do not need to be pre-programmed with the protocol. When an agent session starts or is poked, its first action is to read the project annotations, discover the communication protocol and shared contracts stored there, compare the declared spec against its own current annotation state, and reconcile any drift.

The user's only job is to write a prompt. Everything else — state reporting, coordination, contract compliance — is infrastructure that agents resolve among themselves by reading and writing project annotations.

---

## Mental Model

The project is the source of truth. It carries:

1. **`ambient.io/protocol`** — how agents communicate (check-in cadence, blocker escalation, handoff rules)
2. **`ambient.io/contracts`** — shared agreements across agents (API shapes, branch conventions, review gates)
3. **`ambient.io/agent-roster`** — the live fleet summary (each agent writes its own entry)
4. **`ambient.io/summary`** — human-readable project state for the user

Agents are responsible for reading the protocol, reading the contracts, and keeping their own annotations consistent with both. No human coordination is required.

---

## Project Annotations — The Self-Describing Layer

### `ambient.io/summary`

Human-readable current state of the project. Written by any agent after significant state changes. This is what the user sees when they ask "what's going on."

```json
{
  "as_of": "2026-03-22T01:30:00Z",
  "release": "v0.4.0",
  "status": "in-progress",
  "narrative": "Three agents active. api-agent completing inbox fixes; cp-agent blocked on PLAT-112 merge; fe-agent building session panel. Two issues blocking release.",
  "blocked_count": 1,
  "ready_count": 1,
  "active_count": 3
}
```

---

### `ambient.io/protocol`

Defines how agents in this project coordinate. Written once at project setup (or by an orchestrator agent). Agents read this on every session start and comply.

```json
{
  "version": "1",
  "checkin": {
    "trigger": "session_start | session_end | blocker | handoff",
    "action": "read_project_annotations, compare_self_state, reconcile, update_roster_entry, update_summary"
  },
  "poke": {
    "trigger": "inbox_message",
    "action": "read_project_annotations, compare_self_state, reconcile, report_status_to_sender"
  },
  "blocker": {
    "action": "set ambient.io/blocked=true, write ambient.io/blocker, update roster entry, send inbox message to peer-agents"
  },
  "handoff": {
    "action": "write work.ambient.io/next-tasks, send inbox message to target agent with context, update ambient.io/last-handoff"
  },
  "roster_entry": {
    "fields": ["id", "name", "ready", "blocked", "blocker", "issue", "branch", "current-task", "pr-url", "pr-status"]
  }
}
```

---

### `ambient.io/contracts`

Shared agreements that all agents in the project must honor. Written by orchestrator or lead agent. Any agent can propose a contract amendment via inbox message to peers; unanimous acknowledgment required before the contract annotation is updated.

```json
{
  "version": "3",
  "git": {
    "base_branch": "main",
    "branch_convention": "feat/<issue-slug>",
    "worktree_convention": "feat-<issue-slug>",
    "commit_convention": "conventional-commits",
    "pr_required": true,
    "review_gate": "1-approval"
  },
  "api": {
    "openapi_source_of_truth": "components/ambient-api-server/openapi/",
    "no_manual_openapi_edits": true,
    "sdk_regenerate_after_openapi_change": true,
    "breaking_changes_require_announcement": true
  },
  "coordination": {
    "blocking_issue_threshold": 2,
    "escalate_after_blocked_minutes": 60,
    "peer_notification_on_blocker": true
  },
  "jira": {
    "project_key": "PLAT",
    "issue_status_on_pr_open": "In Review",
    "issue_status_on_merge": "Done"
  }
}
```

---

### `ambient.io/agent-roster`

Live fleet state. Each agent owns its own entry — it reads the array, updates its entry, writes the array back. Agents never overwrite entries they do not own.

```json
[
  {
    "id": "uuid-api",
    "name": "api-agent",
    "ready": false,
    "blocked": false,
    "issue": "PLAT-117",
    "issue-status": "In Review",
    "branch": "feat/session-messages",
    "current-task": "",
    "pr-url": "https://github.com/org/repo/pull/88",
    "pr-status": "open",
    "last-seen": "2026-03-22T01:30:00Z"
  },
  {
    "id": "uuid-cp",
    "name": "cp-agent",
    "ready": false,
    "blocked": true,
    "blocker": {"type": "issue", "ref": "PLAT-112", "detail": "Inbox handler fix must merge first"},
    "issue": "PLAT-118",
    "branch": "feat/grpc-watch",
    "current-task": "Waiting for PLAT-112",
    "last-seen": "2026-03-22T01:15:00Z"
  },
  {
    "id": "uuid-fe",
    "name": "fe-agent",
    "ready": true,
    "blocked": false,
    "issue": "PLAT-119",
    "branch": "feat/session-panel",
    "current-task": "Building session message stream UI",
    "last-seen": "2026-03-22T01:28:00Z"
  }
]
```

---

## Agent Lifecycle — What Happens on Session Start

When a session is ignited (user prompt or scheduled ignition), the agent executes this sequence before doing any work:

```
1. GET project annotations
   → read ambient.io/protocol       (how to behave)
   → read ambient.io/contracts      (what to honor)
   → read ambient.io/agent-roster   (peer state)
   → read ambient.io/summary        (current narrative)

2. GET own agent annotations
   → read work.ambient.io/current-task, next-tasks
   → read ambient.io/blocked, ambient.io/blocker
   → read git.ambient.io/branch, pr-status

3. Reconcile
   → compare own annotations against contracts (branch convention, commit convention, etc.)
   → compare own roster entry against own agent annotations
   → identify drift, fix discrepancies

4. Update own state
   → write updated agent annotations (ready, current-task, issue, branch, etc.)
   → write updated roster entry into project ambient.io/agent-roster
   → write updated ambient.io/summary if narrative has changed

5. Proceed with work
```

---

## Agent Lifecycle — What Happens on Poke (Inbox Message)

When an agent receives an inbox message (from a peer, a user, or the platform):

```
1. Read the message
2. GET project annotations (same as session start steps 1-3)
3. Compare own current state against protocol and contracts
4. Report status to sender via inbox reply:
   → current task, issue, branch, pr-status
   → blocked state and blocker detail if applicable
   → next-tasks queue
5. If message contains a task or handoff: incorporate into next-tasks, acknowledge
6. Update roster entry and summary
```

---

## Agent Annotations (Self-Reported)

These are written by the agent to itself. They are the source of truth for the agent's roster entry.

### Jira

| Key | Type | Example |
|---|---|---|
| `work.ambient.io/epic` | string | `"PLAT-42"` |
| `work.ambient.io/epic-summary` | string | `"MCP Server — Phase 1"` |
| `work.ambient.io/issue` | string | `"PLAT-117"` |
| `work.ambient.io/issue-summary` | string | `"Implement inbox handler scoping"` |
| `work.ambient.io/issue-status` | string | `"In Progress"` \| `"In Review"` \| `"Done"` \| `"Blocked"` |

### Git

| Key | Type | Example |
|---|---|---|
| `git.ambient.io/worktree` | string | `"feat-session-messages"` |
| `git.ambient.io/branch` | string | `"feat/session-messages"` |
| `git.ambient.io/base-branch` | string | `"main"` |
| `git.ambient.io/last-commit-sha` | string | `"6da3794d"` |
| `git.ambient.io/last-commit-msg` | string | `"fix(api): API consistency audit..."` |
| `git.ambient.io/pr-url` | string | `"https://github.com/org/repo/pull/88"` |
| `git.ambient.io/pr-status` | string | `"open"` \| `"draft"` \| `"merged"` \| `"closed"` |

### Ready and Blocked

| Key | Type | Example |
|---|---|---|
| `ambient.io/ready` | string | `"true"` \| `"false"` |
| `ambient.io/ready-reason` | string | `"Waiting for PLAT-112 to merge"` |
| `ambient.io/blocked` | string | `"true"` \| `"false"` |
| `ambient.io/blocker` | JSON | `{"type":"issue","ref":"PLAT-112","detail":"..."}` |

Blocker types: `"issue"` \| `"agent"` \| `"review"` \| `"external"`

### Work Queue

| Key | Type | Description |
|---|---|---|
| `work.ambient.io/current-task` | string | Active task (empty when idle) |
| `work.ambient.io/next-tasks` | JSON array | Ordered upcoming tasks |
| `work.ambient.io/completed-tasks` | JSON array | Append-only log, trim to last 20 |

### Coordination

| Key | Type | Description |
|---|---|---|
| `ambient.io/peer-agents` | JSON array | UUIDs of agents this agent coordinates with |
| `ambient.io/last-handoff` | JSON | Most recent handoff record |

```json
// ambient.io/last-handoff
{
  "to": "uuid-cp",
  "to-name": "cp-agent",
  "issue": "PLAT-117",
  "context": "inbox handler fix merged as 6da3794d; your PLAT-112 dependency is resolved",
  "at": "2026-03-22T01:30:00Z"
}
```

---

## Session Annotations (Ephemeral)

Reset each ignition. Track what this specific run did.

| Key | Type | Description |
|---|---|---|
| `work.ambient.io/task` | string | Task this session was ignited to perform |
| `work.ambient.io/issue` | string | Issue in scope |
| `git.ambient.io/branch` | string | Branch at session start |
| `git.ambient.io/base-commit-sha` | string | HEAD at session start |
| `ambient.io/blocked` | string | Session-local blocked state |
| `ambient.io/blocker` | JSON | Session-local blocker |
| `ambient.io/last-commit-sha` | string | Commits made this session |
| `ambient.io/last-pr-url` | string | PR opened or updated this session |

---

## Labels (Filterable)

Applied to Agent and Session. Enable TSL search queries across the fleet.

| Key | Values |
|---|---|
| `ambient.io/ready` | `"true"` \| `"false"` |
| `ambient.io/blocked` | `"true"` \| `"false"` |
| `work.ambient.io/epic` | e.g. `"PLAT-42"` |
| `work.ambient.io/project` | e.g. `"PLAT"` |
| `git.ambient.io/worktree` | e.g. `"feat-session-messages"` |

---

## Contract Amendment Protocol

Agents do not unilaterally change `ambient.io/contracts`. The protocol for amendments:

```
1. Proposing agent writes a draft to its own annotations:
      work.ambient.io/contract-proposal = { "field": "...", "proposed": "...", "reason": "..." }

2. Proposing agent sends inbox message to all peer-agents listed in ambient.io/peer-agents:
      "Contract amendment proposed: [field] from [old] to [new]. Reason: [reason]. Reply ACK or NACK."

3. Each peer reads the proposal, evaluates against current work, replies ACK or NACK.

4. If all peers ACK: proposing agent updates ambient.io/contracts, increments version, clears proposal.

5. If any peer NACKs: proposal is dropped. Blocker noted if the disagreement is blocking.
```

---

## Fleet Rollup Queries

### What is the state of the fleet right now?

```
GET /api/ambient/v1/projects
→ for each project: annotations["ambient.io/summary"]          # narrative
→ for each project: annotations["ambient.io/agent-roster"]     # per-agent state
→ aggregate: blocked_count, ready_count, active epics
```

### Which agents are blocked and why?

```
GET /api/ambient/v1/projects/{id}/agents?search=ambient.io/blocked = 'true'
→ for each result: annotations["ambient.io/blocker"]
```

### What is the current release status?

```
GET /api/ambient/v1/projects/{id}
→ annotations["work.ambient.io/release"]
→ annotations["work.ambient.io/release-status"]
→ annotations["work.ambient.io/blocking-issues"]
→ annotations["ambient.io/agent-roster"] → filter blocked entries
```

---

## What the User Never Touches

- Label or annotation keys — agents self-report using the protocol spec stored in the project
- Coordination rules — defined once in `ambient.io/protocol`, read by every agent on every start
- Contract versions — agents manage amendment and versioning among themselves
- Roster entries — each agent owns its entry; the project annotation is the aggregate

The user writes a prompt. The agent reads the project, finds the protocol, finds the contracts, reconciles its state, and proceeds. The annotation system is infrastructure, not user interface.
