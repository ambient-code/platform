# Feature Specification: Event-Driven Feedback Loop

**Feature Branch**: `007-event-driven-feedback-loop`
**Created**: 2026-04-15
**Status**: Draft

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Threshold Crossing Triggers Improvement Session (Priority: P1)

A developer corrects the agent twice on the same workflow within an hour. The backend detects that the correction count for that target has crossed the configured threshold (default: 2 within 24 hours), constructs an improvement prompt using the same logic as `query_corrections.py`, and creates an improvement session targeting that workflow/repo -- all within minutes rather than waiting for the next weekly batch run.

**Why this priority**: This is the core value proposition. Without real-time triggering, users wait up to 7 days for the platform to learn from corrections, which degrades trust in the feedback loop.

**Independent Test**: Log 2 `session-correction` scores against the same target via the corrections pipeline, then verify an improvement session is created within 5 minutes and contains both corrections in its prompt.

**Acceptance Scenarios**:

1. **Given** a project with event-driven feedback enabled (default), **When** 2 corrections targeting the same workflow are logged within 24 hours, **Then** the backend creates an improvement session within 5 minutes with a prompt containing both corrections.
2. **Given** a project with event-driven feedback enabled, **When** 1 correction is logged for a target, **Then** no improvement session is created (threshold not met).
3. **Given** a project with the threshold set to 5, **When** 4 corrections are logged for the same target, **Then** no improvement session is created until the 5th correction arrives.
4. **Given** a correction is logged, **When** the backend creates an improvement session, **Then** the session has labels `feedback-loop=true`, `source=event-driven`, and `target-type={workflow|repo}`, and the session prompt matches the structure produced by `build_improvement_prompt()`.

---

### User Story 2 - Admin Configures Feedback Loop Threshold (Priority: P1)

A project admin opens the project settings and adjusts the feedback loop configuration: raising the minimum corrections to 3, narrowing the time window to 12 hours, or disabling auto-trigger entirely so only the weekly batch runs. The backend persists these settings per project and applies them immediately to future threshold evaluations.

**Why this priority**: Different projects have different correction volumes. A high-traffic project needs a higher threshold to avoid noisy improvement sessions; a low-traffic project may want a lower threshold or opt out entirely.

**Independent Test**: `PUT /api/projects/:projectName/feedback-loop/config` with `{"minCorrections": 3, "timeWindowHours": 12, "autoTriggerEnabled": true}`, then `GET` the same endpoint and confirm the values are persisted.

**Acceptance Scenarios**:

1. **Given** a project with no custom config, **When** `GET /api/projects/:projectName/feedback-loop/config` is called, **Then** the response returns defaults: `{"minCorrections": 2, "timeWindowHours": 24, "autoTriggerEnabled": true}`.
2. **Given** an admin, **When** they `PUT` a config with `minCorrections: 5`, **Then** subsequent threshold evaluations for that project require 5 corrections before triggering.
3. **Given** an admin, **When** they `PUT` a config with `autoTriggerEnabled: false`, **Then** no event-driven improvement sessions are created for that project regardless of correction volume. The weekly GHA batch still runs.
4. **Given** a non-admin user, **When** they attempt to `PUT` the config, **Then** the request is rejected with 403.

---

### User Story 3 - Deduplication Prevents Duplicate Sessions (Priority: P1)

After an improvement session is triggered for a target, additional corrections continue to arrive for the same target within the time window. The backend recognizes that an improvement session was already created for this target within the window and does not create a duplicate. Corrections that arrive after the window expires can trigger a new session.

**Why this priority**: Without deduplication, a burst of corrections (e.g., a team of 3 developers all hitting the same workflow bug) would create 3 separate improvement sessions for the same problem, wasting compute and creating conflicting PRs.

**Independent Test**: Trigger a threshold crossing to create an improvement session, then log 2 more corrections for the same target within the time window. Verify no second session is created. Wait for the window to expire (or shorten it via config), log 2 more corrections, and verify a new session is created.

**Acceptance Scenarios**:

1. **Given** an improvement session was created for target X at time T, **When** 2 more corrections for target X arrive at T+1h (within the 24h window), **Then** no new improvement session is created.
2. **Given** an improvement session was created for target X at time T with a 24h window, **When** 2 corrections for target X arrive at T+25h, **Then** a new improvement session is created.
3. **Given** an improvement session was created for target X, **When** 2 corrections arrive for target Y (different target), **Then** an improvement session is created for target Y (dedup is per-target, not global).
4. **Given** the weekly GHA creates an improvement session for target X, **When** 2 corrections arrive for target X within 24 hours of that GHA session, **Then** the event-driven path recognizes the GHA-created session and does not create a duplicate.

---

### User Story 4 - Weekly GHA and Real-Time Coexist Without Conflict (Priority: P2)

The weekly `feedback-loop.yml` GHA continues running on its Monday 9am UTC schedule. It acts as a sweep for corrections that did not individually meet the real-time threshold -- for example, a target with 1 correction per day over 7 days (7 total, but never 2 within 24 hours). The GHA and the event-driven path do not create duplicate sessions for the same corrections.

**Why this priority**: The batch sweep catches long-tail patterns that the real-time threshold misses. Both paths must coexist cleanly without doubling up on improvement sessions.

**Independent Test**: Configure a project with `minCorrections: 3` for real-time. Log 1 correction per day for 5 days against the same target (never crossing real-time threshold). Trigger the weekly GHA with `--since-days 7 --min-corrections 2`. Verify the GHA creates exactly one improvement session. Verify the event-driven path did not create any sessions during the week.

**Acceptance Scenarios**:

1. **Given** the weekly GHA runs, **When** it finds corrections that already triggered real-time improvement sessions, **Then** it skips those corrections (checks for existing sessions with `feedback-loop=true` label targeting the same target within the lookback window).
2. **Given** the weekly GHA runs, **When** it finds corrections that did not meet the real-time threshold, **Then** it creates improvement sessions for groups meeting its own `--min-corrections` threshold as before.
3. **Given** both paths are active, **When** the `GET /api/projects/:projectName/feedback-loop/history` endpoint is called, **Then** all triggered improvement sessions are listed with their source (`event-driven` or `github-action`) and the correction IDs that triggered them.
4. **Given** a project has `autoTriggerEnabled: false`, **When** the weekly GHA runs, **Then** the GHA operates normally (it does not consult the project config -- it uses its own CLI args).

---

### Edge Cases

- **Corrections arrive simultaneously**: If multiple corrections cross the threshold in the same evaluation cycle, only one improvement session is created per target.
- **Backend restarts mid-window**: The corrections buffer must be durable (persisted in CRD or ConfigMap), not in-memory only. A backend restart must not lose pending correction counts or deduplication state.
- **Langfuse unavailable**: If the corrections pipeline cannot deliver scores, the event-driven path gracefully degrades -- no sessions are created, no errors are surfaced to users. The weekly GHA retries independently.
- **Target no longer exists**: If a workflow or repo referenced in corrections has been deleted, the improvement session is still created (it will fail at clone time, which is acceptable -- the session logs capture the context).
- **Config updated mid-window**: If the threshold is raised from 2 to 5 while 3 corrections are already buffered, the new threshold applies immediately. The 3 buffered corrections do not trigger a session.
- **Zero time window**: A `timeWindowHours` of 0 is rejected with 400 -- minimum is 1 hour.
- **Very large correction bursts**: If 100 corrections arrive for the same target, only one improvement session is created. The prompt includes all corrections up to a reasonable limit (50), with a summary count for the remainder.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST watch the corrections buffer for threshold crossings per target (unique combination of `target_type`, `target_repo_url`, `target_branch` for workflows, `target_path`).
- **FR-002**: System MUST support per-project configuration of: minimum corrections count (`minCorrections`, default 2), time window in hours (`timeWindowHours`, default 24), and auto-trigger toggle (`autoTriggerEnabled`, default true).
- **FR-003**: System MUST create an improvement session when the correction count for a target crosses the configured threshold within the configured time window.
- **FR-004**: System MUST construct improvement session prompts using the same logic as `build_improvement_prompt()` in `scripts/feedback-loop/query_corrections.py` (ported to Go or invoked as a subprocess).
- **FR-005**: System MUST expose `GET /api/projects/:projectName/feedback-loop/config` returning current config (defaults if none set).
- **FR-006**: System MUST expose `PUT /api/projects/:projectName/feedback-loop/config` accepting `minCorrections` (int, >= 1), `timeWindowHours` (int, >= 1), and `autoTriggerEnabled` (bool). Requires project admin permissions.
- **FR-007**: System MUST expose `GET /api/projects/:projectName/feedback-loop/history` returning a list of triggered improvement sessions with: session name, creation time, source (`event-driven` or `github-action`), target info, and correction trace IDs that contributed.
- **FR-008**: System MUST NOT create a duplicate improvement session for the same target within the active time window (deduplication).
- **FR-009**: System MUST label event-driven improvement sessions with `feedback-loop=true`, `source=event-driven`, and `target-type={workflow|repo}` to distinguish them from GHA-created sessions (which use `source=github-action`).
- **FR-010**: The weekly GHA batch MUST continue to run as a sweep and MUST skip corrections that already triggered event-driven sessions within its lookback window.
- **FR-011**: System MUST persist correction counts and deduplication state durably (not in-memory only) so that backend restarts do not lose state.
- **FR-012**: System MUST cap the number of corrections included in a single improvement prompt at 50, with a summary count for any remainder.

### Non-Functional Requirements

- **NFR-001**: Event-driven improvement sessions MUST be created within 5 minutes of the threshold-crossing correction being logged.
- **NFR-002**: The corrections buffer evaluation MUST NOT add measurable latency to the corrections logging path (async processing).
- **NFR-003**: Config and history endpoints MUST respond within 500ms under normal load.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An improvement session is created within 5 minutes of the threshold-crossing correction, verified by timestamp comparison between the last correction score and the session creation time.
- **SC-002**: Zero duplicate improvement sessions are created for the same target within a single time window, verified by querying sessions with `feedback-loop=true` label grouped by target.
- **SC-003**: The weekly GHA run creates zero sessions for targets already addressed by event-driven sessions, verified by GHA logs showing skip count.
- **SC-004**: Config changes via `PUT` take effect within 60 seconds for subsequent threshold evaluations.
- **SC-005**: Backend restart does not reset correction counts -- after restart, previously buffered corrections still contribute toward the threshold.

## Assumptions

- The corrections realtime pipeline (spec 003) is implemented and actively logging `session-correction` scores to Langfuse with the expected metadata schema (`target_type`, `target_repo_url`, `target_branch`, `target_path`).
- The backend has access to Langfuse credentials (already available via environment variables for other features).
- The `build_improvement_prompt()` logic in `query_corrections.py` is stable and can be ported to Go without behavioral changes, or invoked as a subprocess if porting is deferred.
- Project admin permissions are already enforced by `ValidateProjectContext()` middleware for write operations on project-scoped resources.
- The `ambient-code/ambient-action` GitHub Action used by the weekly GHA can query existing sessions by label to detect event-driven sessions.

## Dependencies

- **Spec 003 - Corrections Realtime Pipeline**: Provides the corrections buffer that this feature watches. Without it, there are no real-time corrections to evaluate.
- **Langfuse API**: Scores are queried for history and deduplication cross-referencing between event-driven and GHA paths.
- **Backend session creation**: Reuses `CreateSession` handler logic (`components/backend/handlers/sessions.go`) for creating improvement sessions programmatically.
- **Weekly GHA**: `.github/workflows/feedback-loop.yml` must be updated to check for existing event-driven sessions before creating batch sessions.
- **Existing query script**: `scripts/feedback-loop/query_corrections.py` contains the prompt construction and grouping logic that must be consistent between both paths.
