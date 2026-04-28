# Tasks: Complete GFM Element Support

**Input**: Design documents from `/specs/012c-markdown-gfm-elements/`
**Prerequisite**: Spec 012a (`feat/markdown-layout-bugs`) merged to main and this branch rebased on it
**Coordination with 012b**: Both 012b and 012c write to `components/frontend/src/lib/markdown-components.ts` and export `sharedMarkdownComponents`.

- **Option A (preferred)**: Merge spec 012b to main first, then rebase this branch. `markdown-components.ts` already exists with the `sharedMarkdownComponents` export — add the new GFM element overrides directly to the same object. No conflict possible.
- **Option B (parallel development)**: Create `markdown-components.ts` at `src/lib/markdown-components.ts` with only the 012c overrides under the same export name. When 012b merges first and you rebase: git will report a conflict in `markdown-components.ts`. Resolve by merging BOTH sets of overrides into the single `sharedMarkdownComponents` object — keep all table entries from 012b AND all GFM element entries from 012c. Do NOT discard either set.

## Phase 1: Blockquote (P1)

- [ ] T001 [FR-001] Add `blockquote` to the shared component map in `markdown-components.ts`:
  `blockquote: ({ children }) => <blockquote className="border-l-4 border-border pl-4 py-1 my-2 italic text-muted-foreground">{children}</blockquote>`
- [ ] T002 Manual visual test: bot message with `> quoted text` — confirm left border, italic, muted color in light and dark mode

### Commit: `feat(frontend): add blockquote override to markdown renderer`

---

## Phase 2: Horizontal Rule and Image (P2)

- [ ] T010 [FR-002] Add `hr` to the shared component map:
  `hr: () => <hr className="border-t border-border my-4" />`
- [ ] T011 [FR-003] Add `img` to the shared component map:
  `img: ({ src, alt }) => <img className="max-w-full rounded" src={src ?? ''} alt={alt ?? ''} />`
- [ ] T012 Manual visual test: bot message with `---` — confirm full-width themed separator
- [ ] T013 Manual visual test: bot message with a wide image — confirm no horizontal page scroll

### Commit: `feat(frontend): add hr and img overrides to markdown renderer`

---

## Phase 3: H4–H6 Headings (P3 — confirm before implementing)

- [ ] T020 [Decision] Confirm with team: do LLM responses in this product ever generate H4–H6? If NO, skip T021–T023 and document the decision in the PR.
- [ ] T021 [FR-004] Add `h4` override: `h4: ({ children }) => <h4 className="text-xs font-medium text-foreground mb-1">{children}</h4>`
- [ ] T022 [FR-004] Add `h5` override: `h5: ({ children }) => <h5 className="text-xs font-normal text-foreground mb-1">{children}</h5>`
- [ ] T023 [FR-004] Add `h6` override: `h6: ({ children }) => <h6 className="text-xs font-normal text-muted-foreground mb-1">{children}</h6>`

### Commit: `feat(frontend): add h4-h6 overrides to markdown renderer`

---

## Phase 4: Verify

- [ ] T030 Run `cd components/frontend && npx vitest run` — confirm no failures
- [ ] T031 Confirm `tool-message.tsx` tables, blockquotes, and separators pick up the new overrides automatically (they spread `markdown-components.ts` — no file change needed if 012b is merged)
- [ ] T032 Grep for `any` types in changed files: `grep -n ': any' components/frontend/src/lib/markdown-components.ts`

### Commit (if lint fixes needed): `chore: lint fixes`

---

## Dependencies

- Phase 1 (blockquote) → independent; start here
- Phase 2 (hr, img) → independent of Phase 1
- Phase 3 (h4–h6) → independent; gated on team decision
- Phase 4 → depends on Phases 1–3
