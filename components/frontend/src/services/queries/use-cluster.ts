import { useQuery } from '@tanstack/react-query';
import { clusterAdapter } from '../adapters/cluster';
import type { ClusterPort } from '../ports/cluster';

export function useClusterInfo(port: ClusterPort = clusterAdapter) {
  return useQuery({
    queryKey: ['cluster-info'],
    queryFn: port.getClusterInfo,
    staleTime: Infinity,
    retry: 3,
  });
}
