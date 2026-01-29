# Testing Documentation

Comprehensive testing documentation for the Ambient Code Platform.

## 🧪 Test Types

### End-to-End (E2E) Tests
**Location:** `e2e/`  
**Framework:** Cypress  
**Environment:** Kind cluster (Kubernetes in Docker)

**Purpose:** Test complete user journeys against a deployed platform instance.

**Quick Start:**
```bash
make kind-up
make test-e2e
make kind-down
```

**Documentation:**
- [E2E Testing README](../../e2e/README.md) - Complete guide
- [E2E Testing Guide](e2e-guide.md) - Writing tests
- [Kind Local Dev](../developer/local-development/kind.md) - Environment setup

**Test Suites:**
- `vteam.cy.ts` (5 tests) - Platform smoke tests
- `sessions.cy.ts` (7 tests) - Session management
- **Runtime:** ~15 seconds total

---

### Backend Tests (Go)
**Location:** `components/backend/tests/`

**Test Types:**
- **Unit Tests** - Component logic in isolation
- **Contract Tests** - API contract validation
- **Integration Tests** - End-to-end with real Kubernetes cluster

**Quick Start:**
```bash
cd components/backend
make test              # All tests
make test-unit         # Unit only
make test-contract     # Contract only
make test-integration  # Integration (requires cluster)
```

**Documentation:** [Backend Test Guide](../../components/backend/TEST_GUIDE.md)

---

### Frontend Tests (Next.js)
**Location:** `components/frontend/`

**Test Types:**
- **Component Tests** - React component testing (Jest)
- **E2E Tests** - User interface testing (Cypress)

**Quick Start:**
```bash
cd components/frontend
npm test
npm run lint
npm run build  # Must pass with 0 errors, 0 warnings
```

**Documentation:** [Frontend README](../../components/frontend/README.md)

---

### Operator Tests (Go)
**Location:** `components/operator/`

**Test Types:**
- Controller reconciliation tests
- CRD validation tests
- Watch loop tests

**Quick Start:**
```bash
cd components/operator
go test ./... -v
```

**Documentation:** [Operator README](../../components/operator/README.md)

---

## 🎯 Testing Strategy

### Development Workflow

**Local Development:**
1. Run unit tests during development
2. Run contract tests before commit
3. Run integration tests before PR

**Pull Request:**
1. All tests run automatically in CI
2. E2E tests run in Kind cluster
3. Linting and formatting checks

**Before Merge:**
- ✅ All tests passing
- ✅ Linting clean
- ✅ Code reviewed

### Test Environments

| Environment | Purpose | Setup |
|-------------|---------|-------|
| **Unit** | Fast feedback | Local machine |
| **Contract** | API validation | Local machine |
| **Integration** | K8s integration | Kind or test cluster |
| **E2E** | Full system | Kind cluster |

### CI/CD Testing

**GitHub Actions Workflows:**
- `e2e.yml` - E2E tests in Kind on every PR
- `go-lint.yml` - Go code quality checks
- `frontend-lint.yml` - Frontend quality checks
- `test-local-dev.yml` - Local dev environment validation

## 🔧 Running Tests Locally

### Quick Commands

```bash
# All E2E tests
make test-e2e-local

# Backend tests
cd components/backend && make test

# Frontend tests
cd components/frontend && npm test

# Operator tests
cd components/operator && go test ./...

# Run linters
make lint
```

### Test Against Different Environments

**Kind (Local):**
```bash
make kind-up
make test-e2e
```

**External Cluster:**
```bash
export CYPRESS_BASE_URL=https://your-frontend.com
export TEST_TOKEN=$(oc whoami -t)
cd e2e && npm test
```

## 📊 Test Coverage

### Current Coverage
- **Backend:** Check with `make test-coverage` in backend directory
- **Frontend:** Check with `npm run coverage` (if configured)
- **E2E:** 12 tests covering critical user journeys

### Coverage Goals
- **Backend:** Aim for 60%+ coverage
- **Critical paths:** 80%+ coverage
- **New features:** Must include tests

## 🐛 Debugging Tests

### E2E Test Debugging
```bash
cd e2e
npm run test:headed  # Opens Cypress UI
```

### Backend Test Debugging
```bash
cd components/backend
go test ./... -v -run TestSpecificTest
```

### View Test Logs
```bash
# E2E test logs
cat e2e/cypress/videos/*.mp4  # Test recordings
cat e2e/cypress/screenshots/*.png  # Failure screenshots

# Backend test logs
cd components/backend && go test ./... -v 2>&1 | tee test.log
```

## 📝 Writing Tests

### Best Practices

**E2E Tests:**
- Test user journeys, not isolated elements
- Reuse workspaces across tests
- Use meaningful test descriptions
- Keep tests fast (<30 seconds each)

**Backend Tests:**
- Use table-driven tests (Go convention)
- Mock external dependencies
- Test error cases
- Follow patterns in existing tests

**Frontend Tests:**
- Test component behavior, not implementation
- Mock API calls
- Test accessibility
- Test error states

### Test Templates

See existing tests for patterns:
- `e2e/cypress/e2e/vteam.cy.ts` - E2E test patterns
- `components/backend/handlers/*_test.go` - Backend test patterns

## 🆘 Troubleshooting

### E2E Tests Failing
- Check Kind cluster is running: `kubectl get pods -n ambient-code`
- Verify frontend is accessible: `curl http://localhost:8080`
- Check test logs in `e2e/cypress/videos/`

### Integration Tests Failing
- Check cluster connection: `kubectl cluster-info`
- Verify namespace exists: `kubectl get ns ambient-code`
- Check permissions: `kubectl auth can-i create jobs -n ambient-code`

### CI Tests Failing but Local Passes
- Environment differences (check GitHub Actions logs)
- Timeout issues (CI may be slower)
- Resource constraints (CI has memory limits)

---

**Related Documentation:**
- [Developer Guide](../developer/)
- [Contributing Guidelines](../../CONTRIBUTING.md)
- [E2E Testing Full Guide](../../e2e/README.md)
