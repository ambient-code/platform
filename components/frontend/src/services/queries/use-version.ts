import { useQuery } from '@tanstack/react-query';
import { versionAdapter } from '../adapters/version';
import type { VersionPort } from '../ports/version';

export const versionKeys = {
  all: ['version'] as const,
  current: () => [...versionKeys.all, 'current'] as const,
};

export function useVersion(port: VersionPort = versionAdapter) {
  return useQuery({
    queryKey: versionKeys.current(),
    queryFn: port.getVersion,
    staleTime: 5 * 60 * 1000,
    retry: false,
  });
}
