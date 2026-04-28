# Feature Specification: Automated Documentation Screenshots

**Feature Branch**: `001-auto-doc-screenshots`
**Created**: 2026-04-15
**Status**: Draft

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Theme-Aware Screenshots in Docs (Priority: P1)

A new user reading the docs sees screenshots alongside each step. The screenshots match their chosen theme (light or dark) and switch instantly when they toggle the Starlight theme selector — no page reload.

**Why this priority**: Visual onboarding is the highest-impact improvement for new user success.

**Independent Test**: Open a docs page with screenshots, toggle between light and dark themes, confirm correct images display instantly.

**Acceptance Scenarios**:

1. **Given** a reader in dark mode, **When** they view a page with screenshots, **Then** they see only the dark-themed variants.
2. **Given** a reader in dark mode, **When** they toggle to light mode, **Then** all screenshots on the page switch to light variants instantly with no reload.
3. **Given** a reader on mobile, **When** they view a page with screenshots, **Then** screenshots scale responsively.

---

### User Story 2 - Daily Automated Screenshot Capture (Priority: P1)

A CI workflow runs daily, spins up a kind cluster with the latest platform images, captures screenshots of key pages in both themes using Cypress, and opens a PR if any have changed.

**Why this priority**: Without automation, screenshots go stale within weeks. Stale screenshots are worse than no screenshots.

**Independent Test**: Trigger the GHA workflow via `workflow_dispatch`, confirm it completes and creates a PR when screenshots differ.

**Acceptance Scenarios**:

1. **Given** the daily cron fires, **When** the workflow runs, **Then** it captures all manifest-defined screenshots in both themes and cleans up the cluster.
2. **Given** screenshots have changed, **When** capture completes, **Then** it opens a PR with updated PNGs.
3. **Given** no screenshots changed, **When** capture completes, **Then** no PR is created.
4. **Given** the cluster or Cypress fails, **When** the workflow errors, **Then** it uploads debug logs, cleans up, and reports failure.

---

### User Story 3 - Two-Tier Documentation Hosting (Priority: P2)

Netlify deploys docs on every push to `main` (current/live). GitHub Pages deploys only on release tags (stable). Contributors see changes live immediately; the public release site stays stable.

**Why this priority**: Separates in-progress content from the stable release site.

**Independent Test**: Push a docs change to `main` — appears on Netlify, not on GitHub Pages. Push a release tag — GitHub Pages updates.

**Acceptance Scenarios**:

1. **Given** a docs change merges to `main`, **When** Netlify rebuilds, **Then** the updated content is live with correct CSS and images.
2. **Given** a docs change merges to `main`, **When** visiting GitHub Pages, **Then** the previous release version is unchanged.
3. **Given** a `v*` tag is pushed, **When** the docs workflow triggers, **Then** GitHub Pages rebuilds from that tag.

---

### Edge Cases

- Screenshot target page URL changed: Cypress fails for that entry but captures remaining screenshots.
- Theme toggle dropdown doesn't appear: Helper throws a clear error identifying which screenshot failed.
- Netlify build fails: Build command must not require elevated privileges (no `--with-deps`).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST capture screenshots of specified pages in both light and dark themes, producing paired PNGs named `{id}-light.png` and `{id}-dark.png`.
- **FR-002**: Screenshot targets MUST be declared in a JSON manifest (page URL, wait condition, setup steps).
- **FR-003**: Documentation MUST display the correct screenshot variant based on Starlight theme, using pure CSS in standard markdown (no MDX).
- **FR-004**: The daily CI workflow MUST open a PR with updated screenshots only when changes are detected, never committing directly to `main`.
- **FR-005**: The CI workflow MUST clean up all resources (kind cluster) on completion regardless of outcome.
- **FR-006**: Netlify MUST deploy docs on every push to `main` with correct assets.
- **FR-007**: GitHub Pages MUST deploy only on `v*` tags.
- **FR-008**: Image paths MUST resolve on both hosts (Netlify at `/` and GitHub Pages at `/platform/`).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Tier 1 docs pages (Quick Start, Sessions, Integrations) contain theme-aware screenshots.
- **SC-002**: The daily CI workflow completes within 30 minutes including cluster lifecycle.
- **SC-003**: Adding a new screenshot requires modifying exactly 2 files: the manifest and the target doc page.
- **SC-004**: Netlify deploys within 5 minutes of a push to `main` with zero broken images.
- **SC-005**: GitHub Pages remains unchanged after pushes to `main`.

## Assumptions

- Existing Cypress E2E infrastructure (auth, kind scripts, mock SDK) is reusable without modification.
- Mock SDK provides sufficient UI state for meaningful screenshots.
- Starlight's `data-theme` attribute is stable across minor versions.
- Netlify site is already provisioned at https://cheerful-kitten-f556a0.netlify.app/ (jeremyeder GitHub auth). Build command: `npm ci && npx playwright install chromium && npm run build`.

## Dependencies

- Kind cluster setup scripts (`e2e/scripts/`)
- Cypress 15+ with Chrome
- Quay.io container registry for platform images
- GitHub Actions for CI
- Netlify (provisioned) and GitHub Pages (existing)
