import { describe, it, expect, vi } from 'vitest'
import { createSessionCapabilitiesAdapter } from '../session-capabilities'

describe('sessionCapabilitiesAdapter', () => {
  it('delegates getCapabilities to API', async () => {
    const fakeApi = {
      getCapabilities: vi.fn().mockResolvedValue({
        framework: 'claude-code',
        agent_features: ['tools'],
        platform_features: ['mcp'],
        file_system: true,
        mcp: true,
        tracing: null,
        session_persistence: true,
        model: 'claude-sonnet',
        session_id: 'abc',
      }),
    }
    const adapter = createSessionCapabilitiesAdapter(fakeApi)

    const result = await adapter.getCapabilities('p', 's')

    expect(result.framework).toBe('claude-code')
    expect(result.model).toBe('claude-sonnet')
  })
})
