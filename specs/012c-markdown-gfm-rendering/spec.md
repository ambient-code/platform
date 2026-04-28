# Feature Specification: Complete GFM Element Support

**Feature Branch**: `feat/markdown-gfm-elements`
**Created**: 2026-04-28
**Status**: Draft
**Input**: Visual styling requirements for blockquotes, horizontal rules, images, and extended heading levels in bot messages. Depends on spec 012a being merged first.

## Overview

Bot messages containing blockquotes, horizontal rules, images, or extended heading levels render with appropriate visual treatment consistent with the rest of the message design:

- **`blockquote`** (P1): Themed left border (`border-l-4 border-border`) with indented italic text in `text-muted-foreground`, making quoted content immediately distinguishable from prose. Elevated to P1 because AI responses use `>` quoting frequently — citations, emphasis, user input echoes.
- **`hr`** (P2): Full-width line in the theme's border color (`border-t border-border`) with vertical margin, providing visual section separation in dark and light mode.
- **`img`** (P2): Constrained to container width (`max-w-full`) with rounded corners, preventing layout overflow regardless of source image dimensions.
- **`h4`/`h5`/`h6`** (P3): Follow the established heading scale — progressively smaller than H3 (`text-sm font-medium`). May be cut if the team confirms LLM output does not generate these levels.

All overrides apply identically to `tool-message.tsx` via the shared `sharedMarkdownComponents` constant from spec 012b. After this spec, `sharedMarkdownComponents` covers all common GFM elements: tables, blockquote, hr, img, and h4–h6.

**Prerequisite**: Spec 012a must be merged. The `font-mono` and `inline` wrapper fixes in 012a are required for blockquotes and other block-level elements to render in proper block flow with correct typography.

**Parallel development risk**: Both this spec and spec 012b add entries to `src/lib/markdown-components.ts` and export `sharedMarkdownComponents`. If 012b and 012c are developed on concurrent branches, the shared file is a merge conflict surface. The recommended sequence is to merge 012b first and rebase this branch on top. If parallel development is unavoidable, see tasks.md for the conflict resolution protocol.

---

## User Scenarios & Testing

### User Story 1 — Blockquotes Are Visually Distinct (Priority: P1)

A user receives a bot response that quotes a user's earlier message or cites a source using `>`. The quoted text is immediately recognizable as quoted content, distinct from prose.

**Why this priority**: Blockquotes appear in nearly every multi-turn conversation where the agent references prior context. Without styling, quoted text is indistinguishable from regular prose — users lose the attribution signal entirely.

**Independent Test**: Send a bot message with `> quoted text`. Verify a left border and indentation appear.

**Acceptance Scenarios**:

1. **Given** a blockquote (`> text`) in a bot message, **When** rendered, **Then** the `<blockquote>` element has a `border-l-4` left border using `border-border` color and `pl-4` left padding
2. **Given** a blockquote, **When** rendered, **Then** the blockquote text uses `text-muted-foreground` color and `italic` style, visually lighter than surrounding prose
3. **Given** a multi-line blockquote (`> line 1\n> line 2`), **When** rendered, **Then** both lines are contained within the same styled blockquote block with consistent border and padding
4. **Given** a blockquote in dark mode, **When** the theme is toggled, **Then** the border uses `border-border` (theme-aware) and remains visible

---

### User Story 2 — Horizontal Rules Separate Content (Priority: P2)

A user receives a response using `---` to separate sections. The separator renders as a full-width horizontal line in the theme's border color.

**Why this priority**: Section separators are a common document structure element. Browser-default `<hr>` may render with incorrect color in dark mode.

**Independent Test**: Send a bot message containing `---`. Verify a horizontal line appears across the message width.

**Acceptance Scenarios**:

1. **Given** a horizontal rule (`---`) in a bot message, **When** rendered, **Then** an `<hr>` element appears with `border-t border-border my-4` — full-width, theme-colored, with vertical margin
2. **Given** a horizontal rule in dark mode, **When** the theme is toggled, **Then** the border color resolves to the dark-mode `--border` value (no hardcoded color)

---

### User Story 3 — Images Do Not Break Layout (Priority: P2)

A bot message contains a markdown image (`![alt](url)`). The image renders within the message container without causing horizontal overflow or covering adjacent content.

**Why this priority**: Without `max-width`, a large image expands beyond the message container and breaks the session layout. This is a layout-safety requirement, not an aesthetic one.

**Independent Test**: Render a bot message with an image URL pointing to a 2000px-wide image. Verify the image does not cause horizontal overflow.

**Acceptance Scenarios**:

1. **Given** a bot message containing `![alt](url)` where the image is wider than the container, **When** rendered, **Then** the `<img>` element has `max-w-full` and `rounded` classes and does not cause `scrollWidth > clientWidth` on the page
2. **Given** a broken image URL, **When** rendered, **Then** the `alt` text is visible and the element does not stretch beyond its container

---

### User Story 4 — H4–H6 Headings Follow the Size Scale (Priority: P3)

A bot message uses four or more heading levels. H4–H6 are progressively smaller than H3 and larger than body text.

**Why this priority**: Lower heading levels are rarely generated by LLMs. This is a completeness item. If the team determines H4–H6 never appear in practice, this story may be cut without affecting P1 or P2 deliverables.

**Independent Test**: Send a message with `####`, `#####`, `######`. Verify each is smaller than the one above and larger than `text-sm` body text.

**Acceptance Scenarios**:

1. **Given** an H4 heading, **When** rendered, **Then** the element has `text-xs font-medium text-foreground mb-1` — visually smaller than H3 (`text-sm font-medium`) and larger than `text-xs text-muted-foreground` body text by weight contrast
2. **Given** H4, H5, H6 in the same message, **When** rendered, **Then** each level's `font-size` (computed) is less than or equal to the level above it

---

### Edge Cases

- A blockquote containing a nested blockquote (`>> nested`) must render with correct nesting (the inner blockquote gets its own border-l-4 and pl-4 inside the outer one).
- A blockquote containing inline code must not apply `font-mono` to the blockquote prose — only to the `<code>` element within it.
- An `<img>` rendered inside a table cell must still obey `max-w-full` to avoid breaking the cell.
- H4–H6 immediately following H1–H3 must have consistent `mb-*` spacing so the heading hierarchy feels uniform.
- A `---` rule following a paragraph with `mb-2` must not produce excessive whitespace — the `my-4` on `<hr>` accounts for both top and bottom gap.

---

## Requirements

### Functional Requirements

- **FR-001**: `defaultComponents` in `message.tsx` MUST add a `blockquote` override: `<blockquote className="border-l-4 border-border pl-4 py-1 my-2 italic text-muted-foreground">{children}</blockquote>`.
- **FR-002**: `defaultComponents` MUST add an `hr` override: `<hr className="border-t border-border my-4" />`.
- **FR-003**: `defaultComponents` MUST add an `img` override: `<img className="max-w-full rounded" src={src ?? ''} alt={alt ?? ''} />`. The `??` fallback is required because react-markdown types `src` and `alt` as `string | undefined`; omitting the fallback produces a TypeScript error. The override MUST pass through `src` and `alt` props from the markdown source.
- **FR-004**: `defaultComponents` MUST add `h4`, `h5`, and `h6` overrides following the existing h1–h3 scale:
  - `h4`: `<h4 className="text-xs font-medium text-foreground mb-1">{children}</h4>`
  - `h5`: `<h5 className="text-xs font-normal text-foreground mb-1">{children}</h5>`
  - `h6`: `<h6 className="text-xs font-normal text-muted-foreground mb-1">{children}</h6>`
- **FR-005**: The identical `blockquote`, `hr`, `img`, and `h4`/`h5`/`h6` overrides MUST be applied to `tool-message.tsx`'s `markdownComponents`. These SHOULD be extracted into the shared `markdown-components.ts` constant introduced in spec 012b (FR-007) to avoid duplication.
- **FR-006**: All colors MUST use Tailwind utilities referencing theme variables (`border-border`, `text-muted-foreground`, `text-foreground`) — no hardcoded hex or `rgba` values.

### Key Entities

- **`defaultComponents`** (`message.tsx:38`): Receives all new element overrides in this spec.
- **`markdownComponents`** (`tool-message.tsx:66`): Receives the same overrides via the shared constant from 012b.
- **`markdown-components.ts`** (shared constant from spec 012b): The canonical source for all shared component overrides; extended in this spec with blockquote, hr, img, h4–h6.

---

## Success Criteria

### Measurable Outcomes

- **SC-001**: A bot message with `> blockquote` renders with a visible left border and indented italic text in both light and dark mode (manual visual test; Cypress: `cy.get('blockquote').should('have.css', 'border-left-width', '4px')`)
- **SC-002**: A bot message with `---` renders a full-width `<hr>` element with no hardcoded color (inspect computed `border-color` resolves to the theme's `--border` value)
- **SC-003**: A bot message with a 2000px-wide image does not cause horizontal page scroll (`document.documentElement.scrollWidth === document.documentElement.clientWidth`)
- **SC-004**: A bot message with `####` renders an element with `font-size` smaller than the H3 override's `text-sm` (14px) — verifiable with `cy.get('h4').invoke('css', 'font-size').then(parseFloat).should('be.lte', 14)` — or the story is cut if team confirms LLMs never generate H4–H6
- **SC-005**: `cd components/frontend && npx vitest run` passes with no new failures
