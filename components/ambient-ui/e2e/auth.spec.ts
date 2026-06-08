import { test, expect } from '@playwright/test'

test.describe('Authentication endpoints', () => {
  test('GET /api/me returns unauthenticated when no session cookie', async ({
    request,
  }) => {
    const response = await request.get('/api/me')

    expect(response.status()).toBe(200)
    const body = await response.json()
    expect(body.authenticated).toBe(false)
  })

  test('GET /api/config returns config shape', async ({ request }) => {
    const response = await request.get('/api/config')

    expect(response.status()).toBe(200)
    const body = await response.json()
    expect(body).toHaveProperty('apiServerUrl')
    expect(body).toHaveProperty('isCustomContext')
    expect(typeof body.apiServerUrl).toBe('string')
    expect(typeof body.isCustomContext).toBe('boolean')
  })
})
