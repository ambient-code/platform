# Adding a Feature Behind a Feature Flag

All new user-facing features must be gated behind a feature flag ([ADR-0007](../adr/0007-unleash-feature-flags.md)). This guide walks through the end-to-end process.

## Quick Checklist

- [ ] Choose a flag name following the naming convention
- [ ] Register the flag in `flags.json` (or `models.json` for model flags)
- [ ] Gate the backend handler
- [ ] Gate the frontend component
- [ ] Decide visibility: platform-only or workspace-configurable
- [ ] Test with Unleash disabled (verify fail-closed behavior)

---

## 1. Choose a Flag Name

Follow the `<component>.<feature>.<aspect>` convention:

| Category | Pattern | Example | Fail Mode |
|----------|---------|---------|-----------|
| General | `<component>.<feature>.<aspect>` | `frontend.file-explorer.enabled` | Fail-closed |
| Runner | `runner.<runnerId>.enabled` | `runner.gemini-cli.enabled` | Fail-closed |
| Model | `model.<modelId>.enabled` | `model.claude-opus-4-6.enabled` | Fail-open |

General and runner flags default to **off** when Unleash is unavailable (fail-closed). Model flags default to **on** (fail-open) so model availability is never blocked by flag infrastructure outages.

## 2. Register the Flag

### General / runner flags

Add an entry to `components/manifests/base/core/flags.json`:

```json
{
  "flags": [
    {
      "name": "frontend.my-feature.enabled",
      "description": "Enable the my-feature UI for session creation",
      "tags": [
        {
          "type": "scope",
          "value": "workspace"
        }
      ]
    }
  ]
}
```

Omit the `tags` array to make the flag platform-only (not visible in the workspace admin UI).

### Model flags

Model flags are auto-generated from `components/manifests/base/core/models.json` at startup. Set `"featureGated": true` on the model entry. No `flags.json` entry is needed.

### What happens at startup

The backend syncs `flags.json` and `models.json` to Unleash on boot (`cmd/sync_flags.go`). New flags are created automatically; flags for removed models are archived. The sync requires `UNLEASH_ADMIN_URL` and `UNLEASH_ADMIN_TOKEN` and skips silently if they are not set.

## 3. Gate the Backend

Use the handler-level wrappers in `handlers/featureflags.go`. Choose the right function based on your use case:

```go
import "net/http"

// Option A: Hide the feature entirely (404 when disabled)
if !handlers.FeatureEnabled("frontend.my-feature.enabled") {
    c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
    return
}

// Option B: Branch behavior (legacy vs new)
if handlers.FeatureEnabled("frontend.my-feature.enabled") {
    handleNewBehavior(c)
} else {
    handleLegacyBehavior(c)
}

// Option C: Per-user rollout (Unleash strategies with user context)
if !handlers.FeatureEnabledForRequest(c, "frontend.my-feature.enabled") {
    c.JSON(http.StatusForbidden, gin.H{"error": "feature not enabled for you"})
    return
}
```

| Function | Fail Mode | When to Use |
|----------|-----------|-------------|
| `handlers.FeatureEnabled(flag)` | Fail-closed | Same result for all users |
| `handlers.FeatureEnabledForRequest(c, flag)` | Fail-closed | Per-user rollout, A/B tests |
| `featureflags.IsModelEnabled(flag)` | Fail-open | Model availability checks |

### Reference files

- `components/backend/featureflags/featureflags.go` — SDK init, `IsEnabled`, `IsModelEnabled`
- `components/backend/handlers/featureflags.go` — handler wrappers

## 4. Gate the Frontend

### Client-side flag (Unleash SDK)

For flags evaluated purely in the browser via the Unleash React SDK:

```tsx
import { useFlag } from '@/lib/feature-flags';

export function MyComponent() {
  const enabled = useFlag('frontend.my-feature.enabled');
  if (!enabled) return null;

  return <NewFeature />;
}
```

### Workspace-scoped flag (backend evaluation)

For flags that respect workspace ConfigMap overrides:

```tsx
import { useWorkspaceFlag } from '@/services/queries/use-feature-flags-admin';

export function MyComponent({ projectName }: { projectName: string }) {
  const { enabled, isLoading } = useWorkspaceFlag(projectName, 'frontend.my-feature.enabled');

  if (isLoading) return <Spinner />;
  if (!enabled) return null;

  return <NewFeature />;
}
```

Use `useWorkspaceFlag` when workspace admins should be able to override the flag independently.

## 5. Choose Visibility

| Visibility | Unleash Tag | Who Controls | Use When |
|------------|-------------|--------------|----------|
| Workspace-configurable | `scope: workspace` | Workspace admins + Platform team | Beta opt-in, experimental UI |
| Platform-only | _(no tag)_ | Platform team only | Infrastructure, security, kill switches |

To make a flag workspace-configurable, include the tag in `flags.json` (shown in step 2) or add it manually in the Unleash UI: open the flag > Tags > add type `scope`, value `workspace`.

## 6. Test Locally

### Without Unleash (most common)

When `UNLEASH_URL` is not set, the SDK is not initialized:

- General flags → `false` (fail-closed). Your feature should be **hidden**.
- Model flags → `true` (fail-open). Models should be **available**.

Run through the UI and verify the gated feature is not visible.

### With Unleash

```bash
make deploy-unleash-kind    # Deploy Unleash to Kind cluster
make unleash-port-forward   # Access at http://localhost:4242
```

Then toggle the flag in the Unleash UI and verify the feature turns on/off without a redeploy.

### Verify fail-closed behavior

1. Stop or remove the Unleash deployment
2. Restart the backend
3. Confirm your feature is hidden (general flags) or available (model flags)

---

## Evaluation Precedence

When a flag is evaluated for a workspace, three layers are checked in order:

1. **Workspace ConfigMap override** (highest priority) — `feature-flag-overrides` ConfigMap in the workspace namespace
2. **Unleash SDK evaluation** — respects strategies, rollout percentages, A/B tests
3. **Code default** (lowest priority) — general: `false`, model: `true`

See [Fail Modes Reference](../feature-flags/fail-modes.md) for the full matrix.

## Further Reading

- [Feature Flags Overview](../feature-flags/) — Unleash integration, admin UI, API endpoints
- [Fail Modes Reference](../feature-flags/fail-modes.md) — fail-open vs fail-closed details
- [ADR-0007](../adr/0007-unleash-feature-flags.md) — architectural decision and rationale
