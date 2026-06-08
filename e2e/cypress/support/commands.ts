/// <reference types="cypress" />

// Custom command to set auth token for all requests
declare global {
  namespace Cypress {
    interface Chainable {
      /**
       * Custom command to set Bearer token for API authentication
       * @example cy.setAuthToken('my-token-here')
       */
      setAuthToken(token: string): Chainable<void>
    }
  }
}

Cypress.Commands.add('setAuthToken', (token: string) => {
  // Intercept all HTTP requests (including fetch, XHR, etc) and add Authorization header
  cy.intercept('**', (req) => {
    req.headers['Authorization'] = `Bearer ${token}`
  }).as('authInterceptor')
})

// Set up auth before each test.
// In SSO mode, creates a session cookie via the E2E login route so that
// cy.visit() page navigations pass the middleware without Keycloak redirect.
// Also intercepts all fetch/XHR requests to add the Authorization header.
beforeEach(() => {
  const token = Cypress.env('TEST_TOKEN')
  if (!token) return

  // Create SSO session cookie if the frontend is in SSO mode
  if (Cypress.env('SSO_MODE')) {
    cy.request({
      method: 'POST',
      url: '/api/auth/sso/e2e-login',
      body: { token },
      failOnStatusCode: false,
    })
  }

  // Intercept all fetch/XHR requests and add auth header
  cy.intercept('**', (req) => {
    if (!req.headers['Authorization']) {
      req.headers['Authorization'] = `Bearer ${token}`
    }
  })
})

// Prevent TypeScript from reading file as legacy script
export {}
