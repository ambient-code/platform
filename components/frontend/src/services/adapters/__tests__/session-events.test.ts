import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createSessionEventsAdapter } from '../session-events'

describe('sessionEventsAdapter', () => {
  const adapter = createSessionEventsAdapter()

  describe('createEventSource', () => {
    it('creates EventSource with correct URL', () => {
      const MockEventSource = vi.fn()
      vi.stubGlobal('EventSource', MockEventSource)

      adapter.createEventSource('my-project', 'my-session')
      expect(MockEventSource).toHaveBeenCalledWith(
        '/api/projects/my-project/agentic-sessions/my-session/agui/events'
      )

      vi.unstubAllGlobals()
    })

    it('appends runId as query parameter', () => {
      const MockEventSource = vi.fn()
      vi.stubGlobal('EventSource', MockEventSource)

      adapter.createEventSource('proj', 'sess', 'run-123')
      expect(MockEventSource).toHaveBeenCalledWith(
        '/api/projects/proj/agentic-sessions/sess/agui/events?runId=run-123'
      )

      vi.unstubAllGlobals()
    })

    it('encodes special characters in project and session names', () => {
      const MockEventSource = vi.fn()
      vi.stubGlobal('EventSource', MockEventSource)

      adapter.createEventSource('my project', 'my session')
      expect(MockEventSource).toHaveBeenCalledWith(
        '/api/projects/my%20project/agentic-sessions/my%20session/agui/events'
      )

      vi.unstubAllGlobals()
    })
  })

  describe('sendMessage', () => {
    beforeEach(() => {
      vi.restoreAllMocks()
    })

    it('sends POST request with payload', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ runId: 'run-abc' }),
      })
      vi.stubGlobal('fetch', mockFetch)

      const payload = {
        threadId: 'thread-1',
        messages: [{ id: '1', role: 'user' as const, content: 'Hello' }],
        tools: [],
      }

      const result = await adapter.sendMessage('proj', 'sess', payload)
      expect(result).toEqual({ runId: 'run-abc' })
      expect(mockFetch).toHaveBeenCalledWith(
        '/api/projects/proj/agentic-sessions/sess/agui/run',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
        })
      )

      vi.unstubAllGlobals()
    })

    it('throws on non-ok response', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: false,
        text: () => Promise.resolve('Bad request'),
      })
      vi.stubGlobal('fetch', mockFetch)

      const payload = {
        threadId: 'thread-1',
        messages: [{ id: '1', role: 'user' as const, content: 'Hello' }],
        tools: [],
      }

      await expect(adapter.sendMessage('proj', 'sess', payload)).rejects.toThrow('Bad request')

      vi.unstubAllGlobals()
    })
  })

  describe('interrupt', () => {
    beforeEach(() => {
      vi.restoreAllMocks()
    })

    it('sends POST to interrupt endpoint', async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true })
      vi.stubGlobal('fetch', mockFetch)

      await adapter.interrupt('proj', 'sess', 'run-123')
      expect(mockFetch).toHaveBeenCalledWith(
        '/api/projects/proj/agentic-sessions/sess/agui/interrupt',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ runId: 'run-123' }),
        })
      )

      vi.unstubAllGlobals()
    })

    it('throws on non-ok response', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: false,
        statusText: 'Internal Server Error',
      })
      vi.stubGlobal('fetch', mockFetch)

      await expect(adapter.interrupt('proj', 'sess', 'run-123')).rejects.toThrow('Failed to interrupt')

      vi.unstubAllGlobals()
    })
  })
})
