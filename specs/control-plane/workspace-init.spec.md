# Workspace Initialization Specification

## Purpose

When the Control Plane provisions a runner pod, the workspace (`/workspace`) must be prepared before the runner starts. This includes creating directory structure, cloning repositories, restoring prior workspace state from object storage, and cloning workflow repositories. The CP achieves this by adding an **init container** to the pod spec that runs a hydration script. The operator already implements this pattern; this spec defines the same behavior for the CP path.

## Requirements

### Requirement: Init Container Presence

The CP SHALL add an init container named `init-hydrate` to the runner pod when either:

- The session specifies repositories (`RepoURL` or `Repos` is non-empty), OR
- S3 state persistence is configured for the session's project

When neither condition is met, no init container SHALL be added and `/workspace` SHALL start as an empty `emptyDir` volume.

#### Scenario: Session with a single repo URL

- GIVEN a session where `RepoURL` is `https://github.com/org/repo`
- WHEN the CP provisions the runner pod
- THEN the pod spec SHALL include an `init-hydrate` init container
- AND the init container SHALL receive `REPOS_JSON` set to `[{"url":"https://github.com/org/repo"}]`

#### Scenario: Session with multiple repos

- GIVEN a session where `Repos` is `[{"url":"https://github.com/org/a","branch":"main"},{"url":"https://github.com/org/b"}]`
- WHEN the CP provisions the runner pod
- THEN the init container SHALL receive `REPOS_JSON` set to the value of `Repos` verbatim

#### Scenario: Both RepoURL and Repos are set

- GIVEN a session where both `RepoURL` and `Repos` are non-empty
- WHEN the CP provisions the runner pod
- THEN `Repos` SHALL take precedence
- AND `RepoURL` SHALL be ignored

#### Scenario: No repos and no S3 configured

- GIVEN a session with no `RepoURL`, no `Repos`, and no S3 persistence configured
- WHEN the CP provisions the runner pod
- THEN no init container SHALL be added

#### Scenario: No repos but S3 is configured

- GIVEN a session with no repos but S3 persistence is configured
- WHEN the CP provisions the runner pod
- THEN an `init-hydrate` init container SHALL be added
- AND the init container SHALL restore prior workspace state from S3 (if any exists)

### Requirement: Init Container Image

The init container SHALL use the same `state-sync` image used by the operator (`quay.io/ambient_code/vteam_state_sync`). The image reference SHOULD be configurable via a `STATE_SYNC_IMAGE` environment variable on the CP deployment, which MUST be added to the CP's `KubeReconcilerConfig`.

#### Scenario: Image configuration

- GIVEN the CP deployment has `STATE_SYNC_IMAGE=quay.io/ambient_code/vteam_state_sync:v2`
- WHEN the CP provisions a pod with an init container
- THEN the init container image SHALL be `quay.io/ambient_code/vteam_state_sync:v2`

### Requirement: Init Container Authentication

The init container SHALL authenticate to the api-server using the CP token endpoint (`GET /token`), not a pre-injected `BOT_TOKEN`. The init container calls the CP's token endpoint using the pod's mounted Kubernetes service account token (at `/var/run/secrets/kubernetes.io/serviceaccount/token`), the same mechanism the runner container uses.

The CP SHALL inject `AMBIENT_CP_TOKEN_URL` into the init container environment. The `hydrate.sh` script SHALL check `AMBIENT_CP_TOKEN_URL` first and obtain a bearer token from the CP before calling the backend API for git credentials. When `AMBIENT_CP_TOKEN_URL` is not set (operator path), `hydrate.sh` SHALL fall back to `BOT_TOKEN` for backward compatibility.

**Implementation note:** `hydrate.sh` does not yet support `AMBIENT_CP_TOKEN_URL` — it currently uses `BOT_TOKEN` only. The script must be updated to implement this requirement (see Status section).

#### Scenario: Init container obtains token from CP endpoint

- GIVEN a CP-provisioned pod with `AMBIENT_CP_TOKEN_URL` set
- WHEN `hydrate.sh` needs to fetch git credentials from the backend API
- THEN it SHALL call `GET <AMBIENT_CP_TOKEN_URL>` with the SA token in the `Authorization` header
- AND it SHALL use the returned bearer token for subsequent backend API calls

#### Scenario: Operator-provisioned pod with BOT_TOKEN

- GIVEN an operator-provisioned pod with `BOT_TOKEN` set and no `AMBIENT_CP_TOKEN_URL`
- WHEN `hydrate.sh` needs to fetch git credentials
- THEN it SHALL use `BOT_TOKEN` directly (backward-compatible path)

### Requirement: Init Container Environment

The init container SHALL receive the following environment variables:

| Variable | Source | Required |
|---|---|---|
| `SESSION_NAME` | session ID | always |
| `NAMESPACE` | project ID | always |
| `PROJECT_NAME` | project ID | always |
| `BACKEND_API_URL` | CP config | always |
| `AMBIENT_CP_TOKEN_URL` | CP config | always (CP path) |
| `REPOS_JSON` | session model | when repos specified |
| `ACTIVE_WORKFLOW_GIT_URL` | Workflow resource (resolved from `session.WorkflowID`) | when workflow specified |
| `ACTIVE_WORKFLOW_BRANCH` | Workflow resource | when workflow specified |
| `ACTIVE_WORKFLOW_PATH` | Workflow resource | when workflow specified |
| `S3_ENDPOINT` | project or cluster config | when S3 configured |
| `S3_BUCKET` | project or cluster config | when S3 configured |
| `AWS_ACCESS_KEY_ID` | project or cluster secret | when S3 configured |
| `AWS_SECRET_ACCESS_KEY` | project or cluster secret | when S3 configured |
| `RUNNER_STATE_DIR` | CP config | always (default: `.claude`) |

**Note on workflow fields:** The session model has `WorkflowID` (a foreign key), not the git URL/branch/path directly. The CP MUST resolve `session.WorkflowID` by fetching the Workflow resource from the api-server and extracting its `gitUrl`, `branch`, and `path` fields.

**Note on S3 access credentials:** `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` are S3 object storage credentials, not Ambient Credential Kind entities. They are resolved from project-level or cluster-level Kubernetes Secrets, not from the `CREDENTIAL_IDS` credential flow.

### Requirement: Input Sanitization

The init container SHALL sanitize `SESSION_NAME` and `NAMESPACE` values to prevent path traversal in S3 paths and local filesystem operations. Only alphanumeric characters and hyphens SHALL be permitted.

### Requirement: Volume Sharing

The init container and runner container SHALL share a single `emptyDir` volume mounted at `/workspace`. The init container writes to this volume; the runner reads from it after the init container exits.

#### Scenario: Workspace volume lifecycle

- GIVEN an init container and runner container in the same pod
- WHEN the init container clones a repo into `/workspace/repos/myrepo`
- THEN the runner container SHALL see `/workspace/repos/myrepo` with the cloned content

### Requirement: Directory Structure

The init container SHALL create the following directories before any clone or restore operation:

- `/workspace/repos/`
- `/workspace/artifacts/`
- `/workspace/file-uploads/`
- `/workspace/<RUNNER_STATE_DIR>/` (default: `/workspace/.claude/`)
- `/workspace/<RUNNER_STATE_DIR>/debug/` (when `RUNNER_STATE_DIR` is `.claude`)

Ownership SHALL be set to UID 1001 (runner user). Permissions on `/workspace/repos/` and `/workspace/file-uploads/` SHALL be world-writable (0777) because the init container and runner container may run as different UIDs. `/workspace/artifacts/` SHALL be 0755.

### Requirement: Repo URL Normalization

The CP SHALL normalize the session's repo specification into the `REPOS_JSON` format consumed by `hydrate.sh`:

- `session.RepoURL` (single string) SHALL be converted to `[{"url":"<value>"}]`
- `session.Repos` (JSON array string) SHALL be passed through as-is
- If both are set, `Repos` SHALL take precedence

**Note:** The CP currently only handles `session.RepoURL`. Support for `session.Repos` (multi-repo array) MUST be added — it is a field on the session model (`types.Session.Repos`) and the api-server data model but is not read by the CP today.

#### Scenario: RepoURL normalization

- GIVEN a session with `RepoURL = "https://github.com/org/repo"`
- WHEN the CP builds the init container env
- THEN `REPOS_JSON` SHALL equal `[{"url":"https://github.com/org/repo"}]`

### Requirement: Git Credential Fetch for Private Repos

The init container SHALL fetch git credentials at runtime by calling the backend API, not by receiving pre-injected `GITHUB_TOKEN` or `GITLAB_TOKEN` environment variables. The `hydrate.sh` script uses `BACKEND_API_URL` and its bearer token to call the credentials endpoint for each provider:

```
GET <BACKEND_API_URL>/projects/<PROJECT_NAME>/agentic-sessions/<SESSION_NAME>/credentials/github
GET <BACKEND_API_URL>/projects/<PROJECT_NAME>/agentic-sessions/<SESSION_NAME>/credentials/gitlab
```

If a provider credential is available, the script sets the corresponding environment variable (`GITHUB_TOKEN` or `GITLAB_TOKEN`) internally and configures a git credential helper that returns these tokens for matching hosts.

If no credentials are available or the fetch fails, `hydrate.sh` SHALL log a distinct warning (distinguishing auth failure from clone failure) and continue — public repos will clone successfully, private repos will fail with a non-fatal warning.

#### Scenario: Private GitHub repo with credentials

- GIVEN a session with a `github` credential resolved for this project
- AND `RepoURL` is `https://github.com/org/private-repo`
- WHEN the init container runs
- THEN `hydrate.sh` SHALL fetch `GITHUB_TOKEN` from the backend API at runtime
- AND the clone SHALL succeed using the git credential helper

#### Scenario: Private repo without credentials

- GIVEN a session with no git credentials configured
- AND `RepoURL` is `https://github.com/org/private-repo`
- WHEN the init container runs
- THEN the credential fetch SHALL return empty
- AND the clone SHALL fail with a warning
- AND the init container SHALL exit with code 0 (clone failures are non-fatal; a non-zero exit would prevent the pod from starting)

### Requirement: S3 Workspace State Hydration

When S3 access credentials are provided, the init container SHALL check for existing workspace state in S3 at the path `s3://<bucket>/<namespace>/<session-name>/` and restore it in two phases:

**Phase 1 — Before repo cloning:**
- Framework state (`<RUNNER_STATE_DIR>/`) to `/workspace/<RUNNER_STATE_DIR>/`
- Artifacts to `/workspace/artifacts/`
- File uploads to `/workspace/file-uploads/`

**Phase 2 — After repo cloning:**
- Git repo state from `repo-state/<repo-name>/` — restore bundles, checkout saved branch, apply uncommitted and staged patches

This two-phase ordering is required because git state patches can only be applied to cloned repositories.

#### Scenario: Resuming a session with S3 state

- GIVEN a session that previously ran and synced workspace state to S3
- AND S3 access credentials are configured
- WHEN the init container runs for a new pod
- THEN framework state, artifacts, and file uploads SHALL be restored first (Phase 1)
- AND repos SHALL be cloned from their remote URLs
- AND git branch, uncommitted changes, and staged changes SHALL be re-applied from S3 backup (Phase 2)

#### Scenario: First run with no S3 state

- GIVEN a session running for the first time
- AND S3 access credentials are configured
- WHEN the init container checks S3
- THEN no state SHALL be found
- AND the init container SHALL proceed to clone repos and create directories normally

### Requirement: Workflow Cloning

When `ACTIVE_WORKFLOW_GIT_URL` is set, the init container SHALL clone the workflow repository into `/workspace/workflows/<repo-name>`. If `ACTIVE_WORKFLOW_PATH` is set, only the specified subdirectory SHALL be extracted to the target path.

#### Scenario: Workflow with subpath

- GIVEN `ACTIVE_WORKFLOW_GIT_URL = "https://github.com/org/my-workflows"` and `ACTIVE_WORKFLOW_PATH = "session-setup"`
- WHEN the init container runs
- THEN only the `session-setup/` subdirectory SHALL appear at `/workspace/workflows/my-workflows/`

#### Scenario: Workflow subpath not found

- GIVEN a workflow with `ACTIVE_WORKFLOW_PATH` pointing to a non-existent subdirectory
- WHEN the init container runs
- THEN the init container SHALL log a warning
- AND SHALL fall back to using the entire cloned repository

### Requirement: Security Context

The init container SHALL run with a restricted security context:

- `allowPrivilegeEscalation: false`
- `capabilities: drop: ["ALL"]`
- `readOnlyRootFilesystem: false` (required because `hydrate.sh` writes rclone config to `/tmp` and creates workspace directories)

The init container runs as root (not `runAsNonRoot`) because it must set ownership (`chown`) of workspace directories to UID 1001 before the runner container starts. This is an intentional exception to the platform's `runAsNonRoot` convention.

## Status

**Not implemented.** The CP currently sets `REPOS_JSON` as an env var on the runner container (for `RepoURL` only — `Repos` is not handled) but does not create an init container. Repos appear as empty directories at `/workspace/repos/<name>/`. The operator implements this correctly in `reconcileSpecReposWithPatch` (`components/operator/internal/handlers/sessions.go:944-1015`). The CP implementation should add the same `init-hydrate` container in `ensurePod` (`components/ambient-control-plane/internal/reconciler/kube_reconciler.go:394-484`).

Additionally, `hydrate.sh` (`components/runners/state-sync/hydrate.sh`) must be updated to support the CP token endpoint (`AMBIENT_CP_TOKEN_URL`) as the primary authentication mechanism, falling back to `BOT_TOKEN` for operator compatibility.
