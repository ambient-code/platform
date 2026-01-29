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

// Add global beforeEach to set up auth
// Note: In e2e environment, NEXT_PUBLIC_E2E_TOKEN is baked into the frontend build
// This intercept is kept as backup for direct backend API calls (if any)
beforeEach(() => {
  const token = Cypress.env('TEST_TOKEN')
  if (token) {
    // Intercept all requests and add auth header (backup)
    cy.intercept('**', (req) => {
      // Only add header if not already present (frontend adds it automatically in e2e)
      if (!req.headers['Authorization']) {
        req.headers['Authorization'] = `Bearer ${token}`
      }
    })
  }
})

// Prevent TypeScript from reading file as legacy script
export {}

