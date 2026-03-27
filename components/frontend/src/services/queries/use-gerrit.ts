import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as gerritAuthApi from '../api/gerrit-auth'

/** Hook to fetch all connected Gerrit instances. */
export function useGerritInstances() {
  return useQuery({
    queryKey: ['gerrit', 'instances'],
    queryFn: () => gerritAuthApi.getGerritInstances(),
  })
}

/** Hook to connect a new Gerrit instance with credential validation. */
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

/** Hook to disconnect a Gerrit instance and remove its credentials. */
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

/** Hook to test Gerrit credentials without storing them. */
export function useTestGerritConnection() {
  return useMutation({
    mutationFn: gerritAuthApi.testGerritConnection,
  })
}
