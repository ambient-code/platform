import { describe, it, expect, vi } from 'vitest'
import { createSessionMcpAdapter } from '../session-mcp'

describe('sessionMcpAdapter', () => {
  it('delegates getMcpStatus to API', async () => {
    const fakeApi = {
      getMcpStatus: vi.fn().mockResolvedValue({ servers: [{ name: 'test', displayName: 'Test', status: 'connected' }], totalCount: 1 }),
      updateSessionMcpServers: vi.fn(),
    }
    const adapter = createSessionMcpAdapter(fakeApi)

    const result = await adapter.getMcpStatus('p', 's')

    expect(result.servers).toHaveLength(1)
    expect(result.servers[0].name).toBe('test')
  })

  it('delegates updateSessionMcpServers to API', async () => {
    const fakeApi = {
      getMcpStatus: vi.fn(),
      updateSessionMcpServers: vi.fn().mockResolvedValue({ metadata: { name: 's' }, spec: { llmSettings: { model: 'claude', temperature: 0, maxTokens: 0 }, timeout: 0 } }),
    }
    const adapter = createSessionMcpAdapter(fakeApi)

    await adapter.updateSessionMcpServers('p', 's', {})

    expect(fakeApi.updateSessionMcpServers).toHaveBeenCalledWith('p', 's', {})
  })
})
