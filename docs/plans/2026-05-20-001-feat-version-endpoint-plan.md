---
title: "feat: Add unauthenticated /version endpoint to ambient-api-server"
type: feat
status: active
date: 2026-05-20
---

# feat: Add unauthenticated /version endpoint to ambient-api-server

## Overview

The ambient-api-server needs a `/version` endpoint that returns build metadata (git SHA, build time, git tag) without requiring authentication. The Makefile already has ldflags infrastructure but the target Go variables don't exist yet. The endpoint should be consumable by CLI (`acpctl version`) and both Go and Python SDKs.

---

## Problem Frame

Operators and developers have no way to verify which version of the API server is running. The Makefile injects build metadata via ldflags, but the Go variables that receive those values don't exist, and there is no HTTP endpoint to expose them. This blocks basic operational needs: deployment verification, debugging version mismatches, and CLI client/server version comparison.

---

## Requirements Trace

- R1. Expose `GET /api/ambient/v1/version` returning `version`, `build_time`, `git_tag`, and `api_version` as JSON
- R2. The endpoint must be unauthenticated — no bearer token required
- R3. Build metadata must be injected at compile time via ldflags (Makefile and Dockerfile)
- R4. The Go SDK must expose both an authenticated client method and a standalone function (for use without a full client)
- R5. The Python SDK must expose a standalone `fetch_server_version()` function
- R6. `acpctl version` must show both client and server version in a single command
- R7. The response must be pre-marshaled at startup to avoid per-request serialization overhead

---

## Scope Boundaries

- No CRD changes
- No database changes
- No authentication middleware changes beyond bypassing auth for this single path
- No frontend changes (CLI and SDK only)

---

## Context & Research

### Relevant Code and Patterns

- `components/ambient-api-server/plugins/proxy/plugin.go` — existing `RegisterPreAuthMiddleware` pattern for unauthenticated paths
- `components/ambient-api-server/cmd/ambient-api-server/main.go` — side-effect plugin imports
- `components/ambient-api-server/pkg/api/api.go` — package-level vars for ldflags targets
- `components/ambient-api-server/Makefile` — existing ldflags injection for `Version` and `BuildTime`
- `components/ambient-cli/cmd/acpctl/version/cmd.go` — existing CLI version command (client-only)
- `components/ambient-sdk/go-sdk/client/` — Go SDK client structure
- `components/ambient-sdk/python-sdk/ambient_platform/` — Python SDK structure
- `pkgserver.RegisterPreAuthMiddleware` from rh-trex-ai — registers middleware that runs before auth, used by the proxy plugin for the same purpose

### Institutional Learnings

- The plugin system uses `init()` side-effects — import the plugin package with `_` in `main.go` and it self-registers
- Pre-auth middleware is the correct hook for unauthenticated endpoints in the rh-trex-ai framework

---

## Key Technical Decisions

- **Pre-auth middleware, not a route handler**: The version endpoint must bypass authentication entirely. Using `RegisterPreAuthMiddleware` (same pattern as the proxy plugin) intercepts the request before the auth stack runs. This avoids modifying the auth middleware's bypass list.
- **Pre-marshaled JSON response**: Marshal the response once at startup into a `[]byte` and write it directly on each request. Eliminates per-request allocation and serialization for a response that never changes.
- **Standalone SDK function alongside client method**: The CLI needs to fetch server version without constructing a full authenticated SDK client. Provide both `Client.ServerVersion()` (for authenticated clients) and `FetchServerVersion()` (standalone, no auth needed).
- **`git describe --tags --always --dirty` for git_tag**: Produces a human-readable version string that includes the nearest tag, distance from it, and dirty state.

---

## Open Questions

### Resolved During Planning

- **Where to add the endpoint?**: As a new plugin (`plugins/version/`) following the existing plugin pattern, not inline in an existing plugin.
- **How to bypass auth?**: `RegisterPreAuthMiddleware` — matches the proxy plugin pattern exactly.

### Deferred to Implementation

- None — this is a straightforward, well-patterned change.

---do

## Implementation Units

- U1. **Declare ldflags target variables in api package**

**Goal:** Create the Go variables that `go build -ldflags -X` will populate with build metadata.

**Requirements:** R3

**Dependencies:** None

**Files:**
- Modify: `components/ambient-api-server/pkg/api/api.go`

**Approach:**
- Add package-level `var` block with `Version`, `BuildTime`, and `GitTag` string variables, initialized to empty strings
- These are the targets for `-X` ldflags in the Makefile and Dockerfile

**Patterns to follow:**
- Existing `var NewID` declaration in the same file

**Test scenarios:**
- Test expectation: none — pure variable declarations with no behavior

**Verification:**
- `go build` succeeds with the new variables
- `go vet ./...` passes

---

- U2. **Create version plugin with pre-auth middleware**

**Goal:** Serve `GET /api/ambient/v1/version` as a JSON response without authentication.

**Requirements:** R1, R2, R7

**Dependencies:** U1

**Files:**
- Create: `components/ambient-api-server/plugins/version/plugin.go`
- Modify: `components/ambient-api-server/cmd/ambient-api-server/main.go`

**Approach:**
- Define a `versionResponse` struct with `json` tags matching the wire format
- In `init()`, marshal the response once into a package-level `[]byte`
- Register a `PreAuthMiddleware` that matches `GET` on `/api/ambient/v1/version` (with trailing slash tolerance via `strings.TrimSuffix`), writes the pre-marshaled bytes with `Content-Type: application/json`, and calls `next.ServeHTTP` for all other requests
- Add `_ "github.com/ambient-code/platform/components/ambient-api-server/plugins/version"` to `main.go` imports

**Patterns to follow:**
- `plugins/proxy/plugin.go` — same `RegisterPreAuthMiddleware` pattern, same `init()` side-effect import

**Test scenarios:**
- Happy path: `GET /api/ambient/v1/version` returns 200 with JSON body containing `version`, `build_time`, `git_tag` fields
- Happy path: Response `Content-Type` is `application/json`
- Happy path: Endpoint is accessible without an `Authorization` header
- Edge case: `GET /api/ambient/v1/version/` (trailing slash) returns the same response
- Edge case: `POST /api/ambient/v1/version` falls through to the next handler (not intercepted)

**Verification:**
- `curl http://localhost:8000/api/ambient/v1/version` returns valid JSON with all three fields
- No auth token required

---

- U3. **Update Makefile and Dockerfile to inject git_tag**

**Goal:** Pass `GitTag` via ldflags at build time, both locally and in container builds.

**Requirements:** R3

**Dependencies:** U1

**Files:**
- Modify: `components/ambient-api-server/Makefile`
- Modify: `components/ambient-api-server/Dockerfile`

**Approach:**
- Makefile: Add `git_tag` variable using `git describe --tags --always --dirty`, append `-X ...GitTag=$(git_tag)` to `ldflags`
- Makefile: Update `build-image` target to pass `--build-arg GIT_VERSION`, `--build-arg BUILD_TIME`, `--build-arg GIT_TAG`
- Dockerfile: Add `ARG` declarations for `GIT_VERSION`, `BUILD_TIME`, `GIT_TAG`, and expand the `go build -ldflags` line to include all three `-X` flags

**Patterns to follow:**
- Existing `build_version` and `build_time` Makefile variables

**Test scenarios:**
- Happy path: `make binary` produces a binary; running it and hitting `/version` shows non-empty `version` and `build_time`
- Happy path: `make build-image` passes build args and the container's `/version` endpoint returns populated fields
- Edge case: Building from a detached HEAD (no tags) — `git describe --tags --always` falls back to the short SHA

**Verification:**
- `make binary && ./ambient-api-server` serves a version endpoint with populated build metadata

---

- U4. **Add Go SDK version client**

**Goal:** Provide Go SDK consumers with both authenticated and standalone methods to fetch server version.

**Requirements:** R4

**Dependencies:** U2

**Files:**
- Create: `components/ambient-sdk/go-sdk/client/version_api.go`

**Approach:**
- `ServerVersion` struct with `json` tags matching the wire format
- `Client.ServerVersion(ctx)` method using the existing `c.do()` helper for authenticated requests
- `FetchServerVersion(ctx, baseURL, insecureSkipVerify)` standalone function that constructs the full URL, makes a plain HTTP GET, and unmarshals the response — no auth required
- The standalone function constructs its own `http.Client` with optional TLS skip and a 10s timeout

**Patterns to follow:**
- Existing `Client.do()` method pattern in the Go SDK

**Test scenarios:**
- Happy path: `FetchServerVersion` against a running server returns a populated `ServerVersion`
- Error path: `FetchServerVersion` against an unreachable host returns a wrapped error
- Error path: Server returns non-200 status — function returns an error with the status code
- Edge case: `baseURL` with trailing slash is handled (trimmed before appending path)

**Verification:**
- `go build ./...` in `go-sdk/` succeeds
- `go vet ./...` passes

---

- U5. **Add Python SDK version function**

**Goal:** Provide Python SDK consumers with a standalone function to fetch server version.

**Requirements:** R5

**Dependencies:** U2

**Files:**
- Create: `components/ambient-sdk/python-sdk/ambient_platform/_version_api.py`

**Approach:**
- `ServerVersion` frozen dataclass with `version`, `build_time`, `git_tag` fields
- `from_dict` classmethod for safe deserialization with defaults
- `fetch_server_version(base_url, timeout, verify_ssl)` function using `httpx.Client`

**Patterns to follow:**
- Existing Python SDK module structure in `ambient_platform/`

**Test scenarios:**
- Happy path: `fetch_server_version` against a running server returns a populated `ServerVersion`
- Error path: Unreachable host raises `httpx.ConnectError`
- Error path: Non-200 response raises `httpx.HTTPStatusError`

**Verification:**
- Module imports without error: `python -c "from ambient_platform._version_api import fetch_server_version"`

---

- U6. **Update acpctl version command to show server version**

**Goal:** `acpctl version` displays both client build info and server version in one output.

**Requirements:** R6

**Dependencies:** U4

**Files:**
- Modify: `components/ambient-cli/cmd/acpctl/version/cmd.go`

**Approach:**
- Change output prefix from `acpctl` to `Client:` for the existing client version line
- After printing client info, load CLI config to get the API URL
- If API URL is available, call `sdkclient.FetchServerVersion` with a 5-second timeout
- Print `Server: <version> (tag: <git_tag>, built: <build_time>)` on success, or `Server: unavailable (<error>)` on failure
- Gracefully degrade: if no config or no API URL, silently skip the server line

**Patterns to follow:**
- Existing `config.Load()` and `cfg.GetAPIUrl()` usage in other CLI commands

**Test scenarios:**
- Happy path: With a running server, `acpctl version` prints both `Client:` and `Server:` lines
- Edge case: No config file — only `Client:` line is printed (no error)
- Edge case: Config exists but no API URL — only `Client:` line
- Error path: Server unreachable — prints `Server: unavailable (...)` with the error message
- Happy path: Server version fields are formatted as `Server: <version> (tag: <tag>, built: <time>)`

**Verification:**
- `acpctl version` produces two-line output when connected to a running server

---

## System-Wide Impact

- **Interaction graph:** The version plugin registers as pre-auth middleware alongside the proxy plugin. Order between pre-auth middlewares does not matter since they match on disjoint paths.
- **Error propagation:** The endpoint has no failure modes — it writes a pre-computed byte slice. The CLI and SDK functions handle connection errors locally.
- **API surface parity:** Go SDK, Python SDK, and CLI all consume the same endpoint. No frontend changes needed.
- **Unchanged invariants:** All existing authenticated endpoints, the proxy plugin, and the auth middleware are unaffected. The version plugin only intercepts `GET /api/ambient/v1/version`.

---

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Pre-auth middleware ordering conflicts with proxy | Both plugins match disjoint URL paths — no conflict possible |
| ldflags variables empty in dev builds | Variables default to empty string; endpoint still returns valid JSON with empty fields |

---

## Sources & References

- Related issue: #1598
- Related code: `components/ambient-api-server/plugins/proxy/plugin.go` (pre-auth middleware pattern)
- Framework: `github.com/openshift-online/rh-trex-ai/pkg/server.RegisterPreAuthMiddleware`
