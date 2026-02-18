# ADR-0006: Unleash for Feature Flag Management

**Status:** Proposed
**Date:** 2026-02-17
**Deciders:** Platform Team
**Technical Story:** Constitution Principle XI requires all new features gated behind feature flags

## Context and Problem Statement

The project constitution (Principle XI: Feature Flag Discipline) mandates that all new features must be gated behind feature flags. We need a feature flag system that supports:

1. Gradual rollouts and percentage-based targeting
2. A/B testing and experimentation
3. Environment-specific configurations
4. Workspace-scoped overrides (workspace admins can opt-in/out of features)
5. An admin UI for operations to toggle flags without code deploys
6. Client SDK for frontend feature gating

How should we implement feature flag management for the platform?

## Decision Drivers

* **Constitution compliance:** Principle XI requires feature flags for all new features
* **Workspace autonomy:** Workspace admins need to control features for their workspace
* **Platform control:** Platform team needs gradual rollout and A/B testing capabilities
* **Operational control:** Need to enable/disable features without redeployment
* **Multi-environment:** Different flag states for dev, staging, production
* **Security:** Admin API credentials must not be exposed to frontend
* **Ephemeral flags:** Flag definitions change frequently, avoid rigid schemas
* **Flag visibility governance:** Some flags should only be controllable by platform team, not workspace admins

## Considered Options

### Feature Flag Backend

1. **Unleash (self-hosted)** - Open-source feature management platform
2. **LaunchDarkly** - SaaS feature flag service
3. **ConfigMaps/Environment Variables** - Simple Kubernetes-native approach
4. **Custom solution** - Build feature flag system from scratch
5. **Flipt** - Open-source alternative to Unleash

### Workspace Scoping

A. **Unleash strategies with workspace constraints** - Pass workspace as context, use Unleash constraints
B. **ConfigMap overrides per workspace** - Store overrides in K8s ConfigMap, fall back to Unleash
C. **Separate Unleash projects per workspace** - One Unleash project per workspace
D. **ProjectSettings CRD field** - Add featureFlagOverrides to ProjectSettings spec

## Decision Outcome

### Feature Flag Backend

Chosen option: **"Unleash (self-hosted)"**, because:

1. **Open source:** Self-hosted, no vendor lock-in, data sovereignty
2. **Rich feature set:** Strategies, variants, A/B testing, gradual rollouts
3. **Mature ecosystem:** React SDK, REST APIs, well-documented
4. **Admin UI:** Built-in web interface for flag management
5. **Kubernetes-friendly:** Easy to deploy via Helm charts
6. **Cost:** Free for self-hosted (vs LaunchDarkly pricing)

### Workspace Scoping

Chosen option: **"ConfigMap overrides per workspace"**, because:

1. **Decoupled from CRD schema:** Flags are ephemeral; ConfigMaps don't require schema changes
2. **Kubernetes-native:** ConfigMaps are the standard pattern for configuration
3. **Simple evaluation:** Check ConfigMap first, fall back to Unleash
4. **Preserves Unleash capabilities:** A/B testing and gradual rollouts still work via Unleash
5. **Workspace autonomy:** Admins can override without affecting other workspaces

### Flag Visibility Control

Chosen option: **"Tag-based filtering"**, because:

1. **Platform-controlled:** Platform team decides which flags are workspace-configurable via Unleash tags
2. **No code changes:** Adding/removing workspace-configurable flags requires only tag changes in Unleash
3. **Clear governance:** Explicit separation between platform-only and workspace-configurable flags
4. **Audit trail:** Unleash tracks tag changes with history

**Implementation:**
- Flags with tag `scope: workspace` appear in the workspace admin UI
- Flags without this tag are platform-only (controllable only via Unleash UI)
- Tag type/value configurable via environment variables

**Alternatives considered:**
- Naming convention (e.g., `workspace.*` prefix) - Less flexible, requires flag renaming
- Separate Unleash projects - More complex, harder to manage
- Backend allowlist - Requires code changes for each new flag
- Flag type filtering - Unleash types have semantic meaning, shouldn't overload

### Consequences

**Positive:**

* Full control over feature flag data and availability
* Rich targeting strategies (user ID, percentage, custom constraints) via Unleash
* Workspace admins can opt-in/out of features independently
* Platform team retains A/B testing and gradual rollout capabilities
* No CRD schema changes when flags are added/removed
* React SDK with hooks (`useFlag`) for clean frontend integration

**Negative:**

* Additional infrastructure to maintain (Unleash server + PostgreSQL)
* Two-layer evaluation adds complexity (ConfigMap + Unleash)
* Must implement backend proxy to hide Admin API credentials
* Frontend SDK requires proxy endpoint for Client API

**Risks:**

* Unleash server downtime affects feature flag evaluation (mitigated by ConfigMap overrides)
* ConfigMap and Unleash state could diverge (workspace override vs global state)
* SDK polling interval affects flag update latency (default 15s)

## Implementation Notes

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Frontend (NextJS)                              │
├─────────────────────────────────────────────────────────────────────────┤
│  useWorkspaceFlag(flagName)                                             │
│    └─ Calls /api/projects/:name/feature-flags/evaluate/:flagName        │
│    └─ Returns merged result (ConfigMap override OR Unleash default)     │
│                                                                         │
│  FeatureFlagsSettings (Admin UI in Workspace Settings tab)              │
│    └─ Lists flags from Unleash (global definitions)                     │
│    └─ Shows workspace override status from ConfigMap                    │
│    └─ Batch save pattern: toggles tracked locally, saved on click       │
│    └─ Reset button removes override (reverts to Unleash default)        │
└─────────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          Backend (Go)                                   │
├─────────────────────────────────────────────────────────────────────────┤
│  Flag Evaluation (/projects/:name/feature-flags/evaluate/:flagName)     │
│    1. Read ConfigMap "feature-flag-overrides" in workspace namespace    │
│    2. If override exists → return override value                        │
│    3. If no override → query Unleash with workspace + user context      │
│                                                                         │
│  Override Management (/projects/:name/feature-flags/:flagName/override) │
│    └─ PUT: Set override in ConfigMap (true/false)                       │
│    └─ DELETE: Remove override from ConfigMap (use Unleash default)     │
│                                                                         │
│  Flag Listing (/projects/:name/feature-flags)                           │
│    └─ Returns Unleash flags filtered by tag (scope: workspace)          │
│    └─ Includes workspace override status from ConfigMap                 │
└─────────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌───────────────────────────────────┐    ┌────────────────────────────────┐
│     ConfigMap (per workspace)     │    │        Unleash Server          │
├───────────────────────────────────┤    ├────────────────────────────────┤
│  Name: feature-flag-overrides     │    │  Global flag definitions       │
│  Namespace: workspace-foo         │    │  A/B testing strategies        │
│                                   │    │  Gradual rollout %             │
│  data:                            │    │  Environment configs           │
│    frontend.feature.enabled: true │    │                                │
│    backend.feature.enabled: false │    │  PostgreSQL storage            │
└───────────────────────────────────┘    └────────────────────────────────┘
```

### Evaluation Logic (Three-State)

| ConfigMap Override | Unleash State | Result | Who Controls |
|--------------------|---------------|--------|--------------|
| `"true"` | (any) | `true` | Workspace admin |
| `"false"` | (any) | `false` | Workspace admin |
| (not set) | enabled | `true` | Platform team |
| (not set) | disabled | `false` | Platform team |
| (not set) | 50% rollout | (evaluated) | Platform team |

### Use Cases

| Scenario | Implementation |
|----------|----------------|
| Global rollout to X% of workspaces | Unleash gradual rollout strategy with workspace context |
| A/B test within workspaces | Unleash A/B strategy with user ID context |
| Workspace opts into beta | Workspace admin sets ConfigMap override = `"true"` |
| Workspace opts out of feature | Workspace admin sets ConfigMap override = `"false"` |
| Reset to platform default | Workspace admin deletes ConfigMap key |

### ConfigMap Structure

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: feature-flag-overrides
  namespace: workspace-foo
  labels:
    app.kubernetes.io/managed-by: ambient-code
    app.kubernetes.io/component: feature-flags
data:
  # Override format: flag-name: "true" | "false"
  # Absence of key = use Unleash default
  frontend.file-explorer.enabled: "true"
  frontend.new-chat-ui.enabled: "false"
```

### Flag Naming Convention (Constitution Principle XI)

All feature flags MUST follow the naming pattern:

```
<component>.<feature>.<aspect>
```

Examples:
* `frontend.file-explorer.enabled` - File explorer feature in frontend
* `backend.multi-repo.enabled` - Multi-repo support in backend
* `runner.langfuse.tracing` - Langfuse tracing in runner

### Flag Visibility (Platform-Only vs Workspace-Configurable)

Not all feature flags should be controllable by workspace admins. The platform uses **tag-based filtering** to control which flags appear in the workspace admin UI:

| Flag Type | Unleash Tag | Visible in Workspace UI | Controllable By |
|-----------|-------------|------------------------|-----------------|
| Workspace-configurable | `scope: workspace` | ✅ Yes | Workspace admins + Platform team |
| Platform-only | (no tag) | ❌ No | Platform team only (via Unleash UI) |

**When to use each:**

| Use Case | Flag Type | Rationale |
|----------|-----------|-----------|
| Beta features users can opt into | Workspace-configurable | User choice |
| Experimental UI changes | Workspace-configurable | Users can revert if issues |
| Infrastructure/operational flags | Platform-only | Requires platform expertise |
| Security-related flags | Platform-only | Must be centrally controlled |
| Gradual rollouts (A/B tests) | Platform-only | Platform controls rollout % |
| Kill switches | Platform-only | Emergency platform control |

**Adding the tag in Unleash:**
1. Navigate to the feature flag in Unleash UI
2. Click "Add tag"
3. Type: `scope`, Value: `workspace`
4. Save

**Filtering logic in backend:**
```go
// Only include flags with scope:workspace tag
func isWorkspaceConfigurable(tags []Tag) bool {
    tagType := getEnvOrDefault("UNLEASH_WORKSPACE_TAG_TYPE", "scope")
    tagValue := getEnvOrDefault("UNLEASH_WORKSPACE_TAG_VALUE", "workspace")

    for _, tag := range tags {
        if tag.Type == tagType && tag.Value == tagValue {
            return true
        }
    }
    return false
}
```

### API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/projects/:name/feature-flags` | GET | List all flags with override status |
| `/projects/:name/feature-flags/evaluate/:flagName` | GET | Evaluate flag for workspace |
| `/projects/:name/feature-flags/:flagName/override` | PUT | Set workspace override |
| `/projects/:name/feature-flags/:flagName/override` | DELETE | Remove workspace override |

### Key Files

**Backend:**

* `handlers/featureflags.go` - Client API proxy for frontend SDK
* `handlers/featureflags_admin.go` - Flag evaluation, override management
* `routes.go` - Route registration

**Frontend:**

* `src/lib/feature-flags.ts` - Re-exports Unleash SDK hooks (`useFlag`, `useVariant`)
* `src/components/providers/feature-flag-provider.tsx` - Unleash provider with environment context
* `src/components/workspace-sections/feature-flags-settings.tsx` - Admin UI (in Settings tab) with batch save
* `src/services/queries/use-feature-flags-admin.ts` - React Query hooks including `useWorkspaceFlag`
* `src/services/api/feature-flags-admin.ts` - API service functions
* `src/app/api/projects/[name]/feature-flags/*` - Next.js proxy routes

**Deployment:**

* `e2e/scripts/deploy-unleash.sh` - Unleash + PostgreSQL deployment script
* `Makefile:deploy-unleash*` - Deployment targets for KinD/CRC

### Environment Variables

| Variable | Component | Description |
|----------|-----------|-------------|
| `UNLEASH_URL` | Backend | Unleash server URL |
| `UNLEASH_CLIENT_KEY` | Backend | Client API token (read-only) |
| `UNLEASH_ADMIN_URL` | Backend | Unleash Admin API URL |
| `UNLEASH_ADMIN_TOKEN` | Backend | Admin API token (read-write) |
| `UNLEASH_PROJECT` | Backend | Unleash project ID (default: "default") |
| `UNLEASH_ENVIRONMENT` | Backend | Target environment (default: "development") |
| `UNLEASH_WORKSPACE_TAG_TYPE` | Backend | Tag type for workspace-configurable flags (default: "scope") |
| `UNLEASH_WORKSPACE_TAG_VALUE` | Backend | Tag value for workspace-configurable flags (default: "workspace") |
| `NEXT_PUBLIC_UNLEASH_ENV_CONTEXT_FIELD` | Frontend | Environment value sent in SDK context (default: "development"). Note: This does NOT select the Unleash environment—that's determined by the token scope. Used for strategy constraints that check `context.environment`. |

### Patterns Established

**Pattern 1: Workspace-Scoped Flag Evaluation**

```go
func EvaluateFeatureFlag(c *gin.Context) {
    namespace := c.Param("projectName")
    flagName := c.Param("flagName")

    // 1. Check ConfigMap for workspace override
    cm, err := k8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, "feature-flag-overrides", metav1.GetOptions{})
    if err == nil {
        if override, exists := cm.Data[flagName]; exists {
            enabled := override == "true"
            c.JSON(http.StatusOK, gin.H{"flag": flagName, "enabled": enabled, "source": "workspace-override"})
            return
        }
    }

    // 2. Fall back to Unleash
    enabled := unleashClient.IsEnabled(flagName, unleash.WithContext(unleash.Context{
        Properties: map[string]string{"workspace": namespace},
    }))
    c.JSON(http.StatusOK, gin.H{"flag": flagName, "enabled": enabled, "source": "unleash"})
}
```

**Pattern 2: Setting Workspace Override**

```go
func SetFeatureFlagOverride(c *gin.Context) {
    namespace := c.Param("projectName")
    flagName := c.Param("flagName")

    var req struct {
        Enabled bool `json:"enabled"`
    }
    c.ShouldBindJSON(&req)

    // Get or create ConfigMap
    cm, err := k8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, "feature-flag-overrides", metav1.GetOptions{})
    if errors.IsNotFound(err) {
        cm = &corev1.ConfigMap{
            ObjectMeta: metav1.ObjectMeta{
                Name:      "feature-flag-overrides",
                Namespace: namespace,
            },
            Data: map[string]string{},
        }
        cm, err = k8sClient.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
    }

    // Set override
    if cm.Data == nil {
        cm.Data = map[string]string{}
    }
    cm.Data[flagName] = strconv.FormatBool(req.Enabled)

    _, err = k8sClient.CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{})
    c.JSON(http.StatusOK, gin.H{"message": "Override set", "flag": flagName, "enabled": req.Enabled})
}
```

**Pattern 3: Frontend Workspace Flag Hook**

```typescript
import { useQuery } from "@tanstack/react-query";

export function useWorkspaceFlag(projectName: string, flagName: string) {
  const { data, isLoading } = useQuery({
    queryKey: ["feature-flag", projectName, flagName],
    queryFn: async () => {
      const res = await fetch(`/api/projects/${projectName}/feature-flags/evaluate/${flagName}`);
      return res.json();
    },
    staleTime: 15000, // 15s cache
    enabled: !!projectName && !!flagName,
  });

  return {
    enabled: data?.enabled ?? false,
    source: data?.source,
    isLoading,
  };
}
```

**Pattern 4: Batch Save for Admin UI**

The Feature Flags admin UI (located in Workspace Settings tab) uses a batch save pattern to prevent excessive ConfigMap updates:

1. **Local state tracking**: Toggle changes are tracked in React state, not immediately saved
2. **Visual indicators**: "Unsaved" and "Will Reset" badges show pending changes
3. **Batch operations**: Save button commits all changes (toggles + resets) in parallel
4. **Discard option**: Users can revert all pending changes without saving

This pattern prevents ConfigMap update spam when users toggle multiple flags and provides a familiar "Save/Discard" UX consistent with other Settings sections.

## Validation

**Functional Testing:**

* Workspace override takes precedence over Unleash global state
* Removing override reverts to Unleash default
* A/B testing works when no override is set
* Gradual rollout respects workspace context
* `useWorkspaceFlag()` hook returns correct values
* Only flags with `scope: workspace` tag appear in workspace admin UI
* Platform-only flags (without tag) are hidden from workspace admin UI

**Security Testing:**

* Admin API credentials not exposed to frontend
* User authorization validated before override operations
* ConfigMap access restricted to workspace namespace

**Deployment Verification:**

```bash
# Deploy Unleash to cluster
make deploy-unleash-kind  # or deploy-unleash-openshift

# Verify deployment
make unleash-status

# Port-forward for local access
make unleash-port-forward
# Access at http://localhost:4242
```

## Links

* [Unleash Documentation](https://docs.getunleash.io/)
* [Unleash React SDK](https://docs.getunleash.io/reference/sdks/react)
* [Unleash Admin API](https://docs.getunleash.io/reference/api/unleash/admin)
* [Kubernetes ConfigMaps](https://kubernetes.io/docs/concepts/configuration/configmap/)
* Constitution Principle XI: Feature Flag Discipline
* Related: docs/feature-flags/README.md
