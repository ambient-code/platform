import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as jiraAuthApi from '../api/jira-auth'

export function useJiraStatus() {
  return useQuery({
    queryKey: ['jira', 'status'],
    queryFn: () => jiraAuthApi.getJiraStatus(),
  })
}

export function useConnectJira() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: jiraAuthApi.connectJira,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jira', 'status'] })
    },
  })
}

export function useDisconnectJira() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: jiraAuthApi.disconnectJira,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jira', 'status'] })
    },
  })
}
