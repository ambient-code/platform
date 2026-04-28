# Tasks: Markdown Table Rendering

**Input**: Design documents from `/specs/012b-markdown-table-rendering/`
**Prerequisite**: Spec 012a (`feat/markdown-layout-bugs`) merged to main and this branch rebased on it

## Phase 1: Shared Component File

- [ ] T001 [FR-007] Create `components/frontend/src/lib/markdown-components.ts`
- [ ] T002 [FR-001–006] Add `sharedMarkdownComponents` export containing overrides for `table`, `thead`, `tbody`, `tr`, `th`, `td` using the exact Tailwind classes from spec FR-001 through FR-006
- [ ] T003 Verify the file has no `any` types and exports a `Components`-compatible type from `react-markdown`

### Commit: `feat(frontend): add shared markdown table component overrides`

---

## Phase 2: Wire into message.tsx

- [ ] T010 [FR-001] In `message.tsx`, import `sharedMarkdownComponents` from `../../lib/markdown-components`
- [ ] T011 Spread `sharedMarkdownComponents` into `defaultComponents`: `const defaultComponents: Components = { ...sharedMarkdownComponents, code: ..., p: ..., ... }`

### Commit: `feat(frontend): wire table components into message.tsx markdown renderer`

---

## Phase 3: Wire into tool-message.tsx

- [ ] T020 [FR-007] In `tool-message.tsx`, move the `markdownComponents` definition from inside the `ExpandableMarkdown` function body to module level (currently lines 66–93, inside the function). This is required before the shared import can work without re-instantiating on every render.
- [ ] T021 [FR-007] Import `sharedMarkdownComponents` from `../../lib/markdown-components` and spread into the now-module-level `markdownComponents`: `const markdownComponents: Components = { ...sharedMarkdownComponents, ... }`
- [ ] T022 [FR-009] Open browser devtools and inspect a rendered tool-result table: verify `th`/`td` computed styles show `border-width: 1px` and `padding-left: 12px` (px-3). Note: `prose-sm` is declared via `@plugin "@tailwindcss/typography"` in `globals.css` but NOT registered in `tailwind.config.js` plugins array — if computed styles show no `prose-sm` interference, that may be why (no action needed). If conflicts ARE observed, remove the `prose-sm` class from the relevant `ExpandableMarkdown` wrappers (lines 467, 643, 692) rather than using `!important` overrides.

### Commit: `feat(frontend): wire table components into tool-message.tsx markdown renderer`

---

## Phase 4: Verify

- [ ] T030 Run `cd components/frontend && npx vitest run` — confirm no failures
- [ ] T031 Manually send a bot message with a 3-column, 4-row GFM table in light mode — confirm borders, padding, header background visible
- [ ] T032 Toggle to dark mode — confirm borders and header remain visible (no hardcoded colors)
- [ ] T033 Send a bot message with an 8-column table — confirm no horizontal page scroll (`scrollWidth === clientWidth`)
- [ ] T034 Send a tool-use message that produces a table in the result — confirm same styling as bot message table
- [ ] T035 Check mobile viewport (375px) — confirm table scrolls horizontally within its container, page body does not scroll

### Commit (if lint fixes needed): `chore: lint fixes`

---

## Dependencies

- Phase 1 → start here (no dependencies on other phases)
- Phase 2 → depends on Phase 1
- Phase 3 → depends on Phase 1 (independent of Phase 2)
- Phase 4 → depends on Phases 1–3
- Spec 012c → can share the `markdown-components.ts` file introduced here
