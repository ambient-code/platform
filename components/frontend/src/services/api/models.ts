/**
 * Models API service
 * Handles model listing API calls
 */

import { apiClient } from './client';
import type { ListModelsResponse } from '@/types/api';

/**
 * Get available models for a project (workspace-aware, checks overrides)
 */
export async function getModelsForProject(projectName: string): Promise<ListModelsResponse> {
  return apiClient.get<ListModelsResponse>(`/projects/${projectName}/models`);
}
