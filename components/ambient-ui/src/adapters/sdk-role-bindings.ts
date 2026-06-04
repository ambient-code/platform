import { RoleBindingAPI } from 'ambient-sdk'
import type { RoleBindingCreateRequest } from 'ambient-sdk'
import type { RoleBindingsPort } from '@/ports/role-bindings'
import type {
  DomainRoleBinding,
  DomainRoleBindingCreateRequest,
  ListParams,
  PaginatedResult,
} from '@/domain/types'
import { mapSdkRoleBindingToDomain } from './mappers'
import { getConfig } from './sdk-client'

function sanitizeId(value: string): string {
  return value.replace(/[^a-zA-Z0-9_-]/g, '')
}

function sanitizeSearch(value: string): string {
  return value.replace(/['"%;\\]/g, '')
}

function getAPI(): RoleBindingAPI {
  return new RoleBindingAPI(getConfig())
}

function buildSdkListOptions(params?: ListParams) {
  const page = Math.max(1, params?.page ?? 1)
  const size = Math.min(100, Math.max(1, params?.size ?? 100))
  return {
    page,
    size,
    search: params?.search ?? undefined,
    orderBy: params?.orderBy,
  }
}

function mapDomainCreateToSdk(request: DomainRoleBindingCreateRequest): RoleBindingCreateRequest {
  const sdkReq: RoleBindingCreateRequest = {
    role_id: request.roleId,
    scope: request.scope,
  }
  if (request.userId) sdkReq.user_id = request.userId
  if (request.projectId) sdkReq.project_id = request.projectId
  if (request.agentId) sdkReq.agent_id = request.agentId
  if (request.credentialId) sdkReq.credential_id = request.credentialId
  if (request.sessionId) sdkReq.session_id = request.sessionId
  return sdkReq
}

export function createRoleBindingsAdapter(): RoleBindingsPort {
  return {
    async list(params?: ListParams): Promise<PaginatedResult<DomainRoleBinding>> {
      const api = getAPI()
      const opts = buildSdkListOptions(params)
      const result = await api.list(opts)
      const page = opts.page
      const size = opts.size
      return {
        items: result.items.map(mapSdkRoleBindingToDomain),
        total: result.total,
        page,
        size,
        hasMore: page * size < result.total,
      }
    },

    async create(request: DomainRoleBindingCreateRequest): Promise<DomainRoleBinding> {
      const api = getAPI()
      const sdkReq = mapDomainCreateToSdk(request)
      const roleBinding = await api.create(sdkReq)
      return mapSdkRoleBindingToDomain(roleBinding)
    },

    async delete(id: string): Promise<void> {
      const api = getAPI()
      await api.delete(id)
    },
  }
}
