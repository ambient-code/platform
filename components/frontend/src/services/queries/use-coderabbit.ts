import { useMutation, useQueryClient } from '@tanstack/react-query'
import { coderabbitAdapter } from '../adapters/coderabbit'
import type { CodeRabbitPort } from '../ports/coderabbit'
import { BACKEND_VERSION } from './query-keys'
import { integrationsKeys } from './use-integrations'

export const coderabbitKeys = {
  all: [BACKEND_VERSION, 'coderabbit'] as const,
  status: () => [...coderabbitKeys.all, 'status'] as const,
};

export function useConnectCodeRabbit(port: CodeRabbitPort = coderabbitAdapter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: port.connectCodeRabbit,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: coderabbitKeys.status() })
      queryClient.invalidateQueries({ queryKey: integrationsKeys.status() })
    },
  })
}

export function useDisconnectCodeRabbit(port: CodeRabbitPort = coderabbitAdapter) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: port.disconnectCodeRabbit,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: coderabbitKeys.status() })
      queryClient.invalidateQueries({ queryKey: integrationsKeys.status() })
    },
  })
}
