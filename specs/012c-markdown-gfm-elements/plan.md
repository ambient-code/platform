# Implementation Plan: Complete GFM Element Support

**Branch**: `feat/markdown-gfm-elements` | **Date**: 2026-04-28 | **Spec**: [spec.md](spec.md)
**Depends on**: `feat/markdown-layout-bugs` (spec 012a) merged to main
**Extends**: `markdown-components.ts` introduced in spec 012b (if merged; otherwise create it here)

## Summary

Add four element overrides to the shared `markdown-components.ts` constant: `blockquote`, `hr`, `img`, and `h4`/`h5`/`h6`. Both `message.tsx` and `tool-message.tsx` inherit them automatically through the spread pattern introduced in 012b. No new npm dependencies.

## Technical Context

**Language**: TypeScript/React
**Target files**: `components/frontend/src/components/ui/markdown-components.ts` (primary), `message.tsx` and `tool-message.tsx` only if 012b is not yet merged
**Testing**: vitest + manual visual verification
**Risk**: Low — all additions are new component overrides that do not interact with existing overrides

## Files

```
components/frontend/src/components/ui/
└── markdown-components.ts    # MODIFY: add blockquote, hr, img, h4, h5, h6 overrides
```

If spec 012b has not been merged when this branch is cut, `markdown-components.ts` must be created here and wired into both `message.tsx` and `tool-message.tsx` following the same pattern described in 012b's plan.

## H4–H6 Decision Gate

Before implementing T020–T022, confirm with the team whether LLM output in this product ever generates H4–H6 headings. If not, skip those tasks and note the decision in the PR description. The spec marks this story P3 and explicitly allows it to be cut.
