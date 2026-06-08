import type { DomainSessionMessage, ListParams, PaginatedResult } from '@/domain/types'

export type SessionMessagesPort = {
  send: (sessionId: string, message: { eventType: string; payload: string }) => Promise<DomainSessionMessage>
  list: (sessionId: string, params?: ListParams) => Promise<PaginatedResult<DomainSessionMessage>>
}
