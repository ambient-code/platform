# Session Options Menu Infrastructure

**Branch**: `feat/advanced-options-menu`
**Date**: 2026-04-16
**Status**: Draft

## Overview

Restructure the `+` dropdown in `new-session-view.tsx` to serve as the unified entry point for all per-session configuration. Add a separator dividing context actions (Add Repository, Upload File) from session options (toggles, form-heavy config). Add a discovery dot on the `+` button to highlight new menu items users haven't seen.

This PR adds no new menu items — it establishes the infrastructure that PR #1328 (project intelligence toggle) and PR #1326 (SDK options modal) will use.

## Menu Structure

```
[+] button (discovery dot when new items exist)
├── Add Repository          → existing AddContextModal
├── Upload File             → existing (disabled until PR #1282 merges)
├── DropdownMenuSeparator
└── (future items added here by other PRs)
```

## Patterns Established

### Boolean toggles
Use `DropdownMenuCheckboxItem`. State lives in `new-session-view.tsx` as `useState`. Example (PR #1328 will add):
```tsx
<DropdownMenuCheckboxItem
  checked={disableIntelligence}
  onCheckedChange={(checked) => setDisableIntelligence(checked === true)}
>
  Disable project intelligence
</DropdownMenuCheckboxItem>
```

### Form-heavy config
Use `DropdownMenuItem` with `onClick` that opens a `Dialog`. The dialog contains the full form with save/preview flow. Example (PR #1326 will add):
```tsx
<DropdownMenuItem onClick={() => setSdkOptionsOpen(true)}>
  SDK Options...
</DropdownMenuItem>
```

### New item badges
Menu items can render a `Badge` with "New" to indicate features the user hasn't interacted with yet. Tracked client-side.

### Discovery dot
A small circle indicator (`h-2 w-2 rounded-full bg-primary`) positioned on the top-right of the `+` button. Visible when the menu contains items newer than the user's last-seen version. Cleared when the dropdown opens. Tracked via `localStorage` key `acp-menu-seen-version` compared against a constant `MENU_VERSION` string.

## Components Changed

- `new-session-view.tsx` — add separator, discovery dot logic, import new dropdown primitives (`DropdownMenuSeparator`, `DropdownMenuCheckboxItem`)

## What This PR Does NOT Include

- No new menu items (those come from #1328 and #1326)
- No backend changes
- No new feature flags
- No modals or dialogs (those come with the items that need them)
