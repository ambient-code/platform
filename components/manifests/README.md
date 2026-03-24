# Manifests

Kustomize-based deployment manifests for the Ambient Code platform.

## Structure

```
base/           Shared base resources (deployments, services, RBAC, CRDs)
  core/         Core platform deployments (backend, frontend, operator, control-plane, ...)
  rbac/         ClusterRoles, ClusterRoleBindings, ServiceAccounts
  platform/     Platform-level resources (namespaces, etc.)
components/     Reusable Kustomize components (oauth-proxy, postgresql-rhel, api-server-db)
overlays/       Per-environment configuration layered on top of base
```

## Overlays

### `e2e`
Cypress end-to-end tests running in a kind cluster during CI. Uses mock SDK mode — no real Claude execution, no control-plane deployed. Images built from PR code or pulled from quay.io.

### `kind`
Local kind cluster using mostly quay.io images. The control-plane and api-server are remapped to `localhost/` because they may not be in the registry yet. Intended for testing against real published images with minimal local builds.

### `kind-local`
Layers on top of `kind`. Remaps **all** images to `localhost/` and sets `imagePullPolicy: IfNotPresent`. Intended for developers who have built everything locally with `make build-all` and want a fully offline cluster.

### `local-dev`
OpenShift CRC or internal dev cluster (`vteam-dev` namespace). Images are pulled from the OpenShift internal registry (`image-registry.openshift-image-registry.svc:5000/vteam-dev/...`) for backend, frontend, operator, and control-plane — built via OpenShift BuildConfigs. Runner and api-server pull from quay.io. Uses `namePrefix: vteam-`.

### `production`
OpenShift production cluster (`ambient-code` namespace). All images pulled from `quay.io/ambient_code/` — tags are pinned at deploy time by the CI `deploy-to-openshift` job via `kustomize edit set image`. No internal registry references.

## Image Naming Convention

All images use the `vteam_` prefix on quay.io:

 < /dev/null |  Component | Image |
|---|---|
| Backend | `quay.io/ambient_code/vteam_backend` |
| Frontend | `quay.io/ambient_code/vteam_frontend` |
| Operator | `quay.io/ambient_code/vteam_operator` |
| Runner | `quay.io/ambient_code/vteam_claude_runner` |
| State Sync | `quay.io/ambient_code/vteam_state_sync` |
| Public API | `quay.io/ambient_code/vteam_public_api` |
| API Server | `quay.io/ambient_code/vteam_api_server` |
| Control Plane | `quay.io/ambient_code/vteam_control_plane` |

## Control Plane Image per Environment

| Overlay | Image source |
|---|---|
| `e2e` | Not deployed |
| `kind` | `localhost/vteam_control_plane:latest` |
| `kind-local` | `localhost/vteam_control_plane:latest` |
| `local-dev` | `image-registry.openshift-image-registry.svc:5000/vteam-dev/vteam_control_plane:latest` |
| `production` | `quay.io/ambient_code/vteam_control_plane:<sha>` (set by CI) |
