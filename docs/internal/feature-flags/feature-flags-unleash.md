# Feature Flags with Unleash

The platform uses [Unleash](https://www.getunleash.io/) for optional feature toggles in the frontend and backend. When Unleash is not configured, general flags are disabled (fail-closed) and model flags remain enabled (fail-open). See [fail-modes.md](fail-modes.md) for the full reference.

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

\*If either is missing, `featureflags.Init()` does nothing. General flag checks return `false` (fail-closed); model flag checks return `true` (fail-open).

### Usage in handlers

- **Global check** (same for all requests): `handlers.FeatureEnabled("flag-name")` — fail-closed
- **Per-request** (user/session/IP for strategies): `handlers.FeatureEnabledForRequest(c, "flag-name")` — fail-closed
- **Model check**: `featureflags.IsModelEnabled("flag-name")` — fail-open (models stay available)
- **Model check with context**: `featureflags.IsModelEnabledWithContext("flag-name", userID, sessionID, remoteAddr)` — fail-open

General flags return `false` when Unleash is not configured. Model flags return `true` so that model availability is not blocked by flag infrastructure outages.

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

## Unleash Server UI

The Unleash server has a built-in web UI for managing flags, strategies, API tokens, and projects. This is distinct from the workspace admin UI — the Unleash UI gives platform team members full control over all flags globally.

### Accessing the UI

**Local development (kind):**

```bash
make unleash-port-forward
# Access at http://localhost:4242
```

**OpenShift (local-dev / production):**

The Unleash UI is exposed via an OpenShift Route:

```bash
echo "https://$(oc get route unleash-route -n ambient-code -o jsonpath='{.spec.host}')"
```

### Default credentials

On first startup, Unleash creates an `admin` user. The password is set by the `default-admin-password` key in the `unleash-credentials` secret. See `unleash-credentials-secret.yaml.example` for the template.

### Common tasks

- **Create API tokens:** Admin > API tokens. You need separate tokens for Admin API (`UNLEASH_ADMIN_TOKEN`), Client API (`UNLEASH_CLIENT_KEY`), and optionally Frontend API.
- **Create flags manually:** New feature toggle > type "release" > add strategies and tags.
- **Tag a flag as workspace-configurable:** Open the flag > Tags > add `scope:workspace`. The flag will then appear in the workspace admin UI.
- **View flag metrics:** Open a flag > Metrics tab to see evaluation counts and SDK usage.

---

## Workspace Feature Flags (Ambient UI)

The Ambient platform UI includes a built-in feature flags section within each workspace's settings. This is separate from the Unleash server UI — it only shows flags tagged `scope:workspace` and lets workspace admins set per-workspace overrides without affecting other workspaces or needing access to Unleash directly.

### Evaluation Precedence

Flag evaluation uses a three-tier system:

1. **Workspace override (ConfigMap)** — highest priority. Stored in a `feature-flag-overrides` ConfigMap in the workspace namespace.
2. **Unleash global default** — fallback. Respects Unleash strategies, rollouts, and A/B tests.
3. **Code default** — absolute fallback. General flags: `false` (fail-closed). Model flags: `true` (fail-open).

### Environment Variables

Set these for the **backend** to enable the Admin UI:

| Variable | Required | Description |
|----------|----------|-------------|
| `UNLEASH_ADMIN_URL` | Yes | Unleash server base URL (e.g. `https://unleash.example.com`) |
| `UNLEASH_ADMIN_TOKEN` | Yes | Admin API token (from Unleash > Admin > API tokens) |
| `UNLEASH_PROJECT` | No | Unleash project ID (default: `default`) |
| `UNLEASH_ENVIRONMENT` | No | Target environment for toggles (default: `development`) |
| `UNLEASH_WORKSPACE_TAG_TYPE` | No | Tag type for workspace-configurable flags (default: `scope`) |
| `UNLEASH_WORKSPACE_TAG_VALUE` | No | Tag value for workspace-configurable flags (default: `workspace`) |

**Note:** The Admin API token is different from the Client API token. Create one in Unleash UI > Admin > API tokens with Admin permissions.

### Using the Admin UI

1. Navigate to your workspace in the platform
2. Click the **Workspace Settings** tab
3. Scroll down to the **Feature Flags** card at the bottom of the page
4. View all workspace-configurable flags grouped by category
4. Set overrides using the three-state control: **Default** (use platform value), **On** (force enable), **Off** (force disable)
5. Click **Save** to commit all pending changes (batch save pattern)
6. Click **Discard** to revert unsaved changes

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/projects/:name/feature-flags` | GET | List workspace-configurable flags with override status |
| `/api/projects/:name/feature-flags/evaluate/:flagName` | GET | Evaluate flag (ConfigMap override then Unleash fallback) |
| `/api/projects/:name/feature-flags/:flagName` | GET | Get single flag details from Unleash |
| `/api/projects/:name/feature-flags/:flagName/override` | PUT | Set workspace override (`{"enabled": bool}`) |
| `/api/projects/:name/feature-flags/:flagName/override` | DELETE | Remove workspace override (revert to Unleash default) |
| `/api/projects/:name/feature-flags/:flagName/enable` | POST | Enable flag (sets ConfigMap override to `"true"`) |
| `/api/projects/:name/feature-flags/:flagName/disable` | POST | Disable flag (sets ConfigMap override to `"false"`) |

### Example: Toggle a flag via API

```bash
# List all workspace-configurable flags
curl -H "Authorization: Bearer $TOKEN" \
  https://your-backend/api/projects/my-workspace/feature-flags

# Evaluate a flag for the workspace
curl -H "Authorization: Bearer $TOKEN" \
  https://your-backend/api/projects/my-workspace/feature-flags/evaluate/model.claude-opus-4-6.enabled

# Set a workspace override
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"enabled": true}' \
  https://your-backend/api/projects/my-workspace/feature-flags/model.claude-opus-4-6.enabled/override

# Remove a workspace override (revert to Unleash default)
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  https://your-backend/api/projects/my-workspace/feature-flags/model.claude-opus-4-6.enabled/override

# Enable a flag (shorthand for setting override to true)
curl -X POST -H "Authorization: Bearer $TOKEN" \
  https://your-backend/api/projects/my-workspace/feature-flags/my-feature/enable

# Disable a flag
curl -X POST -H "Authorization: Bearer $TOKEN" \
  https://your-backend/api/projects/my-workspace/feature-flags/my-feature/disable
```

### Reference

- `components/backend/featureflags/featureflags.go` – Unleash SDK init, fail-open/closed defaults
- `components/backend/handlers/featureflags_admin.go` – Admin API handlers, workspace override logic
- `components/backend/cmd/sync_flags.go` – Flag sync to Unleash at startup
- `components/frontend/src/components/workspace-sections/feature-flags-section.tsx` – Admin UI component
- `components/frontend/src/services/queries/use-feature-flags-admin.ts` – React Query hooks
