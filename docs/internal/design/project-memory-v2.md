# Project Memory v2 — From Static Summaries to Persistent Cross-Session Knowledge

**Status**: Draft
**Author**: Yossi Ovadia
**Date**: 2026-04-13
**Builds on**: [project-intelligence-memory.md](project-intelligence-memory.md)

---

## Premise

The current intelligence system stores **what the code IS** (language, framework, architecture). Claude Code discovers all of this on its own by reading files. Static summaries add little value.

What DOES add value is **temporal knowledge** — things that cannot be derived from the current state of the codebase:

| Temporal knowledge | Example |
|---|---|
| Why a decision was made | "Used polling over WebSocket because the operator can't guarantee sticky routing" |
| What failed before | "GORM preloading caused N+1 on the List endpoint — switched to subresource handler" |
| Team conventions not in code | "All plugin services follow lockFactory + DAO + events pattern" |
| Bug root causes | "Timeout caused by missing ctx cancellation at sessions.go:342" |
| Recurring patterns | "Three separate PRs introduced race conditions on the same map" |

---

## 1. Knowledge Accumulation

### 1.1 New Finding Categories

Extend the `category` field on `RepoFinding` (TEXT column, no DB enum). Update MCP tool schemas and prompt instructions only.

**Existing**: `investigation`, `caveat`, `review`, `convention`

**New**:

| Category | Stores | Trigger |
|---|---|---|
| `decision` | Why an approach was chosen over alternatives | After making a trade-off |
| `failure` | What was tried and failed, and why | After abandoning an approach |
| `pattern` | Recurring cross-file or cross-session observations | After noticing repetition |

### 1.2 Accumulation is Agent-Initiated

No new automated triggers. The agent calls `memory_store` during normal work. The system prompt tells it when:

**Store after**: fixing a bug (category=`investigation`), making a trade-off (category=`decision`), abandoning an approach (category=`failure`), observing repetition (category=`pattern`), completing a code review (category=`review`).

**Do NOT store**: what the code does, language/framework info, things already in CLAUDE.md, obvious patterns visible in the code.

### 1.3 Supersession (Deduplication)

When an agent stores a finding on a file+category that already has an active finding, the old one should be superseded — not duplicated.

Add `supersedes_id` parameter to `memory_store`. When provided:
1. Create the new finding
2. PATCH old finding: `status=superseded`, `superseded_by=<new_id>`
3. Log a RepoEvent with `action=superseded`

This prevents unbounded growth on hot files. The supersession chain provides an audit trail of how understanding evolved.

---

## 2. Intelligent Retrieval

### 2.1 File-Path-Based Relevance

The current `/context` endpoint dumps all active findings. Change it to accept `touched_paths` — the files the current task will touch.

```
GET /api/ambient/v1/repo_intelligences/context
  ?project_id=X
  &repo_urls=a,b,c
  &touched_paths=components/backend/handlers/sessions.go,components/operator/internal/handlers/sessions.go
  &max_findings=15
```

When `touched_paths` is provided, return findings in priority order:
1. **Exact file match** — findings on the same file
2. **Directory match** — findings sharing a directory prefix
3. **Repo-wide** — findings with no specific file path

When absent (initial session startup), fall back to recency-ordered active findings.

### 2.2 Priority Ordering

```sql
ORDER BY
  CASE category
    WHEN 'failure' THEN 0
    WHEN 'investigation' THEN 0
    WHEN 'caveat' THEN 1
    WHEN 'decision' THEN 1
    WHEN 'pattern' THEN 2
    WHEN 'review' THEN 2
    WHEN 'convention' THEN 3
  END,
  updated_at DESC
```

Failures and investigations surface first — they're the most dangerous to repeat.

### 2.3 Context Budget

Hard cap injected context at **4000 characters** (~1000 tokens). Render each finding as a single line:

```markdown
<!-- BEGIN PROJECT MEMORY -->
## Active Findings (7 total, showing top 5)

- **[failure]** `handlers/sessions.go`: GORM preloading causes N+1 — use subresource handler (Apr 10)
- **[investigation]** `operator/sessions.go`: Timeout root cause was missing ctx cancel at L342 (Apr 8)
- **[decision]** `bridges/claude/bridge.py`: No --resume after dirty reinit — CLI doesn't flush JSONL (Apr 11)
- **[caveat]** `operator/runner_types.go`: runnerStateDirs map is not thread-safe (Apr 9)
- **[pattern]** `plugins/repoFindings/`: All plugin services follow lockFactory+DAO+events (Apr 9)

(+2 more — use memory_query for details)
<!-- END PROJECT MEMORY -->
```

Changes from current format:
- No repo-level summary (language, framework, architecture) — Claude reads code itself
- Date included for recency judgment
- Category as tag, body omitted (title only)
- Overflow count with tool hint

### 2.4 Reactive Retrieval

`memory_warn` already works with partial path matching. No code change needed — prompt instructions tell the agent to call it before modifying files with known findings.

---

## 3. Schema Changes

All additive columns on `repo_findings`. Single migration.

### 3.1 New Columns

```go
// Migration ID: "202604131300"
SupersededBy *string    `json:"superseded_by,omitempty" gorm:"index"`
Tags         *string    `json:"tags,omitempty"          gorm:"type:text"` // JSON array of strings
ExpiresAt    *time.Time `json:"expires_at,omitempty"`
```

- `superseded_by`: FK to the finding that replaced this one
- `tags`: free-form labels for future filtering (e.g., `["performance", "race-condition"]`)
- `expires_at`: optional TTL for findings that become irrelevant (e.g., "workaround until v2 ships")

### 3.2 New Status Value

Add `superseded` to the status lifecycle: `active → superseded | resolved | retracted`

### 3.3 No Changes to RepoIntelligence

The RepoIntelligence table stays as-is. Auto-analysis still populates it (useful for frontend display). The context endpoint just stops including it in the injected prompt.

---

## 4. Prompt and Tool Changes

### 4.1 System Prompt

Replace the current intelligence injection block with:

```
## Project Memory

You have access to cross-session memory tools. Previous sessions stored
findings about this codebase — things you cannot discover by reading code.

### Reading Memory
- Active findings from previous sessions are listed below
- Before modifying a file with known findings, call memory_warn(file_path)
- Use memory_query(repo_url, file_path) for deeper investigation

### Writing Memory
Store findings that future sessions need but CANNOT discover from code:
- **decision**: Why an approach was chosen (rationale, rejected alternatives)
- **failure**: What was tried and failed (approach, why it failed)
- **investigation**: Bug root cause and fix
- **caveat**: Runtime gotchas not visible in code
- **pattern**: Recurring cross-file observations
- **convention**: Team patterns not documented elsewhere

Do NOT store: what the code does, language/framework info, things in CLAUDE.md.
Call memory_store after completing significant work.
```

### 4.2 MCP Tool Updates

**memory_store** — extend the category enum:
```python
"enum": ["investigation", "caveat", "review", "convention", "decision", "failure", "pattern"]
```

Add optional `supersedes_id` parameter:
```python
"supersedes_id": {
    "type": "string",
    "description": "ID of a previous finding this supersedes (marks old as superseded)",
}
```

Implementation: after creating new finding, if `supersedes_id` provided, PATCH old finding with `status=superseded`, `superseded_by=<new_id>`.

**memory_query** — update category enum to match.

### 4.3 Context Endpoint (`buildInjectedContext`)

Rewrite `buildInjectedContext()` in `handler.go` to:
1. Accept `touched_paths` from query params
2. Query findings with file-path priority ordering (Section 2.1)
3. Render compact format (Section 2.3)
4. Omit RepoIntelligence summary from output
5. Cap at 4000 characters

---

## 5. Validation

### 5.1 Metrics

| Metric | How |
|---|---|
| Findings stored per session | Count `memory_store` calls (exclude auto-analysis) |
| Findings read per session | Count `memory_query` + `memory_warn` calls |
| Cross-session reads | % of read findings from a different session_id |
| Category distribution | Group by category |
| Supersession rate | % of findings superseded within 30 days |
| Finding age at read | Average `now() - created_at` when queried |

### 5.2 A/B Testing

Toggle: `AMBIENT_DISABLE_INTELLIGENCE` (already exists, per-session via CR spec).

- **Control**: intelligence disabled — no injection, no MCP tools
- **Treatment**: v2 enabled — findings-only injection, memory instructions

Assign at project level via ProjectSettings. 4 weeks minimum to allow cross-session accumulation.

### 5.3 Success Criteria

| Criterion | Target |
|---|---|
| Agents voluntarily store findings | >0.5/session (excluding auto-analysis) |
| Agents read stored findings | >30% of sessions call memory_query or memory_warn |
| Cross-session memory used | >10% of reads are from a different session |
| No context bloat | Injected memory < 1200 tokens |
| Users don't disable | <10% of projects disable after enabling |

### 5.4 Qualitative Check

Week 2: manually review 20 findings from 5 projects. If >50% store static code descriptions instead of temporal knowledge, revise prompt instructions.

---

## 6. Implementation Sequence

| Phase | What | Files | Size |
|---|---|---|---|
| 1. Schema | Add `superseded_by`, `tags`, `expires_at` to RepoFinding | model.go, presenter.go, new migration | S |
| 2. Context | `touched_paths` param, priority ordering, findings-only format, char cap | handler.go (Context + buildInjectedContext) | M |
| 3. Tools | New categories, `supersedes_id` on memory_store, supersession logic | memory_tools.py, intelligence_api.py | M |
| 4. Prompt | Memory instructions, findings-only injection | prompts.py | S |
| 5. Validate | Logging for read/write counts | memory_tools.py, runner logs | S |

Phases 1-4 ship as one PR. Phase 5 follows after 1 week of observation.

---

## 7. Explicitly NOT Doing

- **No vector DB** — retrieval is file-path prefix matching + recency, not semantic search
- **No new tables** — only columns on existing `repo_findings`
- **No new MCP tools** — extend existing `memory_store`, `memory_query`, `memory_warn`
- **No new services** — all changes in existing api-server plugins and runner
- **No auto-generated session summaries** — agents store findings explicitly; automatic end-of-session summarization is noise-prone
- **No outcome tracking** — supersession provides the correction loop (new findings replace wrong ones)
