import { describe, it, expect, vi } from 'vitest'
import { createKeysAdapter } from '../keys'

describe('keysAdapter', () => {
  it('delegates listKeys to API', async () => {
    const fakeApi = {
      listKeys: vi.fn().mockResolvedValue([{ id: 'k1', name: 'my-key' }]),
      createKey: vi.fn(),
      deleteKey: vi.fn(),
    }
    const adapter = createKeysAdapter(fakeApi)

    const result = await adapter.listKeys('p')

    expect(result).toHaveLength(1)
    expect(result[0].name).toBe('my-key')
  })

  it('delegates createKey to API', async () => {
    const fakeApi = {
      listKeys: vi.fn(),
      createKey: vi.fn().mockResolvedValue({ id: 'k2', name: 'new-key', key: 'secret-value' }),
      deleteKey: vi.fn(),
    }
    const adapter = createKeysAdapter(fakeApi)

    const result = await adapter.createKey('p', { name: 'new-key' })

    expect(result.key).toBe('secret-value')
  })
})
