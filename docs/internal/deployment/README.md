# Deployment Documentation

Guides for deploying the Ambient Code Platform to various environments.

## Components

Two generations of components deploy side by side. V1 remains fully operational while V2 is additive.

### V1 Components (legacy, still active)

| Component | Image | Port |
|-----------|-------|------|
| frontend | `quay.io/ambient_code/vteam_frontend` | 3000 |
| backend-api | `quay.io/ambient_code/vteam_backend` | 8080 |
| agentic-operator | `quay.io/ambient_code/vteam_operator` | — |
| public-api | `quay.io/ambient_code/vteam_public_api` | 8081 |
| claude-runner | `quay.io/ambient_code/vteam_claude_runner` | — |

### V2 Components

| Component | Image | Ports | Notes |
|-----------|-------|-------|-------|
| ambient-api-server | `quay.io/ambient_code/vteam_api_server` | 8000 (API) / 4433 (metrics) / 4434 (health) | REST + gRPC API, PostgreSQL-backed |
| ambient-api-server-db | `postgres:16.2` | 5432 | Dedicated PostgreSQL for API server |
| ambient-control-plane | `quay.io/ambient_code/ambient_control_plane` | — | Pending merge to main |

## Deployment Guides

- **[OpenShift Deployment](OPENSHIFT_DEPLOY.md)** — Production OpenShift cluster
- **[OAuth Configuration](OPENSHIFT_OAUTH.md)** — OpenShift OAuth setup
- **[Git Authentication](git-authentication.md)** — Git credentials for runners
- **[Langfuse](langfuse.md)** — LLM observability
- **[MinIO](minio-quickstart.md)** — S3-compatible storage
- **[S3 Storage](s3-storage-configuration.md)** — S3 configuration

## Deployment

### Prerequisites

- OpenShift or Kubernetes cluster with admin access
- Container registry access (`quay.io/ambient_code` or your own)
- `oc` or `kubectl` configured
- Anthropic API key

### Basic Deployment

```bash
cp components/manifests/env.example components/manifests/.env
# Edit .env: set ANTHROPIC_API_KEY

make deploy
```

Verify:
```bash
oc get pods -n ambient-code
oc get routes -n ambient-code
```

### Custom Images

```bash
make build-all CONTAINER_ENGINE=podman
make push-all REGISTRY=quay.io/your-username
make deploy CONTAINER_REGISTRY=quay.io/your-username
```

### Custom Namespace

```bash
make deploy NAMESPACE=my-namespace
```

## Post-Deployment Configuration

1. **Runner Secrets** — Web UI → Settings → Runner Secrets → add Anthropic API key
2. **Git Authentication** (optional) — see [Git Authentication Guide](git-authentication.md)
3. **Observability** (optional) — see [Langfuse Guide](langfuse.md)

## Secrets

### ambient-api-server

The API server reads database credentials from `Secret/ambient-api-server-db` and auth config from `ConfigMap/ambient-api-server-auth`. These are defined in `components/manifests/base/ambient-api-server-secrets.yml`.

For production, patch `db.password` and populate `jwks.json` with your OIDC provider's JWKS endpoint response.

```bash
# Override DB password
oc create secret generic ambient-api-server-db \
  --from-literal=db.host=ambient-api-server-db \
  --from-literal=db.port=5432 \
  --from-literal=db.name=ambient_api_server \
  --from-literal=db.user=ambient \
  --from-literal=db.password=<your-password> \
  -n ambient-code --dry-run=client -o yaml | oc apply -f -
```

## Health Checks

```bash
# V1
curl https://$(oc get route backend-route -n ambient-code -o jsonpath='{.spec.host}')/health
curl https://$(oc get route frontend-route -n ambient-code -o jsonpath='{.spec.host}')

# V2 — ambient-api-server
oc port-forward svc/ambient-api-server 4434:4434 -n ambient-code
curl http://localhost:4434/health

# All pods
oc get pods -n ambient-code
```

## Logs

```bash
# V1
oc logs -n ambient-code deployment/backend-api -f
oc logs -n ambient-code deployment/frontend -f
oc logs -n ambient-code deployment/agentic-operator -f
oc logs -n <project-namespace> job/<job-name>   # runner jobs

# V2
oc logs -n ambient-code deployment/ambient-api-server -f
oc logs -n ambient-code deployment/ambient-api-server-db -f
```

## Metrics

```bash
# V2 — ambient-api-server Prometheus metrics
oc port-forward svc/ambient-api-server 4433:4433 -n ambient-code
curl http://localhost:4433/metrics
```

See [observability/](../../components/manifests/observability/) for Grafana dashboards and ServiceMonitor configuration.

## Cleanup

```bash
# Uninstall platform
make clean

# Remove CRDs
oc delete crd agenticsessions.vteam.ambient-code
oc delete crd projectsettings.vteam.ambient-code

# Remove namespace (destructive)
oc delete namespace ambient-code
```

## Troubleshooting

| Symptom | Command |
|---------|---------|
| Pod not starting | `oc describe pod <name> -n ambient-code` |
| Image pull error | `oc get deployment <name> -n ambient-code -o jsonpath='{.spec.template.spec.imagePullSecrets}'` |
| Route not accessible | `oc get route <name> -n ambient-code` |
| Operator not creating jobs | `oc logs -n ambient-code deployment/agentic-operator -f` |
| API server DB connection failed | Check `Secret/ambient-api-server-db` credentials match `Deployment/ambient-api-server-db` env vars |
| API server migration failed | Check init container logs: `oc logs -n ambient-code deployment/ambient-api-server -c migrate` |

## Related Documentation

- [Architecture Overview](../architecture/) — System design and V1 vs V2 diagrams
- [Local Development](../developer/local-development/) — Run V2 stack locally without a cluster
- [Manifests README](../../components/manifests/README.md) — Kustomize overlay structure
