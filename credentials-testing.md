# Credentials Testing Log

Testing the end-to-end credential flow via `demo-github.sh` against a local kind cluster.

## Goal

Verify that:
1. `acpctl apply -f` creates a GitHub credential (token injected via env var, not argv)
2. `credential:token-reader` and `credential:reader` roles are seeded by migration
3. Role binding to an agent works
4. Runner pod can call `GET /credentials/{id}/token` and authenticate with GitHub
5. Agent successfully opens a GitHub issue in https://github.com/ambient-code/platform

## Test Repo / Issue Target

- **Repo**: `ambient-code/platform`
- **PR for context**: https://github.com/ambient-code/platform/pull/1032
- **GitHub token**: `~/projects/secrets/github.ambient-pat.token`

## Environment

### OSD cluster (active)

- **Cluster**: OSD `api.dev-osd-east-1.mxty.p1.openshiftapps.com:6443`
- **Namespace**: `ambient-code--ambient-s0`
- **API server URL**: `https://ambient-api-server-ambient-code--ambient-s0.apps.int.spoke.dev.us-east-1.aws.paas.redhat.com`
- **Auth**: RH SSO (OIDC, `redhat.com`/`ambient.code` ACL)
- **Login**: `acpctl login --use-auth-code --url https://ambient-api-server-ambient-code--ambient-s0.apps.int.spoke.dev.us-east-1.aws.paas.redhat.com`
- **acpctl version**: `v0.1.2-59-g85c72153-dirty`
- **Active user**: `mturansk@redhat.com` (project: `demo-github-52477`)

### Local kind cluster (legacy reference)

- **Cluster**: kind (`ambient-feat-ambient-runners`) — already running via `podman ps`
- **API server**: local build pushed to `quay.io/ambient_code/vteam_api_server:<sha>`
- **Control plane**: local build pushed to `quay.io/ambient_code/vteam_control_plane:<sha>`
- **Image tag strategy**: git commit SHA of branch tip

## Branch / Image Tags

| Date | SHA | Component | Notes |
|------|-----|-----------|-------|
| TBD | TBD | api-server | credential roles migration (202603311216) |
| TBD | TBD | control-plane | file-mount token refresh, informer retry fixes |

## Migration Ordering

| Migration ID | Plugin | What it does |
|---|---|---|
| `202603100137` | roles | Creates `roles` table, seeds 8 built-in platform/project/agent roles |
| `202603311215` | credentials | Creates `credentials` table |
| `202603311216` | credentials | Seeds `credential:token-reader` and `credential:reader` roles |

**Cross-plugin timing note**: `202603311216` inserts rows into the `roles` table from the credentials plugin. This is intentional — the migration ID guarantees it runs after roles table exists. Trade-off: coupling between plugins in migration layer.

## Key Code Locations

| File | What |
|------|------|
| `plugins/credentials/migration.go` | `rolesMigration()` — seeds credential roles |
| `plugins/credentials/plugin.go` | Registers both migrations |
| `components/ambient-control-plane/internal/reconciler/kube_reconciler.go` | `buildVolumes()` mounts secret as file; `StartTokenRefreshLoop()` refreshes every 10min |
| `components/ambient-cli/demo-github.sh` | End-to-end test script |
| `components/ambient-api-server/plugins/sessions/grpc_handler.go` | `WatchSessionMessages` — allows GRPC_SERVICE_ACCOUNT to bypass ownership check |

## RBAC Design (from spec)

```
credential:token-reader  — permission: credential:token
  → GET /credentials/{id}/token  returns raw token value

credential:reader        — permission: credential:read, credential:list
  → GET /credentials/{id}        metadata only (no token)
  → GET /credentials             list
```

Role binding scope: `agent` — the runner pod (acting as the agent's service account) can call the token endpoint for credentials bound to that agent.

## Session Prompt Design

The demo sends a prompt that instructs the runner to:
1. Call `GET /api/ambient/v1/credentials/{id}/token`
2. Set `GITHUB_TOKEN=<token>`
3. Use `gh` CLI or `curl` to open an issue in `ambient-code/platform`

The `CREDENTIAL_IDS` env var is injected by the control-plane into the runner pod with a JSON map of `provider → credential_id`.

## Test Plan (OSD ambient-s0)

Steps to execute in order:

1. **Create project** `credential-test` → namespace `ambient-code--credential-test` auto-created by operator
2. **Create agent** `github-agent` in `credential-test`
3. **Apply credential** from `components/ambient-cli/credential.yaml` with `GITHUB_TOKEN` from `~/projects/secrets/github.ambient-pat.token`
4. **Verify credential roles** — check that `credential:token-reader` and `credential:reader` appear in `acpctl get roles` (migration `202603311216` gate)
5. **Create role binding** — bind `credential:token-reader` to `github-agent` so runner pod can call `GET /credentials/{id}/token`
6. **Watch pods** — `oc logs -f -n ambient-code--ambient-s0 deploy/ambient-control-plane` and `oc get pods -n ambient-code--credential-test -w`
7. **Start agent session** — `acpctl agent start github-agent --project-id credential-test --prompt "..."` to trigger the full flow

Namespace convention: every project `$name` created → OSD namespace `ambient-code--$name`

## Run Log

---

### Run 1 — 2026-04-03

**SHA**: `85c72153` (branch: `fix/cp-credential-rolebinding-and-project-delete`)
**Cluster**: OSD `ambient-s0`
**Result**: IN PROGRESS

#### Resources Created

| Resource | Name | ID |
|----------|------|----|
| Project | `credential-test` | `credential-test` |
| Namespace | `ambient-code--credential-test` | (auto-created by operator ✓) |
| Agent | `github-agent` | `3BrhNiF2lBOil50Nn7Ddj9aUUhF` |
| Credential | `my-github-pat` (github) | `3BrhPrCrPhfcZ3cQ6yOmsCEuJXB` |
| Role binding | `credential:token-reader` → agent | **BLOCKED** |

#### Findings

- `credential:token-reader` and `credential:reader` roles **absent** — only 8 base roles present
- Migration `202603311216` has not run on this deployment (not yet deployed to OSD)
- Role binding **cannot be created** until migration runs
- Operator correctly created namespace `ambient-code--credential-test` on project creation ✓
- Credential created successfully via `acpctl apply -f` with env-var token injection ✓

#### Next Steps

- Deploy api-server image containing migration `202603311216` to OSD `ambient-s0`
- Re-run `acpctl get roles` to confirm `credential:token-reader` and `credential:reader` appear
- Create role binding: `credential:token-reader` → agent `3BrhNiF2lBOil50Nn7Ddj9aUUhF`
- Start agent session and watch `oc get pods -n ambient-code--credential-test -w` + control-plane logs

---

## Known Issues / Findings

- `credential:token-reader` was absent from DB — root cause: migration `202603311216` was not in any deployed image yet
- Control-plane pods were in `ErrImagePull` in int environment during prior testing — our changes not deployed
- `GRPC_SERVICE_ACCOUNT` env var must be set on api-server so runner's OIDC JWT (`preferred_username`) bypasses WatchSessionMessages ownership check
- BOT_TOKEN expiry: OIDC JWTs expire in minutes; fix is file mount + 10-min background refresh loop in control-plane

## Commands Reference

```bash
# Login to OSD cluster (RH SSO — opens browser)
acpctl login --use-auth-code --url https://ambient-api-server-ambient-code--ambient-s0.apps.int.spoke.dev.us-east-1.aws.paas.redhat.com

# Login to local kind cluster (token-based)
TOKEN=$(kubectl get secret test-user-token -n ambient-code -o jsonpath='{.data.token}' | base64 -d)
acpctl login http://localhost:8081 --token "$TOKEN" --insecure-skip-tls-verify

# Run demo
GITHUB_TOKEN=$(cat ~/projects/secrets/github.ambient-pat.token) \
  GITHUB_REPO=ambient-code/platform \
  NO_CLEANUP=1 \
  ./demo-github.sh

# Build and push api-server with SHA tag
SHA=$(git rev-parse HEAD)
podman build -t quay.io/ambient_code/vteam_api_server:${SHA} components/ambient-api-server
podman push quay.io/ambient_code/vteam_api_server:${SHA}

# Reload into kind
make local-reload-api-server
# or manually:
podman save localhost/vteam_api_server:latest | \
  podman exec -i ambient-feat-ambient-runners-control-plane \
  ctr --namespace=k8s.io images import -
kubectl rollout restart deployment/ambient-api-server -n ambient-code
```
