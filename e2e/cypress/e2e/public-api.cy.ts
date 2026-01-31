/**
 * E2E Tests for Public API (/v1/*)
 *
 * Tests the public API gateway that provides simplified REST endpoints
 * for SDK, CLI, and MCP clients.
 *
 * NOTE: These tests require the public-api service to be deployed (port 30081).
 * If the service is not available, tests will be skipped.
 */
describe('Public API E2E Tests', () => {
  // Use a dedicated workspace for public API tests
  const workspaceName = `e2e-public-api-${Date.now()}`
  let createdSessionId: string
  let publicApiAvailable = false

  // Public API base URL (NodePort 30081 in kind)
  const getPublicApiUrl = () => {
    // Check for explicit public API URL first
    const publicApiUrl = Cypress.env('PUBLIC_API_URL')
    if (publicApiUrl) {
      return publicApiUrl
    }

    const baseUrl = Cypress.config('baseUrl') || 'http://localhost:8080'
    // Parse the URL to extract host
    try {
      const url = new URL(baseUrl)
      // Replace port with public-api port (30081 for kind NodePort)
      url.port = '30081'
      return url.origin
    } catch {
      // Fallback: try simple replacement
      if (baseUrl.includes(':8080')) {
        return baseUrl.replace(':8080', ':30081')
      } else if (baseUrl.includes(':80')) {
        return baseUrl.replace(':80', ':30081')
      } else {
        // No port specified, append port 30081
        return baseUrl.replace(/\/?$/, '') + ':30081'
      }
    }
  }

  before(() => {
    const token = Cypress.env('TEST_TOKEN')
    expect(token, 'TEST_TOKEN should be set').to.exist

    // First, check if public-api is available using a simple fetch
    // We use cy.wrap to handle the async operation
    const publicApiUrl = getPublicApiUrl()
    cy.log(`🔍 Checking public-api availability at ${publicApiUrl}/health`)

    cy.wrap(null).then(() => {
      return new Cypress.Promise((resolve) => {
        // Use native fetch with a timeout to check availability
        const controller = new AbortController()
        const timeoutId = setTimeout(() => controller.abort(), 3000)

        fetch(`${publicApiUrl}/health`, { signal: controller.signal })
          .then((response) => {
            clearTimeout(timeoutId)
            if (response.ok) {
              return response.json().then((body) => {
                if (body?.status === 'ok') {
                  publicApiAvailable = true
                  cy.log('✅ Public API is available')
                } else {
                  publicApiAvailable = false
                  cy.log(`⚠️ Public API returned unexpected response. Tests will be skipped.`)
                }
                resolve(null)
              })
            } else {
              publicApiAvailable = false
              cy.log(`⚠️ Public API not available (status: ${response.status}). Tests will be skipped.`)
              resolve(null)
            }
          })
          .catch(() => {
            clearTimeout(timeoutId)
            publicApiAvailable = false
            cy.log(`⚠️ Public API not reachable at ${publicApiUrl}. Tests will be skipped.`)
            resolve(null)
          })
      })
    }).then(() => {
      // Only set up workspace if public-api is available
      if (!publicApiAvailable) {
        cy.log('⏭️ Skipping workspace setup - Public API not available')
        return
      }

      // Create workspace via existing backend API (public API doesn't have project management yet)
      cy.log(`📋 Creating workspace for public API tests: ${workspaceName}`)
      cy.request({
        method: 'POST',
        url: `/api/projects`,
        headers: { 'Authorization': `Bearer ${token}` },
        body: { name: workspaceName },
        failOnStatusCode: false
      }).then((response) => {
        if (response.status === 201 || response.status === 200 || response.status === 409) {
          cy.log(`✅ Workspace created or exists: ${workspaceName}`)
        } else {
          throw new Error(`Failed to create workspace: ${response.status}`)
        }
      })

      // Wait for namespace to be ready
      cy.log('⏳ Waiting for namespace to be ready...')
      const pollProject = (attempt = 1) => {
        if (attempt > 20) throw new Error('Namespace timeout')
        cy.request({
          url: `/api/projects/${workspaceName}`,
          headers: { 'Authorization': `Bearer ${token}` },
          failOnStatusCode: false
        }).then((response) => {
          if (response.status === 200) {
            cy.log(`✅ Namespace ready after ${attempt} attempts`)
          } else {
            cy.wait(1000, { log: false })
            pollProject(attempt + 1)
          }
        })
      }
      pollProject()
    })
  })

  // Helper to skip test if public-api is not available
  const skipIfUnavailable = () => {
    if (!publicApiAvailable) {
      cy.log('⏭️ Skipping: Public API not available')
      return true
    }
    return false
  }

  after(() => {
    // Cleanup workspace
    if (!Cypress.env('KEEP_WORKSPACES')) {
      cy.log(`🗑️ Cleaning up workspace: ${workspaceName}`)
      const token = Cypress.env('TEST_TOKEN')
      cy.request({
        method: 'DELETE',
        url: `/api/projects/${workspaceName}`,
        headers: { 'Authorization': `Bearer ${token}` },
        failOnStatusCode: false
      })
    }
  })

  describe('Health Endpoints', () => {
    it('should return healthy status from /health', function() {
      if (skipIfUnavailable()) return this.skip()
      cy.request({
        url: `${getPublicApiUrl()}/health`,
        failOnStatusCode: false
      }).then((response) => {
        expect(response.status).to.eq(200)
        expect(response.body).to.have.property('status', 'ok')
      })
    })

    it('should return ready status from /ready', function() {
      if (skipIfUnavailable()) return this.skip()
      cy.request({
        url: `${getPublicApiUrl()}/ready`,
        failOnStatusCode: false
      }).then((response) => {
        expect(response.status).to.eq(200)
        expect(response.body).to.have.property('status', 'ready')
      })
    })
  })

  describe('Authentication', () => {
    it('should reject requests without token', function() {
      if (skipIfUnavailable()) return this.skip()
      cy.request({
        url: `${getPublicApiUrl()}/v1/sessions`,
        failOnStatusCode: false
      }).then((response) => {
        expect(response.status).to.eq(401)
        expect(response.body).to.have.property('error')
      })
    })

    it('should reject requests without project header', function() {
      if (skipIfUnavailable()) return this.skip()
      const token = Cypress.env('TEST_TOKEN')
      cy.request({
        url: `${getPublicApiUrl()}/v1/sessions`,
        headers: { 'Authorization': `Bearer ${token}` },
        failOnStatusCode: false
      }).then((response) => {
        expect(response.status).to.eq(400)
        expect(response.body.error).to.include('Project required')
      })
    })

    it('should accept requests with token and project header', function() {
      if (skipIfUnavailable()) return this.skip()
      const token = Cypress.env('TEST_TOKEN')
      cy.request({
        url: `${getPublicApiUrl()}/v1/sessions`,
        headers: {
          'Authorization': `Bearer ${token}`,
          'X-Ambient-Project': workspaceName
        },
        failOnStatusCode: false
      }).then((response) => {
        expect(response.status).to.eq(200)
        expect(response.body).to.have.property('items')
        expect(response.body).to.have.property('total')
      })
    })
  })

  describe('Sessions CRUD', () => {
    it('should list sessions (initially empty)', function() {
      if (skipIfUnavailable()) return this.skip()
      const token = Cypress.env('TEST_TOKEN')
      cy.request({
        url: `${getPublicApiUrl()}/v1/sessions`,
        headers: {
          'Authorization': `Bearer ${token}`,
          'X-Ambient-Project': workspaceName
        }
      }).then((response) => {
        expect(response.status).to.eq(200)
        expect(response.body.items).to.be.an('array')
        expect(response.body.total).to.be.a('number')
      })
    })

    it('should create a session', function() {
      if (skipIfUnavailable()) return this.skip()
      const token = Cypress.env('TEST_TOKEN')
      cy.request({
        method: 'POST',
        url: `${getPublicApiUrl()}/v1/sessions`,
        headers: {
          'Authorization': `Bearer ${token}`,
          'X-Ambient-Project': workspaceName,
          'Content-Type': 'application/json'
        },
        body: {
          task: 'E2E test task for public API'
        }
      }).then((response) => {
        expect(response.status).to.eq(201)
        expect(response.body).to.have.property('id')
        expect(response.body).to.have.property('message', 'Session created')
        createdSessionId = response.body.id
        cy.log(`✅ Created session: ${createdSessionId}`)
      })
    })

    it('should get session by ID', function() {
      if (skipIfUnavailable()) return this.skip()
      const token = Cypress.env('TEST_TOKEN')
      expect(createdSessionId, 'Session ID should be set from previous test').to.exist

      cy.request({
        url: `${getPublicApiUrl()}/v1/sessions/${createdSessionId}`,
        headers: {
          'Authorization': `Bearer ${token}`,
          'X-Ambient-Project': workspaceName
        }
      }).then((response) => {
        expect(response.status).to.eq(200)
        expect(response.body).to.have.property('id', createdSessionId)
        expect(response.body).to.have.property('status')
        expect(response.body).to.have.property('task', 'E2E test task for public API')
        // Verify simplified status (not raw K8s phase)
        expect(['pending', 'running', 'completed', 'failed']).to.include(response.body.status)
      })
    })

    it('should list sessions (should include created session)', function() {
      if (skipIfUnavailable()) return this.skip()
      const token = Cypress.env('TEST_TOKEN')
      cy.request({
        url: `${getPublicApiUrl()}/v1/sessions`,
        headers: {
          'Authorization': `Bearer ${token}`,
          'X-Ambient-Project': workspaceName
        }
      }).then((response) => {
        expect(response.status).to.eq(200)
        expect(response.body.items).to.be.an('array')
        expect(response.body.total).to.be.at.least(1)

        const session = response.body.items.find((s: { id: string }) => s.id === createdSessionId)
        expect(session, 'Created session should be in list').to.exist
      })
    })

    it('should delete session', function() {
      if (skipIfUnavailable()) return this.skip()
      const token = Cypress.env('TEST_TOKEN')
      expect(createdSessionId, 'Session ID should be set').to.exist

      cy.request({
        method: 'DELETE',
        url: `${getPublicApiUrl()}/v1/sessions/${createdSessionId}`,
        headers: {
          'Authorization': `Bearer ${token}`,
          'X-Ambient-Project': workspaceName
        }
      }).then((response) => {
        expect(response.status).to.eq(204)
      })
    })

    it('should return 404 for deleted session', function() {
      if (skipIfUnavailable()) return this.skip()
      const token = Cypress.env('TEST_TOKEN')
      expect(createdSessionId, 'Session ID should be set').to.exist

      cy.request({
        url: `${getPublicApiUrl()}/v1/sessions/${createdSessionId}`,
        headers: {
          'Authorization': `Bearer ${token}`,
          'X-Ambient-Project': workspaceName
        },
        failOnStatusCode: false
      }).then((response) => {
        expect(response.status).to.eq(404)
      })
    })
  })

  describe('Error Handling', () => {
    it('should return 404 for non-existent session', function() {
      if (skipIfUnavailable()) return this.skip()
      const token = Cypress.env('TEST_TOKEN')
      cy.request({
        url: `${getPublicApiUrl()}/v1/sessions/non-existent-session-id`,
        headers: {
          'Authorization': `Bearer ${token}`,
          'X-Ambient-Project': workspaceName
        },
        failOnStatusCode: false
      }).then((response) => {
        expect(response.status).to.eq(404)
      })
    })

    it('should return 400 for invalid create request (missing task)', function() {
      if (skipIfUnavailable()) return this.skip()
      const token = Cypress.env('TEST_TOKEN')
      cy.request({
        method: 'POST',
        url: `${getPublicApiUrl()}/v1/sessions`,
        headers: {
          'Authorization': `Bearer ${token}`,
          'X-Ambient-Project': workspaceName,
          'Content-Type': 'application/json'
        },
        body: {},
        failOnStatusCode: false
      }).then((response) => {
        expect(response.status).to.eq(400)
        expect(response.body).to.have.property('error')
      })
    })
  })
})
