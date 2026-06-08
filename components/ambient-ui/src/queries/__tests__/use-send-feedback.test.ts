import { describe, it, expect, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement } from 'react'
import type { SessionMessagesPort } from '@/ports/session-messages'
import type { FeedbackBatch, FeedbackItem, DomainSessionMessage } from '@/domain/types'
import { useSendFeedback } from '../use-send-feedback'

function makeFeedbackItem(overrides: Partial<FeedbackItem> = {}): FeedbackItem {
  return {
    id: 'fb-001',
    type: 'element',
    comment: 'This button text is misleading',
    position: { x: 120, y: 340 },
    viewportWidth: 1280,
    viewportHeight: 720,
    deviceSize: 'desktop',
    timestamp: '2026-05-28T10:00:00Z',
    ...overrides,
  }
}

function makeBatch(overrides: Partial<FeedbackBatch> = {}): FeedbackBatch {
  return {
    items: [makeFeedbackItem()],
    sessionId: 'sess-001',
    previewUrl: 'https://app.example.com',
    ...overrides,
  }
}

function makeDomainMessage(overrides: Partial<DomainSessionMessage> = {}): DomainSessionMessage {
  return {
    id: 'msg-001',
    sessionId: 'sess-001',
    eventType: 'user_feedback',
    payload: 'test payload',
    seq: 1,
    createdAt: '2026-05-28T10:00:00Z',
    ...overrides,
  }
}

function createFakePort(options: {
  sendResult?: DomainSessionMessage
  sendCapture?: { calls: Array<{ sessionId: string; message: { eventType: string; payload: string } }> }
} = {}): SessionMessagesPort {
  return {
    send: async (sessionId, message) => {
      options.sendCapture?.calls.push({ sessionId, message })
      return options.sendResult ?? makeDomainMessage()
    },
    list: async () => ({ items: [], total: 0, page: 1, size: 20, hasMore: false }),
  }
}

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children)
  }
}

describe('useSendFeedback', () => {
  it('sends feedback batch via session messages port', async () => {
    const capture = { calls: [] as Array<{ sessionId: string; message: { eventType: string; payload: string } }> }
    const port = createFakePort({ sendCapture: capture })
    const { result } = renderHook(() => useSendFeedback(port), { wrapper: createWrapper() })

    result.current.mutate(makeBatch())

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(capture.calls).toHaveLength(1)
    expect(capture.calls[0].sessionId).toBe('sess-001')
    expect(capture.calls[0].message.eventType).toBe('user')
  })

  it('formats payload with element feedback', async () => {
    const capture = { calls: [] as Array<{ sessionId: string; message: { eventType: string; payload: string } }> }
    const port = createFakePort({ sendCapture: capture })
    const { result } = renderHook(() => useSendFeedback(port), { wrapper: createWrapper() })

    result.current.mutate(makeBatch({
      items: [makeFeedbackItem({
        comment: 'Fix the color',
        position: { x: 100, y: 200 },
        capturedHtml: '<button class="submit">Save</button>',
      })],
    }))

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const payload = capture.calls[0].message.payload
    expect(payload).toContain('Fix the color')
    expect(payload).toContain('Position: (100, 200)')
    expect(payload).toContain('<button class="submit">Save</button>')
    expect(payload).toContain('https://app.example.com')
  })

  it('formats payload with region feedback', async () => {
    const capture = { calls: [] as Array<{ sessionId: string; message: { eventType: string; payload: string } }> }
    const port = createFakePort({ sendCapture: capture })
    const { result } = renderHook(() => useSendFeedback(port), { wrapper: createWrapper() })

    result.current.mutate(makeBatch({
      items: [makeFeedbackItem({
        type: 'region',
        dimensions: { width: 200, height: 150 },
      })],
    }))

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const payload = capture.calls[0].message.payload
    expect(payload).toContain('Region: 200x150px')
  })

  it('formats payload with multiple items', async () => {
    const capture = { calls: [] as Array<{ sessionId: string; message: { eventType: string; payload: string } }> }
    const port = createFakePort({ sendCapture: capture })
    const { result } = renderHook(() => useSendFeedback(port), { wrapper: createWrapper() })

    result.current.mutate(makeBatch({
      items: [
        makeFeedbackItem({ id: 'fb-1', comment: 'First issue' }),
        makeFeedbackItem({ id: 'fb-2', comment: 'Second issue' }),
      ],
    }))

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const payload = capture.calls[0].message.payload
    expect(payload).toContain('Feedback #1')
    expect(payload).toContain('First issue')
    expect(payload).toContain('Feedback #2')
    expect(payload).toContain('Second issue')
  })

  it('includes viewport metadata', async () => {
    const capture = { calls: [] as Array<{ sessionId: string; message: { eventType: string; payload: string } }> }
    const port = createFakePort({ sendCapture: capture })
    const { result } = renderHook(() => useSendFeedback(port), { wrapper: createWrapper() })

    result.current.mutate(makeBatch({
      items: [makeFeedbackItem({
        viewportWidth: 375,
        viewportHeight: 667,
        deviceSize: 'mobile',
      })],
    }))

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const payload = capture.calls[0].message.payload
    expect(payload).toContain('Viewport: 375x667 (mobile)')
  })

  it('reports error on failure', async () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    const port: SessionMessagesPort = {
      send: async () => { throw new Error('Network error') },
      list: async () => ({ items: [], total: 0, page: 1, size: 20, hasMore: false }),
    }
    const { result } = renderHook(() => useSendFeedback(port), { wrapper: createWrapper() })

    result.current.mutate(makeBatch())

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.message).toBe('Network error')
    consoleSpy.mockRestore()
  })
})
