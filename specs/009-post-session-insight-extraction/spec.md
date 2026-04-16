# Feature Specification: Post-Session Insight Extraction

**Feature Branch**: `009-post-session-insight-extraction`
**Created**: 2026-04-15
**Revised**: 2026-04-16
**Status**: Draft

> **Architecture change**: This spec originally wrote extracted insights to a CRD-based Project
> Memory Store. It now writes them as markdown files in `docs/learned/` and submits them as draft
> PRs with the `continuous-learning` label, following the file-based pipeline from spec 002.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Extraction Runs After Session Completion (Priority: P1)

A developer finishes a multi-turn agentic session that discovered a non-obvious build workaround. After the session transitions to Completed, the backend automatically runs a lightweight LLM extraction pass against the session transcript. Within seconds, the extracted insights are written as markdown files to a new branch in the workspace repo and submitted as a draft PR with the `continuous-learning` label, ready for human review.

**Why this priority**: This is the core value loop. Without automatic extraction, reusable knowledge is lost when the session ends.

**Independent Test**: Create a session with at least 5 turns that includes a discoverable pattern. Let it complete. Verify that a draft PR appears in the workspace repo with learned files in `docs/learned/` and the `continuous-learning` label.

**Acceptance Scenarios**:

1. **Given** a session transitions to Completed in a CL-enabled workspace, **When** the backend detects the status change, **Then** it enqueues a background extraction task within 5 seconds.
2. **Given** the extraction task runs, **When** it fetches the session transcript and sends it to the extraction model, **Then** the LLM returns structured JSON with candidate entries.
3. **Given** the LLM returns 4 candidate insights, **When** the extraction task processes them, **Then** each candidate is written as a markdown file in `docs/learned/` on a new branch, and a single draft PR is opened with the `continuous-learning` label and `source: insight-extraction` noted in the PR description.
4. **Given** the project is configured with `maxMemoriesPerSession=3` and the LLM returns 4 candidates, **Then** only the top 3 ranked by confidence are included in the PR.

---

### User Story 2 - Extracted Insights Appear for Human Review (Priority: P1)

A team lead opens the Project Memory panel (spec 008) and sees draft PRs in the "Pending Review" section. Among them are PRs created by the insight extraction pipeline, identifiable by their PR description mentioning `source: insight-extraction` and the originating session. The lead reviews the PR diff on GitHub, approves useful entries, edits others for clarity, and closes PRs with entries that are too session-specific.

**Why this priority**: Unreviewed machine-generated content degrades trust. Human review via PR curation is the quality gate.

**Independent Test**: After extraction completes, open the Project Memory panel. Confirm the draft PR appears in "Pending Review". Click through to GitHub, review the diff, merge or close.

**Acceptance Scenarios**:

1. **Given** extraction has produced a draft PR, **When** the user opens the Project Memory panel, **Then** the PR appears in "Pending Review" with a label indicating it came from insight extraction.
2. **Given** a draft PR from extraction, **When** the user merges it on GitHub, **Then** the learned files become available to future sessions via `docs/learned/` on the default branch.
3. **Given** a draft PR from extraction, **When** the user closes it without merging, **Then** the insights are discarded and do not affect future sessions.
4. **Given** a draft PR with 3 files, **When** the user edits one file in the PR before merging, **Then** the edited version is what lands on the default branch.

---

### User Story 3 - Admin Configures Extraction Settings (Priority: P2)

A project admin configures insight extraction via `.ambient/config.json` in the workspace repo:

```json
{
  "learning": {
    "enabled": true,
    "extraction": {
      "enabled": true,
      "model": "claude-haiku-4",
      "maxMemoriesPerSession": 5,
      "minTurnThreshold": 5
    }
  }
}
```

The configuration is read from the workspace repo at extraction time.

**Why this priority**: Teams need control over cost, model selection, and volume.

**Independent Test**: Update `.ambient/config.json` with extraction settings, then complete a session. Verify extraction uses the configured model and respects limits.

**Acceptance Scenarios**:

1. **Given** `.ambient/config.json` with `extraction.enabled: true`, **When** a session completes, **Then** extraction runs using the configured model.
2. **Given** `.ambient/config.json` with `extraction.enabled: false`, **When** a session completes, **Then** no extraction runs.
3. **Given** no `extraction` key in config, **When** a session completes, **Then** extraction is disabled by default (opt-in).
4. **Given** extraction config specifies `model: "claude-haiku-4"`, **When** extraction runs, **Then** it uses that model, not the project's default session model.

---

### User Story 4 - Short Sessions Are Skipped (Priority: P2)

A developer creates a quick 2-turn session. The session completes, but no extraction runs because the session is too short to contain reusable knowledge.

**Why this priority**: Short sessions rarely contain extractable insights. Skipping reduces cost and noise.

**Acceptance Scenarios**:

1. **Given** a session completes with 3 turns and `minTurnThreshold` is 5, **When** the backend evaluates extraction, **Then** it skips and logs "session below minimum turn threshold".
2. **Given** a session completes with 5 turns, **When** extraction is enabled, **Then** the extraction task runs normally.
3. **Given** a session completes with 0 events, **When** the backend evaluates, **Then** it skips and logs "empty transcript".

---

### Edge Cases

- **LLM extraction returns malformed JSON**: Extraction task logs error, records `extractionStatus=failed` on the session, no PR created. No retry.
- **LLM extraction returns zero candidates**: Extraction completes successfully, no PR created. Valid outcome.
- **Duplicate insights across sessions**: Each extraction creates its own PR. Deduplication is deferred to human review during PR curation.
- **Empty transcript**: Extraction skipped. Backend logs warning.
- **Extraction model quota exceeded or API error**: Extraction logs error, records `extractionStatus=failed`. Session completes normally.
- **Concurrent extraction for same session**: Backend enforces at-most-once via `extractionStatus` field on session. Second trigger is a no-op.
- **Session transitions to Failed**: Extraction still runs for Failed sessions (may contain debugging insights), subject to min turn threshold.
- **Git push fails**: Extraction records `extractionStatus=partial-failure`. Insights are logged but no PR is created.
- **Workspace not CL-enabled**: Extraction is skipped entirely.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST trigger a background extraction task when a session transitions to Completed or Failed, provided extraction is enabled in `.ambient/config.json`.
- **FR-002**: System MUST skip extraction for sessions with fewer than `minTurnThreshold` turns (default: 5, configurable).
- **FR-003**: System MUST fetch the session transcript from the AG-UI event store, apply compaction, and send it to the configured extraction model.
- **FR-004**: System MUST use a structured extraction prompt that asks for reusable knowledge: corrections (mistakes to avoid) and patterns (conventions to follow). The prompt MUST instruct the LLM to ignore session-specific details.
- **FR-005**: System MUST parse the LLM response as structured JSON: an array of candidates, each with `title`, `content`, `type` (correction|pattern), and `confidence` (0.0-1.0).
- **FR-006**: System MUST write candidate entries as markdown files in `docs/learned/<type>s/<date>-<slug>.md` on a new branch and open a single draft PR with the `continuous-learning` label. The PR description MUST include `source: insight-extraction` and the originating session name.
- **FR-007**: System MUST enforce `maxMemoriesPerSession` by ranking candidates by confidence and truncating.
- **FR-008**: System MUST enforce at-most-once extraction per session via `extractionStatus` field on the session CR status.
- **FR-009**: Extraction configuration MUST be read from `.ambient/config.json` in the workspace repo (`learning.extraction.*` keys).
- **FR-010**: System MUST run extraction as a background goroutine in the backend, not as a separate Job.
- **FR-011**: System MUST NOT block session completion on extraction.
- **FR-012**: System MUST record extraction status on the session: `extractionStatus` with values `pending`, `running`, `completed`, `skipped`, `failed`, `partial-failure`.
- **FR-013**: Feature MUST be gated behind the `learning-agent-loop` feature flag.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Extraction runs automatically for 100% of eligible sessions within 10 seconds of completion.
- **SC-002**: Extracted insights appear as a draft PR in the workspace repo with the `continuous-learning` label.
- **SC-003**: Extraction adds less than 2 seconds of overhead to session completion (non-blocking, measures only enqueue).
- **SC-004**: Extraction cost per session is under $0.01 when using a Haiku-class model.
- **SC-005**: Zero data loss on extraction failure: session completion and existing learned files are unaffected.

## Assumptions

- Spec 002 establishes the `docs/learned/` file format, directory structure, and wiki compiler pipeline for delivering knowledge to agents.
- Spec 008 provides the "Pending Review" UI section that surfaces draft PRs.
- The AG-UI event store retains session events long enough for post-completion extraction.
- The extraction model is available via the same LLM provider credentials configured for the project.
- The backend has git push access to the workspace repo for creating branches and PRs.
- `.ambient/config.json` is the single source of truth for CL configuration (no backend-side config store needed).

## Dependencies

- **Spec 002**: File format and directory structure for `docs/learned/`.
- **Spec 008**: "Pending Review" UI for surfacing draft PRs.
- AG-UI event store (`components/backend/websocket/agui_store.go`) — transcript retrieval.
- Session lifecycle in operator (`components/operator/internal/handlers/sessions.go`) — status transition detection.
- LLM provider integration — API access for extraction model.
- GitHub API — branch creation, file commit, PR creation.
- `.ambient/config.json` — extraction configuration.
