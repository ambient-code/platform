import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as gerritAuthApi from '../api/gerrit-auth'

export function useGerritInstances() {
  return useQuery({
    queryKey: ['gerrit', 'instances'],
    queryFn: () => gerritAuthApi.getGerritInstances(),
  })
}

export function useConnectGerrit() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: gerritAuthApi.connectGerrit,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['integrations', 'status'] })
      queryClient.invalidateQueries({ queryKey: ['gerrit', 'instances'] })
    },
  })
}

export function useDisconnectGerrit() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: gerritAuthApi.disconnectGerrit,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['integrations', 'status'] })
      queryClient.invalidateQueries({ queryKey: ['gerrit', 'instances'] })
    },
  })
}

export function useTestGerritConnection() {
  return useMutation({
    mutationFn: gerritAuthApi.testGerritConnection,
  })
}
