# Ambient Annotation System

Annotations are the runtime memory of the Ambient agent platform. They are key-value
strings stored on Project and Agent resources, read and written by agents via MCP tools
during every session. This document describes how they work, what they contain, and
what is expected of agents that use them.

---

## The Core Distinction: `prompt` vs `annotations`

Every resource (`Project`, `Agent`) has two conceptually different fields.

### `prompt` — Permanent Identity

The `prompt` field is compiled into the agent's system prompt at creation time and
**included verbatim in every session**. It does not change at runtime.

It answers: *who am I, what do I own, and what do I always do?*

```
┌──────────────────────────────────────────────────────────┐
│  Every Agent Session                                     │
│                                                          │
│  ┌────────────────────────────────┐                      │
│  │  project.prompt                │  ← baked in, static │
│  │  "This workspace builds the    │                      │
│  │   Ambient Code Platform..."    │                      │
│  └────────────────────────────────┘                      │
│                                                          │
│  ┌────────────────────────────────┐                      │
│  │  agent.prompt                  │  ← baked in, static │
│  │  "You are the API Engineer.    │                      │
│  │   Bootstrap: call              │                      │
│  │   ambient_get_project..."      │                      │
│  └────────────────────────────────┘                      │
│                                                          │
│  ↓ agent calls MCP to load runtime state ↓              │
│                                                          │
│  ┌────────────────────────────────┐                      │
│  │  annotations (live, mutable)   │  ← read via MCP     │
│  │  protocol, rules, roster,      │                      │
│  │  epics, task state, blockers   │                      │
│  └────────────────────────────────┘                      │
└──────────────────────────────────────────────────────────┘
```

Keep prompts **stable and minimal**. They should tell the agent *where* to find live
information (annotation keys, bookmark references), not duplicate that information inline.
When a rule or piece of state might change, it belongs in an annotation — not the prompt.

### `annotations` — Ephemeral Runtime State

Annotations are the system's **live memory**. They are read by agents at session start via
`ambient_get_project` / `ambient_get_agent`, and written back at session end via
`ambient_update_project` / `ambient_update_agent`.

Annotations answer: *what is the current state of the world?*

---

## Annotation Value Format

All annotation values are **strings**. The content may be structured or unstructured:

| Format | Example | Use |
|---|---|---|
| Plain string | `"main"` | Single scalar values |
| JSON array | `'["api","sdk","reviewer"]'` | Lists of names or IDs |
| JSON object | `'{"status":"active","as_of":"..."}' ` | Structured status records |
| Multi-line prose | `"SESSION START:\n  1. Call..."` | Protocols, rules, guides |

The platform treats all annotation values as opaque strings. Agents are responsible for
parsing them correctly based on context.

### Examples

**Scalar:**
```yaml
ambient.io/ready: "true"
git.ambient.io/base-branch: main
```

**JSON list — peer agents:**
```yaml
ambient.io/peer-agents: '["lead","sdk","reviewer","cp"]'
```

**JSON object — fleet summary:**
```yaml
ambient.io/summary: |
  {
    "as_of": "2026-03-22T00:00:00Z",
    "release": "v0.4.0",
    "status": "active",
    "narrative": "Wave 3 complete. SDK regenerated. CLI and FE unblocked.",
    "blocked_count": 1,
    "ready_count": 8,
    "active_count": 2
  }
```

**Multi-line prose — protocol:**
```yaml
ambient.io/protocol: |
  SESSION START:
    1. Call ambient_get_project → read annotations.
    2. Call ambient_get_agent (self) → read own state.
    3. Update roster entry.
    4. Process inbox.

  SESSION END:
    1. Clear current-task.
    2. Update next-tasks and completed-tasks.
    3. Notify lead.
```

**Multi-line prose — bookmark index:**
```yaml
ambient.io/bookmarks: |
  Implementation guide (waves, pipeline, pitfalls, playbooks):
    docs/internal/design/ambient-model.guide.md

  SDK generator pitfalls:
    docs/internal/design/ambient-model.guide.md#sdk-generator-pitfalls

  Runner SSE proxy pattern:
    docs/internal/design/ambient-model.guide.md#runner-pod-addressing
```

---

## The Bookmark Pattern

Some annotation values are not data — they are **navigational indices** into the codebase
or documentation. Rather than duplicating a 200-line implementation guide into every agent
prompt (wasteful and stale-prone), a single annotation holds a labeled list of file paths
and section anchors. Agent prompts reference the annotation key; agents resolve the file
at runtime.

```
agent.prompt says:
  "SDK generator pitfalls: see bookmark
   ambient.io/bookmarks → 'SDK generator pitfalls'."

         │
         ▼  (agent calls ambient_get_project at session start)

ambient.io/bookmarks contains:
  "SDK generator pitfalls:
     docs/internal/design/ambient-model.guide.md#sdk-generator-pitfalls"

         │
         ▼  (agent reads that file section)

agent has the full pitfall list in context, fresh from source.
```

This pattern also enables annotation values to become **elaborate knowledge stores**:
a multi-line annotation can contain a full command reference, a contract definition, or
an architecture decision record. The annotation is the document; agents consume it on
demand.

---

## Project Annotations

Project annotations are the shared state of the entire fleet. Most are written at
bootstrap by a human and updated at runtime only by the `lead` agent.

```
┌──────────────────────────────────────────────────────────┐
│  Project Annotations                                     │
│                                                          │
│  Bootstrap-time (human sets, rarely changes)            │
│  ─────────────────────────────────────────              │
│  ambient.io/mcp-endpoint    → MCP server SSE URL        │
│  ambient.io/protocol        → session lifecycle rules   │
│  ambient.io/rules           → coding standards          │
│  ambient.io/contracts       → git/api/jira contracts    │
│  ambient.io/bookmarks       → doc/file path index       │
│  ambient.io/fleet           → agent ownership table     │
│  ambient.io/feature-flags   → {"grpc-watch":"enabled"}  │
│  github.com/repo            → org/repo slug             │
│  work.ambient.io/release    → v0.4.0                    │
│                                                          │
│  Runtime (lead writes, agents read)                     │
│  ──────────────────────────────────                     │
│  ambient.io/summary         → current fleet narrative   │
│  ambient.io/agent-roster    → all agent states          │
│  work.ambient.io/epics      → work queue                │
│  work.ambient.io/blocking-issues → escalated blockers   │
│  work.ambient.io/release-status  → in-progress / done   │
└──────────────────────────────────────────────────────────┘
```

### `ambient.io/protocol`

The full session lifecycle. Every agent reads this at session start and follows it. It
defines what to do at session start, session end, on inbox message, on blocker, and on
handoff. See `teams/project.yaml` for the current value.

### `ambient.io/rules`

All coding standards: Go, TypeScript, OpenAPI, Git, Security. Written once at bootstrap.
Agents read this rather than relying on inline prompt text, so the rules are consistent
across the entire fleet and can be updated in one place.

### `ambient.io/contracts`

Git conventions, API source-of-truth rules, coordination thresholds, and Jira integration
settings. Written at bootstrap; updated only when contracts change.

### `ambient.io/bookmarks`

A labeled index of file paths and section anchors. Agents reference this to find deep
documentation without duplicating it in their prompts. New bookmarks are added here when
new guides or playbooks are written.

### `ambient.io/agent-roster`

A JSON array of agent state snapshots, maintained by the `lead` agent. Each entry
reflects the last known state of an agent (ready, blocked, current task, PR URL, etc.).
Individual agents do not write to the roster directly — they update their own agent
annotations, and lead reconciles those into the roster.

Roster entry fields:
```
id, name, ready, blocked, blocker, issue, branch,
current-task, pr-url, pr-status, last-seen
```

### `work.ambient.io/epics`

The work queue — a JSON array of issues assigned to the fleet. Only `lead` writes this.
Agents read it to understand what work exists; they receive individual assignments via
inbox rather than polling the epics list directly.

---

## Agent Annotations

Each agent owns its own annotations. Agents read them at session start to restore state,
update them as work progresses, and write final state at session end.

```
┌──────────────────────────────────────────────────────────┐
│  Agent Annotations                                       │
│                                                          │
│  Bootstrap-time (human sets)                            │
│  ─────────────────────────────────────────             │
│  ambient.io/peer-agents    → ["lead","sdk","reviewer"]  │
│  git.ambient.io/base-branch → "main"                    │
│                                                          │
│  Runtime (agent writes self)                            │
│  ──────────────────────────────────                     │
│  ambient.io/ready          → "true" or "false"          │
│  ambient.io/blocked        → "true" or "false"          │
│  ambient.io/blocker        → "" or description of block │
│  ambient.io/last-handoff   → timestamp of last handoff  │
│  work.ambient.io/current-task  → active task text       │
│  work.ambient.io/next-tasks    → ["task1","task2",...]  │
│  work.ambient.io/completed-tasks → ["done1","done2"...] │
│                                                          │
│  Set on PR open                                         │
│  ──────────────────────────────────                     │
│  work.ambient.io/pr-url    → https://github.com/...     │
│  work.ambient.io/pr-status → "In Review" / "Merged"    │
└──────────────────────────────────────────────────────────┘
```

### `ambient.io/blocked` + `ambient.io/blocker`

When an agent cannot proceed due to an unmet dependency:
1. Set `ambient.io/blocked = "true"`.
2. Write `ambient.io/blocker = "<what is blocking and why>"`.
3. Send inbox messages to the blocking agent and to `lead`.

When unblocked: set `blocked = "false"`, clear `blocker = ""`.

### `work.ambient.io/current-task`

The single task the agent is actively working on. Cleared to `""` at session end.
An agent should have at most one current task. If multiple tasks arrive via inbox, they
go into `next-tasks` and are processed in order.

---

## Session Lifecycle — Annotation Access Pattern

### Session Start

```
ambient_get_project
        │
        ├─► ambient.io/protocol       → how to behave this session
        ├─► ambient.io/rules          → coding standards
        ├─► ambient.io/contracts      → git/api contracts
        ├─► ambient.io/bookmarks      → where to find deep docs
        ├─► ambient.io/agent-roster   → fleet state
        └─► work.ambient.io/epics     → work queue
        │
        ▼
ambient_get_agent (self)
        │
        ├─► ambient.io/blocked        → am I blocked?
        ├─► ambient.io/blocker        → what is blocking me?
        ├─► work.ambient.io/current-task → resume or clear?
        └─► work.ambient.io/next-tasks   → what's queued?
        │
        ▼
ambient_update_agent
        │
        └─► write last-seen timestamp in roster entry
        │
        ▼
ambient_list_inbox_messages
        │
        └─► process any pending assignments or unblock messages
```

### Session End

```
ambient_update_agent
        │
        ├─► work.ambient.io/current-task = "" (cleared)
        ├─► work.ambient.io/next-tasks (updated queue)
        ├─► work.ambient.io/completed-tasks (task appended)
        ├─► ambient.io/ready = "true" / "false"
        └─► ambient.io/blocked = "true" / "false"
        │
        ▼
ambient_send_inbox_message (if handing off or blocked)
        │
        └─► notify lead and/or peer agents
```

---

## Annotation Ownership

| Annotation | Resource | Writer | Reader |
|---|---|---|---|
| `ambient.io/mcp-endpoint` | Project | Human | All agents |
| `ambient.io/protocol` | Project | Human | All agents |
| `ambient.io/rules` | Project | Human | All agents |
| `ambient.io/contracts` | Project | Human | All agents |
| `ambient.io/bookmarks` | Project | Human | All agents |
| `ambient.io/fleet` | Project | Human | All agents |
| `ambient.io/feature-flags` | Project | Human | All agents |
| `ambient.io/summary` | Project | `lead` only | All agents |
| `ambient.io/agent-roster` | Project | `lead` only | All agents |
| `work.ambient.io/epics` | Project | `lead` only | All agents |
| `work.ambient.io/blocking-issues` | Project | `lead` only | All agents |
| `work.ambient.io/release-status` | Project | `lead` only | All agents |
| `ambient.io/ready` | Agent | Self | `lead`, peers |
| `ambient.io/blocked` | Agent | Self | `lead`, peers |
| `ambient.io/blocker` | Agent | Self | `lead`, peers |
| `ambient.io/last-handoff` | Agent | Self | `lead` |
| `ambient.io/peer-agents` | Agent | Human | Self |
| `work.ambient.io/current-task` | Agent | Self | `lead` |
| `work.ambient.io/next-tasks` | Agent | Self, `lead` | Self |
| `work.ambient.io/completed-tasks` | Agent | Self | `lead` |
| `work.ambient.io/pr-url` | Agent | Self | `lead`, `reviewer` |
| `work.ambient.io/pr-status` | Agent | Self | `lead`, `reviewer` |
| `git.ambient.io/base-branch` | Agent | Human | Self |

---

## MCP Tools Reference

All annotation reads and writes happen through MCP. The server endpoint is published in
the project annotation `ambient.io/mcp-endpoint`.

| Tool | Reads/Writes | Who Can Use |
|---|---|---|
| `ambient_get_project` | Reads project + all annotations | All agents |
| `ambient_update_project` | Writes a project annotation | `lead` only |
| `ambient_get_agent` | Reads agent record + all annotations | All agents (own or peer) |
| `ambient_update_agent` | Writes own agent annotations | Self only |
| `ambient_list_inbox_messages` | Reads inbox messages | Self |
| `ambient_send_inbox_message` | Sends message to a peer | All agents |
| `ambient_ignite_agent` | Starts an agent session | `lead` only |

---

## Namespace Conventions

Annotation keys follow a `domain/name` convention:

| Prefix | Domain | Examples |
|---|---|---|
| `ambient.io/` | Core platform state | `ready`, `blocked`, `protocol`, `summary` |
| `work.ambient.io/` | Work tracking | `current-task`, `epics`, `blocking-issues` |
| `git.ambient.io/` | Git context | `base-branch` |
| `github.com/` | GitHub integration | `repo` |

Use `ambient.io/` for any new annotation unless it clearly belongs to a sub-domain.
