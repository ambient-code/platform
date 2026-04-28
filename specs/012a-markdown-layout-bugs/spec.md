# Feature Specification: Fix Core Markdown Layout Bugs

**Feature Branch**: `feat/markdown-layout-bugs`
**Created**: 2026-04-28
**Status**: Draft
**Input**: Desired prose rendering for bot messages: block layout, proportional typography, paragraph spacing, and inline element styling. Prerequisite for specs 012b and 012c.

## Overview

Bot message prose renders in the application's proportional sans-serif font with clear vertical rhythm. All block-level elements — paragraphs, headings, lists, tables — stack vertically in normal document flow and occupy the full message container width. Paragraph spacing is visually distinct. Bold, italic, and strikethrough text carry the correct weight, style, and `text-foreground` color. Inline code is the only monospace element in a message.

The animated streaming cursor appears at the text baseline of the last rendered character, staying on the same line via `inline-block` and `align-middle`.

Explicit component overrides for `<strong>`, `<em>`, and `<del>` ensure these elements receive the correct Tailwind color utilities rather than relying on browser defaults. List item gap (`space-y-1.5`) is harmonized with paragraph spacing (`mb-2`) for consistent vertical rhythm throughout a message.

This spec is prerequisite for specs 012b (table rendering) and 012c (GFM elements) — block-level elements require vertical document flow to render correctly.

---

## User Scenarios & Testing

### Block Layout for ReactMarkdown Output

The `<ReactMarkdown>` output wrapper uses block layout (`w-full`) so that block-level elements — paragraphs, tables, lists, headings — stack vertically and fill the container. This is the structural foundation that all other markdown rendering (tables, blockquotes, list indentation) depends on.

The streaming cursor `<span>` uses `inline-block` and `align-middle`, which positions it at the text baseline regardless of the parent wrapper's display type — no special handling is required.

**Acceptance**: Given any bot message with multiple block-level markdown elements, when rendered, the elements stack vertically, table rows are horizontal, and the streaming cursor appears at the text baseline of the last rendered character.

---

### User Story 1 — Prose Text Is Readable (Priority: P1)

A user reads a multi-paragraph bot response. Paragraphs are clearly separated. Headings, bold, and italic text render in sans-serif, not monospace. Inline code is the only monospace element.

**Why this priority**: The `font-mono` wrapper affects every bot message. Its removal is the highest-impact single-line change in this file.

**Independent Test**: Send a message with two paragraphs, `**bold**`, `*italic*`, and `` `code` ``. Verify paragraphs are in a proportional font, bold is heavier, italic is slanted, only inline code is monospace.

**Acceptance Scenarios**:

1. **Given** a bot message with body text, **When** rendered, **Then** the message content wrapper (`div` at line 314) does NOT carry the `font-mono` CSS class; `font-family` resolves to the project's sans-serif stack (Inter / system-ui)
2. **Given** `**bold**` in a bot message, **When** rendered, **Then** the `<strong>` element has `font-weight: 600` or higher and `font-family` resolves to sans-serif
3. **Given** `*italic*` in a bot message, **When** rendered, **Then** the `<em>` element has `font-style: italic` and `font-family` resolves to sans-serif
4. **Given** inline `` `code` `` in a bot message, **When** rendered, **Then** the `<code>` element has `font-family` resolving to a monospace stack — the only element in the message that does

---

### User Story 2 — Paragraphs Have Readable Spacing (Priority: P1)

A user reads a bot message with three or more paragraphs. Each paragraph is visually distinct from the next with clear vertical separation.

**Why this priority**: `mb-[0.2rem]` (3.2px) affects every multi-paragraph response. At `text-sm` this is imperceptible.

**Independent Test**: Send a message with three short paragraphs. Verify each paragraph's bottom margin is at least 8px (Tailwind `mb-2`).

**Acceptance Scenarios**:

1. **Given** a bot message with 3+ paragraphs, **When** rendered, **Then** each `<p>` (rendered as a `<div>` by the component override) has `margin-bottom` of exactly `0.5rem` (8px, `mb-2`)
2. **Given** a bot message with 3+ paragraphs in a scroll container with `max-h-96`, **When** rendered, **Then** the container scrolls correctly and does not clip the last paragraph's bottom margin
3. **Given** the session messages list (`MessagesTab.tsx`) with 20+ messages already rendered, **When** a new bot message arrives with the corrected paragraph spacing, **Then** the viewport position of previously rendered messages is unaffected — the layout change to the new message does not cause a visible scroll jump in content above it

---

### User Story 3 — Bold, Italic, and Strikethrough Render Correctly (Priority: P1)

A user receives a message using `**bold**`, `*italic*`, and `~~strikethrough~~`. All three render with correct visual style in sans-serif.

**Why this priority**: These are the most common inline formatting elements. They were broken by the `font-mono` wrapper — once the wrapper is fixed, explicit overrides ensure the correct color and weight.

**Independent Test**: Send a message with all three inline formats. Verify each is visually distinct from plain text.

**Acceptance Scenarios**:

1. **Given** `**bold text**` in a bot message, **When** rendered, **Then** the `<strong>` element uses `font-semibold` (`font-weight: 600`) and `text-foreground` color
2. **Given** `*italic text*` in a bot message, **When** rendered, **Then** the `<em>` element uses `italic` font style and `text-foreground` color
3. **Given** `~~strikethrough~~` in a bot message, **When** rendered, **Then** the `<del>` element has `text-decoration: line-through` and `opacity: 0.7` (`opacity-70`)

---

### Edge Cases

- When a bot message contains only a code block (no paragraphs), the `font-mono` removal must not affect code rendering — the `code` override already sets `font-mono` explicitly.
- When the message is actively streaming and the inline wrapper changes to block, the animated cursor `<span>` must appear at the baseline of the last rendered character on the same line (not on a new line below). Verify by streaming a multi-word sentence and confirming cursor stays inline.
- When a message contains nested formatting (`**bold with _italic_ inside**`), both overrides must compose correctly.
- When a scroll container loads earlier messages above the current viewport, paragraph spacing increase must not cause visible scroll jump (MessagesTab scroll-preservation logic).

---

## Requirements

### Functional Requirements

- **FR-001**: `message.tsx` line 321 MUST change `className="inline"` to `className="w-full"`. The streaming cursor `<span>` at line 329 MUST NOT be modified — it already positions correctly with `inline-block` and `align-middle`.
- **FR-002**: `message.tsx` line 314 MUST remove `font-mono` from the class list. The resulting class string must be `"text-sm text-foreground"` (plus the existing conditional `!isBot && "py-2 px-4"`). `font-mono` must remain on the `code` component override (lines 57 and 68) and nowhere else in the message content wrapper tree.
- **FR-003**: The `p` component override MUST change `mb-[0.2rem]` to `mb-2`. The `leading-relaxed` class and `text-muted-foreground` color MUST be retained.
- **FR-004**: `defaultComponents` MUST include a `strong` override: `<strong className="font-semibold text-foreground">{children}</strong>`.
- **FR-005**: `defaultComponents` MUST include an `em` override: `<em className="italic text-foreground">{children}</em>`.
- **FR-006**: `defaultComponents` MUST include a `del` override: `<del className="line-through opacity-70">{children}</del>`.
- **FR-007**: The `ul` and `ol` overrides MUST change `space-y-1` to `space-y-1.5` to harmonize list item gap with corrected paragraph spacing.
- **FR-008**: All Tailwind classes using theme colors MUST reference Tailwind utilities (`bg-muted`, `text-muted-foreground`, `text-foreground`, `border-border`) — not raw CSS variable names.

### Key Entities

- **`defaultComponents`** (`message.tsx:38`): The `Components` map passed to `<ReactMarkdown>`; primary target for all inline-element overrides in this spec.
- **Content wrapper div** (`message.tsx:314`): The outer `<div>` applying `font-mono` and `text-sm` to all markdown output. FR-002 targets this line.
- **ReactMarkdown wrapper div** (`message.tsx:321`): The `<div className="inline">` immediately wrapping `<ReactMarkdown>`. FR-001 targets this line.
- **Streaming cursor `<span>`** (`message.tsx:329`): The blinking cursor rendered during streaming. Not modified by this spec.

---

## Success Criteria

### Measurable Outcomes

- **SC-001**: The content wrapper div at line 314 does not contain `font-mono` in its `className` (grep-verifiable)
- **SC-002**: The ReactMarkdown wrapper div at line 321 has `className="w-full"` (grep-verifiable)
- **SC-003**: `defaultComponents` contains entries for `strong`, `em`, and `del` (grep-verifiable)
- **SC-004**: The `p` override contains `mb-2` not `mb-[0.2rem]` (grep-verifiable)
- **SC-005**: A streaming bot message with paragraph text shows the animated cursor on the same line as the last rendered character, not on a new line below it (manual visual test during streaming)
- **SC-006**: `cd components/frontend && npx vitest run` passes with no new failures
