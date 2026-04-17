# Frontend Development Context

> Part of [CLAUDE.md Critical Conventions](../../CLAUDE.md#critical-conventions)

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

## Feature Flags

**FORBIDDEN:** `useFlag()` from `@unleash/proxy-client-react` — it doesn't work with workspace overrides.

**REQUIRED:** `useWorkspaceFlag(projectName, flagName)` for workspace-scoped flags (shared hook, 15s staleTime). Evaluates: ConfigMap override > Unleash default.

```typescript
// ❌ BAD
import { useFlag } from "@unleash/proxy-client-react"
const enabled = useFlag("my-feature")

// ✅ GOOD
import { useWorkspaceFlag } from "@/services/queries/feature-flags"
const enabled = useWorkspaceFlag(projectName, "my-feature")
```

If a specific page needs instant flag freshness, use `useQuery` with `evaluateFeatureFlag()` directly and set `staleTime: 0, refetchOnMount: 'always'` — don't modify the shared hook.

New flags go in `components/manifests/base/core/flags.json` with `scope:workspace` tag. Use `/unleash-flag` to scaffold.

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

## Session Creation: Adding Options to the + Menu

The `+` button dropdown in `new-session-view.tsx` is the single entry point for all per-session configuration.

**File:** `src/app/projects/[name]/sessions/[sessionName]/components/new-session-view.tsx`

### Adding a Boolean Toggle

1. Add state: `const [myToggle, setMyToggle] = useState(false)`
2. Add `DropdownMenuCheckboxItem` after `<DropdownMenuSeparator />`
3. Wire into `handleSubmit` → `onCreateSession({ ..., myToggle: myToggle || undefined })`
4. Update `NewSessionViewProps.onCreateSession` config type
5. Wire through `page.tsx` into the mutation payload
6. Bump `MENU_VERSION` constant (triggers the discovery dot)

### Adding a Form-Heavy Config

1. Create form component (e.g., `components/my-config.tsx`) with save/preview/dirty-guard
2. Add `DropdownMenuItem` that opens a `Dialog`
3. Render the `Dialog` outside the dropdown (at component root level)
4. Wire saved values into `handleSubmit` → `onCreateSession`
5. Bump `MENU_VERSION` constant

### Discovery Dot

`MENU_VERSION` constant (date string) + `useLocalStorage` with key `acp-menu-seen-version`. Bump the version when adding new menu items — the dot appears until the user opens the menu.

### Backend Wiring

New options flow: `onCreateSession` config → `page.tsx` → `createSessionMutation.mutate()` → POST `/api/projects/{project}/agentic-sessions` → backend handler → CR spec. For booleans, add to `CreateAgenticSessionRequest` in `types/session.go`. For complex options, serialize to env var on the CR and parse in the runner.

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
- `src/components/ui/` - Shadcn UI components
- `src/services/queries/` - React Query hooks
- `src/services/api/` - API client layer
