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

### Iterative Development

Quick iteration without recreating cluster:

```bash
# Initial setup
make kind-up

# Edit e2e/.env to change images or add API key
vim e2e/.env

# Recreate cluster to pick up changes
make kind-down
make kind-up

# Test
make test-e2e

# Repeat...
```

**Example `e2e/.env`:**
```bash
# Test custom backend build
IMAGE_BACKEND=quay.io/your-org/vteam_backend:fix-123

# Enable agent testing
ANTHROPIC_API_KEY=sk-ant-api03-...
```

---

## Configuration

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
