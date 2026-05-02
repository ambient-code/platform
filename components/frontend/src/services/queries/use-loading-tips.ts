import { useQuery } from '@tanstack/react-query';
import { configAdapter } from '../adapters/config';
import type { ConfigPort } from '../ports/config';
import { BACKEND_VERSION } from './query-keys';

export const configKeys = {
  all: [BACKEND_VERSION, 'config'] as const,
  loadingTips: () => [...configKeys.all, 'loading-tips'] as const,
};

export function useLoadingTips(port: ConfigPort = configAdapter) {
  return useQuery({
    queryKey: configKeys.loadingTips(),
    queryFn: port.getLoadingTips,
    staleTime: Infinity,
    gcTime: Infinity,
    retry: 1,
  });
}
