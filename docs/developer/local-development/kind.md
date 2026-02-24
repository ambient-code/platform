# Local Development with Kind

Run the Ambient Code Platform locally using kind (Kubernetes in Podman/Docker) for development and testing.

> **Cluster Name**: `ambient-local`  
> **Default Engine**: Podman (use `CONTAINER_ENGINE=docker` for more stable networking on macOS)

## Quick Start

```bash
# Start cluster (uses podman by default)
make kind-up

# In another terminal, port-forward for access
make kind-port-forward

# Run tests
make test-e2e

# Cleanup
make kind-down
```

**With Docker:**
```bash
make kind-up CONTAINER_ENGINE=docker
```

## Prerequisites

- **Podman** OR **Docker (more stable on macOS)**:
  - Podman: `brew install podman && podman machine init && podman machine start`
  - Docker: https://docs.docker.com/get-docker/
  - **Note:** Docker is more stable for kind on macOS (Podman's port forwarding can become flaky)
- **kind**: `brew install kind`
- **kubectl**: `brew install kubectl`

**Verify:**
```bash
# With Podman (default)
podman ps && kind --version && kubectl version --client

# With Docker
docker ps && kind --version && kubectl version --client
```

## Architecture Support

The platform auto-detects your host architecture and builds native images:

- **Apple Silicon (M1/M2/M3):** `linux/arm64`
- **Intel/AMD:** `linux/amd64`

**Verify native builds:**
```bash
make check-architecture  # Should show "✓ Using native architecture"
```

**Manual override (if needed):**
```bash
make build-all PLATFORM=linux/arm64  # Force specific architecture
```

⚠️ **Warning:** Cross-compiling (building non-native architecture) is 4-6x slower and may crash.

## Commands

### `make kind-up`

Creates kind cluster and deploys platform with Quay.io images.

**What it does:**
1. Creates minimal kind cluster (no ingress)
2. Deploys platform (backend, frontend, operator, minio)
3. Initializes MinIO storage
4. Extracts test token to `e2e/.env.test`

**Access:**
- Run `make kind-port-forward` in another terminal
- Frontend: `http://localhost:8080`
- Backend: `http://localhost:8081`
- Token: `kubectl get secret test-user-token -n ambient-code -o jsonpath='{.data.token}' | base64 -d`

### `make test-e2e`

Runs Cypress e2e tests against the cluster.

**Runtime:** ~20 seconds (12 tests)

### `make kind-down`

Deletes the kind cluster.

---

## Local Development

### With Quay Images (Default)

Best for testing without rebuilding:

```bash
make kind-up       # Deploy
make test-e2e      # Test
make kind-down     # Cleanup
```

### Building from Source

Build all components from your local source tree and deploy to kind.

> **Note:** `LOCAL_IMAGES=true` requires **Podman** as the container engine. The `kind-local` overlay expects the `localhost/` image prefix that Podman uses natively.

```bash
# Build, load, and deploy in one step (requires CONTAINER_ENGINE=podman)
make kind-up LOCAL_IMAGES=true

# Combine with Vertex AI
make kind-up LOCAL_IMAGES=true LOCAL_VERTEX=true
```

This builds all container images from source, loads them into the kind cluster, and deploys using the `kind-local` overlay (which sets `imagePullPolicy: IfNotPresent`).

#### Iterating After Code Changes

After the initial `kind-up LOCAL_IMAGES=true`, use `kind-rebuild` to pick up code changes without recreating the cluster:

```bash
# Rebuild all components, reload into kind, restart deployments
make kind-rebuild
```

To rebuild a single component (faster):

```bash
# Example: backend change
make build-backend && \
  kind load docker-image vteam_backend:latest --name ambient-local && \
  kubectl rollout restart deployment/backend-api -n ambient-code
```

| Component | Build target | Deployment to restart |
|-----------|-------------|----------------------|
| Backend | `make build-backend` | `backend-api` |
| Frontend | `make build-frontend` | `frontend` |
| Operator | `make build-operator` | `agentic-operator` |
| Public API | `make build-public-api` | `public-api` |
| Runner | `make build-runner` | *(none -- picked up by next session)* |
| State Sync | `make build-state-sync` | *(none -- picked up by next session)* |

#### Verify Which Images Are Running

```bash
# Check deployment images
kubectl get deployments -n ambient-code \
  -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.template.spec.containers[0].image}{"\n"}{end}'

# Check runner image configured in operator
kubectl get configmap operator-config -n ambient-code \
  -o jsonpath='{.data.AMBIENT_CODE_RUNNER_IMAGE}'
```

With `LOCAL_IMAGES=true`, images show as `localhost/vteam_*:latest` (Podman prefix, no `quay.io`).

### With Quay Images (Default)

Best for testing without rebuilding:

```bash
make kind-up       # Deploy with pre-built Quay.io images
make test-e2e      # Test
make kind-down     # Cleanup
```

---

## Configuration

### Vertex AI (Optional)

Use Google Cloud Vertex AI instead of direct Anthropic API:

```bash
# If you already have these in .zshrc (e.g., for Claude Code CLI):
# - ANTHROPIC_VERTEX_PROJECT_ID
# - CLOUD_ML_REGION

# Just add LOCAL_VERTEX=true
make kind-up LOCAL_VERTEX=true
```

**Default credentials:** `~/.config/gcloud/application_default_credentials.json`
(Created by `gcloud auth application-default login`)

**Override credentials path:**
```bash
make kind-up LOCAL_VERTEX=true GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
```

**Override all values:**
```bash
make kind-up LOCAL_VERTEX=true \
    ANTHROPIC_VERTEX_PROJECT_ID=my-project \
    CLOUD_ML_REGION=us-east5 \
    GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json
```

**Reconfigure existing cluster:**
```bash
# If cluster is already running, run the setup script directly
./scripts/setup-vertex-kind.sh
```

### Environment Variables (`e2e/.env`)

Create `e2e/.env` to customize the deployment:

```bash
# Copy example
cp e2e/env.example e2e/.env
```

**Available options:**

```bash
# Enable agent testing
ANTHROPIC_API_KEY=sk-ant-api03-your-key-here

# Override specific images (for testing custom builds)
IMAGE_BACKEND=quay.io/your-org/vteam_backend:custom-tag
IMAGE_FRONTEND=quay.io/your-org/vteam_frontend:custom-tag
IMAGE_OPERATOR=quay.io/your-org/vteam_operator:custom-tag
IMAGE_RUNNER=quay.io/your-org/vteam_claude_runner:custom-tag
IMAGE_STATE_SYNC=quay.io/your-org/vteam_state_sync:custom-tag

# Or override registry for all images
CONTAINER_REGISTRY=quay.io/your-org
```

**Apply changes:**

```bash
make kind-down && make kind-up
```

---

## Troubleshooting

### Cluster won't start

```bash
# Verify container runtime is running
podman ps  # or docker ps

# Recreate cluster
make kind-down
make kind-up
```

### Pods not starting

```bash
kubectl get pods -n ambient-code
kubectl logs -n ambient-code deployment/backend-api
```

### Port 8080 stops working (Podman on macOS)

**Symptom:** Ingress works initially, then hangs after 10-30 minutes.  
**Cause:** Podman's gvproxy port forwarding can become flaky on macOS.

**Workaround - Use port-forward:**
```bash
# Stop using ingress on 8080, use direct port-forward instead
kubectl port-forward -n ambient-code svc/frontend-service 18080:3000

# Update test config
cd e2e
perl -pi -e 's|http://localhost:8080|http://localhost:18080|' .env.test

# Access at http://localhost:18080
```

**Permanent fix:** Use Docker instead of Podman on macOS:
```bash
# Switch to Docker
make kind-down CONTAINER_ENGINE=podman
make kind-up CONTAINER_ENGINE=docker
# Access at http://localhost (port 80)
```

### Port conflict (8080)

```bash
lsof -i:8080  # Find what's using the port
# Kill it or edit e2e/scripts/setup-kind.sh to use different ports
```

### Build crashes with segmentation fault

**Symptom:** `qemu: uncaught target signal 11 (Segmentation fault)` during Next.js build

**Fix:**
```bash
# Auto-detect and use native architecture
make local-clean
make local-up
```

**Diagnosis:** Run `make check-architecture` to verify native builds are enabled.

### MinIO errors

```bash
cd e2e && ./scripts/init-minio.sh
```

---

## Quick Reference

```bash
# View logs
kubectl logs -n ambient-code -l app=backend-api -f

# Restart component
kubectl rollout restart -n ambient-code deployment/backend-api

# List sessions
kubectl get agenticsessions -A

# Delete cluster
make kind-down
```

---

## See Also

- [Hybrid Local Development](hybrid.md) - Run components locally (faster iteration)
- [E2E Testing Guide](../e2e/README.md) - Running e2e tests
- [Testing Strategy](../CLAUDE.md#testing-strategy) - Overview
- [kind Documentation](https://kind.sigs.k8s.io/)
