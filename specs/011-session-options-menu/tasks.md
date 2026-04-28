# Tasks: Session Options Menu Infrastructure

**Input**: Design documents from `/specs/011-session-options-menu/`

## Phase 1: Separator + Imports

- [ ] T001 [US1] In `new-session-view.tsx`, import `DropdownMenuSeparator` and `DropdownMenuCheckboxItem` from `@/components/ui/dropdown-menu`
- [ ] T002 [US1] Add `<DropdownMenuSeparator />` after the Upload File menu item in the `+` dropdown

### Commit: `feat(frontend): add separator to + dropdown for session options`

---

## Phase 2: Discovery Dot (TDD)

- [ ] T010 [US2] Add `MENU_VERSION` constant (e.g., `"2026-04-16"`) to `new-session-view.tsx`
- [ ] T011 [US2] Add state + effect for discovery dot: compare `MENU_VERSION` against `localStorage.getItem("acp-menu-seen-version")`, set `showDot` state. Wrap in try/catch for localStorage unavailability.
- [ ] T012 [US2] On `DropdownMenu` `onOpenChange(open)`: when `open` is true, clear the dot and write `MENU_VERSION` to localStorage
- [ ] T013 [US2] Render the dot: when `showDot` is true, render a `<span className="absolute -top-0.5 -right-0.5 h-2 w-2 rounded-full bg-primary" />` inside the `+` button wrapper (make wrapper `relative`)
- [ ] T014 [US2] Add tests in `__tests__/new-session-view.test.tsx`: dot visible when localStorage version is old, dot clears on menu open, dot hidden when versions match

### Commit: `feat(frontend): add discovery dot to + button for new menu items`

---

## Phase 3: Verify

- [ ] T020 Run frontend tests: `cd components/frontend && npx vitest run`
- [ ] T021 Run frontend build: `cd components/frontend && npm run build`
- [ ] T022 Grep for `any` types in changed files

### Commit (if fixes needed): `chore: lint fixes`

---

## Dependencies

- Phase 1 → independent
- Phase 2 → independent (but logically follows Phase 1)
- Phase 3 → depends on Phases 1 + 2
