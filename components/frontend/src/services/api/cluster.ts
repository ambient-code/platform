/**
 * Cluster API service
 * Handles cluster information and detection
 */

import { apiClient } from './client';

export type ModelInfo = {
  name: string;
  displayName: string;
  vertexId?: string;
  default?: boolean;
};

export type ClusterInfo = {
  isOpenShift: boolean;
  vertexEnabled: boolean;
  models: ModelInfo[];
};

/**
 * Get cluster information (OpenShift vs vanilla Kubernetes, Vertex AI status)
 * This endpoint does not require authentication
 */
export async function getClusterInfo(): Promise<ClusterInfo> {
  return apiClient.get<ClusterInfo>('/cluster-info');
}

