# CI Improvements: Reduce PR Feedback Loop

## Problem

PR wall-clock time is gated by the slowest workflow. Current P50 times:

| Workflow | P50 | Primary Bottleneck |
|----------|-----|--------------------|
| E2E Tests | 10.4m | Docker builds from scratch, no layer cache |
| Test Local Dev | 4.9m | Full kind cluster + image build |
| Docker Builds | 3.5m | Already cached (GHA layer cache) |
| Unit Tests | 3.3m | Well-parallelized |
| Lint | 1.6m | golangci-lint runs twice per Go component |

## Approach

Two complementary strategies:

### A: Targeted Caching + Quick Wins

Incremental improvements across all PR workflows without structural changes.

### B: E2E Image Reuse

Restructure E2E to consume images already built by the `components-build-deploy` workflow instead of rebuilding them.

## Changes

### 1. E2E Docker Layer Caching (Approach A)

**File:** `.github/workflows/e2e.yml`

Replace plain `docker build` commands with `docker buildx build` using GHA cache:

```yaml
- name: Build frontend
  if: needs.detect-changes.outputs.frontend == 'true'
  uses: docker/build-push-action@v7
  with:
    context: components/frontend
    file: components/frontend/Dockerfile
    load: true
    tags: quay.io/ambient_code/vteam_frontend:e2e-test
    cache-from: type=gha,scope=e2e-frontend
    cache-to: type=gha,mode=max,scope=e2e-frontend
```

Repeat for backend, operator, ambient-runner (4 images total).

**Expected savings:** 3-5 min on rebuilds (layer cache hits for unchanged layers).

### 2. Cache kind Binary in E2E (Approach A)

**File:** `.github/workflows/e2e.yml`

Currently downloads kind v0.27.0 every run. Add `actions/cache` for the binary, matching the pattern already used in `test-local-dev.yml`.

Pin version in an env var for cache key stability.

**Expected savings:** ~10-15s per run.

### 3. Consolidate golangci-lint Passes (Approach A)

**File:** `.github/workflows/lint.yml`

The `go-backend` job runs `golangci-lint-action` twice:
1. Default build tags
2. `--build-tags=test`

The test tag is a superset — files with `//go:build test` are only compiled when the tag is present, so linting with `--build-tags=test` covers all files. Consolidate to a single pass with `--build-tags=test`.

This applies only to `go-backend` — the other Go components (operator, api-server, cli) only run golangci-lint once.

**Expected savings:** ~30s on backend lint.

### 4. Cache junit2html in Unit Tests (Approach A)

**File:** `.github/workflows/unit-tests.yml`

The backend job runs `pip install junit2html` without caching. Add pip caching to the Go job's post-test reporting step, or pre-install in a cached venv.

Simpler: use `pipx run junit2html` which avoids install entirely (pipx is pre-installed on GitHub runners).

**Expected savings:** ~10s.

### 5. E2E Image Reuse from Components Build (Approach B)

**File:** `.github/workflows/e2e.yml`

The `components-build-deploy` workflow already builds all images on PR and tags them `pr-<number>`. Instead of E2E rebuilding images, it should:

1. Add `needs` dependency or use `workflow_run` trigger to wait for components-build
2. Pull pre-built `pr-<number>` images from Quay.io
3. Fall back to local build only if components-build was skipped (no component changes)

**Implementation options:**

**Option 1: workflow_run trigger** — E2E triggers after components-build completes. Clean separation but adds latency waiting for the full build matrix (14 jobs).

**Option 2: Direct pull in E2E job** — E2E attempts to pull `pr-<number>` tagged images. If pull fails (image not pushed yet or components-build skipped), falls back to local build. This is simpler and doesn't create a hard dependency.

**Recommendation: Option 2 (direct pull with fallback).** The components-build workflow only pushes on non-PR events currently (`if: github.event_name != 'pull_request'`), so we need to either:
- Enable PR image pushes in components-build (to a PR-specific tag), or
- Use the GHA Docker layer cache directly (E2E reads from same cache scope as components-build)

The cleanest path: **share GHA cache scopes.** E2E's `cache-from` references the same scope that components-build writes to. No image push/pull needed — just shared BuildKit cache layers.

```yaml
# In e2e.yml - reuse cache from components-build
cache-from: |
  type=gha,scope=frontend-amd64
  type=gha,scope=e2e-frontend
cache-to: type=gha,mode=max,scope=e2e-frontend
```

This way E2E gets warm cache from the last components-build run on main, plus its own cache from prior E2E runs.

**Expected savings:** 3-5 min (near-instant builds when layers haven't changed).

## What We're NOT Changing

- **Test Local Dev workflow:** Already well-optimized with k8s tools caching. The ~5 min is inherent to kind cluster bootstrap + image build + deploy. No easy wins without fundamentally changing the approach.
- **Workflow consolidation:** Not merging lint + unit-tests. The separation is clean and the overhead is marginal (~30s for detect-changes + summary).
- **Components-build-deploy push behavior:** Not enabling image pushes on PRs — the shared cache approach achieves the same benefit without registry overhead.

## Risk Assessment

| Change | Risk | Mitigation |
|--------|------|------------|
| Docker layer caching in E2E | Low — additive, fallback is uncached build | GHA cache has 10GB limit; scope names prevent collision |
| kind caching | None — identical pattern to test-local-dev | Pin version in env var |
| golangci-lint consolidation | Low — test tag is superset | Verify locally that `--build-tags=test` finds all issues |
| junit2html via pipx | None — pipx pre-installed on runners | Can fall back to pip install |
| Shared cache scopes | Medium — cache eviction if 10GB limit hit | Use `mode=max` and scope naming to partition |

## Expected Outcome

| Workflow | Current P50 | Expected P50 | Savings |
|----------|-------------|--------------|---------|
| E2E Tests | 10.4m | ~5-7m | 3-5m |
| Lint | 1.6m | ~1.2m | ~30s |
| Unit Tests | 3.3m | ~3.1m | ~10s |
| **PR wall clock** | **~10.4m** | **~5-7m** | **3-5m** |
