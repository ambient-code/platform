import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { sessionMcpAdapter } from '../adapters/session-mcp';
import type { SessionMcpPort } from '../ports/session-mcp';
import type { MCPServersConfig } from '@/types/agentic-session';
import { BACKEND_VERSION } from './query-keys';

export const mcpKeys = {
  all: [BACKEND_VERSION, 'mcp'] as const,
  status: (projectName: string, sessionName: string) =>
    [...mcpKeys.all, 'status', projectName, sessionName] as const,
};

export function useMcpStatus(
  projectName: string,
  sessionName: string,
  enabled: boolean = true,
  port: SessionMcpPort = sessionMcpAdapter,
) {
  return useQuery({
    queryKey: mcpKeys.status(projectName, sessionName),
    queryFn: () => port.getMcpStatus(projectName, sessionName),
    enabled: enabled && !!projectName && !!sessionName,
    staleTime: 30 * 1000,
    retry: false,
    refetchInterval: (query) => {
      const servers = query.state.data?.servers
      if (servers && servers.length > 0) return false
      const updatedCount = (query.state as { dataUpdatedCount?: number }).dataUpdatedCount ?? 0
      if (updatedCount >= 12) return false
      return 10 * 1000
    },
  });
}

export function useUpdateSessionMcpServers(
  projectName: string,
  sessionName: string,
  port: SessionMcpPort = sessionMcpAdapter,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (mcpServers: MCPServersConfig) =>
      port.updateSessionMcpServers(projectName, sessionName, mcpServers),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: mcpKeys.status(projectName, sessionName),
      });
    },
  });
}
