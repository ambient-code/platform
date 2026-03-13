# Frontend Development Context

**When to load:** Working on NextJS application, UI components, or React Query integration

## Quick Reference

- **Framework:** Next.js 14 (App Router)
- **UI Library:** Shadcn UI (built on Radix UI primitives)
- **Styling:** Tailwind CSS
- **Data Fetching:** TanStack React Query
- **Primary Directory:** `components/frontend/src/`

## Critical Rules (Zero Tolerance)

### 1. Zero `any` Types

**FORBIDDEN:**

```typescript
// ❌ BAD
function processData(data: any) { ... }
```

**REQUIRED:**

```typescript
// ✅ GOOD - use proper types
function processData(data: AgenticSession) { ... }

// ✅ GOOD - use unknown if type truly unknown
function processData(data: unknown) {
  if (isAgenticSession(data)) { ... }
}
```

### 2. Shadcn UI Components Only

**FORBIDDEN:** Creating custom UI components from scratch for buttons, inputs, dialogs, etc.

**REQUIRED:** Use `@/components/ui/*` components

```typescript
// ❌ BAD
<button className="px-4 py-2 bg-blue-500">Click</button>

// ✅ GOOD
import { Button } from "@/components/ui/button"
<Button>Click</Button>
```

**Available Shadcn components:** button, card, dialog, form, input, select, table, toast, etc.
**Check:** `components/frontend/src/components/ui/` for full list

### 3. React Query for ALL Data Operations

**FORBIDDEN:** Manual `fetch()` calls in components

**REQUIRED:** Use hooks from `@/services/queries/*`

```typescript
// ❌ BAD
const [sessions, setSessions] = useState([])
useEffect(() => {
  fetch('/api/sessions').then(r => r.json()).then(setSessions)
}, [])

// ✅ GOOD
import { useSessions } from "@/services/queries/sessions"
const { data: sessions, isLoading } = useSessions(projectName)
```

### 4. Use `type` Over `interface`

**REQUIRED:** Always prefer `type` for type definitions

```typescript
// ❌ AVOID
interface User { name: string }

// ✅ PREFERRED
type User = { name: string }
```

### 5. Colocate Single-Use Components

**FORBIDDEN:** Creating components in shared directories if only used once

**REQUIRED:** Keep page-specific components with their pages

```
app/
  projects/
    [projectName]/
      sessions/
        _components/        # Components only used in sessions pages
          session-card.tsx
        page.tsx           # Uses session-card
```

## Common Patterns

### Page Structure

```typescript
// app/projects/[projectName]/sessions/page.tsx
import { useSessions } from "@/services/queries/sessions"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"

export default function SessionsPage({
  params,
}: {
  params: { projectName: string }
}) {
  const { data: sessions, isLoading, error } = useSessions(params.projectName)

  if (isLoading) return <div>Loading...</div>
  if (error) return <div>Error: {error.message}</div>
  if (!sessions?.length) return <div>No sessions found</div>

  return (
    <div>
      {sessions.map(session => (
        <Card key={session.metadata.name}>
          {/* ... */}
        </Card>
      ))}
    </div>
  )
}
```

### React Query Hook Pattern

```typescript
// services/queries/sessions.ts
import { useQuery, useMutation } from "@tanstack/react-query"
import { sessionApi } from "@/services/api/sessions"

export function useSessions(projectName: string) {
  return useQuery({
    queryKey: ["sessions", projectName],
    queryFn: () => sessionApi.list(projectName),
  })
}

export function useCreateSession(projectName: string) {
  return useMutation({
    mutationFn: (data: CreateSessionRequest) =>
      sessionApi.create(projectName, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sessions", projectName] })
    },
  })
}
```

## Pre-Commit Checklist

- [ ] Zero `any` types (or justified with eslint-disable)
- [ ] All UI uses Shadcn components
- [ ] All data operations use React Query
- [ ] Components under 200 lines
- [ ] Single-use components colocated
- [ ] All buttons have loading states
- [ ] All lists have empty states
- [ ] All nested pages have breadcrumbs
- [ ] `npm run build` passes with 0 errors, 0 warnings
- [ ] All types use `type` instead of `interface`

## Key Files

- `components/frontend/DESIGN_GUIDELINES.md` - Comprehensive patterns
- `components/frontend/COMPONENT_PATTERNS.md` - Architecture patterns
- `.claude/patterns/react-state-stability.md` - React rendering & state stability patterns
- `.claude/patterns/react-query-usage.md` - React Query data fetching patterns
- `src/components/ui/` - Shadcn UI components
- `src/services/queries/` - React Query hooks
- `src/services/api/` - API client layer

## Theme Creation Guidelines

When creating or modifying UI themes (light, dark, custom variants):

### Visual Distinction Requirements

**CRITICAL:** New theme variants MUST be visually distinct from existing themes.

**What "visually distinct" means:**
- Background colors differ by at least 20% lightness (L in OKLCH)
- Primary/accent colors use different hue ranges (not just different shades of the same color)
- At a glance, a user can immediately identify which theme is active

**Example - What NOT to do:**
```css
/* ❌ BAD: LibreChat theme too similar to light theme */
.librechat {
  --background: oklch(0.98 0 0);  /* Nearly white, like light theme */
  --foreground: oklch(0.15 0 0);  /* Nearly same as light theme */
  --primary: oklch(0.51 0.21 265); /* Very similar to light theme primary */
}

/* Light theme for comparison */
:root {
  --background: oklch(1 0 0);     /* White */
  --foreground: oklch(0.145 0 0); /* Dark gray */
  --primary: oklch(0.5 0.22 264); /* Purple-blue */
}
/* These are nearly indistinguishable! */
```

**Example - What to do:**
```css
/* ✅ GOOD: Solarized theme clearly distinct */
.solarized-light {
  --background: oklch(0.97 0.01 85);  /* Warm cream background */
  --foreground: oklch(0.35 0.05 192); /* Cool teal foreground */
  --primary: oklch(0.55 0.15 192);    /* Blue-cyan accent */
  /* Clearly different warm/cool palette */
}

/* ✅ GOOD: High-contrast theme */
.high-contrast {
  --background: oklch(1 0 0);      /* Pure white */
  --foreground: oklch(0 0 0);      /* Pure black */
  --primary: oklch(0.45 0.3 240);  /* Vibrant blue */
  /* Much stronger contrast than default */
}
```

### Theme Creation Checklist

Before creating a new theme variant:

- [ ] Compare background lightness values (L in OKLCH) - minimum 20% difference
- [ ] Check primary color hue - should be different color family (not just darker/lighter)
- [ ] Test with actual UI - can you immediately tell themes apart?
- [ ] Verify contrast ratios meet WCAG AA standards (4.5:1 for text)
- [ ] Check both light and dark variants if creating a full theme

### Quick Comparison Test

After creating a theme:
1. Take screenshot of UI in new theme
2. Take screenshot of UI in similar existing theme
3. Place side-by-side
4. If you have to squint or look carefully to tell them apart → **Not distinct enough**

## Recent Issues & Learnings

- **2026-03-02:** Added React state stability patterns - prevent timestamp re-calculation bugs
- **2026-03-02:** Added theme creation guidelines - ensure visual distinction between theme variants
- **2024-11-18:** Migrated all data fetching to React Query - no more manual fetch calls
- **2024-11-15:** Enforced Shadcn UI only - removed custom button components
- **2024-11-10:** Added breadcrumb pattern for nested pages
