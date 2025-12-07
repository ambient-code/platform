describe('vTeam E2E Tests', () => {
  before(() => {
    // Verify auth token is available
    const token = Cypress.env('TEST_TOKEN')
    expect(token, 'TEST_TOKEN environment variable should be set').to.exist
    // Note: Auth header is automatically added via beforeEach in commands.ts
  })

  it('should access the UI with token authentication', () => {
    // Visit root, which redirects to /projects
    cy.visit('/', { failOnStatusCode: false })

    // Wait for redirect and page to load
    cy.url({ timeout: 15000 }).should('include', '/projects')
    // UI now shows "Workspaces" instead of "Projects"
    cy.contains('Workspaces', { timeout: 15000 }).should('be.visible')
  })

  it('should open create workspace dialog', () => {
    cy.visit('/projects')

    // Wait for page to be fully loaded and workspaces card to render
    cy.contains('Workspaces').should('be.visible')

    // Click the "New Workspace" button (changed from "New Project")
    cy.contains('button', 'New Workspace').click()

    // Verify dialog opens (no route change - it's a modal now)
    cy.contains('Create New Workspace').should('be.visible')

    // Close the dialog to clean up for next test
    cy.contains('button', 'Cancel').click()
  })

  it('should create a new workspace', () => {
    cy.visit('/projects')

    // Wait for page to be fully loaded
    cy.contains('Workspaces').should('be.visible')

    // Generate unique project name
    const projectName = `e2e-test-${Date.now()}`

    // Click the "New Workspace" button to open dialog
    cy.contains('button', 'New Workspace').click()

    // Wait for dialog to appear
    cy.contains('Create New Workspace').should('be.visible')

    // Fill in workspace form (vanilla k8s uses #name field)
    cy.get('#name').clear().type(projectName)

    // Wait for validation to pass and button to be enabled
    cy.contains('button', 'Create Workspace').should('not.be.disabled')

    // Submit the form (button text changed to "Create Workspace")
    cy.contains('button', 'Create Workspace').click()

    // Verify redirect to project page
    cy.url({ timeout: 15000 }).should('include', `/projects/${projectName}`)
    cy.contains(projectName).should('be.visible')
  })

  it('should list the created workspaces', () => {
    cy.visit('/projects')

    // Wait for projects list to load
    cy.get('body', { timeout: 10000 }).should('be.visible')

    // Verify we can see workspaces (terminology changed from "Projects")
    cy.contains('Workspaces').should('be.visible')
  })

  it('should access backend API cluster-info endpoint', () => {
    // Test that backend API is accessible
    // Note: /health is at root level, not under /api
    // Auth header is added automatically via interceptor
    cy.request('/api/cluster-info').then((response) => {
      expect(response.status).to.eq(200)
      expect(response.body).to.have.property('isOpenShift')
      expect(response.body.isOpenShift).to.eq(false)  // kind is vanilla k8s
    })
  })

  it('should verify MCP server can reach backend endpoints', () => {
    // Test backend endpoints that MCP server uses are accessible
    cy.request('/api/cluster-info').then((response) => {
      expect(response.status).to.eq(200)
      expect(response.body).to.have.property('isOpenShift')
    })

    cy.request('/api/workflows/ootb').then((response) => {
      expect(response.status).to.eq(200)
      expect(response.body).to.have.property('workflows')
    })
  })

  it('should verify project endpoints work with auth', () => {
    // Test authenticated endpoints that MCP uses
    cy.request({
      url: '/api/projects',
      headers: {'Authorization': `Bearer ${Cypress.env('TEST_TOKEN')}`}
    }).then((response) => {
      expect(response.status).to.eq(200)
      expect(response.body).to.have.property('items')
    })
  })

  it('should create project and verify via API (like MCP would)', () => {
    const projectName = `e2e-mcp-${Date.now()}`

    // Create via UI
    cy.visit('/projects')
    cy.contains('button', 'New Workspace').click()
    cy.get('#name').type(projectName)
    cy.contains('button', 'Create Workspace').click()
    cy.url({ timeout: 15000 }).should('include', `/projects/${projectName}`)

    // Verify via API (like MCP would do)
    cy.request({
      url: `/api/projects/${projectName}`,
      headers: {'Authorization': `Bearer ${Cypress.env('TEST_TOKEN')}`}
    }).then((response) => {
      expect(response.status).to.eq(200)
      expect(response.body.metadata.name).to.eq(projectName)
    })

    // Verify session list endpoint works
    cy.request({
      url: `/api/projects/${projectName}/agentic-sessions`,
      headers: {'Authorization': `Bearer ${Cypress.env('TEST_TOKEN')}`}
    }).then((response) => {
      expect(response.status).to.eq(200)
      expect(response.body).to.have.property('items')
      expect(response.body.items).to.be.an('array')
    })
  })
})

