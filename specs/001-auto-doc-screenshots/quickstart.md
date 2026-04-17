# Quickstart: Documentation Screenshots

## Capture screenshots locally

```bash
# 1. Start a kind cluster (if not already running)
make kind-up CONTAINER_ENGINE=docker

# 2. Capture all screenshots (light + dark)
make screenshots

# 3. Preview in docs dev server
cd docs && npm run dev
# Visit http://localhost:4321/platform/getting-started/quickstart-ui/
# Toggle theme to verify switching works

# 4. Clean up
make kind-down
```

## Add a new screenshot

1. Add entry to `e2e/cypress/screenshots/manifest.json`:
```json
{
  "id": "my-new-page",
  "page": "/projects/{workspace}/my-page",
  "waitFor": "Page Title",
  "setupSteps": []
}
```

2. Add HTML to the target docs page:
```html
<figure class="screenshot-pair">
  <img class="screenshot-light" src="/platform/images/screenshots/my-new-page-light.png" alt="Description" />
  <img class="screenshot-dark" src="/platform/images/screenshots/my-new-page-dark.png" alt="Description" />
</figure>
```

3. Run `make screenshots` to capture.

## Add a setup step

If a screenshot needs navigation before capture (e.g., clicking a tab), add a named step:

1. Add `"setupSteps": ["myStep"]` to the manifest entry
2. Add a case to `runSetupStep()` in `e2e/cypress/e2e/screenshots.cy.ts`

## CI workflow

The `screenshots.yml` workflow runs daily at 6AM UTC. It:
1. Pulls latest images from Quay.io
2. Spins up a kind cluster
3. Runs the screenshot Cypress spec
4. Opens a PR if any images changed

Trigger manually: Actions > "Update Documentation Screenshots" > "Run workflow"
