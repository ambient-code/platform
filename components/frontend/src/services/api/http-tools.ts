import { apiClient } from './client';

export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';

export type HttpToolConfig = {
  name: string;
  description: string;
  method: HttpMethod;
  endpoint: string;
  headers: Record<string, string>;
  params: Record<string, string>;
};

export type HttpToolsData = {
  tools: HttpToolConfig[];
};

export async function getHttpTools(projectName: string): Promise<HttpToolsData> {
  return apiClient.get<HttpToolsData>(`/projects/${projectName}/http-tools`);
}

export async function updateHttpTools(projectName: string, data: HttpToolsData): Promise<void> {
  await apiClient.put<void, HttpToolsData>(`/projects/${projectName}/http-tools`, data);
}
