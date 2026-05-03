---
name: cypress-demo
description: >
  Create a Cypress-based video demo for a feature branch with cursor, click
  effects, and captions. Use when recording feature demos, creating PR videos,
  showcasing UI changes, or generating visual walkthroughs. Triggers on: "demo",
  "record demo", "create demo video", "cypress demo", "feature walkthrough",
  "PR video", "showcase".
---

# Cypress Demo

Create a polished Cypress demo test that records a human-paced video walkthrough of UI features on the current branch.

## Usage

```bash
/cypress-demo                          # Auto-detect features from branch diff
/cypress-demo chat input refactoring   # Describe what to demo
```

## User Input

```text
$ARGUMENTS
```

## Behavior

When invoked, create a Cypress test file in `e2e/cypress/e2e/` that records a demo video with:

- **Synthetic cursor** (white dot) that glides smoothly to each interaction target
- **Click ripple** (blue expanding ring) on every click action
- **Caption bar** (compact dark bar at top of viewport) describing each step
- **Human-paced timing** so every action is clearly visible
- **`--no-runner-ui`** flag to exclude the Cypress sidebar from the recording

### 1. Determine what to demo

- If `$ARGUMENTS` is provided, use it as the demo description
- If empty, run `git diff main..HEAD --stat` to identify changed files and infer features
- Read the changed/new component files to understand what UI to showcase
- Ask the user if clarification is needed

### 2. Check prerequisites

- Verify `e2e/.env.test` or `e2e/.env` exists with `TEST_TOKEN`
- Check if `ANTHROPIC_API_KEY` is available (needed for Running state)
- Verify the kind cluster is up: `kubectl get pods -n ambient-code`
- Verify the frontend is accessible: `curl -s -o /dev/null -w "%{http_code}" http://localhost:3000`

### 3. Create the demo test file

Create `e2e/cypress/e2e/<feature-name>-demo.cy.ts` using the helpers below.

#### Required helpers

Each demo file must define the following helpers inline (they are not shared utilities — embed them directly in the test file):

- **Functions**: `caption()`, `clearCaption()`, `initCursor()`, `moveTo()`, `moveToText()`, `clickEffect()`, `cursorClickText()`
- **Timing constants**: `LONG`, `PAUSE`, `SHORT`, `TYPE_DELAY`

See the patterns table below for usage. If a prior demo file already exists in `e2e/cypress/e2e/`, copy the helpers from there; otherwise implement them fresh using the patterns below.

### 4. Key patterns

| Pattern | Rule |
|---------|------|
| **Dual layout** | Session page renders desktop + mobile. Always use `.first()` |
| **Caption scoping** | Scope `cy.contains` to a tag to avoid matching caption overlay |
| **Workspace setup** | Create workspace, poll `/api/projects/:name` until 200 |
| **Caption position** | Always `top:0` — bottom obscures chat toolbar |
| **Timing** | Aim for ~2 min total. Adjust constants if too fast/slow |
| **Video output** | `e2e/cypress/videos/<name>.cy.ts.mp4` at 2560x1440 |

### 5. Run the demo

```bash
cd e2e
npx cypress run --no-runner-ui --spec "cypress/e2e/<name>-demo.cy.ts"
```

### 6. Reference implementation

`e2e/cypress/e2e/sessions.cy.ts` is a reference for test structure and Cypress patterns in this repo. Note that sessions.cy.ts is a functional test, not a demo test — it does not contain the cursor/caption/click-ripple helpers listed above. Use it as a guide for selectors, workspace setup, and polling patterns; implement the demo-specific helpers fresh in your new file.
