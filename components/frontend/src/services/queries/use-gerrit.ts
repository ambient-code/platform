import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { gerritAdapter } from '../adapters/gerrit'
import type { GerritPort } from '../ports/gerrit'
import { BACKEND_VERSION } from './query-keys'
import { integrationsKeys } from './use-integrations'

export const gerritKeys = {
  all: [BACKEND_VERSION, 'gerrit'] as const,
  instances: () => [...gerritKeys.all, 'instances'] as const,
};

export function useGerritInstances(port: GerritPort = gerritAdapter) {
  return useQuery({
    queryKey: gerritKeys.instances(),
    queryFn: () => port.getGerritInstances(),
  })
}

export function useConnectGerrit(port: GerritPort = gerritAdapter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: port.connectGerrit,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: integrationsKeys.status() })
      queryClient.invalidateQueries({ queryKey: gerritKeys.instances() })
    },
  })
}

export function useDisconnectGerrit(port: GerritPort = gerritAdapter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: port.disconnectGerrit,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: integrationsKeys.status() })
      queryClient.invalidateQueries({ queryKey: gerritKeys.instances() })
    },
  })
}

export function useTestGerritConnection(port: GerritPort = gerritAdapter) {
  return useMutation({
    mutationFn: port.testGerritConnection,
  })
}
