import type { ListParams, PaginatedResult } from '@/domain/types'

export type DomainRole = {
  id: string
  name: string
  displayName: string
  description: string
  builtIn: boolean
  permissions: string
}

export type RolesPort = {
  list: (params?: ListParams) => Promise<PaginatedResult<DomainRole>>
}
