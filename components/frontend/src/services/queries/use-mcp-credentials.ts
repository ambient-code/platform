import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { mcpCredentialsAdapter } from '../adapters/mcp-credentials'
import type { McpCredentialsPort } from '../ports/mcp-credentials'
import type { MCPConnectRequest } from '../ports/types'

export function useMCPServerStatus(serverName: string, port: McpCredentialsPort = mcpCredentialsAdapter) {
  return useQuery({
    queryKey: ['mcp-credentials', serverName, 'status'],
    queryFn: () => port.getMCPServerStatus(serverName),
    enabled: !!serverName,
  })
}

export function useConnectMCPServer(serverName: string, port: McpCredentialsPort = mcpCredentialsAdapter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: MCPConnectRequest) =>
      port.connectMCPServer(serverName, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['mcp-credentials', serverName, 'status'] })
      queryClient.invalidateQueries({ queryKey: ['integrations', 'status'] })
    },
  })
}

export function useDisconnectMCPServer(serverName: string, port: McpCredentialsPort = mcpCredentialsAdapter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => port.disconnectMCPServer(serverName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['mcp-credentials', serverName, 'status'] })
      queryClient.invalidateQueries({ queryKey: ['integrations', 'status'] })
    },
  })
}
