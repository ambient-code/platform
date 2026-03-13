import { apiClient } from './client'

export type MCPServerStatus = {
  connected: boolean
  serverName: string
  fieldNames?: string[]
  updatedAt?: string
}

export type MCPConnectRequest = {
  fields: Record<string, string>
}

/**
 * Get MCP server credential status for the authenticated user
 */
export async function getMCPServerStatus(serverName: string): Promise<MCPServerStatus> {
  return apiClient.get<MCPServerStatus>(`/auth/mcp/${serverName}/status`)
}

/**
 * Connect (store credentials for) an MCP server
 */
export async function connectMCPServer(
  serverName: string,
  data: MCPConnectRequest
): Promise<void> {
  await apiClient.post<void, MCPConnectRequest>(`/auth/mcp/${serverName}/connect`, data)
}

/**
 * Disconnect (remove credentials for) an MCP server
 */
export async function disconnectMCPServer(serverName: string): Promise<void> {
  await apiClient.delete<void>(`/auth/mcp/${serverName}/disconnect`)
}
