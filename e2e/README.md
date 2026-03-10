# E2E Testing Suite

Cypress E2E tests for the Ambient Code Platform. Tests run against a live cluster using a **mock SDK client** — no real Anthropic API key needed.

> **Tests**: 58 | **Runtime**: ~3 min (Chrome) | **Coverage**: Integration confidence

## Quick Start

```bash
# Prerequisites: frontend running (npm run dev), backend port-forwarded
cd e2e && npm install

# Run headless
TEST_TOKEN=$(kubectl get secret test-user-token -n ambient-code \
  -o jsonpath='{.data.token}' | base64 -d) \
CYPRESS_BASE_URL=http://localhost:3000 \
npx cypress run --browser chrome --spec "cypress/e2e/sessions.cy.ts"

# Interactive mode (for debugging)
TEST_TOKEN=$(kubectl get secret test-user-token -n ambient-code \
  -o jsonpath='{.data.token}' | base64 -d) \
CYPRESS_BASE_URL=http://localhost:3000 \
npx cypress open
```

## Mock SDK Client

Tests always use `ANTHROPIC_API_KEY=mock-replay-key`. When the runner pod sees this key, `MockClaudeSDKClient` replays pre-recorded SDK messages from JSONL fixtures through the real `ClaudeAgentAdapter`. This tests the full AG-UI translation pipeline without calling the Anthropic API.

**Fixtures**: `components/runners/ambient-runner/ambient_runner/bridges/claude/fixtures/`

**Capturing new fixtures** (requires real API key):
```bash
cd components/runners/ambient-runner
ANTHROPIC_API_KEY=sk-ant-... uv run --extra claude python scripts/capture-fixtures.py 'your prompt'
```

**Prompt matching**: `hello` → `hello.jsonl`, `comprehensive` → `comprehensive.jsonl`, default → `default.jsonl`

## Test Structure

One file: `cypress/e2e/sessions.cy.ts` — 58 tests across 15 describe blocks:

| Block | Tests | What it covers |
|-------|-------|----------------|
| Workspace & Session Creation | 1 | Create workspace, wait for namespace, create session |
| Session Page UI | 4 | Phase badge, accordions, breadcrumbs, chat area |
| Workspace Page | 3 | Sessions list, admin tabs, create session dialog |
| Projects List | 2 | Workspace list, status badges |
| Agent Interaction | 1 | Send message, verify response, workflow selection |
| Session Header Actions | 1 | Three-dot menu interactions |
| Workspace Admin Tabs | 3 | Settings, sharing, keys tab rendering |
| Session Header Menu Deep | 4 | View details, edit name, clone, export chat |
| Chat Input Features | 3 | Toolbar buttons, autocomplete, history |
| Feedback Buttons | 1 | Thumbs up/down on agent messages |
| Theme & Navigation | 2 | Dark/light/system toggle, nav component |
| Session Page Modals | 6 | Add context, upload, workflow, clone, details, edit name |
| Workspace Admin Form Submissions | 7 | Save settings, create keys, grant permissions, feature flags |
| Chat Input Deep | 3 | Slash commands, Ctrl+Space, agents/commands buttons |
| Welcome Experience | 3 | Workflow cards, view all, search |

## Writing Tests

All tests go in `sessions.cy.ts`. They share one workspace created in `before()`.

```typescript
// Add new tests inside an existing describe block, or create a new one:
describe('My Feature', () => {
  it('should do the thing', () => {
    cy.visit(`/projects/${workspaceSlug}/sessions/${pendingSessionId}`)

    // Use data-testid for reliable selectors
    cy.get('[data-testid="my-button"]').click({ force: true })

    // Conditional checks for optional elements
    cy.get('body').then(($body) => {
      if ($body.find('.optional-element').length) {
        cy.get('.optional-element').click({ force: true })
      }
    })

    // Close modals with Escape
    cy.get('body').type('{esc}')
  })
})
```

**Rules:**
- Use `{ force: true }` on all clicks (elements may be in overflow containers)
- Wrap optional elements in `.then($body => { if ($body.find(...).length) { ... } })`
- Use `data-testid` attributes — add them to frontend components as needed
- Never use real API keys — tests always use `mock-replay-key`
- Use Chrome, not Electron (Electron drops SSE events)

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `TEST_TOKEN` | Yes | K8s service account token for API auth |
| `CYPRESS_BASE_URL` | Yes | Frontend URL (e.g., `http://localhost:3000`) |
| `CYPRESS_OC_TOKEN` | No | OpenShift OAuth token (only for OCP clusters) |
| `KEEP_WORKSPACES` | No | Set to `true` to skip cleanup after tests |

## Debugging

```bash
# Screenshots (on failure)
open cypress/screenshots/

# Videos (always captured)
open cypress/videos/sessions.cy.ts.mp4

# Interactive mode — best for debugging
npx cypress open
```
