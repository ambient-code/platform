# State Persistence Specification

## Purpose

Sessions produce workspace state (framework data, artifacts, file uploads, git repo changes) that must survive pod restarts and session resumes. The CP achieves this by adding a **state-sync sidecar** container to the runner pod that periodically uploads workspace state to S3-compatible object storage. On the next pod start, the init container (see `workspace-init.spec.md`) restores this state. The operator already implements this pattern; this spec defines the same behavior for the CP path.

## Requirements

### Requirement: State-Sync Sidecar Presence

The CP SHALL add a `state-sync` sidecar container to the runner pod when S3 persistence is configured for the session's project.

#### Scenario: Project with S3 configured

- GIVEN a project with S3 access credentials (endpoint, bucket, access key, secret key) configured
- WHEN the CP provisions a runner pod for a session in that project
- THEN the pod spec SHALL include a `state-sync` sidecar container

#### Scenario: Project without S3

- GIVEN a project with no S3 configuration
- WHEN the CP provisions a runner pod
- THEN no sidecar SHALL be added
- AND workspace state SHALL be ephemeral (lost on pod termination)

### Requirement: Sidecar Image

The sidecar SHALL use the same `state-sync` image as the init container (`quay.io/ambient_code/vteam_state_sync`). The image reference SHOULD be configurable via the same `STATE_SYNC_IMAGE` environment variable used for the init container.

### Requirement: Initial Delay

The sidecar SHALL wait 30 seconds after starting before performing its first sync cycle. This prevents syncing a partially populated workspace while the runner is still initializing.

#### Scenario: Sidecar startup

- GIVEN a newly started runner pod with a state-sync sidecar
- WHEN the sidecar starts
- THEN it SHALL wait 30 seconds before the first sync
- AND workspace content generated during the delay SHALL be captured in the first sync

### Requirement: Periodic Sync

The sidecar SHALL sync workspace state to S3 at a configurable interval (default: 60 seconds).

Synced paths:

| Path | Content |
|---|---|
| `/workspace/<RUNNER_STATE_DIR>/` | Framework state (e.g. `.claude/` databases, config) |
| `/workspace/artifacts/` | Session-produced artifacts |
| `/workspace/file-uploads/` | User-uploaded files |

The sidecar SHALL NOT sync `/workspace/repos/` during periodic syncs — git repo state is handled separately via bundle backups.

After each sync cycle, the sidecar SHALL upload a `metadata.json` file to the S3 session path containing `lastSync` timestamp, session name, namespace, and number of paths synced.

#### Scenario: Periodic sync cycle

- GIVEN a running session with S3 configured and `SYNC_INTERVAL=60`
- WHEN 60 seconds elapse since the last sync
- THEN the sidecar SHALL upload changed files from the synced paths to S3
- AND the sidecar SHALL use checksum-based comparison to avoid re-uploading unchanged files

#### Scenario: Size limit exceeded

- GIVEN workspace content exceeding `MAX_SYNC_SIZE` (default: 1 GB)
- WHEN a sync cycle runs
- THEN the sidecar SHALL log a warning
- AND the sidecar SHALL continue syncing (best-effort, files over the limit may be skipped)

### Requirement: Git Repo Backup

The sidecar SHALL periodically back up git repository state to S3 at a configurable interval (default: every 5th sync cycle). For each git repository in `/workspace/repos/`:

- A **git bundle** (`repo.bundle`) containing all refs
- An **uncommitted changes patch** (`uncommitted.patch`)
- A **staged changes patch** (`staged.patch`)
- **Metadata** (`metadata.json`) including remote URL (with credentials stripped), current branch, HEAD SHA, and local branch list

Backups SHALL be stored at `s3://<bucket>/<namespace>/<session-name>/repo-state/<repo-name>/`.

#### Scenario: Git repo backup cycle

- GIVEN a session with a cloned repo at `/workspace/repos/platform`
- AND the agent has created a new branch and made uncommitted changes
- WHEN a repo backup cycle runs
- THEN the sidecar SHALL create a bundle with all refs including the new branch
- AND the sidecar SHALL capture the uncommitted changes as a patch
- AND the sidecar SHALL upload both to `repo-state/platform/` in S3

#### Scenario: Repo with embedded credentials in remote URL

- GIVEN a repo cloned with `https://x-access-token:TOKEN@github.com/org/repo`
- WHEN the sidecar writes `metadata.json`
- THEN the remote URL SHALL be sanitized to `https://github.com/org/repo`
- AND the token SHALL NOT appear in any persisted metadata

### Requirement: Graceful Shutdown

On `SIGTERM` or `SIGINT`, the sidecar SHALL perform a final sync that includes both workspace state and a full git repo backup before exiting. This ensures state captured between the last periodic sync and pod termination is not lost.

#### Scenario: Pod termination

- GIVEN a running session with unsaved workspace changes
- WHEN the pod receives `SIGTERM`
- THEN the sidecar SHALL perform a final git repo backup
- AND the sidecar SHALL perform a final workspace sync
- AND the sidecar SHALL exit after both complete

### Requirement: Sidecar Environment

The sidecar SHALL receive the following environment variables:

| Variable | Source | Default |
|---|---|---|
| `SESSION_NAME` | session ID | (required) |
| `NAMESPACE` | project ID | (required) |
| `S3_ENDPOINT` | project or cluster config | `http://minio.ambient-code.svc:9000` |
| `S3_BUCKET` | project or cluster config | `ambient-sessions` |
| `AWS_ACCESS_KEY_ID` | project or cluster secret | (required) |
| `AWS_SECRET_ACCESS_KEY` | project or cluster secret | (required) |
| `SYNC_INTERVAL` | CP config | `60` |
| `MAX_SYNC_SIZE` | CP config | `1073741824` (1 GB) |
| `REPO_BACKUP_INTERVAL` | CP config | `5` |
| `RUNNER_STATE_DIR` | CP config | `.claude` |

**Note on S3 access credentials:** `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` are S3 object storage credentials, not Ambient Credential Kind entities. They are resolved from project-level or cluster-level Kubernetes Secrets.

### Requirement: Input Sanitization

The sidecar SHALL sanitize `SESSION_NAME` and `NAMESPACE` values to prevent path traversal in S3 paths. Only alphanumeric characters and hyphens SHALL be permitted.

### Requirement: S3 Configuration Source

The CP SHALL resolve S3 configuration from the session's project. The project MAY provide S3 access credentials via a Kubernetes Secret or via the api-server's project settings. When no S3 configuration is found, the CP SHALL skip both the init container's S3 hydration and the state-sync sidecar entirely — the session runs with ephemeral storage only.

#### Scenario: Project with shared cluster S3

- GIVEN a project with no custom S3 config
- AND the cluster has shared MinIO credentials available
- WHEN the CP provisions a pod
- THEN the CP SHALL use the shared MinIO credentials for S3 operations

#### Scenario: Project with custom S3

- GIVEN a project with a custom S3 endpoint and credentials configured
- WHEN the CP provisions a pod
- THEN the CP SHALL use the project's custom S3 config

### Requirement: Exclude Patterns

The sidecar SHALL exclude the following patterns from sync to avoid uploading build artifacts and caches:

- `repos/**` (git-managed separately)
- `node_modules/**`, `.venv/**`, `__pycache__/**`, `.cache/**`, `*.pyc`
- `target/**`, `dist/**`, `build/**`
- `.git/**`
- `debug/**` (symlinks that break rclone)

### Requirement: Volume Sharing

The sidecar SHALL mount the same `emptyDir` volume at `/workspace` as the runner container. It SHALL also mount the framework state subdirectory (e.g., `/workspace/.claude`) at `/app/<RUNNER_STATE_DIR>` via a subPath mount for direct access to framework databases.

### Requirement: Security Context

The sidecar SHALL run with a restricted security context:

- `allowPrivilegeEscalation: false`
- `capabilities: drop: ["ALL"]`
- `readOnlyRootFilesystem: false` (required because the sidecar writes rclone config to `/tmp` and creates temporary files for git bundles)

The sidecar does not require root privileges. Unlike the init container, it does not need `chown` access and SHOULD run as a non-root user where the pod security policy permits it.

### Requirement: SQLite Consistency

Before syncing framework state directories that contain SQLite databases, the sidecar SHALL issue a WAL checkpoint (`PRAGMA wal_checkpoint(TRUNCATE)`) to ensure database files are in a consistent state for upload.

#### Scenario: Claude Code SQLite database sync

- GIVEN the runner has an active SQLite database at `/workspace/.claude/projects.db`
- WHEN a sync cycle runs
- THEN the sidecar SHALL checkpoint the WAL before uploading
- AND the uploaded database SHALL be in a consistent, readable state

## Status

**Not implemented.** The CP does not create a state-sync sidecar, does not resolve S3 configuration, and does not support workspace persistence. All CP-provisioned sessions use ephemeral storage. The operator implements this in `components/operator/internal/handlers/sessions.go:1376-1419` (sidecar) and `components/operator/internal/handlers/sessions.go:2016-2090` (S3 config resolution).
