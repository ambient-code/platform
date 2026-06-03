import type {
  DomainAgent,
  DomainAgentCreateRequest,
  DomainAgentUpdateRequest,
  ListParams,
  PaginatedResult,
} from '@/domain/types'

export type AgentsPort = {
  list: (projectId: string, params?: ListParams) => Promise<PaginatedResult<DomainAgent>>
  get: (projectId: string, agentId: string) => Promise<DomainAgent>
  create: (projectId: string, request: DomainAgentCreateRequest) => Promise<DomainAgent>
  update: (projectId: string, agentId: string, request: DomainAgentUpdateRequest) => Promise<DomainAgent>
  delete: (projectId: string, agentId: string) => Promise<void>
}
