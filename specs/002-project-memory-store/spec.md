# Feature Specification: Project Memory Store (File-Based)

**Feature Branch**: `002-project-memory-store`
**Created**: 2026-04-15
**Revised**: 2026-04-16
**Status**: Draft

> **Architecture**: Memories are stored as markdown files in the workspace repository's `docs/learned/`
> directory and submitted as draft PRs with the `continuous-learning` label. Knowledge reaches agents
> through the **wiki compiler → CLAUDE.md** pipeline — not through custom runner code. This follows the
> pattern established in `jeremyeder/continuous-learning-example` and the `llm-wiki-compiler` plugin.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Knowledge Flows Through Wiki Compiler (Priority: P1)

When learned files are merged into `docs/learned/` on the default branch, the wiki compiler (`llm-wiki-compiler`) runs — either automatically via GitHub Action on push, or manually via `/wiki-compile`. It compiles `docs/learned/` alongside other documentation sources into topic-based wiki articles in `docs/wiki/`. At session start, the wiki-compiler SessionStart hook injects `docs/wiki/INDEX.md` into the agent's context. The agent picks up accumulated knowledge naturally through the same path as all other project documentation.

**Why this priority**: This is the delivery mechanism. Without it, merged learned files sit in the repo with no effect on agent behavior.

**Independent Test**: Create a workspace repo with `.wiki-compiler.json` including `docs/learned/` as a source, 3 learned files in `docs/learned/corrections/` and 2 in `docs/learned/patterns/`. Run the wiki compiler. Start a session. Confirm the agent has access to the compiled wiki content.

**Acceptance Scenarios**:

1. **Given** a workspace repo with `.wiki-compiler.json` sourcing `docs/learned/`, **When** the wiki compiler runs, **Then** learned files are compiled into topic-based articles in `docs/wiki/`.
2. **Given** compiled wiki articles exist in `docs/wiki/`, **When** a session starts, **Then** the wiki-compiler SessionStart hook injects the wiki index into the agent's context.
3. **Given** a workspace repo with no `docs/learned/` directory, **When** the wiki compiler runs, **Then** it compiles other sources normally (no error, no empty section).
4. **Given** a workspace repo without `.wiki-compiler.json`, **When** a session starts, **Then** no wiki injection occurs (standard behavior, no error).

---

### User Story 2 - Opt-In via .ambient/config.json (Priority: P1)

A project maintainer enables continuous learning by adding `.ambient/config.json` with `{"learning": {"enabled": true}}` to the workspace repository. This config controls whether the `suggest_memory` tool (spec 008) and insight extraction (spec 009) write new learned files. It does NOT control wiki compilation or knowledge delivery — those are handled by `.wiki-compiler.json` and the SessionStart hook independently.

**Why this priority**: Opt-in prevents agents from writing unsolicited learned files to repos that haven't adopted the CL pipeline.

**Acceptance Scenarios**:

1. **Given** `.ambient/config.json` with `{"learning": {"enabled": true}}`, **When** the agent calls `suggest_memory`, **Then** a draft PR is created.
2. **Given** `.ambient/config.json` with `{"learning": {"enabled": false}}`, **When** the agent calls `suggest_memory`, **Then** the tool returns an error.
3. **Given** no `.ambient/config.json` exists, **When** the agent calls `suggest_memory`, **Then** the tool returns an error (opt-in, not opt-out).
4. **Given** `.ambient/config.json` with malformed JSON, **When** the agent calls `suggest_memory`, **Then** the tool returns an error and logs a warning.

---

### User Story 3 - Backend Lists Learned Files via Workspace API (Priority: P1)

A frontend component needs to display what the project has learned. The backend exposes a thin API that reads `docs/learned/` from the workspace repo via the GitHub API. It returns a list of learned entries with their frontmatter metadata and content. No new CRUD handlers, no new CRDs — just a read-only view of files in the repo.

**Why this priority**: The UI needs a data source for the Project Memory tab (spec 008).

**Independent Test**: With 5 files in `docs/learned/`, call `GET /api/projects/:projectName/learned`. Verify all 5 entries are returned with parsed frontmatter and content.

**Acceptance Scenarios**:

1. **Given** a workspace repo with learned files, **When** `GET /api/projects/:projectName/learned` is called, **Then** the response contains an array of entries with `title`, `type`, `date`, `author`, `content`, and `filePath`.
2. **Given** a workspace repo with no `docs/learned/` directory, **When** the endpoint is called, **Then** it returns an empty array with 200 (not 404).
3. **Given** a user without access to the project, **When** they call the endpoint, **Then** the request is rejected with 403 (enforced via user token, `GetK8sClientsForRequest`).
4. **Given** filtering by `?type=correction`, **When** the endpoint is called, **Then** only entries with `type: correction` frontmatter are returned.

---

### User Story 4 - Learned File Format and Directory Structure (Priority: P1)

Learned files follow the format established in `jeremyeder/continuous-learning-example`:

```
docs/learned/
  corrections/           # Corrections captured from sessions
    2026-04-01-use-pydantic-v2.md
  patterns/              # Patterns and conventions discovered
    2026-04-02-error-response-format.md
```

Each file has YAML frontmatter:

```yaml
---
type: correction | pattern
date: 2026-04-01T14:30:00Z
session: session-name
project: project-name
author: Agent | User Name
title: "Short descriptive title"
---

Body text describing the correction or pattern.
```

**Why this priority**: A stable file format is required before any component can read or write learned files.

**Acceptance Scenarios**:

1. **Given** a learned file with valid frontmatter, **When** the backend parses it, **Then** all frontmatter fields are extracted correctly.
2. **Given** a learned file with missing optional fields (session, project), **When** the backend parses it, **Then** the file is still loaded with available fields.
3. **Given** a file in `docs/learned/` without valid frontmatter, **When** the backend parses it, **Then** it is skipped with a warning (not a fatal error).

---

### User Story 5 - Wiki Compiler Configuration (Priority: P1)

A workspace repo that opts in to CL must have a `.wiki-compiler.json` with `docs/learned/` in the `sources` array. A GitHub Action (`compile-wiki-ambient.yml`) triggers wiki compilation when `docs/learned/` or `docs/` files change on push to main.

**Acceptance Scenarios**:

1. **Given** `.wiki-compiler.json` with `{"sources": [{"path": "docs/", "exclude": ["wiki/"]}]}`, **When** learned files are merged to main, **Then** the GitHub Action triggers wiki compilation.
2. **Given** compiled wiki output in `docs/wiki/`, **When** the wiki-compiler SessionStart hook runs, **Then** the compiled wiki is injected into the agent's context based on the configured mode (staging/recommended/primary).

---

### Edge Cases

- **Empty `docs/learned/`**: Wiki compiler has no learned sources but compiles other docs normally. Backend returns empty array.
- **Draft PRs with learned files**: Files on unmerged branches are NOT compiled. Only merged files on the default branch are compiled.
- **Binary files in `docs/learned/`**: Skipped by wiki compiler and backend (only `.md` files are processed).
- **Concurrent sessions writing**: Each session writes to a unique branch (`learned/<type>-<date>-<slug>`), so no git conflicts.
- **Wiki compiler not installed**: Knowledge delivery degrades gracefully — agents don't get compiled wiki, but the system still works. Backend endpoint still serves raw learned files for the UI.
- **Large number of learned files**: Wiki compiler synthesizes them into topic articles, naturally compressing the knowledge. No token budget management needed in the runner.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Workspace repos MUST opt in to CL via `.ambient/config.json` with `{"learning": {"enabled": true}}`. This controls whether write-side features (`suggest_memory`, insight extraction) are active.
- **FR-002**: Workspace repos MUST include a `.wiki-compiler.json` with `docs/learned/` in the `sources` array for knowledge to reach agents via the wiki compiler pipeline.
- **FR-003**: A GitHub Action (`compile-wiki-ambient.yml`) MUST trigger wiki compilation when files under `docs/` change on push to the default branch.
- **FR-004**: The wiki-compiler SessionStart hook handles knowledge injection into agent context — no custom runner code is needed.
- **FR-005**: Backend MUST expose `GET /api/projects/:projectName/learned` returning parsed learned file entries with frontmatter metadata and content.
- **FR-006**: Backend MUST support filtering the learned endpoint by `type` query parameter.
- **FR-007**: Backend MUST use `GetK8sClientsForRequest` for user-scoped RBAC on the learned endpoint.
- **FR-008**: Learned files MUST use the frontmatter schema: `type` (required, enum: `correction`, `pattern`), `date` (required, ISO 8601), `title` (required, string), `session` (optional), `project` (optional), `author` (optional).
- **FR-009**: Feature MUST be gated behind the `learning-agent-loop` feature flag.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Learned files merged to the default branch are compiled by the wiki compiler into topic articles accessible to agents.
- **SC-002**: A session started in a workspace with compiled wiki articles has access to the accumulated knowledge via the SessionStart hook.
- **SC-003**: `GET /api/projects/:projectName/learned` returns correct entries matching files in the repo, verified by integration test.
- **SC-004**: The system works end-to-end: learned file merged → wiki compiled → agent reads knowledge → cites it in responses.

## Assumptions

- The `llm-wiki-compiler` plugin is installed and its SessionStart hook is active.
- The `.wiki-compiler.json` and GitHub Action setup is a one-time per-repo configuration.
- The wiki compiler handles topic synthesis, coverage indicators, and context compression — the platform does not need to duplicate this logic.
- The backend can read workspace repo files via the GitHub API.

## Dependencies

- `llm-wiki-compiler` plugin: SessionStart hook for injection, `/wiki-compile` skill for compilation
- `.wiki-compiler.json`: Per-repo compiler configuration
- GitHub Actions: `compile-wiki-ambient.yml` for auto-compilation on push
- Backend workspace file access: `components/backend/handlers/` (existing pattern)
- Route registration: `components/backend/routes.go` under `projectGroup`
- Feature flag: `learning-agent-loop` in `components/manifests/base/core/flags.json`
- Reference implementation: `jeremyeder/continuous-learning-example`
