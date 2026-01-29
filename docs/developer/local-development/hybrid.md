# Hybrid Local Development

Run components locally (outside cluster) while using kind for dependencies. **Fastest iteration cycle.**

## Overview

Choose which components to run locally based on what you're developing:

| Scenario | Local | In Cluster | Port-Forward | Best For |
|----------|-------|------------|--------------|----------|
| **Frontend Only** | Frontend | Backend, Operator, MinIO | Backend → 8090 | UI/UX work |
| **Frontend + Backend** | Frontend, Backend | Operator, MinIO | None | API development |
| **Full Stack** | Frontend, Backend, Operator | MinIO only | None | Operator work |

**Benefits:**
- ⚡ Instant reloads (no image build/push)
- 🐛 Better debugging (direct logs, breakpoints)
- 🚀 Faster iteration (seconds vs minutes)

---

## Scenario 1: Frontend Only

**Best for:** UI/UX work, React components, styling

Run Next.js dev server locally, connect to backend in cluster via port-forward.

```
Frontend (localhost:3000) → Backend (cluster:8090) → K8s API
```

### Setup

**Terminal 1 - Port-forward backend:**
```bash
# Forward backend service to localhost:8090
kubectl port-forward -n ambient-code svc/backend-service 8090:8080
```

**Terminal 2 - Run frontend:**
```bash
cd components/frontend

# Set backend URL to port-forwarded backend
export BACKEND_URL=http://localhost:8090/api

# Run dev server
npm run dev

# Access at http://localhost:3000
```

### What's Happening

- Frontend talks to backend via port-forward tunnel
- Backend runs in cluster, has full K8s access
- Operator in cluster handles sessions

### Fast Iteration

- Edit React components → instant hot reload
- Edit styles → instant update
- No backend restarts needed

---

## Scenario 2: Frontend + Backend

**Best for:** Backend API work, handler logic, new endpoints

Run frontend and backend locally, operator stays in cluster.

```
Frontend (localhost:3000) → Backend (localhost:8090) → K8s API (via KUBECONFIG)
                                                       ↓
                                          Operator (cluster) watches CRs
```

### Setup

**One-time: Create minimal cluster**
```bash
# Start kind, scale down components we'll run locally
make kind-up
kubectl scale -n ambient-code deployment/backend-api deployment/frontend --replicas=0
```

**Terminal 1 - Backend:**
```bash
cd components/backend
export KUBECONFIG=~/.kube/config  # Direct K8s API access
export PORT=8090
go run .
```

**Terminal 2 - Frontend:**
```bash
cd components/frontend
export BACKEND_URL=http://localhost:8090/api
npm run dev

# Access at http://localhost:3000
```

### What's Happening

- **No port-forwarding needed!**
- Backend uses `KUBECONFIG` to talk directly to K8s API
- Backend creates/reads CRs, operator in cluster reacts to them
- Frontend talks to local backend

### Fast Iteration

- Edit backend code → restart (few seconds)
- Edit frontend code → instant hot reload
- See logs directly in terminal
- Full debugging with breakpoints

---

## Scenario 3: Full Local Stack

**Best for:** Operator development, reconciliation logic, full integration testing

Run everything locally except MinIO and runner jobs.

```
Frontend (localhost:3000) → Backend (localhost:8090) → K8s API (via KUBECONFIG)
                                                       ↓
                                          Operator (localhost) → K8s API (via KUBECONFIG)
                                                                ↓
                                                   Creates runner jobs in cluster
```

### Setup

**One-time: Create minimal cluster**
```bash
# Start kind, scale down all components we'll run locally
make kind-up
kubectl scale -n ambient-code deployment/backend-api deployment/frontend deployment/agentic-operator --replicas=0
```

**Terminal 1 - Operator:**
```bash
cd components/operator
export KUBECONFIG=~/.kube/config
export AMBIENT_CODE_RUNNER_IMAGE=quay.io/ambient_code/vteam_claude_runner:latest
export STATE_SYNC_IMAGE=quay.io/ambient_code/vteam_state_sync:latest
go run .
```

**Terminal 2 - Backend:**
```bash
cd components/backend
export KUBECONFIG=~/.kube/config
export PORT=8090
go run .
```

**Terminal 3 - Frontend:**
```bash
cd components/frontend
export BACKEND_URL=http://localhost:8090/api
npm run dev

# Access at http://localhost:3000
```

### What's Happening

- **No port-forwarding needed!**
- All components use `KUBECONFIG` for direct K8s API access
- Operator watches for CR changes, creates runner jobs in cluster
- MinIO stays in cluster (for session state storage)
- Runner jobs still run as pods (containerized execution)

### Fast Iteration

- Edit operator code → restart (~10 seconds)
- Edit backend code → restart (~5 seconds)  
- Edit frontend code → instant hot reload
- See all logs in separate terminals
- Full debugging across entire stack

---

## VS Code Tasks

We've created VS Code tasks for quick access:

**Kind Cluster:**
- `Kind: Start Cluster` - Create kind cluster with all components
- `Kind: Stop Cluster` - Delete kind cluster
- `Kind: Port-Forward Backend` - Forward backend to localhost:8090
- `Kind: Port-Forward Frontend` - Forward frontend to localhost:3000

**Hybrid Development:**
- `Hybrid: Frontend Only` - Run frontend + port-forward backend
- `Hybrid: Frontend + Backend` - Run frontend + backend locally
- `Hybrid: Full Local Stack` - Run all three locally

Access via `Cmd+Shift+P` → "Tasks: Run Task"

---

## Understanding KUBECONFIG vs Port-Forwarding

**Common confusion:** Many think `export KUBECONFIG=~/.kube/config` is port-forwarding. It's not!

**`KUBECONFIG`:**
- Gives your local Go processes direct access to the Kubernetes API
- They can create/read CRs, pods, secrets, deployments, etc.
- This is why backend and operator don't need port-forwarding

**Port-forwarding (`kubectl port-forward`):**
- Tunnels traffic to a **service** running inside the cluster
- Only needed when you want to access a service's HTTP endpoint from localhost
- Example: Frontend needs to call backend API running in cluster

**When you need port-forwarding:**
- ✅ Scenario 1 (Frontend Only) - frontend needs to reach backend service in cluster
- ❌ Scenario 2 (Frontend + Backend) - backend runs locally, frontend talks to localhost
- ❌ Scenario 3 (Full Stack) - everything local, no services in cluster to reach

---

## Tips & Troubleshooting

### Required Environment Variables

**Frontend:**
- `BACKEND_URL=http://localhost:8090/api` - Backend URL for Next.js server-side routes
- `NEXT_PUBLIC_API_BASE_URL=/api` - Client-side API base (use `/api` for Next.js proxy)

**Backend:**
- `KUBECONFIG=~/.kube/config` - Path to kubeconfig (for K8s API access)
- `PORT=8090` - Server port (avoid 8080 conflict with ingress)

**Operator:**
- `KUBECONFIG=~/.kube/config` - Path to kubeconfig
- `AMBIENT_CODE_RUNNER_IMAGE` - Runner image (e.g., `quay.io/ambient_code/vteam_claude_runner:latest`)
- `STATE_SYNC_IMAGE` - State-sync image (e.g., `quay.io/ambient_code/vteam_state_sync:latest`)

### Debugging

Local processes are much easier to debug:
- **VS Code Go Debugger**: Set breakpoints in backend/operator code
- **Browser DevTools**: Full React component inspection, network tab
- **Direct logs**: See logs in terminal, no `kubectl logs` needed
- **Fast iteration**: Change code → see results in seconds

### Common Issues

**Backend can't connect to K8s API:**
```bash
# Verify KUBECONFIG is set and valid
echo $KUBECONFIG
kubectl get pods -n ambient-code
```

**Frontend can't reach backend:**
```bash
# Scenario 1: Check port-forward is running
lsof -i:8090

# Scenario 2/3: Check backend is running locally
curl http://localhost:8090/health
```

**Operator not creating jobs:**
```bash
# Check operator is running and watching
# Should see logs about watching AgenticSessions

# Check CRDs exist
kubectl get crd agenticsessions.vteam.ambient-code
```

---

## When to Use Each Scenario

| Task | Recommended Scenario | Why |
|------|---------------------|-----|
| **UI/UX changes** | Frontend Only | Fastest - only need frontend hot reload |
| **New API endpoint** | Frontend + Backend | Test backend logic with fast restarts |
| **Handler debugging** | Frontend + Backend | Set breakpoints in backend code |
| **Operator reconciliation** | Full Stack | See operator logs directly |
| **Integration testing** | Full Kind Cluster | Test real container behavior |
| **E2E testing** | Full Kind Cluster | Run Cypress tests |

**General rule:** Run the minimum number of components locally that you need to work on.

---

## See Also

- [Kind Local Dev](kind.md) - Full cluster in kind
- [VS Code Tasks](.vscode/tasks.json) - Quick access to dev commands
- [Testing Strategy](../testing/e2e-guide.md) - E2E testing
