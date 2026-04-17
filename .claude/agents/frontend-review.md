---
name: frontend-review
description: >
  Review frontend TypeScript/React code for convention violations. Use after
  modifying files under components/frontend/src/. Checks for raw HTML elements,
  manual fetch, any types, interface usage, component size, and missing states.
tools:
  - Read
  - Grep
  - Glob
  - Bash
---

# Frontend Review Agent

Review frontend code against documented conventions.

## Context

Load these files before running checks:

1. `components/frontend/DEVELOPMENT.md`
2. `components/frontend/REACT_QUERY_PATTERNS.md`
3. `components/frontend/DESIGN_GUIDELINES.md` (if it exists)

## Checks

### F1: No raw HTML elements (Critical)

```bash
grep -rn "<button\|<input\|<select\|<dialog\|<textarea" components/frontend/src/ --include="*.tsx" | grep -v "components/ui/"
```

Must use Shadcn UI components from `@/components/ui/`.

### F2: No manual fetch() in components (Critical)

```bash
grep -rn "fetch(" components/frontend/src/app/ components/frontend/src/components/ --include="*.tsx" --include="*.ts" | grep -v "src/app/api/"
```

Use React Query hooks from `@/services/queries/`.

### F3: No interface declarations (Major)

```bash
grep -rn "^export interface \|^interface " components/frontend/src/ --include="*.ts" --include="*.tsx" | grep -v "node_modules"
```

Use `type` instead of `interface`.

### F4: No any types (Critical)

```bash
grep -rn ": any\b\|as any\b\|<any>" components/frontend/src/ --include="*.ts" --include="*.tsx" | grep -v "node_modules\|\.d\.ts"
```

Use proper types, `unknown`, or generic constraints.

### F5: Components under 200 lines (Minor)

```bash
find components/frontend/src/ -name "*.tsx" -print0 | xargs -0 wc -l | sort -rn | head -20
```

Flag components exceeding 200 lines. Consider splitting.

### F6: Loading/error/empty states (Major)

For components using `useQuery`:
- Must reference `isLoading` or `isPending`
- Must reference `error`
- Should handle empty data

```bash
grep -rl "useQuery\|useSessions\|useSession" \
  components/frontend/src/app/ components/frontend/src/components/ --include="*.tsx"
```

Then check each file for `isLoading\|isPending` and `error` references.

### F7: Single-use components in shared directories (Minor)

Check `components/frontend/src/components/` for components imported only once. These should be co-located with their page in `_components/`.

### F8: Feature flag on new pages (Major)

New `page.tsx` files should reference `useWorkspaceFlag` or `useFlag` for feature gating.

## Output Format

```markdown
# Frontend Review

## Summary
[1-2 sentence overview]

## Findings

### Blocker
[Must fix — or "None"]

### Critical
[Should fix — or "None"]

### Major
[Important — or "None"]

### Minor
[Nice-to-have — or "None"]

## Score
[X/8 checks passed]
```

Each finding includes: file:line, problem description, convention violated, suggested fix.
