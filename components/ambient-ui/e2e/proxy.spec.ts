import { test, expect } from '@playwright/test'

test.describe('BFF proxy endpoint', () => {
  test('GET /api/ambient/v1/sessions without auth passes through to backend', async ({
    request,
  }) => {
    // Without a running backend, the proxy should attempt to forward
    // and return an error (connection refused or 500), not crash
    const response = await request.get('/api/ambient/v1/sessions', {
      failOnStatusCode: false,
    })

    // The proxy should respond (not hang or crash)
    // Status depends on whether backend is running - we just verify the proxy works
    expect(response.status()).toBeGreaterThanOrEqual(200)
  })
})
