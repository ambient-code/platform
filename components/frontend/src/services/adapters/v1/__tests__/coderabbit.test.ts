import { describe, it, expect, vi } from 'vitest'
import { createCodeRabbitAdapter } from '../coderabbit'

describe('coderabbitAdapter', () => {
  it('delegates getCodeRabbitStatus to API', async () => {
    const fakeApi = {
      getCodeRabbitStatus: vi.fn().mockResolvedValue({ connected: true }),
      connectCodeRabbit: vi.fn(),
      disconnectCodeRabbit: vi.fn(),
    }
    const adapter = createCodeRabbitAdapter(fakeApi)

    const result = await adapter.getCodeRabbitStatus()

    expect(result.connected).toBe(true)
  })
})
