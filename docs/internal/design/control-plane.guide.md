# Control Plane + Runner: Implementation Guide

**Date:** 2026-03-22
**Status:** Living Document
**Spec:** `control-plane.spec.md` — CP architecture, runner structure, message streams, proposed changes
**Dev Context:** `.claude/context/control-plane-development.md` — build commands, known invariants, pre-commit checklists, runner architecture

---

## This Document Is Iterative

Each time this guide is invoked, start from Step 1, follow the steps in order, and update this document with what you learned. The goal is convergence. Expect gaps. Fix the guide before moving on.

---

## Overview

This guide covers implementation work on two components:

- **CP** (`components/ambient-control-plane/`) — Go service; K8s reconciler; session provisioner
- **Runner** (`components/runners/ambient-runner/`) — Python FastAPI service; Claude bridge; gRPC client

Changes to these components are independent of the api-server pipeline (no openapi.yaml, no SDK generator). They are deployed as container images to a kind cluster and tested via `acpctl`.

---

> **Build commands, known invariants, and pre-commit checklists** → see `.claude/context/control-plane-development.md`

IMPORTANT!!! **ONLY PUSH TO quay.io/vteam-*:mgt-001**  ensure the tag is *only* mgt-001
This is important! we are bypassing the build process by pushing directly to quay.io.  this is risky
so *only* push with tag mgt-001

---

## Gap Table (Current)

```
ITEM                                         COMPONENT    STATUS      GAP
──────────────────────────────────────────────────────────────────────────────
assistant payload → plain string             Runner       closed      GRPCMessageWriter._write_message() fixed (Wave 1)
reasoning leaks into DB record               Runner       closed      reasoning stays in /events SSE only (Wave 1)
GET /events/{thread_id}                      Runner       closed      endpoints/events.py added
Namespace delete RBAC                        CP manifests closed      delete added to namespaces ClusterRole (Wave 2)
MPP namespace naming (ambient-code--test)    CP           closed      NamespaceName() on provisioner interface (Run 3)
OIDC token provider (was static k8s SA)      CP           closed      mgt-001 image + OIDC env vars (Run 3)
Per-project RBAC in session namespaces       CP           closed      ensureControlPlaneRBAC() in project reconciler (Run 3)
AMBIENT_GRPC_ENABLED not injected            CP           closed      boolToStr(RunnerGRPCURL != "") in buildEnv (Run 3)
gRPC auth: RH SSO token rejected             api-server   closed      --grpc-jwk-cert-url=sso.redhat.com JWKS (Run 3)
NetworkPolicy blocks runner->api-server      manifests    closed      allow-ambient-tenant-ingress netpol (Run 3)
GET /sessions/{id}/events (proxy)            api-server   closed      StreamRunnerEvents in plugins/sessions/handler.go:282
acpctl session events <id>                   CLI          closed      events.go exists; fixed missing X-Ambient-Project header
INITIAL_PROMPT gRPC push warning             Runner       closed      skip push when grpc_url set (message already in DB)
acpctl messages -f hang                      CLI          closed      replaced gRPC watch with HTTP long-poll (ListMessages)
acpctl send -f                               CLI          closed      added --follow flag; calls streamMessages after push
assistant payload JSON blob (grpc_transport)  Runner       closed      _write_message() now pushes plain text (Run 8)
TenantSA RBAC race on namespace re-create    CP           open        see Known Races below
Runner credential fetch → /credentials/{id}/token  Runner    open        Credential Kind live (PR #1110); CP integration + runner update pending (Wave 5)
```

---

## Workflow Steps

### Step 1 — Acknowledge Iteration

- [ ] Read `control-plane.spec.md` top to bottom
- [ ] Note the gap table above
- [ ] Confirm the running kind cluster name: `podman ps | grep kind | grep control-plane`
- [ ] Confirm CP is running: `kubectl get deploy ambient-control-plane -n ambient-code`

### Step 2 — Read the Spec

Read `control-plane.spec.md` in full. Hold in working memory:

- The two message streams and what belongs in each
- The proposed `GRPCMessageWriter` payload change
- The `GET /events/{thread_id}` runner endpoint (already done)
- The `GET /sessions/{id}/events` api-server proxy (not yet done)
- The namespace delete RBAC gap

### Step 3 — Current Gap Table

Use the table above. Update it as items close.

### Step 4 — Waves

#### Wave 1 — Runner: Fix assistant payload (no upstream dependency)

**File:** `components/runners/ambient-runner/ambient_runner/bridges/claude/grpc_transport.py`

**Target:** `GRPCMessageWriter._write_message()`

**What to do:**

Replace the full JSON blob with the assistant text only:

```python
async def _write_message(self, status: str) -> None:
    if self._grpc_client is None:
        logger.warning(
            "[GRPC WRITER] No gRPC client — cannot push: session=%s",
            self._session_id,
        )
        return

    assistant_text = next(
        (
            m.get("content", "")
            for m in self._accumulated_messages
            if m.get("role") == "assistant"
        ),
        "",
    )

    if not assistant_text:
        logger.warning(
            "[GRPC WRITER] No assistant message in snapshot: session=%s run=%s messages=%d",
            self._session_id,
            self._run_id,
            len(self._accumulated_messages),
        )

    logger.info(
        "[GRPC WRITER] PushSessionMessage: session=%s run=%s status=%s text_len=%d",
        self._session_id,
        self._run_id,
        status,
        len(assistant_text),
    )

    self._grpc_client.session_messages.push(
        self._session_id,
        event_type="assistant",
        payload=assistant_text,
    )
```

**Acceptance:**
- Create a session, send a message, check `acpctl session messages <id> -o json`
- `event_type=assistant` payload is plain text, not JSON
- `reasoning` content is absent from the DB record
- CLI `-f` can display it alongside `event_type=user` without JSON parsing

**Build + push runner image after this change.**

---

#### Wave 2 — CP Manifests: Namespace delete RBAC

**Files:** `components/manifests/base/` (or wherever CP RBAC is defined)

**What to do:**

Find the CP ClusterRole and add `delete` on `namespaces`:

```yaml
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list", "create", "delete"]
```

**Verify:**

After deploy, delete a session and confirm namespace is removed:

```bash
acpctl delete session <id>
kubectl get ns  # should not show the session namespace
```

---

#### Wave 3 — api-server: `GET /sessions/{id}/events` proxy

**Repo:** `platform-api-server` (separate repo — file this as a Wave 4 BE item in the ambient-model guide)

**What to do:**

In `components/ambient-api-server/plugins/sessions/`:

1. Add `StreamRunnerEvents` handler to `handler.go`:

```go
func (h *sessionHandler) StreamRunnerEvents(w http.ResponseWriter, r *http.Request) {
    id := mux.Vars(r)["id"]
    session, err := h.sessionSvc.Get(r.Context(), id)
    if err != nil || session.KubeCrName == nil || session.KubeNamespace == nil {
        w.WriteHeader(http.StatusNotFound)
        return
    }

    runnerURL := fmt.Sprintf(
        "http://session-%s.%s.svc.cluster.local:8001/events/%s",
        *session.KubeCrName, *session.KubeNamespace, *session.KubeCrName,
    )

    req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, runnerURL, nil)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    req.Header.Set("Accept", "text/event-stream")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        w.WriteHeader(http.StatusBadGateway)
        return
    }
    defer resp.Body.Close()

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no")
    w.WriteHeader(http.StatusOK)
    if f, ok := w.(http.Flusher); ok {
        f.Flush()
    }

    io.Copy(w, resp.Body)
    if f, ok := w.(http.Flusher); ok {
        f.Flush()
    }
}
```

2. Register in `plugin.go`:

```go
sessionsRouter.HandleFunc("/{id}/events", sessionHandler.StreamRunnerEvents).Methods(http.MethodGet)
```

3. Add to `openapi/openapi.sessions.yaml`:

```yaml
/sessions/{id}/events:
  get:
    summary: Stream live AG-UI events from runner pod
    description: |
      SSE stream of all AG-UI events for the active run. Proxies the runner pod's
      /events/{thread_id} endpoint. Ephemeral — no replay. Ends when RUN_FINISHED
      or RUN_ERROR is received, or the client disconnects.
    parameters:
      - name: id
        in: path
        required: true
        schema:
          type: string
    responses:
      '200':
        description: SSE event stream
        content:
          text/event-stream:
            schema:
              type: string
      '404':
        description: Session not found
      '502':
        description: Runner pod not reachable
```

**Acceptance:**

```bash
# With a running session and active run:
curl -N http://localhost:8000/api/ambient/v1/sessions/{id}/events
# Should stream AG-UI events until RUN_FINISHED
```

---

#### Wave 5 — Runner: Migrate credential fetch to Credential Kind API

**File:** `components/runners/ambient-runner/ambient_runner/platform/auth.py`

**What to do:**

1. The CP must inject a `CREDENTIAL_IDS` env var into the runner pod — a JSON-encoded map of `provider → credential_id` resolved for this session. Resolution follows the RBAC scope resolver (agent → project → global, narrower wins per provider). The CP must read visible credentials from the api-server and build this map before pod creation.

2. The runner's `_fetch_credential(context, credential_type)` must be updated to call the new endpoint:

```python
# Instead of:
url = f"{base}/projects/{project}/agentic-sessions/{session_id}/credentials/{credential_type}"

# New:
credential_ids = json.loads(os.getenv("CREDENTIAL_IDS", "{}"))
credential_id = credential_ids.get(credential_type)
if not credential_id:
    logger.warning(f"No credential_id for provider {credential_type}")
    return {}
url = f"{base}/api/ambient/v1/credentials/{credential_id}/token"
```

3. The hostname allowlist on `BACKEND_API_URL` must be preserved (same env var, same check).

4. The response field mapping in `populate_runtime_credentials()` must be updated — the new token response shape uses `token` uniformly (no more `apiToken` for Jira):

| Provider | Old field | New field |
|----------|-----------|-----------|
| `github`  | `token`    | `token`   |
| `gitlab`  | `token`    | `token`   |
| `jira`    | `apiToken` | `token`   |
| `google`  | `accessToken` | `token` (full SA JSON string) |

5. The CP must grant `credential:token-reader` on each injected credential ID to the runner pod's service account at session start. This is a platform-internal RoleBinding created by the CP's KubeReconciler, not via user-facing `POST /role_bindings`.

**Acceptance:**
- Create a `gitlab` Credential via `acpctl credential create`
- Create a session; verify CP injects `CREDENTIAL_IDS={"gitlab": "<id>"}` env var into the pod
- Runner fetches `GET /credentials/<id>/token`; `GITLAB_TOKEN` is set in the pod
- `ruff format .` and `ruff check .` pass

---

#### Wave 4 — CLI: `acpctl session events`

**Repo:** `platform-control-plane/components/ambient-cli/`

**What to do:**

Add `eventsCmd` to `cmd/acpctl/session/`:

```go
var eventsCmd = &cobra.Command{
    Use:   "events <session-id>",
    Short: "Stream live AG-UI events from an active session run",
    Args:  cobra.ExactArgs(1),
    RunE:  runEvents,
}

func runEvents(cmd *cobra.Command, args []string) error {
    sessionID := args[0]
    client := // get SDK client
    ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
    defer cancel()

    url := fmt.Sprintf("%s/api/ambient/v1/sessions/%s/events",
        client.BaseURL(), url.PathEscape(sessionID))

    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    req.Header.Set("Accept", "text/event-stream")
    req.Header.Set("Authorization", "Bearer "+client.Token())

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    scanner := bufio.NewScanner(resp.Body)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "data: ") {
            printEventLine(cmd, line[6:])
        }
    }
    return scanner.Err()
}
```

Register in `cmd.go`:
```go
Cmd.AddCommand(eventsCmd)
```

**Acceptance:**

```bash
acpctl session events <id>
# Shows tokens streaming as Claude responds
# Exits on RUN_FINISHED or Ctrl+C
```

---

## MPP One-Time Bootstrap (Manual Steps)

These two steps are performed **once per cluster** — not per session, not per namespace. Both are cluster-scoped bootstraps that the automation cannot self-provision due to MPP RBAC constraints.

### Step A — TenantServiceAccount (⚠️ manual, one-time)

Applied directly to `ambient-code--config` (not via kustomize — the global `namespace:` in kustomization.yaml would override it):

```bash
kubectl apply -f components/manifests/overlays/mpp-openshift/ambient-cp-tenant-sa.yaml
```

**What it does:** The tenant-access-operator watches for this CR and:
- Creates SA `tenantaccess-ambient-control-plane` in `ambient-code--config`
- Creates a long-lived token Secret `tenantaccess-ambient-control-plane-token` in `ambient-code--config`
- Automatically injects a `namespace-admin` RoleBinding into every current and **future** tenant namespace

This is **not** required per session namespace. The operator handles propagation automatically whenever MPP creates a new `ambient-code--<project>` namespace.

**After applying**, copy the token Secret to `ambient-code--runtime-int` so the CP can mount it:

```bash
kubectl get secret tenantaccess-ambient-control-plane-token \
  -n ambient-code--config \
  -o json \
  | python3 -c "
import json, sys
s = json.load(sys.stdin)
del s['metadata']['namespace']
del s['metadata']['resourceVersion']
del s['metadata']['uid']
del s['metadata']['creationTimestamp']
s['metadata'].pop('ownerReferences', None)
s['metadata'].pop('annotations', None)
s['type'] = 'Opaque'
print(json.dumps(s))
" | kubectl apply -n ambient-code--runtime-int -f -
```

### Step B — Static Runner API Token (obsolete as of Run 5)

This step is **no longer required**. The runner authenticates to the api-server using the CP's OIDC client-credentials JWT (`preferred_username: service-account-ocm-ams-service`). The api-server validates both RH SSO service account JWTs and end-user JWTs via `kid`-keyed JWKS lookup (rh-trex-ai v0.0.27). `isServiceCaller()` in the gRPC handler compares the JWT `preferred_username` claim against `GRPC_SERVICE_ACCOUNT` env var — no static token needed.

---

## Known Races

### TenantServiceAccount RBAC race on namespace re-create

**Symptom:**

```
secrets is forbidden: User "system:serviceaccount:ambient-code--config:tenantaccess-ambient-control-plane"
cannot create resource "secrets" in API group "" in the namespace "ambient-code--test"
```

**When it occurs:** Delete a session → immediately create a new session in the **same project** (`test`). MPP deletes and re-creates the `ambient-code--test` namespace. The tenant-access-operator needs time to inject the `namespace-admin` RoleBinding into the fresh namespace. If CP reconciles before that propagation completes (~20-40s), secret creation is forbidden.

**The handler fails and does not retry** — the informer only re-fires on the next API server event. The session gets stuck in `""` phase with no pod.

**Workaround (manual):** After deleting a session, wait at least 30-60 seconds before creating a new one in the same project. Or create the new session in a different project.

**Proper fix (not yet implemented):** CP's `ensureSecret` (and other namespace-scoped ops) should retry on `k8serrors.IsForbidden` with exponential backoff, since the cause is transient RBAC propagation. See `kube_reconciler.go:328`.

---

> **Known invariants** → see `.claude/context/control-plane-development.md`

---

## Verification Playbook

After any wave:

```bash
# 0. Login (OCM token expires ~15 min — always refresh before testing)
ocm login --use-auth-code   # browser popup
acpctl login --token $(ocm token) \
  --url https://ambient-api-server-ambient-code--runtime-int.internal-router-shard.mpp-w2-preprod.cfln.p1.openshiftapps.com \
  --insecure-skip-tls-verify

# 1. List sessions
acpctl get sessions

# 2. Create a test session
acpctl create session --project foo --name verify-N "what is 2+2?"

# 3. Watch CP logs for provisioning
oc logs -n ambient-code--runtime-int deployment/ambient-control-plane -f --tail=20

# 4. Watch runner logs
POD=$(oc get pods -n ambient-code--test -l ambient-code.io/session-id=<id> -o name | head -1)
oc logs -n ambient-code--test $POD -f

# 5. Check messages in DB
acpctl session messages <id>

# 6. Verify assistant payload is plain text
acpctl session messages <id> -o json | python3 -c "
import json, sys
msgs = json.load(sys.stdin)
for m in msgs:
    print(m['event_type'], repr(m['payload'][:80]))
"
```

Expected after Wave 1:
```
user 'what is 2+2?'
assistant '2+2 equals 4.'
```

Not:
```
assistant '{"run_id": "...", "status": "completed", "messages": [...]}'
```

---

## Mandatory Image Push Playbook

After every code change, run this sequence before testing:

```bash
# Find cluster container name
CLUSTER_CTR=$(podman ps --format '{{.Names}}' | grep 'control-plane' | head -1)
echo "Cluster container: $CLUSTER_CTR"

# Build runner (always --no-cache to pick up Python source changes)
cd components/runners && podman build --no-cache -t localhost/vteam_claude_runner:latest -f ambient-runner/Dockerfile . && cd ../..
podman save localhost/vteam_claude_runner:latest | \
  podman exec -i ${CLUSTER_CTR} ctr --namespace=k8s.io images import -

# Build CP (always --no-cache to pick up Go source changes)
podman build --no-cache -f components/ambient-control-plane/Dockerfile -t localhost/ambient_control_plane:latest components
# Remove old image from containerd before importing (prevents stale digest)
podman exec ${CLUSTER_CTR} ctr --namespace=k8s.io images rm localhost/ambient_control_plane:latest 2>/dev/null || true
podman save localhost/ambient_control_plane:latest | \
  podman exec -i ${CLUSTER_CTR} ctr --namespace=k8s.io images import -
kubectl rollout restart deployment/ambient-control-plane -n ambient-code
kubectl rollout status deployment/ambient-control-plane -n ambient-code --timeout=90s

# Verify CP pod is running the new digest
kubectl get pod -n ambient-code -l app=ambient-control-plane \
  -o jsonpath='{.items[0].status.containerStatuses[0].imageID}'
```

Runner image changes take effect on the next new session pod — no restart needed.

**Image names (actual deployment):**
- CP deployment image: `localhost/ambient_control_plane:latest`
- Runner pod image: `localhost/vteam_claude_runner:latest`
- `make build-control-plane` builds `localhost/vteam_control_plane:latest` — **wrong name**, use the `podman build` command above instead

---

## Run Log

### Run 1 — 2026-03-22

**Status:** Spec and guide written. Wave 1 (assistant payload) queued. `GET /events/{thread_id}` already implemented.

**Gap table at start:**

```
ITEM                                    COMPONENT    STATUS
GET /events/{thread_id}                 Runner       closed (endpoints/events.py)
assistant payload → plain string        Runner       open
GET /sessions/{id}/events (proxy)       api-server   open
acpctl session events <id>              CLI          open
Namespace delete RBAC                   CP manifests open
```

**Lessons:**
- Runner image must be rebuilt and pushed to kind for every Python change — no hot reload in pods
- `make build-runner` must be run from the repo root (not the component dir)
- kind cluster name is `ambient-main` (not derived from branch name) — always verify with `podman ps`
- `acpctl session messages -f` now shows assistant payloads as raw JSON — Wave 1 will fix this

### Run 2 — 2026-03-22

**Status:** Wave 1 + Wave 2 complete.

**Changes:**
- `grpc_transport.py`: `_write_message()` now pushes plain assistant text only; `json` import removed; ruff clean
- `components/manifests/base/rbac/control-plane-clusterrole.yaml`: added `delete` to namespaces verbs

**Gap table after Run 2:**

```
ITEM                                    COMPONENT    STATUS
GET /events/{thread_id}                 Runner       closed
assistant payload → plain string        Runner       closed (Wave 1)
Namespace delete RBAC                   CP manifests closed (Wave 2)
GET /sessions/{id}/events (proxy)       api-server   open
acpctl session events <id>              CLI          open
```

**Next steps:**
- Build + push runner image: `make build-runner` then push to kind
- Apply manifests for RBAC fix: `kubectl apply -f components/manifests/base/rbac/control-plane-clusterrole.yaml`
- Verify: create session, check `acpctl session messages <id> -o json` — assistant payload should be plain text

---

### Run 3 — 2026-03-27 (MPP OpenShift Integration)

**Status:** All MPP-specific gaps closed. NetworkPolicy applied. Pending: end-to-end gRPC push verification.

**Context:** This run targeted the MPP/OpenShift environment (`ambient-code--runtime-int`), not a kind cluster. Multiple layered issues resolved to get a runner pod to reach the api-server via gRPC.

**Changes made:**

| File | Change |
|------|--------|
| `overlays/mpp-openshift/ambient-control-plane.yaml` | Image `mgt-001`, `imagePullPolicy: Always`, added `OIDC_CLIENT_ID`/`OIDC_CLIENT_SECRET`/`RUNNER_IMAGE` env vars |
| `overlays/mpp-openshift/ambient-api-server-args-patch.yaml` | Added `--grpc-jwk-cert-url` pointing to RH SSO JWKS endpoint |
| `overlays/mpp-openshift/ambient-tenant-ingress-netpol.yaml` | New NetworkPolicy allowing ports 8000+9000 ingress from `tenant.paas.redhat.com/tenant: ambient-code` namespaces |
| `overlays/mpp-openshift/kustomization.yaml` | Added netpol to resources |
| `kubeclient/namespace_provisioner.go` | `NamespaceName(projectID)` added to interface; `MPPNamespaceProvisioner` returns `ambient-code--<id>` |
| `reconciler/project_reconciler.go` | `namespaceForProject()` replaced with `provisioner.NamespaceName()`; added `ensureControlPlaneRBAC()` |
| `reconciler/shared.go` | Removed `namespaceForSession` free function and unused imports |
| `reconciler/kube_reconciler.go` | `namespaceForSession` as method; added `AMBIENT_GRPC_ENABLED` env var injection |
| `config/config.go` | Added `CPRuntimeNamespace` field (`CP_RUNTIME_NAMESPACE`, default `ambient-code--runtime-int`) |
| `cmd/ambient-control-plane/main.go` | Updated `NewProjectReconciler` call with `cfg.CPRuntimeNamespace` |

**Root causes resolved (in order encountered):**

1. **Static token provider** — CP image was `latest`/`IfNotPresent`. Fixed: `mgt-001` + `imagePullPolicy: Always` + OIDC env vars.
2. **`namespace test did not become Active`** — `waitForNamespaceActive` polled for `test` but MPP creates `ambient-code--test`. Fixed: `namespaceName(instanceID)` helper.
3. **Secrets/pods forbidden in `ambient-code--test`** — `namespaceForProject` and `namespaceForSession` returned raw project ID. Fixed: all callers use `provisioner.NamespaceName()`.
4. **Pods forbidden in `default`** — tombstone events had no namespace. Fixed by `namespaceForSession` method.
5. **No RBAC for CP SA in session namespaces** — cannot create ClusterRoles. Fixed: `ensureControlPlaneRBAC()` creates per-namespace Role+RoleBinding on project reconcile.
6. **Runner used HTTP backend** — `AMBIENT_GRPC_ENABLED` not injected. Fixed: `boolToStr(r.cfg.RunnerGRPCURL != "")`.
7. **gRPC auth rejected** — Session creds are RH SSO OIDC tokens; api-server only validated cluster JWKS. Fixed: `--grpc-jwk-cert-url` arg.
8. **UNAVAILABLE: failed to connect** — `internal-1` NetworkPolicy blocks cross-namespace ingress. Fixed: additive `allow-ambient-tenant-ingress` NetworkPolicy.

**MPP-specific invariants learned:**

- MPP creates namespaces as `ambient-code--<tenantNamespaceCRName>` — never the raw project ID
- All tenant namespaces carry label `tenant.paas.redhat.com/tenant: ambient-code` — use for NetworkPolicy selectors
- MPP `internal-1` NetworkPolicy is managed by platform operator — add additive policies alongside it
- CP SA cannot create ClusterRoles — use per-namespace Roles only
- OCM login: `ocm login --use-auth-code` -> `acpctl login --token $(ocm token)`
- Use internal router shard hostname for `acpctl login` URL (not `.apps.` route)
- Two manual one-time bootstrap steps required — see **MPP One-Time Bootstrap** section above

**Gap table after Run 3:**

```
ITEM                                         COMPONENT    STATUS
──────────────────────────────────────────────────────────────────
assistant payload -> plain string            Runner       closed
reasoning leaks into DB record               Runner       closed
GET /events/{thread_id}                      Runner       closed
Namespace delete RBAC                        CP manifests closed
MPP namespace naming                         CP           closed
OIDC token provider                          CP           closed
Per-project RBAC in session namespaces       CP           closed
AMBIENT_GRPC_ENABLED injection               CP           closed
gRPC auth: RH SSO token                      api-server   closed
NetworkPolicy: runner->api-server            manifests    closed
GET /sessions/{id}/events (proxy)            api-server   open
acpctl session events <id>                   CLI          open
```

**Next:** Delete stale sessions, create `mpp-verify-5`, confirm gRPC push succeeds end-to-end.

---

### Run 5 — 2026-03-27 (Multi-JWKS auth + isServiceCaller via JWT claim)

**Status:** `WatchSessionMessages PERMISSION_DENIED` resolved. Runner receives user messages and invokes Claude. New blocker: `ImportError: cannot import name 'clear_runtime_credentials'` — fixed and runner image rebuilt.

**Root cause of PERMISSION_DENIED:** Runner's `BOT_TOKEN` is the CP's OIDC client-credentials JWT with `preferred_username: service-account-ocm-ams-service`. The api-server's `WatchSessionMessages` checked `middleware.IsServiceCaller(ctx)` which relied on a static opaque token pre-auth interceptor — not compatible with JWT-only auth chain.

**Solution:** JWT-based service identity via `kid`-keyed JWKS lookup.

**Changes made:**

| File | Change |
|------|--------|
| `ambient-api-server/go.mod` | Bumped `rh-trex-ai` v0.0.26 → v0.0.27 (`JwkCertURLs []string`, multi-URL JWKS support) |
| `plugins/sessions/grpc_handler.go` | Removed `middleware` import; added `serviceAccountName` field + `isServiceCaller()` checking JWT `preferred_username` claim |
| `plugins/sessions/plugin.go` | Reads `GRPC_SERVICE_ACCOUNT` env, passes to `NewSessionGRPCHandler` |
| `environments/e_*.go` | `JwkCertURL` → `JwkCertURLs []string` (compile fix for v0.0.27) |
| `overlays/mpp-openshift/ambient-api-server.yaml` | Added `GRPC_SERVICE_ACCOUNT=service-account-ocm-ams-service` env var |
| `overlays/mpp-openshift/ambient-api-server-args-patch.yaml` | Added K8s cluster JWKS as second URL (comma-separated) |
| `runners/ambient_runner/platform/auth.py` | Added `clear_runtime_credentials()` — clears `JIRA_URL/TOKEN/EMAIL`, `GITLAB_TOKEN`, `GITHUB_TOKEN`, `USER_GOOGLE_EMAIL`, and Google credentials file |

**How `isServiceCaller()` works:**
- `GRPC_SERVICE_ACCOUNT=service-account-ocm-ams-service` set on api-server deployment
- `AuthStreamInterceptor` validates JWT via JWKS `kid` lookup (RH SSO or K8s cluster), extracts `preferred_username` into context
- Handler's `isServiceCaller(ctx)` compares `auth.GetUsernameFromContext(ctx)` to the configured SA name
- Match → ownership check bypassed → `WatchSessionMessages` opens
- No match → user-path → ownership check enforced

**MPP login invariant (must do before every test):**

```bash
ocm login --use-auth-code   # browser popup — approve on laptop
acpctl login --token $(ocm token) \
  --url https://ambient-api-server-ambient-code--runtime-int.internal-router-shard.mpp-w2-preprod.cfln.p1.openshiftapps.com \
  --insecure-skip-tls-verify
acpctl get sessions
```

**Gap table after Run 5:**

```
ITEM                                         COMPONENT    STATUS
──────────────────────────────────────────────────────────────────────────
assistant payload -> plain string            Runner       closed
reasoning leaks into DB record               Runner       closed
GET /events/{thread_id}                      Runner       closed
Namespace delete RBAC                        CP manifests closed
MPP namespace naming                         CP           closed
OIDC token provider                          CP           closed
Per-project RBAC in session namespaces       CP           closed (TenantServiceAccount)
AMBIENT_GRPC_ENABLED injection               CP           closed
gRPC auth: RH SSO token                      api-server   closed
NetworkPolicy: runner->api-server            manifests    closed
TenantServiceAccount two-client RBAC         CP           closed
WatchSessionMessages PERMISSION_DENIED       api-server   closed (isServiceCaller via JWT)
clear_runtime_credentials missing           Runner       closed
GET /sessions/{id}/events (proxy)            api-server   open
acpctl session events <id>                   CLI          open
```

**Next:** Test mpp-verify-10 — confirm Claude executes and pushes assistant message end-to-end.

---

### Run 6 — 2026-03-27 (Vertex AI + RunnerContext user fields)

**Status:** End-to-end success. Claude executed and pushed assistant message (`seq=25`, `payload_len=806`) for mpp-verify-16.

**Root causes resolved:**

1. **`RunnerContext` missing `current_user_id`/`set_current_user`** — `bridge.py:_initialize_run()` accessed `self._context.current_user_id` but `RunnerContext` dataclass had no such field. Fixed: added `current_user_id`, `current_user_name`, `caller_token` fields with empty defaults + `set_current_user()` method to `platform/context.py`.

2. **Vertex AI not wired in CP overlay** — `USE_VERTEX`, `ANTHROPIC_VERTEX_PROJECT_ID`, `CLOUD_ML_REGION`, `GOOGLE_APPLICATION_CREDENTIALS`, `VERTEX_SECRET_NAME`, `VERTEX_SECRET_NAMESPACE` were all missing from `ambient-control-plane.yaml`. CP passed `USE_VERTEX=0` to runner pods → runner raised `Either ANTHROPIC_API_KEY or USE_VERTEX=1 must be set`.

3. **`VERTEX_SECRET_NAMESPACE` wrong default** — Default is `ambient-code`; secret is in `ambient-code--runtime-int`. `ensureVertexSecret()` called `r.nsKube().GetSecret(ctx, "ambient-code", "ambient-vertex")` → forbidden. Fixed: `VERTEX_SECRET_NAMESPACE=ambient-code--runtime-int`.

4. **`GOOGLE_APPLICATION_CREDENTIALS` path mismatch** — CP overlay set `/var/run/secrets/vertex/ambient-code-key.json` but `buildVolumeMounts()` mounts vertex secret at `/app/vertex`. Runner pod had the file at `/app/vertex/ambient-code-key.json`. Fixed: `GOOGLE_APPLICATION_CREDENTIALS=/app/vertex/ambient-code-key.json`.

**Changes made:**

| File | Change |
|------|--------|
| `runners/ambient_runner/platform/context.py` | Added `current_user_id`, `current_user_name`, `caller_token` fields + `set_current_user()` method to `RunnerContext` |
| `overlays/mpp-openshift/ambient-control-plane.yaml` | Added `USE_VERTEX=1`, `ANTHROPIC_VERTEX_PROJECT_ID=ambient-code-platform`, `CLOUD_ML_REGION=us-east5`, `GOOGLE_APPLICATION_CREDENTIALS=/app/vertex/ambient-code-key.json`, `VERTEX_SECRET_NAME=ambient-vertex`, `VERTEX_SECRET_NAMESPACE=ambient-code--runtime-int`; added `vertex-credentials` volumeMount + volume from `ambient-vertex` secret |

**Vertex invariants for MPP:**
- Secret: `secret/ambient-vertex` in `ambient-code--runtime-int` — contains `ambient-code-key.json` (GCP SA key)
- CP reads it via `nsKube()` (tenant SA) and copies to session namespace via `ensureVertexSecret()`
- Runner pod mounts it at `/app/vertex/` — set `GOOGLE_APPLICATION_CREDENTIALS=/app/vertex/ambient-code-key.json`
- `VERTEX_SECRET_NAMESPACE` must be `ambient-code--runtime-int` (not default `ambient-code`)
- GCP project: `ambient-code-platform`, region: `us-east5`

**acpctl session commands:**
```bash
acpctl get sessions                        # list all sessions
acpctl session messages <id>               # show messages for session
acpctl session messages <id> -o json       # JSON output
acpctl delete session <id> --yes           # delete without confirmation prompt
acpctl create session --name <n> --prompt "<text>"  # create session
```

**Gap table after Run 6:**

```
ITEM                                         COMPONENT    STATUS
────────────────────────────────────────────────────────────────────────────────
assistant payload -> plain string            Runner       closed
reasoning leaks into DB record               Runner       closed
GET /events/{thread_id}                      Runner       closed
Namespace delete RBAC                        CP manifests closed
MPP namespace naming                         CP           closed
OIDC token provider                          CP           closed
Per-project RBAC in session namespaces       CP           closed (TenantServiceAccount)
AMBIENT_GRPC_ENABLED injection               CP           closed
gRPC auth: RH SSO token                      api-server   closed
NetworkPolicy: runner->api-server            manifests    closed
TenantServiceAccount two-client RBAC         CP           closed
WatchSessionMessages PERMISSION_DENIED       api-server   closed (isServiceCaller via JWT)
clear_runtime_credentials missing           Runner       closed
RunnerContext missing user fields            Runner       closed
Vertex AI not wired in CP overlay            manifests    closed
GET /sessions/{id}/events (proxy)            api-server   open
acpctl session events <id>                   CLI          open
```

**Confirmed working (mpp-verify-16):**
- CP provisioned `ambient-code--test` namespace ✓
- Runner pod started, gRPC watch open ✓
- User message received via gRPC watch ✓
- Claude CLI spawned via Vertex AI ✓
- Assistant message pushed: `seq=25`, `payload_len=806` ✓
- `PushSessionMessage OK` confirmed ✓

---

### Run 4 — 2026-03-27 (TenantServiceAccount two-client RBAC fix)

**Status:** Runner pod created and started. gRPC push of user message succeeds. New blocker: `WatchSessionMessages` returns `PERMISSION_DENIED` — runner loops on reconnect and never executes Claude.

**Context:** This run implemented the `TenantServiceAccount` + second KubeClient pattern to fix the durable RBAC bootstrap problem. The CP now uses `tenantaccess-ambient-control-plane` token (mounted from Secret) for all namespace-scoped ops; the in-cluster SA token is used only for watch/list on the api-server informer.

**Changes made:**

| File | Change |
|------|--------|
| `overlays/mpp-openshift/ambient-cp-tenant-sa.yaml` | New `TenantServiceAccount` CR in `ambient-code--config`; grants `namespace-admin` to all tenant namespaces via operator |
| `overlays/mpp-openshift/ambient-control-plane.yaml` | Added `PROJECT_KUBE_TOKEN_FILE` env var + `project-kube-token` volumeMount + volume from `tenantaccess-ambient-control-plane-token` Secret |
| `kubeclient/kubeclient.go` | Added `NewFromTokenFile(tokenFile, logger)` constructor — uses in-cluster host/CA + explicit bearer token |
| `config/config.go` | Added `ProjectKubeTokenFile string` field (`PROJECT_KUBE_TOKEN_FILE` env) |
| `cmd/ambient-control-plane/main.go` | Builds `projectKube` when token file set; passes to both `NewProjectReconciler` and `NewKubeReconciler` |
| `reconciler/project_reconciler.go` | Added `projectKube` field + `nsKube()` helper; all namespace-scoped ops use `nsKube()` |
| `reconciler/kube_reconciler.go` | Added `projectKube` field + `nsKube()` helper; all namespace-scoped ops use `nsKube()`; removed `ensureSessionNamespaceRBAC` (operator handles RBAC propagation now) |

**Root causes resolved:**

1. **CP SA had zero permissions in project namespaces** — `ensureControlPlaneRBAC` in project reconciler only fires on `ADDED` event; pre-existing projects never get RBAC. And manually applied RBAC is wiped when MPP deprovisions/reprovisions namespace. Fixed: `TenantServiceAccount` operator injects `namespace-admin` RoleBinding into every current+future tenant namespace automatically and durably.
2. **Bootstrap circularity** — `ensureSessionNamespaceRBAC` needed permissions to create Roles, but CP had none. Fixed: operator handles it before CP acts.
3. **Token Secret cross-namespace** — `tenantaccess-ambient-control-plane-token` lives in `ambient-code--config`; copied as `Opaque` Secret to `ambient-code--runtime-int` (must strip `kubernetes.io/service-account.name` annotation to avoid type mismatch).

**MPP re-login invariant:**

OCM tokens expire after ~15 minutes. Before any test run:

```bash
ocm login --use-auth-code   # browser popup — approve on laptop
acpctl login --token $(ocm token) \
  --url https://ambient-api-server-ambient-code--runtime-int.internal-router-shard.mpp-w2-preprod.cfln.p1.openshiftapps.com \
  --insecure-skip-tls-verify
```

**Observed result (mpp-verify-7):**

- CP provisioned `ambient-code--test` namespace ✓
- Runner pod created + started ✓
- gRPC channel to `ambient-api-server.ambient-code--runtime-int.svc:9000` established ✓
- `PushSessionMessage` (user event) succeeded: `seq=9` ✓
- `WatchSessionMessages` returns `PERMISSION_DENIED: not authorized to watch this session` ✗
- Runner loops on watch reconnect; never executes Claude ✗

**New gap:** `WatchSessionMessages` PERMISSION_DENIED — runner's `BOT_TOKEN` (OIDC token for `mturansk`) is rejected by the api-server's watch authorization. The push path uses the same token and succeeds. Root cause: api-server's `WatchSessionMessages` handler likely checks session ownership against the user context and the session's `ProjectID=test` project is owned differently than the token subject.

**Gap table after Run 4:**

```
ITEM                                         COMPONENT    STATUS
──────────────────────────────────────────────────────────────────────────
assistant payload -> plain string            Runner       closed
reasoning leaks into DB record               Runner       closed
GET /events/{thread_id}                      Runner       closed
Namespace delete RBAC                        CP manifests closed
MPP namespace naming                         CP           closed
OIDC token provider                          CP           closed
Per-project RBAC in session namespaces       CP           closed (TenantServiceAccount)
AMBIENT_GRPC_ENABLED injection               CP           closed
gRPC auth: RH SSO token                      api-server   closed
NetworkPolicy: runner->api-server            manifests    closed
TenantServiceAccount two-client RBAC         CP           closed
WatchSessionMessages PERMISSION_DENIED       api-server   open
GET /sessions/{id}/events (proxy)            api-server   open
acpctl session events <id>                   CLI          open
```

**Next:** Investigate `WatchSessionMessages` authorization in api-server — check `@.claude/skills/ambient-api-server/` or `components/ambient-api-server/plugins/sessions/` for watch auth logic. The push succeeds with the same token so the issue is specific to the watch handler's authorization check.

---

### Run 7 — 2026-03-27 (CLI fixes + INITIAL_PROMPT gRPC warning removed)

**Status:** Four improvements to developer workflow. End-to-end path confirmed working from Run 6. These changes close remaining CLI/runner usability gaps.

**Changes made:**

| File | Change |
|------|--------|
| `runners/ambient_runner/app.py` | Skip `_push_initial_prompt_via_grpc()` when `grpc_url` is set — message already in DB; replace call with `logger.debug()` explaining why |
| `ambient-cli/cmd/acpctl/session/messages.go` | `streamMessages()` rewritten from gRPC `WatchSessionMessages` to HTTP long-poll on `ListMessages` every 2s — fixes hang when gRPC port 9000 unreachable from laptop |
| `ambient-cli/cmd/acpctl/session/send.go` | Added `--follow`/`-f` flag; after `PushMessage` succeeds, sets `msgArgs.afterSeq = msg.Seq` and calls `streamMessages()` |
| `ambient-cli/cmd/acpctl/session/events.go` | Added missing `X-Ambient-Project` header to SSE request |

**Root causes fixed:**

1. **`INITIAL_PROMPT gRPC push PERMISSION_DENIED` warning** — When `AMBIENT_GRPC_ENABLED=true`, the initial prompt is already stored in the DB by the `acpctl create session` HTTP call. The runner's `WatchSessionMessages` listener delivers it automatically. The redundant gRPC push used the SA token which cannot push `event_type=user` → harmless but noisy `PERMISSION_DENIED` logged at WARN level every run. Fixed: skip the push branch entirely when `grpc_url` is set.

2. **`acpctl session messages -f` hang** — `streamMessages()` called `WatchSessionMessages` via gRPC. The SDK's `deriveGRPCAddress()` maps `internal-router-shard` URLs to port 9000, which is only reachable in-cluster. From a laptop, the connection hangs indefinitely. Fixed: replaced with HTTP long-poll using `ListMessages(ctx, sessionID, afterSeq)` every 2 seconds, tracking the highest `Seq` seen.

3. **`acpctl session send -f` missing** — No `--follow` flag existed on `sendCmd`. Fixed: added `--follow`/`-f` flag; after successful push, calls `streamMessages()` starting from the pushed message's seq.

4. **`acpctl session events` missing project header** — The SSE request to the api-server was missing `X-Ambient-Project`, which is required by the auth middleware. Fixed: added `req.Header.Set("X-Ambient-Project", cfg.GetProject())`.

**api-server `GET /sessions/{id}/events` status:**

This endpoint is **already implemented** in `plugins/sessions/handler.go:282` (`StreamRunnerEvents`). It proxies to `http://session-{KubeCrName}.{KubeNamespace}.svc.cluster.local:8001/events/{KubeCrName}`. Registered in `plugin.go:102` (note: duplicate at line 104 is harmless dead code). The `acpctl session events <id>` command is fully wired and should work against a running session.

**acpctl session commands (updated):**

```bash
acpctl get sessions                                # list all sessions
acpctl session messages <id>                       # snapshot
acpctl session messages <id> -f                    # live poll every 2s (Ctrl+C to stop)
acpctl session messages <id> -o json               # JSON snapshot
acpctl session messages <id> --after 5             # messages after seq 5
acpctl session send <id> "message"                 # send and return
acpctl session send <id> "message" -f              # send and follow conversation
acpctl session events <id>                         # stream raw AG-UI SSE events from runner (session must be actively running)
acpctl delete session <id> --yes                   # delete without confirmation
acpctl create session --name <n> --prompt "<text>" # create session
```

**Gap table after Run 7:**

```
ITEM                                         COMPONENT    STATUS
────────────────────────────────────────────────────────────────────────────────
assistant payload -> plain string            Runner       closed
reasoning leaks into DB record               Runner       closed
GET /events/{thread_id}                      Runner       closed
Namespace delete RBAC                        CP manifests closed
MPP namespace naming                         CP           closed
OIDC token provider                          CP           closed
Per-project RBAC in session namespaces       CP           closed (TenantServiceAccount)
AMBIENT_GRPC_ENABLED injection               CP           closed
gRPC auth: RH SSO token                      api-server   closed
NetworkPolicy: runner->api-server            manifests    closed
TenantServiceAccount two-client RBAC         CP           closed
WatchSessionMessages PERMISSION_DENIED       api-server   closed (isServiceCaller via JWT)
clear_runtime_credentials missing            Runner       closed
RunnerContext missing user fields            Runner       closed
Vertex AI not wired in CP overlay            manifests    closed
INITIAL_PROMPT gRPC push warning             Runner       closed
acpctl messages -f hang (gRPC)               CLI          closed (HTTP long-poll)
acpctl send -f missing                       CLI          closed
acpctl events missing project header         CLI          closed
GET /sessions/{id}/events (proxy)            api-server   closed (already implemented)
```

**All known gaps closed.** End-to-end path: session creation → CP provisioning → runner pod → gRPC watch → Claude via Vertex AI → assistant message pushed to DB → `acpctl session messages <id> -f` displays it.

---

### Run 8 — 2026-03-27 (Plain-text assistant payload + RBAC race documented)

**Status:** `_write_message()` fixed to push plain assistant text. New gap discovered: TenantSA RBAC race on namespace re-create.

**Changes made:**

| File | Change |
|------|--------|
| `runners/ambient_runner/bridges/claude/grpc_transport.py` | `_write_message()` now extracts plain assistant text from `_accumulated_messages` instead of pushing the full JSON blob; `import json` removed |

**Root cause of JSON blob:** The Wave 1 fix described in Run 2 was applied to the kind cluster codebase but never to this (MPP) codebase. `_write_message()` was still calling `json.dumps({"run_id": ..., "status": ..., "messages": [...]})` and pushing the full blob as `payload`. `displayPayload()` in the CLI uses `extractAGUIText()` which expects `{"messages": [{"role": "assistant", "content": "..."}]}` — the wrong shape. Result: assistant messages existed in DB but showed blank in `acpctl session messages`.

**New gap discovered — TenantSA RBAC race:**

When a session is deleted and a new session immediately created in the same project, MPP re-creates `ambient-code--test`. The tenant-access-operator injects `namespace-admin` RoleBinding asynchronously (~20-60s). If CP reconciles before propagation completes, `ensureSecret()` at `kube_reconciler.go:328` fails with `secrets is forbidden`. The informer does not retry — session sticks in `""` phase with no pod.

**Workaround:** Wait 30-60 seconds between deleting a session and creating a new one in the same project. See **Known Races** section above.

**Gap table after Run 8:**

```
ITEM                                         COMPONENT    STATUS
────────────────────────────────────────────────────────────────────────────────
assistant payload -> plain string            Runner       closed (Run 8)
TenantSA RBAC race on namespace re-create   CP           open (workaround: wait 60s)
```
