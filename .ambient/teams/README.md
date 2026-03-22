# `.ambient/teams` — Agent Team Definitions

This directory defines the AI agent fleet for the Ambient Code Platform using a
Kustomize-style composition model. A base set of agent resources is composed into
environment-specific overlays.

For how annotations, prompts, MCP tools, and runtime state work, see
[`../.ambient/annotation-system.md`](../annotation-system.md).

---

## Directory Structure

```
.ambient/
├── annotation-system.md       # Annotation protocol — shared by all overlays
└── teams/
    ├── README.md              # This file
    ├── base/                  # Full fleet, environment-neutral defaults
    │   ├── kustomization.yaml
    │   ├── project.yaml
    │   ├── lead.yaml
    │   ├── api.yaml
    │   ├── be.yaml
    │   ├── cp.yaml
    │   ├── cli.yaml
    │   ├── fe.yaml
    │   ├── sdk.yaml
    │   ├── runners.yaml
    │   ├── public.yaml
    │   ├── reviewer.yaml
    │   └── infra.yaml
    └── overlays/
        ├── dev/               # Full fleet, all features enabled
        ├── int/               # Backend fleet, integration gates
        ├── stage/             # Delivery fleet, release branch, strict contracts
        └── prod/              # Ops fleet only (lead + reviewer + infra)
```

---

## How Composition Works

Each overlay's `kustomization.yaml` selects a subset of base resources and applies
strategic merge patches to override environment-specific fields. The platform tool
resolves the composed set before applying:

```
base/project.yaml  ──┐
base/lead.yaml     ──┤               overlays/prod/project-patch.yaml ──┐
base/reviewer.yaml ──┼── compose ──► overlays/prod/agents-patch.yaml  ──┼── merge ──► final resources
base/infra.yaml    ──┘               overlays/prod/kustomization.yaml ──┘
       ↑
  (other agents excluded by prod's resource list)
```

Patches are strategic merges: annotation keys are added or overridden individually;
unpatched keys from the base are preserved.

---

## Overlay Summary

| Overlay | Fleet | Branch | Key Differences |
|---|---|---|---|
| `dev` | Full (11 agents) | `main` | `mcp-sidecar` enabled; permissive flags |
| `int` | Backend (9 agents, no fe/public) | `main` | Integration test gates; tighter escalation |
| `stage` | Delivery (8 agents, no fe/cli/be/public) | `release/v0.4.0` | 2-approval gate; e2e required; no new endpoints |
| `prod` | Ops (3 agents: lead/reviewer/infra) | `release/v0.4.0` | Human deploy approval; hotfix-only; all flags off |

---

## Apply an Overlay

```bash
# Preview the composed output
acpctl kustomize overlays/dev/

# Apply an environment
acpctl apply -k overlays/dev/
acpctl apply -k overlays/int/
acpctl apply -k overlays/stage/
acpctl apply -k overlays/prod/

# Bootstrap: project first, then ignite lead
acpctl apply -k overlays/dev/
acpctl ignite lead --prompt "Initial bootstrap ignition."
```

---

## The Two Resource Kinds

### `kind: Project`

One per team. The project `prompt` is injected into every agent session as shared
context. Its annotations hold fleet-wide state: protocol, rules, contracts, bookmarks,
roster, epics. Overlays patch the project to set environment-specific flags and contracts.

### `kind: Agent`

One per agent. The `prompt` is the agent's permanent identity — who it is, what it owns,
what MCP calls it makes at session start. Its annotations hold runtime state. Overlays
may patch agent annotations (e.g. `git.ambient.io/base-branch`) to pin them to a
release branch.

---

## Agent Fleet

| Agent | Owns | dev | int | stage | prod |
|---|---|:---:|:---:|:---:|:---:|
| `lead` | *(orchestrator)* | ✅ | ✅ | ✅ | ✅ |
| `reviewer` | *(cross-cutting)* — `/amber.review` + test gates | ✅ | ✅ | ✅ | ✅ |
| `infra` | `components/manifests/` — builds, deploys | ✅ | ✅ | ✅ | ✅ |
| `api` | `components/ambient-api-server/` — OpenAPI, plugins, gRPC | ✅ | ✅ | ✅ | — |
| `sdk` | `components/ambient-sdk/` — Go/Python/TS SDKs | ✅ | ✅ | ✅ | — |
| `cp` | `components/operator/` — CRD reconcilers, gRPC watch | ✅ | ✅ | ✅ | — |
| `runners` | `components/runners/ambient-runner/` — Python runner | ✅ | ✅ | ✅ | — |
| `be` | `components/backend/` — Gin/K8s backend | ✅ | ✅ | — | — |
| `cli` | `components/ambient-cli/` — acpctl | ✅ | ✅ | — | — |
| `fe` | `components/frontend/` — NextJS + Shadcn | ✅ | — | — | — |
| `public` | `components/public-api/` — HTTP gateway | ✅ | — | — | — |

The three agents present in every environment — `lead`, `reviewer`, `infra` — form the
**ops triad**: orchestration, quality gate, and deployment. They're the minimum viable
fleet for any operational task.

---

## How Agents Interact

Agents communicate exclusively through the inbox. Work flows downstream; blockers flow
upstream. `lead` is the only agent that starts work by igniting agents and sending
assignments.

```
                        ┌──────┐
                        │ lead │
                        └──┬───┘
           assigns          │            gates downstream waves
           ┌────────────────┼─────────────────────────────┐
           │                │                              │
           ▼                ▼                              ▼
        ┌─────┐          ┌─────┐                      ┌──────┐
        │ api │          │ cp  │                      │ infra│
        └──┬──┘          └──┬──┘                      └──────┘
   openapi │         gRPC   │  watch stream                ▲
   change  │                │                       build  │ request
           ▼                ▼                      ────────┘
        ┌─────┐          ┌─────────┐
        │ sdk │          │ runners │
        └──┬──┘          └─────────┘
  regen    │
           ├──────────────► fe
           ├──────────────► cli
           └──────────────► cp  (notified of SDK changes)

  all component agents ──► reviewer  (PR review request)
  reviewer ──────────────► lead      (wave quality status)
```

### Work Assignment Flow

```
lead reads epics
  → sends inbox to component agent
  → agent executes, opens PR, notifies reviewer
  → reviewer runs /amber.review + tests, replies LGTM
  → agent merges, notifies infra (if container changed)
  → infra builds + loads image, notifies lead
  → lead marks wave complete, assigns next wave
```

### Blocker Flow

```
agent blocked on dependency
  → sets ambient.io/blocked="true", writes ambient.io/blocker
  → sends inbox to blocking peer + to lead
  → lead escalates to blocking peer
  → blocking peer resolves, sends inbox to blocked agent
  → blocked agent resumes, clears blocked flag
```

---

## Implementation Pipeline

Changes flow downstream. Agents must not start downstream work against an unstable
upstream. Each wave is gated by the wave above it.

```
Spec (ambient-model.spec.md)
  └─► api    (openapi.yaml, route stubs)
        └─► sdk    (Go/Python/TS SDKs regenerated)
              ├─► be      (handlers, DAOs, migrations)
              ├─► cli     (acpctl commands)
              ├─► cp      (gRPC middleware)
              ├─► runners (Python SDK calls, gRPC push)
              └─► fe      (TypeScript API layer, UI)
```

| Wave | Agents | Gate |
|---|---|---|
| 1 | Human + lead | Spec frozen, gap table agreed |
| 2 | api | openapi.yaml complete; `make test` passes |
| 3 | sdk | All three SDKs regenerated; tests pass |
| 4 | be, cp | Parallel; `make test` + `golangci-lint` clean |
| 5 | cli, cp, runners | Parallel; unblocked after Wave 3 + be |
| 6 | fe | After Wave 4 be |
| 7 | infra + reviewer | Smoke test; image push; `/amber.review` on all components |

---

## Adding a New Overlay

To compose a custom fleet (e.g. a single-agent spike, a security-only fleet, a
read-only observer):

1. Create `overlays/<name>/kustomization.yaml` — list only the base resources you need.
2. Add `project-patch.yaml` — set `labels.env`, `ambient.io/feature-flags`, any
   contract overrides.
3. Optionally add `agents-patch.yaml` — patch all agents (no name filter) or a
   specific agent (name filter) to override annotations like `git.ambient.io/base-branch`.
4. Apply: `acpctl apply -k overlays/<name>/`
