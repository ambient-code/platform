/**
 * Cluster information hook
 * Detects cluster type (OpenShift vs vanilla Kubernetes) by calling the /api/cluster-info endpoint
 */

import { useClusterInfo as useClusterInfoQuery } from '@/services/queries/use-cluster';
import type { ModelInfo } from '@/services/api/cluster';

export type ClusterInfo = {
  isOpenShift: boolean;
  vertexEnabled: boolean;
  models: ModelInfo[];
  isLoading: boolean;
  isError: boolean;
};

/**
 * Detects whether the cluster is OpenShift or vanilla Kubernetes,
 * whether Vertex AI is enabled, and available models
 * Calls the /api/cluster-info endpoint which checks for project.openshift.io API group,
 * CLAUDE_CODE_USE_VERTEX environment variable, and MODELS_JSON ConfigMap
 */
export function useClusterInfo(): ClusterInfo {
  const { data, isLoading, isError } = useClusterInfoQuery();

  return {
    isOpenShift: data?.isOpenShift ?? false,
    vertexEnabled: data?.vertexEnabled ?? false,
    models: data?.models ?? [],
    isLoading,
    isError,
  };
}

