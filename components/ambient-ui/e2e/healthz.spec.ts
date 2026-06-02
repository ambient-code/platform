import { test, expect } from '@playwright/test'

test.describe('Health check endpoint', () => {
  test('GET /api/healthz returns 200 with status ok', async ({ request }) => {
    const response = await request.get('/api/healthz')

    expect(response.status()).toBe(200)
    const body = await response.json()
    expect(body).toEqual({ status: 'ok' })
  })
})
