# CI Improvements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce PR feedback loop wall-clock time from ~10.4m P50 to ~5-7m through targeted caching and shared Docker layer reuse.

**Architecture:** Replace plain `docker build` in E2E with BuildKit builds that read from the same GHA cache scopes written by components-build-deploy. Add kind binary caching. Consolidate redundant golangci-lint passes. Replace pip-installed junit2html with pipx.

**Tech Stack:** GitHub Actions, Docker BuildKit, GHA cache, golangci-lint, pipx

---

### Task 1: Add Docker BuildKit Layer Caching to E2E Image Builds

**Files:**
- Modify: `.github/workflows/e2e.yml:86-148`

This is the highest-impact change. The E2E workflow currently uses plain `docker build` for 4 component images with no layer caching. We replace the monolithic shell script with individual `docker/build-push-action@v7` steps that use `cache-from` to read layers from both the components-build-deploy workflow's cache and the E2E's own cache.

The components-build-deploy workflow writes cache with scopes like `frontend-amd64`, `backend-amd64`, etc. E2E runs on `ubuntu-latest` (amd64), so it can read those layers directly.

- [ ] **Step 1: Replace the monolithic build step with individual buildx build steps**

Replace the single "Build component images from PR code" step (lines 90-148) with 4 individual conditional steps. Each uses `docker/build-push-action@v7` with `load: true` (loads into local Docker daemon instead of pushing to registry) and reads from the components-build cache scope.

Replace this block (lines 90-148):

```yaml
    - name: Build component images from PR code
      run: |
        echo "======================================"
        ...entire shell script...
```

With these 4 steps:

```yaml
    - name: Build or pull frontend image
      if: needs.detect-changes.outputs.frontend == 'true'
      uses: docker/build-push-action@v7
      with:
        context: components/frontend
        file: components/frontend/Dockerfile
        load: true
        tags: quay.io/ambient_code/vteam_frontend:e2e-test
        cache-from: |
          type=gha,scope=frontend-amd64
          type=gha,scope=e2e-frontend
        cache-to: type=gha,mode=max,scope=e2e-frontend

    - name: Pull frontend latest (unchanged)
      if: needs.detect-changes.outputs.frontend != 'true'
      run: |
        docker pull quay.io/ambient_code/vteam_frontend:latest
        docker tag quay.io/ambient_code/vteam_frontend:latest quay.io/ambient_code/vteam_frontend:e2e-test

    - name: Build or pull backend image
      if: needs.detect-changes.outputs.backend == 'true'
      uses: docker/build-push-action@v7
      with:
        context: components/backend
        file: components/backend/Dockerfile
        load: true
        tags: quay.io/ambient_code/vteam_backend:e2e-test
        cache-from: |
          type=gha,scope=backend-amd64
          type=gha,scope=e2e-backend
        cache-to: type=gha,mode=max,scope=e2e-backend

    - name: Pull backend latest (unchanged)
      if: needs.detect-changes.outputs.backend != 'true'
      run: |
        docker pull quay.io/ambient_code/vteam_backend:latest
        docker tag quay.io/ambient_code/vteam_backend:latest quay.io/ambient_code/vteam_backend:e2e-test

    - name: Build or pull operator image
      if: needs.detect-changes.outputs.operator == 'true'
      uses: docker/build-push-action@v7
      with:
        context: components/operator
        file: components/operator/Dockerfile
        load: true
        tags: quay.io/ambient_code/vteam_operator:e2e-test
        cache-from: |
          type=gha,scope=operator-amd64
          type=gha,scope=e2e-operator
        cache-to: type=gha,mode=max,scope=e2e-operator

    - name: Pull operator latest (unchanged)
      if: needs.detect-changes.outputs.operator != 'true'
      run: |
        docker pull quay.io/ambient_code/vteam_operator:latest
        docker tag quay.io/ambient_code/vteam_operator:latest quay.io/ambient_code/vteam_operator:e2e-test

    - name: Build or pull ambient-runner image
      if: needs.detect-changes.outputs.claude-runner == 'true'
      uses: docker/build-push-action@v7
      with:
        context: components/runners
        file: components/runners/ambient-runner/Dockerfile
        load: true
        tags: quay.io/ambient_code/vteam_claude_runner:e2e-test
        cache-from: |
          type=gha,scope=ambient-runner-amd64
          type=gha,scope=e2e-ambient-runner
        cache-to: type=gha,mode=max,scope=e2e-ambient-runner

    - name: Pull ambient-runner latest (unchanged)
      if: needs.detect-changes.outputs.claude-runner != 'true'
      run: |
        docker pull quay.io/ambient_code/vteam_claude_runner:latest
        docker tag quay.io/ambient_code/vteam_claude_runner:latest quay.io/ambient_code/vteam_claude_runner:e2e-test

    - name: Show built images
      run: docker images | grep e2e-test
```

- [ ] **Step 2: Validate YAML syntax**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/e2e.yml'))"`
Expected: No output (valid YAML)

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/e2e.yml
git commit -m "ci(e2e): add Docker BuildKit layer caching to image builds

Read from components-build-deploy cache scopes (frontend-amd64, etc.)
so E2E gets warm layers from the last main build. Falls back to
building uncached if cache misses."
```

---

### Task 2: Cache kind Binary in E2E

**Files:**
- Modify: `.github/workflows/e2e.yml:1-10` (add env block)
- Modify: `.github/workflows/e2e.yml:150-155` (replace Install kind step)

The E2E workflow downloads `kind v0.27.0` from the internet every run. Add an `actions/cache` step matching the pattern already used in `test-local-dev.yml`.

- [ ] **Step 1: Add KIND_VERSION env var at the workflow level**

After the `on:` block and before `concurrency:`, add:

```yaml
env:
  KIND_VERSION: "v0.27.0"
```

- [ ] **Step 2: Replace the "Install kind" step with a cached version**

Replace this step:

```yaml
    - name: Install kind
      run: |
        curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.27.0/kind-linux-amd64
        chmod +x ./kind
        sudo mv ./kind /usr/local/bin/kind
        kind version
```

With:

```yaml
    - name: Cache kind binary
      uses: actions/cache@v4
      id: kind-cache
      with:
        path: ~/k8s-tools/kind
        key: kind-${{ runner.os }}-${{ env.KIND_VERSION }}

    - name: Install kind
      run: |
        mkdir -p ~/k8s-tools
        if [[ ! -f ~/k8s-tools/kind ]]; then
          echo "Downloading kind $KIND_VERSION..."
          curl -sLo ~/k8s-tools/kind "https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-amd64"
          chmod +x ~/k8s-tools/kind
        else
          echo "Using cached kind"
        fi
        sudo cp ~/k8s-tools/kind /usr/local/bin/kind
        kind version
```

- [ ] **Step 3: Validate YAML syntax**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/e2e.yml'))"`
Expected: No output (valid YAML)

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/e2e.yml
git commit -m "ci(e2e): cache kind binary between runs

Pin version in env var for cache key stability. Matches pattern
used in test-local-dev.yml."
```

---

### Task 3: Consolidate golangci-lint Passes in Lint Workflow

**Files:**
- Modify: `.github/workflows/lint.yml:144-156`

The `go-backend` job runs `golangci-lint` twice — once with default tags and once with `--build-tags=test`. The test tag is a superset: files with `//go:build test` are only compiled when the tag is present, so linting with `--build-tags=test` covers all production files plus test-tagged files. Replace two passes with one.

- [ ] **Step 1: Replace the two golangci-lint steps with one**

Replace these two steps in the `go-backend` job:

```yaml
      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v9
        with:
          version: latest
          working-directory: components/backend
          args: --timeout=5m

      - name: Run golangci-lint (test build tags)
        uses: golangci/golangci-lint-action@v9
        with:
          version: latest
          working-directory: components/backend
          args: --timeout=5m --build-tags=test
```

With a single step:

```yaml
      - name: Run golangci-lint (all build tags)
        uses: golangci/golangci-lint-action@v9
        with:
          version: latest
          working-directory: components/backend
          args: --timeout=5m --build-tags=test
```

- [ ] **Step 2: Validate YAML syntax**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/lint.yml'))"`
Expected: No output (valid YAML)

- [ ] **Step 3: Verify locally that test-tagged lint catches everything**

Run: `cd components/backend && golangci-lint run --timeout=5m --build-tags=test 2>&1 | tail -5`
Expected: Same or superset of issues found by default tags.

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/lint.yml
git commit -m "ci(lint): consolidate golangci-lint to single pass with test tags

The test build tag is a superset of default — files with
//go:build test are only included when the tag is present. A single
pass with --build-tags=test covers all code."
```

---

### Task 4: Replace pip-installed junit2html with pipx in Unit Tests

**Files:**
- Modify: `.github/workflows/unit-tests.yml:131-137`

The backend unit test job runs `pip install junit2html` without caching on every run. `pipx` is pre-installed on GitHub Actions runners and handles isolated installs. Using `pipx run` avoids the install step entirely — pipx downloads and caches the tool itself.

- [ ] **Step 1: Replace pip install with pipx run**

Replace this step:

```yaml
      - name: Install Junit2Html plugin and generate report
        if: (!cancelled())
        shell: bash
        run: |
          pip install junit2html
          junit2html ${{ env.TESTS_DIR }}/reports/${{ env.JUNIT_FILENAME }} ${{ env.TESTS_DIR }}/reports/test-report.html
        continue-on-error: true
```

With:

```yaml
      - name: Generate HTML test report
        if: (!cancelled())
        shell: bash
        run: |
          pipx run junit2html ${{ env.TESTS_DIR }}/reports/${{ env.JUNIT_FILENAME }} ${{ env.TESTS_DIR }}/reports/test-report.html
        continue-on-error: true
```

- [ ] **Step 2: Validate YAML syntax**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/unit-tests.yml'))"`
Expected: No output (valid YAML)

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/unit-tests.yml
git commit -m "ci(unit-tests): use pipx for junit2html instead of pip install

pipx is pre-installed on GHA runners and handles caching. Avoids
uncached pip install on every run."
```
