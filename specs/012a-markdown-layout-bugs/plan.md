# Implementation Plan: Fix Core Markdown Layout Bugs

**Branch**: `feat/markdown-layout-bugs` | **Date**: 2026-04-28 | **Spec**: [spec.md](spec.md)

## Summary

Five targeted changes to `message.tsx` fixing structural rendering bugs. One file, ~10 lines changed. No new dependencies. Must ship before 012b (tables) and 012c (GFM elements).

## Technical Context

**Language**: TypeScript/React
**Target file**: `components/frontend/src/components/ui/message.tsx`
**Testing**: vitest (`npx vitest run`)
**Risk**: Medium — outer wrapper changes affect every bot message; scroll-position preservation in `MessagesTab.tsx` must be verified

## Files

```
components/frontend/src/components/ui/
└── message.tsx          # MODIFY: 5 targeted changes (lines 78, 90, 93, 314, 321, + new overrides)
```

## Change Summary

| Line | Before | After |
|------|--------|-------|
| 314 | `"text-sm text-foreground font-mono"` | `"text-sm text-foreground"` |
| 321 | `<div className="inline">` | `<div className="w-full">` |
| 78 | `mb-[0.2rem]` | `mb-2` |
| 90 | `space-y-1` (ul) | `space-y-1.5` |
| 93 | `space-y-1` (ol) | `space-y-1.5` |
| new | — | `strong`, `em`, `del` component overrides |

The streaming cursor `<span>` at line 329 is **not modified**. Its `inline-block align-middle` classes keep it at the text baseline regardless of parent display type.
