import { useMutation, useQueryClient } from '@tanstack/react-query'
import { coderabbitAdapter } from '../adapters/coderabbit'
import type { CodeRabbitPort } from '../ports/coderabbit'

export function useConnectCodeRabbit(port: CodeRabbitPort = coderabbitAdapter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: port.connectCodeRabbit,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['coderabbit', 'status'] })
      queryClient.invalidateQueries({ queryKey: ['integrations', 'status'] })
    },
  })
}

export function useDisconnectCodeRabbit(port: CodeRabbitPort = coderabbitAdapter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: port.disconnectCodeRabbit,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['coderabbit', 'status'] })
      queryClient.invalidateQueries({ queryKey: ['integrations', 'status'] })
    },
  })
}
