/**
 * Chatbox Component Demo
 *
 * Records a human-paced walkthrough of the refactored chat UI components:
 *   - WelcomeExperience (quick-start prompt cards)
 *   - ChatInputBox (textarea, toolbar, phase banners)
 *   - QueuedMessageBubble (amber queued messages)
 *   - Settings dropdown (display settings)
 *   - Toolbar buttons (agents, commands, attach)
 *
 * Run:  npx cypress run --spec "cypress/e2e/chatbox-demo.cy.ts"
 * Video: cypress/videos/chatbox-demo.cy.ts.mp4
 */
describe('Chatbox Component Demo', () => {
  const workspaceName = `chatbox-demo-${Date.now()}`
  const PAUSE = 1500  // ms between demo actions
  const TYPE_DELAY = 80  // ms per keystroke

  // Handle React hydration errors gracefully
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

  it('walks through the refactored chat UI components', () => {
    const token = Cypress.env('TEST_TOKEN')
    expect(token, 'TEST_TOKEN should be set').to.exist

    // ── 1. Workspaces page ──────────────────────────────────────────
    cy.visit('/projects')
    cy.contains('Workspaces', { timeout: 15000 }).should('be.visible')
    cy.wait(PAUSE)

    // ── 2. Create workspace ─────────────────────────────────────────
    cy.contains('button', 'New Workspace').click()
    cy.contains('Create New Workspace', { timeout: 10000 }).should('be.visible')
    cy.wait(PAUSE / 2)

    cy.get('#name').clear().type(workspaceName, { delay: 40 })
    cy.wait(PAUSE / 2)

    cy.contains('button', 'Create Workspace').should('not.be.disabled').click({ force: true })
    cy.url({ timeout: 20000 }).should('include', `/projects/${workspaceName}`)

    // Wait for namespace readiness
    const pollProject = (attempt = 1) => {
      if (attempt > 20) throw new Error('Namespace timeout')
      cy.request({
        url: `/api/projects/${workspaceName}`,
        headers: { Authorization: `Bearer ${token}` },
        failOnStatusCode: false,
      }).then((resp) => {
        if (resp.status !== 200) {
          cy.wait(1000, { log: false })
          pollProject(attempt + 1)
        }
      })
    }
    pollProject()
    cy.wait(PAUSE)

    // ── 3. Create session ───────────────────────────────────────────
    cy.contains('button', 'New Session').click()
    cy.contains('button', 'Create').click()
    cy.url({ timeout: 30000 }).should('match', /\/sessions\/[a-z0-9-]+$/)
    cy.wait(PAUSE)

    // ── 4. WelcomeExperience: prompt cards ──────────────────────────
    cy.contains('What would you like to work on?', { timeout: 20000 }).should('be.visible')
    cy.contains('Fix a bug').should('be.visible')
    cy.contains('Add a feature').should('be.visible')
    cy.contains('Understand code').should('be.visible')
    cy.contains('Review a PR').should('be.visible')
    cy.wait(PAUSE)

    // ── 5. Click quick-start card → textarea prefills ───────────────
    cy.contains('button', 'Fix a bug').click()
    cy.wait(PAUSE / 2)

    // Textarea should now contain the prefilled prompt
    cy.get('textarea').should('have.value', 'Help me fix a bug in ')
    cy.wait(PAUSE / 2)

    // ── 6. Continue typing at human speed ───────────────────────────
    cy.get('textarea').type('the login flow', { delay: TYPE_DELAY })
    cy.wait(PAUSE)

    // ── 7. Phase banner: "Session is starting up" ───────────────────
    cy.contains('Session is starting up').should('be.visible')
    cy.wait(PAUSE / 2)

    // ── 8. Send → message gets queued (Creating/Pending phase) ──────
    cy.contains('button', 'Send').click()
    cy.wait(PAUSE)

    // Amber QueuedMessageBubble should appear
    cy.contains('Queued').should('be.visible')
    cy.contains('Help me fix a bug in the login flow').should('be.visible')
    cy.wait(PAUSE)

    // ── 9. Queue a second message ───────────────────────────────────
    cy.get('textarea').type('Also check the session timeout handling', { delay: TYPE_DELAY })
    cy.wait(PAUSE / 2)
    cy.contains('button', 'Send').click()
    cy.wait(PAUSE)

    // ── 10. Settings dropdown ───────────────────────────────────────
    // Settings button is the gear icon in the toolbar
    cy.get('button[class*="ghost"]').find('svg.lucide-settings').parent().click()
    cy.wait(PAUSE / 2)

    cy.contains('Display Settings').should('be.visible')
    cy.contains('Show system messages').should('be.visible')
    cy.wait(PAUSE)

    // Toggle and close
    cy.contains('Show system messages').click()
    cy.wait(PAUSE / 2)
    // Close dropdown by pressing Escape
    cy.get('body').type('{esc}')
    cy.wait(PAUSE / 2)

    // ── 11. Breadcrumb navigation back to workspace ─────────────────
    cy.contains('a', 'Sessions').should('be.visible')
    cy.contains('a', 'Workspaces').should('be.visible')
    cy.wait(PAUSE / 2)

    // Navigate back via breadcrumb
    cy.contains('a', 'Sessions').click({ force: true })
    cy.wait(PAUSE)

    // Show the workspace page with the session listed
    cy.contains('Sessions', { timeout: 10000 }).should('be.visible')
    cy.wait(PAUSE)
  })
})
