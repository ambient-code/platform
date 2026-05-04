import { describe, it, expect, vi } from 'vitest'
import { createGerritAdapter } from '../gerrit'

describe('gerritAdapter', () => {
  it('delegates getGerritInstances to API', async () => {
    const fakeApi = {
      getGerritInstances: vi.fn().mockResolvedValue({ instances: [{ connected: true, instanceName: 'gerrit-1', url: 'https://gerrit.example.com', authMethod: 'http_basic' as const }] }),
      getGerritInstanceStatus: vi.fn(),
      connectGerrit: vi.fn(),
      disconnectGerrit: vi.fn(),
      testGerritConnection: vi.fn(),
    }
    const adapter = createGerritAdapter(fakeApi)

    const result = await adapter.getGerritInstances()

    expect(result.instances).toHaveLength(1)
    expect(result.instances[0].instanceName).toBe('gerrit-1')
  })

  it('delegates testGerritConnection to API', async () => {
    const fakeApi = {
      getGerritInstances: vi.fn(),
      getGerritInstanceStatus: vi.fn(),
      connectGerrit: vi.fn(),
      disconnectGerrit: vi.fn(),
      testGerritConnection: vi.fn().mockResolvedValue({ valid: true }),
    }
    const adapter = createGerritAdapter(fakeApi)

    const result = await adapter.testGerritConnection({ url: 'https://gerrit.example.com', authMethod: 'http_basic', username: 'u', httpToken: 't' })

    expect(result.valid).toBe(true)
  })
})
