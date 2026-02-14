import { apiClient } from './client';

export type McpServerConfig = {
  command: string;
  args: string[];
  env: Record<string, string>;
};

export type McpConfigData = {
  servers: Record<string, McpServerConfig>;
};

export type McpTestResult = {
  valid: boolean;
  serverInfo?: { name?: string; version?: string };
  error?: string;
};

export async function getMcpConfig(projectName: string): Promise<McpConfigData> {
  return apiClient.get<McpConfigData>(`/projects/${projectName}/mcp-config`);
}

export async function updateMcpConfig(projectName: string, config: McpConfigData): Promise<void> {
  await apiClient.put<void, McpConfigData>(`/projects/${projectName}/mcp-config`, config);
}

export async function testMcpServer(
  projectName: string,
  config: McpServerConfig,
): Promise<McpTestResult> {
  return apiClient.post<McpTestResult, McpServerConfig>(
    `/projects/${projectName}/mcp-config/test`,
    config,
  );
}
