// Timing constants
const LONG = 3200    // hold on important visuals
const PAUSE = 2400   // standard pause between actions
const SHORT = 1600   // brief pause after small actions

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

function moveTo(selector: string) {
  cy.get(selector).then(($el) => {
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

function cursorClickText(text: string, tag?: string, options?: { force?: boolean }) {
  moveToText(text, tag)
  clickEffect()
  const chain = tag ? cy.contains(tag, text) : cy.contains(text)
  chain.click({ force: options?.force })
}

describe('Jira Integration Demo', () => {
  Cypress.on('uncaught:exception', (err) => {
    if (err.message.includes('Minified React error') || err.message.includes('Hydration')) {
      return false
    }
    return true
  })

  it('demonstrates simplified Jira connection form', () => {
    const token = Cypress.env('TEST_TOKEN')

    // Visit integrations page
    cy.visit('/integrations', {
      headers: { Authorization: `Bearer ${token}` },
    })

    initCursor()

    caption('Navigate to Integrations page')
    cy.wait(PAUSE)

    // Scroll to Jira card if needed
    caption('Locate Jira integration card')
    cy.contains('h3', 'Jira').scrollIntoView()
    cy.wait(SHORT)

    // Click Connect Jira button
    caption('Click "Connect Jira" to open the form')
    cursorClickText('Connect Jira', 'button')
    cy.wait(PAUSE)

    // Show the simplified form
    caption('New simplified form: Only Username and API Token required')
    cy.wait(LONG)

    // Take a screenshot of the form
    cy.screenshot('jira-integration-form-simplified', { capture: 'viewport' })

    // Highlight the username field
    caption('Username field accepts any login format (e.g., rh-dept-kerberos)')
    moveTo('input[id="jira-username"]')
    cy.wait(LONG)

    // Show placeholder
    cy.get('input[id="jira-username"]').should('have.attr', 'placeholder').and('include', 'rh-dept-kerberos')
    cy.wait(SHORT)

    // Type example username
    caption('Enter your Jira username')
    clickEffect()
    cy.get('input[id="jira-username"]').type('rh-engineering-jeder', { delay: 80 })
    cy.wait(PAUSE)

    // Highlight the API token field
    caption('API Token field for authentication')
    moveTo('input[id="jira-token"]')
    cy.wait(SHORT)

    // Show that only these two fields are needed
    caption('No need to manually enter Jira URL or email - it\'s automatic!')
    cy.wait(LONG)

    // Take final screenshot
    cy.screenshot('jira-integration-form-filled', { capture: 'viewport' })

    clearCaption()
    cy.wait(SHORT)
  })
})
