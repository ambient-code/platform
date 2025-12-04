# Testing Infrastructure Summary - Ambient Code Platform

**Last Updated**: 2025-12-02
**Status**: Production testing infrastructure with identified gaps

---

## Quick Navigation
- [Test Inventory](#test-inventory) - What tests exist and where
- [GitHub Actions Workflows](#github-actions-test-workflows) - CI/CD orchestration
- [Local Test Execution](#local-test-execution) - How to run tests locally
- [Results & Reporting](#test-results--reporting) - Where to find test results
- [Blocking & Governance](#blocking--merge-governance) - What prevents merges
- [Known Gaps](#known-gaps--recommendations) - Improvement opportunities

---

## Test Inventory

### Component Test Matrix

| Component | Location | Framework | Test Types | CI Enforcement | Coverage |
|-----------|----------|-----------|------------|----------------|----------|
| **E2E** | `e2e/cypress/e2e/` | Cypress 13.x | Full-stack integration | ✅ **Blocking** | 5 test cases |
| **Backend** | `components/backend/tests/` | Go test | Unit, Contract, Integration | ⚠️ **Not enforced** | Partial |
| **Frontend** | N/A | N/A | None | ❌ **Missing** | 0% |
| **Operator** | `components/operator/internal/handlers/` | Go test | Unit | ⚠️ **Not enforced** | Minimal |
| **Claude Runner** | `components/runners/claude-code-runner/tests/` | pytest | Unit | ✅ **Blocking** | Codecov tracked |

### Test File Locations

**E2E Tests** (`e2e/`):
- `cypress/e2e/vteam.cy.ts` - Main test suite (5 test cases)
- `cypress/support/commands.ts` - Custom Cypress commands
- `cypress.config.ts` - Cypress configuration

**Backend Tests** (`components/backend/tests/`):
- `unit/` - Unit tests (handlers, utilities)
- `contract/` - API contract validation
- `integration/` - K8s integration tests (requires cluster)
  - `gitlab/gitlab_integration_test.go` - GitLab integration
- `regression/backward_compat_test.go` - Backward compatibility

**Operator Tests** (`components/operator/`):
- `internal/handlers/sessions_test.go` - Session handler tests

**Claude Runner Tests** (`components/runners/claude-code-runner/tests/`):
- `test_observability.py` - Observability utilities
- `test_security_utils.py` - Security utilities
- `test_model_mapping.py` - Model mapping logic
- `test_wrapper_vertex.py` - Vertex AI wrapper
- `test_duplicate_turn_prevention.py` - Duplicate turn handling
- `test_langfuse_model_metadata.py` - Langfuse integration

---

## GitHub Actions Test Workflows

### Workflow Summary Table

| Workflow | File | Triggers | Tests Executed | Blocking | Artifacts |
|----------|------|----------|----------------|----------|-----------|
| **E2E Tests** | `e2e.yml` | PR, push to main, manual | Cypress full-stack | ✅ Yes | Screenshots, videos, logs (7-day retention) |
| **Go Linting** | `go-lint.yml` | PR, push to main, manual | gofmt, go vet, golangci-lint | ✅ Yes | Lint reports |
| **Frontend Linting** | `frontend-lint.yml` | PR, push to main, manual | ESLint, TypeScript, build | ✅ Yes | Build logs |
| **Runner Tests** | `runner-tests.yml` | PR (runner changes), push | pytest (observability, security) | ✅ Yes | Coverage XML (Codecov) |
| **Local Dev Tests** | `test-local-dev.yml` | Manual | CRC smoke tests | ⚠️ Advisory | Logs |

### Detailed Workflow Descriptions

#### 1. E2E Tests (`e2e.yml`)

**Purpose**: Full-stack integration testing in a real Kubernetes environment

**Workflow Steps**:
1. **Change Detection** - Identifies modified components (frontend, backend, operator, runner)
2. **Conditional Builds** - Builds only changed components, pulls `latest` for unchanged ones
3. **Kind Cluster Setup** - Creates `vteam-e2e` cluster (vanilla Kubernetes)
4. **Image Loading** - Loads all 4 component images into Kind cluster
5. **Deployment** - Deploys complete vTeam stack via kustomize
6. **Cypress Tests** - Runs 5 test cases covering:
   - UI authentication and loading
   - Workspace creation dialog
   - Creating new workspace
   - Listing workspaces
   - Backend API connectivity
7. **Failure Handling** - Uploads screenshots, videos, and component logs on failure
8. **Cleanup** - Destroys cluster and artifacts

**Test Coverage** (from `e2e/cypress/e2e/vteam.cy.ts`):
- ✅ Token authentication flow
- ✅ Frontend UI rendering and navigation
- ✅ Workspace creation (end-to-end user journey)
- ✅ Backend API `/api/cluster-info` endpoint
- ❌ Actual session execution (requires Anthropic API key)

**Optimization Features**:
- Change detection reduces build time (only builds changed components)
- Conditional image pulls leverage existing `latest` tags
- 20-minute timeout prevents hung tests

**Artifacts on Failure**:
- Cypress screenshots (`cypress/screenshots/`) - 7-day retention
- Cypress videos (`cypress/videos/`) - 7-day retention
- Frontend logs (last 100 lines)
- Backend logs (last 100 lines)
- Operator logs (last 100 lines)

#### 2. Go Linting (`go-lint.yml`)

**Purpose**: Enforce Go code quality standards for backend and operator

**Checks Performed**:
- **gofmt** - Verifies code formatting (zero tolerance policy)
- **go vet** - Detects suspicious constructs (unreachable code, incorrect formats, etc.)
- **golangci-lint** - Comprehensive linting (20+ linters)

**Trigger Optimization**: Only runs when Go files or `go.mod`/`go.sum` change

**Components Tested**:
- `components/backend/` - Backend API code
- `components/operator/` - Kubernetes operator code

**Blocking**: ✅ Yes - All checks must pass with zero errors/warnings

#### 3. Frontend Linting (`frontend-lint.yml`)

**Purpose**: Enforce TypeScript/JavaScript quality and validate builds

**Checks Performed**:
- **ESLint** - Code linting (style, best practices, potential bugs)
- **TypeScript Type Checking** - `npm run type-check` (no emit, validation only)
- **Build Validation** - `npm run build` ensures production build succeeds

**Trigger Optimization**: Only runs when TS/TSX/JS/JSX files or config files change

**Blocking**: ✅ Yes - Build must succeed with zero errors

**Note**: This workflow enforces code quality but does NOT run unit tests (Jest configured but no tests written)

#### 4. Claude Runner Tests (`runner-tests.yml`)

**Purpose**: Unit test critical runner utilities with coverage tracking

**Tests Executed**:
- `tests/test_observability.py` - Langfuse integration, observability utilities
- `tests/test_security_utils.py` - Token redaction, security helpers

**Coverage Reporting**:
- Generates coverage XML for `observability` and `security_utils` modules
- Uploads to Codecov with `runner` flag
- Coverage fails are advisory (CI continues)

**Trigger Optimization**: Only runs when runner code or workflow changes

**Python Version**: 3.11

**Note**: Some tests (`test_model_mapping.py`, `test_wrapper_vertex.py`) require full runtime environment and are NOT run in CI

#### 5. Local Dev Tests (`test-local-dev.yml`)

**Purpose**: Validate local development environment (OpenShift Local/CRC)

**Trigger**: Manual workflow dispatch only

**Status**: Advisory (failures don't block PRs)

---

## Local Test Execution

### Complete Test Command Reference

#### E2E Tests (Full Stack)

```bash
# Complete E2E test suite (setup → test → cleanup)
make e2e-test

# With podman (rootless container runtime)
make e2e-test CONTAINER_ENGINE=podman

# Manual E2E workflow (step-by-step)
cd e2e
./scripts/setup-kind.sh           # Create Kind cluster
./scripts/deploy.sh                # Deploy vTeam stack
./scripts/run-tests.sh             # Run Cypress tests
./scripts/cleanup.sh               # Clean up cluster
```

#### Backend Tests

```bash
cd components/backend

# Run all tests (unit + contract + integration)
make test-all

# Unit tests only
make test-unit

# Contract tests only
make test-contract

# Integration tests (requires running K8s cluster)
TEST_NAMESPACE=test-vteam make test-integration-local

# Integration tests with cleanup
CLEANUP_RESOURCES=true make test-integration-local

# Permission/RBAC tests
make test-permissions

# Coverage report (HTML output)
make test-coverage
open coverage.html
```

#### Frontend Tests

```bash
cd components/frontend

# Linting
npm run lint

# Type checking
npm run type-check

# Build validation
npm run build

# All quality checks
npm run lint && npm run type-check && npm run build
```

#### Operator Tests

```bash
cd components/operator

# Run all tests
go test ./...

# Verbose output
go test -v ./internal/handlers/...

# With coverage
go test -cover ./...
```

#### Claude Runner Tests

```bash
cd components/runners/claude-code-runner

# Install dependencies
pip install -e .
pip install pytest pytest-asyncio pytest-cov

# Run all tests
pytest

# Run specific test files
pytest tests/test_observability.py tests/test_security_utils.py -v

# With coverage
pytest --cov=observability --cov=security_utils --cov-report=html
open htmlcov/index.html
```

#### Linting (All Components)

```bash
# Backend/Operator Go linting
cd components/backend  # or components/operator
gofmt -l .                    # Check formatting (should output nothing)
go vet ./...                  # Detect suspicious constructs
golangci-lint run             # Comprehensive linting

# Auto-format Go code
gofmt -w .

# Frontend linting
cd components/frontend
npm run lint                  # ESLint
npm run lint:fix              # Auto-fix ESLint issues
```

#### Local Development Smoke Tests

```bash
# Test local OpenShift Local (CRC) environment
make dev-test

# View component logs
make dev-logs              # All components
make dev-logs-backend      # Backend only
make dev-logs-frontend     # Frontend only
make dev-logs-operator     # Operator only
```

---

## Test Results & Reporting

### GitHub Actions Results

**Location**: GitHub PR → Checks tab → Expand workflow name

**Check Status**:
- ✅ Green checkmark = All tests passed
- ❌ Red X = Tests failed (click for logs)
- 🟡 Yellow circle = Tests running
- ⚪ Gray circle = Tests queued/pending

**Viewing Logs**:
1. Click workflow name (e.g., "E2E Tests")
2. Click job name (e.g., "End-to-End Tests")
3. Expand step to see detailed output

### Artifacts (E2E Failures)

**When E2E tests fail**, artifacts are automatically uploaded:

1. **Cypress Screenshots**: `cypress-screenshots` artifact
   - Location: PR → Checks → E2E Tests → Summary → Artifacts
   - Content: PNG screenshots of failures
   - Retention: 7 days

2. **Cypress Videos**: `cypress-videos` artifact
   - Location: PR → Checks → E2E Tests → Summary → Artifacts
   - Content: MP4 videos of full test runs
   - Retention: 7 days

3. **Component Logs**: Shown in "Debug logs on failure" step
   - Frontend logs (last 100 lines)
   - Backend logs (last 100 lines)
   - Operator logs (last 100 lines)

### Coverage Reports

**Codecov Integration** (Claude Runner only):
- Dashboard: https://codecov.io (requires org access)
- Coverage tracked for: `observability`, `security_utils` modules
- Flag: `runner`
- Failure mode: Advisory (doesn't block CI)

**Local Coverage**:
```bash
# Backend
cd components/backend && make test-coverage
open coverage.html

# Runner
cd components/runners/claude-code-runner
pytest --cov-report=html
open htmlcov/index.html
```

### Test Summary Format

**E2E Test Output**:
```
✅ should access the UI with token authentication (5.2s)
✅ should open create workspace dialog (3.1s)
✅ should create a new workspace (8.4s)
✅ should list the created workspaces (2.3s)
✅ should access backend API cluster-info endpoint (1.1s)

5 passing (20s)
```

**Go Test Output**:
```
=== RUN   TestSessionHandler
=== RUN   TestSessionHandler/CreateSession
=== RUN   TestSessionHandler/GetSession
--- PASS: TestSessionHandler (0.45s)
    --- PASS: TestSessionHandler/CreateSession (0.23s)
    --- PASS: TestSessionHandler/GetSession (0.22s)
PASS
ok  	components/backend/handlers	0.456s
```

---

## Blocking & Merge Governance

### Required Status Checks (Branch Protection)

**Pull requests to `main` MUST pass**:
- ✅ E2E Tests (`e2e.yml`) - Full-stack integration tests
- ✅ Go Linting (`go-lint.yml`) - Backend/operator code quality
- ✅ Frontend Linting (`frontend-lint.yml`) - Frontend code quality
- ✅ Component Builds (`components-build-deploy.yml`) - Multi-arch builds
- ✅ Runner Tests (`runner-tests.yml`) - Python unit tests (if runner modified)

**Advisory Checks** (failures don't block):
- ⚠️ Local Dev Tests (`test-local-dev.yml`) - Manual validation only

### What Blocks Merges

**E2E Tests Fail**:
- Any of the 5 Cypress tests fail
- Deployment fails (pods not ready)
- Timeout exceeded (20 minutes)

**Linting Fails**:
- Go code not formatted (`gofmt` reports differences)
- Go vet finds suspicious constructs
- golangci-lint reports errors
- ESLint reports errors
- TypeScript type errors
- Frontend build fails

**Build Fails**:
- Docker/Podman build errors
- Image tagging/pushing errors

### Critical Gap: Backend/Operator Go Tests NOT Enforced

**Problem**: Backend and operator have comprehensive test suites (`components/backend/tests/`, `components/operator/internal/handlers/sessions_test.go`), but these tests are NOT run in CI.

**Current State**:
- ✅ Linting is enforced (gofmt, go vet, golangci-lint)
- ✅ Build is enforced (code must compile)
- ❌ `go test` is NOT run in CI

**Risk**: Breaking changes can merge if they pass linting/build but fail tests

**Recommendation**: Add to `go-lint.yml`:
```yaml
- name: Run Go tests
  run: |
    cd components/backend && go test ./...
    cd components/operator && go test ./...
```

---

## Test Architecture & Patterns

### E2E Test Architecture

**Infrastructure**:
- **Kind (Kubernetes in Docker)**: Vanilla K8s cluster (not OpenShift)
- **Cluster Name**: `vteam-e2e`
- **Namespace**: `ambient-code`
- **Ingress**: Nginx ingress controller with path-based routing

**Authentication**:
- Uses ServiceAccount tokens (not OAuth proxy)
- Test user: `test-user` ServiceAccount with `cluster-admin` permissions
- Token injected via environment variables (`OC_TOKEN`, `OC_USER`, `OC_EMAIL`)

**Change Detection Optimization**:
```bash
# If component changed: build from PR code
docker build -t quay.io/ambient_code/vteam_frontend:e2e-test ...

# If component unchanged: pull latest
docker pull quay.io/ambient_code/vteam_frontend:latest
docker tag quay.io/ambient_code/vteam_frontend:latest ...
```

**Deployment Pattern**:
1. Build/pull all 4 component images (frontend, backend, operator, runner)
2. Load images into Kind cluster
3. Update kustomization to use `e2e-test` tag
4. Deploy via `kubectl apply -k components/manifests/overlays/e2e/`
5. Wait for all pods to be ready
6. Run Cypress tests
7. Clean up cluster

### Backend Test Patterns

**Unit Tests** (`tests/unit/`):
- Isolated handler logic
- Mocked Kubernetes clients
- No external dependencies

**Contract Tests** (`tests/contract/`):
- API endpoint validation
- Request/response schemas
- HTTP status codes

**Integration Tests** (`tests/integration/`):
- Real Kubernetes cluster required
- Tests actual CR creation/deletion
- RBAC permission validation
- Uses `TEST_NAMESPACE` environment variable
- Optional cleanup via `CLEANUP_RESOURCES=true`

**Example Integration Test**:
```go
func TestSessionCreation(t *testing.T) {
    namespace := os.Getenv("TEST_NAMESPACE")

    // Create AgenticSession CR
    session := createTestSession(namespace)

    // Verify CR exists
    obj, err := getDynamicClient().Get(ctx, session.Name, namespace)
    assert.NoError(t, err)

    // Cleanup (if enabled)
    if os.Getenv("CLEANUP_RESOURCES") == "true" {
        deleteSession(namespace, session.Name)
    }
}
```

### Frontend Test Gaps

**Current State**: No unit or component tests exist

**Configured But Unused**:
- Jest framework configured
- Testing Library installed
- No `.test.tsx` or `.spec.tsx` files

**Recommended Test Structure**:
```
components/frontend/src/
├── app/
│   └── projects/
│       ├── [projectName]/
│       │   └── page.test.tsx        # Page component tests
│       └── page.test.tsx             # Projects page tests
├── components/
│   └── ui/
│       └── button.test.tsx           # Component tests
└── services/
    └── queries/
        └── projects.test.ts          # React Query hooks tests
```

---

## Known Gaps & Recommendations

### Critical Gaps (High Priority)

#### 1. Backend/Operator Go Tests Not Enforced in CI ⚠️

**Problem**: Comprehensive test suites exist but aren't run in CI

**Impact**: Breaking changes can merge if they pass linting but fail tests

**Solution**: Add to `go-lint.yml`:
```yaml
- name: Test Backend
  working-directory: components/backend
  run: go test ./...

- name: Test Operator
  working-directory: components/operator
  run: go test ./...
```

**Effort**: Low (15 minutes) | **Value**: High

#### 2. No Frontend Unit Tests ❌

**Problem**: Zero test coverage for NextJS frontend

**Impact**: UI bugs can merge undetected, refactoring is risky

**Solution**:
1. Create example component tests (`Button.test.tsx`)
2. Add React Query hook tests (`useProjects.test.ts`)
3. Add page integration tests (`projects/page.test.tsx`)
4. Add `npm test` to `frontend-lint.yml`

**Effort**: Medium (2-3 hours) | **Value**: High

#### 3. E2E Tests Skip Session Execution ⚠️

**Problem**: E2E tests validate deployment but not actual Claude Code execution

**Impact**: Session creation/execution bugs can slip through

**Solution**:
- Add mock Claude API responses for testing
- Create test session that doesn't require Anthropic API key
- Verify session lifecycle (Pending → Running → Completed)

**Effort**: Medium (3-4 hours) | **Value**: Medium

### Additional Gaps (Medium Priority)

#### 4. No Performance/Load Testing ⚠️

**Problem**: No tests for concurrent sessions, resource limits, timeout handling

**Impact**: Production performance issues unknown until deployment

**Solution**:
- Add k6 or Locust load tests
- Test concurrent session creation (10, 50, 100 sessions)
- Measure operator reconciliation latency
- Test resource limits (CPU, memory, storage)

**Effort**: High (1-2 days) | **Value**: Medium

#### 5. No Security Scanning Workflow ⚠️

**Problem**: No automated vulnerability scanning for dependencies or images

**Impact**: Security vulnerabilities can be introduced unnoticed

**Solution**:
- Add Trivy image scanning to `components-build-deploy.yml`
- Add Dependabot security updates (already configured)
- Add `govulncheck` for Go vulnerabilities
- Add `npm audit` to frontend-lint.yml

**Effort**: Low (30 minutes) | **Value**: High

#### 6. Operator Tests Not Isolated ⚠️

**Problem**: Operator tests require manual setup, not automated in CI

**Impact**: Operator changes can break reconciliation logic undetected

**Solution**:
- Create dedicated `operator-tests.yml` workflow
- Use envtest for isolated controller testing
- Test watch loop reconnection, status updates, Job creation

**Effort**: Medium (2-3 hours) | **Value**: Medium

### Quick Wins (Low Effort, High Value)

1. **Enforce Backend Go Tests** (15 min) - Add `go test` to go-lint.yml
2. **Add Security Scanning** (30 min) - Trivy + govulncheck workflows
3. **Document Test Conventions** (30 min) - Add to CLAUDE.md
4. **Create Frontend Test Examples** (1 hour) - 2-3 example component tests
5. **Add Test Coverage Badges** (15 min) - Codecov badges in README

---

## Testing Best Practices

### Before Committing Code

**Backend/Operator**:
```bash
cd components/backend  # or components/operator
gofmt -w .                    # Auto-format
go vet ./...                  # Check for issues
golangci-lint run             # Comprehensive linting
go test ./...                 # Run all tests
```

**Frontend**:
```bash
cd components/frontend
npm run lint:fix              # Auto-fix ESLint issues
npm run type-check            # Validate TypeScript
npm run build                 # Ensure build succeeds
# npm test                    # Run tests (when added)
```

**Claude Runner**:
```bash
cd components/runners/claude-code-runner
black .                       # Auto-format Python
flake8 .                      # Lint Python
pytest                        # Run tests
```

### Before Opening a PR

1. ✅ Run all local tests for modified components
2. ✅ Fix all linting errors
3. ✅ Ensure builds succeed locally
4. ✅ Run E2E tests if changing core functionality: `make e2e-test`
5. ✅ Update tests if changing behavior
6. ✅ Add tests for new features

### Debugging Test Failures

**E2E Failures**:
1. Check GitHub Actions artifacts (screenshots, videos)
2. Review component logs in "Debug logs on failure" step
3. Run locally: `make e2e-test` then `cd e2e && npm run test:headed`
4. Check Kind cluster state: `kubectl get pods -n ambient-code`

**Go Test Failures**:
1. Run with verbose output: `go test -v ./...`
2. Run specific test: `go test -v -run TestName`
3. Check test logs for error details
4. Verify Kubernetes cluster access (integration tests)

**Frontend Build Failures**:
1. Check TypeScript errors: `npm run type-check`
2. Check ESLint errors: `npm run lint`
3. Clear cache: `rm -rf .next && npm run build`

---

## References

### Workflow Files
- [`.github/workflows/e2e.yml`](https://github.com/ambient-code/platform/blob/main/.github/workflows/e2e.yml) - E2E test orchestration
- [`.github/workflows/go-lint.yml`](https://github.com/ambient-code/platform/blob/main/.github/workflows/go-lint.yml) - Go linting
- [`.github/workflows/frontend-lint.yml`](https://github.com/ambient-code/platform/blob/main/.github/workflows/frontend-lint.yml) - Frontend quality
- [`.github/workflows/runner-tests.yml`](https://github.com/ambient-code/platform/blob/main/.github/workflows/runner-tests.yml) - Runner tests
- [`.github/workflows/test-local-dev.yml`](https://github.com/ambient-code/platform/blob/main/.github/workflows/test-local-dev.yml) - Local dev tests

### Test Files
- [`e2e/cypress/e2e/vteam.cy.ts`](https://github.com/ambient-code/platform/blob/main/e2e/cypress/e2e/vteam.cy.ts) - E2E test suite
- [`components/backend/tests/`](https://github.com/ambient-code/platform/tree/main/components/backend/tests) - Backend tests
- [`components/operator/internal/handlers/sessions_test.go`](https://github.com/ambient-code/platform/blob/main/components/operator/internal/handlers/sessions_test.go) - Operator tests
- [`components/runners/claude-code-runner/tests/`](https://github.com/ambient-code/platform/tree/main/components/runners/claude-code-runner/tests) - Runner tests

### Documentation
- [`e2e/README.md`](https://github.com/ambient-code/platform/blob/main/e2e/README.md) - E2E testing guide
- [`docs/testing/e2e-guide.md`](e2e-guide.md) - Comprehensive E2E documentation
- [`CLAUDE.md`](https://github.com/ambient-code/platform/blob/main/CLAUDE.md) - Project standards (includes testing section)
- [`components/backend/README.md`](https://github.com/ambient-code/platform/blob/main/components/backend/README.md) - Backend testing commands
- [`components/frontend/README.md`](https://github.com/ambient-code/platform/blob/main/components/frontend/README.md) - Frontend development guide

---

## Summary

**Testing Infrastructure Maturity**: 🟡 **Moderate** (some gaps, solid foundation)

**Strengths**:
- ✅ Comprehensive E2E testing in real Kubernetes environment
- ✅ All linting enforced in CI (Go, TypeScript)
- ✅ Change detection optimizes CI performance
- ✅ Good artifact collection on failures
- ✅ Codecov integration for runner tests
- ✅ Clear local test execution patterns

**Weaknesses**:
- ❌ Backend/operator Go tests exist but not enforced in CI
- ❌ Zero frontend unit tests
- ❌ E2E tests skip actual session execution
- ❌ No performance or load testing
- ❌ No security scanning workflow

**Immediate Action Items**:
1. **Add `go test` to CI** (15 min, high value)
2. **Add security scanning** (30 min, high value)
3. **Create frontend test examples** (1 hour, high value)
4. **Document test conventions** (30 min, medium value)

**Contact**: See [`CLAUDE.md`](https://github.com/ambient-code/platform/blob/main/CLAUDE.md) for development standards and [`docs/testing/e2e-guide.md`](e2e-guide.md) for detailed E2E testing guide.
