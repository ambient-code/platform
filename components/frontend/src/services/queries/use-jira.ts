import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { jiraAdapter } from '../adapters/jira'
import type { JiraPort } from '../ports/jira'

export function useJiraStatus(port: JiraPort = jiraAdapter) {
  return useQuery({
    queryKey: ['jira', 'status'],
    queryFn: () => port.getJiraStatus(),
  })
}

export function useConnectJira(port: JiraPort = jiraAdapter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: port.connectJira,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jira', 'status'] })
    },
  })
}

export function useDisconnectJira(port: JiraPort = jiraAdapter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: port.disconnectJira,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jira', 'status'] })
    },
  })
}
