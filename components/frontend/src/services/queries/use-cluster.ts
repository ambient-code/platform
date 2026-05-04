import { useQuery } from '@tanstack/react-query';
import { clusterAdapter } from '../adapters/cluster';
import type { ClusterPort } from '../ports/cluster';
import { BACKEND_VERSION } from './query-keys';

export const clusterKeys = {
  all: [BACKEND_VERSION, 'cluster'] as const,
  info: () => [...clusterKeys.all, 'info'] as const,
};

export function useClusterInfo(port: ClusterPort = clusterAdapter) {
  return useQuery({
    queryKey: clusterKeys.info(),
    queryFn: port.getClusterInfo,
    staleTime: Infinity,
    retry: 3,
  });
}
