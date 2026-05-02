import { useQuery } from '@tanstack/react-query'
import { integrationsAdapter } from '../adapters/integrations'
import type { IntegrationsPort } from '../ports/integrations'
import { BACKEND_VERSION } from './query-keys'

export const integrationsKeys = {
  all: [BACKEND_VERSION, 'integrations'] as const,
  status: () => [...integrationsKeys.all, 'status'] as const,
};

export function useIntegrationsStatus(port: IntegrationsPort = integrationsAdapter) {
  return useQuery({
    queryKey: integrationsKeys.status(),
    queryFn: () => port.getIntegrationsStatus(),
    staleTime: 30 * 1000,
  })
}
