'use client'

import { useQuery } from '@tanstack/react-query'
import type { RolesPort } from '@/ports/roles'
import type { ListParams } from '@/domain/types'
import { createRolesAdapter } from '@/adapters/sdk-roles'
import { queryKeys } from './query-keys'

let defaultPort: RolesPort | null = null

function getDefaultPort(): RolesPort {
  if (!defaultPort) {
    defaultPort = createRolesAdapter()
  }
  return defaultPort
}

export function useRoles(
  params?: ListParams,
  port?: RolesPort,
) {
  const adapter = port ?? getDefaultPort()
  return useQuery({
    queryKey: queryKeys.roles.list(params),
    queryFn: () => adapter.list(params),
  })
}
