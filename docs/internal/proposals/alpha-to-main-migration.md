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

### PR 1 — Migration Plan + Docs, Skills, and Claude Config ✅ Merged
> Merged as PR #1354.

- [x] Analyze alpha→main delta and component dependencies
- [x] Write migration plan (`docs/internal/proposals/alpha-to-main-migration.md`)
- [x] Fix alpha→main branch references in `.claude/skills/devflow/SKILL.md`
- [x] Merge to main

### PR 2 — ambient-api-server: OpenAPI Specs, Generated Client, New Kinds ✅ Merged
> Merged as PR #1368.

- [x] All items completed
- [x] Merge to main

### PR 3 — ambient-sdk: Go + TypeScript Client Updates ✅ Merged
> Merged as PR #1373. Adapted alpha code to main's generated SDK signatures
> (project-scoped basePath, URL naming). Deferred gRPC MessageWatcher/InboxWatcher
> (proto types not in published module).

- [x] All items completed
- [x] Merge to main

### PR 4 — ambient-control-plane: New Component ✅ Merged
> Merged as PR #1375. Purely additive, 21 new files. Fixed Credential API
> signature (removed projectID param) and gofmt.

- [x] All items completed
- [x] Merge to main

### PR 5 — ambient-cli: acpctl Enhancements ✅ Merged
> Merged as PR #1377. Adapted credential API calls (removed projectID),
> fixed Url→URL naming, replaced WatchSessionMessages with WatchMessages,
> removed agent.Version references, removed dead code for golangci-lint.

- [x] All items completed
- [x] Merge to main

### PR 6 — runners + manifests: Auth, Credentials, gRPC, SSE, and Kustomize Overlays
> Depends on: PR 2 (api-server), PR 4 (control-plane token endpoint).
> Combined with PR 7 (manifests) since all component PRs have landed.

- [x] Credential system (`platform/auth.py`)
- [x] gRPC transport and delta buffer
- [x] SSE flush per chunk, unbounded tap queue
- [x] CP OIDC token for backend credential fetches
- [x] All runner tests pass (707 passed)
- [x] Ruff lint and format clean
- [x] `pyproject.toml` — added `cryptography`, `grpcio`, `protobuf`
- [x] `mpp-openshift` overlay (NetworkPolicy, gRPC Route, CP token, RBAC)
- [x] `production` overlay updates
- [x] `openshift-dev` overlay
- [x] `kustomize build` succeeds for all overlays
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
