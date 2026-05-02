import { describe, it, expect, vi } from 'vitest'
import { createRunnerTypesAdapter } from '../runner-types'

describe('runnerTypesAdapter', () => {
  it('delegates getRunnerTypes to API', async () => {
    const fakeApi = {
      getRunnerTypes: vi.fn().mockResolvedValue([{ id: 'claude-agent-sdk', displayName: 'Claude Agent SDK', description: '', framework: 'claude', provider: 'anthropic', auth: { requiredSecretKeys: [], secretKeyLogic: 'any' as const, vertexSupported: false } }]),
    }
    const adapter = createRunnerTypesAdapter(fakeApi)

    const result = await adapter.getRunnerTypes('p')

    expect(result).toHaveLength(1)
    expect(result[0].id).toBe('claude-agent-sdk')
  })
})
