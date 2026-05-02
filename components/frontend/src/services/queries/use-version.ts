import { useQuery } from '@tanstack/react-query';
import { versionAdapter } from '../adapters/version';
import type { VersionPort } from '../ports/version';
import { BACKEND_VERSION } from './query-keys';

export const versionKeys = {
  all: [BACKEND_VERSION, 'version'] as const,
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
