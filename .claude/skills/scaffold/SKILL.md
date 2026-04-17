---
name: scaffold
description: >
  Generate the complete file set for a new integration, endpoint, or feature
  flag following established project patterns. Use when adding a new
  integration (like Jira, CodeRabbit, Google Drive), creating a new API
  endpoint with full stack, or setting up a new feature flag. Triggers on:
  "new integration", "add an integration", "scaffold", "create endpoint",
  "add a new provider", "wire up a new service". Always includes a feature
  flag. Follows the Jira integration pattern.
---

# Scaffold

Generate the complete file set for a new integration, endpoint, or feature flag.

## Usage

```bash
/scaffold integration <name>    # Full integration scaffold
/scaffold endpoint <name>       # API endpoint scaffold
/scaffold feature-flag <name>   # Feature flag scaffold (delegates to /unleash-flag)
```

## User Input

```text
$ARGUMENTS
```

Parse the scaffold type and name from `$ARGUMENTS`. If ambiguous, ask the user.

## Integration Scaffold

Based on the established integration pattern (Jira, CodeRabbit, Google Drive), generate the full file set.

### Backend Files

| File | Purpose | Template |
|------|---------|----------|
| `components/backend/handlers/{provider}_auth.go` | Auth handlers + K8s Secret CRUD | Follow `jira_auth.go` pattern |
| `components/backend/handlers/integration_validation.go` | Add validation + test endpoint | Add `Validate{Provider}` function |
| `components/backend/handlers/integrations_status.go` | Add to unified status | Add provider to status aggregation |
| `components/backend/handlers/runtime_credentials.go` | Session credential fetch with RBAC | Add `fetch{Provider}Credentials` |
| `components/backend/routes.go` | Register endpoints | Add route group with auth middleware |

### Frontend Files

| File | Purpose | Template |
|------|---------|----------|
| `components/frontend/src/components/integrations/{provider}-connection-card.tsx` | Integration card UI | Follow existing integration card (e.g., Jira) |
| `components/frontend/src/services/api/{provider}-auth.ts` | API client | Follow existing auth service pattern |
| `components/frontend/src/services/queries/use-{provider}.ts` | React Query hooks | Follow existing query hook pattern |
| `components/frontend/src/app/api/auth/{provider}/route.ts` | Next.js proxy route | Follow existing auth proxy |
| `components/frontend/src/components/integrations/IntegrationsClient.tsx` | Add card import | Update imports + render |
| `components/frontend/src/components/integrations/integrations-panel.tsx` | Add to panel | Update panel |

> **Note:** Before scaffolding, verify the reference files exist by checking `components/frontend/src/components/integrations/` and `components/frontend/src/services/`. File names may differ from examples above.

### Runner Files

| File | Purpose | Template |
|------|---------|----------|
| `components/runners/ambient-runner/src/auth.py` | Add `fetch_{provider}_credentials()` | Follow `fetch_jira_credentials` pattern |

### Feature Flag

| File | Purpose |
|------|---------|
| `components/manifests/base/core/flags.json` | Add `integration.{provider}.enabled` |

### Checklist

After scaffolding, verify:

- [ ] All backend handlers use `GetK8sClientsForRequest` for user operations
- [ ] Credentials stored in K8s Secret with OwnerReferences
- [ ] Frontend uses React Query hooks (no manual fetch)
- [ ] Frontend uses Shadcn UI components
- [ ] Feature flag gates the integration card
- [ ] Tests mock the feature flag hook
- [ ] Runner credential fetch is added to `populate_runtime_credentials()`

## Endpoint Scaffold

For adding a new API endpoint with full-stack support.

### Files to Create/Modify

| Layer | File | Action |
|-------|------|--------|
| Backend handler | `components/backend/handlers/{resource}.go` | Create |
| Backend routes | `components/backend/routes.go` | Add routes |
| Backend types | `components/backend/types/{resource}.go` | Create if needed |
| Frontend API | `components/frontend/src/services/api/{resource}.ts` | Create |
| Frontend queries | `components/frontend/src/services/queries/{resource}.ts` | Create |
| Frontend proxy | `components/frontend/src/app/api/{resource}/route.ts` | Create |

### Backend Handler Template

```go
func List{Resource}(c *gin.Context) {
    projectName := c.Param("projectName")

    reqK8s, reqDyn := GetK8sClientsForRequest(c)
    if reqK8s == nil || reqDyn == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
        return
    }

    // RBAC check
    // List operation with reqDyn
    // Return response
}
```

### Frontend Query Template

```typescript
export function use{Resource}(projectName: string) {
  return useQuery({
    queryKey: ["{resource}", projectName],
    queryFn: () => {resource}Api.list(projectName),
  })
}
```

## Feature Flag Scaffold

Delegates to the `/unleash-flag` skill for the full feature flag workflow.

Run `/unleash-flag` with the flag name and follow its checklist.
