import { useQuery } from '@tanstack/react-query'
import { integrationsAdapter } from '../adapters/integrations'
import type { IntegrationsPort } from '../ports/integrations'

export function useIntegrationsStatus(port: IntegrationsPort = integrationsAdapter) {
  return useQuery({
    queryKey: ['integrations', 'status'],
    queryFn: () => port.getIntegrationsStatus(),
    staleTime: 30 * 1000,
  })
}
