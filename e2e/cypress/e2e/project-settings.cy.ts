/**
 * E2E Tests for Project Settings - Default Config Repo
 *
 * Tests the project-settings API (GET/PUT) for storing a workspace-level
 * default session-config repo that pre-fills into new sessions.
 */
describe('Project Settings - Default Config Repo', () => {
  const workspaceName = `e2e-settings-${Date.now()}`
  const token = Cypress.env('TEST_TOKEN')

  Cypress.on('uncaught:exception', (err) => {
    if (err.message.includes('Minified React error #418') ||
        err.message.includes('Minified React error #423') ||
        err.message.includes('Hydration')) {
      return false
    }
    return true
  })

  before(() => {
    expect(token, 'TEST_TOKEN should be set').to.exist

    // Create workspace
    cy.log(`Creating workspace: ${workspaceName}`)
    cy.visit('/projects')
    cy.contains('Workspaces', { timeout: 15000 }).should('be.visible')
    cy.contains('button', 'New Workspace').click()
    cy.contains('Create New Workspace', { timeout: 10000 }).should('be.visible')
    cy.get('#name').clear().type(workspaceName)
    cy.contains('button', 'Create Workspace').should('not.be.disabled').click({ force: true })
    cy.url({ timeout: 20000 }).should('include', `/projects/${workspaceName}`)

    // Wait for namespace ready
    const pollProject = (attempt = 1) => {
      if (attempt > 20) throw new Error('Namespace timeout')
      cy.request({
        url: `/api/projects/${workspaceName}`,
        headers: { 'Authorization': `Bearer ${token}` },
        failOnStatusCode: false
      }).then((response) => {
        if (response.status === 200) {
          cy.log(`Namespace ready after ${attempt} attempts`)
        } else {
          cy.wait(1000, { log: false })
          pollProject(attempt + 1)
        }
      })
    }
    pollProject()
  })

  after(() => {
    if (!Cypress.env('KEEP_WORKSPACES')) {
      cy.request({
        method: 'DELETE',
        url: `/api/projects/${workspaceName}`,
        headers: { 'Authorization': `Bearer ${token}` },
        failOnStatusCode: false
      })
    }
  })

  it('should return empty settings for a new workspace', () => {
    cy.request({
      url: `/api/projects/${workspaceName}/project-settings`,
      headers: { 'Authorization': `Bearer ${token}` },
    }).then((response) => {
      expect(response.status).to.eq(200)
      expect(response.body).to.not.have.property('defaultConfigRepo')
    })
  })

  it('should save and retrieve default config repo', () => {
    // PUT config repo
    cy.request({
      method: 'PUT',
      url: `/api/projects/${workspaceName}/project-settings`,
      headers: { 'Authorization': `Bearer ${token}` },
      body: {
        defaultConfigRepo: {
          gitUrl: 'https://github.com/example/session-config.git',
          branch: 'develop',
        }
      }
    }).then((response) => {
      expect(response.status).to.eq(200)
      expect(response.body.defaultConfigRepo.gitUrl).to.eq('https://github.com/example/session-config.git')
      expect(response.body.defaultConfigRepo.branch).to.eq('develop')
    })

    // GET and verify persistence
    cy.request({
      url: `/api/projects/${workspaceName}/project-settings`,
      headers: { 'Authorization': `Bearer ${token}` },
    }).then((response) => {
      expect(response.status).to.eq(200)
      expect(response.body.defaultConfigRepo.gitUrl).to.eq('https://github.com/example/session-config.git')
      expect(response.body.defaultConfigRepo.branch).to.eq('develop')
    })
  })

  it('should clear config repo when gitUrl is empty', () => {
    // Set it first
    cy.request({
      method: 'PUT',
      url: `/api/projects/${workspaceName}/project-settings`,
      headers: { 'Authorization': `Bearer ${token}` },
      body: {
        defaultConfigRepo: {
          gitUrl: 'https://github.com/example/to-be-cleared.git',
        }
      }
    })

    // Clear it
    cy.request({
      method: 'PUT',
      url: `/api/projects/${workspaceName}/project-settings`,
      headers: { 'Authorization': `Bearer ${token}` },
      body: {
        defaultConfigRepo: {
          gitUrl: '',
        }
      }
    }).then((response) => {
      expect(response.status).to.eq(200)
      expect(response.body).to.not.have.property('defaultConfigRepo')
    })

    // Verify cleared
    cy.request({
      url: `/api/projects/${workspaceName}/project-settings`,
      headers: { 'Authorization': `Bearer ${token}` },
    }).then((response) => {
      expect(response.status).to.eq(200)
      expect(response.body).to.not.have.property('defaultConfigRepo')
    })
  })

  it('should pre-fill config repo in Create Session dialog', () => {
    // Set default config repo via API
    cy.request({
      method: 'PUT',
      url: `/api/projects/${workspaceName}/project-settings`,
      headers: { 'Authorization': `Bearer ${token}` },
      body: {
        defaultConfigRepo: {
          gitUrl: 'https://github.com/example/prefill-test.git',
          branch: 'staging',
        }
      }
    })

    // Navigate to workspace and open Create Session dialog
    cy.visit(`/projects/${workspaceName}`)
    cy.contains('button', 'New Session', { timeout: 15000 }).click()
    cy.contains('Create Session', { timeout: 10000 }).should('be.visible')

    // Verify config repo fields are pre-filled
    cy.get('input[placeholder*="github.com/org/session-config"]', { timeout: 5000 })
      .should('have.value', 'https://github.com/example/prefill-test.git')
    cy.get('input[placeholder="main"]')
      .should('have.value', 'staging')
  })
})
