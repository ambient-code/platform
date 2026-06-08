import { test, expect } from '@playwright/test'

test.describe('BFF proxy endpoint', () => {
  test('GET /api/ambient/v1/sessions without auth returns 401 or 502', async ({
    request,
  }) => {
    const response = await request.get('/api/ambient/v1/sessions', {
      failOnStatusCode: false,
    })

    // Without auth: 401 (no token). Without backend: 502 (upstream unreachable).
    // 404/405 would indicate a broken route — reject those.
    expect([401, 502]).toContain(response.status())
  })
})
