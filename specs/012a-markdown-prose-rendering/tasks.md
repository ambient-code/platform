# Tasks: Fix Core Markdown Layout Bugs

**Input**: Design documents from `/specs/012a-markdown-prose-rendering/`

## Phase 1: Structural Wrapper Fixes

- [ ] T001 [Prereq] `message.tsx:321` — change `className="inline"` to `className="w-full"`. Do NOT touch line 329 (streaming cursor).
- [ ] T002 [FR-002] `message.tsx:314` — remove `font-mono` from the class string. Result: `cn("text-sm text-foreground", !isBot && "py-2 px-4")`.

### Commit: `fix(frontend): remove inline wrapper and font-mono from message content`

---

## Phase 2: Spacing Fixes

- [ ] T010 [FR-003] `message.tsx` `p` override — change `mb-[0.2rem]` to `mb-2`
- [ ] T011 [FR-007] `message.tsx` `ul` override — change `space-y-1` to `space-y-1.5`
- [ ] T012 [FR-007] `message.tsx` `ol` override — change `space-y-1` to `space-y-1.5`

### Commit: `fix(frontend): increase paragraph and list spacing in markdown renderer`

---

## Phase 3: Inline Element Overrides

- [ ] T020 [FR-004] Add `strong` to `defaultComponents`: `strong: ({ children }) => <strong className="font-semibold text-foreground">{children}</strong>`
- [ ] T021 [FR-005] Add `em` to `defaultComponents`: `em: ({ children }) => <em className="italic text-foreground">{children}</em>`
- [ ] T022 [FR-006] Add `del` to `defaultComponents`: `del: ({ children }) => <del className="line-through opacity-70">{children}</del>`

### Commit: `feat(frontend): add strong, em, del component overrides to markdown renderer`

---

## Phase 4: Verify

- [ ] T030 Run `cd components/frontend && npx vitest run` — confirm no failures
- [ ] T031 Manually stream a bot message with `**bold** *italic* ~~strike~~` — confirm sans-serif, no monospace leak
- [ ] T032 Manually send a 3-paragraph bot message — confirm 8px margin visible between paragraphs
- [ ] T033 Visually confirm streaming cursor stays on same line as last character during active streaming
- [ ] T034 Check `MessagesTab` scroll behavior: load a session with 20+ messages, scroll to a middle message, trigger a new message — confirm viewport does not jump

### Commit (if lint fixes needed): `chore: lint fixes`

---

## Dependencies

- Phase 1 → independent (start here)
- Phase 2 → independent of Phase 1 (can be parallel)
- Phase 3 → independent of Phases 1 and 2
- Phase 4 → depends on Phases 1–3
- Spec 012b → depends on this spec being merged
- Spec 012c → depends on this spec being merged
