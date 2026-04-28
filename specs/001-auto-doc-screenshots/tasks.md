# Tasks: Automated Documentation Screenshots

**Input**: Design documents from `/specs/001-auto-doc-screenshots/`
**Prerequisites**: plan.md, spec.md, research.md, quickstart.md

**Tests**: Not explicitly requested. The screenshot Cypress spec IS the validation mechanism.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Exact file paths included in descriptions

---

## Phase 1: Setup

**Purpose**: Create directories and config scaffolding

- [x] T001 Create screenshots directory `docs/public/images/screenshots/.gitkeep`
- [x] T002 [P] Add `cypress/screenshots/output/` to `e2e/.gitignore`

---

## Phase 2: Foundational

**Purpose**: CSS and Cypress infrastructure that all stories depend on

- [x] T003 Add screenshot theme-switching CSS rules to `docs/src/styles/custom.css`
- [x] T004 [P] Create screenshot manifest at `e2e/cypress/screenshots/manifest.json` with all 10 entries
- [x] T005 Create Cypress screenshot spec at `e2e/cypress/e2e/screenshots.cy.ts` with manifest-driven capture, theme toggling, workspace/session setup, and setup step handlers
- [x] T006 [P] Add `screenshotsFolder` and DPI normalization (`CYPRESS_SCREENSHOT_MODE`) to `e2e/cypress.config.ts`
- [x] T007 Add `screenshots`, `screenshots-headed`, `screenshots-clean` targets to `Makefile`
- [x] T007a Create Netlify config at `docs/netlify.toml` with build command (`npm ci && npx playwright install chromium && npm run build`), `/platform/*` redirect
- [x] T007b Update `docs/astro.config.mjs` for environment-aware `site`/`base` using `process.env.NETLIFY`

**Checkpoint**: `make screenshots` can run against a kind cluster; Netlify site renders with correct CSS

---

## Phase 3: User Story 1 — Theme-Aware Screenshots in Docs (Priority: P1)

**Goal**: Documentation pages display screenshots that auto-switch with the reader's theme.

**Independent Test**: Open docs dev server, navigate to quickstart page, toggle Starlight theme — correct screenshot variant displays.

- [x] T008 [US1] Embed screenshot HTML in `docs/src/content/docs/getting-started/quickstart-ui.md` (workspaces-page, integrations-page, new-session-dialog, session-page)
- [x] T009 [P] [US1] Embed screenshot HTML in `docs/src/content/docs/concepts/sessions.md` (session-list, session-page, new-session-dialog)
- [x] T010 [P] [US1] Embed screenshot HTML in `docs/src/content/docs/concepts/integrations.md` (integrations-page)
- [x] T011 [P] [US1] Embed screenshot HTML in `docs/src/content/docs/concepts/workspaces.md` (workspace-settings, workspace-sharing, api-keys)
- [x] T012 [P] [US1] Embed screenshot HTML in `docs/src/content/docs/concepts/scheduled-sessions.md` (scheduled-sessions)
- [x] T013 [P] [US1] Embed screenshot HTML in `docs/src/content/docs/concepts/context-and-artifacts.md` (file-browser)
- [x] T014 [US1] Verify docs build succeeds: `cd docs && npm run build`
- [x] T015 [US1] Capture screenshots locally: `make kind-up CONTAINER_ENGINE=docker && make screenshots`
- [x] T016 [US1] Verify theme switching in docs dev server: `cd docs && npm run dev`, toggle themes
- [x] T017 [US1] Commit captured screenshots in `docs/public/images/screenshots/`

**Checkpoint**: Docs pages show theme-aware screenshots locally

---

## Phase 4: User Story 2 — Daily Automated Screenshot Capture (Priority: P1)

**Goal**: CI workflow captures screenshots daily and opens PRs when images change.

**Independent Test**: Trigger workflow via `workflow_dispatch`, confirm it completes and creates a PR.

- [x] T018 [US2] Create GHA workflow at `.github/workflows/screenshots.yml` with daily cron, kind setup, Cypress capture, change detection, and auto-PR creation
- [ ] T019 [US2] Push branch and trigger workflow via `workflow_dispatch` to validate

**Checkpoint**: CI pipeline captures screenshots and opens PRs autonomously

---

## Phase 5: User Story 3 — Two-Tier Documentation Hosting (Priority: P2)

**Goal**: Netlify deploys main continuously, GitHub Pages deploys release tags only.

**Independent Test**: Push docs change to main — appears on Netlify, not on GitHub Pages.

- [x] T020 [US3] Change `.github/workflows/docs.yml` trigger from `push: branches: [main]` to `push: tags: ['v*']`
- [x] T021 [US3] Verify local docs build: `cd docs && npm run build`
- [ ] T022 [US3] Push to main, verify Netlify deploys with correct CSS and screenshot paths

**Checkpoint**: Two-tier hosting operational — Netlify for main, GitHub Pages for releases

---

## Phase 6: Polish

- [x] T025 Verify all 9 screenshot pairs exist in `docs/public/images/screenshots/`
- [ ] T026 Run quickstart.md validation — follow all steps, confirm they work
- [ ] T027 Clean up kind cluster: `make kind-down`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies
- **Foundational (Phase 2)**: Depends on Phase 1
- **US1 (Phase 3)**: Depends on Phase 2 (needs CSS + Cypress spec)
- **US2 (Phase 4)**: Depends on Phase 2 (needs Cypress spec + manifest)
- **US3 (Phase 5)**: No dependency on US1 or US2 — can run in parallel
- **Polish (Phase 6)**: Depends on all stories complete

### User Story Dependencies

- **US1 (P1)**: Needs kind cluster running for T015. Foundational phase must be done.
- **US2 (P1)**: Independent of US1. Can start after Foundational.
- **US3 (P2)**: Fully independent — only touches docs config and GHA, no Cypress dependency.

### Parallel Opportunities

- T001 and T002 can run in parallel (Phase 1)
- T004 and T006 can run in parallel with T003 (Phase 2)
- T009, T010, T011, T012, T013 can all run in parallel (Phase 3 — different files)
- US2 and US3 can run in parallel (different files, no shared state)

---

## Parallel Example: Phase 3 (US1) Doc Embeds

```bash
# All doc page embeds can be done simultaneously:
Task: "Embed screenshots in quickstart-ui.md"   # T008
Task: "Embed screenshots in sessions.md"         # T009
Task: "Embed screenshots in integrations.md"     # T010
Task: "Embed screenshots in workspaces.md"       # T011
Task: "Embed screenshots in scheduled-sessions.md" # T012
Task: "Embed screenshots in context-and-artifacts.md" # T013
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Complete Phase 1 + 2: Setup + Foundational
2. Complete Phase 3: Embed screenshots + capture locally
3. **STOP and VALIDATE**: Theme switching works in docs dev server
4. Demo to stakeholders

### Incremental Delivery

1. Setup + Foundational → Cypress spec works
2. US1 → Screenshots in docs, theme switching works (MVP)
3. US2 → Daily automation keeps screenshots fresh
4. US3 → Two-tier hosting separates stable from current
5. Polish → Final validation

---

## Metrics

- **Total tasks**: 27
- **Phase 1 (Setup)**: 2
- **Phase 2 (Foundational)**: 5
- **Phase 3 (US1)**: 10
- **Phase 4 (US2)**: 2
- **Phase 5 (US3)**: 5
- **Phase 6 (Polish)**: 3
- **Parallel opportunities**: 12 tasks marked [P] or parallelizable within phase
- **MVP scope**: Phases 1-3 (17 tasks)
