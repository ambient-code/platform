'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type { RoleBindingsPort } from '@/ports/role-bindings'
import type { DomainRoleBinding, DomainRoleBindingCreateRequest, ListParams } from '@/domain/types'
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

/**
 * Fetches all role bindings matching a search filter by paging through all
 * results. This avoids silently dropping bindings beyond a single page limit.
 */
export function useAllRoleBindings(search?: string, port?: RoleBindingsPort) {
  const adapter = port ?? getDefaultPort()
  return useQuery({
    queryKey: [...queryKeys.roleBindings.all, 'all-pages', search],
    queryFn: async () => {
      const allItems: DomainRoleBinding[] = []
      let page = 1
      const size = 100
      const maxPages = 50
      while (page <= maxPages) {
        const result = await adapter.list({ page, size, search })
        allItems.push(...result.items)
        if (!result.hasMore) break
        page++
      }
      return allItems
    },
    enabled: search !== undefined,
  })
}
