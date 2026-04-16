# Feature Specification: Cross-Session Memory

**Feature Branch**: `008-cross-session-memory`
**Created**: 2026-04-15
**Revised**: 2026-04-16
**Status**: Draft

> **Architecture change**: This spec originally consumed a CRD-based Project Memory Store.
> It now consumes the file-based `docs/learned/` pipeline from spec 002. Memories are markdown
> files in the workspace repo, submitted as draft PRs with the `continuous-learning` label.
> Human curation happens through git PR review.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Viewing and Managing Learned Files in the UI (Priority: P1)

A project maintainer navigates to the "Project Memory" tab in their project view to review the knowledge the project has accumulated. The UI calls `GET /api/projects/:projectName/learned` (spec 002) and displays the parsed entries. They see corrections and patterns with their titles, dates, authors, and content previews. They can filter by type. To edit or archive a learned file, they open a PR against the repo (the UI links to the file on GitHub).

**Why this priority**: The management UI is the primary interface for reviewing accumulated knowledge. Without it, users must browse the repo manually.

**Independent Test**: Navigate to a project with 5 learned files across corrections/ and patterns/. Filter by type. Click through to GitHub links. Confirm all entries render correctly.

**Acceptance Scenarios**:

1. **Given** a project with learned files in `docs/learned/`, **When** a user navigates to the Project Memory tab, **Then** they see entries listed with title, type badge, author, date, and content preview (first 200 characters).
2. **Given** the memory list is displayed, **When** a user selects the "Correction" type filter, **Then** only correction entries are shown.
3. **Given** a learned file entry, **When** a user clicks "View on GitHub", **Then** they are taken to the file on the repo's default branch.
4. **Given** draft PRs exist with the `continuous-learning` label, **When** a user views the "Pending Review" section, **Then** they see a list of draft PRs awaiting curation, each linking to the PR on GitHub.
5. **Given** a project with no learned files, **When** a user navigates to the Project Memory tab, **Then** an empty state is displayed explaining what memories are and how they accumulate.

---

### User Story 2 - Runner Loads Memories at Session Init (Priority: P1)

When a new agentic session starts, the wiki-compiler SessionStart hook injects compiled wiki articles (which include synthesized knowledge from `docs/learned/`) into the agent's context. The agent begins the session aware of past corrections and patterns through the same path as all other project documentation.

**Why this priority**: This is the core value proposition — sessions that learn from prior sessions.

**Independent Test**: Create a workspace repo with learned files, run the wiki compiler, then start a session. Confirm the agent has access to the compiled knowledge.

**Acceptance Scenarios**:

1. **Given** a project with compiled wiki articles including learned knowledge, **When** a session starts, **Then** the wiki-compiler SessionStart hook injects the wiki index into the agent's context.
2. **Given** no learned files or wiki articles exist, **When** a session starts, **Then** no wiki injection occurs (standard behavior).
3. **Given** the wiki compiler has not been run yet, **When** a session starts, **Then** the session proceeds normally without compiled knowledge (graceful degradation).

---

### User Story 3 - Agent Suggests a New Memory via Tool (Priority: P1)

During a session, the agent discovers something worth remembering — a non-obvious environment detail, a correction from the user, or a pattern it found effective. It calls the `suggest_memory` tool to propose a new learned file. The runner writes the file to a new branch (`learned/<type>-<date>-<slug>`) in the workspace repo and opens a draft PR with the `continuous-learning` label. The learned file appears in the "Pending Review" section of the UI for human curation.

**Why this priority**: Automated memory suggestion is what makes the system self-improving. Manual-only entry does not scale.

**Independent Test**: Start a session in a CL-enabled workspace. Invoke the `suggest_memory` tool with content, type, and title. Confirm a draft PR is created in the workspace repo with the correct file in `docs/learned/` and the `continuous-learning` label.

**Acceptance Scenarios**:

1. **Given** an active session in a CL-enabled workspace, **When** the agent calls `suggest_memory` with `content`, `type` (correction|pattern), and `title`, **Then** the runner writes a markdown file to `docs/learned/<type>s/<date>-<slug>.md` on a new branch and opens a draft PR with the `continuous-learning` label.
2. **Given** a memory suggested by an agent, **When** a user views the "Pending Review" section, **Then** the draft PR appears with the file content, originating session name, and a link to the PR.
3. **Given** an active session, **When** the agent calls `suggest_memory` with an invalid type, **Then** the tool returns an error message indicating valid types (`correction`, `pattern`).
4. **Given** an active session, **When** the agent calls `suggest_memory` with empty content, **Then** the tool returns an error and no file is created.
5. **Given** an active session, **When** the agent calls `suggest_memory` and the git push fails, **Then** the tool returns an error message but does not crash the session.
6. **Given** a CL-disabled workspace, **When** the agent calls `suggest_memory`, **Then** the tool returns an error: "Continuous learning is not enabled for this workspace."

---

### User Story 4 - User Manually Adds a Memory (Priority: P2)

A project maintainer wants to record a known fact. They click "Add Memory" in the Project Memory tab, enter a title, type, and content. The frontend creates a PR in the workspace repo with the new file in `docs/learned/`. Since it's a manual entry by a project maintainer, the PR can be merged directly (not draft).

**Why this priority**: Manual entry is a secondary input channel for bootstrapping knowledge that doesn't emerge from sessions.

**Independent Test**: Click "Add Memory", fill in fields, submit. Confirm a PR is created in the workspace repo with the correct file.

**Acceptance Scenarios**:

1. **Given** the Project Memory tab, **When** a user clicks "Add Memory", **Then** a form appears with fields for title (text), content (multiline), and type (dropdown: correction, pattern).
2. **Given** the Add Memory form, **When** a user submits with valid fields, **Then** a PR is created in the workspace repo with a file in `docs/learned/<type>s/<date>-<slug>.md`.
3. **Given** the Add Memory form, **When** a user submits with empty content, **Then** a validation error is shown and no PR is created.

---

### Edge Cases

- **Empty memory state**: Project Memory tab shows empty state with guidance. No `## Project Memory` section injected.
- **Large number of learned files**: Runner enforces token budget (~4,000 tokens), truncating oldest entries with a count note. UI paginates (50 per page).
- **Stale memories**: No automatic expiration. Users archive by deleting files via PR or moving to an `archived/` subdirectory.
- **Duplicate content**: Deduplication is a human judgment call during PR review. No automated dedup.
- **Draft PR not yet merged**: Files on unmerged branches are NOT injected into sessions. They appear only in "Pending Review".
- **Workspace repo without git write access**: `suggest_memory` tool returns a clear error. Session continues.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a "Project Memory" tab in the project view displaying learned file entries from `GET /api/projects/:projectName/learned` (spec 002).
- **FR-002**: System MUST display learned entries with: title, type badge (correction/pattern), author, date, and content preview (first 200 characters).
- **FR-003**: System MUST support filtering entries by type.
- **FR-004**: System MUST display a "Pending Review" section listing open draft PRs with the `continuous-learning` label in the workspace repo, with links to each PR on GitHub.
- **FR-005**: System MUST provide "View on GitHub" links for each learned file, pointing to the file on the repo's default branch.
- **FR-006**: Knowledge from `docs/learned/` reaches agents via the wiki compiler → CLAUDE.md pipeline (spec 002). No custom runner code is needed for knowledge delivery.
- **FR-007**: Runner MUST register a `suggest_memory` tool that accepts `content` (string, required), `type` (enum: correction|pattern, required), and `title` (string, required).
- **FR-008**: The `suggest_memory` tool MUST write a markdown file with proper frontmatter to `docs/learned/<type>s/<date>-<slug>.md` on a new branch named `learned/<type>-<date>-<slug>`.
- **FR-009**: The `suggest_memory` tool MUST open a draft PR with the `continuous-learning` label, including the originating session name in the PR description.
- **FR-010**: The `suggest_memory` tool MUST validate inputs and return actionable error messages for invalid type, empty content, or CL-disabled workspace.
- **FR-011**: The `suggest_memory` tool MUST treat git/GitHub failures as non-fatal — return error to agent, do not crash session.
- **FR-012**: System MUST provide an "Add Memory" form for manual entry that creates a PR in the workspace repo.
- **FR-013**: System MUST paginate the learned entries list (50 per page).
- **FR-014**: Feature MUST be gated behind the `learning-agent-loop` feature flag.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can view, filter, and link to learned files from the Project Memory tab.
- **SC-002**: A new session's system prompt contains all learned files from the default branch, grouped by type.
- **SC-003**: An agent can call `suggest_memory` during a session and a draft PR appears in the workspace repo within 30 seconds.
- **SC-004**: Memory fetch failure at session init does not prevent the session from starting.
- **SC-005**: The "Pending Review" section accurately reflects open draft PRs with the `continuous-learning` label.

## Assumptions

- Spec 002 provides the runner-side `docs/learned/` reading and system prompt injection.
- The runner has git push access to the workspace repo (via the session's credentials or a configured deploy key).
- The GitHub API is available for creating branches, commits, and PRs from the runner.
- The existing project layout can accommodate a new top-level "Project Memory" tab.
- The Claude Agent SDK supports registering additional tools at session init time.

## Dependencies

- **Spec 002 — Project Memory Store (File-Based)**: Backend read API, file format, and wiki compiler pipeline.
- **Spec 007 — Event-Driven Feedback Loop**: Improvement sessions may write learned files. Cross-session memory can ship without this.
- **Spec 009 — Post-Session Insight Extraction**: Writes learned files after session completion. Can ship independently.
- **Runner tool infrastructure**: `components/runners/ambient-runner/ambient_runner/bridges/claude/tools.py`
- **Frontend project navigation**: `components/frontend/src/app/projects/[name]/layout.tsx`
- **GitHub API access**: For creating branches, commits, PRs from the runner.
