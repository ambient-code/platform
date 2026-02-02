import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as gitlabAuthApi from '../api/gitlab-auth'

export function useGitLabStatus() {
  return useQuery({
    queryKey: ['gitlab', 'status'],
    queryFn: () => gitlabAuthApi.getGitLabStatus(),
  })
}

export function useConnectGitLab() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: gitlabAuthApi.connectGitLab,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gitlab', 'status'] })
    },
  })
}

export function useDisconnectGitLab() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: gitlabAuthApi.disconnectGitLab,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gitlab', 'status'] })
    },
  })
}
