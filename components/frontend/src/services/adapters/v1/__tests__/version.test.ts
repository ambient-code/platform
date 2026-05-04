import { describe, it, expect, vi } from 'vitest'
import { createVersionAdapter } from '../version'

describe('versionAdapter', () => {
  it('delegates getVersion to API', async () => {
    const fakeApi = {
      getVersion: vi.fn().mockResolvedValue('1.2.3'),
    }
    const adapter = createVersionAdapter(fakeApi)

    const result = await adapter.getVersion()

    expect(result).toBe('1.2.3')
  })
})
