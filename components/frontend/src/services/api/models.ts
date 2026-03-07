/**
 * Models API service
 * Handles model listing API calls
 */

import { apiClient } from './client';
import type { ListModelsResponse } from '@/types/api';

/**
 * Get available models for a project (workspace-aware, checks overrides).
 * Optionally filter by provider (e.g. "anthropic", "google").
 */
export async function getModelsForProject(projectName: string, provider?: string): Promise<ListModelsResponse> {
  const params = provider ? `?provider=${encodeURIComponent(provider)}` : '';
  return apiClient.get<ListModelsResponse>(`/projects/${projectName}/models${params}`);
}
