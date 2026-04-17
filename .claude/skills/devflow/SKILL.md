---
name: devflow
description: >-
  End-to-end development workflow for the Ambient Code Platform.
  Covers ticket → branch → spec → code → commit → PR → review → deploy → verify.
  Self-improving: update this skill whenever a failure path is discovered or a
  shortcut is validated. Git history is the audit trail.
---

# Devflow Skill

You are executing the Ambient Code Platform development workflow. This skill is the literal sequence of steps performed every time code is changed — from filing a ticket to verifying a deployment.

> **Self-improving skill.** Whenever you discover something that doesn't work, or discover something that does, **update this file immediately** before moving on. The next invocation of this skill benefits from every correction. Examples of self-improvement triggers:
> - A build command fails and you find the right flags → add them here
> - A test requires an env var you didn't know about → add it here
> - A path you tried is a dead end → document the dead end and the correct path
> - A review comment reveals a missing step → add the step here

---

## Turn Counter

**Remind the human every 10 turns to re-read this skill.** Context compaction can lose workflow state. At turns 10, 20, 30, etc., say:

> "Reminder: we are on turn N. Re-reading `.claude/skills/devflow/SKILL.md` to refresh workflow state."

Then re-read this file before continuing.

---

## Workflow Overview

```text
1. Ticket (optional)
2. Branch (from ticket name if available)
3. Spec change (load + modify the component's .spec.md)
4. Commit spec → push → PR against `main`
5. Review (agentic: thorough PR review; human: desired but optional)
6. On spec acceptance → load the component's .guide.md (the implementation workflow)
7. Branch (from ticket name if available)
8. Commit order:
   a. Guide/skill/workflow self-improvement changes (1 commit)
   b. Code changes grouped by component
9. Push → PR against `main`
10. Review (human + bot comments; local review too)
11. On merge → await quay.io image
12. Update gitops repository with new image SHA (Skill TBD)
13. After deployment → run tests / e2e suite (TBD)
```

---

## Phase 1: Ticket & Branch

### 1a. File a Ticket (optional)

If there is a Jira or GitHub issue, note the ticket ID. If not, proceed without one.

### 1b. Create a Branch

```bash
# With ticket
git checkout -b <ticket-id>/<short-description>

# Without ticket
git checkout -b <type>/<short-description>
# type: feat, fix, refactor, docs, chore
```

---

## Phase 2: Spec Change

### 2a. Load the Spec

Every component has a spec file. Load the spec for the component being changed:

| Component | Spec | Guide |
|-----------|------|-------|
| Data Model / API / CLI / RBAC | `docs/internal/design/ambient-model.spec.md` | `docs/internal/design/ambient-model.guide.md` |

### 2b. Modify the Spec

Edit the spec to reflect the desired change. The spec is the source of truth — code is reconciled against it.

### 2c. Commit Spec Change

```bash
git add docs/internal/design/
git commit -m "spec(<component>): <description of spec change>"
```

### 2d. Push and Create PR

```bash
git push origin <branch-name> -u
gh pr create --base main --title "spec(<component>): <description>" --body "Spec change for review."
```

- **Agentic workflow:** thorough review occurs in the PR. Feedback is incorporated. On approval/merge, the spec change becomes requirements.
- **Human workflow:** review is desired but spec commit doesn't need to be merged before code work begins.

---

## Phase 3: Implementation

### 3a. Load the Guide

After spec acceptance (or in parallel for human workflow), load the implementation guide for the component:

| Component | Guide | Context |
|-----------|-------|---------|
| Data Model / API | `docs/internal/design/ambient-model.guide.md` | `.claude/context/api-server-development.md` |
| SDK | (guide section in ambient-model.guide.md Wave 3) | `.claude/context/sdk-development.md` |
| CLI | (guide section Wave 5) | `.claude/context/cli-development.md` |
| Control Plane | (guide section Wave 4) | `.claude/context/control-plane-development.md` |
| Operator | (guide section Wave 5) | `.claude/context/operator-development.md` |
| Frontend | (guide section Wave 6) | `.claude/context/frontend-development.md` |
| Backend (V1) | — | `.claude/context/backend-development.md` |

### 3b. Create Implementation Branch

```bash
# If separate from spec branch
git checkout -b <ticket-id>/<short-description>-impl

# Or continue on the same branch (human workflow, spec not yet merged)
```

### 3c. Commit Order

Commits are ordered deliberately:

1. **Self-improvement commit (first).** Any changes to guides, skills, workflows, or this devflow skill. Group all such changes into one commit:
   ```bash
   git add .claude/skills/ .claude/context/ docs/internal/design/*.guide.md
   git commit -m "chore(skills): self-improvement updates from <context>"
   ```
   - Agentic workflow: this is the first commit on the branch.
   - Human workflow: this is the second commit (behind the spec change), or first if spec is on a separate branch.

2. **Code changes, grouped by component:**
   ```bash
   git add components/ambient-api-server/
   git commit -m "feat(api-server): <description>"

   git add components/ambient-cli/
   git commit -m "feat(cli): <description>"

   git add components/ambient-sdk/
   git commit -m "feat(sdk): <description>"
   ```

### 3d. Push and Create PR

```bash
git push origin <branch-name> -u

# Agentic: new PR against main
gh pr create --base main --title "feat(<component>): <description>" --body "Implementation of <spec change>."

# Human: push adds to existing PR, or create new PR
```

---

## Phase 4: Review

- Human and bot comments on the PR.
- Conduct local reviews as well.
- Incorporate feedback → push additional commits.

---

## Phase 5: Post-Merge Deployment

### 5a. Await Image Build

After merge to `main`, images are built and pushed to `quay.io/ambient_code/`. Watch for the image:

```bash
# Check if image exists for a given SHA
skopeo inspect docker://quay.io/ambient_code/vteam_api_server:<sha-or-tag>
```

### 5b. Update GitOps Repository

> **Skill TBD.** This step will be formalized as experience accumulates.

Get the image SHA from quay.io and update the gitops repository's image references. Due to lack of autodeploy, this is a manual step.

### 5c. Verify Deployment

After deployment, run all available tests. E2E test suite TBD.

---

## Component Build & Test Quick Reference

### API Server (`components/ambient-api-server/`)

```bash
cd components/ambient-api-server

# CRITICAL: podman, not docker. And RYUK must be disabled.
systemctl --user start podman.socket
export DOCKER_HOST=unix:///run/user/$(id -u)/podman/podman.sock
export TESTCONTAINERS_RYUK_DISABLED=true

make generate    # after openapi changes
make proto       # after .proto changes
make binary      # compile
make test        # integration tests (testcontainer Postgres)
go fmt ./...
go vet ./...
golangci-lint run
```

### CLI (`components/ambient-cli/`)

```bash
cd components/ambient-cli
go build ./...
go fmt ./...
go vet ./...
golangci-lint run
```

### SDK (`components/ambient-sdk/`)

```bash
cd components/ambient-sdk
make generate    # regenerate from openapi
cd go-sdk && go build ./...
```

### Frontend (`components/frontend/`)

```bash
cd components/frontend
npm run build    # must pass with 0 errors, 0 warnings
npx vitest run --coverage
```

### Runner (`components/runners/ambient-runner/`)

```bash
cd components/runners/ambient-runner
uv venv && uv pip install -e .
python -m pytest tests/
```

### Control Plane (`components/ambient-control-plane/`)

```bash
cd components/ambient-control-plane
go build ./...
go fmt ./...
go vet ./...
golangci-lint run
```

---

## Known Pitfalls (Self-Improving Section)

This section is the most important part of the skill. Every dead end, every wasted cycle, every "oh, you need to do X first" gets recorded here.

### API Server Tests Require Podman + RYUK Disabled

**Dead end:** Running `make test` with Docker or without `TESTCONTAINERS_RYUK_DISABLED=true`.
**What happens:** Ryuk container tries to connect to `/var/run/docker.sock`, which doesn't exist on this system. Tests abort before running.
**Correct path:**
```bash
systemctl --user start podman.socket
export DOCKER_HOST=unix:///run/user/$(id -u)/podman/podman.sock
export TESTCONTAINERS_RYUK_DISABLED=true
make test
```

### Kind Image Loading with Podman

**Dead end:** `kind load docker-image` fails with podman `localhost/` prefix images.
**Correct path:** Use `podman save | ctr import`:
```bash
CLUSTER=$(podman ps --format '{{.Names}}' | grep 'kind' | grep 'control-plane' | sed 's/-control-plane//')
podman build --no-cache -t localhost/vteam_api_server:latest components/ambient-api-server
podman save localhost/vteam_api_server:latest | \
  podman exec -i ${CLUSTER}-control-plane ctr --namespace=k8s.io images import -
kubectl rollout restart deployment/ambient-api-server -n ambient-code
```

### `--no-cache` Required for Podman Builds

**Dead end:** Building without `--no-cache` when only Go source changed (go.mod/go.sum unchanged).
**What happens:** The `go build` layer hits cache and emits the old binary.
**Correct path:** Always `podman build --no-cache` for Go components.

### Generated Code Middleware Import

**Dead end:** Using generated plugin code as-is.
**What happens:** Generated `RegisterRoutes` uses `auth.JWTMiddleware` — wrong.
**Correct path:** Replace with `environments.JWTMiddleware` after every generation.

### Nested Route mux.Vars Keys

**Dead end:** Using generated handler code for nested routes without fixing variable names.
**What happens:** `mux.Vars(r)["id"]` returns empty string because the route uses `{pa_id}` or `{msg_id}`.
**Correct path:** Fix `mux.Vars` keys to match actual route variable names after generation.

### Go SDK Streaming Endpoints

**Dead end:** Trying to use `do()` or `doMultiStatus()` for SSE endpoints.
**What happens:** These methods unmarshal and close the response body.
**Correct path:** Use `a.client.httpClient.Do(req)` directly and return `resp.Body` as `io.ReadCloser`.

### CLI Dependencies

**Dead end:** Adding a new import to CLI code without running `go build` immediately.
**What happens:** `missing go.sum entry` at commit time.
**Correct path:** Run `go build ./...` after adding any new import. Fix `go.mod` before committing.

### gRPC Port Conflict

**Dead end:** Assuming port 9000 is available locally.
**What happens:** MinIO or another service occupies 9000. gRPC streaming fails silently.
**Correct path:**
```bash
kubectl port-forward svc/ambient-api-server 19000:9000 -n ambient-code &
export AMBIENT_GRPC_URL=127.0.0.1:19000
```

### Pre-commit Hooks

The project uses pre-commit hooks (`.pre-commit-config.yaml`). If hooks block a commit:
```bash
git commit --no-verify    # skip pre-commit hooks (use sparingly)
```
But prefer fixing the lint/format issue instead.

---

## Related Skills & Context

| Skill/Context | When to use |
|---------------|-------------|
| `.claude/skills/ambient-api-server/SKILL.md` | Working in `components/ambient-api-server/` |
| `.claude/skills/dev-cluster/SKILL.md` | Managing kind clusters for testing |
| `.claude/skills/grpc-dev/SKILL.md` | gRPC streaming, AG-UI events |
| `.claude/skills/ambient-pr-test/SKILL.md` | PR validation on MPP dev cluster |
| `.claude/context/api-server-development.md` | API server plugin architecture, OpenAPI, testing |
| `.claude/context/cli-development.md` | CLI command patterns |
| `.claude/context/sdk-development.md` | SDK generation |
| `.claude/context/control-plane-development.md` | CP fan-out, runner contract |
| `.claude/context/frontend-development.md` | Frontend build, React Query |
| `docs/internal/design/ambient-model.spec.md` | Data model spec (source of truth) |
| `docs/internal/design/ambient-model.guide.md` | Implementation guide (wave-based workflow) |

---

## Self-Improvement Log

Record every correction here with a date stamp. This section grows over time.

### 2026-04-10 — Initial creation

- Established devflow skill from user requirements.
- Seeded Known Pitfalls from existing context files and guide lessons learned.
- Documented TESTCONTAINERS_RYUK_DISABLED requirement (discovered in api-server-development.md).
- Documented podman `--no-cache` and `ctr import` workarounds (discovered in ambient-model.guide.md).
- Documented gRPC port 19000 workaround (discovered in ambient-model.guide.md).
- GitOps deployment step marked as TBD — will be formalized as the workflow runs.
- E2E test suite marked as TBD.
