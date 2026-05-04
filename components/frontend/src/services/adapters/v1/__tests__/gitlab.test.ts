import { describe, it, expect, vi } from 'vitest'
import { createGitLabAdapter } from '../gitlab'

describe('gitlabAdapter', () => {
  it('delegates getGitLabStatus to API', async () => {
    const fakeApi = {
      getGitLabStatus: vi.fn().mockResolvedValue({ connected: true, instanceUrl: 'https://gitlab.com' }),
      connectGitLab: vi.fn(),
      disconnectGitLab: vi.fn(),
    }
    const adapter = createGitLabAdapter(fakeApi)

    const result = await adapter.getGitLabStatus()

    expect(result.connected).toBe(true)
    expect(result.instanceUrl).toBe('https://gitlab.com')
  })
})
