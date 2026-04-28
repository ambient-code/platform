# Implementation Plan: Markdown Table Rendering

**Branch**: `feat/markdown-tables` | **Date**: 2026-04-28 | **Spec**: [spec.md](spec.md)
**Depends on**: `feat/markdown-layout-bugs` (spec 012a) merged to main

## Summary

Add six table component overrides to `message.tsx` and share them with `tool-message.tsx` via a new shared constant file. Move `markdownComponents` in `tool-message.tsx` from inside the `ExpandableMarkdown` function body to module level. No new npm dependencies; uses existing Tailwind utilities and react-markdown API.

## Technical Context

**Language**: TypeScript/React
**Target files**: `components/frontend/src/components/ui/message.tsx`, `tool-message.tsx`, new `markdown-components.ts`
**Testing**: vitest + manual visual verification
**Risk**: Low — new component overrides do not affect existing elements; `prose-sm` on ExpandableMarkdown may conflict (verify with devtools)

## Files

```
components/frontend/src/
├── lib/
│   └── markdown-components.ts    # NEW: exports sharedMarkdownComponents (table overrides; extended by 012c)
└── components/ui/
    ├── message.tsx               # MODIFY: import and spread sharedMarkdownComponents into defaultComponents
    └── tool-message.tsx          # MODIFY: move markdownComponents to module level; spread sharedMarkdownComponents
```

## Architecture Note

Extracting table overrides into `src/lib/markdown-components.ts` (not `src/components/ui/`) keeps library-style shared utilities separate from UI components. The exported constant is named `sharedMarkdownComponents` — not `tableComponents` — because spec 012c adds blockquote, hr, img, and h4–h6 overrides to the same constant; a table-specific name would be misleading. Both `message.tsx` and `tool-message.tsx` spread the shared map into their local component definitions, preserving any component-specific overrides.

`markdownComponents` in `tool-message.tsx` is currently defined inside the `ExpandableMarkdown` function body (lines 66–93). It must be lifted to module level so it can be defined once and reference the imported `sharedMarkdownComponents` without re-instantiating the object on every render.
