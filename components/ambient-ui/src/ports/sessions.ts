import type { DomainSession, ListParams, PaginatedResult } from '@/domain/types'

export type SessionsPort = {
  list: (projectId: string, params?: ListParams) => Promise<PaginatedResult<DomainSession>>
  get: (sessionId: string) => Promise<DomainSession>
  stop: (sessionId: string) => Promise<void>
  start: (sessionId: string) => Promise<DomainSession>
  delete: (sessionId: string) => Promise<void>
}
