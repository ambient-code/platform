# E2E Testing Suite

Automated end-to-end testing for the Ambient Code Platform using Cypress. Tests can run against **any deployed instance** — kind, CRC, dev cluster, or production.

> **Status**: ✅ Production Ready | **Tests**: 12 | **Runtime**: ~10 seconds | **CI**: Automated on PRs

## Quick Start

### Test Against Kind (Local)

```bash
make kind-up          # Start local cluster
make test-e2e         # Run tests
make kind-down        # Cleanup
```

**Iterative testing:**
```bash
make kind-up
# Edit e2e/.env to override images
make kind-down && make kind-up
make test-e2e
```

### Test Against External Cluster

```bash
# Set environment
export CYPRESS_BASE_URL=https://ambient-code.apps.your-cluster.com
export TEST_TOKEN=$(oc whoami -t)  # or kubectl get secret...

# Run tests
cd e2e && npm test
```

## Test Suites

### **vteam.cy.ts** - Platform Smoke Tests (5 tests)

> Note: Filename uses "vteam" prefix for backward compatibility with existing CI/CD workflows.

Core platform functionality:
1. Authentication with token
2. Workspace creation dialog
3. Create new workspace
4. List workspaces
5. Backend API connectivity (`/api/cluster-info`)

**Runtime:** ~2 seconds

---

### **sessions.cy.ts** - Session Management (7 tests)

Complete session user journey (reuses one workspace across all tests):

1. **Workspace & Session Creation** - Creates workspace, waits for namespace, creates session
2. **Session Page UI** - All accordions, status badge, breadcrumbs
3. **Workflow Cards & Selection** - Display cards, links, interactions
4. **Workflow Interactions** - Click card, view all, load workflow
5. **Chat Interface** - Welcome message, chat availability
6. **Breadcrumb Navigation** - Navigate back to workspace
7. **Complete Lifecycle** (requires API key configured via UI):
   - Wait for session Running
   - Send "Hello!" and get REAL Claude response
   - Select workflow and verify acknowledgement
   - Check auto-generated session name

**Runtime:** ~10 seconds (test 7 skipped without API key configuration)

**Note on Agent Testing:**  
Test 7 requires `ANTHROPIC_API_KEY` to be configured in the project via the UI (**Project Settings → API Keys**). Simply having the key in `e2e/.env` isn't sufficient — the backend must create `ambient-runner-secrets` in the project namespace via the proper API flow.

---

## Prerequisites

### Required Software

- **Node.js 20+**: For Cypress
  - Install: `brew install node`
- **kubectl**: For Kubernetes clusters
- **oc CLI**: For OpenShift clusters (optional)

### For Kind Local Development

See [Kind Local Development Guide](../docs/developer/local-development/kind.md) for kind-specific setup.

### Install Test Dependencies

```bash
make test-e2e-setup
# or
cd e2e && npm install
```

## Running Tests

### Option 1: Against Kind (Automated)

```bash
# Full automated flow
make test-e2e-local

# Or step-by-step
make kind-up
make test-e2e
make kind-down
```

### Option 2: Against External Cluster

```bash
cd e2e

# Set config
export CYPRESS_BASE_URL=https://your-frontend.com
export TEST_TOKEN=$(oc whoami -t)  # or your auth token

# Run tests
npm test
```

### Option 3: Headed Mode (With UI)

```bash
cd e2e

# Set config (or source .env.test from kind-up)
export CYPRESS_BASE_URL=http://localhost:8080
export TEST_TOKEN=your-token-here

# Open Cypress UI
npm run test:headed
```

---

## Configuration

### Environment Variables

**Required:**
- `CYPRESS_BASE_URL`: Frontend URL (e.g., `http://localhost:8080`)
- `TEST_TOKEN`: Bearer token for API authentication
- `ANTHROPIC_API_KEY`: Claude API key (required for agent session test)

**Optional:**
- `KEEP_WORKSPACES`: Set to `true` to keep test workspaces after run (debugging)

### For Kind (Local Docker/Podman)

`make kind-up` automatically creates `.env.test`:

```bash
TEST_TOKEN=eyJhbGc...
CYPRESS_BASE_URL=http://localhost:8080
```

Tests auto-load this file. Agent test requires `ANTHROPIC_API_KEY` in `e2e/.env`.

### For External Cluster

Create `.env.test` manually or use env vars:

```bash
# Get token from OpenShift
export TEST_TOKEN=$(oc whoami -t)
export CYPRESS_BASE_URL=https://ambient-code.apps.cluster.com

# Run
cd e2e && npm test
```

---

## Test Organization

### Shared Workspace Strategy

All tests in `sessions.cy.ts` reuse **one workspace and one session**:
- Created in `before()` hook
- Shared across tests 1-6
- Cleaned up in `after()` hook (unless `KEEP_WORKSPACES=true`)
- Test 7 creates its own session (needs Running state)

**Benefits:**
- ✅ Faster (no repeated setup)
- ✅ Tests real user flow
- ✅ Reduced cluster load

### Test Independence

Tests can run in any order within their suite.

---

## Debugging

### View Test Results

```bash
# Screenshots (on failure)
ls cypress/screenshots/

# Videos (always captured)
open cypress/videos/sessions.cy.ts.mp4
```

### Run Single Test

```bash
source .env.test
CYPRESS_TEST_TOKEN="$TEST_TOKEN" npx cypress run --spec "cypress/e2e/vteam.cy.ts"
```

### Debug with UI

```bash
source .env.test
npm run test:headed
# Click on test file to run interactively
```

### Check Cluster State

```bash
# Kind
kubectl get pods -n ambient-code
kubectl logs -n ambient-code deployment/backend-api

# OpenShift
oc get pods -n ambient-code
oc logs -n ambient-code deployment/backend-api
```

---

## Writing New Tests

### Add to Existing Suite

Edit `cypress/e2e/sessions.cy.ts` or `vteam.cy.ts`:

```typescript
it('should test new feature', () => {
  cy.visit('/your-page')
  cy.contains('Expected Content').should('be.visible')
  cy.get('[data-testid="button"]').click()
  cy.url().should('include', '/expected-url')
})
```

### Testing Guidelines

- ✅ Test user journeys, not isolated UI elements
- ✅ Use `data-testid` selectors when possible
- ✅ Wait for conditions, not fixed timeouts
- ✅ Use descriptive test names
- ❌ Don't test implementation details
- ❌ Don't rely on test execution order
- ❌ Don't manually add auth headers (auto-injected)

See [E2E Testing Guide](../docs/testing/e2e-guide.md) for detailed patterns.

---

## CI Integration

GitHub Actions runs tests automatically:
- **Trigger**: All PRs to main
- **Workflow**: `.github/workflows/e2e.yml`
- **Environment**: kind with Docker
- **Runtime**: ~6-7 minutes (includes cluster setup)
- **Artifacts**: Screenshots/videos uploaded on failure

---

## Performance

| Phase | Time | Notes |
|-------|------|-------|
| Cluster setup | ~2 min | kind creation + ingress |
| Deployment | ~2-3 min | Pull images, start pods |
| MinIO init | ~5 sec | Create bucket |
| Test execution | ~10 sec | All 12 tests |
| **Total** | **~5 min** | With Quay images |

---

## Maintenance

### Before Merging PR

- [ ] All tests passing locally
- [ ] Tests passing in CI
- [ ] No new Cypress errors
- [ ] Screenshots/videos reviewed

### After Frontend Changes

- [ ] Update selectors if UI structure changed
- [ ] Update expected text if copy changed
- [ ] Run with UI to verify: `npm run test:headed`

### After Backend Changes

- [ ] Update API assertions if response format changed
- [ ] Update auth if token format changed

---

## Migration from Old E2E Setup

**Old commands** → **New commands**:
- `make e2e-test` → `make test-e2e-local` (still works as alias)
- `make e2e-clean` → `make kind-down` (still works as alias)
- `make e2e-setup` → `make test-e2e-setup` (still works as alias)

**Old overlay** → **New overlay**:
- `overlays/e2e/` → `overlays/kind/` (Quay images)
- New: `overlays/kind-local/` (local images)

**Old cluster name** → **New cluster name**:
- `vteam-e2e` → `ambient-local`

---

## See Also

- [Kind Local Development](../docs/developer/local-development/kind.md) - Using kind for development
- [E2E Testing Guide](../docs/testing/e2e-guide.md) - Writing e2e tests
- [Testing Strategy](../CLAUDE.md#testing-strategy) - Testing overview
- [Cypress Documentation](https://docs.cypress.io/)
