import type { SessionAPI, SessionCreateRequest } from 'ambient-sdk'
import type { SessionsPort } from '@/ports/sessions'
import type { DomainSession, DomainSessionCreateRequest, ListParams, PaginatedResult } from '@/domain/types'
import { mapSdkSessionToDomain } from './mappers'
import { getSessionAPI } from './sdk-client'

function sanitizeSearch(value: string): string {
  return value.replace(/['"%;\\]/g, '')
}

function buildSdkListOptions(projectId: string, params?: ListParams) {
  const search = params?.search
    ? `project_id = '${sanitizeSearch(projectId)}' and name like '%${sanitizeSearch(params.search)}%'`
    : `project_id = '${sanitizeSearch(projectId)}'`

  return {
    page: params?.page ?? 1,
    size: params?.size ?? 20,
    search,
    orderBy: params?.orderBy,
  }
}

function mapDomainCreateToSdk(request: DomainSessionCreateRequest): SessionCreateRequest {
  const sdkReq: SessionCreateRequest = {
    name: request.name,
    project_id: request.projectId,
  }
  if (request.agentId) sdkReq.agent_id = request.agentId
  if (request.prompt) sdkReq.prompt = request.prompt
  if (request.model) sdkReq.llm_model = request.model
  if (request.temperature !== undefined) sdkReq.llm_temperature = request.temperature
  if (request.maxTokens !== undefined) sdkReq.llm_max_tokens = request.maxTokens
  if (request.timeout !== undefined) sdkReq.timeout = request.timeout
  return sdkReq
}

function createSdkSessionsAdapter(api: SessionAPI): SessionsPort {
  return {
    async list(projectId: string, params?: ListParams): Promise<PaginatedResult<DomainSession>> {
      const opts = buildSdkListOptions(projectId, params)
      const result = await api.list(opts)
      const items = result.items.map(mapSdkSessionToDomain)
      const page = opts.page
      const size = opts.size
      return {
        items,
        total: result.total,
        page,
        size,
        hasMore: page * size < result.total,
      }
    },

    async get(sessionId: string): Promise<DomainSession> {
      const session = await api.get(sessionId)
      return mapSdkSessionToDomain(session)
    },

    async create(request: DomainSessionCreateRequest): Promise<DomainSession> {
      const sdkReq = mapDomainCreateToSdk(request)
      const session = await api.create(sdkReq)
      return mapSdkSessionToDomain(session)
    },

    async stop(sessionId: string): Promise<void> {
      await api.stop(sessionId)
    },

    async start(sessionId: string): Promise<DomainSession> {
      const session = await api.start(sessionId)
      return mapSdkSessionToDomain(session)
    },

    async delete(sessionId: string): Promise<void> {
      await api.delete(sessionId)
    },
  }
}

export function createSessionsAdapter(api?: SessionAPI): SessionsPort {
  return createSdkSessionsAdapter(api ?? getSessionAPI())
}
