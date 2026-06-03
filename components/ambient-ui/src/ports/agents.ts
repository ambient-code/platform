import type { DomainAgent, ListParams, PaginatedResult } from '@/domain/types'

export type AgentsPort = {
  list: (projectId: string, params?: ListParams) => Promise<PaginatedResult<DomainAgent>>
  get: (agentId: string) => Promise<DomainAgent>
}
