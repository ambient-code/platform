# Agent Script — Visual Language Proposal

**Date:** 2026-03-22
**Status:** Proposal

---

## Motivation

Scratch proves that a small number of composable visual constructs — hat blocks (triggers), stack blocks (actions), control blocks (if/repeat), and broadcast blocks (messages) — are sufficient to express arbitrarily complex behavior. The same reduction applies to agent orchestration. An agent session is just a trigger, a sequence of actions, some branching, and message passing. That is the complete primitive.

This proposal defines a minimal visual block language for agent scripts that:

1. Renders as Mermaid diagrams (readable in any markdown viewer)
2. Maps 1-for-1 onto the PASM model (Project → Agent → Session → Message)
3. Can be stored as `Agent.prompt` or `Session.prompt` and executed by the runner

---

## The Block Palette

Six block types. No more.

| Block | Scratch Equivalent | Colour |
|---|---|---|
| `WHEN` | Hat block — trigger | Yellow |
| `DO` | Stack block — action | Blue |
| `IF / ELSE` | Control block | Orange |
| `REPEAT` | Loop block | Orange |
| `SEND` | Broadcast block | Purple |
| `WAIT` | Wait-for-broadcast block | Purple |

---

## Mermaid Notation

### Single agent script — ignition flow

```mermaid
flowchart TD
    HAT([🟡 WHEN ignited])
    A[🔵 DO: read CLAUDE.md\nand ambient-data-model.md]
    B{🟠 IF inbox\nhas messages}
    C[🔵 DO: process each\ninbox message as a task]
    D[🔵 DO: pick next task\nfrom session.prompt]
    E[🔵 DO: implement it]
    F[🔵 DO: run tests]
    G{🟠 IF tests fail}
    H[🟣 SEND @overlord\n'blocked on task']
    I[🟣 WAIT for 'unblocked'\nfrom @overlord]
    J{🟠 REPEAT until\ntasks done}
    K[🟣 SEND @overlord\n'session complete']
    L[🔵 DO: post summary\nto blackboard]
    END([🔴 END])

    HAT --> A --> B
    B -- yes --> C --> J
    B -- no --> J
    J -- next task --> D --> E --> F --> G
    G -- yes --> H --> I --> D
    G -- no --> J
    J -- done --> K --> L --> END
```

---

### Multi-agent message passing — broadcast model

```mermaid
sequenceDiagram
    actor Human
    participant Overlord
    participant BE as Backend Engineer
    participant Reviewer

    Human->>Overlord: WHEN ignited (session.prompt = "ship feature X")

    Note over Overlord: REPEAT for each agent in fleet
    Overlord->>BE: SEND "implement session messages handler"
    Overlord->>Reviewer: SEND "review PR when ready"

    Note over BE: WHEN inbox receives message
    BE->>BE: DO: implement
    BE->>BE: DO: run tests
    BE->>Reviewer: SEND "PR #142 ready for review"

    Note over Reviewer: WAIT for "PR ready"
    Reviewer->>Reviewer: DO: review PR
    Reviewer->>BE: SEND "approved" OR "needs changes"

    alt approved
        BE->>Overlord: SEND "feature X complete"
    else needs changes
        BE->>BE: DO: apply changes
        BE->>Reviewer: SEND "PR updated"
    end

    Overlord->>Human: SEND "session complete"
```

---

### The six blocks as a grammar

```mermaid
flowchart LR
    subgraph Triggers["🟡 Triggers (Hat Blocks)"]
        W1([WHEN ignited])
        W2([WHEN inbox receives msg])
        W3([WHEN session.phase = running])
    end

    subgraph Actions["🔵 Actions (Stack Blocks)"]
        D1[DO: read file]
        D2[DO: write code]
        D3[DO: run tests]
        D4[DO: post to blackboard]
    end

    subgraph Control["🟠 Control"]
        C1{IF condition}
        C2{REPEAT n times}
        C3{REPEAT until done}
    end

    subgraph Messages["🟣 Messages (Broadcast)"]
        M1[SEND @agent msg]
        M2[WAIT for msg]
        M3[BROADCAST to all]
    end

    Triggers --> Actions
    Actions --> Control
    Control --> Messages
    Messages --> Actions
```

---

### Full engineering workflow — feature delivery with escalation and retry

A longer example showing nested control, multi-agent coordination, retry loops, and final reconciliation.

```mermaid
flowchart TD
    HAT([🟡 WHEN ignited\nsession.prompt = 'ship feature X'])

    subgraph Bootstrap["Bootstrap"]
        A1[🔵 DO: read CLAUDE.md\nambient-data-model.md\nBOOKMARKS.md]
        A2[🔵 DO: read open PRs\nand recent commits]
        A3{🟠 IF inbox\nhas messages}
        A4[🔵 DO: triage inbox\nadd to task queue]
    end

    subgraph Plan["Plan"]
        B1[🔵 DO: decompose feature X\ninto subtasks]
        B2[🔵 DO: order by dependency]
        B3[🔵 DO: estimate risk\nper subtask]
        B4{🟠 IF any subtask\nhas high risk}
        B5[🟣 SEND @overlord\n'risk detected — awaiting go/no-go']
        B6[🟣 WAIT for 'proceed'\nor 'skip' from @overlord]
    end

    subgraph Implement["Implement — REPEAT until queue empty"]
        C1[🔵 DO: pick next subtask]
        C2[🔵 DO: write code]
        C3[🔵 DO: write tests]
        C4[🔵 DO: run tests]
        C5{🟠 IF tests pass}
        C6{🟠 IF retry count\n≥ 3}
        C7[🔵 DO: increment\nretry count]
        C8[🔵 DO: read error\nrevise approach]
        C9[🟣 SEND @overlord\n'subtask stuck after 3 attempts']
        C10[🟣 WAIT for 'hint'\nor 'skip' from @overlord]
        C11[🔵 DO: mark subtask done\nreset retry count]
    end

    subgraph Review["Review"]
        D1[🔵 DO: open PR]
        D2[🟣 SEND @reviewer\n'PR ready for review']
        D3[🟣 WAIT for 'approved'\nor 'changes-requested']
        D4{🟠 IF approved}
        D5[🔵 DO: apply\nreview feedback]
        D6[🔵 DO: re-run tests]
    end

    subgraph Ship["Ship"]
        E1[🔵 DO: merge PR]
        E2[🔵 DO: post deploy note\nto blackboard]
        E3[🟣 SEND @overlord\n'feature X shipped']
        END([🔴 END])
    end

    HAT --> A1 --> A2 --> A3
    A3 -- yes --> A4 --> B1
    A3 -- no --> B1
    B1 --> B2 --> B3 --> B4
    B4 -- yes --> B5 --> B6 --> C1
    B4 -- no --> C1

    C1 --> C2 --> C3 --> C4 --> C5
    C5 -- pass --> C11 --> C1
    C5 -- fail --> C6
    C6 -- no --> C7 --> C8 --> C2
    C6 -- yes --> C9 --> C10 --> C2

    C11 -- queue empty --> D1
    D1 --> D2 --> D3 --> D4
    D4 -- yes --> E1 --> E2 --> E3 --> END
    D4 -- no --> D5 --> D6 --> D2
```

---

## Mapping to PASM

Every block maps to a field in the data model:

| Block | PASM Field | Notes |
|---|---|---|
| `WHEN ignited` | `Session.prompt` trigger | The ignition event starts the script |
| `DO` | `SessionMessage` (user turn) | Each action becomes a message turn |
| `IF` | Inline in `Session.prompt` | Expressed as conditional instruction text |
| `REPEAT` | Inline in `Session.prompt` | Loop expressed as iterative instruction |
| `SEND @agent msg` | MCP `push_message` / `POST /agents/{id}/inbox` | Writes to recipient's Inbox or spawns a child session via @mention |
| `WAIT for msg` | MCP `watch_session_messages` / `GET /agents/{id}/inbox` | Streams or polls inbox for a matching message |

---

## Purple Blocks Are MCP Calls

The purple SEND and WAIT blocks are not abstract — they map directly to MCP tool calls available to every agent via the sidecar:

| Block | MCP Tool | What it does |
|---|---|---|
| `SEND @agent "task"` | `push_message` with `@mention` | Resolves the agent, spawns a child session, passes the task as `prompt` |
| `SEND @overlord "status"` | `push_message` | Pushes a user message to the Overlord's session |
| `WAIT for "approved"` | `watch_session_messages` | Subscribes to the session stream; unblocks when matching message arrives |
| `BROADCAST to fleet` | `push_message` × N | One `push_message` per agent in the fleet; each spawns a parallel child session |
| `READ project state` | `get_project` | Reads project annotations as shared state |
| `WRITE agent state` | `patch_agent_annotations` | Writes durable key-value state to the agent |

This means every purple block in a Mermaid diagram corresponds to a concrete function call the runner can make. The block language is not metaphorical — it is directly executable via MCP.

---

## Annotations as Programmable State

Annotations on projects, agents, and sessions form a three-level scoped state store accessible from any MCP call:

| Scope | MCP Tool | Lifetime | Use Case |
|---|---|---|---|
| `Session.annotations` | `patch_session_annotations` | Session lifetime | In-flight task status, retry count, current step |
| `Agent.annotations` | `patch_agent_annotations` | Persistent (survives sessions) | Last completed task, accumulated index SHA, external IDs |
| `Project.annotations` | `patch_project_annotations` | Project lifetime | Feature flags, fleet configuration, cross-agent handoff state |

Because annotations are readable via `get_session` / `get_agent` / `get_project` and writable via MCP patch tools, any external application — a CI pipeline, a webhook handler, a frontend dashboard — can read and write agent state using only the REST API. No custom database required. The platform is the state store.

### Example: cross-agent handoff via annotations

```
BE agent (session A):
  SEND patch_agent_annotations({
    "myapp.io/last-pr": "PR #142",
    "myapp.io/status": "review-requested"
  })
  SEND push_message(@reviewer, "PR #142 ready")
  WAIT watch_session_messages → "approved"
  SEND patch_agent_annotations({"myapp.io/status": "shipping"})

Reviewer agent (session B):
  WAIT watch_session_messages → "PR #142 ready"
  DO: review
  SEND push_message(@be, "approved")
  SEND patch_agent_annotations({"myapp.io/status": "idle"})

External CI pipeline:
  GET /api/ambient/v1/projects/{id}/agents/{be-id}
  → reads annotations["myapp.io/status"] = "shipping"
  → triggers deploy job
```

Any application can be built on top of Ambient this way. The agents are the compute. The annotations are the state. The inbox and session messages are the bus.

---

The full script above serialises as `Session.prompt`:

```
## When ignited

- do: read CLAUDE.md and ambient-data-model.md
- if inbox has messages:
    - do: process each inbox message as a task
- repeat until tasks done:
    - do: pick next task
    - do: implement it
    - do: run tests
    - if tests fail:
        - send: @overlord "blocked on {task}"
        - wait: "unblocked" from @overlord
- send: @overlord "session complete"
- do: post summary to blackboard
```

---

## Why This Works

Scratch's insight was that **most programs are just event → sequence → branch → loop → message**. Agent scripts are no different. The runner already executes free-text prompts as instructions — this proposal just adds lightweight structure so that:

1. Humans can read and edit scripts visually (Mermaid renders in GitHub, Obsidian, Notion)
2. Agents can parse and generate scripts in the same format they receive
3. The Overlord can compose multi-agent workflows by wiring SEND/WAIT blocks between agents
4. A future UI can render the Mermaid diagram live as the session executes, highlighting the current block

The block language is not a new runtime. It is structured `Session.prompt`. The runner executes it as natural language. The structure is for humans and orchestrators, not the interpreter.
