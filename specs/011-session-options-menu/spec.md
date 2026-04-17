# Feature Specification: Session Options Menu Infrastructure

**Feature Branch**: `feat/advanced-options-menu`
**Created**: 2026-04-16
**Status**: Draft
**Input**: Restructure the + dropdown as the unified entry point for per-session configuration

## Overview

The `+` button dropdown in `new-session-view.tsx` becomes the single entry point for all per-session options. A separator divides context actions (Add Repository, Upload File) from session options. A discovery dot on the button highlights new menu items. This PR adds no new items — it establishes the patterns for PRs #1326 and #1328 to use.

## User Scenarios & Testing

### User Story 1 - Menu Structure with Separator (Priority: P1)

The `+` menu currently has two items (Add Repository, Upload File). Add a separator after them to visually group future session option items below.

**Why this priority**: Foundation that all other PRs build on.

**Independent Test**: Open the `+` menu, verify the separator renders below Upload File.

**Acceptance Scenarios**:

1. **Given** a user opens the `+` menu, **When** the menu renders, **Then** Add Repository and Upload File appear above a separator line.
2. **Given** no session options are registered yet, **When** the menu renders, **Then** only the separator appears below the context actions (no empty section).

---

### User Story 2 - Discovery Dot (Priority: P1)

A small dot indicator on the `+` button when the menu contains items the user hasn't seen. Clears when they open the menu.

**Why this priority**: Feature discovery mechanism needed before new items are added.

**Independent Test**: Set localStorage to an old version, verify dot appears. Open menu, verify dot disappears.

**Acceptance Scenarios**:

1. **Given** `MENU_VERSION` is newer than the user's `acp-menu-seen-version` in localStorage, **When** the page loads, **Then** a small dot appears on the `+` button.
2. **Given** the discovery dot is visible, **When** the user opens the dropdown, **Then** the dot disappears and `acp-menu-seen-version` is updated in localStorage.
3. **Given** `acp-menu-seen-version` matches `MENU_VERSION`, **When** the page loads, **Then** no dot is shown.

---

### Edge Cases

- localStorage unavailable (private browsing) → no dot, fail silently.
- Multiple tabs → each tab checks localStorage independently; opening in one tab clears for all.

## Requirements

### Functional Requirements

- **FR-001**: `+` dropdown MUST render a `DropdownMenuSeparator` between context actions and session options.
- **FR-002**: `+` button MUST show a discovery dot when `MENU_VERSION` is newer than `localStorage.getItem("acp-menu-seen-version")`.
- **FR-003**: Discovery dot MUST clear when the dropdown opens, updating localStorage.
- **FR-004**: Discovery dot MUST fail silently if localStorage is unavailable.
- **FR-005**: New dropdown primitives (`DropdownMenuSeparator`, `DropdownMenuCheckboxItem`) MUST be imported and available for use by downstream PRs.

### Key Entities

- **MENU_VERSION**: A string constant in `new-session-view.tsx` (e.g., `"2026-04-16"`). Bumped when new menu items are added.

## Success Criteria

- **SC-001**: The separator renders visually in the `+` dropdown.
- **SC-002**: Discovery dot appears/disappears based on localStorage version comparison.
- **SC-003**: No regressions in existing menu behavior (Add Repository still works).
- **SC-004**: Frontend test suite passes, build clean.
