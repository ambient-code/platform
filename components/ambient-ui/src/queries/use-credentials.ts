'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type { CredentialsPort } from '@/ports/credentials'
import type { DomainCredentialCreateRequest, DomainCredentialUpdateRequest, ListParams } from '@/domain/types'
import { createCredentialsAdapter } from '@/adapters/sdk-credentials'
import { queryKeys } from './query-keys'

let defaultPort: CredentialsPort | null = null

function getDefaultPort(): CredentialsPort {
  if (!defaultPort) {
    defaultPort = createCredentialsAdapter()
  }
  return defaultPort
}

export function useCredentials(
  params?: ListParams,
  port?: CredentialsPort,
) {
  const adapter = port ?? getDefaultPort()
  return useQuery({
    queryKey: queryKeys.credentials.list(params),
    queryFn: () => adapter.list(params),
  })
}

export function useCredential(
  id: string,
  port?: CredentialsPort,
) {
  const adapter = port ?? getDefaultPort()
  return useQuery({
    queryKey: queryKeys.credentials.detail(id),
    queryFn: () => adapter.get(id),
    enabled: !!id,
  })
}

export function useCreateCredential(port?: CredentialsPort) {
  const queryClient = useQueryClient()
  const adapter = port ?? getDefaultPort()

  return useMutation({
    mutationFn: (request: DomainCredentialCreateRequest) =>
      adapter.create(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.credentials.all })
    },
  })
}

export function useUpdateCredential(port?: CredentialsPort) {
  const queryClient = useQueryClient()
  const adapter = port ?? getDefaultPort()

  return useMutation({
    mutationFn: ({ id, request }: { id: string; request: DomainCredentialUpdateRequest }) =>
      adapter.update(id, request),
    onSuccess: (updatedCredential, { id }) => {
      queryClient.setQueryData(queryKeys.credentials.detail(id), updatedCredential)
      queryClient.invalidateQueries({ queryKey: queryKeys.credentials.lists() })
    },
  })
}

export function useDeleteCredential(port?: CredentialsPort) {
  const queryClient = useQueryClient()
  const adapter = port ?? getDefaultPort()

  return useMutation({
    mutationFn: (id: string) =>
      adapter.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.credentials.all })
    },
  })
}
