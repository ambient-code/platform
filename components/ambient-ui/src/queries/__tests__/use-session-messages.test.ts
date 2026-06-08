import { describe, it, expect } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement } from 'react'
import type { SessionMessagesPort } from '@/ports/session-messages'
import type { DomainSessionMessage, PaginatedResult } from '@/domain/types'
import { useSessionMessages } from '../use-session-messages'

function makeDomainMessage(overrides: Partial<DomainSessionMessage> = {}): DomainSessionMessage {
  return {
    id: 'msg-001',
    sessionId: 'sess-001',
    eventType: 'text',
    payload: 'Hello world',
    seq: 1,
    createdAt: '2026-05-28T10:00:00Z',
    ...overrides,
  }
}

function createFakePort(options: {
  messages?: DomainSessionMessage[]
  listCapture?: { calls: Array<{ sessionId: string }> }
} = {}): SessionMessagesPort {
  const messages = options.messages ?? [makeDomainMessage()]
  return {
    send: async () => makeDomainMessage(),
    list: async (sessionId) => {
      options.listCapture?.calls.push({ sessionId })
      const result: PaginatedResult<DomainSessionMessage> = {
        items: messages,
        total: messages.length,
        page: 1,
        size: 100,
        hasMore: false,
      }
      return result
    },
  }
}

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children)
  }
}

describe('useSessionMessages', () => {
  it('fetches messages for a session', async () => {
    const messages = [
      makeDomainMessage({ id: 'msg-1', seq: 1, eventType: 'text', payload: 'First' }),
      makeDomainMessage({ id: 'msg-2', seq: 2, eventType: 'tool_use', payload: 'ls -la' }),
    ]
    const port = createFakePort({ messages })
    const { result } = renderHook(
      () => useSessionMessages('sess-001', port),
      { wrapper: createWrapper() },
    )

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.items).toHaveLength(2)
    expect(result.current.data?.items[0].eventType).toBe('text')
    expect(result.current.data?.items[1].eventType).toBe('tool_use')
  })

  it('passes the session ID to the port', async () => {
    const capture = { calls: [] as Array<{ sessionId: string }> }
    const port = createFakePort({ listCapture: capture })
    const { result } = renderHook(
      () => useSessionMessages('sess-042', port),
      { wrapper: createWrapper() },
    )

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(capture.calls).toHaveLength(1)
    expect(capture.calls[0].sessionId).toBe('sess-042')
  })

  it('does not fetch when sessionId is empty', async () => {
    const capture = { calls: [] as Array<{ sessionId: string }> }
    const port = createFakePort({ listCapture: capture })
    const { result } = renderHook(
      () => useSessionMessages('', port),
      { wrapper: createWrapper() },
    )

    // Give it a moment to ensure it doesn't fire
    await new Promise(resolve => setTimeout(resolve, 50))
    expect(result.current.isFetching).toBe(false)
    expect(capture.calls).toHaveLength(0)
  })

  it('returns paginated result structure', async () => {
    const messages = [
      makeDomainMessage({ id: 'msg-1', seq: 1 }),
      makeDomainMessage({ id: 'msg-2', seq: 2 }),
      makeDomainMessage({ id: 'msg-3', seq: 3 }),
    ]
    const port = createFakePort({ messages })
    const { result } = renderHook(
      () => useSessionMessages('sess-001', port),
      { wrapper: createWrapper() },
    )

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual({
      items: messages,
      total: 3,
      page: 1,
      size: 100,
      hasMore: false,
    })
  })
})
