import { describe, it, expect, vi } from 'vitest'
import { createModelsAdapter } from '../models'

describe('modelsAdapter', () => {
  it('delegates getModelsForProject to API', async () => {
    const fakeApi = {
      getModelsForProject: vi.fn().mockResolvedValue({ models: [{ id: 'claude-sonnet', label: 'Claude Sonnet', provider: 'anthropic', isDefault: true }], defaultModel: 'claude-sonnet' }),
    }
    const adapter = createModelsAdapter(fakeApi)

    const result = await adapter.getModelsForProject('p')

    expect(result.models).toHaveLength(1)
    expect(result.defaultModel).toBe('claude-sonnet')
  })
})
