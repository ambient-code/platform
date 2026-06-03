import type { AgentAPI } from 'ambient-sdk'
import type { AgentsPort } from '@/ports/agents'
import type { DomainAgent, ListParams, PaginatedResult } from '@/domain/types'
import { mapSdkAgentToDomain } from './mappers'
import { getAgentAPI } from './sdk-client'

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

function createSdkAgentsAdapter(api: AgentAPI): AgentsPort {
  return {
    async list(projectId: string, params?: ListParams): Promise<PaginatedResult<DomainAgent>> {
      const opts = buildSdkListOptions(projectId, params)
      const result = await api.list(opts)
      const items = result.items.map(mapSdkAgentToDomain)
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

    async get(agentId: string): Promise<DomainAgent> {
      const agent = await api.get(agentId)
      return mapSdkAgentToDomain(agent)
    },
  }
}

export function createAgentsAdapter(api?: AgentAPI): AgentsPort {
  return createSdkAgentsAdapter(api ?? getAgentAPI())
}
