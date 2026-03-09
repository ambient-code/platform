/**
 * Config API service
 * Handles runtime configuration endpoints
 */

import { apiClient } from './client';

export type LoadingTipsResponse = {
  tips: string[];
};

/**
 * Get loading tips from runtime configuration
 * Falls back to defaults if not configured
 */
export async function getLoadingTips(): Promise<LoadingTipsResponse> {
  return apiClient.get<LoadingTipsResponse>('/config/loading-tips');
}
