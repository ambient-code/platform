import { useQuery } from '@tanstack/react-query'
import * as integrationsApi from '../api/integrations'

/**
 * Hook to fetch unified integrations status
 */
export function useIntegrationsStatus() {
  return useQuery({
    queryKey: ['integrations', 'status'],
    queryFn: () => integrationsApi.getIntegrationsStatus(),
    staleTime: 30 * 1000, // 30 seconds
  })
}
