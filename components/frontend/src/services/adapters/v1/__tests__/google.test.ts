import { describe, it, expect, vi } from 'vitest'
import { createGoogleAdapter } from '../google'

describe('googleAdapter', () => {
  it('delegates getGoogleStatus to API', async () => {
    const fakeApi = {
      getGoogleOAuthURL: vi.fn(),
      getGoogleStatus: vi.fn().mockResolvedValue({ connected: true, email: 'user@gmail.com' }),
      disconnectGoogle: vi.fn(),
    }
    const adapter = createGoogleAdapter(fakeApi)

    const result = await adapter.getGoogleStatus()

    expect(result.connected).toBe(true)
    expect(result.email).toBe('user@gmail.com')
  })
})
