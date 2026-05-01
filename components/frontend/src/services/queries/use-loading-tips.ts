import { useQuery } from '@tanstack/react-query';
import { configAdapter } from '../adapters/config';
import type { ConfigPort } from '../ports/config';

export function useLoadingTips(port: ConfigPort = configAdapter) {
  return useQuery({
    queryKey: ['config', 'loading-tips'],
    queryFn: port.getLoadingTips,
    staleTime: Infinity,
    gcTime: Infinity,
    retry: 1,
  });
}
