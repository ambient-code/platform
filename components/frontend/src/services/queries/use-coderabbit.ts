import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as coderabbitAuthApi from '../api/coderabbit-auth'

export function useCodeRabbitStatus() {
  return useQuery({
    queryKey: ['coderabbit', 'status'],
    queryFn: () => coderabbitAuthApi.getCodeRabbitStatus(),
  })
}

export function useConnectCodeRabbit() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: coderabbitAuthApi.connectCodeRabbit,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['coderabbit', 'status'] })
    },
  })
}

export function useDisconnectCodeRabbit() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: coderabbitAuthApi.disconnectCodeRabbit,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['coderabbit', 'status'] })
    },
  })
}
