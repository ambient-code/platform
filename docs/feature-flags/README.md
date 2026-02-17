# Feature Flags

Documentation for feature flag integration in the Ambient Code Platform.

## Available Integrations

### Unleash – Feature Toggles
**[Unleash Integration Guide](feature-flags-unleash.md)**

Use [Unleash](https://www.getunleash.io/) to enable or disable features without redeploying:

- **Frontend**: Next.js proxy at `/api/feature-flags`; use `useFlag()` / `useVariant()` from `@/lib/feature-flags` in client components
- **Backend**: Go SDK; use `handlers.FeatureEnabled()` or `handlers.FeatureEnabledForRequest()` in handlers
- **Admin UI**: Manage feature toggles directly from the workspace UI (see below)
- When Unleash is not configured, all flags are disabled (safe default)

**Environment variables:** `UNLEASH_URL`, `UNLEASH_CLIENT_KEY` (and optionally `UNLEASH_APP_NAME` for frontend). See the guide for per-component details.

### Feature Flags Admin UI

The platform includes a built-in admin UI for managing feature flags directly from the workspace. Navigate to **Workspace > Feature Flags** to:

- View all feature toggles and their current state
- Enable/disable toggles with a single click
- See toggle types (release, experiment, operational, kill-switch)
- View toggle descriptions and stale status

**Additional environment variables for Admin UI:**

| Variable | Required | Description |
|----------|----------|-------------|
| `UNLEASH_ADMIN_URL` | Yes | Unleash server base URL (same as `UNLEASH_URL` typically) |
| `UNLEASH_ADMIN_TOKEN` | Yes | Admin API token (different from Client API token) |
| `UNLEASH_PROJECT` | No | Unleash project ID (default: `default`) |
| `UNLEASH_ENVIRONMENT` | No | Target environment for toggles (default: `development`) |

To get an Admin API token, go to Unleash UI > Admin > API tokens and create a token with Admin permissions.

---

## Quick Start

### 1. Configure Unleash

Set environment variables for the components you use:

**Frontend** (ConfigMap/Secret or `.env.local`):

```bash
UNLEASH_URL=https://unleash.example.com
UNLEASH_CLIENT_KEY=your-frontend-api-token
```

**Backend** (ConfigMap/Secret):

```bash
UNLEASH_URL=https://unleash.example.com
UNLEASH_CLIENT_KEY=your-client-api-token
```

### 2. Create a toggle in Unleash

In the Unleash UI, create a feature toggle (e.g. `my-feature`). Enable or disable it at any time; no redeploy needed.

### 3. Use in code

**Frontend (client component):**

```ts
import { useFlag } from '@/lib/feature-flags';
const enabled = useFlag('my-feature');
```

**Backend (handler):**

```go
if handlers.FeatureEnabled("my-feature") {
    // new behavior
}
```

---

## Related Documentation

- [Frontend README](../../components/frontend/README.md) – Development and env overview
- [Backend README](../../components/backend/README.md) – Development and env overview
- [Architecture](../architecture/) – System design

## References

- **Unleash**: https://docs.getunleash.io/
- **Unleash Go SDK**: https://pkg.go.dev/github.com/Unleash/unleash-go-sdk/v5
- **Unleash React SDK**: https://docs.getunleash.io/sdks/react
