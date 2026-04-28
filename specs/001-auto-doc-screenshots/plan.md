# Implementation Plan: Automated Documentation Screenshots

**Branch**: `001-auto-doc-screenshots` | **Date**: 2026-04-15 | **Spec**: [spec.md](spec.md)

## Summary

Build a manifest-driven Cypress screenshot pipeline that captures the web UI in both light and dark themes against a kind cluster, embeds paired PNGs in Astro Starlight docs with CSS-based theme auto-switching, and automates the flow via a daily GHA workflow that opens PRs when screenshots change. Split docs hosting: Netlify deploys main continuously, GitHub Pages deploys release tags only.

## Technical Context

**Language/Version**: TypeScript (Cypress 15), CSS, YAML (GHA), Bash (Makefile), Astro/Starlight 0.34
**Primary Dependencies**: Cypress 15, Astro Starlight 0.34, next-themes (existing frontend), kind 0.27
**Storage**: PNG files in `docs/public/images/screenshots/` committed to git (~5-10MB total)
**Testing**: Cypress spec validation against kind cluster in mock SDK mode
**Target Platform**: GitHub Actions (Ubuntu), local macOS (kind + Docker)
**Project Type**: Tooling/CI — no new services, APIs, or CRDs
**Performance Goals**: Daily CI workflow completes within 30 minutes
**Constraints**: Screenshots must be 1x DPI for cross-platform consistency; no MDX conversion; no system-level package installs on Netlify
**Scale/Scope**: 10 screenshot entries (20 PNGs), 6 docs pages modified

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Applicable? | Status | Notes |
|-----------|-------------|--------|-------|
| I. K8s-Native | No | N/A | No new CRDs, operators, or K8s resources created |
| II. Security | Minimal | PASS | No secrets, tokens, or auth changes. GHA uses `GITHUB_TOKEN` (default) |
| III. Type Safety | Yes | PASS | Cypress spec uses typed interfaces; no `any` types |
| IV. TDD | Partial | PASS | Screenshot spec IS the test — it validates screenshot capture works. No business logic to unit test |
| V. Modularity | Yes | PASS | Single Cypress spec file (~150 lines), single manifest JSON, single CSS block |
| VI. Observability | No | N/A | No new services or endpoints |
| VII. Resource Lifecycle | No | N/A | No K8s child resources |
| VIII. Context Engineering | No | N/A | No AI prompts or agent changes |
| IX. Data Access | No | N/A | No RAG/MCP/RLHF changes |
| X. Commit Discipline | Yes | PASS | Plan targets ~7 focused commits, each under 300 lines |

No constitution violations. No complexity justifications required.

## Project Structure

### Documentation (this feature)

```text
specs/001-auto-doc-screenshots/
├── spec.md
├── plan.md              # This file
├── research.md          # Phase 0
└── checklists/
    └── requirements.md
```

### Source Code (repository root)

```text
# New files
e2e/cypress/e2e/screenshots.cy.ts          # Manifest-driven capture spec
e2e/cypress/screenshots/manifest.json       # Screenshot targets
.github/workflows/screenshots.yml           # Daily cron + auto-PR
docs/netlify.toml                           # Netlify build config
docs/public/images/screenshots/             # Committed PNG outputs

# Modified files
docs/src/styles/custom.css                  # Theme-switching CSS
docs/astro.config.mjs                       # Env-aware site/base
.github/workflows/docs.yml                  # Release-tag trigger
e2e/cypress.config.ts                       # Screenshot output dir, DPI
e2e/.gitignore                              # Exclude output/
Makefile                                    # Screenshot targets
docs/src/content/docs/**/*.md               # 6 pages get screenshot embeds
```

**Structure Decision**: This feature touches existing directories only. No new top-level directories. Cypress screenshots live alongside existing E2E tests. Docs images go in the standard Starlight `public/` assets directory.
