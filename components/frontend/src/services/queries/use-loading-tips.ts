/**
 * React Query hook for loading tips
 */

import { useQuery } from '@tanstack/react-query';
import { getLoadingTips } from '@/services/api/config';

/**
 * Hook to get loading tips from runtime configuration
 * Cached indefinitely since tips rarely change
 */
export function useLoadingTips() {
  return useQuery({
    queryKey: ['loading-tips'],
    queryFn: getLoadingTips,
    staleTime: Infinity, // Tips don't change often, cache for session lifetime
    retry: 1, // Only retry once, fall back to defaults on failure
  });
}
