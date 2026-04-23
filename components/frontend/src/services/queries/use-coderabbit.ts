import { useMutation, useQueryClient } from '@tanstack/react-query'
import * as codeRabbitAuthApi from '../api/coderabbit-auth'

export function useConnectCodeRabbit() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: codeRabbitAuthApi.connectCodeRabbit,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['coderabbit', 'status'] })
      queryClient.invalidateQueries({ queryKey: ['integrations', 'status'] })
    },
  })
}

export function useDisconnectCodeRabbit() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: codeRabbitAuthApi.disconnectCodeRabbit,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['coderabbit', 'status'] })
      queryClient.invalidateQueries({ queryKey: ['integrations', 'status'] })
    },
  })
}
