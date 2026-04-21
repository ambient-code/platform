/**
 * Documentation Screenshot Capture
 *
 * Manifest-driven: reads targets from manifest.json, captures each in
 * both light and dark themes.
 *
 * Run: npx cypress run --browser chrome --spec cypress/e2e/screenshots.cy.ts
 */
import manifest from '../screenshots/manifest.json'

interface ScreenshotEntry {
  id: string
  page: string
  waitFor: string
  setupSteps: string[]
}

describe('Documentation Screenshots', () => {
  let workspaceSlug: string
  let sessionId: string

  Cypress.on('uncaught:exception', (err) => {
    if (
      err.message.includes('Minified React error #418') ||
      err.message.includes('Minified React error #423') ||
      err.message.includes('Hydration')
    ) {
      return false
    }
    return true
  })

  before(() => {
    const token = Cypress.env('TEST_TOKEN')
    expect(token, 'TEST_TOKEN must be set').to.exist

    const name = `docs-screenshots-${Date.now()}`
    cy.request({
      method: 'POST',
      url: '/api/projects',
      headers: { Authorization: `Bearer ${token}` },
      body: { name, displayName: name },
    }).then((resp) => {
      expect(resp.status).to.be.oneOf([200, 201])
      workspaceSlug = resp.body.name || name

      const poll = (attempt: number): void => {
        if (attempt > 30) throw new Error('Workspace namespace timeout')
        cy.request({
          url: `/api/projects/${workspaceSlug}`,
          headers: { Authorization: `Bearer ${token}` },
          failOnStatusCode: false,
        }).then((r) => {
          if (r.status !== 200) {
            cy.wait(1500, { log: false })
            poll(attempt + 1)
          }
        })
      }
      poll(1)

      cy.request({
        method: 'PUT',
        url: `/api/projects/${workspaceSlug}/runner-secrets`,
        headers: { Authorization: `Bearer ${token}` },
        body: { data: { ANTHROPIC_API_KEY: 'mock-replay-key' } },
      }).then((r) => expect(r.status).to.eq(200))

      cy.request({
        method: 'POST',
        url: `/api/projects/${workspaceSlug}/agentic-sessions`,
        headers: { Authorization: `Bearer ${token}` },
        body: { initialPrompt: '' },
      }).then((r) => {
        expect(r.status).to.eq(201)
        sessionId = r.body.name
      })
    })
  })

  after(() => {
    if (workspaceSlug && !Cypress.env('KEEP_WORKSPACES')) {
      cy.request({
        method: 'DELETE',
        url: `/api/projects/${workspaceSlug}`,
        headers: { Authorization: `Bearer ${Cypress.env('TEST_TOKEN')}` },
        failOnStatusCode: false,
      })
    }
  })

  beforeEach(() => {
    cy.viewport(manifest.viewport.width, manifest.viewport.height)
  })

  for (const entry of manifest.screenshots as ScreenshotEntry[]) {
    it(`captures ${entry.id} (light + dark)`, () => {
      const url = entry.page
        .replace('{workspace}', workspaceSlug)
        .replace('{session}', sessionId)

      cy.visit(url)
      if (entry.waitFor) {
        cy.contains(entry.waitFor, { timeout: 15000 }).should('be.visible')
      } else {
        cy.get('body', { timeout: 10000 }).should('not.be.empty')
      }

      for (const step of entry.setupSteps) {
        runSetupStep(step)
      }

      setTheme('light')
      waitForFonts()
      cy.screenshot(`${entry.id}-light`, { overwrite: true, capture: 'viewport' })

      setTheme('dark')
      waitForFonts()
      cy.screenshot(`${entry.id}-dark`, { overwrite: true, capture: 'viewport' })
    })
  }
})

function setTheme(theme: 'light' | 'dark'): void {
  const label = theme === 'dark' ? 'Switch to dark theme' : 'Switch to light theme'
  cy.get('button[aria-label="Toggle theme"]').first().click({ force: true })
  // 10 s timeout: slow CI environments can take > 5 s for Radix to mount the dropdown content
  cy.get(`[aria-label="${label}"]`, { timeout: 10000 }).first().click({ force: true })
  if (theme === 'dark') {
    cy.get('html').should('have.class', 'dark')
  } else {
    cy.get('html').should('not.have.class', 'dark')
  }
}

function waitForFonts(): void {
  cy.document().then((doc) => cy.wrap((doc as any).fonts?.ready))
  cy.wait(200)
}

function runSetupStep(step: string): void {
  switch (step) {
    case 'navigateToIntegrations':
      cy.contains('Integrations', { timeout: 5000 }).click()
      cy.wait(500)
      break
    case 'navigateToSharing':
      cy.contains('Sharing', { timeout: 5000 }).click()
      cy.wait(500)
      break
    case 'waitForThemeToggle':
      cy.get('button[aria-label="Toggle theme"]', { timeout: 10000 }).should('be.visible')
      cy.wait(500)
      break
    default:
      throw new Error(`Unknown setup step: ${step}`)
  }
}
