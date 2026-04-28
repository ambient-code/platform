# Feature Specification: Markdown Table Rendering

**Feature Branch**: `feat/markdown-tables`
**Created**: 2026-04-28
**Status**: Draft
**Input**: GFM table rendering requirements for bot messages and tool results. Depends on spec 012a being merged first.

## Overview

GFM tables in bot messages and tool results render with visible borders on all cells, padded content, a visually distinct header row, and horizontal overflow protection for wide tables. All colors use theme-aware Tailwind utilities (`border-border`, `bg-muted`, `text-foreground`, `text-muted-foreground`) so tables adapt automatically to light and dark mode without hardcoded values.

Both `message.tsx` and `tool-message.tsx` share identical table component overrides via `sharedMarkdownComponents` in `src/lib/markdown-components.ts`, ensuring consistent table rendering wherever markdown appears in the UI.

**Prerequisite**: Spec 012a must be merged. The `className="inline"` wrapper removed in 012a is what allows tables to render in block flow — without that fix, table rows collapse into inline content regardless of component overrides.

**Design note on prose vs. custom overrides**: The project uses `prose prose-sm dark:prose-invert` in `file-content-viewer.tsx` and `recent-updates-dialog.tsx`. The custom-overrides approach chosen here for `message.tsx` and `tool-message.tsx` is a pre-existing architectural inconsistency, not a new decision. This spec does not resolve that inconsistency; it extends the custom-overrides pattern already established in `message.tsx` since introducing `prose` would require removing the existing component map, re-testing all message rendering, and aligning `tool-message.tsx`'s `prose-sm` usage — scope for a separate architectural decision.

---

## User Scenarios & Testing

### User Story 1 — Tables Are Readable (Priority: P1)

A user asks the agent to summarize data. The agent responds with a GFM table. The user can distinguish columns, read cells, and identify headers.

**Why this priority**: Tables are currently the most visually broken element — completely unstyled. This is the primary trigger for this spec set.

**Independent Test**: Send a bot message containing a 3-column, 4-row GFM table. Verify borders, padding, and header background are visible.

**Acceptance Scenarios**:

1. **Given** a bot message containing a GFM table, **When** rendered in light mode, **Then** every `<td>` and `<th>` element has a `1px solid` border using the `border-border` Tailwind utility, and padding of `px-3 py-2`
2. **Given** the same table in dark mode, **When** the theme is toggled, **Then** borders and header backgrounds use the same `border-border`/`bg-muted` utilities and remain visible without hardcoded colors
3. **Given** a table's `<th>` cells, **When** rendered, **Then** headers have `bg-muted` background, `font-medium` weight, and `text-left` alignment — visually distinct from `<td>` data cells
4. **Given** a table with `<tr>` rows in `<tbody>`, **When** rendered, **Then** alternating rows do NOT require background striping (out of scope); rows are separated by cell borders alone

---

### User Story 2 — Wide Tables Do Not Break Layout (Priority: P1)

A user receives a table with 6+ columns whose total width exceeds the message container. The page does not scroll horizontally; the table scrolls independently within its container.

**Why this priority**: An unstyled wide table causes horizontal page overflow, breaking the entire session layout.

**Independent Test**: Send a bot message with an 8-column table. Verify the page body has no horizontal scrollbar; the table itself is scrollable.

**Acceptance Scenarios**:

1. **Given** a table wider than the message container, **When** rendered, **Then** the `<table>` element is wrapped in a `<div className="overflow-x-auto">` and the page body does not scroll horizontally
2. **Given** the overflow wrapper, **When** the viewport is 375px wide (mobile), **Then** the table is independently scrollable and no content is clipped

---

### User Story 3 — Tables Render Consistently in Tool Messages (Priority: P1)

The same GFM table renders identically whether it appears in a bot message or inside a tool result's `ExpandableMarkdown`.

**Why this priority**: `tool-message.tsx` uses a separate `markdownComponents` map with no table overrides, so tool result tables are completely unstyled even after message.tsx is fixed.

**Independent Test**: Render the same table markdown in a bot message and in a tool result. Compare side-by-side — borders, padding, and header styling must match.

**Acceptance Scenarios**:

1. **Given** a tool result containing a GFM table rendered via `ExpandableMarkdown`, **When** displayed, **Then** `<td>` and `<th>` elements have the same `border-border`, `px-3 py-2`, and `bg-muted` (for headers) as bot message tables
2. **Given** `tool-message.tsx` applies `prose-sm` as a class on the `ExpandableMarkdown` wrapper, **When** the custom table overrides are applied, **Then** the Tailwind component utility classes on `<th>`/`<td>` take precedence over any `prose-sm` base styles (verified by inspecting computed styles in browser devtools)

---

### Edge Cases

- A table cell containing inline code, bold text, or a link must render correctly — nested inline elements inside cells inherit the table override's `text-sm text-foreground`.
- A single-column table must render with borders on both sides of the single column.
- A table with no `<thead>` section (some GFM parsers allow this) must still render with bordered `<tbody>` cells.
- A table immediately followed by a paragraph must have `my-2` (8px vertical margin) to separate it from adjacent content.

---

## Requirements

### Functional Requirements

- **FR-001**: `defaultComponents` in `message.tsx` MUST add a `table` override: `<div className="overflow-x-auto my-2"><table className="w-full border-collapse text-sm">{children}</table></div>`. The `overflow-x-auto` wrapper is on the containing `<div>`, not the `<table>` element itself. The `<table>` element itself does NOT carry `border border-border` — with `border-collapse: collapse`, the outermost cell borders already form the table's outer edge; a separate table border is redundant.
- **FR-002**: `defaultComponents` MUST add a `thead` override: `<thead className="bg-muted">{children}</thead>`.
- **FR-003**: `defaultComponents` MUST add a `tbody` override: `<tbody>{children}</tbody>` (no additional classes required; row separation comes from cell borders).
- **FR-004**: `defaultComponents` MUST add a `tr` override: `<tr>{children}</tr>`. The `<tr>` element does NOT carry border classes — with `border-collapse: collapse`, row separation comes from the `border border-border` on `<th>` and `<td>` cells; a separate `border-b` on `<tr>` is redundant and produces double borders.
- **FR-005**: `defaultComponents` MUST add a `th` override: `<th className="px-3 py-2 text-left font-medium text-foreground border border-border">{children}</th>`.
- **FR-006**: `defaultComponents` MUST add a `td` override: `<td className="px-3 py-2 text-muted-foreground border border-border">{children}</td>`.
- **FR-007**: `tool-message.tsx`'s `markdownComponents` MUST receive the identical `table`, `thead`, `tbody`, `tr`, `th`, and `td` overrides defined in FR-001 through FR-006. The shared overrides MUST be extracted into a named export `sharedMarkdownComponents` in `src/lib/markdown-components.ts`, importable by both files to prevent drift. The name `sharedMarkdownComponents` (not `tableComponents`) is required because spec 012c adds non-table overrides to the same constant. Additionally, `markdownComponents` in `tool-message.tsx` is currently defined inside the `ExpandableMarkdown` function body — it MUST be moved to module level before the spread can import the shared constant without re-instantiating it on every render.
- **FR-008**: All border, background, and text colors MUST use Tailwind utilities that reference theme variables (`border-border`, `bg-muted`, `text-foreground`, `text-muted-foreground`) — no hardcoded colors.
- **FR-009**: The `prose-sm` class applied to `ExpandableMarkdown` wrappers in `tool-message.tsx` (lines 467, 643, 692) MUST NOT override the custom table component classes. If computed-style conflicts are found during implementation, the `markdownComponents` table overrides take precedence (add `!important` via arbitrary Tailwind values only as a last resort; prefer removing conflicting `prose-sm` classes first).

### Key Entities

- **`defaultComponents`** (`message.tsx:38`): Primary target; receives table/thead/tbody/tr/th/td overrides.
- **`markdownComponents`** (`tool-message.tsx:66`): Parallel map that must receive identical table overrides via shared constant. Currently defined inside the `ExpandableMarkdown` function body — must be moved to module level as part of this spec.
- **`sharedMarkdownComponents`** (new shared constant): Extracted component override map to be imported by both `message.tsx` and `tool-message.tsx`. Location: `components/frontend/src/lib/markdown-components.ts`. Named `sharedMarkdownComponents` (not `tableComponents`) to accommodate the non-table entries added in spec 012c.
- **`ExpandableMarkdown`** (`tool-message.tsx`): The component applying `prose-sm` wrapper class; must not conflict with table overrides.

---

## Success Criteria

### Measurable Outcomes

- **SC-001**: A 3-column, 4-row GFM table in a bot message renders with visible borders on all cells in both light and dark mode (manual visual test; Cypress: `cy.get('table td').should('have.css', 'border-width', '1px')`)
- **SC-002**: A table with 8 columns in a 1280px viewport does not cause `document.documentElement.scrollWidth > document.documentElement.clientWidth` (Cypress automatable)
- **SC-003**: The same table markdown in a bot message and in a tool result `ExpandableMarkdown` produces visually identical border, padding, and header styling (manual side-by-side comparison)
- **SC-004**: `border-collapse` is applied to the `<table>` element so adjacent cell borders do not double up (Cypress: `cy.get('table').should('have.css', 'border-collapse', 'collapse')`)
- **SC-005**: `cd components/frontend && npx vitest run` passes with no new failures
