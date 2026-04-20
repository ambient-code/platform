/**
 * E2E Tests for Scheduled Sessions
 *
 * Covers: creation (preset & custom cron), cron validation (client & server),
 * schedule update, and deletion.
 */
describe('Scheduled Sessions', () => {
  const workspaceName = `e2e-sched-${Date.now()}`
  let workspaceSlug: string

  Cypress.on('uncaught:exception', (err) => {
    if (err.message.includes('Minified React error #418') ||
        err.message.includes('Minified React error #423') ||
        err.message.includes('Hydration')) {
      return false
    }
    return true
  })

  function apiHeaders() {
    return { Authorization: `Bearer ${Cypress.env('TEST_TOKEN')}` }
  }

  before(() => {
    const token = Cypress.env('TEST_TOKEN')
    expect(token, 'TEST_TOKEN should be set').to.exist

    // Create workspace via API
    cy.request({
      method: 'POST',
      url: '/api/projects',
      headers: apiHeaders(),
      body: { name: workspaceName, displayName: workspaceName },
    }).then((resp) => {
      expect(resp.status).to.be.oneOf([200, 201])
      workspaceSlug = resp.body.name || workspaceName

      // Poll until namespace is ready
      const pollProject = (attempt: number): void => {
        if (attempt > 30) throw new Error('Namespace timeout')
        cy.request({
          url: `/api/projects/${workspaceSlug}`,
          headers: apiHeaders(),
          failOnStatusCode: false,
        }).then((response) => {
          if (response.status !== 200) {
            cy.wait(1500, { log: false })
            pollProject(attempt + 1)
          }
        })
      }
      pollProject(1)
    })

    // Set runner secrets (needed for session template validation)
    cy.then(() => cy.request({
      method: 'PUT',
      url: `/api/projects/${workspaceSlug}/runner-secrets`,
      headers: apiHeaders(),
      body: { data: { ANTHROPIC_API_KEY: 'mock-replay-key' } },
    })).then((r) => expect(r.status).to.eq(200))
  })

  after(() => {
    if (!Cypress.env('KEEP_WORKSPACES')) {
      cy.request({
        method: 'DELETE',
        url: `/api/projects/${workspaceSlug}`,
        headers: apiHeaders(),
        failOnStatusCode: false,
      })
    }
  })

  // Helper: create a scheduled session via API, return its name
  function createScheduledSessionViaApi(schedule: string, displayName?: string): Cypress.Chainable<string> {
    return cy.request({
      method: 'POST',
      url: `/api/projects/${workspaceSlug}/scheduled-sessions`,
      headers: apiHeaders(),
      body: {
        ...(displayName ? { displayName } : {}),
        schedule,
        sessionTemplate: {
          initialPrompt: 'e2e test prompt',
          runnerType: 'claude-code',
          llmSettings: { model: 'claude-sonnet-4-20250514', temperature: 0.7, maxTokens: 4000 },
          timeout: 300,
        },
      },
    }).then((resp) => {
      expect(resp.status).to.be.oneOf([200, 201])
      return resp.body.name as string
    })
  }

  // ─── Schedule Creation ────────────────────────────────────────

  describe('Schedule Creation', () => {
    it('should create a scheduled session with a preset schedule', () => {
      cy.visit(`/projects/${workspaceSlug}/scheduled-sessions/new`)

      // Fill display name
      cy.get('[data-testid="scheduled-session-name-input"]', { timeout: 10000 })
        .type('Preset Schedule Test')

      // Select "Daily at 9:00 AM" preset
      cy.get('[data-testid="schedule-preset-select"]').click()
      cy.get('[role="option"]').contains('Daily at 9:00 AM').click()

      // Verify cron preview shows description and next runs
      cy.get('[data-testid="cron-preview"]', { timeout: 5000 }).should('exist')
      cy.get('[data-testid="cron-preview"]').should('contain.text', 'Next 3 runs')

      // Fill initial prompt
      cy.get('[data-testid="initial-prompt-input"]').type('Run daily health check')

      // Wait for runner type and model to load, leave defaults
      cy.get('[data-testid="runner-type-select"]', { timeout: 10000 }).should('not.be.disabled')
      cy.get('[data-testid="model-select"]', { timeout: 10000 }).should('not.be.disabled')

      // Submit
      cy.get('[data-testid="scheduled-session-submit"]').click()

      // Verify redirect to scheduled sessions list
      cy.url({ timeout: 15000 }).should('include', `/projects/${workspaceSlug}/scheduled-sessions`)
      cy.url().should('not.include', '/new')

      // Verify the schedule appears in the list
      cy.contains('Preset Schedule Test', { timeout: 10000 }).should('be.visible')
      cy.contains('Active').should('exist')
    })

    it('should create a scheduled session with a custom cron expression', () => {
      cy.visit(`/projects/${workspaceSlug}/scheduled-sessions/new`)

      // Fill display name
      cy.get('[data-testid="scheduled-session-name-input"]', { timeout: 10000 })
        .type('Custom Cron Test')

      // Select "Custom" preset
      cy.get('[data-testid="schedule-preset-select"]').click()
      cy.get('[role="option"]').contains('Custom').click()

      // Enter custom cron expression
      cy.get('[data-testid="custom-cron-input"]').type('*/30 * * * *')

      // Verify cron preview updates
      cy.get('[data-testid="cron-preview"]').should('contain.text', 'Every 30 minutes')
      cy.get('[data-testid="cron-preview"]').should('contain.text', 'Next 3 runs')

      // Fill initial prompt
      cy.get('[data-testid="initial-prompt-input"]').type('Run every 30 minutes')

      // Wait for runner/model to load
      cy.get('[data-testid="runner-type-select"]', { timeout: 10000 }).should('not.be.disabled')
      cy.get('[data-testid="model-select"]', { timeout: 10000 }).should('not.be.disabled')

      // Submit
      cy.get('[data-testid="scheduled-session-submit"]').click()

      // Verify redirect and listing
      cy.url({ timeout: 15000 }).should('include', `/projects/${workspaceSlug}/scheduled-sessions`)
      cy.url().should('not.include', '/new')
      cy.contains('Custom Cron Test', { timeout: 10000 }).should('be.visible')
    })
  })

  // ─── Cron Validation ──────────────────────────────────────────

  describe('Cron Validation', () => {
    it('should show client-side validation error when custom cron is empty', () => {
      cy.visit(`/projects/${workspaceSlug}/scheduled-sessions/new`)

      // Select "Custom" preset
      cy.get('[data-testid="schedule-preset-select"]', { timeout: 10000 }).click()
      cy.get('[role="option"]').contains('Custom').click()

      // Leave custom cron input empty

      // Fill initial prompt so only the cron field blocks submission
      cy.get('[data-testid="initial-prompt-input"]').type('This should not submit')

      // Wait for runner/model to load
      cy.get('[data-testid="runner-type-select"]', { timeout: 10000 }).should('not.be.disabled')
      cy.get('[data-testid="model-select"]', { timeout: 10000 }).should('not.be.disabled')

      // Click submit
      cy.get('[data-testid="scheduled-session-submit"]').click()

      // Verify form validation error appears
      cy.contains('Cron expression is required', { timeout: 5000 }).should('be.visible')

      // Verify we are still on the new page (no redirect)
      cy.url().should('include', '/new')
    })

    it('should show server-side validation error for invalid cron syntax', () => {
      cy.visit(`/projects/${workspaceSlug}/scheduled-sessions/new`)

      // Select "Custom" preset
      cy.get('[data-testid="schedule-preset-select"]', { timeout: 10000 }).click()
      cy.get('[role="option"]').contains('Custom').click()

      // Enter invalid cron (3 fields instead of required 5)
      cy.get('[data-testid="custom-cron-input"]').type('* * *')

      // Fill initial prompt
      cy.get('[data-testid="initial-prompt-input"]').type('This should fail server-side')

      // Wait for runner/model to load
      cy.get('[data-testid="runner-type-select"]', { timeout: 10000 }).should('not.be.disabled')
      cy.get('[data-testid="model-select"]', { timeout: 10000 }).should('not.be.disabled')

      // Submit — should pass client-side validation but fail server-side
      cy.get('[data-testid="scheduled-session-submit"]').click()

      // Verify error toast from backend (assert type, not message text, to avoid coupling to backend wording)
      cy.get('[data-sonner-toast][data-type="error"]', { timeout: 10000 })
        .should('exist')

      // Verify we are still on the new page (no redirect)
      cy.url().should('include', '/new')
    })
  })

  // ─── Schedule Update ──────────────────────────────────────────

  describe('Schedule Update', () => {
    let scheduleName: string

    before(() => {
      createScheduledSessionViaApi('0 * * * *', 'Update Test Schedule').then((name) => {
        scheduleName = name
      })
    })

    it('should update a scheduled session cron schedule', () => {
      cy.visit(`/projects/${workspaceSlug}/scheduled-sessions/${scheduleName}/edit`)

      // Wait for form to load with existing data and verify pre-selected value
      cy.get('[data-testid="schedule-preset-select"]', { timeout: 10000 })
        .should('contain.text', 'Every hour')

      // Change schedule to "Weekly on Monday"
      cy.get('[data-testid="schedule-preset-select"]').click()
      cy.get('[role="option"]').contains('Weekly on Monday').click()

      // Verify cron preview updates
      cy.get('[data-testid="cron-preview"]').should('contain.text', 'Next 3 runs')

      // Submit
      cy.get('[data-testid="scheduled-session-submit"]').click()

      // Verify redirect back to list
      cy.url({ timeout: 15000 }).should('include', `/projects/${workspaceSlug}/scheduled-sessions`)
      cy.url().should('not.include', '/edit')

      // Verify via API that the schedule was updated
      cy.request({
        url: `/api/projects/${workspaceSlug}/scheduled-sessions/${scheduleName}`,
        headers: apiHeaders(),
      }).then((resp) => {
        expect(resp.status).to.eq(200)
        expect(resp.body.schedule).to.eq('0 9 * * 1')
      })
    })
  })

  // ─── Custom Workflow ────────────────────────────────────────

  describe('Custom Workflow', () => {
    it('should create a scheduled session with a custom workflow', () => {
      cy.visit(`/projects/${workspaceSlug}/scheduled-sessions/new`)

      // Fill display name
      cy.get('[data-testid="scheduled-session-name-input"]', { timeout: 10000 })
        .type('Custom Workflow Test')

      // Leave default schedule preset (Every hour)

      // Fill initial prompt
      cy.get('[data-testid="initial-prompt-input"]').type('Run with custom workflow')

      // Select "Custom workflow..." from workflow dropdown
      cy.get('[data-testid="workflow-select"]').click()
      cy.get('[role="option"]').contains('Custom workflow...').click()

      // Fill custom workflow fields
      cy.get('[data-testid="workflow-git-url"]', { timeout: 5000 })
        .should('be.visible')
        .type('https://github.com/ambient-code/workflows.git')
      cy.get('[data-testid="workflow-branch"]').clear().type('main')
      cy.get('[data-testid="workflow-path"]').type('workflows/bugfix')

      // Wait for runner/model to load
      cy.get('[data-testid="runner-type-select"]', { timeout: 10000 }).should('not.be.disabled')
      cy.get('[data-testid="model-select"]', { timeout: 10000 }).should('not.be.disabled')

      // Submit
      cy.get('[data-testid="scheduled-session-submit"]').click()

      // Verify redirect to list
      cy.url({ timeout: 15000 }).should('include', `/projects/${workspaceSlug}/scheduled-sessions`)
      cy.url().should('not.include', '/new')

      // Verify it appears in the list
      cy.contains('Custom Workflow Test', { timeout: 10000 }).should('be.visible')
    })

    it('should edit the custom workflow on an existing scheduled session', () => {
      // Create a scheduled session with a custom workflow via API
      // Use fields that do NOT match any OOTB workflow so it stays as "Custom workflow..."
      cy.request({
        method: 'POST',
        url: `/api/projects/${workspaceSlug}/scheduled-sessions`,
        headers: apiHeaders(),
        body: {
          displayName: 'Workflow Edit Test',
          schedule: '0 * * * *',
          sessionTemplate: {
            initialPrompt: 'original prompt',
            runnerType: 'claude-code',
            llmSettings: { model: 'claude-sonnet-4-20250514', temperature: 0.7, maxTokens: 4000 },
            timeout: 300,
            activeWorkflow: {
              gitUrl: 'https://github.com/my-org/my-custom-workflows.git',
              branch: 'main',
              path: 'workflows/custom-one',
            },
          },
        },
      }).then((resp) => {
        expect(resp.status).to.be.oneOf([200, 201])
        const scheduleName = resp.body.name

        // Navigate to edit page
        cy.visit(`/projects/${workspaceSlug}/scheduled-sessions/${scheduleName}/edit`)

        // Wait for form to load — should show "Custom workflow..." (no OOTB match)
        cy.get('[data-testid="workflow-select"]', { timeout: 10000 })
          .should('contain.text', 'Custom workflow...')

        // Verify pre-populated custom workflow fields
        cy.get('[data-testid="workflow-git-url"]')
          .should('have.value', 'https://github.com/my-org/my-custom-workflows.git')
        cy.get('[data-testid="workflow-branch"]')
          .should('have.value', 'main')
        cy.get('[data-testid="workflow-path"]')
          .should('have.value', 'workflows/custom-one')

        // Update the workflow fields
        cy.get('[data-testid="workflow-branch"]').clear().type('develop')
        cy.get('[data-testid="workflow-path"]').clear().type('workflows/custom-two')

        // Submit
        cy.get('[data-testid="scheduled-session-submit"]').click()

        // Verify redirect
        cy.url({ timeout: 15000 }).should('include', `/projects/${workspaceSlug}/scheduled-sessions`)
        cy.url().should('not.include', '/edit')

        // Verify via API that the workflow was updated
        cy.request({
          url: `/api/projects/${workspaceSlug}/scheduled-sessions/${scheduleName}`,
          headers: apiHeaders(),
        }).then((getResp) => {
          expect(getResp.status).to.eq(200)
          const workflow = getResp.body.sessionTemplate.activeWorkflow
          expect(workflow.gitUrl).to.eq('https://github.com/my-org/my-custom-workflows.git')
          expect(workflow.branch).to.eq('develop')
          expect(workflow.path).to.eq('workflows/custom-two')
        })
      })
    })

    it('should show OOTB workflow name when editing a session created with an OOTB workflow', () => {
      // Create a session with fields that match the "bugfix" OOTB workflow
      cy.request({
        method: 'POST',
        url: `/api/projects/${workspaceSlug}/scheduled-sessions`,
        headers: apiHeaders(),
        body: {
          displayName: 'OOTB Workflow Edit Test',
          schedule: '0 * * * *',
          sessionTemplate: {
            initialPrompt: 'run bugfix workflow',
            runnerType: 'claude-code',
            llmSettings: { model: 'claude-sonnet-4-20250514', temperature: 0.7, maxTokens: 4000 },
            timeout: 300,
            activeWorkflow: {
              gitUrl: 'https://github.com/ambient-code/workflows.git',
              branch: 'main',
              path: 'workflows/bugfix',
            },
          },
        },
      }).then((resp) => {
        expect(resp.status).to.be.oneOf([200, 201])
        const scheduleName = resp.body.name

        // Navigate to edit page
        cy.visit(`/projects/${workspaceSlug}/scheduled-sessions/${scheduleName}/edit`)

        // The workflow select should show the OOTB workflow name, not "Custom workflow..."
        cy.get('[data-testid="workflow-select"]', { timeout: 10000 })
          .should('contain.text', 'Fix a bug')

        // Custom workflow fields should NOT be visible (OOTB selected, not custom)
        cy.get('[data-testid="workflow-git-url"]').should('not.exist')
      })
    })

    it('should show General chat when editing a session with no workflow', () => {
      // Create a session without any workflow
      createScheduledSessionViaApi('0 * * * *', 'No Workflow Edit Test').then((scheduleName) => {
        // Navigate to edit page
        cy.visit(`/projects/${workspaceSlug}/scheduled-sessions/${scheduleName}/edit`)

        // The workflow select should show "General chat"
        cy.get('[data-testid="workflow-select"]', { timeout: 10000 })
          .should('contain.text', 'General chat')

        // Custom workflow fields should NOT be visible
        cy.get('[data-testid="workflow-git-url"]').should('not.exist')
      })
    })
  })

  // ─── Suspend / Resume ─────────────────────────────────────────

  describe('Suspend and Resume', () => {
    let scheduleName: string

    before(() => {
      createScheduledSessionViaApi('0 * * * *', 'Suspend Resume Test').then((name) => {
        scheduleName = name
      })
    })

    it('should suspend an active scheduled session', () => {
      cy.visit(`/projects/${workspaceSlug}/scheduled-sessions`)

      cy.get(`[data-testid="scheduled-session-row-${scheduleName}"]`, { timeout: 10000 })
        .should('exist')

      cy.get(`[data-testid="scheduled-session-actions-${scheduleName}"]`).click()
      cy.get('[data-testid="scheduled-session-suspend"]').click()

      cy.get('[data-sonner-toast]', { timeout: 10000 }).should('exist')

      cy.request({
        url: `/api/projects/${workspaceSlug}/scheduled-sessions/${scheduleName}`,
        headers: apiHeaders(),
      }).then((resp) => {
        expect(resp.status).to.eq(200)
        expect(resp.body.suspend).to.eq(true)
      })
    })

    it('should resume a suspended scheduled session', () => {
      cy.visit(`/projects/${workspaceSlug}/scheduled-sessions`)

      cy.get(`[data-testid="scheduled-session-row-${scheduleName}"]`, { timeout: 10000 })
        .should('exist')

      cy.get(`[data-testid="scheduled-session-actions-${scheduleName}"]`).click()
      cy.get('[data-testid="scheduled-session-resume"]').click()

      cy.get('[data-sonner-toast]', { timeout: 10000 }).should('exist')

      cy.request({
        url: `/api/projects/${workspaceSlug}/scheduled-sessions/${scheduleName}`,
        headers: apiHeaders(),
      }).then((resp) => {
        expect(resp.status).to.eq(200)
        expect(resp.body.suspend).to.eq(false)
      })
    })
  })

  // ─── Trigger Now ──────────────────────────────────────────────

  describe('Trigger Now', () => {
    let scheduleName: string

    before(() => {
      createScheduledSessionViaApi('0 * * * *', 'Trigger Now Test').then((name) => {
        scheduleName = name
      })
    })

    it('should trigger a scheduled session immediately', () => {
      cy.visit(`/projects/${workspaceSlug}/scheduled-sessions`)

      cy.get(`[data-testid="scheduled-session-row-${scheduleName}"]`, { timeout: 10000 })
        .should('exist')

      cy.get(`[data-testid="scheduled-session-actions-${scheduleName}"]`).click()
      cy.get('[data-testid="scheduled-session-trigger"]').click()

      cy.get('[data-sonner-toast]', { timeout: 10000 }).should('exist')
    })
  })

  // ─── Schedule Deletion ────────────────────────────────────────

  describe('Schedule Deletion', () => {
    let scheduleName: string

    before(() => {
      createScheduledSessionViaApi('0 * * * *', 'Delete Test Schedule').then((name) => {
        scheduleName = name
      })
    })

    it('should delete a scheduled session from the list', () => {
      cy.visit(`/projects/${workspaceSlug}/scheduled-sessions`)

      // Wait for the list to load and find the session row
      cy.get(`[data-testid="scheduled-session-row-${scheduleName}"]`, { timeout: 10000 })
        .should('exist')

      // Stub confirm dialog to accept
      cy.on('window:confirm', () => true)

      // Open actions dropdown
      cy.get(`[data-testid="scheduled-session-actions-${scheduleName}"]`).click()

      // Click delete
      cy.get('[data-testid="scheduled-session-delete"]').click()

      // Verify success toast
      cy.get('[data-sonner-toast]', { timeout: 10000 })
        .should('contain.text', 'Deleted')

      // Verify the session is removed from the list
      cy.get(`[data-testid="scheduled-session-row-${scheduleName}"]`).should('not.exist')
    })
  })
})
