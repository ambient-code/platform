# Project Intelligence Memory

**Status**: Proposal
**Author**: Yossi Ovadia
**Date**: 2026-04-09

---

## 1. Problem Statement

Every Ambient session starts with amnesia. An agent that spent 45 minutes mapping a repository's architecture, identifying a flaky test root cause, or reviewing a complex PR produces valuable findings — then discards all of it when the session ends. The next session on the same project starts from zero.

This creates three concrete problems:

| Problem | Impact | Frequency |
|---------|--------|-----------|
| **Redundant analysis** | Agent re-discovers architecture, conventions, bug-prone areas every session | Every session |
| **Lost investigation context** | "File X was the root cause of bug #42" is forgotten; next agent touching X has no warning | Every bug investigation |
| **No accountability signal** | Agent said "this PR looks safe" — was it? No feedback loop exists to calibrate future confidence | Every verdict |

The GPS MCP server solved a similar problem for *organizational* data (people, teams, issues) by materializing it into a queryable cache. Project Intelligence Memory solves the same problem for *project-level technical knowledge* — the findings agents produce while working on code.

---

## 2. Goals

1. **Persistent project knowledge** — Architecture analysis, coding conventions, dependency maps, and known problem areas survive across sessions
2. **File-scoped investigation findings** — When an agent investigates a bug or reviews a PR, file→finding mappings are stored; future sessions touching those files receive contextual warnings
3. **Outcome tracking** — After an agent's verdict (bug confirmed, PR approved, fix suggested), track what actually happened; build a calibration signal
4. **Audit trail** — Every memory mutation is logged with actor, reason, and timestamp; no silent overwrites
5. **Zero disruption** — Existing sessions, APIs, and operator behavior are completely unaffected

## 3. Non-Goals

- **Cross-project memory sharing** — Memory is strictly project-scoped. Cross-project patterns are a future concern.
- **Automated outcome resolution** — V1 records outcomes manually (API call or agent action). Webhook-driven resolution (GitHub/Jira event → outcome update) is Phase 2.
- **Memory-driven autonomous actions** — Memory informs agents; it does not trigger sessions, create issues, or take action on its own.
- **Replacing CLAUDE.md / system prompts** — Static project instructions belong in CLAUDE.md. Memory stores *discovered* knowledge that changes over time.
- **Full-text code indexing** — Memory stores findings *about* code, not the code itself. Code search is a separate capability.

---

## 4. Architecture

### 4.1 Conceptual Model

Memory is **application data**, not runtime state. It does not need Kubernetes CRDs, operator reconciliation, or pod-level resources. It lives in PostgreSQL (via ambient-api-server), is exposed as a REST API, and is consumed by sessions through MCP tools.

```
┌─────────────────────────────────────────────────────────────┐
│                    Session (Runner Pod)                      │
│                                                             │
│  ┌─────────────┐    ┌──────────────────────────────────┐   │
│  │ Claude SDK   │◄──►│  ACP MCP Server (platform tools) │   │
│  │ / Gemini CLI │    │                                  │   │
│  └─────────────┘    │  memory_query    ← search/list   │   │
│                      │  memory_store    ← create/update │   │
│                      │  memory_warn     ← file warnings │   │
│                      │  memory_outcome  ← record result │   │
│                      └───────────┬──────────────────────┘   │
│                                  │ HTTP                      │
└──────────────────────────────────┼──────────────────────────┘
                                   │
                    ┌──────────────▼──────────────┐
                    │     ambient-api-server       │
                    │     (Go, REST API)           │
                    │                              │
                    │  GET  /memory_entries         │
                    │  POST /memory_entries         │
                    │  PATCH /memory_entries/{id}   │
                    │  GET  /memory_outcomes        │
                    │  POST /memory_outcomes        │
                    │  GET  /memory_audit_log       │
                    └──────────────┬───────────────┘
                                   │
                    ┌──────────────▼──────────────┐
                    │       PostgreSQL              │
                    │                              │
                    │  memory_entries    (findings) │
                    │  memory_outcomes   (verdicts) │
                    │  memory_audit_log  (history)  │
                    └──────────────────────────────┘
```

### 4.2 Data Flow: Writing Memory

```
Agent analyzes repo architecture
  → Agent calls memory_store MCP tool
    → Runner POST /api/ambient/v1/memory_entries
      → API server validates, writes to PostgreSQL
        → Audit log entry created (trigger)
          → 201 Created returned to agent
```

### 4.3 Data Flow: Reading Memory

Two paths — **proactive** (push) and **reactive** (pull):

```
PROACTIVE (session startup):
  Runner starts
    → Runner calls GET /memory_entries?project_id=X&scope_ref=<repo_paths>&status=active
      → API server returns relevant entries
        → Runner injects summary into session context
          → Agent starts with awareness of known findings

REACTIVE (during session):
  Agent touches file components/backend/handlers/sessions.go
    → Agent calls memory_warn("components/backend/handlers/sessions.go")
      → Runner GET /memory_entries?scope_ref=...sessions.go&status=active
        → Returns: "This file was root cause of session timeout bug (2026-03-15)"
          → Agent proceeds with caution
```

### 4.4 Component Boundaries

```
┌──────────────────────────────────────────────────────────────┐
│ ambient-api-server (Go)                                      │
│  plugins/memoryEntries/     ← NEW: CRUD + search + lifecycle │
│  plugins/memoryOutcomes/    ← NEW: outcome recording         │
│  plugins/memoryAuditLog/    ← NEW: read-only audit trail     │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│ ambient-runner (Python)                                      │
│  bridges/claude/mcp.py      ← MODIFIED: add memory tools    │
│  bridges/gemini_cli/mcp.py  ← MODIFIED: add memory tools    │
│  platform/memory.py         ← NEW: memory API client        │
│  platform/memory_tools.py   ← NEW: MCP tool definitions     │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│ ambient-api-server OpenAPI spec                              │
│  openapi/openapi.yaml       ← MODIFIED: memory schemas      │
└──────────────────────────────────────────────────────────────┘
```

No changes to: backend, operator, frontend, manifests, CRDs.

---

## 5. Data Model

### 5.1 PostgreSQL Schema

#### `memory_entries` — Core knowledge store

```sql
CREATE TABLE memory_entries (
    id              TEXT PRIMARY KEY,           -- KSUID (api.NewID())
    project_id      TEXT NOT NULL REFERENCES projects(id),

    -- Attribution
    session_id      TEXT REFERENCES sessions(id),
    agent_id        TEXT REFERENCES agents(id),
    user_id         TEXT REFERENCES users(id),

    -- Content
    category        TEXT NOT NULL,              -- see Category enum
    scope           TEXT NOT NULL,              -- 'repo', 'directory', 'file', 'function', 'project'
    scope_ref       TEXT NOT NULL,              -- path or identifier within scope
    title           TEXT NOT NULL,              -- one-line summary
    body            TEXT NOT NULL,              -- markdown detail
    confidence      REAL CHECK (confidence BETWEEN 0.0 AND 1.0),

    -- Lifecycle
    status          TEXT NOT NULL DEFAULT 'active',  -- 'active', 'superseded', 'retracted'
    superseded_by   TEXT REFERENCES memory_entries(id),
    expires_at      TIMESTAMPTZ,

    -- Provenance
    source_type     TEXT NOT NULL,              -- 'agent_analysis', 'investigation', 'review', 'user_annotation'
    source_ref      TEXT,                       -- 'session:abc123', 'pr:456', 'issue:PROJ-789'
    tags            TEXT,                       -- JSON array, indexed via GIN

    -- GORM base
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ                -- soft delete
);

CREATE INDEX idx_memory_entries_project   ON memory_entries(project_id);
CREATE INDEX idx_memory_entries_scope_ref ON memory_entries(scope_ref);
CREATE INDEX idx_memory_entries_category  ON memory_entries(category);
CREATE INDEX idx_memory_entries_status    ON memory_entries(status) WHERE status = 'active';
```

#### `memory_outcomes` — Verdict tracking

```sql
CREATE TABLE memory_outcomes (
    id              TEXT PRIMARY KEY,
    entry_id        TEXT NOT NULL REFERENCES memory_entries(id),

    -- What the agent concluded
    verdict         TEXT NOT NULL,              -- 'bug_confirmed', 'pr_approved', 'fix_suggested', 'safe_to_merge'

    -- What actually happened
    actual_outcome  TEXT,                       -- 'issue_closed_fixed', 'pr_merged', 'pr_reverted', 'regression_found'
    outcome_ref     TEXT,                       -- 'github:org/repo#123', 'jira:PROJ-456'
    correct         BOOLEAN,                   -- was the agent right?

    -- Attribution
    recorded_by     TEXT NOT NULL,              -- 'agent', 'user', 'webhook'
    session_id      TEXT REFERENCES sessions(id),

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_memory_outcomes_entry ON memory_outcomes(entry_id);
```

#### `memory_audit_log` — Immutable change log

```sql
CREATE TABLE memory_audit_log (
    id              TEXT PRIMARY KEY,
    entry_id        TEXT NOT NULL REFERENCES memory_entries(id),

    action          TEXT NOT NULL,              -- 'created', 'updated', 'superseded', 'retracted', 'expired'
    actor_type      TEXT NOT NULL,              -- 'session', 'agent', 'user', 'system'
    actor_id        TEXT NOT NULL,
    reason          TEXT,                       -- why the change was made
    diff            TEXT,                       -- JSON: what fields changed

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
    -- No updated_at, no deleted_at: audit log is append-only
);

CREATE INDEX idx_memory_audit_entry ON memory_audit_log(entry_id);
```

### 5.2 Category Enum

| Category | Description | Example |
|----------|-------------|---------|
| `architecture` | Repo structure, module boundaries, key abstractions | "Backend uses Gin with dynamic K8s client per request" |
| `convention` | Coding patterns, naming rules, test strategies | "All handlers follow validate→fetch→transform→respond pattern" |
| `investigation` | Bug root cause, debugging findings | "Session timeout caused by missing context cancellation in sessions.go:342" |
| `review` | PR review observations, code quality notes | "This migration is not idempotent — will fail on re-run" |
| `caveat` | Known footguns, race conditions, gotchas | "runnerStateDirs map is not thread-safe; accessed from multiple goroutines" |
| `dependency` | External dependency notes, version constraints | "golangci-lint v1.55+ required; v1.54 has false positives on this codebase" |

### 5.3 Go Types (ambient-api-server plugin)

```go
// plugins/memoryEntries/model.go

type MemoryEntry struct {
    api.Meta

    ProjectID   string  `gorm:"not null;index" json:"project_id"`
    SessionID   *string `gorm:"index" json:"session_id,omitempty"`
    AgentID     *string `gorm:"index" json:"agent_id,omitempty"`
    UserID      *string `json:"user_id,omitempty"`

    Category    string  `gorm:"not null" json:"category"`
    Scope       string  `gorm:"not null" json:"scope"`
    ScopeRef    string  `gorm:"not null;index" json:"scope_ref"`
    Title       string  `gorm:"not null" json:"title"`
    Body        string  `gorm:"not null" json:"body"`
    Confidence  *float64 `json:"confidence,omitempty"`

    Status       string  `gorm:"not null;default:active" json:"status"`
    SupersededBy *string `json:"superseded_by,omitempty"`
    ExpiresAt    *time.Time `json:"expires_at,omitempty"`

    SourceType  string  `gorm:"not null" json:"source_type"`
    SourceRef   *string `json:"source_ref,omitempty"`
    Tags        *string `gorm:"type:text" json:"tags,omitempty"` // JSON array
}
```

```go
// plugins/memoryOutcomes/model.go

type MemoryOutcome struct {
    api.Meta

    EntryID        string  `gorm:"not null;index" json:"entry_id"`
    Verdict        string  `gorm:"not null" json:"verdict"`
    ActualOutcome  *string `json:"actual_outcome,omitempty"`
    OutcomeRef     *string `json:"outcome_ref,omitempty"`
    Correct        *bool   `json:"correct,omitempty"`
    RecordedBy     string  `gorm:"not null" json:"recorded_by"`
    SessionID      *string `json:"session_id,omitempty"`
}
```

---

## 6. API Design

### 6.1 REST Endpoints (ambient-api-server)

Following the existing Kind plugin pattern:

```
# Memory Entries — full CRUD + search
GET    /api/ambient/v1/memory_entries                    List/search entries
POST   /api/ambient/v1/memory_entries                    Create entry
GET    /api/ambient/v1/memory_entries/{id}               Get entry
PATCH  /api/ambient/v1/memory_entries/{id}               Update entry
DELETE /api/ambient/v1/memory_entries/{id}               Soft-delete entry

# Custom actions
POST   /api/ambient/v1/memory_entries/{id}/supersede     Mark superseded by new entry
POST   /api/ambient/v1/memory_entries/{id}/retract       Retract (with reason)

# Memory Outcomes
GET    /api/ambient/v1/memory_outcomes                   List outcomes
POST   /api/ambient/v1/memory_outcomes                   Record outcome
GET    /api/ambient/v1/memory_outcomes/{id}              Get outcome
PATCH  /api/ambient/v1/memory_outcomes/{id}              Update outcome (e.g., set correct=true)

# Audit Log — read-only
GET    /api/ambient/v1/memory_audit_log                  List audit entries
GET    /api/ambient/v1/memory_audit_log/{id}             Get audit entry
```

### 6.2 Search Queries

Leveraging the existing TSL (Tree Search Language) support:

```
# All active architecture findings for a project
GET /memory_entries?search=project_id='proj-123' and category='architecture' and status='active'

# All findings touching a specific file
GET /memory_entries?search=scope_ref like '%sessions.go%' and status='active'

# Investigation findings from the last 30 days
GET /memory_entries?search=category='investigation' and created_at > '2026-03-10'

# Entries with low confidence (for review)
GET /memory_entries?search=confidence < 0.5 and status='active'

# Outcome accuracy for a project
GET /memory_outcomes?search=correct=true&size=0  (count only)
```

### 6.3 Proactive Context Endpoint

One custom endpoint optimized for session startup — returns a condensed summary rather than full entries:

```
GET /api/ambient/v1/memory_entries/context?project_id=X&paths=a,b,c&max_tokens=2000

Response:
{
  "project_summary": "3 architecture entries, 2 active investigations, 1 caveat",
  "file_warnings": [
    {
      "path": "components/backend/handlers/sessions.go",
      "entries": [
        {"title": "Root cause of session timeout bug", "category": "investigation", "confidence": 0.9}
      ]
    }
  ],
  "active_caveats": [
    {"title": "runnerStateDirs not thread-safe", "scope_ref": "components/operator/..."}
  ],
  "injected_context": "## Project Memory\n\n### Known Issues\n- ..."
}
```

The `injected_context` field is a pre-formatted markdown string that the runner can inject directly into the agent's system prompt or initial context. The `max_tokens` parameter caps the output to avoid context bloat.

---

## 7. Session Integration

### 7.1 MCP Tools (Runner-Side)

Added to the existing `acp` platform MCP server (Python wrapper calling Go API):

```python
# platform/memory_tools.py

@mcp.tool()
def memory_query(
    category: str | None = None,
    scope_ref: str | None = None,
    keyword: str | None = None,
    status: str = "active",
    limit: int = 20,
) -> str:
    """Search project memory for relevant findings.

    Use this when starting work on a file or area you haven't seen before,
    or when you want to check if previous sessions found anything relevant.
    """
    # Builds TSL query, calls GET /memory_entries
    ...

@mcp.tool()
def memory_store(
    category: str,
    scope: str,
    scope_ref: str,
    title: str,
    body: str,
    confidence: float | None = None,
    source_ref: str | None = None,
    tags: list[str] | None = None,
) -> str:
    """Store a finding in project memory for future sessions.

    Use this when you discover something important about the codebase that
    future sessions should know: architecture decisions, bug root causes,
    code caveats, or investigation findings.

    Categories: architecture, convention, investigation, review, caveat, dependency
    Scopes: project, repo, directory, file, function
    """
    # Calls POST /memory_entries with session attribution
    ...

@mcp.tool()
def memory_warn(file_path: str) -> str:
    """Check if there are any known findings for a specific file.

    Use this before making changes to a file, especially during bug fixes
    or reviews. Returns any active warnings, investigation findings, or
    caveats associated with the file.
    """
    # Calls GET /memory_entries?scope_ref=<file_path>&status=active
    ...

@mcp.tool()
def memory_outcome(
    entry_id: str,
    verdict: str,
    actual_outcome: str | None = None,
    correct: bool | None = None,
) -> str:
    """Record the outcome of an investigation or review finding.

    Use this when you can verify whether a previous finding was correct.
    For example, if a previous session flagged a bug and it was confirmed
    and fixed, record that outcome.
    """
    # Calls POST /memory_outcomes
    ...
```

### 7.2 Proactive Injection at Session Startup

The runner queries the context endpoint during workspace preparation and injects relevant memory as part of the agent's initial context:

```python
# platform/memory.py

async def inject_memory_context(session_name: str, project_id: str, repo_paths: list[str]) -> str | None:
    """Fetch and format relevant memory for session startup injection."""
    try:
        resp = await api_client.get(
            f"/memory_entries/context",
            params={
                "project_id": project_id,
                "paths": ",".join(repo_paths),
                "max_tokens": 2000,
            },
        )
        if resp.status_code == 200:
            data = resp.json()
            if data.get("injected_context"):
                return data["injected_context"]
    except Exception as e:
        logging.debug(f"Memory context injection failed (non-critical): {e}")
    return None
```

This context is appended to the CLAUDE.md or system prompt, not the initial prompt. It is clearly delimited:

```markdown
<!-- BEGIN PROJECT MEMORY (auto-injected, do not edit) -->
## Known Findings

### Active Investigations
- **Session timeout bug** (file: handlers/sessions.go, confidence: 0.9)
  Missing context cancellation at L342. Fixed in PR #1186 but regression possible.

### Caveats
- **runnerStateDirs not thread-safe** (file: operator/internal/handlers/runner_types.go)
  Concurrent access during reconciliation. No mutex. Low-severity.

### Architecture Notes
- Backend uses Gin with per-request K8s dynamic client (never service account for user ops)
- All CRD access via unstructured API, not generated types
<!-- END PROJECT MEMORY -->
```

### 7.3 Integration with Existing MCP Plumbing

The memory tools are registered alongside existing `acp` platform tools. No new MCP server process is needed — memory tools are additional functions in the existing server that already has backend API access via `BOT_TOKEN`.

```python
# In bridges/claude/mcp.py, within build_mcp_servers():

platform_tools = {
    "session": { ... },        # existing
    "rubric": { ... },         # existing
    "corrections": { ... },    # existing
    "acp": { ... },            # existing — backend API tools
    # Memory tools are added to the "acp" server's tool list
}
```

The `acp` MCP server already has authenticated access to the backend API. Since ambient-api-server is accessible within the cluster, the memory API calls route through the same authenticated path.

---

## 8. Memory Lifecycle

### 8.1 Creation

Memories are created by three actors:

| Actor | How | Example |
|-------|-----|---------|
| **Agent (via MCP)** | `memory_store` tool during session | Agent analyzes repo, stores architecture findings |
| **User (via API)** | Direct POST to REST endpoint | Engineer annotates known tech debt |
| **System (via event handler)** | Automated on session completion | Session summary auto-stored on exit |

Every creation produces an audit log entry with `action=created`.

### 8.2 Supersession

When new information replaces old, the old entry is **superseded**, not deleted:

```
Agent finds: "timeout bug is in sessions.go:342" (entry A)
Later agent finds: "actually it's in middleware.go:89" (entry B)
  → entry A.status = 'superseded', entry A.superseded_by = B.id
  → entry B is the new active finding
  → Audit log: action=superseded, reason="Root cause was in middleware, not handler"
```

Superseded entries remain queryable for historical analysis but are excluded from proactive injection.

### 8.3 Retraction

When a finding is determined to be wrong:

```
POST /memory_entries/{id}/retract
Body: {"reason": "Bug was in test setup, not production code"}
  → entry.status = 'retracted'
  → Audit log: action=retracted, reason="..."
```

### 8.4 Expiration

Entries can have an optional `expires_at` timestamp. A background event handler (ambient-api-server controller) periodically marks expired entries:

```go
// In event controller OnSchedule (runs every hour)
func (s *memoryService) ExpireStaleEntries(ctx context.Context) error {
    return s.db.Model(&MemoryEntry{}).
        Where("status = ? AND expires_at < ?", "active", time.Now()).
        Updates(map[string]interface{}{
            "status":     "expired",
            "updated_at": time.Now(),
        }).Error
}
```

### 8.5 Eviction Strategy

To prevent unbounded growth:

| Rule | Threshold | Action |
|------|-----------|--------|
| **Per-project cap** | 500 active entries | Oldest low-confidence entries expired first |
| **Age-based default** | 90 days without update | Entry status set to `stale` (still queryable, excluded from injection) |
| **Supersession chain** | 3+ deep | Oldest entries in chain soft-deleted |

These thresholds are configurable per-project via ProjectSettings. The defaults are conservative — most projects will not hit them.

---

## 9. Scoping Decision: Project-Scoped with Agent Attribution

### Why Project-Scoped (Not Agent-Scoped)

| Consideration | Project-scoped | Agent-scoped |
|---------------|---------------|--------------|
| **Knowledge sharing** | All agents in project share findings | Agent A's discovery is invisible to Agent B |
| **Investigation continuity** | Bug-fix agent's findings help review agent | Each agent rediscovers same issues |
| **Matches ACP model** | Projects are the multi-tenancy boundary | Agents are definitions, not tenancy units |
| **RBAC alignment** | Project RBAC controls memory access | Would need new agent-level RBAC |
| **Practical usage** | Most projects have 1-3 active agents | Agent-scoping would fragment small memory pools |

**Decision**: Memory is **project-scoped** with **agent attribution**. Every entry records which agent (and session) created it, but all entries are visible to all sessions in the project.

An agent can filter by its own entries if needed:

```
GET /memory_entries?search=agent_id='agent-123' and status='active'
```

---

## 10. Implementation Plan

### Phase 1: Data Layer (ambient-api-server)

| # | Task | Files | Effort |
|---|------|-------|--------|
| 1.1 | Generate `memoryEntries` Kind via code generator | `plugins/memoryEntries/` | S |
| 1.2 | Add custom fields to generated model (category, scope, confidence, etc.) | `plugins/memoryEntries/model.go` | S |
| 1.3 | Add `supersede` and `retract` custom actions to handler | `plugins/memoryEntries/handler.go` | M |
| 1.4 | Add `/context` endpoint for proactive injection | `plugins/memoryEntries/handler.go` | M |
| 1.5 | Generate `memoryOutcomes` Kind | `plugins/memoryOutcomes/` | S |
| 1.6 | Generate `memoryAuditLog` Kind (read-only, no create/update/delete handlers) | `plugins/memoryAuditLog/` | S |
| 1.7 | Add audit log trigger: service-layer hook on entry mutations | `plugins/memoryEntries/service.go` | M |
| 1.8 | Add expiration controller (hourly sweep) | `plugins/memoryEntries/controller.go` | S |
| 1.9 | Database migrations for all three tables | `plugins/*/migration.go` | S |
| 1.10 | OpenAPI spec updates | `openapi/openapi.yaml` | M |

### Phase 2: MCP Tools (Runner)

| # | Task | Files | Effort |
|---|------|-------|--------|
| 2.1 | Add memory API client module | `platform/memory.py` | S |
| 2.2 | Implement `memory_query`, `memory_store`, `memory_warn`, `memory_outcome` tools | `platform/memory_tools.py` | M |
| 2.3 | Register memory tools in Claude bridge MCP server | `bridges/claude/mcp.py` | S |
| 2.4 | Register memory tools in Gemini bridge MCP server | `bridges/gemini_cli/mcp.py` | S |
| 2.5 | Add proactive context injection at session startup | `platform/memory.py`, runner main | M |

### Phase 3: Tests

| # | Task | Files | Effort |
|---|------|-------|--------|
| 3.1 | Integration tests for memory CRUD (api-server) | `plugins/memoryEntries/*_test.go` | M |
| 3.2 | Integration tests for outcomes and audit log | `plugins/memoryOutcomes/*_test.go` | M |
| 3.3 | Tests for supersede/retract lifecycle | `plugins/memoryEntries/*_test.go` | M |
| 3.4 | Tests for context endpoint (max_tokens, path filtering) | `plugins/memoryEntries/*_test.go` | S |
| 3.5 | Runner MCP tool unit tests | `tests/test_memory_tools.py` | M |
| 3.6 | Runner context injection tests | `tests/test_memory_injection.py` | S |

### Phase 4: Hardening (Post-Merge)

| # | Task | Files | Effort |
|---|------|-------|--------|
| 4.1 | Eviction controller (per-project cap, age-based staleness) | `plugins/memoryEntries/controller.go` | M |
| 4.2 | Webhook-driven outcome resolution (GitHub, Jira) | New controller + webhook handler | L |
| 4.3 | Memory dashboard in frontend (read-only view) | `components/frontend/...` | L |
| 4.4 | Confidence calibration analytics (aggregate accuracy metrics) | API endpoint + frontend | L |

---

## 11. Security Considerations

| Concern | Mitigation |
|---------|------------|
| **Memory access control** | Memory inherits project RBAC. Only users/agents with project access can read or write. API server enforces via existing `AuthorizeApi` middleware. |
| **Memory poisoning** | All entries have attribution (session, agent, user). Retraction mechanism allows reversal. Confidence scores flag uncertainty. Audit log provides forensics. |
| **Sensitive data in memory** | Memory stores findings *about* code, not secrets. Entries are text, not file contents. API server validates no known secret patterns (regex check on body field). |
| **Cross-project leakage** | `project_id` FK + RBAC enforcement. TSL queries are scoped to authorized projects. No cross-project joins. |
| **Audit log integrity** | Audit log table has no UPDATE or DELETE API. Append-only by design. Soft-delete on the model is disabled for this Kind. |
| **Token scope** | Runner's `BOT_TOKEN` is scoped to the session's namespace. Memory API calls use the same token, limiting blast radius. |

---

## 12. Risks and Mitigations

### Risk 1: Memory Staleness

**Problem**: Codebase evolves; memory entries reference files, functions, or patterns that no longer exist.

**Mitigation**:
- `expires_at` with configurable defaults (90 days)
- Age-based staleness: entries untouched for 90 days flagged as `stale`
- Proactive injection includes `created_at` so agents can judge recency
- Future: validate `scope_ref` paths against current repo state at injection time

**Residual risk**: Medium. Stale memory is misleading but not dangerous — agents should verify findings before acting on them. The injected context includes a disclaimer: "Memory entries may be outdated. Verify before relying on them."

### Risk 2: Memory Pollution

**Problem**: Agent writes incorrect finding → future agents inherit and reinforce the error (circular reasoning).

**Mitigation**:
- Confidence scores: low-confidence entries are deprioritized in injection
- Outcome tracking: entries with `correct=false` outcomes are auto-retracted
- Supersession: newer findings replace older ones explicitly
- Retraction: manual or automated reversal with audit trail
- Proactive injection caps (`max_tokens=2000`) limit exposure to any single bad entry

**Residual risk**: Medium-high. This is the most dangerous failure mode. The outcome tracking loop is the primary defense, but it depends on outcomes being recorded. Phase 2 webhook automation will help.

### Risk 3: Context Bloat

**Problem**: Too many memory entries injected at startup → wastes agent context window, dilutes actual instructions.

**Mitigation**:
- Hard cap via `max_tokens` parameter on context endpoint (default: 2000 tokens)
- Relevance filtering: only entries matching the session's repo paths are injected
- Priority ordering: investigations > caveats > architecture > conventions
- Per-project entry cap (500 active)

**Residual risk**: Low. The cap makes this manageable. If 2000 tokens is too much, operators can tune it down per project.

### Risk 4: Performance at Session Startup

**Problem**: Memory context query adds latency to session initialization.

**Mitigation**:
- PostgreSQL with proper indexes: scope_ref, project_id, status
- Context endpoint is a single query with LIMIT
- Failure is non-critical: if memory query fails, session starts without it (graceful degradation)
- Target: < 100ms for context query (PostgreSQL with indexes, no joins needed)

**Residual risk**: Low. GPS achieves sub-ms on SQLite; PostgreSQL with indexes will be comparable for the expected dataset size (< 500 entries per project).

### Risk 5: Agent Adoption

**Problem**: Agents don't use the MCP tools unless explicitly instructed.

**Mitigation**:
- Tool descriptions are written to guide when to use them (not just what they do)
- Proactive injection ensures agents *receive* memory even if they don't query it
- CLAUDE.md can include guidance: "Use memory_warn before modifying files flagged in project memory"
- Workflow system prompts can incorporate memory-aware instructions

**Residual risk**: Low. Proactive injection is the safety net. MCP tools provide additional depth for agents that use them.

---

## 13. Resolved Decisions

| # | Question | Decision | Rationale |
|---|----------|----------|-----------|
| D1 | Storage backend | PostgreSQL (ambient-api-server) | Memory is application data, not runtime state. Postgres is the platform's source of truth. K8s CRDs have 1MB size limits and poor query support. |
| D2 | Memory scope | Project-scoped with agent attribution | Projects are the multi-tenancy boundary. Agents within a project should share findings. Attribution preserves provenance without fragmenting the knowledge base. |
| D3 | Query mechanism | MCP tools + proactive injection | MCP leverages existing, battle-tested plumbing (3-tier config, auth, multi-bridge). Proactive injection ensures agents benefit even without explicit tool use. |
| D4 | New CRD? | No | Memory has no operator reconciliation needs. No pods, no jobs, no runtime resources. A CRD would be unnecessary indirection. |
| D5 | New microservice? | No | ambient-api-server's plugin system is designed for exactly this. Code generation produces the full CRUD stack. A separate service would add operational overhead with no benefit. |
| D6 | Audit log mutability | Append-only (no update, no delete) | Audit integrity is non-negotiable. If you can delete audit entries, the audit trail is meaningless. |
| D7 | Expiration default | 90 days | Balances freshness vs. retention. Investigations from 6 months ago are likely stale. 90 days covers most development cycles. Configurable per project. |
| D8 | Proactive vs. reactive only | Both | Proactive injection ensures baseline awareness. Reactive (MCP tools) provides depth when needed. Either alone is insufficient. |

---

## 14. Open Questions

| # | Question | Options | Leaning | Needs Input From |
|---|----------|---------|---------|-----------------|
| Q1 | Should session completion auto-generate a memory summary? | Auto-generate vs. agent-driven only | Agent-driven — auto-summaries risk noise | Product |
| Q2 | Should memory entries be versioned (full history) or supersession-only? | Full version history vs. supersede chain | Supersession — simpler, audit log covers history | Architecture |
| Q3 | How should webhook-driven outcomes work? | Generic webhook receiver vs. GitHub/Jira-specific | GitHub-specific first (most users) | Platform team |
| Q4 | Should the frontend have a memory management UI in V1? | Yes (read-only) vs. deferred | Deferred — API + MCP tools are sufficient for V1 | Product |
| Q5 | Should memory be included in session export/snapshot? | Include vs. separate | Separate — memory outlives sessions | Architecture |

---

## 15. Success Criteria

- [ ] Memory entries persist across sessions — agent in session N reads findings from session N-1
- [ ] File-scoped warnings work — `memory_warn("sessions.go")` returns relevant findings
- [ ] Outcome tracking records verdicts and actual results with `correct` boolean
- [ ] Audit log captures every create, update, supersede, and retract with actor attribution
- [ ] Proactive injection adds < 200ms to session startup time
- [ ] Context injection respects `max_tokens` cap and does not exceed 2000 tokens by default
- [ ] Memory CRUD respects project RBAC — users without project access cannot read or write
- [ ] All four MCP tools work in both Claude and Gemini bridges
- [ ] No existing tests break; no changes to operator, backend, or CRDs
- [ ] Per-project cap (500 entries) prevents unbounded growth

---

## Appendix A: Comparison with Related Systems

| System | Scope | Storage | Query | Feedback Loop |
|--------|-------|---------|-------|---------------|
| **Claude Code Memory** | Per-user, per-project | Flat files (~/.claude/) | File read at startup | None |
| **GPS MCP Server** | Org-wide | SQLite (materialized) | MCP tools, sub-ms | None |
| **Project Intelligence Memory** (this proposal) | Per-project | PostgreSQL | MCP tools + proactive injection | Outcome tracking with correctness signal |

Key differentiator: outcome tracking creates a **calibration loop** — over time, the system knows which types of findings are reliable and which are not. No existing system in the Ambient ecosystem has this.

## Appendix B: Example Agent Interaction

```
Session starts on project "ambient-platform"
  Runner injects memory context:
    "2 active investigations, 1 caveat for files in your workspace"

Agent begins working on components/backend/handlers/sessions.go

Agent: [calls memory_warn("components/backend/handlers/sessions.go")]
Memory: "Investigation finding (2026-03-15, confidence: 0.9):
  This file was the root cause of the session timeout bug.
  Missing context cancellation at L342. Fixed in PR #1186.
  Outcome: issue closed as fixed."

Agent proceeds with awareness of the history.

Agent discovers a new pattern:
Agent: [calls memory_store(
  category="convention",
  scope="file",
  scope_ref="components/backend/handlers/sessions.go",
  title="All session handlers must validate phase before mutation",
  body="UpdateSession and PatchSession check phase != Running before allowing changes.
        Any new mutation handler must follow this pattern.",
  confidence=0.95,
  source_ref="session:current-session-id"
)]
Memory: "Stored. Entry ID: mem_abc123"

Session ends. Finding persists.
Next session on this project starts with awareness of this convention.
```
