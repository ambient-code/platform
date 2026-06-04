'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type { RoleBindingsPort } from '@/ports/role-bindings'
import type { DomainRoleBindingCreateRequest, ListParams } from '@/domain/types'
import { createRoleBindingsAdapter } from '@/adapters/sdk-role-bindings'
import { queryKeys } from './query-keys'

let defaultPort: RoleBindingsPort | null = null

function getDefaultPort(): RoleBindingsPort {
  if (!defaultPort) {
    defaultPort = createRoleBindingsAdapter()
  }
  return defaultPort
}

export function useRoleBindings(
  params?: ListParams,
  port?: RoleBindingsPort,
) {
  const adapter = port ?? getDefaultPort()
  return useQuery({
    queryKey: queryKeys.roleBindings.list(params),
    queryFn: () => adapter.list(params),
  })
}

export function useCreateRoleBinding(port?: RoleBindingsPort) {
  const queryClient = useQueryClient()
  const adapter = port ?? getDefaultPort()

  return useMutation({
    mutationFn: (request: DomainRoleBindingCreateRequest) =>
      adapter.create(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.roleBindings.all })
    },
  })
}

export function useDeleteRoleBinding(port?: RoleBindingsPort) {
  const queryClient = useQueryClient()
  const adapter = port ?? getDefaultPort()

  return useMutation({
    mutationFn: (id: string) =>
      adapter.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.roleBindings.all })
    },
  })
}
