import { describe, it, expect, vi } from 'vitest'
import { createClusterAdapter } from '../cluster'

describe('clusterAdapter', () => {
  it('delegates getClusterInfo to API', async () => {
    const fakeApi = {
      getClusterInfo: vi.fn().mockResolvedValue({ isOpenShift: true, vertexEnabled: false }),
    }
    const adapter = createClusterAdapter(fakeApi)

    const result = await adapter.getClusterInfo()

    expect(result.isOpenShift).toBe(true)
    expect(result.vertexEnabled).toBe(false)
  })
})
