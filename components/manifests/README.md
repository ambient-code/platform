# Ambient Platform Manifests

Kubernetes/OpenShift manifests organized with **Kustomize** overlays. The base defines a secure,
production-grade default (TLS everywhere, JWT auth enabled, strict RBAC). Overlays progressively
relax constraints for development environments.

## Directory Structure

```
manifests/
├── base/                                  # Secure production-grade defaults
│   ├── kustomization.yaml                 # Delegates to core/, platform/, rbac/
│   ├── core/                              # Application deployments + config
│   │   ├── backend-deployment.yaml
│   │   ├── frontend-deployment.yaml
│   │   ├── operator-deployment.yaml
│   │   ├── public-api-deployment.yaml
│   │   ├── ambient-api-server-service.yml # ambient-api-server Deployment + Service
│   │   ├── agent-registry-configmap.yaml
│   │   ├── minio-deployment.yaml
│   │   ├── postgresql-deployment.yaml
│   │   ├── unleash-deployment.yaml
│   │   ├── workspace-pvc.yaml
│   │   ├── models.json                    # Available LLM models (ConfigMap source)
│   │   └── flags.json                     # Feature flags (ConfigMap source)
│   ├── platform/                          # Cluster-level resources
│   │   ├── namespace.yaml
│   │   ├── ambient-api-server-db.yml      # ambient-api-server PostgreSQL deployment
│   │   └── ambient-api-server-secrets.yml # Secret template (ambient-code-rds secret for DB)
│   ├── crds/                              # Custom Resource Definitions
│   │   ├── agenticsessions-crd.yaml
│   │   └── projectsettings-crd.yaml
│   └── rbac/                              # ClusterRoles for backend and operator
│       ├── backend-clusterrole.yaml
│       └── operator-clusterrole.yaml
│
├── components/                            # Reusable opt-in kustomize components
│   ├── oauth-proxy/                       # OpenShift OAuth proxy sidecar for frontend
│   ├── postgresql-rhel/                   # RHEL PostgreSQL image + env var patches
│   ├── postgresql-init-scripts/           # Init ConfigMap for vanilla postgres DB creation
│   └── ambient-api-server-db/             # RHEL image patch for ambient-api-server DB
│
├── overlays/
│   ├── production/                        # OpenShift production (ROSA / on-prem)
│   ├── kind/                              # Local kind cluster (Quay images)
│   ├── kind-local/                        # Local kind cluster (locally built images)
│   ├── e2e/                               # Cypress E2E test environment (kind)
│   └── local-dev/                         # CRC / OpenShift Local developer environment
│
└── observability/                         # Grafana dashboards, OTel collector, ServiceMonitors
```

## Security Model

The base manifests assume full TLS and JWT authentication. Overlays strip these down as needed:

| Layer | TLS | JWT / JWKS | Auth |
|---|---|---|---|
| `base` | enabled (HTTPS + gRPC TLS) | enabled (Red Hat SSO) | enabled |
| `production` | enabled (OpenShift service-ca) | enabled (Red Hat SSO) | enabled |
| `local-dev` | enabled (OpenShift service-ca) | enabled | enabled |
| `kind` | disabled | disabled | disabled |
| `kind-local` | disabled | disabled | disabled |
| `e2e` | disabled | disabled | disabled |

## Overlays

### `production/` — OpenShift (ROSA / on-prem)
- **Images**: `quay.io/ambient_code/*`
- **Networking**: OpenShift Routes
- **Auth**: OAuth proxy sidecar (`components/oauth-proxy`), Red Hat SSO JWKS
- **Database**: RHEL PostgreSQL (`components/postgresql-rhel`, `components/ambient-api-server-db`)
- **TLS**: Auto-provisioned by OpenShift service-ca

```bash
oc apply -k overlays/production/
```

### `kind/` — Local kind cluster (Quay images)
- **Images**: `quay.io/ambient_code/*` pulled directly
- **Networking**: NodePort services
- **Auth**: JWT disabled, no-TLS patches applied
- **Database**: Vanilla postgres with init scripts (`components/postgresql-init-scripts`)

```bash
make kind-up
kubectl apply -k overlays/kind/
```

### `kind-local/` — Local kind cluster (locally built images)
Extends `kind/` — overrides image refs to locally loaded images (`imagePullPolicy: Never`).

```bash
make local-reload-api-server KIND_CLUSTER_NAME=<cluster>
```

### `e2e/` — Cypress E2E test environment
Kind-based environment used by `make test-e2e-local`. Adds test users, ingress, and
Cypress-compatible service configuration on top of the kind overlay.

```bash
make test-e2e-local
```

### `local-dev/` — CRC / OpenShift Local
- **Images**: Internal OpenShift registry (`image-registry.openshift-image-registry.svc:5000/vteam-dev/*`)
- **Namespace**: `vteam-dev` (with `namePrefix: vteam-`)
- **Auth**: OpenShift service-ca TLS, JWKS enabled
- **Database**: RHEL PostgreSQL with init containers

```bash
make local-up
```

## Reusable Components

Components are opt-in kustomize modules included via the `components:` block in an overlay's
`kustomization.yaml`. They are **not** applied by default.

| Component | Purpose | Used by |
|---|---|---|
| `oauth-proxy` | Adds OpenShift OAuth proxy sidecar to frontend | `production` |
| `postgresql-rhel` | Patches PostgreSQL to use `registry.redhat.io/rhel10/postgresql-16` | `production`, `local-dev` |
| `ambient-api-server-db` | RHEL patch for ambient-api-server DB (updates ambient-code-rds secret refs) | `production`, `local-dev` |
| `postgresql-init-scripts` | ConfigMap + volume for DB init SQL (vanilla postgres only) | `kind`, `e2e` |

## Building and Validating

```bash
# Dry-run any overlay
kustomize build overlays/production/
kustomize build overlays/kind/
kustomize build overlays/e2e/

# Apply
kubectl apply -k overlays/kind/
oc apply -k overlays/production/
```

## Adding a New Environment Resource

1. If it belongs in all environments → add to `base/core/` or `base/platform/` and update the
   relevant `kustomization.yaml`.
2. If it's environment-specific → add to the overlay directory and reference it in that overlay's
   `kustomization.yaml`.
3. If it's a reusable opt-in pattern → create a new `components/<name>/` directory with its own
   `kustomization.yaml` and include it via `components:` in the overlays that need it.
