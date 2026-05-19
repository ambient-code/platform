import { describe, it, expect, vi } from 'vitest'
import { createConfigAdapter } from '../config'

describe('configAdapter', () => {
  it('delegates getLoadingTips to API', async () => {
    const fakeApi = {
      getLoadingTips: vi.fn().mockResolvedValue({ tips: ['Tip 1', 'Tip 2'] }),
    }
    const adapter = createConfigAdapter(fakeApi)

    const result = await adapter.getLoadingTips()

    expect(result.tips).toHaveLength(2)
    expect(result.tips[0]).toBe('Tip 1')
  })
})
