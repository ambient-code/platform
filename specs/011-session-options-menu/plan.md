# Implementation Plan: Session Options Menu Infrastructure

**Branch**: `011-session-options-menu` | **Date**: 2026-04-16 | **Spec**: [spec.md](spec.md)

## Summary

Add a separator and discovery dot to the `+` dropdown in `new-session-view.tsx`. Frontend-only change. One file modified, one test file updated.

## Technical Context

**Language**: TypeScript/React (Next.js 14)
**Dependencies**: Shadcn/ui (`DropdownMenuSeparator`, `DropdownMenuCheckboxItem`), localStorage
**Testing**: vitest
**Target**: `components/frontend/src/app/projects/[name]/sessions/[sessionName]/components/new-session-view.tsx`

## Files

```
components/frontend/src/app/projects/[name]/sessions/[sessionName]/components/
├── new-session-view.tsx                    # MODIFY: add separator, discovery dot, imports
└── __tests__/new-session-view.test.tsx     # MODIFY: add tests for separator and dot
```
