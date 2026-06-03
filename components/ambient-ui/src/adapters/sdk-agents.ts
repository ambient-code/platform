import { AgentAPI } from 'ambient-sdk'
import type { AgentsPort } from '@/ports/agents'
import type { DomainAgent, ListParams, PaginatedResult } from '@/domain/types'
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
  }
}
