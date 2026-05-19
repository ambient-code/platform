import * as mcpApi from '../../api/mcp-credentials'
import type { McpCredentialsPort } from '../../ports/mcp-credentials'

type McpCredentialsApi = typeof mcpApi

export function createMcpCredentialsAdapter(api: McpCredentialsApi): McpCredentialsPort {
  return {
    getMCPServerStatus: api.getMCPServerStatus,
    connectMCPServer: api.connectMCPServer,
    disconnectMCPServer: api.disconnectMCPServer,
  }
}

export const mcpCredentialsAdapter = createMcpCredentialsAdapter(mcpApi)
