import { apiClient } from './client';

export type McpServerConfig = {
  command: string;
  args: string[];
  env: Record<string, string>;
};

export type McpConfigData = {
  servers: Record<string, McpServerConfig>;
};

export async function getMcpConfig(projectName: string): Promise<McpConfigData> {
  return apiClient.get<McpConfigData>(`/projects/${projectName}/mcp-config`);
}

export async function updateMcpConfig(projectName: string, config: McpConfigData): Promise<void> {
  await apiClient.put<void, McpConfigData>(`/projects/${projectName}/mcp-config`, config);
}
