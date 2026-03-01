# ADR-0007: Application-Level Configuration Management

**Status:** Proposed
**Date:** 2026-03-01
**Deciders:** Platform Team
**Technical Story:** [PR #604](https://github.com/ambient-code/platform/pull/604)

## Context and Problem Statement

~15 operational settings (timeouts, model selection, retry config) are hardcoded across the platform. Tuning requires code changes and redeployment. Platform admins lack self-service tooling. PR #604 externalized container resources to env vars—we need a broader strategy.

## Decision Drivers

* **Runtime tuning** — operational changes without deployments
* **Admin self-service** — UI for platform and workspace admins
* **K8s-native** — extend existing ConfigMap/RBAC patterns

## Considered Options

| Option | Pros | Cons |
|--------|------|------|
| **Env vars only** (PR #604) | Simple, proven | Requires restart for all changes |
| **Runtime ConfigMap + UI** | No restart, admin UI | New endpoints/UI to build |
| **Database-backed** | Full audit trail | Over-engineered, new dependency |

## Decision Outcome

**Hybrid approach:** Env vars for infrastructure (restart acceptable), runtime ConfigMap for operations (restart unacceptable).

| Setting Type | Examples | Storage | Admin UX | Restart |
|--------------|----------|---------|----------|:-------:|
| Infrastructure | Container resources, images, K8s QPS | Env var | ArgoCD | Yes |
| Operational | Timeouts, retry config, model selection | ConfigMap | UI | No |
| Workspace override | Per-project tuning | ConfigMap | UI | No |

**Evaluation order:** Workspace override → Platform default → Hardcoded fallback

### Consequences

| | Impact |
|---|--------|
| **+** | Runtime tuning without restarts; admin self-service; reuses feature flags pattern |
| **−** | Two config mechanisms; new UI/endpoints to build |
| **Risk** | Cache staleness (mitigated: 30s TTL + invalidation endpoint) |

> **ArgoCD note:** `platform-settings` ConfigMap uses `argocd.argoproj.io/compare-options: IgnoreExtraneous` annotation to prevent sync conflicts when modified via UI.

## Implementation

| Phase | Scope | Deliverables |
|-------|-------|--------------|
| 1 | Deployment-time | Env vars: `K8S_CLIENT_QPS`, `PARALLEL_SSAR_WORKER_COUNT`, secret names |
| 2 | Runtime platform | `platform-settings` ConfigMap, `/admin/settings` UI, `LoadPlatformSettings()` |
| 3 | Workspace overrides | `workspace-settings` ConfigMap, extend workspace settings UI |

**Key changes:**
* `config/config.go` — cached ConfigMap reader with 30s TTL
* `handlers/platform_settings.go` — CRUD endpoints, RBAC via SelfSubjectAccessReview
* `frontend/admin/settings` — platform admin UI
* `frontend/settings-section.tsx` — "Platform Default" vs "Override" badges

## Validation

* Timeout change applies in <30s without pod restart
* Workspace admin can override model selection
* Platform admin UI accessible only to cluster-admins

## Links

* [PR #604: Externalize container resources](https://github.com/ambient-code/platform/pull/604)
* [ADR-0001: Kubernetes-Native Architecture](0001-kubernetes-native-architecture.md)
