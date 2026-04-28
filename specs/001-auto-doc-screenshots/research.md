# Research: Automated Documentation Screenshots

## Decision 1: Theme Switching in Starlight Docs

**Decision**: Use CSS `data-theme` selectors with paired `<img>` elements in plain markdown.

**Rationale**: Starlight uses `data-theme="light|dark"` on `<html>`. CSS rules can show/hide images based on this attribute. This works in standard `.md` files without MDX conversion. Zero JS overhead — pure CSS display toggling.

**Alternatives considered**:
- MDX components with conditional rendering — requires converting all `.md` to `.mdx`, disruptive
- `<picture>` with `prefers-color-scheme` media queries — doesn't track Starlight's manual theme override, only OS preference
- CSS filter (invert) on single images — produces poor quality for UI screenshots

## Decision 2: Screenshot Architecture

**Decision**: Single Cypress spec file driven by JSON manifest.

**Rationale**: Screenshots are captures, not tests. A manifest decouples "what to capture" from "how to capture." Adding a screenshot is a JSON entry + HTML snippet, not Cypress code. The existing E2E suite uses a single large spec (`sessions.cy.ts`, 2134 lines) — one file is the project convention.

**Alternatives considered**:
- Per-page Cypress spec files — duplicates setup/teardown, harder to maintain
- Playwright-based (already a docs dep) — would duplicate the test harness; Cypress E2E infra is proven
- Storybook + Chromatic — requires adding Storybook, overkill for full-page screenshots

## Decision 3: Image Path Strategy (Dual Hosting)

**Decision**: Use `/platform/` prefix in all image paths. Netlify redirects `/platform/*` to `/:splat`.

**Rationale**: GitHub Pages serves at `/platform/` base. Netlify serves at `/`. Using a single path prefix with a Netlify redirect avoids maintaining two sets of paths. The `astro.config.mjs` detects `process.env.NETLIFY` to set the correct `site`/`base` for link generation.

**Alternatives considered**:
- Relative paths — fragile, different nesting depths per page
- Astro `<Image>` components — requires MDX
- Two separate image directories — double storage, sync nightmares

## Decision 4: CI Cadence

**Decision**: Daily cron (6AM UTC) with `workflow_dispatch` for on-demand runs. Auto-PR on change, no direct commits to main.

**Rationale**: UI changes are frequent enough that weekly misses drift. Daily catches changes promptly. The PR-based flow ensures human review of visual changes. The kind cluster spin-up cost (~10 min) is acceptable for a daily run.

**Alternatives considered**:
- Weekly — misses UI changes for too long
- On every push to main — wasteful, most pushes don't change UI
- Direct commit to main — no human review of visual changes

## Decision 5: DPI Normalization

**Decision**: Force 1x device scale factor via Chrome launch arg when `CYPRESS_SCREENSHOT_MODE` is set.

**Rationale**: macOS Retina displays capture at 2x by default, producing images twice the size of CI (Linux 1x). Forcing 1x ensures identical output across platforms.

**Alternatives considered**:
- Post-process resize — adds complexity, lossy
- Accept 2x everywhere — doubles file sizes unnecessarily
- Only capture in CI — blocks local iteration
