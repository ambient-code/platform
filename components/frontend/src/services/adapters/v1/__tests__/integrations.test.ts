import { describe, it, expect, vi } from 'vitest'
import { createIntegrationsAdapter } from '../integrations'

describe('integrationsAdapter', () => {
  it('delegates getIntegrationsStatus to API', async () => {
    const fakeApi = {
      getIntegrationsStatus: vi.fn().mockResolvedValue({
        github: { installed: true, pat: { configured: false } },
        google: { connected: false },
        jira: { connected: false },
        gitlab: { connected: false },
        coderabbit: { connected: false },
        gerrit: { connected: false },
      }),
    }
    const adapter = createIntegrationsAdapter(fakeApi)

    const result = await adapter.getIntegrationsStatus()

    expect(result.github.installed).toBe(true)
    expect(result.google.connected).toBe(false)
  })
})
