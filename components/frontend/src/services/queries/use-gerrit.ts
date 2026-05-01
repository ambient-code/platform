import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { gerritAdapter } from '../adapters/gerrit'
import type { GerritPort } from '../ports/gerrit'

export function useGerritInstances(port: GerritPort = gerritAdapter) {
  return useQuery({
    queryKey: ['gerrit', 'instances'],
    queryFn: () => port.getGerritInstances(),
  })
}

export function useConnectGerrit(port: GerritPort = gerritAdapter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: port.connectGerrit,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['integrations', 'status'] })
      queryClient.invalidateQueries({ queryKey: ['gerrit', 'instances'] })
    },
  })
}

export function useDisconnectGerrit(port: GerritPort = gerritAdapter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: port.disconnectGerrit,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['integrations', 'status'] })
      queryClient.invalidateQueries({ queryKey: ['gerrit', 'instances'] })
    },
  })
}

export function useTestGerritConnection(port: GerritPort = gerritAdapter) {
  return useMutation({
    mutationFn: port.testGerritConnection,
  })
}
