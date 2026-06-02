import type { SessionMessagesPort } from '@/ports/session-messages'
import type { DomainSessionMessage, ListParams, PaginatedResult } from '@/domain/types'
import { mapSessionMessageToDomain } from './mappers'
import type { SdkSessionMessageShape } from './mappers'

type SessionMessageResponse = SdkSessionMessageShape & {
  kind: string
  href: string
  updated_at: string | null
}

type SessionMessageListResponse = {
  kind: string
  page: number
  size: number
  total: number
  items: SessionMessageResponse[]
}

export type FetchFn = (input: string, init?: RequestInit) => Promise<Response>

function sanitizeSessionId(value: string): string {
  return encodeURIComponent(value)
}

function buildQueryString(params?: ListParams): string {
  const parts: string[] = []
  if (params?.page) parts.push(`page=${params.page}`)
  if (params?.size) parts.push(`size=${params.size}`)
  if (params?.search) parts.push(`search=${encodeURIComponent(params.search)}`)
  if (params?.orderBy) parts.push(`orderBy=${encodeURIComponent(params.orderBy)}`)
  return parts.length > 0 ? `?${parts.join('&')}` : ''
}

function createSessionMessagesAdapter(fetchFn: FetchFn): SessionMessagesPort {
  return {
    async send(
      sessionId: string,
      message: { eventType: string; payload: string },
    ): Promise<DomainSessionMessage> {
      const url = `/api/ambient/v1/sessions/${sanitizeSessionId(sessionId)}/messages`
      const response = await fetchFn(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          event_type: message.eventType,
          payload: message.payload,
        }),
      })
      if (!response.ok) {
        throw new Error(`Failed to send session message: ${response.status}`)
      }
      const data = (await response.json()) as SessionMessageResponse
      return mapSessionMessageToDomain(data)
    },

    async list(
      sessionId: string,
      params?: ListParams,
    ): Promise<PaginatedResult<DomainSessionMessage>> {
      const qs = buildQueryString(params)
      const url = `/api/ambient/v1/sessions/${sanitizeSessionId(sessionId)}/messages${qs}`
      const response = await fetchFn(url, {
        method: 'GET',
      })
      if (!response.ok) {
        throw new Error(`Failed to list session messages: ${response.status}`)
      }
      const raw: unknown = await response.json()
      const page = params?.page ?? 1
      const size = params?.size ?? 20

      // The API may return a plain array or a paginated response object
      if (Array.isArray(raw)) {
        const items = (raw as SessionMessageResponse[]).map(mapSessionMessageToDomain)
        return {
          items,
          total: items.length,
          page,
          size,
          hasMore: false,
        }
      }

      const data = raw as SessionMessageListResponse
      if (!data.items || !Array.isArray(data.items)) {
        return { items: [], total: 0, page, size, hasMore: false }
      }
      const items = data.items.map(mapSessionMessageToDomain)
      return {
        items,
        total: data.total,
        page,
        size,
        hasMore: page * size < data.total,
      }
    },
  }
}

export function createSessionMessagesAdapterWithFetch(fetchFn?: FetchFn): SessionMessagesPort {
  return createSessionMessagesAdapter(fetchFn ?? globalThis.fetch.bind(globalThis))
}
