---
name: dev-cluster
description: >
  Manages Ambient Code Platform development clusters (kind) for testing changes
  locally. Use when deploying PRs to kind, bringing up local clusters, rebuilding
  images, troubleshooting pod issues, or running benchmarks. Triggers on: "test
  in kind", "deploy locally", "kind cluster", "rebuild images", "pod crashing",
  "bring up cluster", "kind-up", "dev environment", "local dev".
---

# Development Cluster Management Skill

> **Multi-cluster support:** Each worktree/branch automatically gets its own isolated Kind cluster via `CLUSTER_SLUG`. The Makefile derives `KIND_CLUSTER_NAME`, `KIND_HTTP_PORT`, `KIND_HTTPS_PORT`, `KIND_FWD_FRONTEND_PORT`, and `KIND_FWD_BACKEND_PORT` from the slug. Run `make kind-status` to see the current assignments. Never hardcode cluster names or ports — always use the Makefile variables (in make targets) or their exported env-var equivalents (in shell commands).

You are an expert **Ambient Code Platform (ACP) DevOps Specialist**. Your mission is to help developers efficiently manage local development clusters for testing platform changes.

## Your Role

Help developers test their code changes in local Kubernetes clusters (kind) by:
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
| **Runner** | `components/runners/ambient-runner` | `vteam_claude_runner:latest` | Python Claude Code runner |
| **State Sync** | `components/runners/state-sync` | `vteam_state_sync:latest` | S3 persistence service |
| **Public API** | `components/public-api` | `vteam_public_api:latest` | External API gateway |

## Development Cluster: Kind

**Best for:** All development, quick testing, CI/CD alignment

**Commands:**
- `make kind-up` - Create cluster, deploy with Quay.io images
- `make kind-up LOCAL_IMAGES=true` - Create cluster, build and load local images
- `make kind-up LOCAL_IMAGES=true LOCAL_VERTEX=true` - Same but with Vertex AI enabled (reads `ANTHROPIC_VERTEX_PROJECT_ID`, `CLOUD_ML_REGION`, `GOOGLE_APPLICATION_CREDENTIALS` from shell env)
- `make kind-down` - Destroy cluster
- `make kind-rebuild` - Rebuild all components, reload images, restart
- `make kind-port-forward` - Setup port forwarding
- `make kind-status` - Show cluster status and port assignments

**Characteristics:**
- Uses production Quay.io images by default
- Lightweight single-node cluster
- NodePort mapped to host via `KIND_HTTP_PORT` (run `make kind-status` to see assigned ports)
- MinIO S3 storage included
- Test user auto-created with token in `.env.test`

**Access:** `http://localhost:$KIND_FWD_FRONTEND_PORT` (run `make kind-status` for assigned ports)

## CRITICAL: Always Use `make kind-up`, Never Manually Recover

**If `make kind-up` fails partway through, fix the root cause and re-run `make kind-up` (after `make kind-down`).** Do NOT manually patch deployments, create buckets, or set env vars to recover — the Makefile handles MinIO bucket creation, Vertex AI setup, token extraction, and port forwarding in a specific order. Manually recovering individual steps is slower, error-prone, and skips steps you don't know about.

## Vertex AI / GCP Configuration

When the user needs Vertex AI (Claude via GCP) instead of Anthropic API:

```bash
# One-command setup — reads env vars from shell profile
make kind-up LOCAL_IMAGES=true LOCAL_VERTEX=true
```

**Prerequisites:** These env vars must be set in the user's shell (check `~/.bashrc` or `~/.zshrc`):
- `ANTHROPIC_VERTEX_PROJECT_ID` — GCP project ID
- `CLOUD_ML_REGION` — Vertex AI region (e.g., `us-east5`)
- `GOOGLE_APPLICATION_CREDENTIALS` — Path to service account JSON or ADC file

The `LOCAL_VERTEX=true` flag runs `scripts/setup-vertex-kind.sh` which creates the `ambient-vertex` secret, patches `operator-config`, and restarts the operator. **Do not do these steps manually.**

## Feature Flags (Unleash)

The platform uses **Unleash** for feature flags, running in-cluster. Some endpoints are gated behind feature flags and will return 404 if the flag is not enabled. If you hit an unexpected 404 on an endpoint that exists in the code, check whether it's behind a `requireFeatureFlag()` middleware and ensure the flag is created and enabled in Unleash (admin token: `*:*.unleash-admin-token`, API at `http://localhost:4242` inside the cluster).

## Custom Workflow Branches

To test workflow changes from a different branch of `ambient-code/workflows`:

```bash
kubectl set env deployment/backend-api -n ambient-code \
  OOTB_WORKFLOWS_BRANCH="your-branch-name"
kubectl rollout restart deployment/backend-api -n ambient-code
```

The backend caches workflows for 5 minutes. Restart clears the cache immediately.

## Google OAuth for Integrations

Testing Google Drive or other Google integrations requires OAuth credentials on the backend:

```bash
kubectl set env deployment/backend-api -n ambient-code \
  GOOGLE_OAUTH_CLIENT_ID="your-client-id" \
  GOOGLE_OAUTH_CLIENT_SECRET="your-secret" \
  OAUTH_STATE_SECRET="$(openssl rand -hex 32)" \
  BACKEND_URL="http://localhost:$KIND_HTTP_PORT"
```

## Workflow: Setting Up from a PR

When a user provides a PR URL or number, follow this process:

### Step 1: Fetch PR Details
```bash
# Get PR metadata (title, branch, changed files, state)
gh pr view <PR_NUMBER> --json title,headRefName,files,state,body
```

### Step 2: Checkout the PR Branch
```bash
git fetch origin <branch_name>
git checkout <branch_name>
```

### Step 3: Determine Affected Components
Analyze the changed files from the PR to identify which components need rebuilding (see component mapping below). Then follow the kind cluster workflow.

## Detecting the Container Engine

**Before any build step**, detect which container engine is available:

```bash
# Check which engine is available
if command -v docker &>/dev/null && docker info &>/dev/null 2>&1; then
    CONTAINER_ENGINE=docker
elif command -v podman &>/dev/null && podman info &>/dev/null 2>&1; then
    CONTAINER_ENGINE=podman
else
    echo "ERROR: No container engine available"
    exit 1
fi
```

**Always pass `CONTAINER_ENGINE=` to make commands:**
```bash
make build-frontend CONTAINER_ENGINE=docker
make build-all CONTAINER_ENGINE=docker
```

## Detecting the Access URL

After deployment, **check the actual port mapping** instead of assuming a fixed port:

```bash
# For kind with Docker: check the container's published ports
docker ps --filter "name=$KIND_CLUSTER_NAME" --format "{{.Ports}}"
# Example output: 0.0.0.0:80->30080/tcp  → access at http://localhost
# Example output: 0.0.0.0:<port>->30080/tcp → access at http://localhost:<port>

# Quick connectivity test
curl -s -o /dev/null -w "%{http_code}" http://localhost:80
```

**Port mapping depends on the container engine:**
- **Docker**: host port 80 → http://localhost
- **Podman**: host port from `KIND_HTTP_PORT` → check `make kind-status`

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
- Changes in `components/runners/ambient-runner/` → runner
- Changes in `components/runners/state-sync/` → state-sync
- Changes in `components/public-api/` → public-api

### Step 2: Explain the Plan
Tell the user:
```
I found changes in: [list of components]

To test these in kind, I'll:
1. Build the affected images: [list components]
2. Load them into the kind cluster
3. Update the kind cluster to use these images
4. Verify the deployment

Note: By default, kind uses production Quay.io images. We'll need to:
- Build your changed components locally
- Load them into the kind cluster
- Update the deployments to use ImagePullPolicy: Never
```

### Step 3: Build Changed Components

**Important:** Detect the container engine first (see "Detecting the Container Engine" above), then pass it to all build commands.

```bash
# Build specific components — always pass CONTAINER_ENGINE
make build-backend CONTAINER_ENGINE=$CONTAINER_ENGINE
make build-frontend CONTAINER_ENGINE=$CONTAINER_ENGINE
make build-operator CONTAINER_ENGINE=$CONTAINER_ENGINE
make build-runner CONTAINER_ENGINE=$CONTAINER_ENGINE
make build-state-sync CONTAINER_ENGINE=$CONTAINER_ENGINE
make build-public-api CONTAINER_ENGINE=$CONTAINER_ENGINE

# Or build all at once
make build-all CONTAINER_ENGINE=$CONTAINER_ENGINE
```

### Step 4: Setup/Update Kind Cluster

**If cluster doesn't exist:**
```bash
# Create kind cluster with local images
make kind-up LOCAL_IMAGES=true
```

**If cluster exists, rebuild and reload:**
```bash
# Rebuild all, load images, restart deployments
make kind-rebuild
```

**Or load individual images:**
```bash
kind load docker-image localhost/vteam_backend:latest --name $KIND_CLUSTER_NAME
kind load docker-image localhost/vteam_frontend:latest --name $KIND_CLUSTER_NAME
kind load docker-image localhost/vteam_operator:latest --name $KIND_CLUSTER_NAME
```

### Step 5: Verify Deployment
```bash
# Wait for rollout to complete
kubectl rollout status deployment/backend -n ambient-code
kubectl rollout status deployment/frontend -n ambient-code
kubectl rollout status deployment/operator -n ambient-code

# Check pod status
kubectl get pods -n ambient-code

# Check for errors
kubectl get events -n ambient-code --sort-by='.lastTimestamp'
```

### Step 6: Provide Access Info

**Detect the actual URL** by checking the kind container's port mapping (see "Detecting the Access URL" above), then provide the correct URL to the user.

```
✓ Deployment complete!

Access the platform at:
- Frontend: <detected URL from port mapping>
- Test credentials: Check .env.test for the token

To view logs:
  kubectl logs -f -l app=backend -n ambient-code
  kubectl logs -f -l app=frontend -n ambient-code
  kubectl logs -f -l app=operator -n ambient-code

To teardown:
  make kind-down
```

## Common Tasks

### "Bring up a fresh cluster"
```bash
make kind-up
```

### "Rebuild everything and test"
```bash
make kind-rebuild
```

### "Just rebuild the backend"
```bash
make build-backend CONTAINER_ENGINE=$CONTAINER_ENGINE
kind load docker-image localhost/vteam_backend:latest --name $KIND_CLUSTER_NAME
kubectl set image deployment/backend backend=localhost/vteam_backend:latest -n ambient-code
kubectl rollout restart deployment/backend -n ambient-code
kubectl rollout status deployment/backend -n ambient-code
```

### "Show me the logs"
```bash
kubectl logs -f -l app=backend -n ambient-code
kubectl logs -f -l app=frontend -n ambient-code
kubectl logs -f -l app=operator -n ambient-code
```

### "Tear down the cluster"
```bash
make kind-down
```

### "Check if cluster is healthy"
```bash
kubectl get pods -n ambient-code
kubectl get events -n ambient-code --sort-by='.lastTimestamp'
kubectl get deployments -n ambient-code
```

## Troubleshooting

### Pods stuck in ImagePullBackOff
**Cause:** Cluster trying to pull images from registry but they don't exist or aren't accessible

**Solution:**
```bash
# Ensure images are built locally
make build-all CONTAINER_ENGINE=$CONTAINER_ENGINE

# Load images into kind
kind load docker-image localhost/vteam_backend:latest --name $KIND_CLUSTER_NAME
kind load docker-image localhost/vteam_frontend:latest --name $KIND_CLUSTER_NAME
kind load docker-image localhost/vteam_operator:latest --name $KIND_CLUSTER_NAME

# Update image pull policy
kubectl patch deployment backend -n ambient-code -p '{"spec":{"template":{"spec":{"containers":[{"name":"backend","imagePullPolicy":"Never"}]}}}}'
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

### Sessions fail with init-hydrate exit code 1
**Cause:** MinIO `ambient-sessions` bucket doesn't exist. This happens when `make kind-up` fails partway through (e.g., due to image pull errors) and the `init-minio.sh` step is skipped.

**Solution:** Fix the underlying issue (e.g., image pull errors) and re-run `make kind-down && make kind-up`. The Makefile runs `init-minio.sh` near the end of `kind-up`, which creates the required buckets. If `make kind-up` completes successfully, the bucket will exist.

### Port forwarding not working
**Cause:** Port already in use or forwarding process died

**Solution:**
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
kind load docker-image localhost/vteam_backend:latest --name $KIND_CLUSTER_NAME

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
# Container runtime (detect automatically — see "Detecting the Container Engine")
CONTAINER_ENGINE=docker  # or podman

# Build platform
PLATFORM=linux/amd64     # or linux/arm64

# Namespace
NAMESPACE=ambient-code

# Registry (for pushing images)
REGISTRY=quay.io/your-org
```

## Fast Inner-Loop: Run Frontend Locally (No Image Rebuilds)

For **frontend-only changes**, skip image rebuilds entirely. Run NextJS locally with hot-reload against the backend in the kind cluster:

```bash
# Terminal 1: port-forward backend from kind cluster
kubectl port-forward svc/backend-service $KIND_FWD_BACKEND_PORT:8080 -n ambient-code

# Terminal 2: set up frontend with auth token
cd components/frontend
npm install  # first time only

# Create .env.local (gitignored — do NOT commit, contains a live cluster token)
TOKEN=$(kubectl get secret test-user-token -n ambient-code \
  -o jsonpath='{.data.token}' | base64 -d)
cat > .env.local <<EOF
OC_TOKEN=$TOKEN
BACKEND_URL=http://localhost:$KIND_FWD_BACKEND_PORT/api
EOF

npm run dev
# Open http://localhost:3000
```

**Why this works:**
- `BACKEND_URL` points NextJS API routes to the port-forwarded backend
- `OC_TOKEN` is forwarded as both `X-Forwarded-Access-Token` and `Authorization: Bearer` headers (the backend's `ExtractServiceAccountFromAuth` reads `Authorization` for JWT parsing)
- Every file save triggers instant hot-reload — no Docker build, no kind load, no rollout restart

**When to use:**
- Frontend-only changes (components, styles, pages, API routes)
- Iterating on UI features rapidly
- Debugging frontend issues

**When NOT to use:**
- Backend, operator, or runner changes (those still need image rebuild + load)
- Testing changes to container configuration or deployment manifests

## Benchmarking Developer Loops

Use the benchmark harness when the user wants measured cold-start or rebuild timing rather than ad hoc impressions.

### Commands

```bash
# Human-friendly local summary
make benchmark

# Agent / automation friendly output
make benchmark FORMAT=tsv

# Single component
make benchmark COMPONENT=frontend MODE=cold
make benchmark COMPONENT=backend MODE=warm
```

### Agent Guidance

- Prefer `FORMAT=tsv` when another agent, script, or evaluation harness will consume the output.
- Prefer the default `human` format for interactive local use in a terminal.
- `frontend` benchmarking requires **Node.js 20+**.
- `warm` currently measures **rebuild proxies**, not browser-observed hot reload latency.
- If `reports/benchmarks/` is not writable in the current environment, the harness will fall back to a temp directory and print a warning.
- Session benchmarking is **contract-only** in v1 (`bench_session_*` stubs in `scripts/benchmarks/bench-manifest.sh`).
- Start with the **smallest relevant benchmark**:
  - backend/operator/public-api change -> `MODE=warm COMPONENT=<component> REPEATS=1`
  - frontend contributor setup -> `MODE=cold COMPONENT=frontend REPEATS=1`
  - only run all components when you explicitly need the whole matrix
- Treat preflight failures as useful environment signals; do not work around them unless the user asks.
- Use full-sweep benchmarking sparingly because each component still performs untimed setup before the measured warm rebuild.

### Interpreting Results

- `cold`: approximates first-contributor setup/install cost with isolated caches
- `warm`: approximates incremental rebuild cost after setup has already completed
- `budget_ok=false` on cold runs means the component exceeded the 60-second contributor budget
- Large deltas on a single repeat should be treated cautiously; use more repeats before drawing conclusions

## Best Practices

1. **Use local dev server for frontend**: Fastest feedback loop, no image rebuilds needed
2. **Use kind for backend/operator validation**: Rebuild with `make kind-rebuild`
3. **Always check logs**: After deploying, verify pods started successfully
4. **Clean up when done**: `make kind-down` to free resources
5. **Check what changed first**: Use `git status` and `git diff` to understand scope
6. **Build only what changed**: Don't rebuild everything if only one component changed
7. **Verify image pull policy**: Ensure deployments use `imagePullPolicy: Never` for local images

## Quick Reference

### Decision Tree: Which Approach?

```
Do you need to test local code changes?
├─ No → Use kind (make kind-up)
│        Fast, uses production images
│
└─ Yes → Is the change frontend-only?
         ├─ Yes → Run locally with npm run dev
         │        Instant hot-reload, no image builds
         │
         └─ No → Use kind with local images
                  make kind-up LOCAL_IMAGES=true (first time)
                  make kind-rebuild (subsequent rebuilds)
```

### Cheat Sheet

| Task | Command |
|------|---------|
| Create cluster | `make kind-up` |
| Create cluster with local images | `make kind-up LOCAL_IMAGES=true` |
| Rebuild all | `make kind-rebuild` |
| Check status | `make kind-status` |
| View logs | `kubectl logs -f -l app=backend -n ambient-code` |
| Tear down | `make kind-down` |
| Access URL | `make kind-status` for ports (`$KIND_FWD_FRONTEND_PORT`) |

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
5. Loads image: `kind load docker-image localhost/vteam_backend:latest --name $KIND_CLUSTER_NAME`
6. Updates deployment with local image and ImagePullPolicy: Never
7. Verifies: `kubectl rollout status deployment/backend -n ambient-code`
8. Provides access URL and log commands

Result: User can test their backend changes at `http://localhost:$KIND_FWD_FRONTEND_PORT` (run `make kind-status` to see the assigned port)

### Example 2: Iterative Development

User: "I'm working on the frontend, set me up for quick iterations"

Assistant (using dev-cluster skill):
1. Runs: `make kind-up` (creates cluster)
2. Explains the local dev server approach for frontend
3. Sets up port-forward and local NextJS dev server
4. User makes frontend changes with instant hot-reload

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

This skill knows all the relevant Makefile targets:

- `make kind-up` - Create kind cluster
- `make kind-up LOCAL_IMAGES=true` - Create kind cluster with locally-built images
- `make kind-down` - Destroy kind cluster
- `make kind-rebuild` - Rebuild all, reload images, restart deployments
- `make kind-port-forward` - Port-forward services to localhost
- `make kind-status` - Show cluster status and port assignments
- `make build-all` - Build all container images
- `make build-backend` - Build backend image only
- `make build-frontend` - Build frontend image only
- `make build-operator` - Build operator image only
- `make local-status` - Check pod status
- `make local-logs` - Follow all component logs
