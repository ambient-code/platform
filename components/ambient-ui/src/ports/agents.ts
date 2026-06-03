import type {
  DomainAgent,
  DomainAgentCreateRequest,
  DomainAgentUpdateRequest,
  ListParams,
  PaginatedResult,
} from '@/domain/types'

export type AgentsPort = {
  list: (projectId: string, params?: ListParams) => Promise<PaginatedResult<DomainAgent>>
  get: (agentId: string) => Promise<DomainAgent>
  create: (projectId: string, request: DomainAgentCreateRequest) => Promise<DomainAgent>
  update: (agentId: string, request: DomainAgentUpdateRequest) => Promise<DomainAgent>
  delete: (agentId: string) => Promise<void>
}
