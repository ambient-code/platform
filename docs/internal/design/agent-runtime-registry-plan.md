# Extensible Agent Runtime Registry

**Status**: Approved
**Branch**: `worktree-gemini-runner`
**Date**: 2026-03-03

---

## 1. Problem Statement

Runner type configuration is scattered across 5 layers with hardcoded logic in each:

| Layer | Current Location | What's hardcoded |
|-------|-----------------|-----------------|
| Registry data | `agent-registry-configmap.yaml` | Display names, models |
| State dirs | `runner_types.go` L24-27 | `runnerStateDirs` map |
| Container image | `config.go` L86-90 | `AMBIENT_CODE_RUNNER_IMAGE` env var |
| Bridge factory | `main.py` L7-20 | `if/elif` chain |
| API key list | `settings-section.tsx` L44-46 | `RUNNER_API_KEYS` const |

Adding a new runner requires touching all 5 layers.

## 2. Goals

1. **Single source of truth**: One registry ConfigMap drives all layers
2. **Registry-driven pod specs**: Operator reads container image, resources, state dir from registry
3. **Zero-code runner addition**: New runners via ConfigMap + container image only
4. **Registry management**: Admin UI to view/manage registered runtimes

## 3. Non-Goals

- Direct-to-LLM runner (future — add as a ConfigMap entry + bridge package when ready)
- Shared execution mode (all runners use sandbox — isolated pod per session)
- CRD-based registry (ConfigMap is permanent)
- Warm pool / pre-provisioned pods
- Auto-scaling

---

## 4. Architecture

### 4.1 Conceptual Model

All runners follow the same execution model: **one pod per session** (sandbox). The registry controls what goes into the pod.

```
                        ┌──────────────────────────┐
                        │    AgentRuntime Registry  │
                        │    (ConfigMap)            │
                        └────────────┬─────────────┘
                                     │
                    ┌────────────────┴────────────────┐
                    │                                │
              Claude SDK                        Gemini CLI
              (full sandbox)                    (full sandbox)
                    │                                │
                    ▼                                ▼
              ┌──────────┐                    ┌──────────┐
              │ Init:    │                    │ Init:    │
              │  hydrate │                    │  hydrate │
              │ Main:    │                    │ Main:    │
              │  runner  │                    │  runner  │
              │ Sidecar: │                    │ Sidecar: │
              │  sync    │                    │  sync    │
              └──────────┘                    └──────────┘

Future runners added via ConfigMap + container image.
Lightweight runners can skip init/sidecar via sandbox.seed config.
```

### 4.2 Registry Schema

```jsonc
// agent-registry-configmap.yaml → runtimes.json
[
  {
    // ─── Identity ───
    "id": "claude-agent-sdk",           // unique key, used in CRD spec.runnerType
    "displayName": "Claude Code",
    "description": "Anthropic Claude with full coding capabilities",
    "framework": "claude-agent-sdk",    // maps to RUNNER_TYPE env / bridge key

    // ─── Container Spec ───
    "container": {
      "image": "quay.io/ambient_code/ambient_runner:latest",
      "port": 8001,
      "env": {                          // injected into runner container
        "RUNNER_TYPE": "claude-agent-sdk",
        "RUNNER_STATE_DIR": ".claude"
      },
      "resources": {
        "requests": { "cpu": "500m", "memory": "512Mi" },
        "limits":   { "cpu": "2",    "memory": "4Gi" }
      }
    },

    // ─── Sandbox Config ───
    "sandbox": {
      "stateDir": ".claude",
      "stateSyncImage": "quay.io/ambient_code/state_sync:latest",
      "persistence": "s3",             // "s3" | "none"
      "workspaceSize": "10Gi",
      "terminationGracePeriod": 60,
      "seed": {
        "cloneRepos": true,             // init container clones repos
        "hydrateState": true            // init container restores from S3
      }
    },

    // ─── Auth ───
    "auth": {
      "requiredSecretKeys": ["ANTHROPIC_API_KEY"],
      "secretKeyLogic": "any",          // "any" = at least one, "all" = every key
      "vertexSupported": true
    },

    // ─── Models ───
    "defaultModel": "claude-sonnet-4-5",
    "models": [
      { "value": "claude-sonnet-4-5", "label": "Claude Sonnet 4.5" },
      { "value": "claude-opus-4-6",   "label": "Claude Opus 4.6" },
      { "value": "claude-opus-4-5",   "label": "Claude Opus 4.5" },
      { "value": "claude-haiku-4-5",  "label": "Claude Haiku 4.5" }
    ],

    // ─── Feature Gate ───
    "featureGate": ""                   // empty = always enabled
    // NOTE: No capabilities here — capabilities come from the runner's
    // /capabilities endpoint at runtime. That is the source of truth.
  },

  {
    "id": "gemini-cli",
    "displayName": "Gemini CLI",
    "description": "Google Gemini coding agent with built-in file, bash, and search tools",
    "framework": "gemini-cli",
    "container": {
      "image": "quay.io/ambient_code/ambient_runner:latest",
      "port": 8001,
      "env": {
        "RUNNER_TYPE": "gemini-cli",
        "RUNNER_STATE_DIR": ".gemini"
      }
    },
    "sandbox": {
      "stateDir": ".gemini",
      "stateSyncImage": "quay.io/ambient_code/state_sync:latest",
      "persistence": "s3",
      "workspaceSize": "10Gi",
      "terminationGracePeriod": 60,
      "seed": { "cloneRepos": true, "hydrateState": true }
    },
    "auth": {
      "requiredSecretKeys": ["GEMINI_API_KEY", "GOOGLE_API_KEY"],
      "secretKeyLogic": "any",
      "vertexSupported": true
    },
    "defaultModel": "gemini-2.5-flash",
    "models": [
      { "value": "gemini-2.5-flash", "label": "Gemini 2.5 Flash" },
      { "value": "gemini-2.5-pro",   "label": "Gemini 2.5 Pro" }
    ],
    "featureGate": "runner.gemini-cli.enabled"
  }
]
```

### 4.3 Request Flow

All runners follow the same flow. The registry controls what the pod looks like.

```
CreateSession POST
  → Backend: resolve runtime from registry
  → Backend: validate auth.requiredSecretKeys
  → Backend: create AgenticSession CRD with runnerType
  → Operator: read runtime from registry
  → Operator: create Pod from runtime.container spec
     - If seed.cloneRepos:  add init-hydrate container
     - If seed.hydrateState: init container restores from S3
     - If persistence != "none": add state-sync sidecar
     - Otherwise: skip init + sidecar (lightweight pod)
  → Operator: create Service session-{name}
  → Backend: proxy to http://session-{name}.{ns}:{port}/

Run POST
  → Backend: getRunnerEndpoint() → per-session Service
  → Runner: bridge.run() → AG-UI events
  → Frontend: SSE stream
```

---

## 5. Data Model

### 5.1 Go Types (Backend + Operator)

```go
// AgentRuntimeSpec — parsed from registry ConfigMap JSON
type AgentRuntimeSpec struct {
    ID           string        `json:"id"`
    DisplayName  string        `json:"displayName"`
    Description  string        `json:"description"`
    Framework    string        `json:"framework"`
    Container    ContainerSpec `json:"container"`
    Sandbox      SandboxSpec   `json:"sandbox"`
    Auth         AuthSpec      `json:"auth"`
    DefaultModel string        `json:"defaultModel"`
    Models       []ModelEntry  `json:"models"`
    FeatureGate  string        `json:"featureGate"`
    // NOTE: No capabilities — runtime-reported by runner's /capabilities endpoint.
}

type ContainerSpec struct {
    Image     string            `json:"image"`
    Port      int               `json:"port"`
    Env       map[string]string `json:"env"`
    Resources *ResourcesSpec    `json:"resources,omitempty"`
}

type SandboxSpec struct {
    StateDir               string `json:"stateDir,omitempty"`
    StateSyncImage         string `json:"stateSyncImage,omitempty"`
    Persistence            string `json:"persistence"`            // "s3" | "none"
    WorkspaceSize          string `json:"workspaceSize,omitempty"`
    TerminationGracePeriod int    `json:"terminationGracePeriod,omitempty"`
    Seed                   struct {
        CloneRepos   bool `json:"cloneRepos"`
        HydrateState bool `json:"hydrateState"`
    } `json:"seed"`
}

type AuthSpec struct {
    RequiredSecretKeys []string `json:"requiredSecretKeys"`
    SecretKeyLogic     string   `json:"secretKeyLogic"` // "any" | "all"
    VertexSupported    bool     `json:"vertexSupported"`
}
```

### 5.2 TypeScript Types (Frontend)

```typescript
interface AgentRuntime {
  id: string;
  displayName: string;
  description: string;
  framework: string;
  defaultModel: string;
  models: RunnerModel[];
  auth: {
    requiredSecretKeys: string[];
    secretKeyLogic: "any" | "all";
    vertexSupported: boolean;
  };
  featureGate: string;
  // Capabilities come from runner's /capabilities endpoint at runtime.
}
```

### 5.3 Python Bridge Registry

```python
# Bridge registry — maps framework ID to (module_path, class_name)
# Lazy imports avoid pulling in unused dependencies
BRIDGE_REGISTRY: dict[str, tuple[str, str]] = {
    "claude-agent-sdk": ("ambient_runner.bridges.claude", "ClaudeBridge"),
    "gemini-cli":       ("ambient_runner.bridges.gemini_cli", "GeminiCLIBridge"),
    "langgraph":        ("ambient_runner.bridges.langgraph", "LangGraphBridge"),
}
```

---

## 6. Operator Changes

### 6.1 Registry-Driven Pod Spec Builder

Replace hardcoded values with registry lookups:

```go
func (h *Handler) createPodForSession(session *v1alpha1.AgenticSession) error {
    runtime, err := h.registryCache.GetRuntime(session.Spec.RunnerType)
    if err != nil {
        return fmt.Errorf("unknown runner type %q: %w", session.Spec.RunnerType, err)
    }

    pod := h.buildPodSpec(session, runtime)
    // ... create pod, service, etc.
}
```

### 6.2 What Changes from Current Code

| Current (hardcoded) | New (from registry) |
|---------------------|---------------------|
| `AMBIENT_CODE_RUNNER_IMAGE` env var | `runtime.Container.Image` |
| Hardcoded resource requests/limits | `runtime.Container.Resources` |
| Hardcoded port 8001 | `runtime.Container.Port` |
| `getRunnerInternalEnvVars()` map | `runtime.Container.Env` |
| `STATE_SYNC_IMAGE` env var | `runtime.Sandbox.StateSyncImage` |
| `runnerStateDirs` Go map | `runtime.Sandbox.StateDir` |
| Hardcoded "10Gi" workspace | `runtime.Sandbox.WorkspaceSize` |
| Hardcoded 60s grace period | `runtime.Sandbox.TerminationGracePeriod` |

### 6.3 Conditional Pod Components

The operator uses `sandbox.seed` and `sandbox.persistence` to decide what to include:

```go
func (h *Handler) buildPodSpec(session *v1alpha1.AgenticSession, runtime *AgentRuntimeSpec) *corev1.PodSpec {
    spec := &corev1.PodSpec{
        Containers: []corev1.Container{h.buildRunnerContainer(session, runtime)},
    }

    // Only add workspace volume if persistence is configured or repos are seeded
    if runtime.Sandbox.Persistence != "none" || runtime.Sandbox.Seed.CloneRepos {
        spec.Volumes = append(spec.Volumes, h.buildWorkspaceVolume(runtime))
    }

    // Only add init container if seeding is needed
    if runtime.Sandbox.Seed.CloneRepos || runtime.Sandbox.Seed.HydrateState {
        spec.InitContainers = []corev1.Container{h.buildInitContainer(session, runtime)}
    }

    // Only add state-sync sidecar if persistence is enabled
    if runtime.Sandbox.Persistence != "none" {
        spec.Containers = append(spec.Containers, h.buildStateSyncContainer(session, runtime))
    }

    return spec
}
```

### 6.4 Backend Routing

```go
func getRunnerEndpoint(projectName, sessionName string, runtime *AgentRuntimeSpec) string {
    return fmt.Sprintf("http://session-%s.%s.svc.cluster.local:%d/",
        sessionName, projectName, runtime.Container.Port)
}
```

### 6.5 Remove Hardcoded Logic

Delete:
- `runnerStateDirs` map in `runner_types.go`
- `getRunnerInternalEnvVars()` — replaced by `runtime.Container.Env`
- Direct reference to `AMBIENT_CODE_RUNNER_IMAGE` — replaced by `runtime.Container.Image`

---

## 7. Runner (Python) Changes

### 7.1 Registry-Driven Bridge Factory

```python
# main.py
import os
import importlib
import logging

logger = logging.getLogger(__name__)

RUNNER_TYPE = os.getenv("RUNNER_TYPE", "claude-agent-sdk").strip().lower()

BRIDGE_REGISTRY: dict[str, tuple[str, str]] = {
    "claude-agent-sdk": ("ambient_runner.bridges.claude", "ClaudeBridge"),
    "gemini-cli":       ("ambient_runner.bridges.gemini_cli", "GeminiCLIBridge"),
    "langgraph":        ("ambient_runner.bridges.langgraph", "LangGraphBridge"),
}

def _load_bridge():
    if RUNNER_TYPE not in BRIDGE_REGISTRY:
        raise ValueError(
            f"Unknown RUNNER_TYPE={RUNNER_TYPE!r}. "
            f"Available: {sorted(BRIDGE_REGISTRY)}"
        )
    module_path, class_name = BRIDGE_REGISTRY[RUNNER_TYPE]
    module = importlib.import_module(module_path)
    bridge_cls = getattr(module, class_name)
    logger.info(f"Loading bridge: {class_name} from {module_path}")
    return bridge_cls()

app = create_ambient_app(_load_bridge(), title="Ambient Runner AG-UI Server")
```

---

## 8. Frontend Changes

### 8.1 Registry-Driven Settings

The settings page already derives API keys from `useRunnerTypes()` (done in prior work).

### 8.2 Admin Runtimes Page (New)

Simple read-only management view at `/admin/runtimes`:

```
┌──────────────────────────────────────────────────┐
│ Agent Runtimes                                   │
├───────────────┬────────┬─────────────────────────┤
│ Runtime       │ Models │ Status                  │
├───────────────┼────────┼─────────────────────────┤
│ Claude Code   │ 4      │ ● Enabled               │
│ Gemini CLI    │ 2      │ ○ Gated (toggle)        │
└───────────────┴────────┴─────────────────────────┘
```

Clicking a runtime shows its full configuration (read-only from ConfigMap). Feature gate toggle uses the existing Unleash admin API — the ConfigMap's `featureGate` field holds the flag name, and `isRunnerEnabled()` checks Unleash.

### 8.3 Create Session Dialog

Already handles multiple runners. Capability-driven UI works via the `useCapabilities()` hook which polls the runner's `/capabilities` endpoint after session creation. No registry involvement needed.

---

## 9. Implementation Tasks

### Phase 1: Registry Schema & Operator (Foundation)

| # | Task | Files | Effort |
|---|------|-------|--------|
| 1.1 | Extend ConfigMap schema with full AgentRuntimeSpec | `agent-registry-configmap.yaml` | S |
| 1.2 | Add Go types for AgentRuntimeSpec in backend + operator | `backend/handlers/runner_types.go`, `operator/internal/handlers/` | S |
| 1.3 | Operator reads container spec from registry (image, port, resources, env) | `operator/internal/handlers/sessions.go`, `operator/internal/config/config.go` | M |
| 1.4 | Operator conditionally builds pod components from sandbox config | `operator/internal/handlers/sessions.go` | M |
| 1.5 | Remove hardcoded `runnerStateDirs`, `AMBIENT_CODE_RUNNER_IMAGE`, `getRunnerInternalEnvVars()` | `backend/handlers/runner_types.go`, `operator/internal/config/config.go` | S |
| 1.6 | Backend routing reads port from registry | `backend/websocket/agui_proxy.go` | S |

### Phase 2: Python Bridge Registry

| # | Task | Files | Effort |
|---|------|-------|--------|
| 2.1 | Replace if/elif factory with `BRIDGE_REGISTRY` dict | `main.py` | S |

### Phase 3: Frontend & Admin

| # | Task | Files | Effort |
|---|------|-------|--------|
| 3.1 | Extend RunnerType interface with full schema (auth, featureGate) | `services/api/runner-types.ts` | S |
| 3.2 | Admin runtimes page (read-only list + details) | `app/(main)/admin/runtimes/` (new) | M |
| 3.3 | Feature gate toggle via existing Unleash admin API | `admin/runtimes/` | S |

### Phase 4: Tests

| # | Task | Files | Effort |
|---|------|-------|--------|
| 4.1 | Bridge registry tests | `tests/test_bridge_registry.py` | S |
| 4.2 | Operator conditional pod component tests | `operator tests` | M |
| 4.3 | Backend routing tests | `backend tests` | S |

---

## 10. Migration & Backward Compatibility

### ConfigMap Migration

The enhanced ConfigMap is a superset of the current schema. Existing fields preserved. New fields have defaults:

| New Field | Default if Missing |
|-----------|--------------------|
| `framework` | Same as `id` |
| `container` | Built from `AMBIENT_CODE_RUNNER_IMAGE` env var |
| `sandbox` | Built from existing hardcoded values |
| `auth.secretKeyLogic` | `"any"` |
| `featureGate` | `""` (enabled) |

### Operator Fallbacks

```go
// If container.image is empty, fall back to env var (backward compat)
image := runtime.Container.Image
if image == "" {
    image = os.Getenv("AMBIENT_CODE_RUNNER_IMAGE")
}
```

---

## 11. Security Considerations

| Concern | Mitigation |
|---------|------------|
| Credential isolation | Pod-per-session. Each session gets its own pod with its own env vars. |
| Cross-session leakage | Impossible — separate pods, separate volumes, separate processes. |
| Registry tampering | ConfigMap protected by K8s RBAC. Only cluster admins can modify. |
| Container image trust | Images from trusted registries (enforced via admission policy). |

---

## 12. Resolved Decisions

| # | Question | Decision | Rationale |
|---|----------|----------|-----------|
| D1 | Execution mode | All sandbox (no shared mode) | Shared adds complexity without clear benefit. Lightweight sandbox (no init, no sidecar, no workspace) gives fast startup with full isolation. |
| D2 | Admin UI scope | View + feature gate toggle | GitOps for config changes. Prevents UI-driven drift. |
| D3 | CRD promotion | Never — ConfigMap permanent | Simpler to operate. Validation in Go code. 60s cache TTL sufficient. |
| D4 | Capabilities source of truth | Runner's `/capabilities` endpoint, NOT the registry | Registry stores infrastructure config (image, resources, sandbox). Capabilities are runtime-reported by the bridge. Frontend already uses `useCapabilities()` hook. No duplication. |
| D5 | Direct-to-LLM | Deferred — not in this implementation | Add later as a ConfigMap entry + bridge package. Registry is designed to support it without code changes. |

---

## 13. Success Criteria

- [ ] New runners can be added by updating ConfigMap + building a container image (no Go/Python/TS code changes)
- [ ] Operator builds pod spec from registry (image, resources, conditional init/sidecar)
- [ ] Frontend derives all runner-specific UI from runner's `/capabilities` endpoint (no hardcoded runner checks)
- [ ] Existing Claude SDK and Gemini CLI runners work identically after migration
- [ ] Admin page shows registered runtimes with status and feature gates
