import { CredentialAPI } from 'ambient-sdk'
import type { CredentialCreateRequest, CredentialPatchRequest } from 'ambient-sdk'
import type { CredentialsPort } from '@/ports/credentials'
import type {
  DomainCredential,
  DomainCredentialCreateRequest,
  DomainCredentialUpdateRequest,
  ListParams,
  PaginatedResult,
} from '@/domain/types'
import { mapSdkCredentialToDomain } from './mappers'
import { getConfig } from './sdk-client'

function sanitizeSearch(value: string): string {
  return value.replace(/['"%;\\]/g, '')
}

function getAPI(): CredentialAPI {
  return new CredentialAPI(getConfig())
}

function buildSdkListOptions(params?: ListParams) {
  return {
    page: params?.page ?? 1,
    size: params?.size ?? 20,
    search: params?.search
      ? `name like '%${sanitizeSearch(params.search)}%'`
      : undefined,
    orderBy: params?.orderBy,
  }
}

function mapDomainCreateToSdk(request: DomainCredentialCreateRequest): CredentialCreateRequest {
  const sdkReq: CredentialCreateRequest = {
    name: request.name,
    provider: request.provider,
  }
  if (request.description) sdkReq.description = request.description
  if (request.email) sdkReq.email = request.email
  if (request.url) sdkReq.url = request.url
  if (request.token) sdkReq.token = request.token
  return sdkReq
}

function mapDomainUpdateToSdk(request: DomainCredentialUpdateRequest): CredentialPatchRequest {
  const sdkReq: CredentialPatchRequest = {}
  if (request.name !== undefined) sdkReq.name = request.name
  if (request.description !== undefined) sdkReq.description = request.description
  if (request.email !== undefined) sdkReq.email = request.email
  if (request.url !== undefined) sdkReq.url = request.url
  if (request.token !== undefined) sdkReq.token = request.token
  return sdkReq
}

export function createCredentialsAdapter(): CredentialsPort {
  return {
    async list(params?: ListParams): Promise<PaginatedResult<DomainCredential>> {
      const api = getAPI()
      const opts = buildSdkListOptions(params)
      const result = await api.list(opts)
      const page = opts.page
      const size = opts.size
      return {
        items: result.items.map(mapSdkCredentialToDomain),
        total: result.total,
        page,
        size,
        hasMore: page * size < result.total,
      }
    },

    async get(id: string): Promise<DomainCredential> {
      const api = getAPI()
      const credential = await api.get(id)
      return mapSdkCredentialToDomain(credential)
    },

    async create(request: DomainCredentialCreateRequest): Promise<DomainCredential> {
      const api = getAPI()
      const sdkReq = mapDomainCreateToSdk(request)
      const credential = await api.create(sdkReq)
      return mapSdkCredentialToDomain(credential)
    },

    async update(id: string, request: DomainCredentialUpdateRequest): Promise<DomainCredential> {
      const api = getAPI()
      const sdkReq = mapDomainUpdateToSdk(request)
      const credential = await api.update(id, sdkReq)
      return mapSdkCredentialToDomain(credential)
    },

    async delete(id: string): Promise<void> {
      const api = getAPI()
      await api.delete(id)
    },
  }
}
