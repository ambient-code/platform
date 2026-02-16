---
description: Create a Cypress-based video demo for a feature branch with cursor, click effects, and captions.
---

# /cypress-demo Command

Create a polished Cypress demo test that records a human-paced video walkthrough of UI features on the current branch.

## Usage

```
/cypress-demo                          # Auto-detect features from branch diff
/cypress-demo chat input refactoring   # Describe what to demo
```

## User Input

```text
$ARGUMENTS
```

## Behavior

When invoked, Claude will create a Cypress test file in `e2e/cypress/e2e/` that records a demo video with:

- **Synthetic cursor** (white dot) that glides smoothly to each interaction target
- **Click ripple** (blue expanding ring) on every click action
- **Caption bar** (compact dark bar at top of viewport) describing each step
- **Human-paced timing** so every action is clearly visible
- **`--no-runner-ui`** flag to exclude the Cypress sidebar from the recording

### 1. Determine what to demo

- If `$ARGUMENTS` is provided, use it as the demo description
- If empty, run `git diff main..HEAD --stat` to identify changed files and infer features
- Read the changed/new component files to understand what UI to showcase
- Ask the user if clarification is needed on which features to highlight

### 2. Check prerequisites

- Verify `e2e/.env.test` or `e2e/.env` exists with `TEST_TOKEN`
- Check if `ANTHROPIC_API_KEY` is available (needed if the demo requires Running state for workflows, agents, or commands)
- Verify the kind cluster is up: `kubectl get pods -n ambient-code`
- Verify the frontend is accessible: `curl -s -o /dev/null -w "%{http_code}" http://localhost`
- If the frontend was rebuilt from this branch, verify imagePullPolicy is `Never` or `IfNotPresent`

### 3. Create the demo test file

Create `e2e/cypress/e2e/<feature-name>-demo.cy.ts` using the template structure below.

#### Required helpers (copy into every demo file)

```typescript
// Timing constants — adjust per demo, aim for ~2 min total video
const LONG = 3200    // hold on important visuals
const PAUSE = 2400   // standard pause between actions
const SHORT = 1600   // brief pause after small actions
const TYPE_DELAY = 80 // ms per keystroke

// Target first element (session page renders desktop + mobile layout)
const chatInput = () => cy.get('textarea[placeholder*="message"]').first()

// Caption: compact bar at TOP of viewport
function caption(text: string) {
  cy.document().then((doc) => {
    let el = doc.getElementById('demo-caption')
    if (!el) {
      el = doc.createElement('div')
      el.id = 'demo-caption'
      el.style.cssText = [
        'position:fixed', 'top:0', 'left:0', 'right:0', 'z-index:99998',
        'background:rgba(0,0,0,0.80)', 'color:#fff', 'font-size:14px',
        'font-weight:500', 'font-family:system-ui,-apple-system,sans-serif',
        'padding:6px 20px', 'text-align:center', 'letter-spacing:0.2px',
        'pointer-events:none', 'transition:opacity 0.4s ease',
      ].join(';')
      doc.body.appendChild(el)
    }
    el.textContent = text
    el.style.opacity = '1'
  })
}

function clearCaption() {
  cy.document().then((doc) => {
    const el = doc.getElementById('demo-caption')
    if (el) el.style.opacity = '0'
  })
}

// Synthetic cursor + click ripple
function initCursor() {
  cy.document().then((doc) => {
    if (doc.getElementById('demo-cursor')) return
    const cursor = doc.createElement('div')
    cursor.id = 'demo-cursor'
    cursor.style.cssText = [
      'position:fixed', 'z-index:99999', 'pointer-events:none',
      'width:20px', 'height:20px', 'border-radius:50%',
      'background:rgba(255,255,255,0.9)', 'border:2px solid #333',
      'box-shadow:0 0 6px rgba(0,0,0,0.4)',
      'transform:translate(-50%,-50%)',
      'transition:left 0.5s cubic-bezier(0.25,0.1,0.25,1), top 0.5s cubic-bezier(0.25,0.1,0.25,1)',
      'left:-40px', 'top:-40px',
    ].join(';')
    doc.body.appendChild(cursor)
    const ripple = doc.createElement('div')
    ripple.id = 'demo-ripple'
    ripple.style.cssText = [
      'position:fixed', 'z-index:99999', 'pointer-events:none',
      'width:40px', 'height:40px', 'border-radius:50%',
      'border:3px solid rgba(59,130,246,0.8)',
      'transform:translate(-50%,-50%) scale(0)',
      'opacity:0', 'left:-40px', 'top:-40px',
    ].join(';')
    doc.body.appendChild(ripple)
    const style = doc.createElement('style')
    style.textContent = `
      @keyframes demo-ripple-anim {
        0%   { transform: translate(-50%,-50%) scale(0); opacity: 1; }
        100% { transform: translate(-50%,-50%) scale(2.5); opacity: 0; }
      }
    `
    doc.head.appendChild(style)
  })
}

// Move cursor smoothly to element center
function moveTo(selector: string, options?: { first?: boolean }) {
  const chain = options?.first ? cy.get(selector).first() : cy.get(selector)
  chain.then(($el) => {
    const rect = $el[0].getBoundingClientRect()
    cy.document().then((doc) => {
      const cursor = doc.getElementById('demo-cursor')
      if (cursor) {
        cursor.style.left = `${rect.left + rect.width / 2}px`
        cursor.style.top = `${rect.top + rect.height / 2}px`
      }
    })
    cy.wait(600)
  })
}

function moveToText(text: string, tag?: string) {
  const chain = tag ? cy.contains(tag, text) : cy.contains(text)
  chain.then(($el) => {
    const rect = $el[0].getBoundingClientRect()
    cy.document().then((doc) => {
      const cursor = doc.getElementById('demo-cursor')
      if (cursor) {
        cursor.style.left = `${rect.left + rect.width / 2}px`
        cursor.style.top = `${rect.top + rect.height / 2}px`
      }
    })
    cy.wait(600)
  })
}

function moveToEl($el: JQuery<HTMLElement>) {
  const rect = $el[0].getBoundingClientRect()
  cy.document().then((doc) => {
    const cursor = doc.getElementById('demo-cursor')
    if (cursor) {
      cursor.style.left = `${rect.left + rect.width / 2}px`
      cursor.style.top = `${rect.top + rect.height / 2}px`
    }
  })
  cy.wait(600)
}

function clickEffect() {
  cy.document().then((doc) => {
    const cursor = doc.getElementById('demo-cursor')
    const ripple = doc.getElementById('demo-ripple')
    if (cursor && ripple) {
      ripple.style.left = cursor.style.left
      ripple.style.top = cursor.style.top
      ripple.style.animation = 'none'
      void ripple.offsetHeight
      ripple.style.animation = 'demo-ripple-anim 0.5s ease-out forwards'
    }
  })
}

// Compound: move → ripple → click
function cursorClickText(text: string, tag?: string, options?: { force?: boolean }) {
  moveToText(text, tag)
  clickEffect()
  const chain = tag ? cy.contains(tag, text) : cy.contains(text)
  chain.click({ force: options?.force })
}
```

#### Test structure

```typescript
describe('<Feature> Demo', () => {
  const workspaceName = `demo-${Date.now()}`

  // ... helpers above ...

  Cypress.on('uncaught:exception', (err) => {
    if (err.message.includes('Minified React error') || err.message.includes('Hydration')) {
      return false
    }
    return true
  })

  after(() => {
    if (!Cypress.env('KEEP_WORKSPACES')) {
      const token = Cypress.env('TEST_TOKEN')
      cy.request({
        method: 'DELETE',
        url: `/api/projects/${workspaceName}`,
        headers: { Authorization: `Bearer ${token}` },
        failOnStatusCode: false,
      })
    }
  })

  it('demonstrates <feature>', () => {
    // ... single continuous test for one video file ...
  })
})
```

### 4. Key patterns to follow

| Pattern | Rule |
|---------|------|
| **Dual layout** | Session page renders desktop + mobile. Always use `.first()` on element queries that match both |
| **Caption scoping** | When asserting page content with `cy.contains`, scope to a tag (e.g., `cy.contains('p', 'text')`) to avoid matching the caption overlay |
| **Workspace setup** | Create workspace → poll `/api/projects/:name` until 200 → configure runner-secrets if API key needed |
| **Running state** | If demo needs agents/commands, configure `ANTHROPIC_API_KEY` via runner-secrets, select a workflow, and wait for `textarea[placeholder*="attach"]` (Running placeholder) with 180s timeout |
| **Operator pull policy** | For kind clusters, set `IMAGE_PULL_POLICY=IfNotPresent` on the operator to avoid re-pulling the 879MB runner image every session |
| **File attachment** | Use `cy.get('input[type="file"]').first().selectFile({...}, { force: true })` with a `Cypress.Buffer` — no real file needed |
| **Caption position** | Always `top:0` — bottom position obscures the chat toolbar |
| **Timing** | Aim for ~2 min total. LONG=3.2s, PAUSE=2.4s, SHORT=1.6s, TYPE_DELAY=80ms. Adjust if video feels too fast or slow |
| **Video output** | `e2e/cypress/videos/<name>.cy.ts.mp4` at 2560x1440 (Retina) |

### 5. Run the demo

```bash
cd e2e
npx cypress run --no-runner-ui --spec "cypress/e2e/<name>-demo.cy.ts"
```

- Verify the video plays at human-readable speed
- Check that captions don't overlap important UI elements
- Re-run and iterate if needed — adjust timing or add/remove steps

### 6. Commit and push

- Commit the demo test file and any config changes (`cypress.config.ts`)
- Push to the current branch
- If a PR exists, note the demo in the PR description

## Reference implementation

See `e2e/cypress/e2e/chatbox-demo.cy.ts` for a complete working example that demonstrates:
- Workspace creation, session creation
- WelcomeExperience (streaming text, workflow cards)
- Workflow selection ("Fix a bug") with Running state wait
- File attachments (AttachmentPreview)
- Autocomplete popovers (@agents, /commands) with real workflow data
- Message queueing (QueuedMessageBubble)
- Message history and queued message editing
- Settings dropdown
- Breadcrumb navigation

## Config requirements

`e2e/cypress.config.ts` must load `.env.test` and wire `TEST_TOKEN`:

```typescript
// Load env files: .env.local > .env > .env.test
const envFiles = ['.env.local', '.env', '.env.test'].map(f => path.resolve(__dirname, f))
for (const envFile of envFiles) {
  if (fs.existsSync(envFile)) { dotenv.config({ path: envFile }) }
}

// In setupNodeEvents:
config.env.TEST_TOKEN = process.env.CYPRESS_TEST_TOKEN || process.env.TEST_TOKEN || config.env.TEST_TOKEN || ''
config.env.ANTHROPIC_API_KEY = process.env.CYPRESS_ANTHROPIC_API_KEY || process.env.ANTHROPIC_API_KEY || ''
```
