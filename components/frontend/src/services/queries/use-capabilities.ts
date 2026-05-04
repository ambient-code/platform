import { useQuery } from '@tanstack/react-query';
import { sessionCapabilitiesAdapter } from '../adapters/session-capabilities';
import type { SessionCapabilitiesPort } from '../ports/session-capabilities';
import { BACKEND_VERSION } from './query-keys';

export const capabilitiesKeys = {
  all: [BACKEND_VERSION, 'capabilities'] as const,
  session: (projectName: string, sessionName: string) =>
    [...capabilitiesKeys.all, projectName, sessionName] as const,
};

export function useCapabilities(
  projectName: string,
  sessionName: string,
  enabled: boolean = true,
  port: SessionCapabilitiesPort = sessionCapabilitiesAdapter,
) {
  return useQuery({
    queryKey: capabilitiesKeys.session(projectName, sessionName),
    queryFn: () => port.getCapabilities(projectName, sessionName),
    enabled: enabled && !!projectName && !!sessionName,
    staleTime: 60 * 1000,
    retry: 2,
    refetchInterval: (query) => {
      if (query.state.data?.framework && query.state.data.framework !== 'unknown') {
        return false;
      }
      const updatedCount =
        (query.state as { dataUpdatedCount?: number }).dataUpdatedCount ?? 0;
      if (updatedCount >= 6) return false;
      return 10 * 1000;
    },
  });
}
