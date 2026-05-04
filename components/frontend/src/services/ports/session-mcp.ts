import type { AgenticSession, McpStatusResponse, MCPServersConfig } from './types'

export type SessionMcpPort = {
  getMcpStatus: (projectName: string, sessionName: string) => Promise<McpStatusResponse>
  updateSessionMcpServers: (projectName: string, sessionName: string, mcpServers: MCPServersConfig) => Promise<AgenticSession>
}
