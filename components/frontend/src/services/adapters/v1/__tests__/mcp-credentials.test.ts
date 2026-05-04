import { describe, it, expect, vi } from 'vitest'
import { createMcpCredentialsAdapter } from '../mcp-credentials'

describe('mcpCredentialsAdapter', () => {
  it('delegates getMCPServerStatus to API', async () => {
    const fakeApi = {
      getMCPServerStatus: vi.fn().mockResolvedValue({ connected: true, serverName: 'test-server', fieldNames: ['API_KEY'] }),
      connectMCPServer: vi.fn(),
      disconnectMCPServer: vi.fn(),
    }
    const adapter = createMcpCredentialsAdapter(fakeApi)

    const result = await adapter.getMCPServerStatus('test-server')

    expect(result.connected).toBe(true)
    expect(result.serverName).toBe('test-server')
  })
})
