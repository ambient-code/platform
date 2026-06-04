import { RoleAPI } from 'ambient-sdk'
import type { Role } from 'ambient-sdk'
import type { RolesPort, DomainRole } from '@/ports/roles'
import type { ListParams, PaginatedResult } from '@/domain/types'
import { getConfig } from './sdk-client'

function getAPI(): RoleAPI {
  return new RoleAPI(getConfig())
}

function mapSdkRoleToDomain(sdk: Role): DomainRole {
  return {
    id: sdk.id,
    name: sdk.name,
    displayName: sdk.display_name,
    description: sdk.description,
    builtIn: sdk.built_in,
    permissions: sdk.permissions,
  }
}

function buildSdkListOptions(params?: ListParams) {
  const page = Math.max(1, params?.page ?? 1)
  const size = Math.min(100, Math.max(1, params?.size ?? 100))
  return {
    page,
    size,
    search: params?.search,
    orderBy: params?.orderBy,
  }
}

export function createRolesAdapter(): RolesPort {
  return {
    async list(params?: ListParams): Promise<PaginatedResult<DomainRole>> {
      const api = getAPI()
      const opts = buildSdkListOptions(params)
      const result = await api.list(opts)
      const page = opts.page
      const size = opts.size
      return {
        items: result.items.map(mapSdkRoleToDomain),
        total: result.total,
        page,
        size,
        hasMore: page * size < result.total,
      }
    },
  }
}
