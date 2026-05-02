import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { mcpCredentialsAdapter } from '../adapters/mcp-credentials'
import type { McpCredentialsPort } from '../ports/mcp-credentials'
import type { MCPConnectRequest } from '../ports/types'
import { BACKEND_VERSION } from './query-keys'
import { integrationsKeys } from './use-integrations'

export const mcpCredentialsKeys = {
  all: [BACKEND_VERSION, 'mcp-credentials'] as const,
  status: (serverName: string) => [...mcpCredentialsKeys.all, serverName, 'status'] as const,
};

export function useMCPServerStatus(serverName: string, port: McpCredentialsPort = mcpCredentialsAdapter) {
  return useQuery({
    queryKey: mcpCredentialsKeys.status(serverName),
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
      queryClient.invalidateQueries({ queryKey: mcpCredentialsKeys.status(serverName) })
      queryClient.invalidateQueries({ queryKey: integrationsKeys.status() })
    },
  })
}

export function useDisconnectMCPServer(serverName: string, port: McpCredentialsPort = mcpCredentialsAdapter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => port.disconnectMCPServer(serverName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: mcpCredentialsKeys.status(serverName) })
      queryClient.invalidateQueries({ queryKey: integrationsKeys.status() })
    },
  })
}
