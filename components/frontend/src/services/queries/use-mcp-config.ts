import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as mcpConfigApi from '../api/mcp-config';

export function useMcpConfig(projectName: string) {
  return useQuery({
    queryKey: ['mcp-config', projectName],
    queryFn: () => mcpConfigApi.getMcpConfig(projectName),
    enabled: !!projectName,
  });
}

export function useUpdateMcpConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      config,
    }: {
      projectName: string;
      config: mcpConfigApi.McpConfigData;
    }) => mcpConfigApi.updateMcpConfig(projectName, config),
    onSuccess: (_, { projectName }) => {
      queryClient.invalidateQueries({ queryKey: ['mcp-config', projectName] });
    },
  });
}

export function useTestMcpServer() {
  return useMutation({
    mutationFn: ({
      projectName,
      config,
    }: {
      projectName: string;
      config: mcpConfigApi.McpServerConfig;
    }) => mcpConfigApi.testMcpServer(projectName, config),
  });
}
