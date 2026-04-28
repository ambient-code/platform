# Ambient TUI Spec

**Date:** 2026-04-24
**Status:** Draft
**Component:** `components/ambient-cli/cmd/acpctl/ambient/tui/`
**Depends on:** `ambient-model.spec.md` (data model, API surface, RBAC)

---

## Overview

The Ambient TUI is a full-screen terminal interface for operating the Ambient platform. It is a k9s-inspired resource browser backed by the Ambient API (REST/gRPC), not the Kubernetes API.

**Design intent:** k9s's interaction model — table-first resource browsing, command mode, filtering, drill-down, contextual hotkeys — applied to the Ambient data model. Not a k9s fork. Not a generic K8s browser. A purpose-built operator console for Ambient resources.

**Data source:** Ambient API Server exclusively. No `kubectl` exec, no direct K8s API calls. The TUI is a pure API client — if the API Server doesn't expose it, the TUI doesn't show it.

---

## Principles

| Principle | Rationale |
|-----------|-----------|
| API-only data path | CRDs are going away. The TUI must work against the Ambient API Server, not K8s. This also means the TUI works identically against local, staging, and production — no kubeconfig dependency. |
| k9s keyboard vocabulary | Users already know `:` for command mode, `/` for filter, `d`/`e`/`l`/`y` for actions, `Esc` to back out. Don't invent new muscle memory. |
| Resource-centric navigation | Every screen is a resource list or resource detail. The primary axis is: pick a resource kind → browse instances → drill into one. Same as k9s. |
| Live by default | Tables auto-refresh (5s polling). Session messages stream in real time via SSE. No manual refresh button. |
| Session interaction is first-class | k9s shows pods. Ambient's TUI shows sessions — including live message streaming, sending messages to agents, and watching agent output. This is the differentiator. |
| Respect RBAC | The TUI shows only what the authenticated user can see. API 403s are rendered inline, not as crashes. |
| Offline-safe auth | The TUI reuses `acpctl login` credentials from `~/.config/ambient/config.json`. No separate auth flow. |
| Multi-context | Operators work across local, staging, and production. The TUI saves every server the user has logged into as a named context and supports instant switching — same as k9s with kubeconfig clusters. |
| Sanitize all external content | Agent-produced output is rendered in the terminal. All content from the API is stripped of ANSI escape sequences, terminal control characters, and framework-specific tags before display. |
| Consistent chrome | All views use the same UI structure: hotkey hints in the header, filtering via the global `/` command bar, breadcrumbs at the bottom. No view defines its own bottom status bar or proprietary filter mechanism. Status indicators (Autoscroll, Mode, Phase) for the message stream belong in the sub-header line below the title bar, inside the bordered area. |

---

## Architecture

### Framework

**Bubbletea + bubbles + lipgloss** (Charmbracelet stack).

Rationale:
- `bubbles/table` provides column sorting, selection, scrolling, and keyboard navigation.
- `bubbles/textinput` provides command bar and compose input with cursor management.
- Bubbletea's Elm architecture (Model/Update/View) is well-suited for the TUI's state-heavy navigation (command mode, filter mode, compose mode, detail mode, navigation stack).
- `teatest` provides a programmatic test harness (send keystrokes, assert on output).

### Package Layout

```
cmd/acpctl/ambient/
├── cmd.go                    # entry point — unchanged command registration
└── tui/
    ├── app.go                # top-level bubbletea Program, global keybinds, layout
    ├── config.go             # read acpctl config (multi-context: server, token, project per context)
    ├── client.go             # Ambient API client (extracted from fetch.go, wraps Go SDK)
    ├── events.go             # AG-UI event parsing (extracted from dashboard.go)
    ├── sanitize.go           # strip ANSI escapes, control chars from agent output
    ├── model.go              # root Model — navigation stack, view dispatch
    ├── command.go            # command-mode parser, tab completion, dispatch
    ├── filter.go             # filter-mode parser (regex, inverse, label)
    ├── views/
    │   ├── table.go          # base resource table (wraps bubbles/table, adds sorting + hotkeys)
    │   ├── detail.go         # base detail view (key-value + YAML dump)
    │   ├── projects.go       # project list + detail
    │   ├── agents.go         # agent list + detail
    │   ├── sessions.go       # session list + detail
    │   ├── messages.go       # live session message stream + compose
    │   └── inbox.go          # agent inbox list + compose
    └── tui_test.go           # unit + teatest integration tests
```

### Data Flow

```
┌──────────────────────────────────────────────────────────────────────────┐
│  Context: local [RW]                    <?> Help        _    ___ ___     │
│  Server:  localhost:8000                <:> Command     /_\  / __| _ \   │
│  User:    jsell                         <r> Rename    / _ \| (__|  _/   │
│  Project: ambient-platform                           /_/ \_\\___|_|     │
│  ⟳ 3s                                                                   │
├──────────────────────────────────────────────────────────────────────────┤
│  (command bar appears here on `:` or `/`, hidden by default)             │
├───────────────────────── agents(ambient-platform)[12] ───────────────────┤
│                                                                          │
│              Resource Table / Detail View / Message Stream               │
│              (fills remaining vertical space)                            │
│                                                                          │
├──────────────────────────────────────────────────────────────────────────┤
│  <projects>  <agents>  <sessions>                                        │
│                              Viewing agents in project ambient-platform  │
└──────────────────────────────────────────────────────────────────────────┘
```

Layout follows k9s conventions:
1. **Header block** (top) — context, server, user, project on the left. ASCII branding on the right. Key hints alongside.
2. **Command/filter bar** (below header) — hidden by default. Appears on `:` or `/`, disappears on `Esc` or command execution.
3. **Resource view** (fills remaining space) — table title bar shows resource kind, scope, and count.
4. **Breadcrumb trail** (bottom) — shows navigation path as `<kind>` segments. Current view is the rightmost.
5. **Info line** (very bottom) — contextual description of what's being shown.

```
         │                        ▲
         │  poll / SSE stream     │  tea.Msg
         ▼                        │
   ┌──────────┐            ┌──────────┐
   │ API      │◄──REST────►│ client   │
   │ Server   │◄──gRPC────►│ .go      │
   └──────────┘            └──────────┘
```

All data fetching runs in `tea.Cmd` goroutines. The Bubbletea `Update` loop is never blocked by network calls. API responses arrive as `tea.Msg` values. Errors are displayed inline in the table (red status row) or as a flash message on the status line.

Polling is skip-on-inflight: if the previous poll has not returned, the next tick is skipped. This prevents request stacking under slow API responses.

---

## Navigation Model

### v1 Visual Hierarchy

```
:projects (root)
└── Enter on project
    └── :agents (project-scoped)
        ├── Enter on agent
        │   └── :sessions (agent-scoped)
        │       └── Enter on session
        │           └── :messages (live stream + compose)
        └── i on agent
            └── :inbox (agent-scoped)
                └── m to compose
```

Five views. `:sessions` is also accessible globally (all sessions across all projects), same as k9s's `:pods` showing all pods. `:scheduledsessions` (`:ss`) is accessible via command mode only — it is not part of the Enter drill-down hierarchy.

### Screen Stack

Navigation is a stack. `Enter` pushes a child view. `Esc` pops back to the parent. The breadcrumb in the header shows the stack:

```
Projects > ambient-platform > Agents > be > Sessions > 01HABC > Messages
Projects > ambient-platform > Agents > be > Inbox
```

### Command Mode

`:` opens the command bar (bottom of screen). Tab-completion provides inline suggestions for resource kinds and project names.

| Command | Aliases | Action |
|---------|---------|--------|
| `:projects` | `:proj` | Switch to project list (clears stack) |
| `:agents` | `:ag` | Switch to agent list (current project) |
| `:sessions` | `:se` | Switch to session list (global or scoped) |
| `:inbox` | `:ib` | Switch to inbox (requires agent context) |
| `:scheduledsessions` | `:ss` | Switch to scheduled session list (current project) |
| `:messages` | `:msg` | Switch to message stream (requires session context) |
| `:aliases` | | List all available commands and aliases |
| `:context` | `:ctx` | List all saved contexts |
| `:context <name>` | `:ctx <name>` | Switch to a saved context (server + token + project) |
| `:project <name>` | `:proj <name>` | Switch project within current context |
| `:q` / `:quit` | | Exit |

### Filter Mode

`/` opens the filter bar. Supports:

| Syntax | Behavior | Example |
|--------|----------|---------|
| `/term` | Regex match across all visible columns | `/be-agent` |
| `/!term` | Inverse regex — hide matching rows | `/!completed` |
| `/-l key=val` | Server-side label filter (`@>` containment) | `/-l env=prod` |

`Esc` clears the active filter. Filter syntax follows k9s conventions.

---

## Resource Views

### Project List

| Column | Source | Notes |
|--------|--------|-------|
| NAME | `project.name` | |
| DESCRIPTION | `project.description` | Truncated to fit column width |
| STATUS | `project.status` | |
| AGE | computed from `project.created_at` | Relative (3d, 2h, 5m) |

AGENTS and SESSIONS counts are omitted from v1 — they require N+1 API fan-out queries. A future API aggregation endpoint can enable them.

**Hotkeys:**

| Key | Action | k9s equivalent |
|-----|--------|----------------|
| `Enter` | Drill into project → show agents | Enter |
| `d` | Describe — show project detail (prompt, labels, annotations) | d (describe) |
| `n` | New project (inline name + description prompt) | — |
| `Ctrl-D` | Delete project (confirmation modal) | Ctrl-D |

### Agent List

Scoped to current project context.

| Column | Source | Notes |
|--------|--------|-------|
| NAME | `agent.name` | |
| PROMPT | `agent.prompt` | Truncated to 60 chars |
| SESSION | `agent.current_session_id` | `<none>` if null. Short ID form. |
| PHASE | current session phase | Colored. Requires secondary fetch — see Known N+1 Queries. |
| AGE | computed from `agent.created_at` | Relative |

INBOX unread count is omitted from the table — no count-only API. The inbox view (`i`) shows the full list.

**Hotkeys:**

| Key | Action | k9s equivalent |
|-----|--------|----------------|
| `Enter` | Drill into agent → show sessions for this agent | Enter |
| `d` | Describe — show agent detail (full prompt, labels, annotations, current session) | d |
| `e` | Edit agent prompt (inline text input, PATCHes on save) | e (edit) |
| `s` | Start agent — opens prompt input, calls `POST /start` | — (Ambient-specific, k9s uses `s` for shell) |
| `x` | Stop agent — calls session stop with confirmation | — |
| `i` | Show inbox for this agent | — |
| `m` | Send inbox message (opens compose input) | — |
| `n` | New agent (inline name + prompt) | — |
| `l` | Logs — if session is active, open live message stream | l (logs) |
| `Ctrl-D` | Delete agent (confirmation modal) | Ctrl-D |
| `y` | YAML — dump agent as YAML to screen | y |

### Session List

Accessible globally (`:sessions` — all sessions across all projects) or scoped when drilled in from an agent view.

| Column | Source | Notes |
|--------|--------|-------|
| ID | `session.id` | Short form (first 12 chars) |
| AGENT | agent name | Requires secondary fetch — see Known N+1 Queries. |
| PROJECT | project name | |
| PHASE | `session.phase` | Colored per Phase Colors table |
| TRIGGERED BY | `session.triggered_by_user_id` | |
| STARTED | `session.start_time` | Relative |
| DURATION | `completion_time - start_time` | Running timer if still active |

**Hotkeys:**

| Key | Action | k9s equivalent |
|-----|--------|----------------|
| `Enter` | Drill into session → show live message stream | Enter |
| `d` | Describe — show session detail (full metadata, prompt, conditions) | d |
| `l` | Live message stream (same as Enter) | l |
| `m` | Send message to session (`POST /sessions/{id}/messages`) | — |
| `n` | Start a new session for the current agent (opens prompt input) | — |
| `x` | Interrupt running session (confirmation dialog) | — |
| `y` | YAML — dump session as YAML to screen | y |
| `Ctrl-D` | Delete/cancel session (confirmation modal) | Ctrl-D |

### Message Stream View

**UI consistency:** The message stream follows the same layout conventions as all other views. Keyboard shortcuts are shown in the header (not in a bottom status bar). Filtering uses the global `/` command bar. The only view-specific chrome is the status indicator line below the title bar (Autoscroll, Mode, Phase, SSE status).

#### Data Source

The TUI uses a **dual-stream strategy** for session messages:

1. **Live sessions** (`phase == Running`): Connect to **`GET /sessions/{id}/events`** (AG-UI SSE stream). This proxies raw events from the runner pod, including tool calls (`tool_use`, `tool_result`, `TOOL_CALL_START`, `TOOL_CALL_ARGS`, `TOOL_CALL_RESULT`), text deltas, and system events. This gives operators full visibility into agent activity as it happens.

2. **Historical replay / fallback**: Connect to **`GET /sessions/{id}/messages`** (DB-backed SSE). This endpoint serves durable `user`/`assistant` messages from the API server's database. Used when the session is not running (completed, stopped, failed) or when the `/events` stream fails.

The SDK provides `StreamEvents(ctx, sessionID)` for the live AG-UI stream and `WatchMessages(ctx, sessionID, afterSeq)` for the DB-backed stream. The TUI prefers `/events` for running sessions and falls back to `/messages` for completed sessions or on error.

#### Display Modes

**Conversation mode** (default): Messages rendered as a chat transcript.

```
 ┌─ Session 01HABC... ─ Phase: running ─ Agent: be ─────────────────┐
 │                                                                    │
 │  [user]       Begin. Start with the gRPC handler.                  │
 │  [assistant]  I'll start by implementing the WatchSessionMessages  │
 │               handler. Let me read the existing code...            │
 │  [tool_use]   Read plugins/sessions/handler.go (truncated)         │
 │  [tool_result] ✓ 238 lines                                        │
 │  [assistant]  I can see the handler structure. I'll add the watch  │
 │               endpoint following the existing pattern...           │
 │                                                                    │
 │  ▌ streaming...                                                    │
 ├────────────────────────────────────────────────────────────────────┤
 │  > send message: _                                                 │
 └────────────────────────────────────────────────────────────────────┘
```

**Raw mode** (`r` to toggle): Shows AG-UI events as formatted JSON lines — useful for debugging. "Raw" refers to the unaltered JSON schema/payload structure, not raw terminal bytes. Sanitization is mandatory in all modes: control sequences, ANSI escape codes, and unsafe terminal bytes are stripped from every field value before display, identical to conversation mode.

#### Event Type Rendering

| Event type | Rendering |
|------------|-----------|
| `user` | Full text, white |
| `assistant` | Full text, green. For streaming: accumulate `TEXT_MESSAGE_CONTENT` deltas into a growing line, re-render on each delta. Show `▌` cursor at end until `TEXT_MESSAGE_END`. |
| `tool_use` | One-line summary: tool name + first arg, truncated to terminal width. Dim. |
| `tool_result` | One-line summary: `✓` or `✗` + size. Dim. Expandable via `Enter` on the line (future). |
| `TOOL_CALL_START` | One-line summary: `⚙ tool_name`. Dim. |
| `TOOL_CALL_ARGS` | Tool input args (truncated in default mode, full in pretty mode). Dim. |
| `TOOL_CALL_RESULT` | Tool output content (truncated in default mode, full in pretty mode). Dim. |
| `TOOL_CALL_END` | Suppressed (no visual output). |
| `TEXT_MESSAGE_START` | Suppressed (streaming start marker). |
| `TEXT_MESSAGE_CONTENT` | Delta text, accumulated into the current assistant message. |
| `TEXT_MESSAGE_END` | Suppressed (streaming end marker). |
| `RUN_FINISHED` | `[done]` marker. Dim. |
| `RUN_ERROR` | `✗` + error message. Red. |
| `system` | Full text, yellow |
| `error` | Full text, red |

#### Message Buffer

The message stream maintains a ring buffer (default: 2000 messages). When full, oldest messages are evicted. The user can scroll back within the buffer. Messages older than the buffer are not recoverable without reconnecting with a lower `after_seq` — this is a known limitation.

#### Send-While-Streaming

Sending a message (`m` / `Enter`) while the agent is mid-response is permitted. The `POST /sessions/{id}/messages` call is non-blocking. The human turn appears in the stream when the server echoes it back via SSE, maintaining a single source of truth for message ordering. The compose input does not block or queue — the user types, hits Enter, and the message is sent immediately.

**Hotkeys:**

| Key | Action |
|-----|--------|
| `Esc` | Back to session list |
| `r` | Toggle raw/conversation mode |
| `m` / `Enter` | Focus message input — type and send a human turn |
| `s` | Toggle autoscroll (on by default — view follows new messages; scrolling up disables it, `s` or `G` re-enables) |
| `G` | Jump to bottom + re-enable autoscroll |
| `g` | Jump to top (oldest in buffer) |
| `j`/`k` or `↑`/`↓` | Scroll (disables autoscroll) |
| `/` | Search within messages (regex) |
| `x` | Interrupt current session (confirmation dialog) |
| `c` | Copy selected message text to clipboard (via OSC 52) |

### Inbox View

Scoped to an agent. Accessible via `i` from the agent list or `:inbox` in command mode (requires agent context from navigation stack).

| Column | Source | Notes |
|--------|--------|-------|
| ID | `inbox.id` | Short form |
| FROM | `inbox.from_name` | `(human)` if null |
| BODY | `inbox.body` | Truncated to fit column width |
| READ | `inbox.read` | `✓` / `—` |
| AGE | computed from `inbox.created_at` | Relative |

**Hotkeys:**

| Key | Action |
|-----|--------|
| `Enter` | View full message body in detail pane |
| `m` | Compose new inbox message (opens text input) |
| `r` | Mark selected message as read |
| `Ctrl-D` | Delete message (confirmation) |
| `Esc` | Back to agent list |

### Scheduled Session List

Accessible via `:scheduledsessions` or `:ss` in command mode. Project-scoped. Not part of the Enter drill-down hierarchy (project drill-down goes to agents, not scheduled sessions).

| Column | Source | Notes |
|--------|--------|-------|
| NAME | `scheduled_session.name` | |
| SCHEDULE | `scheduled_session.schedule` | Cron expression |
| AGENT | agent name | Resolved from agent_id |
| PROJECT | `scheduled_session.project_id` | |
| SUSPENDED | `scheduled_session.suspend` | `Yes` / `No` |
| LAST RUN | `scheduled_session.last_schedule_time` | Relative |
| AGE | computed from `scheduled_session.created_at` | Relative |

**Hotkeys:**

| Key | Action | k9s equivalent |
|-----|--------|----------------|
| `Enter` | Show runs (sessions created by this schedule) | Enter |
| `d` | Describe — show detail view | d |
| `n` | New scheduled session (name, schedule, agent) | — |
| `s` | Suspend/resume toggle | — |
| `t` | Trigger manual run | — |
| `Ctrl-D` | Delete (confirmation dialog) | Ctrl-D |
| `Esc` | Back | Esc |

---

## Global Keybindings

These work on every screen:

| Key | Action | k9s equivalent |
|-----|--------|----------------|
| `:` | Command mode | `:` |
| `/` | Filter mode | `/` |
| `?` | Help overlay — show keybindings for current view | `?` |
| `Esc` | Pop navigation stack / clear filter / close modal | `Esc` |
| `q` | Quit (from root view) or pop (from child view) | `q` |
| `Ctrl-C` | Quit immediately | `Ctrl-C` |
| `c` | Copy selected row's ID to clipboard (OSC 52) | — |
| Scroll wheel | Scroll up/down in tables and message stream | Scroll wheel |
| `0`-`9` | Switch project by number (shown in header) | — (Ambient-specific, matches k9s namespace switching) |
| `Shift-N` | Sort by name column | `Shift-N` |
| `Shift-A` | Sort by age column | `Shift-A` |

Column sorting uses k9s's Shift-key convention. Additional sort keys are defined per view where meaningful.

---

## Screen Layout

Follows k9s layout conventions: header block at top, command bar on demand, resource table fills the middle, status hints at bottom.

### Header Block (top, multi-line)

```
 Context: local [RW]                     <?> Help
 Server:  localhost:8000                 <:> Command
 User:    jsell                          <r> Rename
 Project: ambient-platform
 ⟳ 3s
```

Left side — context metadata (k9s style, stacked key-value):
- **Context** name + read/write indicator
- **Server** URL
- **User** (from `whoami`)
- **Project** (current project context)
- **Refresh indicator** — seconds since last successful fetch. Shows `(stale)` if >15s.

Right side — ASCII art branding + key hints.

Between the left-side metadata and the right-side hotkey hints, the header shows numbered project shortcuts for quick switching (matching k9s's namespace number keys):

```
<0> all       <1> test       <2> test-jsell       <s> Start  <d> Describe     <?>  Help
                                                    <x> Stop   <i> Inbox        <:>  Command
                                                    <l> Logs   <n> New          </>  Filter
```

Projects are numbered in alphabetical order. `<0>` always means "all" (unscoped). Pressing a number key instantly switches the project context without entering command mode.

The right side of the header shows contextual hotkeys that change based on the active view, displayed to the left of the static `<?>`, `<:>`, `</>` hints. For example, in the agents view:

```
<s> Start  <x> Stop  <i> Inbox  <d> Describe     <?>  Help
<e> Edit   <l> Logs  <n> New    <Ctrl-D> Delete   <:>  Command
                                                   </>  Filter
```

Each view shows only its relevant hotkeys. The hotkeys are rendered in dim text with the key in angle brackets.

### Command/Filter Bar

Hidden by default. Appears when the user presses `:` (command mode) or `/` (filter mode). Renders between the header and the resource table:

```
┌───────────────────────────────────────────────────────────────────┐
│ :sessions                                                         │
└───────────────────────────────────────────────────────────────────┘
```

Disappears on `Esc` or after command execution, returning the space to the resource table.

### Resource Table Title

The table has a title bar showing resource kind, scope, and count — matching k9s's `contexts(all)[12]` convention:

```
┌──────────────────────── agents(ambient-platform)[12] ─────────────┐
│ NAME↑           PROMPT                    SESSION    PHASE    AGE  │
```

Scope shown in parentheses:
- `sessions(all)[47]` — global view
- `sessions(be)[3]` — scoped to agent `be`
- `inbox(be)[5]` — scoped to agent `be`

### Breadcrumb Trail (bottom)

```
 <projects>  <agents>  <sessions>
```

Shows the navigation stack as `<kind>` segments, matching k9s's bottom breadcrumb. Each segment represents a level in the drill-down. The current (rightmost) view is the active one. Clicking/selecting a parent segment is not supported (keyboard-only — use `Esc` to pop back).

### Info Line (very bottom)

```
                        Viewing agents in project ambient-platform
```

Ephemeral toast — appears for 5 seconds on navigation changes, then fades (line clears). Triggered by:
- Entering a new view (drill-down or command switch)
- Switching context (`:ctx`)
- Applying or clearing a filter
- Errors (API failures, permission denied) — these persist until the next action rather than auto-clearing

Examples:
- `Viewing agents in project ambient-platform`
- `Streaming messages for session 01HABC...`
- `Switched to context staging`
- `✗ disconnected — retrying (backoff: Xs)` (persists)

---

## Refresh Strategy

| Resource | Method | Interval |
|----------|--------|----------|
| Projects, Agents, Inbox | REST `GET` polling | 5s (hardcoded) |
| Sessions | gRPC `WatchSessions` stream; fallback to REST polling | Real-time / 5s |
| Session Messages (live) | AG-UI SSE stream (`GET /sessions/{id}/events`) | Real-time |
| Session Messages (replay) | DB-backed SSE (`GET /sessions/{id}/messages`) | Real-time |

Polling is **skip-on-inflight**: if the previous request has not completed, the next tick is skipped. This prevents request stacking under degraded API conditions.

When a view is not visible (user has drilled into a child), its polling pauses. Polling resumes when the user navigates back.

---

## Error Handling

| Scenario | Behavior |
|----------|----------|
| **API unreachable** | Status line: `✗ disconnected — retrying (backoff: Xs)`. Tables show stale data. Header shows `(stale Ns)` with seconds since last successful fetch. Exponential backoff with jitter: start at 1s, double each attempt (1s, 2s, 4s, …), cap at 30s, reset to 1s on a successful fetch. Same algorithm as SSE stream disconnect. No retry limit — the TUI retries indefinitely. |
| **401 Unauthorized** | Attempt to re-read token from `~/.config/ambient/config.json` (another session may have refreshed it). If still 401, status line: `✗ session expired — run 'acpctl login' in another terminal`. Stale data preserved. No modal, no forced exit. |
| **403 Forbidden (resource)** | Inline in table: row shows `ACCESS DENIED` for the specific resource. |
| **403 Forbidden (kind)** | Table-level message: `Insufficient permissions to list <kind>`. Distinct from empty results. |
| **404 Not Found** | Flash message on status line. Resource removed from table on next refresh. |
| **429 Rate Limited** | Back off to `Retry-After` header value (or 30s default). Status line: `⏳ rate limited — backing off`. |
| **5xx Server Error** | Status line shows error summary. Stale data preserved. Retry on next poll cycle. |
| **SSE stream disconnect** | Auto-reconnect with exponential backoff (1s, 2s, 4s, max 30s). Reconnect status shown inline in message stream: `⟳ reconnecting (attempt 3)...`. On reconnect, replay from last received `seq` via `after_seq` parameter. |

---

## Security

| Concern | Mitigation |
|---------|------------|
| **Terminal escape injection** | All agent-produced content (session messages, agent prompts, inbox bodies) is sanitized before rendering. Strip ANSI escape sequences (`\x1b[...`), OSC sequences, C0/C1 control characters, and lipgloss/tview region tags. Implemented in `sanitize.go`. |
| **TLS enforcement** | The TUI refuses plaintext HTTP connections to non-localhost servers by default. `--insecure` flag required to override. Consistent with `acpctl` CLI behavior. |
| **Tokens on disk** | Reuses `acpctl` config at `~/.config/ambient/config.json` with 0600 file permissions (set by `acpctl login`). Contains tokens for all saved contexts. No encryption at rest — file permissions are the defense. Tokens are never logged; `Config.String()` / `Config.GoString()` redact all token fields. |
| **Token in crash output** | `Config` struct implements `fmt.Stringer` and `fmt.GoStringer` to redact `AccessToken`. Panic recovery in `app.go` catches panics and exits cleanly without dumping the model. |
| **Inline editing** | Prompt editing uses inline `bubbles/textinput` (no temp files, no `$EDITOR` subprocess). Content stays in memory. |
| **Credential tokens** | The TUI never calls the credential token endpoint. Credential views show metadata only. |

---

## Configuration

The TUI reads from the same config file as `acpctl`:

```json
// ~/.config/ambient/config.json
{
  "current_context": "local",
  "contexts": {
    "local": {
      "server": "http://localhost:8000",
      "access_token": "eyJ...",
      "project": "ambient-platform"
    },
    "staging": {
      "server": "https://api.staging.ambient.io",
      "access_token": "eyJ...",
      "project": "ambient-platform"
    },
    "prod": {
      "server": "https://api.ambient.io",
      "access_token": "eyJ...",
      "project": "fleet-prod"
    }
  }
}
```

### Context Management

Contexts are auto-created and auto-named by `acpctl login`. The context name is derived from the server hostname:

| Server URL | Auto-generated context name |
|------------|---------------------------|
| `http://localhost:8000` | `local` |
| `https://api.staging.ambient.io` | `staging.ambient.io` |
| `https://api.ambient.io` | `api.ambient.io` |

Rules:
- `localhost` (any port) → `local`
- All other servers → hostname portion of the URL
- If a context with the same name exists, `acpctl login` updates it (token, project) rather than creating a duplicate.
- `acpctl login` sets `current_context` to the newly logged-in context.
- `acpctl logout` removes the current context entry. If other contexts remain, `current_context` is set to the lexically first remaining context name (sorted ascending). This is a stable, deterministic selection — independent of insertion order or platform map iteration.

In the TUI:
- `:ctx` with no argument lists all contexts in a table (name, server, project, active indicator).
- `:ctx <name>` switches immediately — the TUI reconnects to the new server, re-fetches all data, and updates the header. Navigation stack is reset to `:projects`.
- Tab-completion on `:ctx` suggests saved context names.

No other TUI-specific config in v1. Refresh interval is hardcoded at 5s. Message buffer is hardcoded at 2000.

---

## Phase Colors

Carried forward from the existing TUI (`view.go`). These are ANSI 256-color indices, consistent across lipgloss and any terminal that supports 256-color mode.

| Phase | Color | ANSI 256 Index | Lipgloss |
|-------|-------|----------------|----------|
| `pending` | Yellow | 33 | `Color("33")` |
| `running` | Orange | 214 | `Color("214")` |
| `succeeded` / `completed` | Dim grey | 240 | `Color("240")` |
| `failed` | Red | 31 | `Color("31")` |
| `cancelled` | Dim grey | 240 | `Color("240")` |

Full palette (preserved from existing code):

| Name | ANSI 256 | Usage |
|------|----------|-------|
| Orange | 214 | Branding, navigation highlights, selected items |
| Cyan | 36 | Secondary accent |
| Green | 28 | Success indicators |
| Red | 31 | Failed/error phase, delete confirmations |
| Yellow | 33 | Pending phase, in-progress indicators |
| Dim | 240 | Inactive items, separators, hints |
| White | 255 | Primary text |
| Blue | 69 | Command mode, links |

### Row Coloring

Following k9s conventions, entire table rows are colored based on resource phase/status — not just the PHASE column. This provides at-a-glance visibility into fleet health.

| Phase | Row Color | ANSI 256 |
|-------|-----------|----------|
| `running` / `active` | Orange | 214 |
| `pending` | Yellow | 33 |
| `failed` | Red | 31 |
| `succeeded` / `completed` | Dim grey | 240 |
| `idle` / `cancelled` | Dim grey | 240 |

**Selected row highlight:** The selected row uses the phase color as the **background** with black (0) foreground text. The highlight spans the full row width border-to-border. For rows without a phase (projects, contexts), the default orange (214) background is used.

---

## Known API Gaps

These are gaps where the TUI spec requires data the API does not provide efficiently. They are accepted tradeoffs for v1, not blockers.

| Gap | Impact | Workaround | Permanent Fix |
|-----|--------|------------|---------------|
| Agent phase (current session) | Agent table PHASE column requires `GET /sessions/{id}` per agent with `current_session_id` | Fan-out fetch; cached for 5s per poll cycle | Denormalize `phase` onto Agent response |
| Agent name on session | Session table AGENT column requires agent name resolution | Cache agent ID→name map per project; refresh with agent list | Denormalize `agent_name` onto Session response |
| Inbox unread count | No count-only endpoint | Omitted from agent table in v1; visible in inbox view | Add `unread_count` to Agent response or `?count_only=true` param |
| Project agent/session counts | No aggregation endpoint | Omitted from project table in v1 | Add counts to Project list response |

---

## Content Handling

| Content type | Strategy |
|-------------|----------|
| Long text (prompts, message bodies) | Wrap at terminal width. No horizontal scrolling. Detail views show full text with vertical scroll. |
| Long single-line values (URLs, IDs) | Truncate with `…` in table columns. Full value shown in detail view and via `c` (copy). |
| Wide tables (many columns) | Columns have priority. Low-priority columns are hidden when terminal is narrow. |
| Tool use/result payloads | One-line summary in conversation mode. Full payload in raw mode or detail view. |

---

## What This Spec Does NOT Cover

| Topic | Why | Revisit When |
|-------|-----|-------------|
| K8s resource browsing (pods, namespaces) | Not the TUI's job post-CRD-transition. Use k9s. | Never — not in scope. |
| Credential view | Credential CRUD API is not yet implemented in the API server. | API lands. |
| RBAC views (roles, rolebindings) | Low-frequency operation. `acpctl get roles` is sufficient. | User demand. |
| Diagnostic view for failed sessions | Requires API to surface container exit codes, OOM events, failure reasons — not just `phase=failed`. | API exposes failure diagnostics. |
| Mouse click/drag | Keyboard-driven, consistent with k9s. | Never. |
| Plugin/extension system | Premature. Resource kinds are still evolving. | Resource model stabilizes. |
| Theme customization | One color palette (see Phase Colors). | User demand. |
| `$EDITOR` integration | Inline editing via `bubbles/textinput` is simpler and avoids temp file security concerns. | User demand for multi-line editing. |

---

## Implementation Priority

Each wave produces a **shippable `acpctl ambient`** — the binary is usable at the end of every wave, not just scaffolding.

| Wave | Scope | Deliverable |
|------|-------|-------------|
| **0** | `client.go`, `events.go`, `sanitize.go` foundation modules. `bubbles/table`-based project list. Multi-context config format (`contexts` map, `current_context`). | Launches, shows projects in a real table. `acpctl login` auto-creates named context. Smoke-tests pass via `teatest`. |
| **1** | Agent table + command mode (`:projects`, `:agents`, `:sessions`, `:aliases`, `:ctx`, `:project`, `:q`) with tab completion. `:ctx` lists/switches contexts. `/` filter (regex + inverse). Navigation stack (Enter/Esc push/pop). Breadcrumb. Column sorting (Shift-key). | Two-resource browser with full k9s navigation feel. Context switching works. |
| **2a** | Session table (global + agent-scoped). Read-only message stream view via `/messages` SSE. Conversation + raw mode toggle. | Operators can watch agent work in real time. |
| **2b** | Send message (`POST /sessions/{id}/messages`). Streaming partial response rendering (delta accumulation). SSE reconnect with `after_seq` replay. Copy-to-clipboard (`c`). | Full interactive session experience. |
| **3** | Inbox view. Detail views (`d`) for all resources. Agent start (`s`) and stop (`x`). Agent inline edit (`e`). New project/agent (`n`). Delete (`Ctrl-D`). | Full CRUD + inbox. Feature-complete v1. |

---

## Test Strategy

| Layer | What | How | Required per wave |
|-------|------|-----|-------------------|
| **Unit** | Command parser, filter parser, event type rendering, phase color mapping, breadcrumb builder, sanitize logic | Standard Go table-driven tests | All waves |
| **Integration (happy path)** | API client → `httptest` server with fixture JSON → table populated correctly | `teatest`: send keystrokes, assert on rendered output containing expected rows | Wave 0+ |
| **Integration (error paths)** | 401 re-read, 403 kind-level message, 429 backoff, SSE disconnect+reconnect | `httptest` returning error codes; `teatest` asserting status line messages | Wave 2a+ |
| **Navigation** | Enter→drill→Esc→back, command mode `:sessions`→`:agents`, filter→clear | `teatest`: send key sequences, assert on breadcrumb and table content | Wave 1+ |
| **Performance** | Table render time with 500 rows, SSE throughput with rapid deltas | Benchmark tests (`testing.B`) with fixture data | Wave 2a+ |
| **Manual** | Full flow: launch → navigate → filter → drill → send message → back out | Checklist per wave, run against kind cluster | All waves |

---

## CLI Reference

| Command | Description | Status |
|---------|-------------|--------|
| `acpctl ambient` | Launch interactive TUI | ✅ |
