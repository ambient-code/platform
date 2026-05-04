import { describe, it, expect, vi } from 'vitest'
import { createAuthAdapter } from '../auth'

describe('authAdapter', () => {
  it('delegates getCurrentUser to API', async () => {
    const fakeApi = {
      getCurrentUser: vi.fn().mockResolvedValue({ authenticated: true, userId: 'u1', username: 'alice', email: 'alice@test.com' }),
    }
    const adapter = createAuthAdapter(fakeApi)

    const result = await adapter.getCurrentUser()

    expect(result.authenticated).toBe(true)
    expect(result.username).toBe('alice')
  })
})
