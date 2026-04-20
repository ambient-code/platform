# Proposal: Alpha-to-Main Migration Plan

**Date:** 2026-04-17
**Branch:** `chore/alpha-to-main-migration` (from `alpha`)
**Target:** `main`
**Status:** In Progress

---

## Summary

Migrate 62 commits (~61K insertions, ~5.7K deletions across 371 files) from the
`alpha` branch back to `main` in a series of additive, non-breaking pull requests.
Each PR lands independently, compiles, and passes tests against `main`.

This document is the working checklist. It ships in **PR 1** (combined with docs,
skills, and Claude config) and is updated with each subsequent merge. The final PR
removes this file.

---

## Component Delta

| Component | Files Changed | +Lines | -Lines | Dependency Tier |
|---|---|---|---|---|
| ambient-api-server | 109 | 15,970 | 3,964 | T0 — Foundation |
| ambient-sdk | 80 | 15,584 | 844 | T1 — Depends on api-server API |
| ambient-control-plane | 21 | 4,657 | 0 | T1 — Depends on api-server API |
| ambient-cli | 39 | 7,177 | 372 | T2 — Depends on SDK |
| runners | 39 | 4,952 | 163 | T2 — Depends on CP + api-server |
| manifests | 30 | 990 | 0 | T3 — Deploys all components |
| docs / skills / .claude | ~53 | ~8,500 | ~400 | Independent |

---

## PR Checklist

### PR 1 — Migration Plan + Docs, Skills, and Claude Config
> Zero code risk. Safe to merge immediately. Combines the migration plan with all
> non-code documentation, skills, and config changes.

- [x] Analyze alpha→main delta and component dependencies
- [x] Write migration plan (`docs/internal/proposals/alpha-to-main-migration.md`)
- [x] Fix alpha→main branch references in `.claude/skills/devflow/SKILL.md`
- [ ] `.claude/skills/ambient/SKILL.md`
- [ ] `.claude/skills/ambient-pr-test/SKILL.md`
- [ ] `.claude/skills/grpc-dev/SKILL.md`
- [ ] `.claude/settings.json` updates
- [ ] `CLAUDE.md` project-level updates
- [ ] `docs/internal/design/` — specs and guides:
  - [ ] `README.md`
  - [ ] `ambient-model.guide.md`
  - [ ] `ambient-model.spec.md`
  - [ ] `control-plane.guide.md`
  - [ ] `control-plane.spec.md`
  - [ ] `frontend-backend-migration-plan.md`
  - [ ] `frontend-to-api-status.md`
  - [ ] `mcp-server.guide.md`
  - [ ] `mcp-server.spec.md`
  - [ ] `runner.spec.md`
- [ ] `docs/internal/developer/local-development/openshift.md`
- [ ] Update this checklist
- [ ] Merge to main

### PR 2 — ambient-api-server: OpenAPI Specs, Generated Client, New Kinds
> Foundation PR. All other components depend on its API surface.

- [ ] New OpenAPI specs:
  - [ ] `openapi/openapi.credentials.yaml`
  - [ ] `openapi/openapi.inbox.yaml`
  - [ ] `openapi/openapi.sessions.yaml` additions
  - [ ] `openapi/openapi.agents.yaml` changes
  - [ ] `openapi/openapi.projects.yaml` changes
  - [ ] `openapi/openapi.yaml` root spec updates
- [ ] Generated Go client (`pkg/api/openapi/`) — regenerate, do not hand-edit
- [ ] Proto definitions:
  - [ ] `proto/ambient/v1/inbox.proto`
  - [ ] `proto/ambient/v1/sessions.proto` changes
- [ ] Generated proto Go code (`pkg/api/grpc/ambient/v1/`)
- [ ] New plugins:
  - [ ] `plugins/credentials/` (model, handler, service, dao, presenter, migration, tests)
  - [ ] `plugins/inbox/` (model, handler, service, dao, presenter, migration, tests)
- [ ] Service layer additions:
  - [ ] `plugins/sessions/service.go` — `ActiveByAgentID`, `Start`, `Stop`
  - [ ] `plugins/sessions/presenter.go` — new fields
- [ ] `cmd/ambient-api-server/main.go` — new plugin imports
- [ ] `Makefile` updates
- [ ] Verify: `make test` passes
- [ ] Verify: `golangci-lint run` passes
- [ ] Update this checklist
- [ ] Merge to main

### PR 3 — ambient-sdk: Go + TypeScript Client Updates
> Depends on: PR 2 (api-server API surface)

- [ ] Go SDK updates matching new API surface
- [ ] TypeScript SDK updates:
  - [ ] `ts-sdk/src/session_message_api.ts`
  - [ ] `ts-sdk/src/user.ts`
  - [ ] `ts-sdk/src/user_api.ts`
  - [ ] New/updated type definitions
- [ ] Removal of deprecated types (`ProjectAgent`, `ProjectDocument`, `Ignite`)
  - [ ] Verify no main-branch code references removed types before merging
- [ ] New integration tests (`ts-sdk/tests/integration.test.ts`)
- [ ] Verify: SDK builds and tests pass
- [ ] Update this checklist
- [ ] Merge to main

### PR 4 — ambient-control-plane: New Component
> Depends on: PR 2 (api-server API surface). Purely additive (0 deletions).

- [ ] Core control plane:
  - [ ] `cmd/` — entry point
  - [ ] `internal/config/` — configuration
  - [ ] `internal/watcher/watcher.go` — resource watcher
  - [ ] `internal/handlers/` — reconciliation handlers
- [ ] Token server:
  - [ ] `internal/tokenserver/server.go`
  - [ ] `internal/tokenserver/handler.go`
  - [ ] `internal/tokenserver/handler_test.go`
- [ ] Credential injection into runner pods
- [ ] Namespace provisioning
- [ ] Proxy environment forwarding (`HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`)
- [ ] RSA keypair auth for runner token endpoint
- [ ] Exponential backoff retry in informer
- [ ] Verify: `go vet ./...` and `golangci-lint run` pass
- [ ] Update this checklist
- [ ] Merge to main

### PR 5 — ambient-cli: acpctl Enhancements
> Depends on: PR 2 (api-server), PR 3 (SDK)

- [ ] `acpctl login --use-auth-code` — OAuth2 + PKCE flow (RHOAIENG-55812)
- [ ] Agent commands:
  - [ ] `acpctl agent start` with `--all/-A` flag
  - [ ] `acpctl agent stop` with `--all/-A` flag
- [ ] `acpctl session send -f` — follow mode
- [ ] Credential CLI verbs
- [ ] `pkg/config/config.go` — new config fields
- [ ] `pkg/config/token.go` + `token_test.go` — token management
- [ ] `pkg/connection/connection.go` — connection updates
- [ ] Security fixes and idempotent start
- [ ] Verify: `go vet ./...` and `golangci-lint run` pass
- [ ] Update this checklist
- [ ] Merge to main

### PR 6 — runners: Auth, Credentials, gRPC, and SSE
> Depends on: PR 2 (api-server), PR 4 (control-plane token endpoint)

- [ ] Credential system:
  - [ ] `platform/auth.py` — `_fetch_credential` with caller/bot token fallback
  - [ ] `platform/auth.py` — `populate_runtime_credentials`, `clear_runtime_credentials`
  - [ ] `platform/auth.py` — `gh` CLI wrapper (`install_gh_wrapper`)
  - [ ] `platform/auth.py` — `sanitize_user_context`
- [ ] `platform/utils.py` — `get_active_integrations` and helpers
- [ ] `platform/context.py` — `RunnerContext` updates
- [ ] `platform/prompts.py` — prompt additions
- [ ] `middleware/secret_redaction.py` — redaction changes
- [ ] `observability.py` — observability updates
- [ ] `tools/backend_api.py` — API tool updates
- [ ] gRPC transport and delta buffer
- [ ] SSE flush per chunk, unbounded tap queue
- [ ] CP OIDC token for backend credential fetches
- [ ] Tests:
  - [ ] `tests/test_shared_session_credentials.py`
  - [ ] `tests/test_bridge_claude.py`
  - [ ] `tests/test_app_initial_prompt.py`
  - [ ] `tests/test_events_endpoint.py`
  - [ ] `tests/test_grpc_client.py`
  - [ ] `tests/test_grpc_transport.py`
  - [ ] `tests/test_grpc_writer.py`
- [ ] `pyproject.toml` dependency additions
- [ ] Verify: `python -m pytest tests/` passes
- [ ] Update this checklist
- [ ] Merge to main

### PR 7 — manifests: Kustomize Overlays and RBAC
> Depends on: All component PRs (references their images/deployments)

- [ ] `mpp-openshift` overlay:
  - [ ] NetworkPolicy for runner→CP token server
  - [ ] gRPC Route for ambient-api-server
  - [ ] CP token Service + `CP_RUNTIME_NAMESPACE` + `CP_TOKEN_URL`
  - [ ] MCP sidecar image wiring
  - [ ] RBAC (`ambient-control-plane-rbac.yaml`)
  - [ ] RoleBinding namespace fixes via Kustomize replacement
  - [ ] Explicit namespaces per resource
  - [ ] Remove hardcoded preprod hostname from route
- [ ] `production` overlay:
  - [ ] `ambient-api-server-env-patch.yaml`
  - [ ] `ambient-api-server-route.yaml`
  - [ ] `kustomization.yaml` updates (components, patches, images)
- [ ] `openshift-dev` overlay:
  - [ ] `kustomization.yaml`
  - [ ] `ambient-api-server-env-patch.yaml`
- [ ] Verify: `kustomize build` succeeds for each overlay
- [ ] Update this checklist
- [ ] Merge to main

### PR 7.1 — Cleanup
> Final PR. Remove this migration plan.

- [ ] Delete `docs/internal/proposals/alpha-to-main-migration.md`
- [ ] Final verification: main branch matches alpha functionality
- [ ] Merge to main

---

## Ordering Constraints

```
PR 1 (plan + docs/skills) ── no dependencies, merge first            │
PR 2 (api-server) ──┬── foundation, must land before T1/T2           │
PR 3 (sdk) ─────────┤── depends on PR 2                              │
PR 4 (control-plane)┤── depends on PR 2                              │
PR 5 (cli) ─────────┴── depends on PR 2, PR 3                        │
PR 6 (runners) ─────┴── depends on PR 2, PR 4                        │
PR 7 (manifests) ───┴── depends on all component PRs                 │
PR 7.1 (cleanup) ───────────────────────────────────────────────────  │
```

PR 3 and PR 4 can merge in parallel once PR 2 lands.
PR 5 and PR 6 can merge in parallel once their dependencies land.

## Risk Mitigation

- **Additive only:** New endpoints and types are added; nothing is removed from main
  until verified unused.
- **Independent compilation:** Each PR must compile and pass tests against the main
  branch state at merge time.
- **SDK deprecation safety:** PR 3 removes `ProjectAgent`/`ProjectDocument`/`Ignite` —
  verify no main-branch consumers reference them before merging.
- **Feature flags:** Behavior changes that could affect existing deployments should be
  gated behind Unleash flags where practical.
- **Manifest ordering:** Manifests land last to avoid referencing images that don't
  exist in main yet.

## Source Commits

Alpha branch contains 62 commits not in main. Key cross-component commits:

| Commit | Scope | Description |
|---|---|---|
| `259fde05` | cli | Agent stop command, `--all/-A` for start and stop |
| `6d61555a` | cli, control-plane | Security fixes, idempotent start, CLI enhancements |
| `73894441` | security, api-server, control-plane | CodeRabbit fixes, deletecollection fallback |
| `063953ff` | credentials | Project-scoped credentials, MCP sidecar token exchange |
| `23002c1d` | mcp-sidecar | RSA-OAEP token exchange for dynamic token refresh |
| `b25c1443` | runner, api, cli | Kubeconfig credential provider for OpenShift MCP auth |
| `b0ed2b8c` | control-plane | RSA keypair auth for runner token endpoint |
| `00c1a24e` | control-plane | CP `/token` endpoint for runner gRPC auth |
| `7c7ea1bb` | api, sdk, cli, mcp | Remove ProjectAgent, ProjectDocument, Ignite |
| `936ea12b` | integration | MPP OpenShift end-to-end integration |
