# Multi-Agent Iterative Development Workflow

This document defines the protocol for multi-agent development of the Ambient Code Platform.
It governs how agent sessions coordinate, integrate, and iterate using the `feat/integration`
branch and the local kind cluster as the shared testing ground.

---

## Roles

| Agent | Branch | Owns |
|-------|--------|------|
| **Overlord** | `feat/integration` | Integration branch, all make commands, builds, scripts, deployment |
| **API** | `feat/session-messages` | `ambient-api-server`, `ambient-sdk`, `ambient-cli`, `components/frontend` |
| **CP** | `feat/grpc-python-runner` | `ambient-control-plane`, `components/backend`, `components/operator`, `components/runners` |

---

## Protocol

### The Cycle

```
1. RESET     → Overlord resets feat/integration to main
2. PICK      → Overlord cherry-picks all commits from API and CP branches
3. BUILD     → Overlord runs: make kind-up LOCAL_IMAGES=true (or kind-rebuild if cluster exists)
4. OBSERVE   → All agents observe logs, errors, pod status
5. FIX       → API fixes API-owned components; CP fixes CP-owned components
6. COMMIT    → API and CP commit fixes to their respective branches
7. GOTO 1   → Overlord resets and cherry-picks again for a clean build verification
```

### Why Reset + Cherry-Pick?

A clean cherry-pick from main verifies that every change is atomic and purposeful.
It prevents accumulated debt from merge artifacts. Each cycle starts from a known-good
baseline (main) with an explicit set of improvements from each agent.

---

## Overlord Responsibilities

### Integration Branch Management

```bash
# Reset feat/integration to main
git checkout feat/integration
git reset --hard main

# Cherry-pick from API branch (feat/session-messages)
git cherry-pick <commit1> <commit2> ...

# Cherry-pick from CP branch (feat/grpc-python-runner)
git cherry-pick <commit1> <commit2> ...
```

**When to reset:** Start of each new dev cycle, or when the branch accumulates conflicts
that cannot be resolved cleanly.

**Never directly commit** API-owned or CP-owned changes to `feat/integration`.
Cherry-pick only from their branches.

### Building and Deploying

```bash
# Fresh cluster (clean start)
make kind-down && make kind-up LOCAL_IMAGES=true

# Iterate on existing cluster (faster)
make kind-rebuild
```

#### Loading Images into Kind (Podman)

`kind load docker-image` fails on this system with a podman/containerd snapshotter mismatch.
The correct method is `podman save → podman cp → ctr import`:

```bash
# Generic pattern for any image
_load_image_into_kind() {
  local IMAGE=$1
  local TAR=/tmp/kind-load-$(date +%s).tar
  podman save "$IMAGE" -o "$TAR"
  podman cp "$TAR" ambient-local-control-plane:/tmp/kind-load.tar
  podman exec ambient-local-control-plane ctr -n k8s.io images import /tmp/kind-load.tar
  podman exec ambient-local-control-plane rm -f /tmp/kind-load.tar
  rm -f "$TAR"
}

# Rebuild single component after CP or API commits a fix
make build-api-server
_load_image_into_kind localhost/vteam_api_server:latest
kubectl rollout restart deployment/ambient-api-server -n ambient-code
kubectl rollout status deployment/ambient-api-server -n ambient-code

make build-control-plane
_load_image_into_kind localhost/ambient_control_plane:latest
kubectl rollout restart deployment/ambient-control-plane -n ambient-code
kubectl rollout status deployment/ambient-control-plane -n ambient-code

make build-runner
_load_image_into_kind localhost/vteam_claude_runner:latest
# No deployment restart needed — picked up by next runner pod
```

> **Note:** `make kind-rebuild` uses `kind load docker-image` internally and may fail.
> Until the Makefile is updated to use the `podman save | cp | ctr` path,
> run component rebuilds manually using the pattern above.

### Observing Cluster State

```bash
# All pods status
kubectl get pods -n ambient-code

# Control plane reconciler logs
kubectl logs -n ambient-code deployment/ambient-control-plane -f

# API server logs
kubectl logs -n ambient-code deployment/ambient-api-server -f

# Session namespaces (runner pods land here)
kubectl get namespaces | grep session-
kubectl get pods -A | grep session-
```

---

## API Agent Responsibilities

- Own all changes to: `ambient-api-server`, `ambient-sdk`, `ambient-cli`, `components/frontend`
- Commit all fixes and features to `feat/session-messages`
- Do NOT commit to `feat/integration` — Overlord cherry-picks from your branch
- When a build fails due to your component: fix and commit to your branch, notify Overlord to rebuild

### Signaling Overlord

Post to the blackboard when a fix is committed:
```
[API] fix committed: <sha> — <description>
Overlord: please rebuild ambient-api-server and test
```

---

## CP Agent Responsibilities

- Own all changes to: `ambient-control-plane`, `components/backend`, `components/operator`, `components/runners`
- Commit all fixes and features to `feat/grpc-python-runner`
- Do NOT commit to `feat/integration` — Overlord cherry-picks from your branch
- When a build fails due to your component: fix and commit to your branch, notify Overlord

### Signaling Overlord

Post to the blackboard when a fix is committed:
```
[CP] fix committed: <sha> — <description>
Overlord: please rebuild ambient-control-plane and test
```

---

## Documentation as Spec (Spec-Driven Development)

All documentation in `docs/internal/developer/` is **specification**, not description.
It describes what the system *should* do. When behavior diverges from the doc, that's a bug.

**Key spec files:**

| File | Specifies |
|------|-----------|
| `local-development/kind.md` | `make kind-up LOCAL_IMAGES=true` steps, components, known issues |
| `local-development/agent-workflow.md` | This file — multi-agent protocol |
| `local-development/README.md` | Environment selection guide |

**When you find a gap:** Update the relevant spec file as part of the fix commit.
The docs live in `feat/integration` and are maintained by Overlord.

---

## Blackboard Protocol

All agents post status to the coordinator at `http://localhost:8899/spaces/sdk-backend-replacement/`.

**Required fields in every post:**

```json
{
  "status": "active | idle | blocked",
  "summary": "<Agent>: <one-line status>",
  "branch": "feat/<your-branch>",
  "items": ["completed action 1", "completed action 2"],
  "next_steps": "what you're doing next or waiting for"
}
```

**Tag conventions:**
- `[?BOSS]` — needs human decision
- `[?API]` — question for API agent
- `[?CP]` — question for CP agent
- `[?Overlord]` — question for Overlord

---

## Known Issues Tracking

Known issues are tracked in `docs/internal/developer/local-development/kind.md` under
the **Known Issues** section. Each issue includes:

1. Symptom (observable error)
2. Cause (root cause)
3. Impact (what breaks)
4. Owner (API or CP agent)
5. Fix status (pending / in-progress / resolved)

**Current open issues:**

| Issue | Owner | Status | Impact |
|-------|-------|--------|--------|
| RBAC `deletecollection` forbidden | CP | ✅ RESOLVED | Session cleanup leaves orphaned pods |
| `kind-rebuild` skips api-server + control-plane images | Overlord | ✅ RESOLVED | Stale images after rebuild |
| `RUNNER_IMAGE` in kind-local uses quay.io | CP | ✅ RESOLVED | Runner pods fail without internet |
| `kind load docker-image` fails (podman/containerd snapshotter mismatch) | Overlord | ✅ DOCUMENTED | Images not loaded; use `podman save → podman cp → ctr import` |
| `migration` init container does not re-run on `kind-rebuild` | API | 🔴 OPEN | `sessions` table missing after api-server image update without pod delete |
| `ensureImagePullAccess()` creates failing RoleBinding in kind | CP | 🔴 OPEN | 403 error on session provision — kind has no internal registry to pull from |
| Runner pod lands in `namespace=default` (missing `project_id` on session) | API+CP | 🔴 OPEN | Sessions without `project_id` fall back to `default` namespace |

---

## Session End-to-End Test

After a successful `make kind-up LOCAL_IMAGES=true`, verify the full flow:

```bash
# 1. Get token
TOKEN=$(kubectl get secret test-user-token -n ambient-code \
  -o jsonpath='{.data.token}' | base64 -d)

# 2. Port-forward API server (use 18000 to avoid conflicts with stale port-forwards)
kubectl port-forward svc/ambient-api-server 18000:8000 -n ambient-code &

# 3. Create a project
curl -s -X POST http://localhost:18000/api/ambient/v1/projects \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"smoke-test","display_name":"Smoke Test"}' | python3 -m json.tool

# 4. Create a session (no initial_prompt — idle, waiting for messages)
curl -s -X POST http://localhost:18000/api/ambient/v1/sessions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Ambient-Project: smoke-test" \
  -d '{"name":"smoke-session","project_id":"smoke-test"}' | python3 -m json.tool
# Save the returned "id" as SESSION_ID

# 5. Watch control plane provision the runner
kubectl logs -n ambient-code deployment/ambient-control-plane -f --since=30s
# Expected: "session event received", "provisioning session", "runner pod created", "session phase updated Running"

# 6. Wait for runner pod to be Running
kubectl get pods -n smoke-test -w

# 7. Send a message via REST
SESSION_ID=<id from step 4>
curl -s -X POST http://localhost:18000/api/ambient/v1/sessions/${SESSION_ID}/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"event_type":"user","payload":"my name is Mark. remember it."}' | python3 -m json.tool
# Expected: {"seq": N, "event_type": "user", ...}

# 8. Watch runner logs — confirm gRPC watch picked up the message
kubectl logs -n smoke-test -l ambient-code.io/session-id=${SESSION_ID} -f
# Expected: "Inbound user message received: session=... seq=N"
# Expected: "Run: thread_id=..., run_id=..."
# Expected: "Inbound message forwarded to runner (...), consuming SSE stream"

# 9. Poll for Claude's response in the messages stream
curl -s http://localhost:18000/api/ambient/v1/sessions/${SESSION_ID}/messages \
  -H "Authorization: Bearer $TOKEN" | python3 -c "
import sys, json, re
msgs = json.load(sys.stdin)
for m in msgs:
    et = m['event_type']
    p = str(m.get('payload', ''))
    if et == 'user':
        print(f'\n>>> USER (seq={m[\"seq\"]}): {m[\"payload\"]}')
    elif et == 'TEXT_MESSAGE_CONTENT':
        d = re.search(r\"delta='([^']*)'\" , p)
        if d: print(d.group(1), end='', flush=True)
    elif et == 'RUN_FINISHED':
        print(f'\n[RUN_FINISHED seq={m[\"seq\"]}]')
"
# Expected: USER message, Claude reply text, RUN_FINISHED

# 10. Send a follow-up to verify conversation memory
curl -s -X POST http://localhost:18000/api/ambient/v1/sessions/${SESSION_ID}/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"event_type":"user","payload":"what is my name?"}' | python3 -m json.tool
# Re-run step 9 after ~30s — Claude should reply "Your name is Mark."

# 11. Test tool use
curl -s -X POST http://localhost:18000/api/ambient/v1/sessions/${SESSION_ID}/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"event_type":"user","payload":"create a file called hello.txt with the contents \"hello Mark\" in the workspace"}' | python3 -m json.tool

# Verify file was created in runner pod
kubectl exec -n smoke-test \
  $(kubectl get pods -n smoke-test -o jsonpath='{.items[0].metadata.name}') \
  -- cat /workspace/artifacts/hello.txt
# Expected: hello Mark
```

Expected flow:
1. POST project → CP creates namespace `smoke-test`
2. POST session → ambient-api-server stores session
3. gRPC watch → control-plane receives `ADDED` event
4. Control-plane provisions: namespace, secret, runner pod, service
5. Control-plane calls `UpdateStatus` → session transitions to `phase: Running`
6. REST `POST /messages` → stored in api-server (seq N)
7. Runner background watcher receives via gRPC stream → forwards to `POST localhost:8001/`
8. Claude runs, AG-UI events streamed back and pushed to gRPC
9. `GET /messages` returns all events including `TEXT_MESSAGE_CONTENT` and `RUN_FINISHED`

**Verified results (2026-03-13):**

| Turn | User | Claude | Tool |
|------|------|--------|------|
| 1 | `my name is Mark. remember it.` | `Got it, remember that.` | — |
| 2 | `what is my name?` | `Your name is Mark.` | — |
| 3 | `create a file called hello.txt with "hello Mark"` | `Done!` | `Write("/workspace/artifacts/hello.txt")` ✅ |

---

## Quick Reference Commands

```bash
# Cluster lifecycle
make kind-up LOCAL_IMAGES=true   # Full build + deploy (clean start)
make kind-rebuild                 # Rebuild all + reload + restart (faster)
make kind-down                    # Destroy cluster

# Status
kubectl get pods -n ambient-code
kubectl get deployments -n ambient-code -o wide

# Logs
kubectl logs -n ambient-code deployment/ambient-control-plane -f
kubectl logs -n ambient-code deployment/ambient-api-server -f

# Access
make kind-port-forward            # Frontend: http://localhost:8080

# Token
kubectl get secret test-user-token -n ambient-code -o jsonpath='{.data.token}' | base64 -d

# Session namespaces
kubectl get namespaces | grep -v kube | grep -v ambient-code | grep -v default

# Clean sessions
kubectl get namespaces | grep session | awk '{print $1}' | xargs kubectl delete namespace
```
