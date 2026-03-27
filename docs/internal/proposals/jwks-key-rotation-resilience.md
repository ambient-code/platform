# Proposal: JWKS Key Rotation Resilience

**Status:** Draft
**Date:** 2026-03-12
**Affects:** `rh-trex-ai/pkg/auth/middleware.go`, `rh-trex-ai/pkg/server/grpcutil/jwk_provider.go`
**Symptom:** ~120 CP reconnect attempts before the api-server stabilizes after a redeploy

---

## Problem Statement

When the api-server redeploys, its JWKS key cache is cold. The Control Plane's ServiceAccount JWT carries a `kid` that the api-server does not yet recognize. Every CP request fails with `unknown key ID: <kid>` until the key cache is populated with the correct key. At CP retry rates of ~1/second this produces ~120 failures before the system stabilizes — visible as a cascade of auth errors in both the CP logs and the api-server logs.

---

## Root Cause Analysis

There are two independent code paths with two distinct failure modes.

### HTTP path — `JWTHandler` (`pkg/auth/middleware.go`)

The `validateToken` keyfunc (line 150) returns an error immediately on unknown `kid`:

```go
if !exists {
    return nil, fmt.Errorf("unknown key ID: %s", kid)
}
```

No refresh is triggered. The only refresh mechanism is a background goroutine (`refreshKeysLoop`) that ticks every **1 hour**. This means:

- At startup, the server loads whatever JWKS it can reach.
- If the JWKS source is a `JwkCertFile`, **it is never refreshed** — `refreshKeysLoop` only runs when `keysURL` is set.
- If the JWKS source is a URL, unknown `kid`s on any request in the next 59 minutes return 401 with no corrective action.

### gRPC path — `JWKKeyProvider` (`pkg/server/grpcutil/jwk_provider.go`)

The gRPC provider does lazy reload on unknown `kid`, which is better. But `reloadMinWait` is hardcoded to **1 minute**. At 1 retry/second from the CP:

- Unknown `kid` triggers a reload attempt.
- Subsequent requests within 60 seconds skip the reload (cooldown active) and fail immediately.
- After 60 seconds the cooldown expires, one more reload attempt occurs.
- If the JWKS source propagates within ~60 seconds: stabilizes after ~60 retries.
- If propagation takes 60–120 seconds (typical kubelet file sync): stabilizes after ~120 retries.

This matches the observed symptom exactly.

### File source makes both paths worse

In Kubernetes, `JwkCertFile` is typically a volume-mounted Secret or ConfigMap. When the underlying Secret rotates, kubelet propagates the file update in **30–90 seconds**. But:

- HTTP path: file is loaded at pod startup, never re-read.
- gRPC path: file is re-read on reload attempts, but only after the 1-minute cooldown.

Combined effect: the api-server serves 401s for up to 90 seconds after a key rotation even when the correct key is available on disk.

---

## Proposed Fixes

All fixes are in `rh-trex-ai`. The ambient-api-server consumes this library and will benefit from an upstream version bump.

### Fix 1 — "Refresh on unknown kid" in `JWTHandler` (HTTP path)

In `validateToken`, when `kid` is not in the cache, trigger an async refresh and retry the lookup once. A per-handler cooldown prevents hammering the JWKS endpoint on a flood of bad requests.

**Required change in `middleware.go`:**

Add fields to `JWTHandler`:
```go
lastRefresh     time.Time
refreshMu       sync.Mutex
refreshCooldown time.Duration   // default: 30s
```

In the keyfunc, replace the hard error with a refresh-and-retry:
```go
if !exists {
    if j.tryRefresh() {
        j.keysMutex.RLock()
        publicKey, exists = j.publicKeys[kid]
        j.keysMutex.RUnlock()
    }
    if !exists {
        return nil, fmt.Errorf("unknown key ID: %s", kid)
    }
}
```

`tryRefresh()` acquires `refreshMu`, checks `time.Since(lastRefresh) > refreshCooldown`, calls `loadKeys()`, updates `lastRefresh`. Returns true if a refresh was performed.

This matches the behavior `JWKKeyProvider` already has for gRPC.

### Fix 2 — Reduce gRPC cooldown

`reloadMinWait` in `JWKKeyProvider` is hardcoded to 1 minute. Reduce to **15 seconds**. This bounds the worst-case stabilization time to 15 retries (15 seconds at 1/s) once the correct JWKS is available on disk or at the URL.

**Required change in `jwk_provider.go`:**
```go
reloadMinWait: 15 * time.Second,  // was: 1 * time.Minute
```

Make it configurable via `NewJWKKeyProvider(keysURL, keysFile string, opts ...ProviderOption)` so callers can tune it.

### Fix 3 — Periodic refresh for file-backed sources

Currently `refreshKeysLoop` is only started when `keysURL != ""`. Add equivalent refresh for `keysFile` sources so that kubelet-propagated file updates are picked up within a bounded window.

**Required change in `middleware.go`:**
```go
// Start automatic key refresh (both URL and file sources)
go j.refreshKeysLoop()
```

And in `refreshKeysLoop`, reduce the default tick to **5 minutes** (was: 1 hour):
```go
ticker := time.NewTicker(j.refreshInterval)  // default: 5 * time.Minute
```

Expose `refreshInterval` as a configurable field with a `WithRefreshInterval(d time.Duration)` builder method, and add it to `AuthConfig`:
```go
type AuthConfig struct {
    ...
    JwkRefreshInterval time.Duration `json:"jwk_refresh_interval"`
}
```

### Fix 4 — Readiness gate (ambient-api-server, not upstream)

The api-server's `/healthcheck` endpoint currently returns 200 immediately after the HTTP server binds. It should remain `503` until `JWTHandler` has loaded at least one key. This prevents the CP from receiving 401s during the key-loading window after a rolling deploy — instead, the CP's retry hits the old pod until the new pod's readiness probe passes.

This is a change in `ambient-api-server/cmd/ambient-api-server/` to extend the existing healthcheck handler.

---

## Stabilization Timeline After Each Fix

| Scenario | Before fixes | After Fix 1+2 | After all fixes |
|---|---|---|---|
| Cold start, URL source | Up to 3600s (hourly refresh) | ≤ 30s (HTTP cooldown) | ≤ 30s |
| Cold start, file source | ∞ (never refreshed) | ≤ 15s (gRPC cooldown) + 401 on HTTP | ≤ 15s |
| Kubelet file sync delay (60s) | ~120 retries (gRPC), ∞ (HTTP) | ~4 retries (15s gRPC cooldown) | 0 retries (readiness gate) |
| Key rotation mid-session | Up to 3600s | ≤ 30s | ≤ 30s |

---

## Implementation Plan

### Phase 1 — Upstream (rh-trex-ai)

1. Add `refreshCooldown` field and `tryRefresh()` to `JWTHandler`
2. Trigger refresh on unknown `kid` in `validateToken` keyfunc
3. Change `refreshKeysLoop` to run for both file and URL sources
4. Reduce default refresh interval from 1 hour to 5 minutes
5. Expose `WithRefreshInterval(d time.Duration)` builder method
6. Reduce `JWKKeyProvider.reloadMinWait` from 1 minute to 15 seconds
7. Make `reloadMinWait` configurable via `ProviderOption`
8. Add `JwkRefreshInterval` to `AuthConfig` and wire into both builders
9. Add unit tests: unknown kid triggers refresh, cooldown prevents hammering, file-backed refresh on tick

### Phase 2 — ambient-api-server

1. Bump `rh-trex-ai` dependency to the version that includes Phase 1
2. Add readiness gate to healthcheck handler: `503` until JWTHandler has ≥1 key loaded
3. Set `JwkRefreshInterval` per environment:
   - `development`: N/A (JWT disabled)
   - `integration_testing`: N/A (mock server, keys loaded synchronously)
   - `production`: 5 minutes (default)
4. Verify with a soak test: redeploy api-server while CP is running, confirm 0 auth errors after readiness probe passes

---

## What This Does Not Fix

- **Expired CP tokens**: if the CP's ServiceAccount token itself expires, this is a CP concern (token renewal). These fixes only address `kid` lookup failures, not expiry failures.
- **Wrong JWKS URL**: if `JwkCertURL` points to the wrong issuer entirely, no amount of retry will help. Configuration is a deployment concern.
- **CP retry strategy**: the CP's ~1 retry/second is aggressive. Exponential backoff with jitter would be a good CP-side improvement orthogonal to these fixes, but the user has confirmed this is an api-server issue.

---

## References

- `rh-trex-ai/pkg/auth/middleware.go` — `JWTHandler`, `validateToken`, `refreshKeysLoop`
- `rh-trex-ai/pkg/server/grpcutil/jwk_provider.go` — `JWKKeyProvider`, `reloadMinWait`
- `ambient-api-server/cmd/ambient-api-server/environments/` — per-environment JWT config
- `rh-trex-ai/pkg/config/auth.go` — `AuthConfig` struct
