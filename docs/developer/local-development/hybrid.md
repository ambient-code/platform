# Hybrid Local Development

Run components locally (outside cluster) while using kind for dependencies. **Fastest iteration cycle.**

## Overview

```
Local Machine:                Kind Cluster:
- Frontend (npm run dev)  →   - Backend API
- Backend (go run)        →   - Operator
- Operator (go run)       →   - MinIO, CRDs

# OR mix and match - run what you're developing locally
```

**Benefits:**
- ⚡ Instant reloads (no image build/push)
- 🐛 Better debugging (direct logs, breakpoints)
- 🚀 Faster iteration (seconds vs minutes)

---

## Frontend Local Development

Run Next.js dev server locally, connect to kind backend:

**Terminal 1 - Port-forward backend:**
```bash
# Forward backend service to localhost:8090 (avoids conflict with ingress on 8080)
kubectl port-forward -n ambient-code svc/backend-service 8090:8080
# Keep this running - you'll see "Forwarding from 127.0.0.1:8090 -> 8080"
```

**Terminal 2 - Run frontend:**
```bash
cd components/frontend

# Create .env.local with config
cp .env.example .env.local

# Get token and add to .env.local
echo "NEXT_PUBLIC_E2E_TOKEN=$(kubectl get secret test-user-token -n ambient-code -o jsonpath='{.data.token}' | base64 -d)" >> .env.local
echo "OC_TOKEN=$(kubectl get secret test-user-token -n ambient-code -o jsonpath='{.data.token}' | base64 -d)" >> .env.local

# Run dev server
npm run dev

# Access at http://localhost:3000
```

**Fast iteration:**
- Edit frontend code
- Save → Auto-reload in browser
- No container builds

**Troubleshooting:**
- `ECONNREFUSED`: Port-forward not running → Check Terminal 1
- `ENOTFOUND`: Wrong backend URL → Check BACKEND_URL in .env.local

---

## Backend Local Development

Run Go backend locally and run frontend locally

```bash
# Setup minimal kind cluster (just operator + minio + CRDs)
make kind-up

# Run backend locally on 8090
cd components/backend
export KUBECONFIG=~/.kube/config
export PORT=8090
go run .

# Run frontend locally following above instructions for frontend local development

# Access at http://localhost:3000
```

**Fast iteration:**
- Edit backend code
- Save → restart (or use `air` for hot reload)
- Full kubectl access to kind cluster

---

## Operator Local Development

Run operator locally, watch kind cluster:

```bash
# Setup kind cluster (backend + frontend + minio)
make kind-up

# Stop operator in cluster
kubectl scale -n ambient-code deployment/agentic-operator --replicas=0

# Run operator locally
cd components/operator
export KUBECONFIG=~/.kube/config
export AMBIENT_CODE_RUNNER_IMAGE=quay.io/ambient_code/vteam_claude_runner:latest
export STATE_SYNC_IMAGE=quay.io/ambient_code/vteam_state_sync:latest
go run .

# Operator watches CRs in kind, creates pods
```

**Fast iteration:**
- Edit operator code
- Stop → rebuild → run (~10 seconds)
- Watch logs directly in terminal

---

## Full Local Stack (Advanced)

Run everything locally except MinIO:

```bash
# Create minimal kind cluster (just MinIO + CRDs)
make kind-up
kubectl scale -n ambient-code deployment/backend-api deployment/frontend deployment/agentic-operator --replicas=0

# Terminal 1: Operator
cd components/operator && go run .

# Terminal 2: Backend
cd components/backend && go run . --port 8080

# Terminal 3: Frontend
cd components/frontend && NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api npm run dev

# Access at http://localhost:3000
```

---

## Tips


### Required Environment Variables

**Frontend:**
- `BACKEND_URL` - Backend URL for Next.js server-side API routes (e.g., `http://localhost:8080/api`)
- `NEXT_PUBLIC_API_BASE_URL` - API base for client-side (use `/api` for Next.js proxy)
- `NEXT_PUBLIC_E2E_TOKEN` - Auth token (same value)

**Backend:**
- `KUBECONFIG` - Path to kubeconfig (for k8s client)
- `PORT` - Server port (default 8080, use 8090 to avoid conflicts)

**Operator:**
- `KUBECONFIG` - Path to kubeconfig
- `AMBIENT_CODE_RUNNER_IMAGE` - Runner image to use
- `STATE_SYNC_IMAGE` - State-sync image to use

### Debugging

Local processes are easier to debug:
- **VS Code**: Attach debugger to Go processes
- **Browser DevTools**: Full React component inspection
- **Direct logs**: No kubectl logs needed

---

## When to Use

**Use hybrid local dev when:**
- ✅ Rapid frontend UI changes
- ✅ Debugging backend API logic
- ✅ Developing operator reconciliation

**Use full kind cluster when:**
- ✅ Testing full integration
- ✅ Testing container-specific issues
- ✅ Running e2e tests

---

## See Also

- [Kind Local Dev](kind.md) - Full cluster in kind
- [Minikube Local Dev](../../QUICK_START.md) - Minikube setup
