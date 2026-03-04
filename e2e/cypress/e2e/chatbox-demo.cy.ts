/**
 * Chatbox Component Demo
 *
 * Records a human-paced walkthrough of ALL refactored chat UI components
 * with a visible cursor, click ripple, and top-bar captions.
 *
 * Requires ANTHROPIC_API_KEY in e2e/.env or .env.local for the session
 * to reach Running state and populate agents/commands from the workflow.
 *
 * Run:  npx cypress run --no-runner-ui --spec "cypress/e2e/chatbox-demo.cy.ts"
 * Video: cypress/videos/chatbox-demo.cy.ts.mp4
 */
describe('Chatbox Component Demo', () => {
  const workspaceName = `chatbox-demo-${Date.now()}`

  const LONG = 3200
  const PAUSE = 2400
  const SHORT = 1600
  const TYPE_DELAY = 80

  const chatInput = () => cy.get('textarea[placeholder*="message"]').first()

  // ─── Caption: compact bar at the TOP ──────────────────────────
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

  // ─── Fake cursor + click ripple ───────────────────────────────
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

  function cursorClickText(text: string, tag?: string, options?: { force?: boolean }) {
    moveToText(text, tag)
    clickEffect()
    const chain = tag ? cy.contains(tag, text) : cy.contains(text)
    chain.click({ force: options?.force })
  }

  // ─── Cypress setup ────────────────────────────────────────────
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

  // ─── Demo ─────────────────────────────────────────────────────
  it('walks through the refactored chat UI components', () => {
    const token = Cypress.env('TEST_TOKEN')
    expect(token, 'TEST_TOKEN should be set').to.exist

    const apiKey = Cypress.env('ANTHROPIC_API_KEY')
    if (!apiKey) {
      throw new Error('ANTHROPIC_API_KEY not set. Required for workflow agents/commands demo.')
    }

    // ── 1. Workspaces page ────────────────────────────────────
    cy.visit('/projects')
    cy.contains('Workspaces', { timeout: 15000 }).should('be.visible')
    initCursor()
    caption('Ambient Code Platform — Workspaces')
    cy.wait(LONG)

    // ── 2. Create workspace ───────────────────────────────────
    caption('Creating a new workspace')
    cy.wait(SHORT)
    cursorClickText('New Workspace', 'button')
    cy.contains('Create New Workspace', { timeout: 10000 }).should('be.visible')
    caption('Create New Workspace dialog')
    cy.wait(PAUSE)

    moveTo('#name')
    clickEffect()
    cy.get('#name').clear().type(workspaceName, { delay: 40 })
    caption('Workspace name entered')
    cy.wait(PAUSE)

    cursorClickText('Create Workspace', 'button', { force: true })
    cy.url({ timeout: 20000 }).should('include', `/projects/${workspaceName}`)

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
    caption('Workspace created — Kubernetes namespace provisioned')
    cy.wait(LONG)

    // ── 2b. Configure API key for session to reach Running ────
    cy.request({
      method: 'PUT',
      url: `/api/projects/${workspaceName}/runner-secrets`,
      headers: { Authorization: `Bearer ${token}` },
      body: { data: { ANTHROPIC_API_KEY: apiKey } },
    }).then((resp) => {
      expect(resp.status).to.eq(200)
    })

    // ── 3. Create session ─────────────────────────────────────
    caption('Click "New Session" to create an agentic session')
    cy.wait(SHORT)
    cursorClickText('New Session', 'button')
    caption('Session creation dialog — click "Create"')
    cy.wait(PAUSE)
    cursorClickText('Create', 'button')
    cy.url({ timeout: 30000 }).should('match', /\/sessions\/[a-z0-9-]+$/)
    caption('Session created — navigated to session detail page')
    cy.wait(LONG)

    // ── 4. WelcomeExperience ──────────────────────────────────
    caption('WelcomeExperience — AI greeting streams in character-by-character')
    cy.contains('p', 'Welcome to Ambient AI', { timeout: 20000 }).should('be.visible')
    cy.wait(LONG)

    cy.contains('button', 'View all workflows', { timeout: 10000 }).should('be.visible')
    caption('Workflow cards — choose a workflow to get started')
    cy.wait(LONG)

    // ── 5. Select "Fix a bug" workflow ────────────────────────
    caption('Select the "Fix a bug" workflow')
    cy.wait(SHORT)
    cy.contains('h3', 'Fix a bug').parents('[class*="cursor-pointer"]').first().then(($card) => {
      moveToEl($card)
      clickEffect()
      cy.wrap($card).click()
    })
    caption('Workflow selected — "Fix a bug" workflow activating')
    cy.wait(LONG)

    // ── 6. Wait for session to reach Running state ────────────
    caption('Waiting for session pod to reach Running state...')
    chatInput().should('be.visible')
    // Wait for the textarea placeholder to change — when Running it drops "queued until session starts"
    cy.get('textarea[placeholder*="attach"]', { timeout: 180000 })
      .first()
      .should('exist')
    caption('Session Running — workflow loaded, agents and commands available')
    cy.wait(LONG)

    // ── 7. File attachment ────────────────────────────────────
    caption('Feature: File Attachments — attaching a file')
    cy.wait(SHORT)
    moveTo('button[title="Attach file"]', { first: true })
    clickEffect()
    cy.get('input[type="file"]').first().selectFile(
      {
        contents: Cypress.Buffer.from('console.log("hello world")'),
        fileName: 'example.js',
        mimeType: 'text/javascript',
      },
      { force: true }
    )
    cy.wait(SHORT)
    caption('AttachmentPreview — file name, size, and remove button')
    cy.contains('example.js').should('be.visible')
    cy.wait(LONG)

    caption('Click X to remove the attachment')
    cy.wait(SHORT)
    cy.contains('example.js').parent().parent().find('button').then(($btn) => {
      moveToEl($btn)
      clickEffect()
      cy.wrap($btn).click({ force: true })
    })
    caption('Attachment removed')
    cy.wait(PAUSE)

    // ── 8. Autocomplete: @ agents (populated from workflow) ───
    caption('Feature: Autocomplete — type @ to mention an agent')
    moveTo('textarea[placeholder*="message"]', { first: true })
    clickEffect()
    cy.wait(SHORT)
    chatInput().type('@', { delay: TYPE_DELAY })
    cy.wait(LONG)
    caption('AutocompletePopover — workflow agents loaded, navigate with arrow keys')
    cy.wait(LONG)
    chatInput().type('{esc}')
    chatInput().clear()
    cy.wait(SHORT)

    // ── 9. Autocomplete: / commands (populated from workflow) ─
    caption('Feature: Autocomplete — type / to trigger slash commands')
    clickEffect()
    cy.wait(SHORT)
    chatInput().type('/', { delay: TYPE_DELAY })
    cy.wait(LONG)
    caption('Slash command popover — commands loaded from bugfix workflow')
    cy.wait(LONG)
    chatInput().type('{esc}')
    chatInput().clear()
    cy.wait(SHORT)

    // ── 10. Toolbar: Agents button ───────────────────────────
    caption('Toolbar: Agents button — browse all agents from the workflow')
    cy.wait(SHORT)
    cy.contains('button', 'Agents').first().then(($btn) => {
      moveToEl($btn)
      clickEffect()
      // Only click if the button is not disabled (agents loaded)
      if (!$btn.prop('disabled')) {
        cy.wrap($btn).click()
        cy.wait(LONG)
        cy.get('body').type('{esc}')
      }
    })
    cy.wait(SHORT)

    // ── 11. Toolbar: Commands button ─────────────────────────
    caption('Toolbar: Commands button — browse all slash commands')
    cy.wait(SHORT)
    cy.contains('button', 'Commands').first().then(($btn) => {
      moveToEl($btn)
      clickEffect()
      if (!$btn.prop('disabled')) {
        cy.wrap($btn).click()
        cy.wait(LONG)
        cy.get('body').type('{esc}')
      }
    })
    cy.wait(SHORT)

    // ── 12. Type and send a message (session is Running) ─────
    caption('Feature: Chat — type and send a message to the running agent')
    moveTo('textarea[placeholder*="message"]', { first: true })
    clickEffect()
    cy.wait(SHORT)
    chatInput().type('Help me fix a bug in the login flow', { delay: TYPE_DELAY })
    cy.wait(PAUSE)

    caption('Click Send — message goes directly to the running agent')
    cy.wait(SHORT)
    cursorClickText('Send', 'button')
    cy.wait(PAUSE)

    // ── 13. Agent is processing ──────────────────────────────
    caption('Agent is processing — Stop button appears to interrupt')
    // Send button disappears, Stop button appears
    cy.contains('button', 'Stop', { timeout: 10000 }).should('be.visible')
    cy.wait(LONG)

    // ── 14. Settings dropdown ───────────────────────────────
    caption('Feature: Settings Dropdown — click the gear icon')
    cy.wait(SHORT)
    cy.get('svg.lucide-settings').first().parent('button').then(($btn) => {
      moveToEl($btn)
      clickEffect()
      cy.wrap($btn).click()
    })
    cy.wait(SHORT)

    cy.contains('Display Settings').should('be.visible')
    caption('Display Settings — toggle system message visibility')
    cy.wait(LONG)

    cursorClickText('Show system messages')
    caption('System messages toggled on')
    cy.wait(PAUSE)
    cy.get('body').type('{esc}')
    cy.wait(SHORT)

    // ── 15. Breadcrumb navigation ───────────────────────────
    caption('Feature: Breadcrumb Navigation — click to go back')
    cy.contains('a', 'Sessions').should('be.visible')
    cy.wait(PAUSE)

    caption('Navigate back to workspace')
    cursorClickText('Sessions', 'a', { force: true })
    cy.wait(SHORT)

    caption('Workspace view — session listed with status badge')
    cy.contains('Sessions', { timeout: 10000 }).should('be.visible')
    cy.wait(LONG)

    clearCaption()
    cy.wait(SHORT)
  })
})
