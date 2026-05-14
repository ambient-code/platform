import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { gitlabAdapter } from '../adapters/gitlab'
import type { GitLabPort } from '../ports/gitlab'
import { BACKEND_VERSION } from './query-keys'

export const gitlabKeys = {
  all: [BACKEND_VERSION, 'gitlab'] as const,
  status: () => [...gitlabKeys.all, 'status'] as const,
};

export function useGitLabStatus(port: GitLabPort = gitlabAdapter) {
  return useQuery({
    queryKey: gitlabKeys.status(),
    queryFn: () => port.getGitLabStatus(),
  })
}

export function useConnectGitLab(port: GitLabPort = gitlabAdapter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: port.connectGitLab,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: gitlabKeys.status() })
    },
  })
}

export function useDisconnectGitLab(port: GitLabPort = gitlabAdapter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: port.disconnectGitLab,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: gitlabKeys.status() })
    },
  })
}
