# Operator Development Context

**When to load:** Working on `components/ambient-control-plane/` — the Kubernetes operator that provisions session pods, secrets, service accounts, and services

## Quick Reference

- **Language:** Go 1.21+
- **Role:** Watches session events from the api-server informer; provisions and deprovisions all K8s resources for each session
- **Primary file:** `components/ambient-control-plane/internal/reconciler/kube_reconciler.go`
- **K8s client:** `internal/kubeclient/` — dynamic unstructured client, not typed client-go
- **Namespace provisioner:** `internal/kubeclient/` — `NamespaceProvisioner` interface
- **Config:** `KubeReconcilerConfig` struct in `kube_reconciler.go`

---

## Reconciler Lifecycle

The operator listens for session events from the api-server informer. Each event triggers `Reconcile()`:

```
EventAdded   + phase=pending  → provisionSession()
EventModified + phase=pending  → provisionSession()
EventModified + phase=stopping → deprovisionSession()
EventDeleted                   → cleanupSession()
```

`provisionSession()` runs these steps in order — all are idempotent:

1. Validate project exists via SDK
2. `ensureNamespaceExists()` — provision project namespace with labels
3. `ensureSecret()` — API token secret for runner
4. `ensureVertexSecret()` — optional; copy Vertex credentials if Vertex enabled
5. `ensureServiceAccount()` — runner service account
6. `ensurePod()` — runner pod + optional MCP sidecar
7. `ensureService()` — ClusterIP service for runner HTTP/SSE
8. `updateSessionPhaseWithNamespace()` — mark session Running

---

## Namespace Naming

Session namespace is derived from `session.ProjectID`:

```go
func (r *SimpleKubeReconciler) namespaceForSession(session types.Session) string {
    if session.ProjectID != "" {
        return r.provisioner.NamespaceName(session.ProjectID)
    }
    if session.KubeNamespace != "" {
        return session.KubeNamespace
    }
    return "default"
}
```

If `ProjectID` is empty, the session lands in `default` namespace — which is wrong for multi-tenant operation. Always ensure `project_id` is set before provisioning.

---

## Resource Naming

All K8s resource names are derived from session ID, truncated to 40 characters:

```go
func safeResourceName(sessionID string) string {
    return strings.ToLower(sessionID[:min(len(sessionID), 40)])
}

func podName(sessionID string) string         { return fmt.Sprintf("session-%s-runner", safeResourceName(sessionID)) }
func secretName(sessionID string) string      { return fmt.Sprintf("session-%s-creds", safeResourceName(sessionID)) }
func serviceAccountName(sessionID string) string { return fmt.Sprintf("session-%s-sa", safeResourceName(sessionID)) }
func serviceName(sessionID string) string     { return fmt.Sprintf("session-%s", safeResourceName(sessionID)) }
```

All resources are labeled with `sessionLabels(sessionID, projectID)` for grouped cleanup.

---

## `ensureSecret()` — API Token

Creates a K8s Secret with the API token for the runner to authenticate against the api-server:

```go
secret := &unstructured.Unstructured{
    Object: map[string]interface{}{
        "apiVersion": "v1",
        "kind":       "Secret",
        "metadata": map[string]interface{}{
            "name":      name,
            "namespace": namespace,
            "labels":    sessionLabels(session.ID, session.ProjectID),
        },
        "stringData": map[string]interface{}{
            "api-token": token,
        },
    },
}
```

Token is resolved via `r.factory.Token(ctx)` — the SDK factory provides a fresh token from the configured source. If this fails with a forbidden error on a new namespace, it is usually an RBAC propagation race — the namespace was just created and the service account permissions haven't propagated yet. Retry with backoff if `k8serrors.IsForbidden(err)`.

---

## `ensurePod()` — Runner Pod

The runner pod spec includes:

- **Container:** `ambient-code-runner` image (`r.cfg.RunnerImage`)
- **Image pull policy:** `Always` for registry images; `IfNotPresent` for `localhost/` prefix
- **Port:** `agui` on 8001 (SSE endpoint for AG-UI events)
- **Volumes:** `workspace` (emptyDir) + `service-ca` (OpenShift service CA cert, optional)
- **Service account:** `session-{id}-sa` (automount disabled)
- **Restart policy:** `Never` — runner is a one-shot job
- **Security context:** `allowPrivilegeEscalation: false`, drop ALL capabilities
- **Optional sidecar:** MCP server if `r.cfg.MCPImage != ""`

### MCP Sidecar

When `MCPImage` is set, `buildMCPSidecar()` appends a second container:
- Port `8090` (SSE transport)
- Env: `MCP_TRANSPORT=sse`, `MCP_BIND_ADDR=:8090`, `AMBIENT_API_URL`, `AMBIENT_TOKEN` from secret
- Runner receives `AMBIENT_MCP_URL=http://localhost:8090`

---

## `buildEnv()` — Runner Environment Variables

The operator assembles all env vars for the runner pod. Key vars:

| Var | Source | Purpose |
|---|---|---|
| `SESSION_ID` | `session.ID` | Runner's identity |
| `AGENT_ID` | `session.AgentID` | Which agent to drain inbox for |
| `PROJECT_NAME` | `session.ProjectID` | Multi-tenant scope |
| `BACKEND_API_URL` | `r.cfg.BackendURL` | Runner calls back to api-server |
| `BOT_TOKEN` | K8s secret `api-token` key | Auth for api-server calls |
| `AMBIENT_GRPC_URL` | `r.cfg.RunnerGRPCURL` | CP gRPC address for event push |
| `AMBIENT_GRPC_ENABLED` | `RunnerGRPCURL != ""` | Enable gRPC transport in runner |
| `AMBIENT_GRPC_USE_TLS` | `r.cfg.RunnerGRPCUseTLS` | TLS for gRPC |
| `AMBIENT_GRPC_CA_CERT_FILE` | `/etc/pki/ca-trust/...` | OpenShift service CA |
| `INITIAL_PROMPT` | `assembleInitialPrompt()` | Assembled prompt from project + agent + session |
| `LLM_MODEL` | `session.LlmModel` | Override default model |
| `REPOS_JSON` | `session.RepoURL` | Git repo to clone at start |

### `assembleInitialPrompt()`

Concatenates (in order):
1. `project.Prompt` — project-level system prompt
2. `agent.Prompt` — agent-level system prompt
3. All unread `InboxMessage.Body` for this agent (up to 100)
4. `session.Prompt` — session-specific prompt

Joined with `\n\n`. If any fetch fails, it logs a warning and continues with the parts that succeeded.

---

## Image Push Playbook (kind cluster)

After any code change, the new image must reach the kind cluster's containerd. `kind load docker-image` fails with podman because it calls `docker inspect` internally and cannot resolve `localhost/` prefix images. Use `ctr import` instead:

```bash
# 0. Find running cluster
CLUSTER=$(podman ps --format '{{.Names}}' | grep 'kind' | grep 'control-plane' | sed 's/-control-plane//')

# 1. Build without cache (cache misses source changes when go.mod/go.sum unchanged)
podman build --no-cache -t localhost/vteam_control_plane:latest components/ambient-control-plane

# 2. Load into kind via ctr import
podman save localhost/vteam_control_plane:latest | \
  podman exec -i ${CLUSTER}-control-plane ctr --namespace=k8s.io images import -

# 3. Restart and verify
kubectl rollout restart deployment/ambient-control-plane -n ambient-code
kubectl rollout status deployment/ambient-control-plane -n ambient-code --timeout=60s
```

**Why `--no-cache`:** Dockerfile copies source in layers. If `go.mod`/`go.sum` are unchanged, the `go build` step hits cache and emits the old binary.

---

## `ensureImagePullAccess()` — OpenShift Image Pull

On OpenShift clusters, runner pods in new namespaces cannot pull from the platform image registry without a `RoleBinding` granting `system:image-puller`:

```go
rb := &unstructured.Unstructured{
    Object: map[string]interface{}{
        "apiVersion": "rbac.authorization.k8s.io/v1",
        "kind":       "RoleBinding",
        "metadata": map[string]interface{}{
            "name":      "ambient-image-puller",
            "namespace": r.cfg.RunnerImageNamespace,  // image registry namespace
        },
        "roleRef": map[string]interface{}{
            "apiGroup": "rbac.authorization.k8s.io",
            "kind":     "ClusterRole",
            "name":     "system:image-puller",
        },
        "subjects": []interface{}{
            map[string]interface{}{
                "apiGroup": "rbac.authorization.k8s.io",
                "kind":     "Group",
                "name":     fmt.Sprintf("system:serviceaccounts:%s", namespace),
            },
        },
    },
}
```

This is guarded by `r.cfg.RunnerImageNamespace != ""` — only runs on OpenShift where images live in a separate namespace. In kind clusters (localhost images), this is skipped.

---

## `cleanupSession()` — Resource Teardown

On session deletion, cleanup runs in this order:

1. Delete pods by label selector
2. Delete secrets by label selector
3. Delete service accounts by label selector
4. Delete services by label selector
5. `DeprovisionNamespace()` — remove the namespace if it was managed

All cleanup operations ignore `IsNotFound` errors — cleanup is idempotent.

---

## Build Commands

```bash
cd components/ambient-control-plane

go build ./...
go test ./...
go fmt ./...
golangci-lint run
```

Top-level: `make build-operator` builds the container image (despite the Makefile target name, it builds the control-plane image).

---

## Pre-Commit Checklist

- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] `golangci-lint run` passes
- [ ] No `panic()` — use `fmt.Errorf` with context
- [ ] All new K8s resources have `sessionLabels()` applied for grouped cleanup
- [ ] `SecurityContext` set on all containers: `allowPrivilegeEscalation: false`, drop ALL
- [ ] `ensureImagePullAccess()` guarded by `RunnerImageNamespace != ""`
- [ ] `assembleInitialPrompt()` updated if new prompt sources added to Spec
- [ ] New env vars added to `buildEnv()` when Spec changes runner behavior
- [ ] Image push playbook run and rollout verified in kind before marking wave complete
