# Feature Flag Fail Modes

How each feature flag evaluation method behaves when Unleash is unavailable or not configured, which code paths use each method, and the full evaluation precedence.

## Evaluation Methods

### Fail-Closed (default: `false`)

These methods return `false` when Unleash is not configured or unreachable. Features gated behind these methods are **off by default** and require Unleash to be running and the flag to be explicitly enabled.

| Method | Location | Signature |
|--------|----------|-----------|
| `IsEnabled` | `featureflags/featureflags.go:44` | `IsEnabled(flagName string) bool` |
| `IsEnabledWithContext` | `featureflags/featureflags.go:54` | `IsEnabledWithContext(flagName, userID, sessionID, remoteAddress string) bool` |
| `FeatureEnabled` | `handlers/featureflags.go:15` | `FeatureEnabled(flagName string) bool` |
| `FeatureEnabledForRequest` | `handlers/featureflags.go:22` | `FeatureEnabledForRequest(c *gin.Context, flagName string) bool` |

`FeatureEnabled` and `FeatureEnabledForRequest` are convenience wrappers in the `handlers` package. `FeatureEnabled` calls `IsEnabled` directly. `FeatureEnabledForRequest` extracts user ID, session ID, and client IP from the Gin context and passes them to `IsEnabledWithContext`. They exist so handlers don't need to import the `featureflags` package.

**Use for:** General features, runner type gates, experimental functionality, kill switches. If Unleash goes down, these features turn off.

### Fail-Open (default: `true`)

These methods return `true` when Unleash is not configured or unreachable. Features gated behind these methods are **on by default** and require Unleash to be running with the flag explicitly disabled to restrict them.

| Method | Location | Signature |
|--------|----------|-----------|
| `IsModelEnabled` | `featureflags/featureflags.go:69` | `IsModelEnabled(flagName string) bool` |
| `IsModelEnabledWithContext` | `featureflags/featureflags.go:78` | `IsModelEnabledWithContext(flagName, userID, sessionID, remoteAddress string) bool` |

**Use for:** Model availability flags. If Unleash goes down, all models remain available. The rationale is that blocking session creation due to flag infrastructure failure is worse than temporarily losing the ability to restrict a model.

### Workspace-Aware Wrappers

These methods check the workspace ConfigMap override first, then fall back to one of the above methods. They inherit the fail mode of their fallback.

| Wrapper | Fallback | Fail Mode | Location |
|---------|----------|-----------|----------|
| `isModelEnabledWithOverrides` | `IsModelEnabled` | Fail-open | `handlers/models.go:149` |
| `isRunnerEnabledWithOverrides` | `FeatureEnabled` | Fail-closed | `handlers/runner_types.go:218` |

## Where Each Method Is Used

### `FeatureEnabled` / `IsEnabled` (fail-closed)

| Caller | File | Purpose |
|--------|------|---------|
| `isRunnerEnabled` | `handlers/runner_types.go:208,213` | Check if a runner type is enabled via its feature gate |
| `GetRunnerTypesGlobal` | `handlers/runner_types.go:245` | Filter runners for admin listing (no workspace context) |

### `FeatureEnabledForRequest` / `IsEnabledWithContext` (fail-closed)

| Caller | File | Purpose |
|--------|------|---------|
| `EvaluateFeatureFlag` | `handlers/featureflags_admin.go:333` | Unleash fallback when no workspace ConfigMap override exists |

### `IsModelEnabled` (fail-open)

| Caller | File | Purpose |
|--------|------|---------|
| `isModelEnabledWithOverrides` | `handlers/models.go:155` | Fallback when no workspace ConfigMap override exists for a model flag |

### `isModelEnabledWithOverrides` (fail-open via `IsModelEnabled`)

| Caller | File | Purpose |
|--------|------|---------|
| `ListModelsForProject` | `handlers/models.go:123` | Filter feature-gated models in the model list endpoint |
| `isModelAvailable` | `handlers/models.go:237` | Validate model is enabled during session creation |

### `isRunnerEnabledWithOverrides` (fail-closed via `FeatureEnabled`)

| Caller | File | Purpose |
|--------|------|---------|
| `GetRunnerTypes` | `handlers/runner_types.go:280` | Filter runners by workspace-scoped feature flags |

## Evaluation Precedence

When a flag is evaluated for a workspace, the system checks three layers in order:

```
1. Workspace ConfigMap override  (highest priority)
   ConfigMap "feature-flag-overrides" in the workspace namespace
   Key: flag name, Value: "true" or "false"
   If present -> return that value, source: "workspace-override"

2. Unleash SDK evaluation  (middle priority)
   Respects strategies, rollout percentages, A/B tests
   User/session/IP context passed when available
   If Unleash is configured -> return SDK result, source: "unleash"

3. Code default  (lowest priority, Unleash not configured)
   General flags (IsEnabled): false  (fail-closed)
   Model flags (IsModelEnabled): true  (fail-open)
```

### Precedence Table

| ConfigMap Override | Unleash State | General Flag Result | Model Flag Result |
|--------------------|---------------|---------------------|-------------------|
| `"true"` | (any) | `true` | `true` |
| `"false"` | (any) | `false` | `false` |
| (not set) | enabled | `true` | `true` |
| (not set) | disabled | `false` | `false` |
| (not set) | 50% rollout | (evaluated per context) | (evaluated per context) |
| (not set) | (not configured) | `false` | `true` |
| (not set) | (unreachable) | `false` | `true` |

## Frontend Fail Behavior

The frontend has its own fail behavior independent of the backend:

| Component | Behavior When Unleash Unavailable |
|-----------|----------------------------------|
| Next.js proxy (`/api/feature-flags`) | Returns `{ toggles: [] }` — all client-side flags `false` |
| `useFlag()` hook | Returns `false` (no toggles loaded) |
| `useVariant()` hook | Returns disabled variant |
| Feature Flags Admin UI | Shows "Feature Flags Not Available" message |
| Workspace flag evaluation (`/evaluate/:flagName`) | Falls through to backend fail mode (closed or open depending on method) |

## Special Cases

### Default Runner Fail-Open

The default runner (`claude-code`) has a special fail-open path in `isRunnerEnabled` (`handlers/runner_types.go:204`). When the agent registry is unavailable, the default runner returns `true` to prevent blocking all session creation during cold start.

### Default Model Bypass

Default models (global `defaultModel` and per-provider `providerDefaults` from `models.json`) bypass feature flag checks entirely. They are always available regardless of flag state. This is enforced in both `ListModelsForProject` and `isModelAvailable`.

### Model Manifest Unavailable

When `models.json` cannot be read and no cached version exists:
- If a `requiredProvider` is specified (runner knows its provider): model is **rejected** to prevent cross-provider mismatches
- If no `requiredProvider` (registry also unavailable): model is **allowed** (fail-open for cold start)

## Flag Naming Conventions

| Category | Pattern | Example | Fail Mode |
|----------|---------|---------|-----------|
| Model flags | `model.<modelId>.enabled` | `model.claude-opus-4-6.enabled` | Fail-open |
| Runner flags | `runner.<runnerId>.enabled` | `runner.gemini-cli.enabled` | Fail-closed |
| General flags | `<component>.<feature>.<aspect>` | `frontend.file-explorer.enabled` | Fail-closed |
