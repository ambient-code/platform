import type { MCPServerStatus, MCPConnectRequest } from './types'

export type McpCredentialsPort = {
  getMCPServerStatus: (serverName: string) => Promise<MCPServerStatus>
  connectMCPServer: (serverName: string, data: MCPConnectRequest) => Promise<void>
  disconnectMCPServer: (serverName: string) => Promise<void>
}
