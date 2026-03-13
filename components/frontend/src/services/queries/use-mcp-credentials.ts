import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as mcpCredentialsApi from '../api/mcp-credentials'

export function useMCPServerStatus(serverName: string) {
  return useQuery({
    queryKey: ['mcp-credentials', serverName, 'status'],
    queryFn: () => mcpCredentialsApi.getMCPServerStatus(serverName),
    enabled: !!serverName,
  })
}

export function useConnectMCPServer(serverName: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: mcpCredentialsApi.MCPConnectRequest) =>
      mcpCredentialsApi.connectMCPServer(serverName, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['mcp-credentials', serverName, 'status'] })
      queryClient.invalidateQueries({ queryKey: ['integrations', 'status'] })
    },
  })
}

export function useDisconnectMCPServer(serverName: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => mcpCredentialsApi.disconnectMCPServer(serverName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['mcp-credentials', serverName, 'status'] })
      queryClient.invalidateQueries({ queryKey: ['integrations', 'status'] })
    },
  })
}
