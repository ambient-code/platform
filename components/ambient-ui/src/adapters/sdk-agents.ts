import { AgentAPI } from 'ambient-sdk'
import type { AgentCreateRequest, AgentPatchRequest } from 'ambient-sdk'
import type { AgentsPort } from '@/ports/agents'
import type {
  DomainAgent,
  DomainAgentCreateRequest,
  DomainAgentUpdateRequest,
  ListParams,
  PaginatedResult,
} from '@/domain/types'
import { mapSdkAgentToDomain } from './mappers'
import { getConfig } from './sdk-client'

function sanitizeSearch(value: string): string {
  return value.replace(/['"%;\\]/g, '')
}

function getProjectScopedAPI(projectId: string): AgentAPI {
  return new AgentAPI({ ...getConfig(), project: projectId })
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

function mapDomainCreateToSdk(request: DomainAgentCreateRequest): AgentCreateRequest {
  const sdkReq: AgentCreateRequest = {
    name: request.name,
    project_id: request.projectId,
  }
  if (request.displayName) sdkReq.display_name = request.displayName
  if (request.model) sdkReq.llm_model = request.model
  if (request.prompt) sdkReq.prompt = request.prompt
  if (request.repoUrl) sdkReq.repo_url = request.repoUrl
  if (request.description) sdkReq.description = request.description
  return sdkReq
}

function mapDomainUpdateToSdk(request: DomainAgentUpdateRequest): AgentPatchRequest {
  const sdkReq: AgentPatchRequest = {}
  if (request.displayName !== undefined) sdkReq.display_name = request.displayName
  if (request.model !== undefined) sdkReq.llm_model = request.model
  if (request.prompt !== undefined) sdkReq.prompt = request.prompt
  if (request.repoUrl !== undefined) sdkReq.repo_url = request.repoUrl
  if (request.description !== undefined) sdkReq.description = request.description
  return sdkReq
}

export function createAgentsAdapter(): AgentsPort {
  return {
    async list(projectId: string, params?: ListParams): Promise<PaginatedResult<DomainAgent>> {
      const api = getProjectScopedAPI(projectId)
      const opts = buildSdkListOptions(params)
      const result = await api.list(opts)
      const page = opts.page
      const size = opts.size
      return {
        items: result.items.map(mapSdkAgentToDomain),
        total: result.total,
        page,
        size,
        hasMore: page * size < result.total,
      }
    },

    async get(agentId: string): Promise<DomainAgent> {
      const api = new AgentAPI({ ...getConfig(), project: '_' })
      const agent = await api.get(agentId)
      return mapSdkAgentToDomain(agent)
    },

    async create(projectId: string, request: DomainAgentCreateRequest): Promise<DomainAgent> {
      const api = getProjectScopedAPI(projectId)
      const sdkReq = mapDomainCreateToSdk(request)
      const agent = await api.create(sdkReq)
      return mapSdkAgentToDomain(agent)
    },

    async update(agentId: string, request: DomainAgentUpdateRequest): Promise<DomainAgent> {
      const api = new AgentAPI({ ...getConfig(), project: '_' })
      const sdkReq = mapDomainUpdateToSdk(request)
      const agent = await api.update(agentId, sdkReq)
      return mapSdkAgentToDomain(agent)
    },

    async delete(agentId: string): Promise<void> {
      const api = new AgentAPI({ ...getConfig(), project: '_' })
      await api.delete(agentId)
    },
  }
}
