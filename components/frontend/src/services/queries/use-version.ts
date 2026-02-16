/**
 * React Query hooks for version
 */

import { useQuery } from '@tanstack/react-query';
import * as versionApi from '../api/version';

/**
 * Query keys for version
 */
export const versionKeys = {
  all: ['version'] as const,
  current: () => [...versionKeys.all, 'current'] as const,
};

/**
 * Hook to fetch application version
 */
export function useVersion(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: versionKeys.current(),
    queryFn: versionApi.getVersion,
    staleTime: 5 * 60 * 1000,
    retry: false,
    enabled: options?.enabled,
  });
}
