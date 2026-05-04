import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { jiraAdapter } from '../adapters/jira'
import type { JiraPort } from '../ports/jira'
import { BACKEND_VERSION } from './query-keys'

export const jiraKeys = {
  all: [BACKEND_VERSION, 'jira'] as const,
  status: () => [...jiraKeys.all, 'status'] as const,
};

export function useJiraStatus(port: JiraPort = jiraAdapter) {
  return useQuery({
    queryKey: jiraKeys.status(),
    queryFn: () => port.getJiraStatus(),
  })
}

export function useConnectJira(port: JiraPort = jiraAdapter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: port.connectJira,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: jiraKeys.status() })
    },
  })
}

export function useDisconnectJira(port: JiraPort = jiraAdapter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: port.disconnectJira,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: jiraKeys.status() })
    },
  })
}
