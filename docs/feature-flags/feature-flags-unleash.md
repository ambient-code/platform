# Feature Flags with Unleash

The platform uses [Unleash](https://www.getunleash.io/) for optional feature toggles in the frontend and backend. When Unleash is not configured, all flags are disabled (safe default).

## Overview

- **Frontend**: Next.js proxy at `/api/feature-flags` forwards to Unleash so the client key is never exposed. Use `useFlag()` / `useVariant()` from `@/lib/feature-flags` in client components.
- **Backend**: Go SDK initializes when `UNLEASH_URL` and `UNLEASH_CLIENT_KEY` are set. Use `handlers.FeatureEnabled()` or `handlers.FeatureEnabledForRequest()` in handlers.

Create toggles in the Unleash UI; enable or disable them without redeploying.

---

## Frontend

### Environment variables

Set these for the **frontend** (e.g. in deployment ConfigMap/Secret or `.env.local`):

| Variable | Required | Description |
|----------|----------|-------------|
| `UNLEASH_URL` | Yes* | Unleash server base URL (e.g. `https://unleash.example.com`) |
| `UNLEASH_CLIENT_KEY` | Yes* | Frontend API token (used by the Next.js proxy only; never sent to the browser) |
| `UNLEASH_APP_NAME` | No | App name sent to Unleash (default: `ambient-code-platform`) |

\*If either is missing, the proxy returns empty toggles and all flags are false.

### Usage in components

In **client components** only:

```ts
import { useFlag, useVariant, useFlagsStatus } from '@/lib/feature-flags';

// Simple toggle
const enabled = useFlag('my-feature-name');
if (enabled) return <NewFeature />;
return <OldFeature />;

// A/B or variant
const variant = useVariant('experiment-name');
// use variant.name to decide which variant to show

// Wait for flags to load before rendering flag-dependent UI
const { flagsReady, flagsError } = useFlagsStatus();
if (!flagsReady) return <Spinner />;
```

### Deployment

Add `UNLEASH_URL`, `UNLEASH_CLIENT_KEY`, and optionally `UNLEASH_APP_NAME` to the frontend deployment (ConfigMap/Secret). The backend does not need these unless you use backend feature flags.

---

## Backend

### Environment variables

Set these for the **backend** (e.g. in deployment ConfigMap/Secret):

| Variable | Required | Description |
|----------|----------|-------------|
| `UNLEASH_URL` | Yes* | Unleash server base URL (e.g. `https://unleash.example.com`) |
| `UNLEASH_CLIENT_KEY` | Yes* | API token for the Unleash Client API (backend token; can be same or different from frontend) |

\*If either is missing, `featureflags.Init()` does nothing and all flag checks return `false`.

### Usage in handlers

- **Global check** (same for all requests): `handlers.FeatureEnabled("flag-name")`
- **Per-request** (user/session/IP for strategies): `handlers.FeatureEnabledForRequest(c, "flag-name")`

When Unleash is not configured, both return `false`.

### Example: enable/disable a feature (e.g. FakeFeature)

1. In Unleash, create a toggle named `fake-feature` (or whatever name you use in code).
2. In the handler (or middleware) that should be gated:

```go
// Option A: Hide the feature entirely (e.g. return 404 when disabled)
if !handlers.FeatureEnabled("fake-feature") {
    c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
    return
}
// ... handle FakeFeature ...

// Option B: Branch behavior (legacy vs new)
if handlers.FeatureEnabled("fake-feature") {
    handleFakeFeatureNew(c)
} else {
    handleFakeFeatureLegacy(c)
}

// Option C: Per-user rollout (e.g. beta users only)
if !handlers.FeatureEnabledForRequest(c, "fake-feature") {
    c.JSON(http.StatusForbidden, gin.H{"error": "feature not enabled for you"})
    return
}
```

Turn the feature on or off in the Unleash UI; no redeploy needed.

### Dependency

Backend uses `github.com/Unleash/unleash-go-sdk/v5`. Ensure it is in `go.mod` and run `go mod tidy` or `go get github.com/Unleash/unleash-go-sdk/v5` if needed.

### Reference

- `components/backend/featureflags/featureflags.go` – Unleash client init, `IsEnabled`, `IsEnabledWithContext`
- `components/backend/handlers/featureflags.go` – `FeatureEnabled`, `FeatureEnabledForRequest`

---

## Admin UI

The platform includes a built-in admin UI for managing feature flags directly from the workspace. This allows users to view and toggle flags without accessing the Unleash dashboard.

### Environment Variables

Set these for the **backend** to enable the Admin UI:

| Variable | Required | Description |
|----------|----------|-------------|
| `UNLEASH_ADMIN_URL` | Yes | Unleash server base URL (e.g. `https://unleash.example.com`) |
| `UNLEASH_ADMIN_TOKEN` | Yes | Admin API token (from Unleash > Admin > API tokens) |
| `UNLEASH_PROJECT` | No | Unleash project ID (default: `default`) |
| `UNLEASH_ENVIRONMENT` | No | Target environment for toggles (default: `development`) |

**Note:** The Admin API token is different from the Client API token. Create one in Unleash UI > Admin > API tokens with Admin permissions.

### Using the Admin UI

1. Navigate to your workspace in the platform
2. Click **Feature Flags** in the sidebar
3. View all toggles with their current enabled/disabled state
4. Click the toggle switch to enable or disable a flag
5. Changes take effect immediately for new sessions

### API Endpoints

The backend exposes these endpoints (proxied to Unleash Admin API):

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/projects/:projectName/feature-flags` | GET | List all feature toggles |
| `/api/projects/:projectName/feature-flags/:flagName` | GET | Get toggle details |
| `/api/projects/:projectName/feature-flags/:flagName/enable` | POST | Enable toggle in environment |
| `/api/projects/:projectName/feature-flags/:flagName/disable` | POST | Disable toggle in environment |

### Example: Toggle a flag via API

```bash
# List all flags
curl -H "Authorization: Bearer $TOKEN" \
  https://your-backend/api/projects/my-workspace/feature-flags

# Enable a flag
curl -X POST -H "Authorization: Bearer $TOKEN" \
  https://your-backend/api/projects/my-workspace/feature-flags/my-feature/enable

# Disable a flag
curl -X POST -H "Authorization: Bearer $TOKEN" \
  https://your-backend/api/projects/my-workspace/feature-flags/my-feature/disable
```

### Reference

- `components/backend/handlers/featureflags_admin.go` – Admin API handlers
- `components/frontend/src/components/workspace-sections/feature-flags-section.tsx` – Admin UI component
- `components/frontend/src/services/queries/use-feature-flags-admin.ts` – React Query hooks
