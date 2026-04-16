# Feature Specification: Visible Attribution

**Feature Branch**: `006-visible-attribution`
**Created**: 2026-04-15
**Status**: Draft

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Agent Cites a Memory in Its Response (Priority: P1)

A user starts a session in a project that has accumulated project memories (via the Project Memory Store, spec 002). During session initialization, the runner injects relevant memory entries into the agent's system prompt, each tagged with a unique ID (e.g., `PM-042`) and source metadata (author, creation date, originating correction ID). The system prompt includes an instruction block telling the agent to cite any memory it relies on using the format `[memory:PM-042]`. When the agent produces a response that draws on a memory — for example, excluding internal instances from error rate analysis — the citation appears inline in the chat output.

**Why this priority**: Attribution is the core value proposition. Without inline citations, users have no visibility into whether corrections are taking effect. Everything else in this spec builds on this.

**Independent Test**: Create a project with at least one memory entry. Start a session whose prompt triggers use of that memory. Verify the agent's response contains a `[memory:PM-XXX]` citation referencing the correct entry.

**Acceptance Scenarios**:

1. **Given** a project with 5 memory entries injected into the system prompt, **When** the agent uses knowledge from memory PM-042 in its response, **Then** the response contains the citation `[memory:PM-042]` inline at the point where that knowledge is applied.
2. **Given** a project with memory entries, **When** the agent's response does not rely on any memory, **Then** no citations appear in the output.
3. **Given** a project with memory entries, **When** the agent uses multiple memories in a single response, **Then** each usage is cited independently with the correct memory ID.
4. **Given** a session in a project with no memories, **When** the agent responds, **Then** the citation instruction block is omitted from the system prompt entirely (no empty scaffolding).

---

### User Story 2 - Frontend Renders Citation as Clickable Badge (Priority: P1)

A user reading the agent's response in the frontend sees memory citations rendered as styled inline badges (not raw bracket text). Each badge displays the memory ID and a truncated summary of the memory content. Badges are visually distinct from surrounding text — using the project's accent color, a pill/chip shape, and a subtle icon — so they are scannable without being distracting.

**Why this priority**: Raw citation syntax is meaningless to users. Rendering citations as interactive UI elements is what transforms them from a debug artifact into a trust-building feature.

**Independent Test**: Send a message that triggers a memory citation. Inspect the rendered chat output. Verify the `[memory:PM-XXX]` text is replaced by a styled badge component, and that clicking it navigates to the memory detail view.

**Acceptance Scenarios**:

1. **Given** an agent response containing `[memory:PM-042]`, **When** the frontend renders the message, **Then** the raw text is replaced by a `<MemoryCitationBadge>` component displaying "PM-042" and a truncated summary.
2. **Given** a rendered citation badge, **When** the user clicks it, **Then** a popover or slide-over panel opens showing the full memory entry (content, author, creation date, originating correction).
3. **Given** an agent response with no citations, **When** the frontend renders the message, **Then** no badge components are injected and the message renders identically to today's behavior.
4. **Given** an agent response containing multiple citations, **When** the frontend renders it, **Then** each citation is rendered as a separate badge at the correct inline position within the text.

---

### User Story 3 - User Views Corrections Impact Dashboard (Priority: P2)

A project admin navigates to a new "Learning" tab in the project view. The dashboard shows a summary of the correction-to-improvement pipeline: total corrections submitted, corrections broken down by type (pie chart), a timeline of recent correction activity, the number of improvement sessions spawned, and the number of memories created. The goal is a simple "input → output" narrative: your team submitted N corrections, which produced M improvement sessions, which created K memories now used in L sessions.

**Why this priority**: The dashboard is high-value for demonstrating ROI of the corrections workflow, but it is a read-only reporting view — the system works without it.

**Independent Test**: In a project with at least 3 corrections (mixed types), navigate to the Learning tab. Verify all summary metrics render correctly, the breakdown chart reflects actual correction types, and the timeline shows entries in chronological order.

**Acceptance Scenarios**:

1. **Given** a project with 10 corrections, 4 improvement sessions, and 6 memories, **When** the user opens the Learning tab, **Then** the summary cards display "10 corrections", "4 improvement sessions", "6 memories created".
2. **Given** a project with corrections of types "factual", "style", and "process", **When** the user views the breakdown chart, **Then** each type is represented proportionally in the chart with correct counts.
3. **Given** a project with correction activity over the past 30 days, **When** the user views the timeline, **Then** entries appear in reverse chronological order with date, type, and a one-line summary.
4. **Given** a project with zero corrections, **When** the user opens the Learning tab, **Then** an empty state is displayed with guidance on how to submit corrections (linking to spec 005 "Correct This" UX).

---

### User Story 4 - User Clicks Citation to View and Edit the Memory (Priority: P2)

A user clicks a citation badge in a chat message and is taken to the full memory detail view. From this view, the user can read the complete memory content, see its provenance (which correction created it, when, by whom), and edit or deprecate the memory if it is no longer accurate. Edits are versioned so the original content is preserved.

**Why this priority**: Citations are only trustworthy if users can verify and correct the underlying knowledge. Without this, stale memories silently degrade agent quality.

**Independent Test**: Click a citation badge. Verify the memory detail view loads with correct content and provenance. Edit the memory text, save, and confirm the edit is reflected in subsequent sessions while the original version is preserved in history.

**Acceptance Scenarios**:

1. **Given** a user clicks a citation badge for PM-042, **When** the memory detail view opens, **Then** it displays: full memory text, creation date, author, originating correction ID (linked), and usage count (how many sessions have cited this memory).
2. **Given** a user edits the memory text and saves, **When** the edit is persisted, **Then** the previous version is stored in a version history list and the updated text is used in future session prompt injection.
3. **Given** a user marks a memory as deprecated, **When** the memory is saved with deprecated status, **Then** it is excluded from future system prompt injection but remains visible in the dashboard and version history with a "deprecated" label.
4. **Given** a user views version history for a memory, **When** the history panel is open, **Then** each version shows the text diff, editor, and timestamp.

---

### Edge Cases

- **Stale memory cited**: Agent cites a memory that has since been edited. The citation badge shows the current (updated) version with a subtle "updated since cited" indicator, and the version history link is accessible.
- **Deleted/deprecated memory cited**: Agent cites a memory that has been deprecated or soft-deleted. The citation badge renders with a strikethrough style and a tooltip: "This memory has been deprecated." Clicking it still opens the detail view in read-only mode.
- **Malformed citation in agent output**: Agent produces a citation with an invalid ID (e.g., `[memory:PM-999]` where PM-999 does not exist). The frontend renders it as plain text with a warning icon and tooltip: "Memory not found."
- **Citation in code block**: Agent output contains `[memory:PM-XXX]` inside a fenced code block. The frontend does NOT transform it into a badge — code blocks are rendered verbatim.
- **High citation density**: Agent cites more than 10 memories in a single response. Badges remain individually rendered but a "N memories cited" summary chip appears at the top of the message for quick scanning.
- **Concurrent memory edit**: Two users edit the same memory simultaneously. Last-write-wins with optimistic concurrency (version field); the losing write receives a conflict error prompting a reload.
- **Memory injection exceeds context window**: The total size of injected memories approaches the system prompt budget. The runner truncates the lowest-priority memories and logs a warning; truncated memories cannot be cited.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The runner MUST inject each project memory entry into the agent's system prompt with a unique ID, source metadata (author, creation date, originating correction ID), and a one-line summary.
- **FR-002**: The system prompt MUST include a citation instruction block directing the agent to cite memories using the format `[memory:PM-XXX]` inline in its responses when it uses knowledge from that memory.
- **FR-003**: The system prompt citation instruction block MUST be omitted when the project has zero active memories.
- **FR-004**: The frontend MUST parse `[memory:PM-XXX]` patterns in agent messages and render them as styled `MemoryCitationBadge` components (pill/chip shape, project accent color, memory ID, truncated summary).
- **FR-005**: The frontend MUST NOT transform `[memory:PM-XXX]` patterns that appear inside fenced code blocks.
- **FR-006**: Clicking a `MemoryCitationBadge` MUST open the memory detail view showing full content, provenance, usage count, and version history.
- **FR-007**: The memory detail view MUST support inline editing of memory text with optimistic concurrency control (version field).
- **FR-008**: Memory edits MUST be versioned — previous versions are preserved and accessible in a version history panel showing diffs, editor, and timestamp.
- **FR-009**: The memory detail view MUST support marking a memory as deprecated, which excludes it from future prompt injection while preserving it in history.
- **FR-010**: The backend MUST expose a `GET /api/v1/projects/{project}/learning/summary` endpoint returning: total corrections, corrections by type, total improvement sessions, total memories created, and total memory citations across sessions.
- **FR-011**: The backend MUST expose a `GET /api/v1/projects/{project}/learning/timeline` endpoint returning a paginated, reverse-chronological list of correction events with date, type, summary, linked improvement session (if any), and linked memory (if any).
- **FR-012**: The frontend MUST render a "Learning" tab in the project view containing summary cards, a correction-type breakdown chart, and a timeline of recent activity.
- **FR-013**: The "Learning" tab MUST display an empty state with guidance when the project has zero corrections.
- **FR-014**: Citation badges for deprecated or deleted memories MUST render with a strikethrough style and a descriptive tooltip.
- **FR-015**: Citation badges referencing non-existent memory IDs MUST render as plain text with a warning icon.
- **FR-016**: When more than 10 memories are cited in a single agent response, the frontend MUST display a summary chip at the top of the message showing the total count.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In sessions where memories are injected, at least 70% of agent responses that demonstrably use a memory include an inline citation (measured over a 30-day sample across all projects).
- **SC-002**: Citation badges render correctly (not as raw text) in 100% of agent messages containing valid `[memory:PM-XXX]` patterns outside code blocks.
- **SC-003**: Clicking a citation badge loads the memory detail view within 500ms (p95).
- **SC-004**: The Learning tab summary metrics match backend data with zero discrepancy (verified by E2E tests).
- **SC-005**: Memory edits are reflected in the next session's prompt injection within 60 seconds of save.
- **SC-006**: Deprecated memories are excluded from prompt injection in 100% of sessions started after deprecation.
- **SC-007**: The Learning tab renders correctly for projects with zero corrections (empty state) and projects with 1000+ corrections (pagination, no UI freezing).

## Assumptions

- The Project Memory Store (spec 002) is implemented and memories are stored as structured entries with unique IDs, content, metadata, and a version field.
- The Corrections Realtime Pipeline (spec 003) is implemented and corrections are persisted with type classifications.
- Knowledge from `docs/learned/` reaches agents via the wiki compiler → CLAUDE.md pipeline (spec 002). The citation instruction block is added to compiled wiki articles or CLAUDE.md.
- The agent (Claude) reliably follows citation format instructions in the system prompt when the instruction is clear and the memory entries are well-structured. Citation rate may be below 100% — this is expected LLM behavior and is reflected in SC-001.
- Shadcn/UI provides the base components (Badge, Popover, Card, Tabs) needed for the citation badges and dashboard.
- The frontend chat message renderer already supports custom inline component injection (or can be extended to support it via a markdown post-processing step).

## Dependencies

- **Spec 002 — Project Memory Store**: Provides the memory entries, unique IDs, CRUD API, and version history that this spec surfaces to users.
- **Spec 003 — Corrections Realtime Pipeline**: Provides the correction events, type classifications, and improvement session linkage that the Learning dashboard aggregates.
- **Spec 005 — "Correct This" UX**: The empty-state guidance in the Learning tab links to the correction submission flow defined in spec 005.
- **Runner prompt injection**: `components/runners/ambient-runner/` — the system prompt assembly logic where memory entries and citation instructions are injected.
- **Frontend chat renderer**: `components/frontend/src/` — the message rendering pipeline where citation patterns are parsed and replaced with badge components.
- **Frontend project views**: `components/frontend/src/app/projects/[name]/` — where the Learning tab is added to the project navigation.
- **Backend API**: `components/backend/handlers/` — where the `/learning/summary` and `/learning/timeline` endpoints are implemented.
