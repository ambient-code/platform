import { describe, it, expect } from 'vitest'
import type { SessionMessagesPort } from '@/ports/session-messages'
import type { FetchFn } from '../session-messages'
import { createSessionMessagesAdapterWithFetch } from '../session-messages'

type SdkMessageResponse = {
  id: string
  kind: string
  href: string
  created_at: string | null
  updated_at: string | null
  session_id: string
  event_type: string
  payload: string
  seq: number
}

function makeSdkMessage(overrides: Partial<SdkMessageResponse> = {}): SdkMessageResponse {
  return {
    id: 'msg-001',
    kind: 'SessionMessage',
    href: '/api/ambient/v1/sessions/sess-001/messages/msg-001',
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
    session_id: 'sess-001',
    event_type: 'tool_use',
    payload: '{"tool":"Read","path":"/app/main.py"}',
    seq: 1,
    ...overrides,
  }
}

type CapturedRequest = {
  url: string
  method: string
  headers: Record<string, string>
  body: unknown
}

function createFakeFetch(options: {
  response?: unknown
  status?: number
  captured?: CapturedRequest[]
}): FetchFn {
  const captured = options.captured ?? []

  return async (input: string, init?: RequestInit): Promise<Response> => {
    captured.push({
      url: input,
      method: init?.method ?? 'GET',
      headers: Object.fromEntries(
        Object.entries(init?.headers ?? {}),
      ) as Record<string, string>,
      body: init?.body ? JSON.parse(init.body as string) : undefined,
    })

    const status = options.status ?? 200
    const body = JSON.stringify(options.response ?? {})

    return {
      ok: status >= 200 && status < 300,
      status,
      json: async () => JSON.parse(body),
    } as Response
  }
}

describe('session-messages adapter', () => {
  describe('send()', () => {
    it('sends a POST to the correct BFF endpoint', async () => {
      const captured: CapturedRequest[] = []
      const responseMessage = makeSdkMessage()
      const fakeFetch = createFakeFetch({ response: responseMessage, captured })
      const adapter: SessionMessagesPort = createSessionMessagesAdapterWithFetch(fakeFetch)

      await adapter.send('sess-001', { eventType: 'tool_use', payload: '{"tool":"Read","path":"/app/main.py"}' })

      expect(captured).toHaveLength(1)
      expect(captured[0].url).toBe('/api/ambient/v1/sessions/sess-001/messages')
      expect(captured[0].method).toBe('POST')
      expect(captured[0].headers['Content-Type']).toBe('application/json')
    })

    it('sends the correct request body with snake_case keys', async () => {
      const captured: CapturedRequest[] = []
      const responseMessage = makeSdkMessage()
      const fakeFetch = createFakeFetch({ response: responseMessage, captured })
      const adapter: SessionMessagesPort = createSessionMessagesAdapterWithFetch(fakeFetch)

      await adapter.send('sess-001', { eventType: 'tool_use', payload: '{"tool":"Read","path":"/app/main.py"}' })

      expect(captured[0].body).toEqual({
        event_type: 'tool_use',
        payload: '{"tool":"Read","path":"/app/main.py"}',
      })
    })

    it('returns a mapped domain session message', async () => {
      const responseMessage = makeSdkMessage({
        id: 'msg-abc',
        session_id: 'sess-xyz',
        event_type: 'tool_result',
        payload: '{"result":"file contents here"}',
        seq: 5,
        created_at: '2026-05-28T12:00:00Z',
      })
      const fakeFetch = createFakeFetch({ response: responseMessage })
      const adapter: SessionMessagesPort = createSessionMessagesAdapterWithFetch(fakeFetch)

      const result = await adapter.send('sess-xyz', {
        eventType: 'tool_result',
        payload: '{"result":"file contents here"}',
      })

      expect(result.id).toBe('msg-abc')
      expect(result.sessionId).toBe('sess-xyz')
      expect(result.eventType).toBe('tool_result')
      expect(result.payload).toBe('{"result":"file contents here"}')
      expect(result.seq).toBe(5)
      expect(result.createdAt).toBe('2026-05-28T12:00:00Z')
    })

    it('throws on non-OK response', async () => {
      const fakeFetch = createFakeFetch({ status: 500 })
      const adapter: SessionMessagesPort = createSessionMessagesAdapterWithFetch(fakeFetch)

      await expect(
        adapter.send('sess-001', { eventType: 'ui.feedback', payload: '{}' }),
      ).rejects.toThrow('Failed to send session message: 500')
    })

    it('encodes special characters in session ID', async () => {
      const captured: CapturedRequest[] = []
      const responseMessage = makeSdkMessage()
      const fakeFetch = createFakeFetch({ response: responseMessage, captured })
      const adapter: SessionMessagesPort = createSessionMessagesAdapterWithFetch(fakeFetch)

      await adapter.send('sess/special&id', { eventType: 'system', payload: '{}' })

      expect(captured[0].url).toBe(
        '/api/ambient/v1/sessions/sess%2Fspecial%26id/messages',
      )
    })
  })

  describe('list()', () => {
    it('sends a GET to the correct BFF endpoint', async () => {
      const captured: CapturedRequest[] = []
      const listResponse = {
        kind: 'SessionMessageList',
        page: 1,
        size: 20,
        total: 0,
        items: [],
      }
      const fakeFetch = createFakeFetch({ response: listResponse, captured })
      const adapter: SessionMessagesPort = createSessionMessagesAdapterWithFetch(fakeFetch)

      await adapter.list('sess-001')

      expect(captured).toHaveLength(1)
      expect(captured[0].url).toBe('/api/ambient/v1/sessions/sess-001/messages')
      expect(captured[0].method).toBe('GET')
    })

    it('returns paginated domain messages', async () => {
      const messages = [
        makeSdkMessage({ id: 'msg-001', seq: 1 }),
        makeSdkMessage({ id: 'msg-002', seq: 2 }),
      ]
      const listResponse = {
        kind: 'SessionMessageList',
        page: 1,
        size: 20,
        total: 50,
        items: messages,
      }
      const fakeFetch = createFakeFetch({ response: listResponse })
      const adapter: SessionMessagesPort = createSessionMessagesAdapterWithFetch(fakeFetch)

      const result = await adapter.list('sess-001')

      expect(result.items).toHaveLength(2)
      expect(result.items[0].id).toBe('msg-001')
      expect(result.items[0].seq).toBe(1)
      expect(result.items[1].id).toBe('msg-002')
      expect(result.items[1].seq).toBe(2)
      expect(result.total).toBe(50)
      expect(result.page).toBe(1)
      expect(result.size).toBe(20)
      expect(result.hasMore).toBe(true)
    })

    it('returns hasMore=false when all items fit', async () => {
      const listResponse = {
        kind: 'SessionMessageList',
        page: 1,
        size: 20,
        total: 2,
        items: [
          makeSdkMessage({ id: 'msg-001' }),
          makeSdkMessage({ id: 'msg-002' }),
        ],
      }
      const fakeFetch = createFakeFetch({ response: listResponse })
      const adapter: SessionMessagesPort = createSessionMessagesAdapterWithFetch(fakeFetch)

      const result = await adapter.list('sess-001')

      expect(result.hasMore).toBe(false)
    })

    it('passes pagination params as query string', async () => {
      const captured: CapturedRequest[] = []
      const listResponse = {
        kind: 'SessionMessageList',
        page: 2,
        size: 10,
        total: 25,
        items: [],
      }
      const fakeFetch = createFakeFetch({ response: listResponse, captured })
      const adapter: SessionMessagesPort = createSessionMessagesAdapterWithFetch(fakeFetch)

      await adapter.list('sess-001', { page: 2, size: 10 })

      expect(captured[0].url).toBe(
        '/api/ambient/v1/sessions/sess-001/messages?page=2&size=10',
      )
    })

    it('throws on non-OK response', async () => {
      const fakeFetch = createFakeFetch({ status: 404 })
      const adapter: SessionMessagesPort = createSessionMessagesAdapterWithFetch(fakeFetch)

      await expect(adapter.list('sess-001')).rejects.toThrow(
        'Failed to list session messages: 404',
      )
    })

    it('maps SDK messages to domain messages', async () => {
      const listResponse = {
        kind: 'SessionMessageList',
        page: 1,
        size: 20,
        total: 1,
        items: [
          makeSdkMessage({
            id: 'msg-mapped',
            session_id: 'sess-mapped',
            event_type: 'error',
            payload: '{"error":"tool execution failed","code":"TOOL_ERROR"}',
            seq: 42,
            created_at: '2026-05-28T15:30:00Z',
          }),
        ],
      }
      const fakeFetch = createFakeFetch({ response: listResponse })
      const adapter: SessionMessagesPort = createSessionMessagesAdapterWithFetch(fakeFetch)

      const result = await adapter.list('sess-mapped')
      const msg = result.items[0]

      expect(msg.id).toBe('msg-mapped')
      expect(msg.sessionId).toBe('sess-mapped')
      expect(msg.eventType).toBe('error')
      expect(msg.payload).toBe('{"error":"tool execution failed","code":"TOOL_ERROR"}')
      expect(msg.seq).toBe(42)
      expect(msg.createdAt).toBe('2026-05-28T15:30:00Z')
    })

    it('handles null created_at by mapping to empty string', async () => {
      const listResponse = {
        kind: 'SessionMessageList',
        page: 1,
        size: 20,
        total: 1,
        items: [
          makeSdkMessage({ created_at: null }),
        ],
      }
      const fakeFetch = createFakeFetch({ response: listResponse })
      const adapter: SessionMessagesPort = createSessionMessagesAdapterWithFetch(fakeFetch)

      const result = await adapter.list('sess-001')

      expect(result.items[0].createdAt).toBe('')
    })
  })
})
