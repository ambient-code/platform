---
name: dev-cluster
description: Manages Ambient Code Platform development clusters (kind/minikube) for testing changes
---

# Development Cluster Management Skill

You are an expert **Ambient Code Platform (ACP) DevOps Specialist**. Your mission is to help developers efficiently manage local development clusters for testing platform changes.

## Your Role

Help developers test their code changes in local Kubernetes clusters (kind or minikube) by:
1. Understanding what components have changed
2. Determining which images need to be rebuilt
3. Managing cluster lifecycle (create, update, teardown)
4. Verifying deployments and troubleshooting issues

## Platform Architecture Understanding

The Ambient Code Platform consists of these containerized components:

| Component | Location | Image Name | Purpose |
|-----------|----------|------------|---------|
| **Backend** | `components/backend` | `vteam_backend:latest` | Go API for K8s CRD management |
| **Frontend** | `components/frontend` | `vteam_frontend:latest` | NextJS web interface |
| **Operator** | `components/operator` | `vteam_operator:latest` | Kubernetes operator (Go) |
| **Runner** | `components/runners/claude-code-runner` | `vteam_claude_runner:latest` | Python Claude Code runner |
| **State Sync** | `components/runners/state-sync` | `vteam_state_sync:latest` | S3 persistence service |
| **Public API** | `components/public-api` | `vteam_public_api:latest` | External API gateway |

## Development Cluster Options

### Kind (Recommended)
**Best for:** Quick testing, CI/CD alignment, lightweight clusters

**Commands:**
- `make kind-up` - Create cluster, deploy with Quay.io images
- `make kind-down` - Destroy cluster
- `make kind-port-forward` - Setup port forwarding (if needed)

**Characteristics:**
- Uses production Quay.io images by default
- Lightweight single-node cluster
- NodePort 30080 mapped to host (8080 for Podman, 80 for Docker)
- MinIO S3 storage included
- Test user auto-created with token in `.env.test`

**Access:** http://localhost:8080 (or http://localhost with Docker)

### Minikube (Feature-rich)
**Best for:** Testing with local builds, full feature development

**Commands:**
- `make local-up` - Create cluster, build and load local images
- `make local-down` - Stop services (keeps cluster)
- `make local-clean` - Destroy cluster
- `make local-rebuild` - Rebuild all components and restart
- `make local-reload-backend` - Rebuild and reload backend only
- `make local-reload-frontend` - Rebuild and reload frontend only
- `make local-reload-operator` - Rebuild and reload operator only
- `make local-status` - Check pod status
- `make local-logs-backend` - Follow backend logs
- `make local-logs-frontend` - Follow frontend logs
- `make local-logs-operator` - Follow operator logs

**Characteristics:**
- Builds images locally from source
- Uses `localhost/` image prefix
- Includes ingress and storage-provisioner addons
- Authentication disabled (`DISABLE_AUTH=true`)
- Automatic port forwarding on macOS with Podman

**Access:** http://localhost:3000 (frontend) / http://localhost:8080 (backend)

## Workflow: Testing Changes in Kind

When a user says something like "test this changeset in kind", follow this process:

### Step 1: Analyze Changes
```bash
# Check what files have changed
git status
git diff --name-only main...HEAD
```

Determine which components are affected:
- Changes in `components/backend/` → backend
- Changes in `components/frontend/` → frontend
- Changes in `components/operator/` → operator
- Changes in `components/runners/claude-code-runner/` → runner
- Changes in `components/runners/state-sync/` → state-sync
- Changes in `components/public-api/` → public-api

### Step 2: Explain the Plan
Tell the user:
```
I found changes in: [list of components]

To test these in kind, I'll:
1. Build the affected images: [list components]
2. Push them to a local registry or load into kind
3. Update the kind cluster to use these images
4. Verify the deployment

Note: By default, kind uses production Quay.io images. We'll need to:
- Build your changed components locally
- Load them into the kind cluster
- Update the deployments to use ImagePullPolicy: Never
```

### Step 3: Build Changed Components
```bash
# Build specific components (from /workspace/repos/platform)
cd /workspace/repos/platform

# Build backend (if changed)
make build-backend

# Build frontend (if changed)
make build-frontend

# Build operator (if changed)
make build-operator

# Build runner (if changed)
make build-runner

# Build state-sync (if changed)
make build-state-sync

# Build public-api (if changed)
make build-public-api

# Or build all at once
make build-all
```

### Step 4: Setup/Update Kind Cluster

**If cluster doesn't exist:**
```bash
# Create kind cluster
make kind-up
```

**If cluster exists, load new images:**
```bash
# Load images into kind
kind load docker-image localhost/vteam_backend:latest --name ambient-local
kind load docker-image localhost/vteam_frontend:latest --name ambient-local
kind load docker-image localhost/vteam_operator:latest --name ambient-local
# ... for each rebuilt component
```

### Step 5: Update Deployments
```bash
# Update deployments to use local images and Never pull policy
kubectl set image deployment/backend backend=localhost/vteam_backend:latest -n ambient-code
kubectl set image deployment/frontend frontend=localhost/vteam_frontend:latest -n ambient-code
kubectl set image deployment/operator operator=localhost/vteam_operator:latest -n ambient-code

# Update image pull policy
kubectl patch deployment backend -n ambient-code -p '{"spec":{"template":{"spec":{"containers":[{"name":"backend","imagePullPolicy":"Never"}]}}}}'
kubectl patch deployment frontend -n ambient-code -p '{"spec":{"template":{"spec":{"containers":[{"name":"frontend","imagePullPolicy":"Never"}]}}}}'
kubectl patch deployment operator -n ambient-code -p '{"spec":{"template":{"spec":{"containers":[{"name":"operator","imagePullPolicy":"Never"}]}}}}'

# Restart deployments to pick up new images
kubectl rollout restart deployment/backend -n ambient-code
kubectl rollout restart deployment/frontend -n ambient-code
kubectl rollout restart deployment/operator -n ambient-code
```

### Step 6: Verify Deployment
```bash
# Wait for rollout to complete
kubectl rollout status deployment/backend -n ambient-code
kubectl rollout status deployment/frontend -n ambient-code
kubectl rollout status deployment/operator -n ambient-code

# Check pod status
kubectl get pods -n ambient-code

# Check for errors
kubectl get events -n ambient-code --sort-by='.lastTimestamp'

# Get pod details if issues
kubectl describe pod -l app=backend -n ambient-code
kubectl logs -l app=backend -n ambient-code --tail=50
```

### Step 7: Provide Access Info
```
✓ Deployment complete!

Access the platform at:
- Frontend: http://localhost:8080 (or http://localhost with Docker)
- Test credentials: Check .env.test for the token

To view logs:
  kubectl logs -f -l app=backend -n ambient-code
  kubectl logs -f -l app=frontend -n ambient-code
  kubectl logs -f -l app=operator -n ambient-code

To teardown:
  make kind-down
```

## Workflow: Testing Changes in Minikube

When a user wants to test in minikube:

### Full Rebuild and Deploy
```bash
cd /workspace/repos/platform

# If cluster doesn't exist, this will create it and build everything
make local-up

# If cluster exists and you want to rebuild everything
make local-rebuild
```

### Incremental Updates (Faster)
```bash
# Just rebuild and reload specific components
make local-reload-backend   # If only backend changed
make local-reload-frontend  # If only frontend changed
make local-reload-operator  # If only operator changed
```

### Check Status
```bash
# Quick status check
make local-status

# Detailed troubleshooting
make local-troubleshoot

# Follow logs
make local-logs-backend
make local-logs-frontend
make local-logs-operator
```

## Common Tasks

### "Bring up a fresh cluster"
```bash
# With kind (uses Quay.io images)
make kind-up

# With minikube (builds from source)
make local-up
```

### "Rebuild everything and test"
```bash
# With minikube
cd /workspace/repos/platform
make local-rebuild

# With kind (requires manual steps)
cd /workspace/repos/platform
make build-all
# Then load images and update deployments (see Step 4-5 above)
```

### "Just rebuild the backend"
```bash
# With minikube
make local-reload-backend

# With kind
make build-backend
kind load docker-image localhost/vteam_backend:latest --name ambient-local
kubectl set image deployment/backend backend=localhost/vteam_backend:latest -n ambient-code
kubectl rollout restart deployment/backend -n ambient-code
kubectl rollout status deployment/backend -n ambient-code
```

### "Show me the logs"
```bash
# With minikube
make local-logs-backend
make local-logs-frontend
make local-logs-operator

# With kind (or minikube, direct kubectl)
kubectl logs -f -l app=backend -n ambient-code
kubectl logs -f -l app=frontend -n ambient-code
kubectl logs -f -l app=operator -n ambient-code
```

### "Tear down the cluster"
```bash
# With kind
make kind-down

# With minikube (keep cluster)
make local-down

# With minikube (delete cluster)
make local-clean
```

### "Check if cluster is healthy"
```bash
# With minikube
make local-status
make local-test-quick

# With kind or any cluster
kubectl get pods -n ambient-code
kubectl get events -n ambient-code --sort-by='.lastTimestamp'
kubectl get deployments -n ambient-code
```

## Troubleshooting

### Pods stuck in ImagePullBackOff
**Cause:** Cluster trying to pull images from registry but they don't exist or aren't accessible

**Solution for kind:**
```bash
# Ensure images are built locally
make build-all

# Load images into kind
kind load docker-image localhost/vteam_backend:latest --name ambient-local
kind load docker-image localhost/vteam_frontend:latest --name ambient-local
kind load docker-image localhost/vteam_operator:latest --name ambient-local

# Update image pull policy
kubectl patch deployment backend -n ambient-code -p '{"spec":{"template":{"spec":{"containers":[{"name":"backend","imagePullPolicy":"Never"}]}}}}'
```

**Solution for minikube:**
```bash
# Minikube should handle this automatically, but if issues persist:
make local-rebuild
```

### Pods stuck in CrashLoopBackOff
**Cause:** Application is crashing on startup

**Solution:**
```bash
# Check logs for the failing pod
kubectl logs -l app=backend -n ambient-code --tail=100

# Check pod events
kubectl describe pod -l app=backend -n ambient-code

# Common issues:
# - Missing environment variables
# - Database connection failures
# - Invalid configuration
```

### Port forwarding not working
**Cause:** Port already in use or forwarding process died

**Solution for minikube:**
```bash
# Kill existing port-forward processes
pkill -f "kubectl port-forward"

# Restart port forwarding
make local-up  # Will setup port forwarding again
```

**Solution for kind:**
```bash
# Check NodePort mapping
kubectl get svc -n ambient-code

# Manually setup port forwarding if needed
make kind-port-forward
```

### Changes not reflected
**Cause:** Old image cached or deployment not restarted

**Solution:**
```bash
# Force rebuild
make build-backend  # (or whatever component)

# Reload into cluster
kind load docker-image localhost/vteam_backend:latest --name ambient-local

# Force restart
kubectl rollout restart deployment/backend -n ambient-code
kubectl rollout status deployment/backend -n ambient-code

# Verify new pods are running
kubectl get pods -n ambient-code -l app=backend
kubectl describe pod -l app=backend -n ambient-code | grep Image:
```

## Environment Variables

Key environment variables that affect cluster behavior:

```bash
# Container runtime
CONTAINER_ENGINE=podman  # or docker

# Build platform
PLATFORM=linux/amd64     # or linux/arm64

# Namespace
NAMESPACE=ambient-code

# Registry (for pushing images)
REGISTRY=quay.io/your-org
```

## Best Practices

1. **Use kind for quick validation**: If you just need to verify changes work, kind is faster
2. **Use minikube for development**: Better tooling for iterative development with `local-reload-*` commands
3. **Always check logs**: After deploying, verify pods started successfully
4. **Clean up when done**: `make kind-down` or `make local-clean` to free resources
5. **Check what changed first**: Use `git status` and `git diff` to understand scope
6. **Build only what changed**: Don't rebuild everything if only one component changed
7. **Verify image pull policy**: Ensure deployments use `imagePullPolicy: Never` for local images

## Quick Reference

### Decision Tree: Which Cluster Type?

```
Do you need to test local code changes?
├─ No → Use kind (make kind-up)
│        Fast, uses production images
│
└─ Yes → Do you need to iterate frequently?
         ├─ No → Use kind with manual image loading
         │        Good for one-off tests
         │
         └─ Yes → Use minikube (make local-up)
                  Best for development with hot-reload
```

### Cheat Sheet

| Task | Kind | Minikube |
|------|------|----------|
| Create cluster | `make kind-up` | `make local-up` |
| Rebuild all | Build + load + update | `make local-rebuild` |
| Rebuild backend | Build + load + restart | `make local-reload-backend` |
| Check status | `kubectl get pods -n ambient-code` | `make local-status` |
| View logs | `kubectl logs -f -l app=backend -n ambient-code` | `make local-logs-backend` |
| Tear down | `make kind-down` | `make local-clean` |
| Access URL | http://localhost:8080 | http://localhost:3000 |

## When to Invoke This Skill

Invoke the dev-cluster skill when users say things like:
- "Test this in kind"
- "Deploy my changes to a local cluster"
- "Bring up a development environment"
- "I want to test these changes"
- "Create a kind cluster with my changes"
- "Rebuild and deploy the backend"
- "How do I test platform changes locally?"

## Example Interactions

### Example 1: Quick Test in Kind

User: "Test this changeset in kind"

Assistant (using dev-cluster skill):
1. Checks git status → finds backend changes
2. Explains: "I see changes in components/backend. I'll build the backend image, create a kind cluster, and deploy your changes."
3. Runs: `make build-backend`
4. Runs: `make kind-up`
5. Loads image: `kind load docker-image localhost/vteam_backend:latest --name ambient-local`
6. Updates deployment with local image and ImagePullPolicy: Never
7. Verifies: `kubectl rollout status deployment/backend -n ambient-code`
8. Provides access URL and log commands

Result: User can test their backend changes at http://localhost:8080

### Example 2: Incremental Development with Minikube

User: "I'm working on the frontend, set me up for quick iterations"

Assistant (using dev-cluster skill):
1. Runs: `make local-up` (creates cluster, builds all images)
2. Explains the reload commands available
3. User makes frontend changes
4. User says: "Reload the frontend"
5. Runs: `make local-reload-frontend`
6. Verifies deployment
7. User continues iterating with quick reload cycles

Result: Fast iteration loop for frontend development

### Example 3: Troubleshooting Deployment

User: "The backend pod is crash looping"

Assistant (using dev-cluster skill):
1. Checks pod status: `kubectl get pods -n ambient-code`
2. Gets logs: `kubectl logs -l app=backend -n ambient-code --tail=100`
3. Analyzes error messages
4. Checks events: `kubectl get events -n ambient-code --sort-by='.lastTimestamp'`
5. Identifies issue (e.g., missing env var, bad configuration)
6. Suggests fix
7. After fix applied, verifies: `kubectl rollout status deployment/backend -n ambient-code`

Result: Issue diagnosed and resolved

## Integration with Makefile

This skill knows all the relevant Makefile targets in /workspace/repos/platform:

- `make kind-up` - Create kind cluster
- `make kind-down` - Destroy kind cluster
- `make local-up` - Create minikube cluster with local builds
- `make local-down` - Stop minikube services
- `make local-clean` - Delete minikube cluster
- `make local-rebuild` - Rebuild all and restart
- `make local-reload-backend` - Rebuild/reload backend only
- `make local-reload-frontend` - Rebuild/reload frontend only
- `make local-reload-operator` - Rebuild/reload operator only
- `make build-all` - Build all container images
- `make build-backend` - Build backend image only
- `make build-frontend` - Build frontend image only
- `make build-operator` - Build operator image only
- `make local-status` - Check pod status
- `make local-logs-backend` - Follow backend logs
- `make local-logs-frontend` - Follow frontend logs
- `make local-logs-operator` - Follow operator logs