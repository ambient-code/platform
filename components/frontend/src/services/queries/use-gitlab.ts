import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { gitlabAdapter } from '../adapters/gitlab'
import type { GitLabPort } from '../ports/gitlab'

export function useGitLabStatus(port: GitLabPort = gitlabAdapter) {
  return useQuery({
    queryKey: ['gitlab', 'status'],
    queryFn: () => port.getGitLabStatus(),
  })
}

export function useConnectGitLab(port: GitLabPort = gitlabAdapter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: port.connectGitLab,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gitlab', 'status'] })
    },
  })
}

export function useDisconnectGitLab(port: GitLabPort = gitlabAdapter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: port.disconnectGitLab,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gitlab', 'status'] })
    },
  })
}
